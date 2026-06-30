# CodeLens Local — ADR Log

**Status:** Active
**Purpose:** Track architecture decisions so future agents can change direction intentionally instead of silently drifting.

An ADR is required when a change affects language, parser/runtime, database, distribution model, safety posture, output contracts, milestone scope, or a dependency with meaningful operational risk.

## ADR format

```md
## ADR-000N — <decision title>

**Status:** Proposed | Accepted | Rejected | Superseded
**Date:** YYYY-MM-DD
**Owner:** <human or agent>

### Context
What problem or tradeoff forced the decision?

### Decision
What is the selected path?

### Consequences
What gets easier, harder, safer, slower, or deferred?

### Guardrails
What must future work preserve?
```

## ADR-0001 — Build CodeLens Local in Go with a Pure-Go Runtime

**Status:** Accepted
**Date:** 2026-06-30
**Owner:** Project lead

### Context
The project needs to be simple, fast, safe, local-first, and easy to distribute. Candidate languages were Go, Rust, C++, C#, and Python.

### Decision
Use Go, require `CGO_ENABLED=0`, and distribute one static binary per supported OS/architecture.

### Consequences
Go gives enough performance for an I/O-heavy codebase analyzer, avoids C/C++ memory-management risk, avoids Python packaging friction, and is easier for AI agents to modify safely than Rust for v1.

### Guardrails
Keep core dependencies pure Go. If a dependency requires CGo, an ADR must justify the change before implementation.

## ADR-0002 — Use Lightweight Extraction First, gotreesitter if Needed

**Status:** Accepted
**Date:** 2026-06-30
**Owner:** Project lead

### Context
The docs originally said "pure-Go tree-sitter runtime" without naming the exact library. That is too vague: official Go tree-sitter bindings expose C-backed allocation patterns and conflict with the pure-Go/no-CGo build goal.

### Decision
Use lightweight safe extractors for the first M4 implementation. If deeper tree-sitter parsing is needed, use `github.com/odvcencio/gotreesitter`.

### Consequences
The parser path now matches the single-binary, no-C-toolchain distribution goal while preserving the KISS rule. M0-M3 remain parser-free, and M4 starts with the smallest useful extractor.

### Guardrails
Do not use `github.com/tree-sitter/go-tree-sitter` in v1 unless this ADR is superseded and the pure-Go/no-CGo requirement is deliberately changed.

## ADR-0003 — Deterministic Outputs Exclude Wall-Clock Time

**Status:** Accepted
**Date:** 2026-06-30
**Owner:** Project lead

### Context
The project requires byte-identical report output for the same input, but timestamps make generated files change every run.

### Decision
Do not include wall-clock timestamps in `CODEBASE_REPORT.md`, `architecture.mmd`, golden snapshots, or default `index.json`.

### Consequences
Reports diff cleanly and tests remain stable. Debug metadata may store a timestamp in SQLite only if it is excluded from deterministic outputs by default.

### Guardrails
Any future generated timestamp must be behind an explicit non-reproducible metadata option.
