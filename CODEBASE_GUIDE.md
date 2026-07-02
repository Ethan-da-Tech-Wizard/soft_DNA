# CodeLens Local — Complete Codebase Guide

A part-by-part description of every package, file, and moving piece in this
repository, including the v1.1.0 BRSL diagnostics loop and exactly how it
changes (and does not change) the behavior of the LLM inside the app.

---

## 0. The big picture

CodeLens Local is a single static Go binary. You point it at a project folder
and it produces four artifacts in `codelens-out/`:

| Artifact | Producer | What it is |
|---|---|---|
| `CODEBASE_REPORT.md` | `internal/cli` | 11-section plain-English architecture report |
| `architecture.mmd` | `internal/diagram` | Mermaid flowchart of folders/entry points/stack |
| `codelens.db` | `internal/index` | SQLite database with FTS5 full-text search |
| `ANSWER.md` (optional) | `internal/cli` | Citation-based answer to an `--ask` question |

The data flows in one direction through a strict pipeline:

```
scanner ─▶ stackdetect ─▶ structure ─▶ symbols ─▶ index (SQLite)
                                                     │
                          ┌──────────────────────────┤
                          ▼                          ▼
                       diagram              grounding ─▶ llm ─▶ validate
                          │                          │
                          └────────────┬─────────────┘
                                       ▼
                              cli.writeReport / writeAnswer
```

Everything left of `grounding` is fully deterministic — no model, no network,
no execution of target code. The LLM stage (`--llm` flag) is optional, runs a
local llama.cpp subprocess, and only populates Section 11 of the report. If it
fails for any reason, the deterministic report still ships.

---

## 1. `cmd/codelens/main.go` — the entry point (13 lines)

The smallest possible `main`: it passes `os.Args[1:]`, `os.Stdout`,
`os.Stderr`, and a build-time `version` string into `cli.Run` and exits with
whatever code that returns. The `version` variable is injected at build time
via `-ldflags "-X main.version=$(VERSION)"` (see the Makefile). Keeping `main`
this thin means the entire CLI is testable in-process — the test suite calls
`cli.Run` directly with byte buffers instead of spawning the binary.

---

## 2. `internal/cli` — orchestration, report writing, and the BRSL loop

This is the conductor. `cli.go` (~860 lines) owns argument parsing, pipeline
sequencing, all Markdown rendering, and the LLM pass.

### 2.1 `Run` — flag parsing and pipeline sequencing

`Run` defines the flag set (`--out`, `--max-file-size`, `--ask`, `--ignore`
(repeatable), `--version`, and the six `--llm-*` flags), validates that the
target path exists and is a directory, then executes the pipeline stages in
order: scan → stack detect → structure analysis → symbol extraction → index
build → diagram → optional LLM pass → report → optional answer. Exit codes
are explicit constants: `0` OK, `1` usage error, `2` bad path, `3` internal
failure.

One subtle helper: `outputAwareIgnores` checks whether the output directory
lives *inside* the scan root and, if so, adds it to the ignore list — so a
second run never indexes its own previous report.

### 2.2 `writeReport` — the 11-section Markdown report

Builds the entire `CODEBASE_REPORT.md` in a `strings.Builder`, one section at
a time: project summary, tech-stack table, folder-role table, important
files, entry points, functions/classes table, imports table, setup-command
guesses, a plain-English explanation, risky areas (anything the scanner
skipped), and Section 11 "AI-Assisted Summaries." Every dynamic string passes
through `escapeMarkdown` (pipes escaped) so file paths can't break table
layout. Section 11 renders the validated LLM output — project overview plus
per-file annotations with symbol tables — or a "disabled" notice when `--llm`
was not used. A raw scanner inventory table closes the report.

### 2.3 `writeAnswer` — `--ask` mode, no model required

Answers a question using only the SQLite index. `entryPointHits` gives
priority to entry-point files when the question contains "start", "entry", or
"main"; the remainder of a 5-hit budget is filled by FTS5 search; `mergeHits`
deduplicates while preserving priority order. The output file explicitly
states "local FTS retrieval only. No model, network call, or target-code
execution was used" — honesty about provenance is a design rule here.

### 2.4 `setupCommands` — build-command guessing

Maps manifest evidence to setup commands (`go.mod` → `go run ./...`,
`Cargo.toml` → `cargo build`/`cargo run`, etc.), with lockfile-aware
refinement: a `package.json` project suggests `yarn`/`pnpm`/`bun`/`npm`
depending on which lockfile exists, and a `pyproject.toml` project suggests
`poetry`/`pipenv`/`pip` the same way.

### 2.5 `runLLMPipeline` + `summarizeFiles` — the hybrid LLM pass and BRSL

