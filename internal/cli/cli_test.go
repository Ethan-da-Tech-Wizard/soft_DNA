package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
