package stackdetect

import (
	"os"
	"path/filepath"
	"testing"

	"codelens-local/internal/scanner"
)

func TestDetectPythonStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	if report.Summary != "Python" {
		t.Fatalf("summary = %q, want Python", report.Summary)
	}
	if len(report.Tech) != 1 || report.Tech[0].EvidenceFile != "requirements.txt" {
		t.Fatalf("tech = %#v, want requirements evidence", report.Tech)
	}
}

func TestDetectNodeStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	want := "Node.js + React + TypeScript + Vite"
	if report.Summary != want {
		t.Fatalf("summary = %q, want %q", report.Summary, want)
	}
}

func TestDetectUnknownStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_empty")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	if report.Summary != "No stack manifest detected" {
		t.Fatalf("summary = %q, want no manifest message", report.Summary)
	}
	if len(report.Tech) != 0 {
		t.Fatalf("tech = %#v, want empty", report.Tech)
	}
}

func TestDetectRubyStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_ruby")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Ruby"]; got.Category != "language" || got.EvidenceFile != "Gemfile" {
		t.Fatalf("Ruby tech = %#v, want language from Gemfile", got)
	}
	if got := byName["Bundler"]; got.Category != "tool" || got.EvidenceFile != "Gemfile" {
		t.Fatalf("Bundler tech = %#v, want tool from Gemfile", got)
	}
}

func TestDetectRustStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_rust")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Rust"]; got.Category != "language" || got.EvidenceFile != "Cargo.toml" {
		t.Fatalf("Rust tech = %#v, want language from Cargo.toml", got)
	}
	if got := byName["Cargo"]; got.Category != "tool" || got.EvidenceFile != "Cargo.toml" {
		t.Fatalf("Cargo tech = %#v, want tool from Cargo.toml", got)
	}
}

func TestDetectCStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_c")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["C"]; got.Category != "language" || got.EvidenceFile != "Makefile" {
		t.Fatalf("C tech = %#v, want language from Makefile", got)
	}
	if got := byName["Make"]; got.Category != "tool" || got.EvidenceFile != "Makefile" {
		t.Fatalf("Make tech = %#v, want tool from Makefile", got)
	}
}

func TestDetectPyprojectWithPoetry(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.poetry]\nname = \"myproject\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "poetry.lock"), []byte("# poetry lock file\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Python"]; got.Category != "language" || got.EvidenceFile != "pyproject.toml" {
		t.Fatalf("Python tech = %#v, want language from pyproject.toml", got)
	}
	if got := byName["pyproject"]; got.Category != "tool" || got.EvidenceFile != "pyproject.toml" {
		t.Fatalf("pyproject tech = %#v, want tool from pyproject.toml", got)
	}
	if got := byName["Poetry"]; got.Category != "tool" || got.EvidenceFile != "poetry.lock" {
		t.Fatalf("Poetry tech = %#v, want tool from poetry.lock", got)
	}
}

func TestDetectPyprojectWithPipenv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname = \"myproject\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Pipfile"), []byte("# Pipfile\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Python"]; got.Category != "language" || got.EvidenceFile != "pyproject.toml" {
		t.Fatalf("Python tech = %#v, want language from pyproject.toml", got)
	}
	if got := byName["Pipenv"]; got.Category != "tool" || got.EvidenceFile != "Pipfile" {
		t.Fatalf("Pipenv tech = %#v, want tool from Pipfile", got)
	}
}

func TestDetectSQLStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_sql")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["SQL"]; got.Category != "database" || got.EvidenceFile != "schema.sql" {
		t.Fatalf("SQL tech = %#v, want database from schema.sql", got)
	}
}

func TestDetectSwiftStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_swift")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Swift"]; got.Category != "language" || got.EvidenceFile != "Package.swift" {
		t.Fatalf("Swift tech = %#v, want language from Package.swift", got)
	}
	if got := byName["Swift Package Manager"]; got.Category != "tool" || got.EvidenceFile != "Package.swift" {
		t.Fatalf("Swift Package Manager tech = %#v, want tool from Package.swift", got)
	}
}

func TestDetectRStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_r")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["R"]; got.Category != "language" || got.EvidenceFile != "DESCRIPTION" {
		t.Fatalf("R tech = %#v, want language from DESCRIPTION", got)
	}
	if got := byName["Rscript"]; got.Category != "tool" || got.EvidenceFile != "DESCRIPTION" {
		t.Fatalf("Rscript tech = %#v, want tool from DESCRIPTION", got)
	}
}

func TestDetectYarnStack(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), `{"name":"myapp"}`)
	mustWriteFile(t, filepath.Join(root, "yarn.lock"), "# yarn lockfile v1\n")

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Yarn"]; got.Category != "tool" || got.EvidenceFile != "yarn.lock" {
		t.Fatalf("Yarn tech = %#v, want tool from yarn.lock", got)
	}
	// Node.js must still be detected from package.json.
	if got := byName["Node.js"]; got.Category != "runtime" || got.EvidenceFile != "package.json" {
		t.Fatalf("Node.js tech = %#v, want runtime from package.json", got)
	}
}

func TestDetectPnpmStack(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), `{"name":"myapp"}`)
	mustWriteFile(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '6.0'\n")

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["pnpm"]; got.Category != "tool" || got.EvidenceFile != "pnpm-lock.yaml" {
		t.Fatalf("pnpm tech = %#v, want tool from pnpm-lock.yaml", got)
	}
	if got := byName["Node.js"]; got.Category != "runtime" || got.EvidenceFile != "package.json" {
		t.Fatalf("Node.js tech = %#v, want runtime from package.json", got)
	}
}

func TestDetectBunStack(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), `{"name":"myapp"}`)
	mustWriteFile(t, filepath.Join(root, "bun.lockb"), "") // binary format; content is irrelevant

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["Bun"]; got.Category != "runtime" || got.EvidenceFile != "bun.lockb" {
		t.Fatalf("Bun tech = %#v, want runtime from bun.lockb", got)
	}
	if got := byName["Node.js"]; got.Category != "runtime" || got.EvidenceFile != "package.json" {
		t.Fatalf("Node.js tech = %#v, want runtime from package.json", got)
	}
}

// mustWriteFile is a test helper that fatals if the file cannot be written.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestDetectCPPStack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_cpp")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	if report.Summary != "C++ + CMake" {
		t.Fatalf("summary = %q, want \"C++ + CMake\"", report.Summary)
	}

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if got := byName["C++"]; got.Category != "language" || got.EvidenceFile != "CMakeLists.txt" {
		t.Fatalf("C++ tech = %#v, want language from CMakeLists.txt", got)
	}
	if got := byName["CMake"]; got.Category != "tool" || got.EvidenceFile != "CMakeLists.txt" {
		t.Fatalf("CMake tech = %#v, want tool from CMakeLists.txt", got)
	}
}

func TestDetectMakefileWithoutCSources(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module myapp\n")
	mustWriteFile(t, filepath.Join(root, "Makefile"), "build:\n\tgo build\n")

	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	report := Detect(files, root)

	byName := make(map[string]Tech, len(report.Tech))
	for _, tech := range report.Tech {
		byName[tech.Name] = tech
	}

	if _, hasC := byName["C"]; hasC {
		t.Fatalf("falsely detected C language from Makefile without C sources")
	}

	if got := byName["Go"]; got.Category != "language" || got.EvidenceFile != "go.mod" {
		t.Fatalf("Go tech = %#v, want language from go.mod", got)
	}

	if got := byName["Make"]; got.Category != "tool" || got.EvidenceFile != "Makefile" {
		t.Fatalf("Make tech = %#v, want tool from Makefile", got)
	}
}
