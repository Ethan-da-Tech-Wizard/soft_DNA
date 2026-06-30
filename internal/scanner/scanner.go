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

const defaultMaxFileSize int64 = 2 * 1024 * 1024

var defaultIgnoredDirs = map[string]struct{}{
	".cache":       {},
	".git":         {},
	".idea":        {},
	".next":        {},
	".venv":        {},
	".vscode":      {},
	"__pycache__":  {},
	"build":        {},
	"coverage":     {},
	"dist":         {},
	"node_modules": {},
	"target":       {},
	"venv":         {},
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

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func shouldSkipDir(rel, name string, ignore ignoreMatcher) bool {
	if _, ok := defaultIgnoredDirs[name]; ok {
		return true
	}
	return ignore.match(rel, name, true)
}

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

func skippedInfo(absRoot, path, reason string) FileInfo {
	file := baseInfo(cleanRel(absRoot, path), 0)
	file.Skipped = true
	file.SkipReason = reason
	return file
}

func baseInfo(rel string, size int64) FileInfo {
	ext := strings.ToLower(filepath.Ext(rel))
	return FileInfo{
		Path:      rel,
		Ext:       ext,
		SizeBytes: size,
		Language:  detectLanguage(ext, filepath.Base(rel)),
	}
}

func cleanRel(absRoot, path string) string {
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}

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

func isBinary(data []byte) bool {
	sample := data
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

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
	case ".rs":
		return "Rust"
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
	default:
		return "Unknown"
	}
}
