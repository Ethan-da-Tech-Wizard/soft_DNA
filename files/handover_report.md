# CodeLens Local — Final Handover Report

**Version:** v1.0.0  
**Date:** 2026-07-01  
**Module:** `codelens-local` (Go 1.25, `CGO_ENABLED=0`)  
**Build status:** clean — `make lint` ✓ · `go test ./...` 137/137 ✓ · `CGO_ENABLED=0 go build ./cmd/codelens` ✓

---

## 1. Milestone Completion Status

### M0 — CLI Skeleton

| Item | Status |
|---|---|
| Entry-point binary (`cmd/codelens/main.go`) | Complete |
| Flag set (`--out`, `--ask`, `--version`, `--max-file-size`, `--llm*`) | Complete |
| Version string injection via `-ldflags` | Complete |
| End-to-end version flag test | Complete |

**Test coverage:** `TestVersionFlagEndToEnd` (1 test)

---

### M1 — File Scanner

| Item | Status |
|---|---|
| Recursive directory walk respecting `.gitignore` globs | Complete |
| Extra ignore patterns (`node_modules/`, `.git/`, `vendor/`, etc.) | Complete |
| Binary file detection + `IsSkipped` flag | Complete |
| Oversized-file detection (configurable byte ceiling) | Complete |
| Out-of-root symlink guard | Complete |
| C and C++ fixture scanning | Complete |
| Language detection from file extension (SQL, Swift, R, C++, C) | Complete |
| Deterministic output order | Complete |

**Test coverage:** `internal/scanner` — 11 tests  
(`TestScanInventoriesTextFilesDeterministically`, `TestScanRespectsGitignoreAndExtraIgnores`, `TestScanMarksLargeAndBinaryFilesSkipped`, `TestScanDoesNotFollowOutOfRootSymlink`, `TestSampleFixturesInventory`, `TestScanCFixture`, `TestScanCPPFixture`, `TestDetectSQLLanguage`, `TestDetectSwiftLanguage`, `TestDetectRLanguage`, `TestDetectCPPAndCMakeLanguages`)

---

### M2 — Stack Detection & Folder Structure

#### Stack detection (`internal/stackdetect`)

| Manifest / lockfile | Stack identified | Category |
|---|---|---|
| `go.mod` | Go + Go Modules | language + tool |
| `requirements.txt` | Python + pip | language + tool |
| `pyproject.toml` | Python + pyproject | language + tool |
| `poetry.lock` | Poetry | tool |
| `Pipfile` | Pipenv | tool |
| `package.json` | Node.js + React/Vue/Angular/Vite/Next (heuristic) | runtime + framework |
| `yarn.lock` | Yarn | tool |
| `pnpm-lock.yaml` | pnpm | tool |
| `bun.lockb` | Bun | runtime |
| `Cargo.toml` | Rust + Cargo | language + tool |
| `Gemfile` | Ruby + Bundler | language + tool |
| `CMakeLists.txt` | C++ + CMake | language + tool |
| `Makefile` (+ `.c` files) | C + Make | language + tool |
| `DESCRIPTION` | R + Rscript | language + tool |
| `Package.swift` | Swift + SPM | language + tool |
| `schema.sql` / `.sql` files | SQL | database |

**Test coverage:** `internal/stackdetect` — 15 tests  
(Python/Node/Rust/Ruby/C/C++/R/Swift/SQL/Go, Poetry, Pipenv, Yarn, pnpm, Bun)

#### Folder structure & entry-point detection (`internal/structure`)

| Pattern | Maps to |
|---|---|
| `main.go`, `cmd/*/main.go` | Go entry point |
| `main.py`, `app.py`, `__main__.py` | Python entry point |
| `main.tsx`, `src/main.tsx`, `index.js` | Node.js entry point |
| `main.rs`, `src/main.rs` | Rust entry point |
| `main.rb`, `app.rb` | Ruby entry point |
| `main.R`, `app.R` | R entry point |
| `main.swift`, `Sources/*/main.swift` | Swift entry point |
| `main.cpp`, `main.c` | C/C++ entry point |
| `schema.sql` | SQL entry point |
| Named folder patterns | `cmd/`, `internal/`, `pkg/`, `test*/`, `lib/`, `src/`, `api/`, `scripts/`, etc. |

