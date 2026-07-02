package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codelens-local/internal/grounding"
	"codelens-local/internal/llm"
	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
)

func TestRunWritesReportForDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.py"), []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	reportPath := filepath.Join(outDir, reportName)
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected report at %s: %v", reportPath, err)
	}

	report := string(content)
	if !strings.Contains(report, "# CodeLens Local Report") {
		t.Fatalf("report missing header: %q", report)
	}
	if _, err := os.Stat(filepath.Join(outDir, dbName)); err != nil {
		t.Fatalf("expected sqlite db: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "architecture.mmd")); err != nil {
		t.Fatalf("expected mermaid diagram: %v", err)
	}
	if !strings.Contains(report, "## 2. Detected Tech Stack") {
		t.Fatalf("report missing stack section: %q", report)
	}
	if !strings.Contains(report, "| main.py | Python | 1 |") {
		t.Fatalf("report missing inventory row: %q", report)
	}
	if !strings.Contains(report, "Status: M5 local retrieval Q&A.") {
		t.Fatalf("report missing M5 status: %q", report)
	}
}

func TestRunPopulatesFullReportSections(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	for _, want := range []string{
		"Detected stack: Node.js + React + TypeScript + Vite.",
		"| src | Application source | 1 |",
		"- package.json",
		"| src/main.tsx | Frontend main module convention |",
		"## 6. Functions & Classes",
		"## 7. Imports/Dependencies",
		"| src/main.tsx | import | React from \"react\"; | 1 |",
		"## 10. Risky/Confusing Areas",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunFullReportGoldenPrefix(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}

	prefix := strings.Split(string(content), "## Scanner Inventory")[0]
	want := `# CodeLens Local Report

Target: ../../testdata/sample_node

Status: M5 local retrieval Q&A. Model-based chat is not implemented.

## 1. Project Summary

Scanned 3 files. Detected stack: Node.js + React + TypeScript + Vite.

Index: ` + filepath.Join(outDir, dbName) + ` (3 files, 1 symbols, 3 search rows).
Diagram: ` + filepath.Join(outDir, "architecture.mmd") + `.

## 2. Detected Tech Stack

| Technology | Category | Evidence |
|---|---|---|
| React | framework | package.json |
| TypeScript | language | package.json |
| TypeScript | language | tsconfig.json |
| Node.js | runtime | package.json |
| Vite | tool | package.json |

## 3. Folder Structure

| Folder | Role | Files |
|---|---|---:|
| . | Root files and project manifests | 2 |
| src | Application source | 1 |

## 4. Important Files

- package.json
- src/main.tsx
- tsconfig.json

## 5. Entry Points

| Path | Reason |
|---|---|
| src/main.tsx | Frontend main module convention |

## 6. Functions & Classes

No symbols identified.

## 7. Imports/Dependencies

| File | Kind | Name | Line | Signature |
|---|---|---|---:|---|
| src/main.tsx | import | React from "react"; | 1 |  |

## 8. Setup Guess

- ` + "`" + `npm install` + "`" + `
- ` + "`" + `npm run dev` + "`" + `

## 9. Plain-English Explanation

This project appears to use Node.js + React + TypeScript + Vite. Start with the entry points above, then read the important files and top-level folders in order.

## 10. Risky/Confusing Areas

No scanner-level risks identified.

## 11. AI-Assisted Summaries

*AI summaries disabled or unavailable. Run with --llm flag to generate.*

`
	if prefix != want {
		t.Fatalf("full report prefix mismatch\n got:\n%s\nwant:\n%s", prefix, want)
	}
}

func TestRunWritesLocalAnswer(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, "--ask", "what does hello do?", root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, answerName))
	if err != nil {
		t.Fatal(err)
	}
	answer := string(content)
	for _, want := range []string{
		"# CodeLens Local Answer",
		"Question: what does hello do?",
		"Mode: local FTS retrieval only.",
		"- main.py",
	} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer missing %q:\n%s", want, answer)
		}
	}
}

func TestRunAnswerPrefersEntryPointQuestions(t *testing.T) {
	root := filepath.Join("..", "..")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, "--ask", "where does the CLI start?", root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, answerName))
	if err != nil {
		t.Fatal(err)
	}
	answer := string(content)
	if !strings.Contains(answer, "- cmd/codelens/main.go") {
		t.Fatalf("answer should cite CLI entry point first:\n%s", answer)
	}
}

