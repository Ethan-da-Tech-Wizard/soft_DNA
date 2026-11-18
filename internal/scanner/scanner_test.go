// Program: soft_DNA - scanner_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests the file scanner: deterministic inventories, ignore rules, skipped large, binary and symlinked files, language detection, and the sample fixtures.

package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestScanInventoriesTextFilesDeterministically checks that scanning the same project twice gives identical, sorted results.
func TestScanInventoriesTextFilesDeterministically(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b.py", "print('b')\n")
	writeFile(t, root, "a.go", "package main\n\nfunc main() {}\n")
	writeFile(t, root, "node_modules/skip.js", "skip()\n")

	first, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("scan output not deterministic:\n%#v\n%#v", first, second)
	}

	gotPaths := paths(first)
	wantPaths := []string{"a.go", "b.py"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", gotPaths, wantPaths)
	}

	if first[0].Language != "Go" || first[0].LineCount != 3 || first[0].ContentHash == "" {
		t.Fatalf("unexpected a.go info: %#v", first[0])
	}
}

// TestScanRespectsGitignoreAndExtraIgnores checks that both .gitignore patterns and caller-supplied patterns exclude files.
func TestScanRespectsGitignoreAndExtraIgnores(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "ignored.txt\nlogs/\n")
	writeFile(t, root, "keep.py", "print('keep')\n")
	writeFile(t, root, "ignored.txt", "ignored\n")
	writeFile(t, root, "logs/app.log", "ignored\n")
	writeFile(t, root, "custom.tmp", "ignored\n")

	files, err := Scan(root, Options{Ignore: []string{"*.tmp"}})
	if err != nil {
		t.Fatal(err)
	}

	gotPaths := paths(files)
	wantPaths := []string{".gitignore", "keep.py"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", gotPaths, wantPaths)
	}
}

// TestScanMarksLargeAndBinaryFilesSkipped checks that an oversized file and a binary file are listed but marked skipped with the right reason.
func TestScanMarksLargeAndBinaryFilesSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "large.txt", "123456")
	if err := os.WriteFile(filepath.Join(root, "image.bin"), []byte{0x01, 0x00, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := Scan(root, Options{MaxFileSize: 4})
	if err != nil {
		t.Fatal(err)
	}

	byPath := mapByPath(files)
	if got := byPath["large.txt"]; !got.Skipped || got.SkipReason != "max_file_size" {
		t.Fatalf("large file = %#v, want max_file_size skip", got)
	}
	if got := byPath["image.bin"]; !got.Skipped || !got.IsBinary || got.SkipReason != "binary" {
		t.Fatalf("binary file = %#v, want binary skip", got)
	}
}

// TestScanDoesNotFollowOutOfRootSymlink checks that a symlink pointing outside the root is recorded as skipped (symlink_escape) rather than read.
func TestScanDoesNotFollowOutOfRootSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "outside-link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	files, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1: %#v", len(files), files)
	}
	if got := files[0]; got.Path != "outside-link.txt" || !got.Skipped || got.SkipReason != "symlink_escape" {
		t.Fatalf("symlink info = %#v, want symlink_escape skip", got)
	}
}

// TestSampleFixturesInventory checks the exact file list found in the sample Python fixture.
func TestSampleFixturesInventory(t *testing.T) {
	files, err := Scan(filepath.Join("..", "..", "testdata", "sample_python"), Options{})
	if err != nil {
		t.Fatal(err)
	}

	gotPaths := paths(files)
	wantPaths := []string{"main.py", "requirements.txt"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("sample_python paths = %#v, want %#v", gotPaths, wantPaths)
	}
}

// TestDetectSQLLanguage checks that SQL files are labelled SQL.
func TestDetectSQLLanguage(t *testing.T) {
	got := detectLanguage(".sql", "schema.sql")
	if got != "SQL" {
		t.Errorf("detectLanguage(%q, %q) = %q, want %q", ".sql", "schema.sql", got, "SQL")
	}
}

// TestDetectSwiftLanguage checks that Swift files are labelled Swift.
func TestDetectSwiftLanguage(t *testing.T) {
	got := detectLanguage(".swift", "main.swift")
	if got != "Swift" {
		t.Errorf("detectLanguage(%q, %q) = %q, want %q", ".swift", "main.swift", got, "Swift")
	}
}