`runLLMPipeline` builds a `grounding.GroundingContext` from the SQLite DB,
makes one LLM call for the project overview, then hands the per-file work to
`summarizeFiles` — the **boundary-safe summarization loop (BRSL)** added in
v1.1.0, which implements the update rules from `KISS_LOOP_EQUATION.md`:

- Each file is one *segment*. A segment only counts as covered once its LLM
  output **passes validation** (previously any run was fire-and-forget).
- **On validation failure the file is retried, not skipped.** The pointer
  freezes, the throttle drops from `ρ = 0.75` to `ρ = 0.50`, and the retry
  feeds the model a 50% chunk (the verified-symbol context is halved from 5
  symbols to 2, shrinking the prompt).
- **On success** the pointer advances, the stall counter resets, and the
  throttle returns to 0.75 for the next file.
- After **3 consecutive failures** on one file (`stall > 2`) the loop
  declares an *atomic block stall*, writes a warning naming the file to
  stderr, stores a forced-cutoff placeholder summary, and force-advances —
  so one pathological file can never wedge or infinitely loop the pass.

Each turn emits one diagnostic line to stderr (never stdout, so the Markdown
report is never corrupted):

```
[LLM Loop] Turn 3: Coverage = 0.33 | Throttle = 0.50 | Health = 1 | Stall = 0
```

`var llmRun = llm.Run` is a package-level seam so tests can substitute a mock
runner without spawning a real llama.cpp process.

**Does BRSL change the model's behavior?** The model itself is untouched —
same binary, same GGUF weights, same temperature (0.1), same GBNF grammar,
same 512-token cap, same prompts on the first attempt. What changed is the
*pipeline around the model*:

1. **Retry on failure** — a file whose first output failed validation used to
   be silently dropped from Section 11; now it gets up to two more attempts,
   so report coverage is strictly better.
2. **Smaller retry prompt** — the 50%-chunk retry gives a struggling model
   less to narrate, which raises the odds the retry validates.
3. **Bounded worst case** — a permanently broken file costs exactly 3 turns,
   then produces an explicit "Summary unavailable: forced cutoff" entry
   instead of vanishing.
4. **Observability** — the loop is watchable in real time: coverage γ must
   climb monotonically toward 1.0; `Throttle = 0.50` means "currently
   repairing"; a rising stall count pinpoints the exact problem file.

### 2.6 `cli_test.go`

Golden-prefix test of the full report, per-language report tests against the
`testdata/` fixtures, `--ask` tests, ignore/size-limit tests, an end-to-end
LLM integration test using a mock shell script as `llama-cli`, and the two
BRSL tests: `TestLLMLoopDiagnostics` (fail-then-succeed, asserts the two
diagnostic lines verbatim) and `TestLLMLoopAtomicBlockStall` (always-fail,
asserts exactly 3 calls, the stall warning, and the cutoff summary).

---

## 3. `internal/scanner` — the file inventory (stage 1)

`Scan` walks the target root with `filepath.WalkDir` and returns a
deterministic (path-sorted) `[]FileInfo` — path, extension, size, line count,
language, SHA-256 content hash, and skip status. Design rules visible in the
code:

- **Never executes target code.** It only reads bytes.
- **Skips rather than fails.** Oversized files (default cap 2 MiB), binary
  files (NUL byte in the first 8 KB), and unreadable paths become inventory
  rows with `Skipped=true` and a `SkipReason` — which later feed the report's
  "Risky/Confusing Areas" section. A broken directory never aborts the scan.
- **Symlink safety.** Symlinks are resolved and rejected if they escape the
  scan root (`symlink_escape`), point at directories, or fail to resolve —
  closing the classic "symlink to /etc/passwd" indexing hole.
- **Noise directories** (`node_modules`, `.git`, `dist`, `__pycache__`,
  `target`, `venv`, …) are pruned wholesale.

`detectLanguage` maps ~20 extensions and special basenames (`Makefile`,
`Dockerfile`, `CMakeLists.txt`) to language names — this single function
decides which symbol extractor runs later.

`ignore.go` implements a deliberately small `.gitignore` subset: it reads the
root `.gitignore` (skipping comments and `!` negations), merges user
`--ignore` patterns, and matches with `filepath.Match` semantics, handling
directory-only patterns (`dist/`), basename patterns (`*.tmp`), and
path-prefix patterns.

---

## 4. `internal/stackdetect` — manifest-based tech detection (stage 2)

