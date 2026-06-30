package symbols

import (
	"path/filepath"
	"reflect"
	"testing"

	"codelens-local/internal/scanner"
)

func TestExtractPythonSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	got := names(ExtractAll(root, files))
	want := []string{"function:hello:main.py:1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
}

func TestExtractTypeScriptSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	got := names(ExtractAll(root, files))
	want := []string{"import:React from \"react\";:src/main.tsx:1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
}

func TestExtractGoSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_go")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	got := names(ExtractAll(root, files))
	want := []string{"function:main:cmd/app/main.go:3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
}

func TestMalformedFileDoesNotCrash(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "edge_cases")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	_ = ExtractAll(root, files)
}

func names(items []Symbol) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Kind+":"+item.Name+":"+item.File+":"+itoa(item.StartLine))
	}
	return out
}

func itoa(value int) string {
	return string(rune('0' + value))
}
