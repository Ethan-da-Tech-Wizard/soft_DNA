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

func TestEntryReasonRuby(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"main.rb", "Ruby entry point convention"},
		{"app.rb", "Ruby entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsRubyEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_ruby")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "main.rb" && e.Reason == "Ruby entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want main.rb with Ruby entry point reason", entries)
	}
}

func TestEntryReasonSQL(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"schema.sql", "SQL initialization entry point convention"},
		{"init.sql", "SQL initialization entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsSQLEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_sql")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "schema.sql" && e.Reason == "SQL initialization entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want schema.sql with SQL initialization entry point reason", entries)
	}
}

func TestEntryReasonSwift(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"main.swift", "Swift entry point convention"},
		{"App.swift", "Swift entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsSwiftEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_swift")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "main.swift" && e.Reason == "Swift entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want main.swift with Swift entry point reason", entries)
	}
}

func TestEntryReasonR(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"main.R", "R script entry point convention"},
		{"run.R", "R script entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsREntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_r")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "main.R" && e.Reason == "R script entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want main.R with R script entry point reason", entries)
	}
}

func TestEntryReasonRust(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"src/main.rs", "Rust main/library entry point convention"},
		{"src/lib.rs", "Rust main/library entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsRustEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_rust")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "src/main.rs" && e.Reason == "Rust main/library entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want src/main.rs with Rust entry point reason", entries)
	}
}

func TestEntryReasonC(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"main.c", "C main entry point convention"},
		{"src/main.c", "C src/main entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsCEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_c")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "main.c" && e.Reason == "C main entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want main.c with C entry point reason", entries)
	}
}

func TestEntryReasonCPP(t *testing.T) {
	cases := []struct {
		path   string
		reason string
	}{
		{"main.cpp", "C++ main entry point convention"},
		{"main.cc", "C++ main entry point convention"},
		{"main.cxx", "C++ main entry point convention"},
		{"src/main.cpp", "C++ src/main entry point convention"},
		{"src/main.cc", "C++ src/main entry point convention"},
		{"src/main.cxx", "C++ src/main entry point convention"},
	}
	for _, tc := range cases {
		got := entryReason(tc.path)
		if got != tc.reason {
			t.Errorf("entryReason(%q) = %q, want %q", tc.path, got, tc.reason)
		}
	}
}

func TestAnalyzeFindsCPPEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_cpp")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_, entries := Analyze(files)

	var found bool
	for _, e := range entries {
		if e.Path == "main.cpp" && e.Reason == "C++ main entry point convention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entries = %#v, want main.cpp with C++ entry point reason", entries)
	}
}