`Detect` looks *only* at manifest file presence — it never guesses from
source content. `go.mod` → Go; `Cargo.toml` → Rust + Cargo; `Gemfile` → Ruby
+ Bundler; `Package.swift`, `DESCRIPTION` (R), `schema.sql`, `Dockerfile`,
`CMakeLists.txt`, and so on. The one deeper inspection is
`detectPackageJSON`, which parses `package.json` dependencies to identify
React, Next.js, Vite, and TypeScript. Every `Tech` entry carries its
`EvidenceFile`, so the report can always answer "why do you think this is a
React project?" Results are deduplicated and sorted by category/name/evidence
for deterministic output, and `summary` joins unique names into the
"Go + Make + SQL"-style headline.

---

## 5. `internal/structure` — folder roles and entry points (stage 3)

Two pure functions over the inventory. `folders` counts non-skipped files per
top-level directory and assigns a conventional role (`cmd` → "Command-line
entry points", `internal` → "Private application packages", `src` →
"Application source", …). `entryPoints` matches known startup-file
conventions across ten languages — `main.go` and `cmd/*/main.go`, `main.py` /
`app.py`, `src/main.tsx`, `src/main.rs` / `src/lib.rs`, `main.c`,
`main.swift`, `schema.sql`, `index.html`, etc. — each with a human-readable
reason string that surfaces in both the report and the diagram.

---

## 6. `internal/symbols` — regex/AST symbol extraction (stage 4)

Extracts functions, classes/types, methods, and imports for ten languages
without compiling anything. Go gets the real treatment — `go/parser` builds
an AST and walks declarations, producing accurate line ranges and rebuilt
signatures. Every other language (Python, JS/TS, Ruby, R, Swift, SQL, Rust,
C, C++) is line-by-line regex matching: fast, dependency-free, and good
enough for naming what exists and where. The C/C++ extractors filter keyword
false-positives (`if`/`while`/`for`/`switch`/`return` "calls"), and the C++
one classifies `Namespace::method` matches as methods. Output is sorted by
file → line → kind → name so downstream stages and the report are
deterministic. These verified symbols later become the *allowlist* the LLM
validator enforces.

---

## 7. `internal/index` — SQLite + FTS5 storage (stage 5)

A CGO-free SQLite database (via `modernc.org/sqlite`) with four regular
tables (`files`, `symbols`, `tech`, `meta`) and one FTS5 virtual table
(`search`) indexing path, symbol names, signatures, docs, and full file
content. `RebuildWithSymbols` wipes and repopulates everything inside a
single transaction, so the DB is always a consistent snapshot of one scan.
Security detail worth noting: `safeJoin`/`insideRoot` verify that every
indexed relative path stays inside the root before its content is read —
even though the paths came from our own scanner, the index re-checks.
`Search` normalizes a natural-language question into an `OR` of quoted
lowercase terms (dropping 1-character tokens) and returns rank-ordered path
hits. This DB is also the *sole* input to the LLM grounding stage.

---

## 8. `internal/diagram` — Mermaid output

Sixty lines that render `architecture.mmd`: a `flowchart TD` with the project
root fanning out to each folder (labeled with its role), each entry point,
and a "Detected Stack" node fanning out to each technology. `nodeID`
sanitizes arbitrary paths into valid Mermaid identifiers; `escape` handles
quotes. Paste the file into any Mermaid renderer for an instant picture.

---

## 9. `internal/grounding` — verified facts for the LLM (anti-hallucination layer 1)

The heart of the "hybrid" design: **the LLM is never shown raw source code —
only verified facts from the database.** `Build` queries `codelens.db` for
the root path, file count, tech stack, entry points, top 50 symbols
(functions/classes/methods/types/tables/views/procedures with file:line), and
folder roles, assembling a read-only `GroundingContext`.

`prompt.go` renders two Go templates with three strictly separated zones:

- **[SYSTEM FACTS — IMMUTABLE]** — the verified data, and nothing else. Raw
  file content is deliberately excluded, which is also the prompt-injection
  defense: text inside the scanned project can never reach the prompt.
