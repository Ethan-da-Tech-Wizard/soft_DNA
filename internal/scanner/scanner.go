// Program: soft_DNA - scanner.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Walks a project folder and builds a sorted inventory of files (size, line count, language, content hash), skipping ignored, oversized, binary and unsafe entries without running any project code.

// Package scanner inventories source files without executing target code.
package scanner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// defaultMaxFileSize is the largest file (2 MiB) that is read when the caller sets no limit.
const defaultMaxFileSize int64 = 2 * 1024 * 1024

// defaultIgnoredDirs lists directory names that are never entered, such as version control, caches, virtual environments and build output.
var defaultIgnoredDirs = map[string]struct{}{
	".cache":        {},
	".claude":       {},
	".eggs":         {},
	".git":          {},
	".gradle":       {},
	".hg":           {},
	".idea":         {},
	".mypy_cache":   {},
	".next":         {},
	".pytest_cache": {},
	".ruff_cache":   {},
	".svn":          {},
	".terraform":    {},
	".tox":          {},
	".venv":         {},
	".vscode":       {},
	"__pycache__":   {},
	"build":         {},
	"coverage":      {},
	"dist":          {},
	"node_modules":  {},
	"target":        {},
	"venv":          {},
}

// FileInfo describes one inventoried path.
type FileInfo struct {
	Path        string
	Ext         string
	SizeBytes   int64
	LineCount   int
	Language    string
	ContentHash string
	IsBinary    bool
	Skipped     bool
	SkipReason  string
}

// Options controls scanner behavior.
type Options struct {
	MaxFileSize int64
	Ignore      []string
}

// Scan walks root and returns a deterministic file inventory.
func Scan(root string, opts Options) ([]FileInfo, error) {
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = defaultMaxFileSize
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}
	if !rootInfo.IsDir() {
		return nil, errors.New("root is not a directory")
	}

	// Walk the tree; read errors are recorded as skipped entries instead of stopping the scan.
	ignore := newIgnoreMatcher(absRoot, opts.Ignore)
	var files []FileInfo

	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			files = append(files, skippedInfo(absRoot, path, "read_error"))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if path == absRoot {
			return nil
		}

		rel := cleanRel(absRoot, path)
		if entry.IsDir() {
			if shouldSkipDir(rel, entry.Name(), ignore) {
				return filepath.SkipDir
			}
			return nil
		}

		if ignore.match(rel, entry.Name(), false) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			files = append(files, skippedInfo(absRoot, path, "read_error"))
			return nil
		}

		// Symlinks get their own handling so a link can never lead the scan outside the root.
		if info.Mode()&os.ModeSymlink != 0 {
			files = append(files, scanSymlink(absRoot, path, rel, opts.MaxFileSize))
			return nil
		}

		files = append(files, scanRegularFile(absRoot, path, rel, info, opts.MaxFileSize))
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by path so the inventory is the same on every run.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// shouldSkipDir reports whether a directory is never entered: it is in the default list or matches an ignore pattern.
func shouldSkipDir(rel, name string, ignore ignoreMatcher) bool {
	if _, ok := defaultIgnoredDirs[name]; ok {
		return true
	}
	return ignore.match(rel, name, true)
}

// scanRegularFile records one file, marking it skipped when it is too large, unreadable or binary, and otherwise computing its SHA-256 hash and line count.
func scanRegularFile(absRoot, path, rel string, info fs.FileInfo, maxFileSize int64) FileInfo {
	file := baseInfo(rel, info.Size())
	if info.Size() > maxFileSize {
		file.Skipped = true
		file.SkipReason = "max_file_size"
		return file
	}

	data, err := os.ReadFile(path)
	if err != nil {
		file.Skipped = true
		file.SkipReason = "read_error"
		return file
	}

	if isBinary(data) {
		file.IsBinary = true
		file.Skipped = true
		file.SkipReason = "binary"
		return file
	}

	sum := sha256.Sum256(data)
	file.ContentHash = hex.EncodeToString(sum[:])
	file.LineCount = lineCount(data)
	return file
}

// scanSymlink resolves a link and scans its target only when the target is a file inside the root; unresolved, escaping and directory links are recorded as skipped.
func scanSymlink(absRoot, path, rel string, maxFileSize int64) FileInfo {
	file := baseInfo(rel, 0)
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		file.Skipped = true
		file.SkipReason = "symlink_unresolved"
		return file
	}
	if !isInside(absRoot, target) {
		file.Skipped = true
		file.SkipReason = "symlink_escape"
		return file
	}

	info, err := os.Stat(target)
	if err != nil {
		file.Skipped = true
		file.SkipReason = "read_error"
		return file
	}
	if info.IsDir() {
		file.Skipped = true
		file.SkipReason = "symlink_directory"
		return file
	}
	return scanRegularFile(absRoot, target, rel, info, maxFileSize)
}

// skippedInfo builds a skipped entry with the given reason for a path that could not be processed.
func skippedInfo(absRoot, path, reason string) FileInfo {
	file := baseInfo(cleanRel(absRoot, path), 0)
	file.Skipped = true
	file.SkipReason = reason
	return file
}

// baseInfo creates a FileInfo with the path, lower-cased extension, size and detected language filled in.
func baseInfo(rel string, size int64) FileInfo {
	ext := strings.ToLower(filepath.Ext(rel))
	return FileInfo{
		Path:      rel,
		Ext:       ext,
		SizeBytes: size,
		Language:  detectLanguage(ext, filepath.Base(rel)),
	}
}

// cleanRel returns path relative to absRoot using forward slashes.
func cleanRel(absRoot, path string) string {
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}

// isInside reports whether target is absRoot itself or lies below it.
func isInside(absRoot, target string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

// isBinary treats data as binary when its first 8000 bytes contain a zero byte.
func isBinary(data []byte) bool {
	sample := data
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

// lineCount counts lines, including a final line that has no trailing newline.
func lineCount(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}

// detectLanguage maps a file extension, or for some files its exact name, to a language label, or "Unknown".
func detectLanguage(ext, base string) string {
	switch ext {
	case ".go":
		return "Go"
	case ".js", ".jsx":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".sql":
		return "SQL"
	case ".swift":
		return "Swift"
	case ".rb":
		return "Ruby"
	case ".r":
		return "R"
	case ".rs":
		return "Rust"
	case ".c":
		return "C"
	case ".cpp", ".hpp", ".h", ".cc", ".cxx":
		return "C++"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".toml":
		return "TOML"
	case ".yaml", ".yml":
		return "YAML"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	}

	switch base {
	case "Dockerfile", "Makefile":
		return base
	case "CMakeLists.txt":
		return "CMake"
	default:
		return "Unknown"
	}
}
