// Package grounding compiles verified facts from codelens.db into structured
// prompting contexts for the M6 hybrid LLM summarization pass.
// All data flows strictly from SQLite into the context; the LLM may only
// narrate these facts, never invent new ones.
package grounding

import (
	"database/sql"
	"fmt"

	"codelens-local/internal/structure"

	_ "modernc.org/sqlite"
)

// TechEntry is one detected technology row from the tech table.
type TechEntry struct {
	Name     string
	Category string
	Evidence string
}

// SymbolEntry is one extracted symbol row from the symbols+files join.
type SymbolEntry struct {
	File      string
	Name      string
	Kind      string
	StartLine int
	Signature string
}

// GroundingContext holds verified, read-only facts derived from codelens.db.
// Every field is populated from SQLite before any LLM prompt is constructed.
// The LLM must only narrate these facts; it may not invent new ones.
type GroundingContext struct {
	Root        string
	FileCount   int
	Stack       []TechEntry
	EntryPoints []string
	TopSymbols  []SymbolEntry
	FolderRoles map[string]string
}

// Build opens the SQLite database at dbPath and returns a GroundingContext
// populated with verified facts. On any error it returns a partial context
// and the first error encountered; callers should treat it as advisory.
func Build(dbPath string) (GroundingContext, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return GroundingContext{}, fmt.Errorf("grounding: open db: %w", err)
	}
	defer db.Close()
	return buildFromDB(db)
}

// buildFromDB performs all queries against an already-open database.
// Extracted as a separate function to allow direct testing with in-memory DBs.
func buildFromDB(db *sql.DB) (GroundingContext, error) {
	ctx := GroundingContext{
		FolderRoles: make(map[string]string),
	}

	// meta: project root path (QueryRow; ignore ErrNoRows — root stays empty).
	_ = db.QueryRow(`SELECT value FROM meta WHERE key = 'root'`).Scan(&ctx.Root)

	// files: count of non-binary source files.
	_ = db.QueryRow(`SELECT COUNT(*) FROM files WHERE is_binary = 0`).Scan(&ctx.FileCount)

	// tech: detected stack, ordered for deterministic output.
	techRows, err := db.Query(
		`SELECT name, category, evidence_file FROM tech ORDER BY category, name`)
	if err != nil {
		return ctx, fmt.Errorf("grounding: query tech: %w", err)
	}
	defer techRows.Close()
	for techRows.Next() {
		var t TechEntry
		if err := techRows.Scan(&t.Name, &t.Category, &t.Evidence); err != nil {
			return ctx, fmt.Errorf("grounding: scan tech row: %w", err)
		}
		ctx.Stack = append(ctx.Stack, t)
	}
	if err := techRows.Err(); err != nil {
		return ctx, fmt.Errorf("grounding: tech rows: %w", err)
	}

	// symbols: top 50 verified symbols (functions/classes/types only).
	// COALESCE guards against NULL start_line and signature from older schema rows.
	symRows, err := db.Query(`
		SELECT f.path, s.name, s.kind, COALESCE(s.start_line, 0), COALESCE(s.signature, '')
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.kind IN ('function', 'class', 'method', 'type', 'table', 'procedure', 'view')
		ORDER BY f.path, s.start_line
		LIMIT 50`)
	if err != nil {
		return ctx, fmt.Errorf("grounding: query symbols: %w", err)
	}
	defer symRows.Close()
	for symRows.Next() {
		var s SymbolEntry
		if err := symRows.Scan(&s.File, &s.Name, &s.Kind, &s.StartLine, &s.Signature); err != nil {
			return ctx, fmt.Errorf("grounding: scan symbol row: %w", err)
		}
		ctx.TopSymbols = append(ctx.TopSymbols, s)
	}
	if err := symRows.Err(); err != nil {
		return ctx, fmt.Errorf("grounding: symbol rows: %w", err)
	}

	// entry points: files matching known entry-point naming conventions.
	epRows, err := db.Query(`
		SELECT path FROM files
		WHERE path IN (
			'main.go', 'main.py', 'app.py', 'main.rb', 'app.rb',
			'main.r', 'run.r', 'main.R', 'run.R', 'main.swift',
			'App.swift', 'main.c', 'main.cpp', 'main.cc', 'main.cxx',
			'index.html', 'schema.sql', 'init.sql'
		)
		OR path LIKE 'cmd/%/main.go'
		OR path LIKE 'src/main.%'
		OR path = 'src/lib.rs'
		ORDER BY path`)
	if err != nil {
		return ctx, fmt.Errorf("grounding: query entry points: %w", err)
	}
	defer epRows.Close()
	for epRows.Next() {
		var path string
		if err := epRows.Scan(&path); err != nil {
			return ctx, fmt.Errorf("grounding: scan entry point row: %w", err)
		}
		ctx.EntryPoints = append(ctx.EntryPoints, path)
	}
	if err := epRows.Err(); err != nil {
		return ctx, fmt.Errorf("grounding: entry point rows: %w", err)
	}

	// folders: derive top-level directory roles from all file paths.
	// SQLite INSTR/SUBSTR extract the first path component.
	folderRows, err := db.Query(`
		SELECT DISTINCT
			CASE
				WHEN INSTR(path, '/') > 0 THEN SUBSTR(path, 1, INSTR(path, '/') - 1)
				ELSE '.'
			END AS folder
		FROM files
		ORDER BY folder`)
	if err != nil {
		return ctx, fmt.Errorf("grounding: query folders: %w", err)
	}
	defer folderRows.Close()
	for folderRows.Next() {
		var folder string
		if err := folderRows.Scan(&folder); err != nil {
			return ctx, fmt.Errorf("grounding: scan folder row: %w", err)
		}
		ctx.FolderRoles[folder] = folderRole(folder)
	}
	if err := folderRows.Err(); err != nil {
		return ctx, fmt.Errorf("grounding: folder rows: %w", err)
	}

	return ctx, nil
}

// folderRole maps a top-level directory name to a human-readable role description.
func folderRole(folder string) string {
	return structure.RoleFor(folder)
}
