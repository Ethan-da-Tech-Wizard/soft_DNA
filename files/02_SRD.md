# CodeLens Local — Software Requirements Document (SRD)

**Status:** Baseline v1 · **Companion docs:** Charter (00), PRD (01), Build Plan (03), Agent Guide (04)

This document is the technical contract. It defines *what the software must do* and *how the pieces fit*, without writing the implementation.

---

## 1. Functional requirements

Grouped by module. IDs are stable; reference them in commits and tests.

### Scanner (`scanner`)
- **FR-S1** Recursively walk a root directory provided by the user.
- **FR-S2** Exclude ignored directories by default: `node_modules`, `.git`, `.venv`, `venv`, `__pycache__`, `dist`, `build`, `target`, `.cache`, `.next`, `coverage`, `.idea`, `.vscode`.
- **FR-S3** Respect `.gitignore` entries found in the root (best-effort).
- **FR-S4** For each file, record: relative path, extension, size (bytes), line count, detected language, content hash (SHA-256).
- **FR-S5** Skip files over a configurable size cap (default 2 MB) and binary files (record them in inventory, do not read contents).
- **FR-S6** Never follow symlinks that resolve outside the root directory.
- **FR-S7** A read error on one file must be logged and must not abort the scan.

### Stack detector (`stackdetect`)
- **FR-D1** Detect languages/frameworks from manifest files (e.g., `package.json`, `requirements.txt`, `pyproject.toml`, `go.mod`, `Cargo.toml`, `Dockerfile`, `tsconfig.json`).
- **FR-D2** Produce a single human-readable stack summary string and a structured list of detected technologies with the evidence file for each.

### Structure & entry points (`structure`)
- **FR-T1** Produce a folder tree summary (depth-limited) with a one-line role guess per significant folder.
- **FR-T2** Identify likely entry points by convention (e.g., `main.py`, `app.py`, `main.go`, `src/main.tsx`, `index.html`, `cmd/*/main.go`).

### Symbol extractor (`symbols`)
- **FR-Y1** Extract functions, classes/types, and imports for **Python, JavaScript/TypeScript, Go**.
- **FR-Y2** Each symbol records: name, kind, start line, end line, signature (where available), docstring/leading comment (where available), owning file.
- **FR-Y3** Parsing a malformed file must not crash; partial results are acceptable and the file is flagged.
- **FR-Y4** For unsupported languages, symbol extraction is skipped (file still appears in inventory).

### Index (`index`)
- **FR-I1** Persist inventory and symbols to a local SQLite database.
- **FR-I2** Provide FTS5 full-text search over file content and symbol names/signatures.
- **FR-I3** Index build is idempotent: re-running on the same input replaces prior data cleanly (keyed by content hash).

### Report & diagram (`report`, `diagram`)
- **FR-R1** Generate `CODEBASE_REPORT.md` with sections: (1) Project Summary, (2) Detected Tech Stack, (3) Folder Structure, (4) Important Files, (5) Entry Points, (6) Functions & Classes, (7) Imports/Dependencies, (8) Setup Guess, (9) Plain-English Explanation, (10) Risky/Confusing Areas.
- **FR-R2** Generate `architecture.mmd` (Mermaid) showing high-level components and their relationships, inferred from structure + entry points + imports.
- **FR-R3** Optionally emit `index.json` (machine-readable mirror of inventory + symbols + stack).

### CLI (`cli`)
- **FR-C1** Usage: `codelens [flags] <path>`.
- **FR-C2** Flags: `--out <dir>` (default `./codelens-out`), `--json` (emit index.json), `--ignore <glob>` (repeatable), `--max-file-size <bytes>`, `--quiet`, `--version`.
- **FR-C3** Exit codes: `0` success, `1` usage error, `2` path not found/not a directory, `3` internal error.

## 2. Non-functional requirements

### Performance (NFR-P)
- **NFR-P1** Index a 1,000+ file repository in < 5 s on a typical laptop.
- **NFR-P2** Memory usage bounded: stream/limit by the file-size cap; never load the whole repo into memory at once.
- **NFR-P3** File scanning runs concurrently (worker pool) but produces deterministic output ordering.

### Reliability & safety (NFR-S)
- **NFR-S1 (critical)** The tool MUST NOT execute, import, build, or run the analyzed code. Static read/parse only.
- **NFR-S2** No network access during core operation. (Future LLM feature: local model only, no source egress — gated and off by default.)
- **NFR-S3** Stay within the root; reject path traversal and out-of-root symlink escapes.
- **NFR-S4** Resource caps: file-size cap, max total files (configurable), parser timeouts so a pathological file can't hang a run.
- **NFR-S5** One bad file never aborts the run (fail soft, log, continue).

### Portability (NFR-X)
- **NFR-X1** Single static binary; **`CGO_ENABLED=0`**; cross-compiles to {darwin, linux, windows} × {amd64, arm64}.
- **NFR-X2** No runtime dependency the end user must install (no Python, no C toolchain, no DLLs).

### Maintainability (NFR-M)
- **NFR-M1** Clear module boundaries (one package per pipeline stage).
- **NFR-M2** Determinism: stable sort on every list that reaches output.
- **NFR-M3** Test coverage target ≥ 70% on core packages (scanner, stackdetect, symbols, index).

## 3. System architecture

A linear pipeline; each stage has one responsibility and a typed input/output. The codebase is understood **structurally first**, then explained.

