// Program: soft_DNA - ignore.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Implements the ignore matcher used by the scanner, combining .gitignore-style patterns from the project with extra caller-supplied patterns.

package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ignoreMatcher is the ordered list of ignore patterns; a path is ignored when any pattern matches.
type ignoreMatcher []string

// newIgnoreMatcher combines the root's .gitignore patterns with the extra patterns supplied by the caller.
func newIgnoreMatcher(root string, extra []string) ignoreMatcher {
	var patterns []string
	patterns = append(patterns, readGitignore(root)...)
	patterns = append(patterns, extra...)
	return ignoreMatcher(patterns)
}

// readGitignore returns the patterns from root's .gitignore, skipping blank lines, comments and negation lines (which this matcher does not support); a missing file yields none.
func readGitignore(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return nil
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, filepath.ToSlash(line))
	}
	return patterns
}

// match reports whether the path rel (base is its final element) is ignored by any pattern.
func (m ignoreMatcher) match(rel, base string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range m {
		if matchPattern(pattern, rel, base, isDir) {
			return true
		}
	}
	return false
}

// matchPattern tests one pattern: a trailing slash limits it to directories (and files below them), a pattern without a slash is matched against the base name, and other patterns against the whole relative path or a directory prefix of it.
func matchPattern(pattern, rel, base string, isDir bool) bool {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" {
		return false
	}

	dirOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	if dirOnly && !isDir {
		return strings.HasPrefix(rel, pattern+"/")
	}

	if !strings.Contains(pattern, "/") {
		if ok, _ := filepath.Match(pattern, base); ok {
			return true
		}
		return base == pattern
	}

	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	return rel == pattern || strings.HasPrefix(rel, pattern+"/")
}
