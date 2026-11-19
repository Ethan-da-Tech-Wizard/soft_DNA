// ==============================================================================
// Program: ignore.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Decides whether the scanner should skip a path, using the repository's root .gitignore
// plus any extra patterns passed by the caller.
//
// How it fits the app: scanner.go builds one ignoreMatcher per scan (newIgnoreMatcher) and calls
// match() for every file and directory it walks (shouldSkipDir additionally applies the scanner's
// built-in defaultIgnoredDirs list before consulting this matcher). Paths that match are left out
// of the file inventory entirely, so patterns here directly shrink what CodeLens analyses.
//
// Supported subset (deliberately NOT full gitignore semantics): shell globs via filepath.Match,
// trailing "/" for directory-only patterns, and plain-prefix matching for patterns that contain a
// slash. Not supported: "!" negation (such lines are dropped), "**" globs, and .gitignore files in
// subdirectories (only the root one is read).
// Known limit: because negations are dropped, a repo that re-includes a file with "!" will see that
// file skipped here.
// ==============================================================================

package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ignoreMatcher is an ordered list of slash-normalised patterns; a path is ignored if any pattern
// matches. It is a plain slice because the lists are small and order does not matter (no negation).
type ignoreMatcher []string

// newIgnoreMatcher combines the root .gitignore patterns with caller-supplied extras (the scanner
// passes its Options.Ignore list), .gitignore first.
//
// root is the repository directory being scanned; a missing or unreadable .gitignore simply
// contributes no patterns.
func newIgnoreMatcher(root string, extra []string) ignoreMatcher {
	var patterns []string
	patterns = append(patterns, readGitignore(root)...)
	patterns = append(patterns, extra...)
	return ignoreMatcher(patterns)
}

// readGitignore returns the usable patterns from root/.gitignore, or nil if the file cannot be read.
// Blank lines, "#" comments and "!" negations are dropped (see the file header for why negation is
// unsupported). Patterns are converted to forward slashes so matching behaves the same on Windows.
func readGitignore(root string) []string {
	// Not-found and permission errors are treated alike: ignoring nothing is the safe default
	// for a read-only analysis tool.
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return nil
	}

	var patterns []string
	// Splitting on "\n" (not "\r\n") is fine because TrimSpace below removes any trailing "\r".
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, filepath.ToSlash(line))
	}
	return patterns
}

// match reports whether the path should be skipped.
//
// rel is the path relative to the scan root, base is its final element (filepath.Base(rel)), and
// isDir says whether it is a directory (directory-only patterns behave differently for files).
func (m ignoreMatcher) match(rel, base string, isDir bool) bool {
	// Normalise separators so patterns written with "/" match on every OS.
	rel = filepath.ToSlash(rel)
	for _, pattern := range m {
		if matchPattern(pattern, rel, base, isDir) {
			return true
		}
	}
	return false
}

// matchPattern tests one pattern against one path; see ignoreMatcher.match for the arguments.
//
// Three cases, mirroring the common gitignore forms:
//   - "build/"  directory-only: matches the directory itself, or any file below it
//   - "*.log"   no slash: matched against the file/dir name only, at any depth
//   - "docs/x"  contains a slash: matched against the whole relative path, or as a prefix
//
// A malformed glob makes filepath.Match return an error; that is ignored, so a bad pattern simply
// never matches instead of aborting the scan.
func matchPattern(pattern, rel, base string, isDir bool) bool {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" {
		return false
	}

	// A trailing slash means "directories only"; strip it and remember the restriction.
	dirOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	if dirOnly && !isDir {
		// A file is still ignored if it lives inside the named directory. This is a safety net: the
		// walker normally prunes the directory first, but the rule keeps match() correct on its own.
		return strings.HasPrefix(rel, pattern+"/")
	}

	// No slash: git matches such patterns at any depth, so compare with the base name only.
	if !strings.Contains(pattern, "/") {
		if ok, _ := filepath.Match(pattern, base); ok {
			return true
		}
		return base == pattern
	}

	// Slash inside the pattern: anchored to the scan root, so compare the full relative path. The
	// prefix test makes "docs/generated" also cover everything beneath that directory.
	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	return rel == pattern || strings.HasPrefix(rel, pattern+"/")
}