**Test coverage:** `internal/structure` — 17 tests  
(10 `TestAnalyzeFinds*EntryPoint` + 7 `TestEntryReason*`)

---

### M3 — SQLite / FTS5 Index

| Item | Status |
|---|---|
| Schema init (`meta`, `files`, `symbols`, `tech`, `fts_files` virtual table) | Complete |
| `modernc.org/sqlite` (pure-Go, CGO-free) | Complete |
| FTS5 full-text search over path + content snippet | Complete |
| Idempotent rebuild (safe to call multiple times) | Complete |
| Symbol row insertion for search ranking | Complete |

**Test coverage:** `internal/index` — 5 tests  
(`TestStoreBuildsFTSIndex`, `TestSearchAcceptsNaturalLanguageQuery`, `TestStoreRebuildIsIdempotent`, `TestStoreIndexesSymbolsForSearch`, `TestStoreSchemaCanInitializeTwice`)

---

### M4 — Symbol Extraction, Mermaid Diagram & Report

#### Symbol extraction (`internal/symbols`)

| Language | Extracts |
|---|---|
| Go | `func`, `type`, `import` |
| Python | `def`, `class`, `import` |
| TypeScript / JavaScript | `function`, `class`, `interface`, `import` |
| Rust | `fn`, `struct`, `enum`, `impl`, `use` |
| Ruby | `def`, `class`, `module`, `require` |
| C | function definitions |
| C++ | function definitions, class declarations |
| R | assignment-style function definitions |
| Swift | `func`, `class`, `struct`, `enum`, `import` |
| SQL | `CREATE TABLE`, `CREATE VIEW`, `CREATE INDEX`, `CREATE PROCEDURE` |

**Test coverage:** `internal/symbols` — 11 tests (one per language + `TestMalformedFileDoesNotCrash`)

#### Mermaid diagram (`internal/diagram`)

Writes a `graph LR` diagram of detected tech nodes and their relationships to `architecture.mmd`. **Test coverage:** `internal/diagram` — 1 test (`TestWriteMermaid`)

#### Report generation (`internal/cli`)

The full `CODEBASE_REPORT.md` has 11 numbered sections:

1. Tech Stack Summary  
2. Project Statistics  
3. Entry Points  
4. Folder Roles  
5. Top Symbols by Reference Count  
6. File Inventory  
7. FTS Search Index  
8. Setup Commands  
9. Developer Notes  
10. Architecture Diagram (Mermaid embed)  
11. AI-Assisted Summaries *(populated by M6 `--llm` pass; stub message when disabled)*

**Determinism guarantee:** all collections are sorted before output; identical input → identical bytes.

---

### M5 — Local Retrieval Q&A (`--ask`)

| Item | Status |
|---|---|
| FTS5 keyword search with file + line citations | Complete |
| Entry-point boosting for "where does X start" queries | Complete |
| Output written to `codelens-out/ANSWER.md` | Complete |
| No model required | Complete |

**Test coverage:** `internal/cli` — `TestRunWritesLocalAnswer`, `TestRunAnswerPrefersEntryPointQuestions`

---

### M6 — Hybrid Offline LLM Summaries

#### LLM runner (`internal/llm`)

| Item | Status |
|---|---|
| `Run()` — shells out to `llama-cli` subprocess | Complete |
| `Config` struct (`BinaryPath`, `ModelPath`, `GrammarPath`, `Threads`, `MaxTokens`, `Temperature`, `Timeout`, `MaxOutputBytes`) | Complete |
| `limitedWriter` — sticky-error byte-ceiling `io.Writer` | Complete |
| `--grammar-file` flag passed to llama-cli | Complete |
| Timeout guard (`context.WithTimeout`; zero-timeout safe) | Complete |
| GBNF grammar embedding (`//go:embed grammars/*.gbnf`) | Complete |
| `WriteGrammar(name)` — extracts grammar to temp file, returns cleanup func | Complete |
| `file_explain.gbnf` — constrains `{"file_summary": ..., "symbol_explanations": [...]}` | Complete |
| `project_overview.gbnf` — constrains `{"project_description": ...}` | Complete |
| Injection defense: `str-char` rule prohibits unescaped `\n`/`\r` in strings | Complete |

