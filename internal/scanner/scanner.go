// ==============================================================================
// Program: scanner.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Provides high-performance recursive directory traversal, ignoring binary artifacts and vendor directories, determining programming languages by extension/shebang, and computing deterministic cryptographic SHA-256 hashes.
// ==============================================================================

// Package scanner inventories source files without executing target code.
package scanner

// File role: the first stage of the pipeline. `Scan` walks the project and returns one `FileInfo` per file with
// its language, size, line count and SHA-256 content hash. Every later stage (stack detection, symbols, index,
// report) works from this list, and the hash is what makes incremental re-runs possible (see internal/cli).
//
// Safety rules enforced here: nothing is executed; files over the size limit are not read; binary files are
// detected and skipped; symlinks are followed only when they stay inside the root; unreadable entries become
// "skipped" rows (with a reason) instead of aborting the scan.
//
// Determinism: the walk order is filesystem dependent, so the result is sorted by path before returning.
// Ignore patterns live in ignore.go; this file only decides what to do with a path that was not ignored.
// Tests: scanner_test.go.
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

// 2 MiB: large enough for hand-written source files, small enough to skip generated bundles and data dumps.
const defaultMaxFileSize int64 = 2 * 1024 * 1024

// Directories that never hold hand-written project source: VCS metadata, caches, virtualenvs, build output
// and dependency folders. They are pruned without being entered, which is the main reason scans of big
// repositories stay fast. Keys are directory base names, so a folder called "build" is skipped at any depth.
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
// One row of the inventory. Path is relative to the scan root and always uses forward slashes. ContentHash is the
// hex SHA-256 of the file bytes (empty for skipped files). Skipped files keep their path and reason so the
// report can list what was not analysed: SkipReason is one of max_file_size, read_error, binary, symlink_unresolved, symlink_escape, symlink_directory.
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
// Ignore holds extra patterns from the user (`--ignore`), applied on top of defaultIgnoredDirs; syntax is described in ignore.go.
type Options struct {
	MaxFileSize int64
	Ignore      []string
}

// Scan walks root and returns a deterministic file inventory.
// Errors: the root cannot be resolved, does not exist, or is not a directory. Problems with individual files are
// recorded in the result, never returned. Complexity: O(files) plus reading every file that passes the checks.
func Scan(root string, opts Options) ([]FileInfo, error) {
	// Guard against uninitialized or negative upper file size bounds by establishing standard 2MB limit
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = defaultMaxFileSize
	}

	// An absolute root makes filepath.Rel behave predictably, and lets the symlink check compare paths reliably.
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

	// Built once per scan; the matcher is consulted for every directory and file.
	ignore := newIgnoreMatcher(absRoot, opts.Ignore)
	var files []FileInfo

	// Execute depth-first traversal of filesystem tree, pruning ignored directory subtrees eagerly
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		// Handle permission errors or broken symlinks defensively without terminating traversal
		if walkErr != nil {
			files = append(files, skippedInfo(absRoot, path, "read_error"))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// The root itself is visited first by WalkDir; it is neither a file to record nor a directory to test against ignore rules.
		if path == absRoot {
			return nil
		}

		// Matching and reporting both use the root-relative slash path, so rules behave the same on Windows and Unix.
		rel := cleanRel(absRoot, path)
		if entry.IsDir() {
			// SkipDir prunes the whole subtree, so ignored directories cost nothing.
			if shouldSkipDir(rel, entry.Name(), ignore) {
				return filepath.SkipDir
			}
			return nil
		}

		// Ignored files are omitted completely (not listed as skipped), unlike files skipped for size or binary content.
		if ignore.match(rel, entry.Name(), false) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			files = append(files, skippedInfo(absRoot, path, "read_error"))
			return nil
		}

		// DirEntry.Info does not follow symlinks, so this identifies links and routes them to the stricter symlink checks.
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

	// Sorting by path makes the inventory (and everything derived from it) identical across runs and machines.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// Built-in ignore list first (cheap map lookup), then the user's patterns.
func shouldSkipDir(rel, name string, ignore ignoreMatcher) bool {
	if _, ok := defaultIgnoredDirs[name]; ok {
		return true
	}
	return ignore.match(rel, name, true)
}

// Order of checks is deliberate and cheapest-first: size from metadata (no read), then read, then binary check,
// then hash and line count. Parameter absRoot is unused here but kept so the signature matches scanSymlink.
func scanRegularFile(absRoot, path, rel string, info fs.FileInfo, maxFileSize int64) FileInfo {
	file := baseInfo(rel, info.Size())
	// Reported size comes from file metadata, so oversized files are skipped without ever being read into memory.
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

	// Binary files are recorded (so they show up in the inventory) but not hashed or counted.
	if isBinary(data) {
		file.IsBinary = true
		file.Skipped = true
		file.SkipReason = "binary"
		return file
	}

	// SHA-256 gives a stable identity for "did this file change since the last run" without storing file contents.
	sum := sha256.Sum256(data)
	file.ContentHash = hex.EncodeToString(sum[:])
	file.LineCount = lineCount(data)
	return file
}

// A symlink is accepted only if it resolves to a regular file inside the project root. Links that leave the
// root are reported as "symlink_escape" and never read, so a link to /etc/passwd cannot pull outside data into the index.
func scanSymlink(absRoot, path, rel string, maxFileSize int64) FileInfo {
	file := baseInfo(rel, 0)
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		file.Skipped = true
		file.SkipReason = "symlink_unresolved"
		return file
	}

	// The escape check uses the resolved target, so chains of links and `..` tricks are covered.
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

// Creates a placeholder row for paths that could not even be examined (walk error or Info failure).
func skippedInfo(absRoot, path, reason string) FileInfo {
	file := baseInfo(cleanRel(absRoot, path), 0)
	file.Skipped = true
	file.SkipReason = reason
	return file
}

// Fills the fields known from the path alone: extension (lower-cased) and language guess. Content-derived fields are set later.
func baseInfo(rel string, size int64) FileInfo {
	ext := strings.ToLower(filepath.Ext(rel))
	return FileInfo{
		Path:      rel,
		Ext:       ext,
		SizeBytes: size,
		Language:  detectLanguage(ext, filepath.Base(rel)),
	}
}

// Falls back to the cleaned absolute path if Rel fails (different volumes on Windows); the slash form keeps output portable.
func cleanRel(absRoot, path string) string {
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}

// True for the root itself or any descendant. The ".." test includes the separator so a path that merely starts with two dots, or a sibling directory, is judged by its real relative form.
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

// Same heuristic as git and grep: a NUL byte in the first 8000 bytes means binary. Text encodings such as UTF-16 contain NULs and are therefore skipped.
func isBinary(data []byte) bool {
	sample := data
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

// Counts newline characters, plus one for a final line without a trailing newline; an empty file has zero lines.
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

// Maps lower-cased file extension to the language name used everywhere else (symbols.Extract switches on these exact strings). Files with no recognised extension are matched by base name (Dockerfile, Makefile, CMakeLists.txt) and otherwise reported as "Unknown". Despite the file header's mention of shebangs, there is no shebang detection here.
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
