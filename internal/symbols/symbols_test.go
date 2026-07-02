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

func TestExtractSQLSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_sql")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"table:users",
		"table:orders",
		"view:active_users",
		"procedure:clear_orders",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractSwiftSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_swift")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:Foundation",
		"import:PackageDescription",
		"type:Point",
		"function:distance",
		"function:main",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractRSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_r")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:ggplot2",
		"import:dplyr",
		"function:greet",
		"function:add",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractRubySymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_ruby")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:json",
		"import:utils",
		"class:Greeter",
		"class:App",
		"method:greet",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractRustSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_rust")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:std::io",
		"type:Point",
		"function:add",
		"function:main",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractCSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_c")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:stdio.h",
		"import:utils.h",
		"function:main",
		"type:Point",
		"function:add",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
}

func TestExtractCPPSymbols(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_cpp")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	syms := ExtractAll(root, files)
	byKindName := make(map[string]bool, len(syms))
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}

	for _, want := range []string{
		"import:iostream",
		"import:tree_node.hpp",
		"class:App",
		"function:main",
		"import:vector",
		"class:TreeNode",
		"function:sum",
	} {
		if !byKindName[want] {
			t.Fatalf("missing symbol %q; got %v", want, syms)
		}
	}
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
