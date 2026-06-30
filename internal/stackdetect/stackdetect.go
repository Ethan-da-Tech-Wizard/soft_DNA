// Package stackdetect infers a project stack from manifest files.
package stackdetect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codelens-local/internal/scanner"
)

// Tech is one detected technology with evidence.
type Tech struct {
	Name         string
	Category     string
	EvidenceFile string
}

// Report summarizes detected technologies.
type Report struct {
	Summary string
	Tech    []Tech
}

// Detect returns manifest-based stack evidence.
func Detect(files []scanner.FileInfo, root string) Report {
	seen := make(map[string]Tech)
	paths := make(map[string]scanner.FileInfo, len(files))
	for _, file := range files {
		paths[file.Path] = file
	}

	add := func(name, category, evidence string) {
		key := category + ":" + name + ":" + evidence
		seen[key] = Tech{Name: name, Category: category, EvidenceFile: evidence}
	}

	if _, ok := paths["go.mod"]; ok {
		add("Go", "language", "go.mod")
	}
	if _, ok := paths["requirements.txt"]; ok {
		add("Python", "language", "requirements.txt")
	}
	if _, ok := paths["pyproject.toml"]; ok {
		add("Python", "language", "pyproject.toml")
		add("pyproject", "tool", "pyproject.toml")
	}
	if _, ok := paths["Cargo.toml"]; ok {
		add("Rust", "language", "Cargo.toml")
		add("Cargo", "tool", "Cargo.toml")
	}
	if _, ok := paths["Dockerfile"]; ok {
		add("Docker", "tool", "Dockerfile")
	}
	if _, ok := paths["tsconfig.json"]; ok {
		add("TypeScript", "language", "tsconfig.json")
	}
	if _, ok := paths["package.json"]; ok {
		add("Node.js", "runtime", "package.json")
		detectPackageJSON(root, add)
	}

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

func detectPackageJSON(root string, add func(string, string, string)) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}

	var manifest struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}

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
