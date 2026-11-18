// Program: soft_DNA - store.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Stores the scanned file inventory, symbols and full-text search content in a SQLite database (FTS5) and answers keyword searches over it.

// Package index stores scanner inventory in SQLite with FTS5 search.
package index

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"codelens-local/internal/scanner"
	"codelens-local/internal/symbols"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database.
type Store struct {
	db *sql.DB
}

// Hit is one FTS search result.
type Hit struct {
	Path string
}

// queryTermRe matches the alphanumeric words kept from a user's search question.
var queryTermRe = regexp.MustCompile(`[A-Za-z0-9_]+`)

// Open opens a SQLite database at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Init creates the M3 schema if needed.
func (s *Store) Init() error {
	_, err := s.db.Exec(schema)
	return err
}

// Rebuild replaces indexed file data for one scan.
func (s *Store) Rebuild(root string, files []scanner.FileInfo) error {
	return s.RebuildWithSymbols(root, files, nil)
}

// RebuildWithSymbols replaces indexed file and symbol data for one scan.
func (s *Store) RebuildWithSymbols(root string, files []scanner.FileInfo, symbolsList []symbols.Symbol) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// Roll back automatically on any early return; after a successful Commit this is a harmless no-op.
	defer tx.Rollback()

	// Rebuilds are full replacements: clear every table inside the same transaction before re-inserting.
	for _, stmt := range []string{
		"DELETE FROM search",
		"DELETE FROM symbols",
		"DELETE FROM files",
		"DELETE FROM tech",
		"DELETE FROM meta",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("INSERT INTO meta(key, value) VALUES('root', ?), ('schema_version', '1')", filepath.Clean(root)); err != nil {
		return err
	}

	// Group symbols by file path so each file's symbols can be inserted next to it.
	symbolsByFile := make(map[string][]symbols.Symbol)
	for _, symbol := range symbolsList {
		symbolsByFile[symbol.File] = append(symbolsByFile[symbol.File], symbol)
	}

	for _, file := range files {
		// Insert the file row, then its symbols, then one search row that reuses the file's id as the FTS rowid.
		result, err := tx.Exec(
			`INSERT INTO files(path, ext, language, size_bytes, line_count, content_hash, is_binary)
			 VALUES(?, ?, ?, ?, ?, ?, ?)`,
			file.Path,
			file.Ext,
			file.Language,
			file.SizeBytes,
			file.LineCount,
			file.ContentHash,
			boolInt(file.IsBinary),
		)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}

		for _, symbol := range symbolsByFile[file.Path] {
			if _, err := tx.Exec(
				`INSERT INTO symbols(file_id, name, kind, signature, doc, start_line, end_line)
				 VALUES(?, ?, ?, ?, ?, ?, ?)`,
				id,
				symbol.Name,
				symbol.Kind,
				symbol.Signature,
				symbol.Doc,
				symbol.StartLine,
				symbol.EndLine,
			); err != nil {
				return err
			}
		}

		content, err := searchableContent(root, file)
		if err != nil {
			return err
		}
		symbolNames, signatures, docs := searchableSymbols(symbolsByFile[file.Path])
		if _, err := tx.Exec(
			`INSERT INTO search(rowid, path, symbol_name, signature, doc, content)
			 VALUES(?, ?, ?, ?, ?, ?)`,
			id,
			file.Path,
			symbolNames,
			signatures,
			docs,
			content,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Search queries the FTS5 table.
func (s *Store) Search(query string, limit int) ([]Hit, error) {
	matchQuery := ftsQuery(query)
	if matchQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(
		`SELECT path FROM search WHERE search MATCH ? ORDER BY rank, path LIMIT ?`,
		matchQuery,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var hit Hit
		if err := rows.Scan(&hit.Path); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

// ftsQuery converts a free-text question into an FTS5 MATCH expression: lower-cased unique words of two or more characters, each quoted and joined with OR; an empty string means nothing searchable.
func ftsQuery(query string) string {
	terms := queryTermRe.FindAllString(strings.ToLower(query), -1)
	if len(terms) == 0 {
		return ""
	}

	seen := make(map[string]struct{}, len(terms))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		if len(term) < 2 {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, `"`+term+`"`)
	}
	return strings.Join(out, " OR ")
}

// Stats returns small row counts for tests and status output.
func (s *Store) Stats() (files int, searchRows int, err error) {
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&files); err != nil {
		return 0, 0, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM search").Scan(&searchRows); err != nil {
		return 0, 0, err
	}
	return files, searchRows, nil
}

// SymbolCount returns the indexed symbol row count.
func (s *Store) SymbolCount() (int, error) {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// searchableContent returns a file's text for indexing, or an empty string for skipped and binary files; the path is checked to stay inside root.
func searchableContent(root string, file scanner.FileInfo) (string, error) {
	if file.Skipped || file.IsBinary {
		return "", nil
	}

	path, err := safeJoin(root, file.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// safeJoin joins root and a relative file path into an absolute path, rejecting absolute inputs and any path that would escape root.
func safeJoin(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("indexed path must be relative")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(absRoot, filepath.FromSlash(rel))
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	inside, err := insideRoot(absRoot, absCandidate)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", fmt.Errorf("indexed path escapes root: %s", rel)
	}
	return absCandidate, nil
}

// insideRoot reports whether path is root itself or lies below it.
func insideRoot(root, path string) (bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."), nil
}

// boolInt converts a bool to 1 or 0 for storage in an INTEGER column.
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// searchableSymbols joins the names, signatures and doc strings of a file's symbols into three space-separated strings for the search table.
func searchableSymbols(items []symbols.Symbol) (names, signatures, docs string) {
	var nameParts, signatureParts, docParts []string
	for _, item := range items {
		nameParts = append(nameParts, item.Name)
		signatureParts = append(signatureParts, item.Signature)
		docParts = append(docParts, item.Doc)
	}
	return strings.Join(nameParts, " "), strings.Join(signatureParts, " "), strings.Join(docParts, " ")
}

// schema creates the tables: files, symbols (linked to files), tech, meta, and the FTS5 virtual table named search.
const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS files (
    id           INTEGER PRIMARY KEY,
    path         TEXT NOT NULL UNIQUE,
    ext          TEXT,
    language     TEXT,
    size_bytes   INTEGER,
    line_count   INTEGER,
    content_hash TEXT,
    is_binary    INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS symbols (
    id         INTEGER PRIMARY KEY,
    file_id    INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL,
    signature  TEXT,
    doc        TEXT,
    start_line INTEGER,
    end_line   INTEGER
);

CREATE TABLE IF NOT EXISTS tech (
    id            INTEGER PRIMARY KEY,
    name          TEXT,
    category      TEXT,
    evidence_file TEXT
);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT
);

CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(
    path, symbol_name, signature, doc, content
);
`