func TestRunIgnoresOutputDirectoryWhenScanningRoot(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "codelens-out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.py"), []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "old.md"), []byte("old output\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	if strings.Contains(report, "| codelens-out |") || strings.Contains(report, "old.md") {
		t.Fatalf("report should not include output directory:\n%s", report)
	}
}

func TestRunRejectsMissingPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{filepath.Join(t.TempDir(), "missing")}, &stdout, &stderr, "test")

	if code != exitBadPath {
		t.Fatalf("exit code = %d, want %d", code, exitBadPath)
	}
	if !strings.Contains(stderr.String(), "invalid path") {
		t.Fatalf("stderr = %q, want invalid path message", stderr.String())
	}
}

func TestRunRejectsFilePath(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{file}, &stdout, &stderr, "test")

	if code != exitBadPath {
		t.Fatalf("exit code = %d, want %d", code, exitBadPath)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr, "test-version")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "codelens test-version" {
		t.Fatalf("stdout = %q, want %q", got, "codelens test-version")
	}
}

func TestRunSupportsIgnoreAndMaxFileSize(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skip.tmp"), []byte("skip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.py"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, "--ignore", "*.tmp", "--max-file-size", "3", root}, &stdout, &stderr, "test")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	if strings.Contains(report, "skip.tmp") {
		t.Fatalf("ignored file appeared in report: %q", report)
	}
	if !strings.Contains(report, "large.py") || !strings.Contains(report, "skipped: max_file_size") {
		t.Fatalf("large file skip missing: %q", report)
	}
}

func TestRunRequiresSinglePath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr, "test")

	if code != exitUsage {
		t.Fatalf("exit code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr = %q, want usage message", stderr.String())
	}
}