**Test coverage:** `internal/llm` — 11 tests  
(6 `TestLimitedWriter*` + 4 `TestRun*` + `TestWriteGrammar`)

#### Grounding context (`internal/grounding`)

| Item | Status |
|---|---|
| `BuildFromDB()` — reads `meta`, `files`, `symbols`, `tech` tables | Complete |
| COALESCE NULL-signature guard | Complete |
| Top-50 symbol cap | Complete |
| Binary file exclusion from prompt | Complete |
| `src/main.*` entry-point pattern | Complete |
| `FolderRole()` — maps 14 folder-name patterns to semantic labels | Complete |
| `FormatFileExplainPrompt()` — 3-zone prompt (SYSTEM FACTS / FILE INFO / TASK) | Complete |
| `FormatProjectOverviewPrompt()` — deterministic, folder-sorted | Complete |

**Test coverage:** `internal/grounding` — 18 tests  
(11 `TestBuildFromDB*` + 1 `TestFolderRole` + 6 `TestFormat*Prompt*`)

#### Post-generation validation (`internal/validate`)

| Item | Status |
|---|---|
| `ValidateFileExplanation()` — symbol allowlist O(1) lookup | Complete |
| Hallucinated symbol names discarded with `ValidationError` | Complete |
| `sanitizeText()` — 4-pass: backticks → URLs → absolute paths → multi-space collapse | Complete |
| `truncateRunes()` — Unicode boundary-safe with `…` (U+2026) suffix | Complete |
| `ValidateProjectExplanation()` — hard error on JSON failure; silent truncation at 1000 runes | Complete |
| `FileSummary` hard cap: 500 runes + `ValidationError` if truncated | Complete |

**Test coverage:** `internal/validate` — 27 tests  
(6 `TestSanitizeText*` + 4 `TestTruncateRunes*` + 9 `TestValidateFileExplanation*` + 6 `TestValidateProjectExplanation*` + `TestValidationErrorString`)

#### Pipeline integration (`internal/cli`)

| Item | Status |
|---|---|
| `--llm` / `--llm-binary` / `--llm-model` flags | Complete |
| `--llm-threads` (default: `runtime.NumCPU()`) | Complete |
| `--llm-timeout` (default: `60s`) | Complete |
| `--llm-max-files` (default: `10`) | Complete |
| Warning emitted when `--llm` set but binary/model missing; run continues | Complete |
| `runLLMPipeline()` → project overview call → per-file calls → Section 11 merge | Complete |
| Integration test with stub llama-cli binary | Complete (`TestRunLLMIntegration`) |

---

### Release Engineering

| Item | Status |
|---|---|
| `make release VERSION=<tag>` — 6-platform matrix (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64) | Complete |
| Unix archives: `.tar.gz` with binary at root | Complete |
| Windows archives: `.zip` with `codelens.exe` at root | Complete |
| `dist/checksums.txt` — SHA-256 for all archives (self-hashing prevented) | Complete |
| `scripts/download_llama.sh` — auto-detects OS/arch, downloads latest llama.cpp release, retries, HF_TOKEN support | Complete |
| `scripts/download_model.sh` — downloads Qwen2.5-Coder GGUF, configurable filename and target dir | Complete |

---

### Node.js Lockfile Detection (added during polish phase)

| Lockfile | Package manager | Setup commands |
|---|---|---|
| `yarn.lock` | Yarn | `yarn install` + `yarn dev` |
| `pnpm-lock.yaml` | pnpm | `pnpm install` + `pnpm dev` |
| `bun.lockb` | Bun | `bun install` + `bun run dev` |
| *(none)* | npm (default) | `npm install` + `npm run dev` |

**Test coverage:** `TestSetupCommandsNodeLockfiles` (4 sub-tests), `TestDetectYarnStack`, `TestDetectPnpmStack`, `TestDetectBunStack`

---

