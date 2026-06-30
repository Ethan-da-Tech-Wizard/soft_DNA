package stackdetect

import (
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
