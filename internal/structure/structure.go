// Package structure derives simple folder roles and entry points.
package structure

import (
	"path/filepath"
	"sort"
	"strings"

	"codelens-local/internal/scanner"
)

// Folder describes a top-level folder or the repository root.
type Folder struct {
	Path      string
	Role      string
	FileCount int
}

// EntryPoint describes a likely starting file.
type EntryPoint struct {
	Path   string
	Reason string
}

// Analyze derives folder roles and likely entry points from inventory.
func Analyze(files []scanner.FileInfo) ([]Folder, []EntryPoint) {
	folders := folders(files)
	entries := entryPoints(files)
	return folders, entries
}

func folders(files []scanner.FileInfo) []Folder {
	counts := map[string]int{}
	for _, file := range files {
		if file.Skipped {
			continue
		}
		folder := topFolder(file.Path)
		counts[folder]++
	}

	out := make([]Folder, 0, len(counts))
	for folder, count := range counts {
		out = append(out, Folder{
			Path:      folder,
			Role:      roleFor(folder),
			FileCount: count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func topFolder(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return "."
	}
	parts := strings.Split(dir, "/")
	return parts[0]
}

func roleFor(folder string) string {
	switch folder {
	case ".":
		return "Root files and project manifests"
	case "cmd":
		return "Command-line entry points"
	case "internal":
		return "Private application packages"
	case "pkg":
		return "Reusable public packages"
	case "src":
		return "Application source"
	case "testdata", "tests", "__tests__":
		return "Tests and fixtures"
	case "docs", "files":
		return "Project documentation"
	case "public", "static":
		return "Static assets"
	default:
		return "Project files"
	}
}

func entryPoints(files []scanner.FileInfo) []EntryPoint {
	var entries []EntryPoint
	for _, file := range files {
		if file.Skipped {
			continue
		}
		if reason := entryReason(file.Path); reason != "" {
			entries = append(entries, EntryPoint{Path: file.Path, Reason: reason})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func entryReason(path string) string {
	base := filepath.Base(path)
	switch {
	case path == "main.py":
		return "Python main script convention"
	case path == "app.py":
		return "Python app convention"
	case path == "main.go":
		return "Go main package convention"
	case strings.HasPrefix(path, "cmd/") && base == "main.go":
		return "Go cmd/*/main.go convention"
	case path == "index.html":
		return "Static HTML entry point"
	case path == "src/main.tsx" || path == "src/main.ts" || path == "src/main.jsx" || path == "src/main.js":
		return "Frontend main module convention"
	default:
		return ""
	}
}