```
            ┌──────────────┐
  folder ─▶ │   scanner    │ ─▶ FileInventory
            └──────────────┘
                   │
                   ▼
            ┌──────────────┐
            │ stackdetect  │ ─▶ StackReport
            └──────────────┘
                   │
                   ▼
            ┌──────────────┐
            │  structure   │ ─▶ FolderModel + EntryPoints
            └──────────────┘
                   │
                   ▼
            ┌──────────────┐
            │   symbols    │ ─▶ []Symbol
            └──────────────┘
                   │
                   ▼
            ┌──────────────┐
            │    index     │ ─▶ SQLite DB (+ FTS5)
            └──────────────┘
                   │
                   ▼
       ┌───────────────────────┐
       │  report   +  diagram  │ ─▶ CODEBASE_REPORT.md
       └───────────────────────┘        architecture.mmd
                                         index.json (optional)
```

> Note: This is CodeLens's *own* architecture. The User→Frontend→Backend→Python→Whisper diagram from the source notes is an **example of CodeLens's output** for a sample app, not this design.

## 4. Module interface contracts (design intent)

Expressed as Go-style signatures to fix the contracts. This is design documentation, not implementation — names and exact types may be refined during build, but responsibilities are fixed.

```go
// scanner
type FileInfo struct {
    Path        string // relative to root
    Ext         string
    SizeBytes   int64
    LineCount   int
    Language    string
    ContentHash string // sha256 hex
    IsBinary    bool
    Skipped     bool
    SkipReason  string
}
func Scan(root string, opts ScanOptions) ([]FileInfo, error)

// stackdetect
type Tech struct{ Name, Category, EvidenceFile string }
type StackReport struct{ Summary string; Tech []Tech }
func Detect(files []FileInfo, root string) StackReport

// structure
type EntryPoint struct{ Path, Reason string }
func Analyze(files []FileInfo) (FolderModel, []EntryPoint)

// symbols
type Symbol struct {
    Name, Kind, Signature, Doc string
    File                       string
    StartLine, EndLine         int
}
func Extract(file FileInfo, source []byte) ([]Symbol, error)

// index
type Store interface {
    Init() error
    UpsertFiles([]FileInfo) error
    UpsertSymbols([]Symbol) error
    Search(query string, limit int) ([]Hit, error) // FTS5-backed
    Close() error
}

// report / diagram
func WriteReport(model ReportModel, outDir string) error
func WriteMermaid(model ReportModel, outDir string) error
```

## 5. Data model (SQLite)

```sql
CREATE TABLE files (
    id           INTEGER PRIMARY KEY,
    path         TEXT NOT NULL UNIQUE,
    ext          TEXT,
    language     TEXT,
    size_bytes   INTEGER,
    line_count   INTEGER,
    content_hash TEXT,
    is_binary    INTEGER DEFAULT 0
);

CREATE TABLE symbols (
    id         INTEGER PRIMARY KEY,
    file_id    INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL,          -- function | class | type | method | import
    signature  TEXT,
    doc        TEXT,
    start_line INTEGER,
    end_line   INTEGER
);

CREATE TABLE tech (
    id            INTEGER PRIMARY KEY,
    name          TEXT,
    category      TEXT,                -- language | framework | tool | runtime
    evidence_file TEXT
);

CREATE TABLE meta (             -- run metadata: root path, tool version, optional timestamp
    key   TEXT PRIMARY KEY,
    value TEXT
);

-- Full-text search (FTS5). Contentless/external-content table mirroring searchable text.
CREATE VIRTUAL TABLE search USING fts5(
    path, symbol_name, signature, doc, content
);
```

Indexing rule: a file is re-indexed only if its `content_hash` changed → idempotent rebuilds (FR-I3).

Determinism rule: generated reports and golden-test outputs must not include wall-clock timestamps by default. If a timestamp is recorded in SQLite metadata for debugging, it must be omitted from deterministic outputs unless the user explicitly enables a non-reproducible metadata flag.

## 6. Output contracts

- **`CODEBASE_REPORT.md`** — the 10 sections in FR-R1, in fixed order, deterministic content.
- **`architecture.mmd`** — valid Mermaid `flowchart` syntax; renders on github.com and mermaid.live.
- **`index.json`** — `{ "meta": {...}, "stack": {...}, "files": [...], "symbols": [...] }`. Stable key order.

## 7. Error handling & logging

- Structured logging (Go `log/slog`), level controlled by `--quiet`.
- Recoverable per-file errors → `WARN`, run continues.
- Fatal errors (bad path, DB init failure) → exit with the code from FR-C3 and a one-line human message on stderr.

## 8. Tech stack (pinned)

| Concern | Choice | Notes |
|--------|--------|-------|
| Language | **Go 1.25 minimum**; build with current stable **Go 1.26.x** | Memory-safe, single binary. Avoid hardcoded patch versions in docs. |
| Build | `CGO_ENABLED=0` | No C toolchain; clean cross-compile. |
| Database | `modernc.org/sqlite` (pure Go) | FTS5 supported without CGo. |
| Parsing | Lightweight safe extractors first; `github.com/odvcencio/gotreesitter` if deeper parsing is needed | Keep v1 pure-Go/no-CGo. Do not use `github.com/tree-sitter/go-tree-sitter` for v1. |
| CLI | stdlib `flag` (or a small lib if justified) | Keep dependencies minimal. |
| Logging | stdlib `log/slog` | No external dependency. |
| Release | `go build` matrix (optionally GoReleaser) | One binary per OS/arch. |

> Agents: verify the local `go version`, resolve dependency versions intentionally, then pin and commit exact module versions in `go.mod` and `go.sum`. Do not leave `go get <module>@latest` as repeatable project guidance.