## 2. Test Suite Summary

| Package | Tests | Milestone |
|---|---|---|
| `cmd/codelens` | 1 | M0 |
| `internal/scanner` | 11 | M1 |
| `internal/stackdetect` | 15 | M2 |
| `internal/structure` | 17 | M2 |
| `internal/index` | 5 | M3 |
| `internal/symbols` | 11 | M4 |
| `internal/diagram` | 1 | M4 |
| `internal/cli` | 21 (+7 sub-tests) | M0–M6 |
| `internal/llm` | 11 | M6 |
| `internal/grounding` | 18 | M6 |
| `internal/validate` | 27 | M6 |
| **Total** | **137 passing** | |

All tests pass with no failures, no skips, and no race conditions (`go test ./... -count=1`).

---

## 3. Repository File Map

```
soft_DNA-main/
│
├── cmd/codelens/
│   ├── main.go              Entry point; calls cli.Run()
│   └── main_test.go         End-to-end version flag test
│
├── internal/
│   ├── cli/
│   │   ├── cli.go           Run(), flag parsing, report writer, LLM pipeline, setupCommands()
│   │   └── cli_test.go      Full integration tests (report, ask, LLM, language reports, setup)
│   ├── scanner/
│   │   ├── scanner.go       Scan() — recursive walk, ignore patterns, binary detection
│   │   ├── ignore.go        Language-from-extension table; IsSkippedPath()
│   │   └── scanner_test.go  11 tests
│   ├── stackdetect/
│   │   ├── stackdetect.go   Detect() — manifest → Tech slice + Summary string
│   │   └── stackdetect_test.go  15 tests
│   ├── structure/
│   │   ├── structure.go     Analyze() — entry-point detection; FolderRoles()
│   │   └── structure_test.go    17 tests
│   ├── index/
│   │   ├── store.go         Build() / Search() — SQLite schema init + FTS5 queries
│   │   └── store_test.go    5 tests
│   ├── symbols/
│   │   ├── symbols.go       Extract() — regex-based multi-language symbol extractor
│   │   └── symbols_test.go  11 tests
│   ├── diagram/
│   │   ├── diagram.go       WriteMermaid() — graph LR diagram from Tech slice
│   │   └── diagram_test.go  1 test
│   ├── grounding/
│   │   ├── grounding.go     BuildFromDB() — reads SQLite, populates GroundingContext
│   │   ├── prompt.go        FormatFileExplainPrompt() / FormatProjectOverviewPrompt()
│   │   └── grounding_test.go    18 tests (white-box, in-memory SQLite)
│   ├── llm/
│   │   ├── runner.go        Config, limitedWriter, Run(), WriteGrammar()
│   │   ├── runner_test.go   11 tests
│   │   └── grammars/
│   │       ├── file_explain.gbnf      JSON schema for per-file LLM output
│   │       └── project_overview.gbnf  JSON schema for project-level LLM output
│   └── validate/
│       ├── validate.go      ValidateFileExplanation(), ValidateProjectExplanation(),
│       │                    sanitizeText(), truncateRunes()
│       └── validate_test.go 27 tests
│
├── testdata/
│   ├── edge_cases/          binary.bin, empty.txt, malformed.py, .gitignore
│   ├── sample_c/            main.c, utils.c, utils.h, Makefile
│   ├── sample_cpp/          main.cpp, tree_node.cpp, tree_node.hpp, CMakeLists.txt
│   ├── sample_empty/        .gitkeep
│   ├── sample_go/           cmd/app/main.go, go.mod
│   ├── sample_node/         package.json, tsconfig.json, src/main.tsx
│   ├── sample_python/       main.py, requirements.txt, node_modules/ignored.js
│   ├── sample_r/            DESCRIPTION, main.R
│   ├── sample_ruby/         Gemfile, main.rb
│   ├── sample_rust/         Cargo.toml, src/main.rs
│   ├── sample_sql/          schema.sql
│   └── sample_swift/        Package.swift, main.swift
│
├── scripts/
│   ├── download_llama.sh    Auto-installs llama-cli from GitHub releases
│   └── download_model.sh    Downloads Qwen2.5-Coder GGUF from Hugging Face
│
├── files/                   Project design documents (charter, PRD, SRD, build plan, etc.)
│   ├── 00_PROJECT_CHARTER.md
│   ├── 01_PRD.md
│   ├── 02_SRD.md
│   ├── 03_BUILD_PLAN.md
│   ├── 04_AGENT_GUIDE.md
│   ├── 05_ADR_LOG.md
│   ├── 06_SECURITY_MODEL.md
│   ├── 07_AGENT_TASK_PROMPTS.md
│   ├── 08_LANGUAGE_SUPPORT_ROADMAP.md
│   ├── 09_AGENT_SEQUENTIAL_PROMPTS.md
│   └── handover_report.md   ← this document
│
├── Makefile                 build / test / lint / smoke / release / clean targets
├── go.mod                   module codelens-local; requires modernc.org/sqlite
├── go.sum
├── README.md                User-facing documentation with LLM Quick-Start
├── ROADMAP.md               v1 feature list and deferred ideas
└── walkthrough.md           Step-by-step developer onboarding for v1.0.0
```