- **[TASK]** — instructions with explicit prohibitions ("Do NOT mention any
  symbol … not in the SYSTEM FACTS list").
- **[OUTPUT SCHEMA]** — the exact JSON shape required.

`FormatProjectOverviewPrompt` sorts folder roles into a slice before
rendering, because Go map iteration is random and prompt bytes must be
deterministic run-to-run.

---

## 10. `internal/llm` — subprocess runner, grammars, and loop diagnostics

### 10.1 `runner.go`

The only place in the entire program that spawns a child process — and it's
the separately installed llama.cpp `llama-cli`, never the target project's
code. `Config` makes every resource limit explicit at the call site: model
path, grammar path, threads, `MaxTokens` (512), temperature (0.1), wall-clock
`Timeout`, and `MaxOutputBytes` (8 KB). A custom `limitedWriter` rejects
output past the byte ceiling, defusing runaway generation. Missing binary or
model paths fail fast with clear errors, and timeout errors include the
subprocess's stderr for diagnosis.

### 10.2 `grammars/*.gbnf` (anti-hallucination layer 2)

Two embedded GBNF grammars constrain llama.cpp's sampling itself so the model
is *physically incapable* of emitting anything but the expected JSON shape —
`{"file_summary": …, "symbol_explanations": […]}` for files,
`{"project_description": …}` for the overview, with bare newlines banned
inside strings. `WriteGrammar` copies the embedded grammar to a temp file
(llama-cli needs a path) and returns a cleanup function.

### 10.3 `diagnostics.go` — the BRSL `LoopState` (new in v1.1.0)

Implements the state vector `S = (Σ, p, ρ)` from `KISS_LOOP_EQUATION.md` as
plain counters: `Turn`, `T` (total segments), `P` (validated segments), `Rho`
(0.75 normal / 0.50 repair), `V` (last-turn health), `Stall` (consecutive
frozen turns). `Coverage()` returns γ = P/T truncated to two decimals.
`BeginTurn` captures the coverage/throttle in effect for the imminent call;
`EndTurn(ok)` applies the update rules

```
p ← v·q + (1−v)·p        advance only if the summary validated
ρ ← ½ + ¼·v              75% when healthy, 50% after a failure
```

and prints the `[LLM Loop]` line. `Stalled()` (stall > `StallLimit`=2) and
`ForceAdvance()` implement the atomic-block guard — the one special case the
equation document proved necessary to guarantee termination. The tracker
holds no LLM logic and writes to an injected `io.Writer`, so it is trivially
unit-testable.

---

## 11. `internal/validate` — output validation (anti-hallucination layer 3)

A pure transform between raw model bytes and the report writer, enforcing
three constraints. (1) **Schema conformance:** the bytes must parse as the
expected JSON — with a pragmatic trim to the last `}` since llama-cli
sometimes appends noise. (2) **Symbol allowlist:** every symbol explanation
must name a symbol verified by static analysis; hallucinated names are
discarded and logged, never rendered. (3) **Text hygiene:** absolute paths,
URLs, and backtick-quoted code are regex-stripped from prose, whitespace
collapsed, and fields length-capped (500 chars per file summary, 1000 per
project description). `ValidateFileExplanation` never hard-fails — it returns
whatever valid content survived plus a list of violations, and it is exactly
this errs/summary result that drives the BRSL loop's pass/fail signal.

So the LLM output passes **three independent gates** before a human reads it:
grammar-constrained sampling → fact-grounded prompting → post-hoc validation.
The BRSL loop then adds a fourth axis: *when* a gate rejects, the pipeline
retries smaller instead of silently dropping the file.

---

## 12. Everything else

- **`Makefile`** — `build` (static, CGO-free), `test`, `lint`
  (gofmt + vet), `run`, `smoke` (builds and runs the binary against a fixture
  and asserts all four artifacts exist), `release` (cross-compiles for
  darwin/linux/windows × amd64/arm64, zips/tars, writes SHA-256 checksums),
  `clean`.
- **`scripts/download_llama.sh` / `download_model.sh`** — helper scripts to
  fetch a llama.cpp build and a GGUF model for the optional `--llm` mode;
  the app itself never downloads anything.
- **`testdata/`** — miniature fixture projects (Go, Python, Node, Ruby, Rust,
  C, C++, SQL, Swift, R, plus edge cases like malformed files) that back the
  per-language report tests.
- **`KISS_LOOP_EQUATION.md`** — the mathematical spec the BRSL implements:
  one state vector, one update rule, one stop condition, one invariant
  (everything behind the pointer is validly summarized), and the one flaw it
  exposed (the atomic-unit infinite loop) that `ForceAdvance` guards.
- **`README.md` / `ROADMAP.md` / `walkthrough.md` / `files/`** — user-facing
  docs, milestone history, and the handover report.

---

## 13. Reading order for a new developer

1. `cmd/codelens/main.go` → `internal/cli/cli.go:Run` — see the whole pipeline in 100 lines.
2. `internal/scanner` — understand what a `FileInfo` is; every stage consumes it.
3. `internal/index/store.go` — the schema; both `--ask` and the LLM stand on it.
4. `internal/grounding/prompt.go` — read the two prompts; they explain the entire anti-hallucination philosophy.
5. `KISS_LOOP_EQUATION.md` then `internal/llm/diagnostics.go` then `cli.go:summarizeFiles` — the math, the state machine, the wiring.
6. `internal/validate/validate.go` — the last gate before the report.
