// Program: soft_DNA - store_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests the SQLite index: full-text search over sample projects, natural-language queries, idempotent rebuilds, symbol indexing, and repeated schema creation.

package index

import (
	"path/filepath"
	"reflect"
	"testing"

	"codelens-local/internal/scanner"
	"codelens-local/internal/symbols"
)

// TestStoreBuildsFTSIndex indexes the sample Python project and checks that a keyword search returns the expected file.
func TestStoreBuildsFTSIndex(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t)
	if err := store.Rebuild(root, files); err != nil {
		t.Fatal(err)
	}

	hits, err := store.Search("hello", 10)
	if err != nil {
		t.Fatal(err)
	}

	want := []Hit{{Path: "main.py"}}
	if !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits = %#v, want %#v", hits, want)
	}
}

// TestSearchAcceptsNaturalLanguageQuery checks that a plain-English question is turned into a working search and returns the expected file.
func TestSearchAcceptsNaturalLanguageQuery(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t)
	if err := store.Rebuild(root, files); err != nil {
		t.Fatal(err)
	}

	hits, err := store.Search("what does hello do?", 10)
	if err != nil {
		t.Fatal(err)
	}

	want := []Hit{{Path: "main.py"}}
	if !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits = %#v, want %#v", hits, want)
	}
}

// TestStoreRebuildIsIdempotent rebuilds the same index repeatedly and checks that the stored counts do not grow.
func TestStoreRebuildIsIdempotent(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_node")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t)
	if err := store.Rebuild(root, files); err != nil {
		t.Fatal(err)
	}
	firstFiles, firstSearchRows, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Rebuild(root, files); err != nil {
		t.Fatal(err)
	}
	secondFiles, secondSearchRows, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}

	if firstFiles != secondFiles || firstSearchRows != secondSearchRows {
		t.Fatalf("counts changed: first=(%d,%d) second=(%d,%d)", firstFiles, firstSearchRows, secondFiles, secondSearchRows)
	}
	if secondFiles != len(files) || secondSearchRows != len(files) {
		t.Fatalf("counts = (%d,%d), want %d", secondFiles, secondSearchRows, len(files))
	}
}

// TestStoreIndexesSymbolsForSearch checks that extracted symbols are stored and counted so they can be found by search.
func TestStoreIndexesSymbolsForSearch(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "sample_python")
	files, err := scanner.Scan(root, scanner.Options{})
	if err != nil {
		t.Fatal(err)
	}
	symbolList := symbols.ExtractAll(root, files)

	store := openTestStore(t)
	if err := store.RebuildWithSymbols(root, files, symbolList); err != nil {
		t.Fatal(err)
	}

	count, err := store.SymbolCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("symbol count = %d, want 1", count)
	}

	hits, err := store.Search("hello", 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []Hit{{Path: "main.py"}}
	if !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits = %#v, want %#v", hits, want)
	}
}

// TestStoreSchemaCanInitializeTwice checks that calling Init more than once on the same database succeeds.
func TestStoreSchemaCanInitializeTwice(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "codelens.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

// openTestStore opens a fresh database in a temporary folder and initialises its schema for a test.
func openTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "codelens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	})

	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}
