package symbols

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
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
		out = append(out, item.Kind+":"+item.Name+":"+item.File+":"+strconv.Itoa(item.StartLine))
	}
	return out
}

// writeFixture writes src into a temp root and returns the root plus the
// FileInfo needed to call Extract directly.
func writeFixture(t *testing.T, name, src string) (string, scanner.FileInfo) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path == name {
			return root, f
		}
	}
	t.Fatalf("fixture %s not scanned", name)
	return "", scanner.FileInfo{}
}

func TestExtractPythonMultilineAndDecorators(t *testing.T) {
	src := `import os

@app.route("/things")
def handler(request,
            response,
            timeout=30):
    pass

class Config:
    pass
`
	root, file := writeFixture(t, "svc.py", src)
	syms := Extract(root, file)

	var handler *Symbol
	for i := range syms {
		if syms[i].Name == "handler" {
			handler = &syms[i]
		}
		if strings.HasPrefix(syms[i].Name, "@") || strings.Contains(syms[i].Name, "route") {
			t.Fatalf("decorator leaked into symbols: %+v", syms[i])
		}
	}
	if handler == nil {
		t.Fatalf("multiline def not extracted: %+v", syms)
	}
	if handler.Kind != "function" || handler.StartLine != 4 || handler.EndLine != 6 {
		t.Fatalf("handler = %+v, want function spanning lines 4-6", *handler)
	}
	if !strings.Contains(handler.Signature, "timeout=30") {
		t.Fatalf("signature not joined across lines: %q", handler.Signature)
	}

	foundConfig := false
	for _, s := range syms {
		if s.Name == "Config" && s.Kind == "class" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Fatalf("class Config missing: %+v", syms)
	}
}

func TestExtractJSMultilineFunction(t *testing.T) {
	src := `export async function fetchData(url,
                                 options,
                                 retries = 3) {
  return null;
}
`
	root, file := writeFixture(t, "api.js", src)
	syms := Extract(root, file)

	if len(syms) != 1 || syms[0].Name != "fetchData" || syms[0].Kind != "function" {
		t.Fatalf("symbols = %+v, want single function fetchData", syms)
	}
	if syms[0].StartLine != 1 || syms[0].EndLine != 3 {
		t.Fatalf("fetchData lines = %d-%d, want 1-3", syms[0].StartLine, syms[0].EndLine)
	}
	if !strings.Contains(syms[0].Signature, "retries = 3") {
		t.Fatalf("signature not joined across lines: %q", syms[0].Signature)
	}
}

func TestExtractRustAttributesAndMultiline(t *testing.T) {
	src := `use std::fmt;

#[derive(Debug, Clone)]
struct Point { x: i32, y: i32 }

#[inline]
pub fn add(
    a: i32,
    b: i32,
) -> i32 {
    a + b
}
`
	root, file := writeFixture(t, "lib.rs", src)
	syms := Extract(root, file)

	byKindName := map[string]bool{}
	var add *Symbol
	for i := range syms {
		byKindName[syms[i].Kind+":"+syms[i].Name] = true
		if syms[i].Name == "add" {
			add = &syms[i]
		}
	}
	for _, want := range []string{"import:std::fmt", "type:Point", "function:add"} {
		if !byKindName[want] {
			t.Fatalf("missing %q; got %+v", want, syms)
		}
	}
	if add.StartLine != 7 || add.EndLine != 10 {
		t.Fatalf("add lines = %d-%d, want 7-10", add.StartLine, add.EndLine)
	}
	if !strings.Contains(add.Signature, "b: i32") {
		t.Fatalf("signature not joined across lines: %q", add.Signature)
	}
}

func TestExtractCRejectsStatementsAndPrototypes(t *testing.T) {
	src := `#include <stdio.h>

int add(int a, int b);

int mul(int a, int b) {
    int result = add(a, b);
    return result;
}
`
	root, file := writeFixture(t, "math.c", src)
	syms := Extract(root, file)

	byKindName := map[string]bool{}
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}
	if !byKindName["function:mul"] || !byKindName["import:stdio.h"] {
		t.Fatalf("expected definition and include; got %+v", syms)
	}
	if byKindName["function:add"] {
		t.Fatalf("prototype/call statement leaked as function: %+v", syms)
	}
}

func TestExtractCPPRejectsStatements(t *testing.T) {
	src := `#include <vector>

void apply(int value);

int compute(int value) {
    std::vector<int> items(value);
    return helper(value);
}
`
	root, file := writeFixture(t, "calc.cpp", src)
	syms := Extract(root, file)

	byKindName := map[string]bool{}
	for _, s := range syms {
		byKindName[s.Kind+":"+s.Name] = true
	}
	if !byKindName["function:compute"] {
		t.Fatalf("definition missing; got %+v", syms)
	}
	for _, reject := range []string{"function:apply", "function:items", "function:helper", "method:helper"} {
		if byKindName[reject] {
			t.Fatalf("statement %q leaked as symbol: %+v", reject, syms)
		}
	}
}

func BenchmarkExtractAll(b *testing.B) {
	root := filepath.Join("..", "..", "testdata", "sample_go")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		b.Fatal(err)
	}

	largeFileList := make([]scanner.FileInfo, 0, len(files)*50)
	for i := 0; i < 50; i++ {
		largeFileList = append(largeFileList, files...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractAll(root, largeFileList)
	}
}

func TestIncrementalRescanHashes(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_go")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	allSyms := ExtractAll(root, files)
	if len(allSyms) == 0 {
		t.Fatal("expected symbols in sample_go")
	}

	cachedHashes := map[string]string{
		files[0].Path: files[0].ContentHash,
	}
	cachedSymbols := map[string][]Symbol{
		files[0].Path: {
			{Name: "CachedSymbol", Kind: "function", File: files[0].Path, StartLine: 42},
		},
	}

	incSyms := ExtractAllIncremental(root, files, cachedHashes, cachedSymbols)

	foundCached := false
	for _, sym := range incSyms {
		if sym.Name == "CachedSymbol" {
			foundCached = true
			break
		}
	}
	if !foundCached {
		t.Errorf("expected cached symbol to be reused, got: %+v", incSyms)
	}
}
