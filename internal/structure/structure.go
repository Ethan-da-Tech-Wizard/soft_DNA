// ==============================================================================
// Program: structure.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Turns the scanner's flat file list into two small summaries: a role for each top-level
// folder (for example "cmd" -> "Command-line entry points") and a list of likely entry-point files
// (main.go, src/main.rs, index.html, ...).
//
// How it fits the app: the CLI calls Analyze after scanner.Scan; diagram.WriteMermaid and the
// reports consume the Folder and EntryPoint results, while the GUI and grounding code reuse RoleFor
// directly. Everything here is heuristic and based on file and folder names only; no file contents
// are read.
// Known limits: only the first path segment is considered (no deeper tree or depth metrics), and
// unknown folder names fall back to the generic role "Project files".
// ==============================================================================

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
//
// files is the scanner output; entries marked Skipped (unreadable, or larger than the scanner's size cap) are not counted.
// Both returned slices are sorted by path so reports and diagrams are stable between runs.
func Analyze(files []scanner.FileInfo) ([]Folder, []EntryPoint) {
	// The local variable intentionally shadows the helper of the same name; the helper is only needed once here.
	folders := folders(files)
	entries := entryPoints(files)
	return folders, entries
}

// folders groups files by top-level folder and returns one Folder per group, sorted by path.
func folders(files []scanner.FileInfo) []Folder {
	counts := map[string]int{}
	for _, file := range files {
		// Skipped files would inflate counts with content the tool never analysed.
		if file.Skipped {
			continue
		}
		folder := topFolder(file.Path)
		counts[folder]++
	}

	// Map iteration order is random in Go, hence the explicit sort below.
	out := make([]Folder, 0, len(counts))
	for folder, count := range counts {
		out = append(out, Folder{
			Path:      folder,
			Role:      RoleFor(folder),
			FileCount: count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

// topFolder returns the first directory segment of a relative file path, or "." for files that sit
// directly in the project root (which are grouped under the "Root files" role).
func topFolder(path string) string {
	// ToSlash so Windows-style separators from the scanner split correctly below.
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return "."
	}
	parts := strings.Split(dir, "/")
	return parts[0]
}

// RoleFor returns a friendly description of a folder's likely purpose.
//
// The match is case-insensitive and based purely on the common naming conventions listed below
// (Go, JS/TS, Python, web and DevOps layouts); "." means the project root. Unknown names return
// "Project files". Exported because the GUI and the grounding prompt reuse the same wording.
func RoleFor(folder string) string {
	switch strings.ToLower(folder) {
	case ".":
		return "Root files and project manifests"
	case "cmd", "bin":
		return "Command-line entry points"
	case "internal":
		return "Private application packages"
	case "pkg":
		return "Reusable public packages"
	case "src", "source", "app", "core":
		return "Application source"
	case "testdata", "tests", "__tests__", "test", "spec", "specs":
		return "Tests and fixtures"
	case "docs", "files", "doc":
		return "Project documentation"
	case "public", "static", "assets", "images", "media", "styles", "css", "js":
		return "Static assets"
	case "api", "controllers", "routes", "handlers", "server":
		return "API & Network Handlers"
	case "models", "db", "migrations", "database", "store", "repository":
		return "Database & Storage Models"
	case "configs", "config", "settings", "env":
		return "Configuration Settings"
	case "web", "frontend", "components", "views", "templates", "pages":
		return "User Interface & Frontend"
	case "backend", "services", "utils", "helpers", "lib", "tools":
		return "Backend Services & Utilities"
	case "notebooks", "notebook", "scripts", "script":
		return "Notebooks & Scripting"
	case ".github", ".gitlab", "ci", "workflows":
		return "CI/CD & Workflows"
	default:
		return "Project files"
	}
}

// entryPoints lists files whose path matches a known "start here" convention, sorted by path.
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

// entryReason returns a human-readable reason if path is a conventional entry point, else "".
//
// Most rules compare the full relative path, so "main.py" only counts at the project root. The one
// exception is Go's cmd/<name>/main.go layout, which is matched by prefix plus base name. Case
// order matters only where patterns overlap (root main.go is reported before the cmd/ rule).
func entryReason(path string) string {
	base := filepath.Base(path)
	switch {
	case path == "main.py":
		return "Python main script convention"
	case path == "app.py":
		return "Python app convention"
	case path == "main.rb" || path == "app.rb":
		return "Ruby entry point convention"
	case path == "main.R" || path == "run.R":
		return "R script entry point convention"
	case path == "main.swift" || path == "App.swift":
		return "Swift entry point convention"
	case path == "schema.sql" || path == "init.sql":
		return "SQL initialization entry point convention"
	case path == "main.go":
		return "Go main package convention"
	case strings.HasPrefix(path, "cmd/") && base == "main.go":
		return "Go cmd/*/main.go convention"
	case path == "index.html":
		return "Static HTML entry point"
	case path == "src/main.tsx" || path == "src/main.ts" || path == "src/main.jsx" || path == "src/main.js":
		return "Frontend main module convention"
	case path == "src/main.rs" || path == "src/lib.rs":
		return "Rust main/library entry point convention"
	case path == "main.c":
		return "C main entry point convention"
	case path == "src/main.c":
		return "C src/main entry point convention"
	case path == "main.cpp" || path == "main.cc" || path == "main.cxx":
		return "C++ main entry point convention"
	case path == "src/main.cpp" || path == "src/main.cc" || path == "src/main.cxx":
		return "C++ src/main entry point convention"
	default:
		return ""
	}
}
