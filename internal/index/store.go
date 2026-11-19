// ==============================================================================
// Program: store.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Provides persistent and in-memory indexing of repository metadata, symbol tables, and scan results to enable instant subsequent queries without re-scanning unchanged files.
// ==============================================================================

// Package index stores scanner inventory in SQLite with FTS5 search.
package index

// File role: persistence and search. `cli.buildIndex` calls Open/Init/RebuildWithSymbols after a scan; the --ask
// path calls Search; grounding.Build reads the same database to give the LLM verified facts.
//
// Storage: one SQLite file (<out>/codelens.db) opened through the pure-Go modernc.org/sqlite driver, so the
// binary needs no CGO and no system SQLite. Search uses an FTS5 virtual table (`search`) over path, symbol
// names, signatures, doc text and file content; the plain tables (files, symbols, tech, meta) hold structured data.
//
// Design decision: the index is rebuilt completely on every run, inside one transaction. That is simpler and
// always consistent; the incremental speed-up lives in symbol extraction, not here.
//
// Tests: store_test.go.
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
// A thin handle over *sql.DB. database/sql manages its own connection pool, so a Store may be shared between goroutines.
type Store struct {
	db *sql.DB
}

// Hit is one FTS search result.
// Only the path is returned today; callers read the file themselves (with their own path-safety checks) when they need snippets.
type Hit struct {
	Path string
}

// A "term" is a run of letters, digits or underscores; everything else (quotes, operators, punctuation) is discarded, which is what keeps user questions from injecting FTS5 syntax.
var queryTermRe = regexp.MustCompile(`[A-Za-z0-9_]+`)

// Open opens a SQLite database at path.
// Opens (and creates, if missing) the database file; the connection itself is established lazily on first use, so errors may surface later from Init.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Init creates the M3 schema if needed.
// Idempotent: every statement in `schema` uses IF NOT EXISTS, so calling it on an existing database is harmless.
func (s *Store) Init() error {
	_, err := s.db.Exec(schema)
	return err
}

// Rebuild replaces indexed file data for one scan.
// Convenience wrapper for callers that have no symbols; kept for tests and older call sites.
func (s *Store) Rebuild(root string, files []scanner.FileInfo) error {
	return s.RebuildWithSymbols(root, files, nil)
}

// RebuildWithSymbols replaces indexed file and symbol data for one scan.
// Replaces everything in one transaction: either the whole new index is committed or the old one stays untouched.
// For each file it inserts a `files` row, its `symbols` rows, and one FTS row whose rowid equals the file id so the two tables join on it.
// Errors: any SQL failure, or a file whose content cannot be read safely (see safeJoin).
func (s *Store) RebuildWithSymbols(root string, files []scanner.FileInfo, symbolsList []symbols.Symbol) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	// Rollback after a successful Commit is a harmless no-op, so this single defer covers every early `return err` below.
	defer tx.Rollback()

	// Children first (search, symbols), then files. symbols also has ON DELETE CASCADE, but an explicit order stays correct if that ever changes.
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

	// The scan root and schema version are stored so later tools can tell which project and which layout a database belongs to.
	if _, err := tx.Exec("INSERT INTO meta(key, value) VALUES('root', ?), ('schema_version', '1')", filepath.Clean(root)); err != nil {
		return err
	}

	// Group once so each file's symbols are found by map lookup instead of scanning the whole list for every file.
	symbolsByFile := make(map[string][]symbols.Symbol)
	for _, symbol := range symbolsList {
		symbolsByFile[symbol.File] = append(symbolsByFile[symbol.File], symbol)
	}

	for _, file := range files {
		// The new row's id is needed as the foreign key for symbols and as the FTS rowid.
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

		// The FTS row stores the full file text, which is why the database can be as large as the project's source.
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
// Returns up to `limit` paths ordered by FTS5 rank (best first), then path for ties so results are deterministic.
// An empty or all-punctuation query returns no hits (not an error). limit <= 0 means 10.
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

// Turns free text into a safe FTS5 expression: unique lower-cased terms of at least two characters, each
// double-quoted (so FTS keywords like AND/NOT lose their meaning) and joined with OR. Example:
// "Where does the CLI start?" -> `"where" OR "does" OR "the" OR "cli" OR "start"`. OR (rather than AND) favours recall; rank sorts the best matches first.
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
// Row counts of the files and search tables; equal in a healthy index because every file gets one FTS row.
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
// Used by the report summary ("N symbols").
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

// Skipped and binary files get an empty string, so they stay searchable by path but contribute no content.
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

// Path-traversal guard: the indexed path must be relative and must still resolve inside the root after cleaning.
// Prevents a crafted inventory entry (for example "../../secret") from making the indexer read outside the project.
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

// True for the root itself or any path below it, using the relative path computed by filepath.Rel.
func insideRoot(root, path string) (bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."), nil
}

// SQLite has no boolean type; flags are stored as 0/1 integers.
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// Concatenates names, signatures and docs of one file's symbols into three space-separated strings, one per FTS column.
func searchableSymbols(items []symbols.Symbol) (names, signatures, docs string) {
	var nameParts, signatureParts, docParts []string
	for _, item := range items {
		nameParts = append(nameParts, item.Name)
		signatureParts = append(signatureParts, item.Signature)
		docParts = append(docParts, item.Doc)
	}
	return strings.Join(nameParts, " "), strings.Join(signatureParts, " "), strings.Join(docParts, " ")
}

// Schema (raw SQL below). files(id, path UNIQUE, ext, language, size, line_count, content_hash, is_binary) is the
// inventory; symbols references files by file_id; tech is created and cleared here but nothing in this file fills it; meta is a key/value table
// (root, schema_version); search is the FTS5 table that is queried with MATCH. Keep this in sync with the column
// lists in RebuildWithSymbols.
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