func TestSetupCommandsPyproject(t *testing.T) {
	makePyprojectStack := func(lockfile string) stackdetect.Report {
		tech := []stackdetect.Tech{
			{Name: "Python", Category: "language", EvidenceFile: "pyproject.toml"},
			{Name: "pyproject", Category: "tool", EvidenceFile: "pyproject.toml"},
		}
		if lockfile != "" {
			tech = append(tech, stackdetect.Tech{Name: lockfile, Category: "tool", EvidenceFile: lockfile})
		}
		return stackdetect.Report{Tech: tech}
	}

	cases := []struct {
		name     string
		lockfile string
		want     []string
	}{
		{"poetry", "poetry.lock", []string{"poetry install"}},
		{"pipenv", "Pipfile", []string{"pipenv install"}},
		{"generic", "", []string{"pip install ."}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := setupCommands(makePyprojectStack(tc.lockfile))
			if len(got) != len(tc.want) {
				t.Fatalf("setupCommands(%q) = %v, want %v", tc.lockfile, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("setupCommands(%q)[%d] = %q, want %q", tc.lockfile, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSetupCommandsNodeLockfiles(t *testing.T) {
	// makeNodeStack builds a minimal Node.js stack report.
	// If lockfile is non-empty it adds a corresponding tech entry so the
	// evidence map inside setupCommands picks up the lockfile.
	makeNodeStack := func(lockfile, category string) stackdetect.Report {
		tech := []stackdetect.Tech{
			{Name: "Node.js", Category: "runtime", EvidenceFile: "package.json"},
		}
		if lockfile != "" {
			tech = append(tech, stackdetect.Tech{
				Name:         lockfile,
				Category:     category,
				EvidenceFile: lockfile,
			})
		}
		return stackdetect.Report{Tech: tech}
	}

	cases := []struct {
		name     string
		lockfile string
		category string
		wantCmds []string
	}{
		// setupCommands returns sort.Strings order, so verify alphabetical.
		{"yarn", "yarn.lock", "tool", []string{"yarn dev", "yarn install"}},
		{"pnpm", "pnpm-lock.yaml", "tool", []string{"pnpm dev", "pnpm install"}},
		{"bun", "bun.lockb", "runtime", []string{"bun install", "bun run dev"}},
		{"npm (no lockfile)", "", "", []string{"npm install", "npm run dev"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := setupCommands(makeNodeStack(tc.lockfile, tc.category))
			if len(got) != len(tc.wantCmds) {
				t.Fatalf("setupCommands(%q) = %v, want %v", tc.lockfile, got, tc.wantCmds)
			}
			for i, want := range tc.wantCmds {
				if got[i] != want {
					t.Errorf("[%d] got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestRunGeneratesRubyReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_ruby")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| Ruby | language | Gemfile |",
		"| Bundler | tool | Gemfile |",
		"| main.rb | Ruby entry point convention |",
		"| import | json |",
		"| class | Greeter |",
		"| class | App |",
		"| method | greet |",
		"bundle install",
		"bundle exec ruby app.rb",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesRustReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_rust")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| Rust | language | Cargo.toml |",
		"| Cargo | tool | Cargo.toml |",
		"| src/main.rs | Rust main/library entry point convention |",
		"| import | std::io |",
		"| type | Point |",
		"| function | add |",
		"| function | main |",
		"- `cargo build`",
		"- `cargo run`",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesCReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_c")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| C | language | Makefile |",
		"| Make | tool | Makefile |",
		"| main.c | C main entry point convention |",
		"| import | stdio.h |",
		"| function | main |",
		"| type | Point |",
		"| function | add |",
		"- `make`",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesSQLReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_sql")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| SQL | database | schema.sql |",
		"| schema.sql | SQL initialization entry point convention |",
		"| table | users |",
		"| table | orders |",
		"| view | active_users |",
		"| procedure | clear_orders |",
		"sqlite3 db.sqlite < schema.sql",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesSwiftReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_swift")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| Swift | language | Package.swift |",
		"| Swift Package Manager | tool | Package.swift |",
		"| main.swift | Swift entry point convention |",
		"| import | Foundation |",
		"| type | Point |",
		"| function | distance |",
		"| function | main |",
		"- `swift build`",
		"- `swift run`",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesRReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_r")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"| R | language | DESCRIPTION |",
		"| Rscript | tool | DESCRIPTION |",
		"| main.R | R script entry point convention |",
		"| import | ggplot2 |",
		"| import | dplyr |",
		"| function | greet |",
		"| function | add |",
		"Rscript",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunGeneratesCPPReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_cpp")
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--out", outDir, root}, &stdout, &stderr, "test")
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	for _, want := range []string{
		"Detected stack: C++ + CMake.",
		"| C++ | language | CMakeLists.txt |",
		"| CMake | tool | CMakeLists.txt |",
		"| main.cpp | C++ main entry point convention |",
		"cmake -B build",
		"cmake --build build",
		"| import | iostream |",
		"| class | App |",
		"| function | main |",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestRunLLMIntegration(t *testing.T) {
	// Create mock model file
	tmpDir := t.TempDir()
	mockModel := filepath.Join(tmpDir, "mock_model.gguf")
	if err := os.WriteFile(mockModel, []byte("mock gguf model content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create mock llama-cli script
	mockBinary := filepath.Join(tmpDir, "mock_llama_cli")
	script := `#!/bin/sh
# Simple mock script to simulate llama-cli output for different prompt types.
# It returns hardcoded JSON conforming to the respective grammar files.

# Find the prompt text in arguments
prompt=""
while [ $# -gt 0 ]; do
    case "$1" in
        -p|--prompt)
            prompt="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

if echo "$prompt" | grep -q "Project root:"; then
    # Project overview prompt
    echo '{"project_description": "This is a mock project summary from the LLM."}'
else
    # File explain prompt
    echo '{"file_summary": "Mock file explanation for unit testing.", "symbol_explanations": [{"name": "hello", "explanation": "A simple greeting function."}]}'
fi
`
	if err := os.WriteFile(mockBinary, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	root := filepath.Join("..", "..", "testdata", "sample_python")
	outDir := filepath.Join(tmpDir, "out")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"--out", outDir,
		"--llm",
		"--llm-binary", mockBinary,
		"--llm-model", mockModel,
		"--llm-max-files", "2",
		root,
	}, &stdout, &stderr, "test")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}

	reportPath := filepath.Join(outDir, reportName)
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}
	report := string(content)

	for _, want := range []string{
		"## 11. AI-Assisted Summaries",
		"*Generated with local LLM (offline).*",
		"This is a mock project summary from the LLM.",
		"#### main.py",
		"Mock file explanation for unit testing.",
		"| Symbol | Explanation |",
		"| hello | A simple greeting function. |",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing LLM output segment %q:\n%s", want, report)
		}
	}
}

func TestLLMLoopDiagnostics(t *testing.T) {
	orig := llmRun
	defer func() { llmRun = orig }()

	calls := 0
	llmRun = func(ctx context.Context, cfg llm.Config, prompt string) ([]byte, error) {
		calls++
		if calls == 1 {
			// Turn 1: unparseable output so validation fails.
			return []byte("not json at all"), nil
		}
		// Turn 2: valid output conforming to the file_explain schema.
		return []byte(`{"file_summary": "A tiny module.", "symbol_explanations": []}`), nil
	}

	fileList := []string{"main.py"}
	infoMap := map[string]scanner.FileInfo{
		"main.py": {Path: "main.py", Language: "Python", LineCount: 3},
	}
	fileSymbols := map[string][]grounding.SymbolEntry{
		"main.py": {{File: "main.py", Name: "hello", Kind: "function", StartLine: 1}},
	}

	var diagOut, warnOut bytes.Buffer
	diag := llm.NewLoopState(len(fileList), &diagOut)
	summaries, _ := summarizeFiles(fileList, infoMap, fileSymbols, llm.Config{}, diag, &warnOut)

	wantLines := []string{
		"[LLM Loop] Turn 1: Coverage = 0.00 | Throttle = 0.75 | Health = 0 | Stall = 1",
		"[LLM Loop] Turn 2: Coverage = 0.00 | Throttle = 0.50 | Health = 1 | Stall = 0",
	}
	gotLines := strings.Split(strings.TrimSpace(diagOut.String()), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("diagnostic lines = %d, want %d:\n%s", len(gotLines), len(wantLines), diagOut.String())
	}
	for i, want := range wantLines {
		if gotLines[i] != want {
			t.Errorf("line %d = %q, want %q", i+1, gotLines[i], want)
		}
	}

	if calls != 2 {
		t.Errorf("llmRun calls = %d, want 2 (one failed turn + one retry)", calls)
	}
	if fe, ok := summaries["main.py"]; !ok || fe.FileSummary != "A tiny module." {
		t.Errorf("summaries[main.py] = %+v, want validated summary", summaries["main.py"])
	}
	if diag.P != 1 || diag.Stall != 0 || diag.Rho != llm.RhoNormal {
		t.Errorf("final state P=%d Stall=%d Rho=%.2f, want P=1 Stall=0 Rho=%.2f", diag.P, diag.Stall, diag.Rho, llm.RhoNormal)
	}
}

func TestLLMLoopAtomicBlockStall(t *testing.T) {
	orig := llmRun
	defer func() { llmRun = orig }()

	calls := 0
	llmRun = func(ctx context.Context, cfg llm.Config, prompt string) ([]byte, error) {
		calls++
		return nil, errors.New("model exploded")
	}

	fileList := []string{"main.py"}
	infoMap := map[string]scanner.FileInfo{
		"main.py": {Path: "main.py", Language: "Python", LineCount: 3},
	}
	fileSymbols := map[string][]grounding.SymbolEntry{}

	var diagOut, warnOut bytes.Buffer
	diag := llm.NewLoopState(len(fileList), &diagOut)
	summaries, stalled := summarizeFiles(fileList, infoMap, fileSymbols, llm.Config{}, diag, &warnOut)

	// Fails at turns 1, 2, 3; stall exceeds the limit and forces a cutoff.
	if calls != llm.StallLimit+1 {
		t.Fatalf("llmRun calls = %d, want %d (retries bounded by stall guard)", calls, llm.StallLimit+1)
	}
	if !strings.Contains(warnOut.String(), "atomic block stall on main.py") {
		t.Errorf("missing atomic block stall warning:\n%s", warnOut.String())
	}
	if len(stalled) != 1 || stalled[0] != "main.py" {
		t.Errorf("stalled = %v, want [main.py]", stalled)
	}
	if _, ok := summaries["main.py"]; ok {
		t.Errorf("summaries should not contain main.py since it has no valid summary")
	}
	if diag.P != 1 || diag.Stall != 0 {
		t.Errorf("ForceAdvance should move past the stalled segment: P=%d Stall=%d", diag.P, diag.Stall)
	}
}

func TestRunWritesStalledFilesToReport(t *testing.T) {
	outDir := t.TempDir()
	llmRes := LLMResult{
		Enabled: true,
		Stalled: []string{"b.py", "a.py"},
	}

	err := writeReport(
		".",
		outDir,
		nil,
		stackdetect.Report{},
		nil,
		nil,
		nil,
		indexStats{},
		"",
		llmRes,
	)
	if err != nil {
		t.Fatalf("writeReport failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, reportName))
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)

	want := "*Skipped after repeated validation failures: a.py, b.py*"
	if !strings.Contains(report, want) {
		t.Fatalf("report missing stalled warning %q:\n%s", want, report)
	}
}