// TestDetectRLanguage checks the language labels for R file extensions.
func TestDetectRLanguage(t *testing.T) {
	cases := []struct {
		ext      string
		base     string
		wantLang string
	}{
		{".r", "main.r", "R"},
		{".r", "analysis.r", "R"},
	}
	for _, tc := range cases {
		got := detectLanguage(tc.ext, tc.base)
		if got != tc.wantLang {
			t.Errorf("detectLanguage(%q, %q) = %q, want %q", tc.ext, tc.base, got, tc.wantLang)
		}
	}
}

// TestDetectCPPAndCMakeLanguages checks the language labels for C++ extensions and for CMakeLists.txt.
func TestDetectCPPAndCMakeLanguages(t *testing.T) {
	cases := []struct {
		ext      string
		base     string
		wantLang string
	}{
		{".cpp", "main.cpp", "C++"},
		{".hpp", "tree_node.hpp", "C++"},
		{".cc", "app.cc", "C++"},
		{".cxx", "main.cxx", "C++"},
		{".h", "header.h", "C++"},
		{".c", "util.c", "C"},
		{".txt", "CMakeLists.txt", "CMake"},
	}
	for _, tc := range cases {
		got := detectLanguage(tc.ext, tc.base)
		if got != tc.wantLang {
			t.Errorf("detectLanguage(%q, %q) = %q, want %q", tc.ext, tc.base, got, tc.wantLang)
		}
	}
}

// TestScanCFixture checks the language labels of the sample C project's files.
func TestScanCFixture(t *testing.T) {
	files, err := Scan(filepath.Join("..", "..", "testdata", "sample_c"), Options{})
	if err != nil {
		t.Fatal(err)
	}

	byPath := mapByPath(files)
	if got := byPath["main.c"]; got.Language != "C" {
		t.Errorf("main.c language = %q, want C", got.Language)
	}
	if got := byPath["utils.c"]; got.Language != "C" {
		t.Errorf("utils.c language = %q, want C", got.Language)
	}
	if got := byPath["utils.h"]; got.Language != "C++" {
		t.Errorf("utils.h language = %q, want C++", got.Language)
	}
}

// TestScanCPPFixture checks the language labels of the sample C++ project's files.
func TestScanCPPFixture(t *testing.T) {
	files, err := Scan(filepath.Join("..", "..", "testdata", "sample_cpp"), Options{})
	if err != nil {
		t.Fatal(err)
	}

	byPath := mapByPath(files)
	if got := byPath["main.cpp"]; got.Language != "C++" {
		t.Errorf("main.cpp language = %q, want C++", got.Language)
	}
	if got := byPath["tree_node.hpp"]; got.Language != "C++" {
		t.Errorf("tree_node.hpp language = %q, want C++", got.Language)
	}
	if got := byPath["CMakeLists.txt"]; got.Language != "CMake" {
		t.Errorf("CMakeLists.txt language = %q, want CMake", got.Language)
	}
}

// paths returns the Path of each FileInfo, in order.
func paths(files []FileInfo) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
}

// mapByPath indexes FileInfo values by their path for easy lookup.
func mapByPath(files []FileInfo) map[string]FileInfo {
	out := make(map[string]FileInfo, len(files))
	for _, file := range files {
		out[file.Path] = file
	}
	return out
}

// writeFile writes a file (creating folders as needed) under root for a test, failing the test on error.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestScanIgnoresToolCacheDirectories checks that tool cache folders such as the pytest, mypy and ruff caches are not scanned.
func TestScanIgnoresToolCacheDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.py", "print('hi')\n")
	for _, dir := range []string{
		".mypy_cache", ".pytest_cache", ".ruff_cache", ".tox",
		".eggs", ".claude", ".gradle", ".terraform", ".svn", ".hg",
	} {
		writeFile(t, root, dir+"/junk.py", "polluted = True\n")
	}

	files, err := Scan(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || files[0].Path != "main.py" {
		paths := make([]string, 0, len(files))
		for _, f := range files {
			paths = append(paths, f.Path)
		}
		t.Fatalf("scan picked up cache-directory contents: %v, want just main.py", paths)
	}
}