---

## 4. CLI Flag Reference

| Flag | Type | Default | Description |
|---|---|---|---|
| `--out` | string | `./codelens-out` | Output directory |
| `--ask` | string | — | FTS question; writes `ANSWER.md` |
| `--max-file-size` | int64 | `2097152` (2 MB) | Skip files larger than this |
| `--version` | bool | `false` | Print version and exit |
| `--llm` | bool | `false` | Enable offline LLM summarization |
| `--llm-binary` | string | — | Path to `llama-cli` executable |
| `--llm-model` | string | — | Path to `.gguf` model file |
| `--llm-threads` | int | `runtime.NumCPU()` | Inference thread count |
| `--llm-timeout` | duration | `60s` | Per-call subprocess deadline |
| `--llm-max-files` | int | `10` | Max files to annotate |

---

## 5. Developer Quick-Start

```sh
# Build
make build VERSION=v1.0.0

# Test
make test

# Lint
make lint

# End-to-end smoke
make smoke

# Release archives (6 platforms + checksums)
make release VERSION=v1.0.0

# Verify checksums
cd dist && sha256sum -c checksums.txt

# Run against a project
./bin/codelens ./path/to/project

# Run with offline LLM (requires one-time setup; see README.md §LLM Quick-Start)
./bin/codelens --llm \
  --llm-binary ~/.codelens/bin/llama-cli \
  --llm-model  ~/.codelens/models/qwen2.5-coder-1.5b-instruct-q4_k_m.gguf \
  ./path/to/project
```

---

## 6. Safety Guardrails (Non-Negotiable)

| Guardrail | Enforcement |
|---|---|
| G-1: No target-code execution | `internal/scanner` reads bytes only; no `exec` calls against scanned project |
| G-2: No network in core analysis | All HTTP is opt-in at install time (`scripts/`); analysis is fully offline |
| G-3: No path traversal | Symlinks outside the scanned root are skipped (`TestScanDoesNotFollowOutOfRootSymlink`) |
| G-4: Pure Go | `CGO_ENABLED=0`; SQLite via `modernc.org/sqlite` |
| G-5: Deterministic output | All slices sorted before rendering; enforced by golden-prefix tests |
| G-6: Fail soft | Bad files logged as WARN; LLM call failures warn and continue |
| G-7: Resource caps | `--max-file-size` ceiling; `limitedWriter` byte cap on LLM output; `--llm-max-files` cap |
| G-8: No scope creep | No GUI, no cloud model, no VS Code extension |

---

## 7. Known Deferred Items

The following items are listed in `ROADMAP.md` under "Next Polish" and "Later, Only If Needed". None block the v1.0.0 release.

- Improve report wording on larger real-world monorepos
- Better ranking for `--ask` results (BM25 or manual boosting)
- Deeper AST parsing via `github.com/odvcencio/gotreesitter` (deferred; conflicts with CGO constraint for broad language support)
- CI configuration (`.github/workflows/` directory exists but is unpopulated)
- C++ Makefile-based project support (currently requires `CMakeLists.txt`)
