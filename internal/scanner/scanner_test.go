package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

func TestDetectSQLLanguage(t *testing.T) {
	got := detectLanguage(".sql", "schema.sql")
	if got != "SQL" {
		t.Errorf("detectLanguage(%q, %q) = %q, want %q", ".sql", "schema.sql", got, "SQL")
	}
}

func TestDetectSwiftLanguage(t *testing.T) {
	got := detectLanguage(".swift", "main.swift")
	if got != "Swift" {
		t.Errorf("detectLanguage(%q, %q) = %q, want %q", ".swift", "main.swift", got, "Swift")
	}
}

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

func paths(files []FileInfo) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
}

func mapByPath(files []FileInfo) map[string]FileInfo {
	out := make(map[string]FileInfo, len(files))
	for _, file := range files {
		out[file.Path] = file
	}
	return out
}

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
