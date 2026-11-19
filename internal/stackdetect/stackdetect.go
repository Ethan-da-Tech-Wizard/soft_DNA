// ==============================================================================
// Program: stackdetect.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Inspects project manifests (package.json, go.mod, Cargo.toml, requirements.txt, etc.) to detect technology stacks, framework dependencies, runtime versions, and system architecture.
// ==============================================================================

// Package stackdetect infers a project stack from manifest files.
package stackdetect

// File role: answers "what is this project built with?" from manifest files only (go.mod, Cargo.toml,
// package.json, ...), never from file contents or by running anything. Called by cli.RunPipeline; the result
// feeds the report's tech-stack table and summary line, the Mermaid diagram, and the setup-command guess in cli.setupCommands.
//
// Design decision: evidence-based. Every Tech entry names the manifest that proves it (EvidenceFile), so a
// reader can verify each claim. Source files with no manifest do not produce a Tech entry (a lone .py file is
// not enough to claim "Python"), which keeps false positives out.
//
// Known limits: only the project root is examined (nested manifests in sub-folders are ignored, except the
// migrations/ folder check); versions are not reported. Tests: stackdetect_test.go.
import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codelens-local/internal/scanner"
)

// Tech is one detected technology with evidence.
// Category is one of: language, tool, runtime, framework, database. Name is a display name ("Node.js", "pnpm").
type Tech struct {
	Name         string
	Category     string
	EvidenceFile string
}

// Report summarizes detected technologies.
// Summary is the names joined with " + " (for example "Go + Make"); Tech is sorted so the report is reproducible.
type Report struct {
	Summary string
	Tech    []Tech
}

