package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

type ignoreMatcher []string

func newIgnoreMatcher(root string, extra []string) ignoreMatcher {
	var patterns []string
	patterns = append(patterns, readGitignore(root)...)
	patterns = append(patterns, extra...)
	return ignoreMatcher(patterns)
}

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

func (m ignoreMatcher) match(rel, base string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range m {
		if matchPattern(pattern, rel, base, isDir) {
			return true
		}
	}
	return false
}

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
