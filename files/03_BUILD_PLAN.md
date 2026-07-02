# CodeLens Local — Build Plan & Test Strategy

**Status:** Baseline v1 · **Companion docs:** Charter (00), PRD (01), SRD (02), Agent Guide (04)

How the project gets built, in what order, and exactly what must pass to move from one milestone to the next.

---

## 1. Build philosophy

- **Walking skeleton first.** Get an end-to-end run working (folder in → file out) before adding real logic. This de-risks wiring early.
- **Vertical slices.** Each milestone adds one complete, usable capability across the whole pipeline — not a half-finished layer.
- **Test before progression.** A milestone is "done" only when its **progression gate** (below) is green. No skipping ahead.
- **Keep it boring.** Smallest dependency set, plain stdlib where possible, no cleverness that an agent or a future-you can't read at a glance.

## 2. Phased plan (mapped to milestones)

### Phase 0 — Project bootstrap (→ M0)
- Init repo, `go.mod`, module layout (see Agent Guide §2), `Makefile`/task runner, `CI` skeleton.
- Implement minimal `cli` only: path validation, output directory creation, report header generation, `--version`, and tests.
- Do **not** implement scanner, stack detection, indexing, symbols, Mermaid generation, config files, or `.gitignore` logic in M0.
- **Deliverable:** building binary that runs end-to-end on any folder.
- **Gate G0:** `go build` succeeds with `CGO_ENABLED=0`; `codelens --version` prints a version; `codelens ./testdata/sample_empty` writes `CODEBASE_REPORT.md` containing the header; invalid paths return exit code 2; 1 unit test + 1 e2e test pass in CI.

### Phase 1 — Scanner & inventory (→ M1)
- Implement `scanner`: recursive walk, ignore rules, `.gitignore`, size/binary handling, symlink safety, content hash, concurrent workers with deterministic output.
- **Gate G1:**
  - Unit tests for ignore rules, size cap, binary detection, symlink-escape rejection.
  - Golden test: inventory of `sample_python` and `sample_tauri` fixtures matches expected snapshot.
  - Determinism test: two runs → identical inventory ordering.

### Phase 2 — Stack detection & structure (→ M2)
- Implement `stackdetect` (manifest → stack string + evidence) and `structure` (folder roles + entry points).
- Populate report sections 1–5.
- **Gate G2:**
  - For every fixture, the detected stack string equals the expected string.
  - Entry points for each fixture match expected set.
  - Golden test on report sections 1–5.

### Phase 3 — Index with SQLite + FTS5 (→ M3)
- Implement `index.Store` over `modernc.org/sqlite`; create schema; FTS5 search; idempotent upsert keyed by content hash.
- **Gate G3:**
  - Integration test: build index from fixture, run FTS5 queries (e.g., search "login", "main") → expected file/symbol hits.
  - Idempotency test: index twice → row counts stable, no duplicates.
  - Schema migration/init test.

### Phase 4 — Symbol extraction, diagram, full report (→ M4)
- Add safe symbol extraction for Python, JS/TS, Go; populate `symbols` table. Start lightweight, and use `github.com/odvcencio/gotreesitter` only when deeper parsing is justified.
- Do not use `github.com/tree-sitter/go-tree-sitter` unless an ADR explicitly changes the pure-Go/no-CGo parser decision.
- Generate Mermaid diagram; complete report sections 6–10; optional `index.json`.
- **Gate G4:**
  - Symbol tests per language: known fixture file → expected symbols with correct line numbers.
  - Malformed-file test: parser does not crash, file is flagged, run completes.
  - Mermaid output passes a syntax validation (renders / parses).
  - Golden test on the **full** 10-section report for each fixture.
  - **Release gate:** cross-compile matrix (darwin/linux/windows × amd64/arm64) all build; `--version` works; no network egress observed during a run.

### Phase 5 — Local Q&A (stretch, → M5)
- FTS5 retrieval → deterministic `ANSWER.md` with source file references. A local model can be added later only if it stays optional, offline, and dependency-light.
- **Gate G5:** offline test returns an answer citing real files; egress monitor confirms zero outbound connections.

## 3. Test strategy

### Test pyramid
- **Unit (most):** pure functions — ignore rules, language detection, line counting, signature formatting, Mermaid string building. Fast, no I/O.
- **Golden / snapshot:** run a stage against a committed fixture repo; compare output to a checked-in expected file. Update snapshots only via an explicit, reviewed `-update` flag.
- **Integration:** scanner→index→search across a real fixture; DB created in a temp dir.
- **End-to-end (CLI):** invoke the built binary on a fixture; assert exit code + generated files + key content.
- **Property/robustness:** feed malformed/huge/binary files; assert "fail soft" (no crash, run completes).
- **Security/safety checks (gating):**
  - **No-exec:** a fixture containing a file that would side-effect if imported/run (e.g., writes a sentinel file on import) — assert the sentinel is **never** created.
  - **No-egress:** run under a network monitor / `CGO_ENABLED=0` offline; assert zero outbound connections.
  - **No-escape:** a symlink pointing outside root is not followed.

### Fixtures (commit these under `testdata/`)
- `sample_python/` — Flask/FastAPI-ish app with `requirements.txt`, classes, functions, `main.py`.
- `sample_node/` — `package.json` + `tsconfig.json` + React-ish `src/`.
- `sample_go/` — `go.mod` + `cmd/app/main.go` + packages.
- `sample_tauri/` — mixed Rust + TS to exercise multi-stack detection.
- `edge_cases/` — empty files, huge file, binary file, malformed source, deep nesting, symlink-out, a `__pycache__`/`node_modules` that must be ignored, an import-side-effect file for the no-exec test.

### Coverage
- ≥ 70% on `scanner`, `stackdetect`, `symbols`, `index`. Report (`report`/`diagram`) covered by golden tests rather than line coverage.

## 4. CI pipeline (per push / PR)
1. `gofmt`/`go vet` (formatting + static checks).
2. `go build` with `CGO_ENABLED=0`.
3. `go test ./... -race` (where applicable) + coverage.
4. Golden tests.
5. (On tag) cross-compile matrix + attach binaries.

## 5. Definition of Ready (before starting a task)
- The task maps to a specific FR/NFR id and a milestone.
- Inputs, outputs, and acceptance criteria are written down.
- A fixture exists (or is created first) to test against.

## 6. Definition of Done (before closing a task)
- [ ] Code matches the SRD contract for the relevant FR/NFR.
- [ ] Unit + relevant golden/integration tests added and passing.
- [ ] `gofmt`/`go vet` clean; builds with `CGO_ENABLED=0`.
- [ ] No new network calls; no execution of target code.
- [ ] Output is deterministic.
- [ ] The milestone's **progression gate** is green.

## 7. Branching & commits
- Trunk-based: short-lived branches `mN/<short-desc>`, merge to `main` when the gate is green.
- Conventional commits: `feat(scanner): ...`, `test(index): ...`, `fix(symbols): ...`, referencing the FR id.
- `main` is always buildable and gate-passing.
