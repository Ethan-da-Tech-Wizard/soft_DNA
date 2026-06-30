package structure

import (
	"path/filepath"
	"reflect"
	"testing"

	"codelens-local/internal/scanner"
)

func TestAnalyzeFindsPythonEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	folders, entries := Analyze(files)

	if len(folders) != 1 || folders[0].Path != "." {
		t.Fatalf("folders = %#v, want root folder only", folders)
	}
	wantEntries := []EntryPoint{{Path: "main.py", Reason: "Python main script convention"}}
	if !reflect.DeepEqual(entries, wantEntries) {
		t.Fatalf("entries = %#v, want %#v", entries, wantEntries)
	}
}

func TestAnalyzeFindsNodeEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	folders, entries := Analyze(files)

	if len(folders) != 2 || folders[1].Path != "src" || folders[1].Role != "Application source" {
		t.Fatalf("folders = %#v, want src application source", folders)
	}
	wantEntries := []EntryPoint{{Path: "src/main.tsx", Reason: "Frontend main module convention"}}
	if !reflect.DeepEqual(entries, wantEntries) {
		t.Fatalf("entries = %#v, want %#v", entries, wantEntries)
	}
}

func TestAnalyzeFindsGoCmdEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_go")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	wantEntries := []EntryPoint{{Path: "cmd/app/main.go", Reason: "Go cmd/*/main.go convention"}}
	if !reflect.DeepEqual(entries, wantEntries) {
		t.Fatalf("entries = %#v, want %#v", entries, wantEntries)
	}
}