// Detect returns manifest-based stack evidence.
// Checks a fixed list of well-known file names at the root of the scanned tree. Complexity: O(files) to build the lookup map plus constant work per rule.
func Detect(files []scanner.FileInfo, root string) Report {
	// Deduplicate detected frameworks across manifest paths using canonical name keys
	// Keyed by category:name:evidence so the same fact found twice collapses to one entry, while the same technology proven by two different manifests (Python via requirements.txt and pyproject.toml) is kept twice.
	seen := make(map[string]Tech)
	// Index files into hash table for O(1) existence lookups by relative path
	// Paths are relative to the root with forward slashes, so a lookup like paths["go.mod"] means "go.mod in the project root".
	paths := make(map[string]scanner.FileInfo, len(files))
	for _, file := range files {
		paths[file.Path] = file
	}

	// Passed down to detectPackageJSON, so the manifest-specific code reports through the same de-duplication.
	add := func(name, category, evidence string) {
		key := category + ":" + name + ":" + evidence
		seen[key] = Tech{Name: name, Category: category, EvidenceFile: evidence}
	}

	// A Makefile alone does not prove C (it could build anything); C is claimed only when the scanner also saw at least one .c file.
	hasC := false
	for _, file := range files {
		if file.Language == "C" {
			hasC = true
			break
		}
	}

	// Rules below: one `if` per manifest. Each adds the language and/or tool it implies. Order does not matter because the result is sorted at the end.
	if _, ok := paths["go.mod"]; ok {
		add("Go", "language", "go.mod")
	}
	if _, ok := paths["Makefile"]; ok {
		if hasC {
			add("C", "language", "Makefile")
		}
		add("Make", "tool", "Makefile")
	}
	if _, ok := paths["configure.ac"]; ok {
		if hasC {
			add("C", "language", "configure.ac")
		}
		add("Make", "tool", "configure.ac")
	}
	if _, ok := paths["requirements.txt"]; ok {
		add("Python", "language", "requirements.txt")
	}
	if _, ok := paths["pyproject.toml"]; ok {
		add("Python", "language", "pyproject.toml")
		add("pyproject", "tool", "pyproject.toml")
	}
	if _, ok := paths["poetry.lock"]; ok {
		add("Poetry", "tool", "poetry.lock")
	}
	if _, ok := paths["Pipfile"]; ok {
		add("Pipenv", "tool", "Pipfile")
	}
	if _, ok := paths["Cargo.toml"]; ok {
		add("Rust", "language", "Cargo.toml")
		add("Cargo", "tool", "Cargo.toml")
	}
	if _, ok := paths["schema.sql"]; ok {
		add("SQL", "database", "schema.sql")
	}

	// Any file under migrations/ is taken as evidence of a SQL database; one match is enough, so the loop stops at the first.
	for p := range paths {
		if strings.HasPrefix(p, "migrations/") {
			add("SQL", "database", p)
			break
		}
	}
	if _, ok := paths["Package.swift"]; ok {
		add("Swift", "language", "Package.swift")
		add("Swift Package Manager", "tool", "Package.swift")
	}
	if _, ok := paths["DESCRIPTION"]; ok {
		add("R", "language", "DESCRIPTION")
		add("Rscript", "tool", "DESCRIPTION")
	}
	if _, ok := paths["Gemfile"]; ok {
		add("Ruby", "language", "Gemfile")
		add("Bundler", "tool", "Gemfile")
	}
	if _, ok := paths["Rakefile"]; ok {
		add("Ruby", "language", "Rakefile")
		add("Rake", "tool", "Rakefile")
	}
	if _, ok := paths["Dockerfile"]; ok {
		add("Docker", "tool", "Dockerfile")
	}
	if _, ok := paths["tsconfig.json"]; ok {
		add("TypeScript", "language", "tsconfig.json")
	}
	if _, ok := paths["CMakeLists.txt"]; ok {
		add("C++", "language", "CMakeLists.txt")
		add("CMake", "tool", "CMakeLists.txt")
	}

	// package.json implies the Node.js runtime; its dependency lists are then read for frameworks.
	if _, ok := paths["package.json"]; ok {
		add("Node.js", "runtime", "package.json")
		detectPackageJSON(root, add)
	}
	if _, ok := paths["yarn.lock"]; ok {
		add("Yarn", "tool", "yarn.lock")
	}
	if _, ok := paths["pnpm-lock.yaml"]; ok {
		add("pnpm", "tool", "pnpm-lock.yaml")
	}
	if _, ok := paths["bun.lockb"]; ok {
		add("Bun", "runtime", "bun.lockb")
	}

	// Map iteration order is random in Go, so entries are copied out and sorted by (category, name, evidence file) for stable output.
	tech := make([]Tech, 0, len(seen))
	for _, item := range seen {
		tech = append(tech, item)
	}
	sort.Slice(tech, func(i, j int) bool {
		if tech[i].Category != tech[j].Category {
			return tech[i].Category < tech[j].Category
		}
		if tech[i].Name != tech[j].Name {
			return tech[i].Name < tech[j].Name
		}
		return tech[i].EvidenceFile < tech[j].EvidenceFile
	})

	return Report{
		Summary: summary(tech),
		Tech:    tech,
	}
}

// Reads the root package.json and reports React, Next.js, Vite and TypeScript when listed as dependencies. A missing or malformed file is silently ignored (the Node.js entry is already recorded by the caller).
func detectPackageJSON(root string, add func(string, string, string)) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}

	// Only the two dependency maps are decoded; every other JSON field is ignored by encoding/json.
	var manifest struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}

	// dependencies and devDependencies are merged because a framework counts regardless of which list names it.
	deps := make(map[string]string)
	for name, version := range manifest.Dependencies {
		deps[name] = version
	}
	for name, version := range manifest.DevDependencies {
		deps[name] = version
	}

	if _, ok := deps["react"]; ok {
		add("React", "framework", "package.json")
	}
	if _, ok := deps["next"]; ok {
		add("Next.js", "framework", "package.json")
	}
	if _, ok := deps["vite"]; ok {
		add("Vite", "tool", "package.json")
	}
	if _, ok := deps["typescript"]; ok {
		add("TypeScript", "language", "package.json")
	}
}

// Unique technology names, sorted alphabetically and joined with " + ". Sorted by name (not by category) so the line reads the same however many manifests prove a technology.
func summary(tech []Tech) string {
	if len(tech) == 0 {
		return "No stack manifest detected"
	}

	names := make([]string, 0, len(tech))
	seen := make(map[string]struct{}, len(tech))
	for _, item := range tech {
		if _, ok := seen[item.Name]; ok {
			continue
		}
		seen[item.Name] = struct{}{}
		names = append(names, item.Name)
	}
	sort.Strings(names)
	return strings.Join(names, " + ")
}
