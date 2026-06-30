# CodeLens Local — Agent Task Prompts

**Status:** Ready to use
**Purpose:** Copy-paste prompts for safe, narrow implementation work. Each prompt is scoped to one section. Do not combine sections unless a human explicitly approves it.

## Section 1 — Documentation Polish Prompt

```text
You are working on CodeLens Local documentation only. Do not write application code.

Read these files in order:
1. files/README.md
2. files/00_PROJECT_CHARTER.md
3. files/01_PRD.md
4. files/02_SRD.md
5. files/03_BUILD_PLAN.md
6. files/04_AGENT_GUIDE.md
7. files/05_ADR_LOG.md
8. files/06_SECURITY_MODEL.md

Goal:
Make the docs internally consistent and safe for an AI build agent.

Required checks:
- Go version language says: minimum Go 1.25; build with current stable Go 1.26.x.
- No docs hardcode a Go patch version unless explicitly marked as an example.
- Dependency guidance says: resolve, pin, and commit exact versions in go.mod/go.sum.
- No durable instruction tells agents to use go get <module>@latest.
- Parser path is lightweight safe extraction first; github.com/odvcencio/gotreesitter is the approved deeper parser if needed.
- github.com/tree-sitter/go-tree-sitter is explicitly excluded for v1 unless an ADR changes it.
- Deterministic outputs exclude timestamps by default.
- M0 is only CLI path validation, output directory creation, report header, --version, and tests.

Safety:
Do not expand product scope. Do not add GUI, LLM, cloud, VS Code extension, or scanner work to M0.

Deliverable:
Return a concise changelog and list any remaining inconsistencies with file references.
```

## Section 2 — Repo Skeleton Prompt

```text
You are creating the initial CodeLens Local repository skeleton. Build only the project structure needed for M0. Do not implement scanner, stack detection, indexing, symbol parsing, Mermaid generation, or LLM features.

Read first:
- files/00_PROJECT_CHARTER.md
- files/02_SRD.md
- files/03_BUILD_PLAN.md
- files/04_AGENT_GUIDE.md
- files/05_ADR_LOG.md
- files/06_SECURITY_MODEL.md

Goal:
Create a minimal Go repo skeleton that can support M0.

Allowed work:
- Create go.mod using minimum Go 1.25.
- Create cmd/codelens/main.go.
- Create internal/cli or internal/report only if needed for M0.
- Create testdata/sample_empty.
- Create Makefile targets: build, test, lint, run.
- Add README pointer to the docs if needed.

Rules:
- CGO_ENABLED=0 must build.
- Use stdlib only for M0 unless a dependency is strictly necessary.
- Do not add gotreesitter yet unless deeper parsing is explicitly needed.
- Do not add modernc.org/sqlite yet; indexing starts at M3.
- Do not add network calls.
- Do not execute target code.

Acceptance:
- make build succeeds.
- make test succeeds.
- make lint succeeds.
- codelens --version works.
- codelens ./testdata/sample_empty creates codelens-out/CODEBASE_REPORT.md.
```

## Section 3 — M0 Ticket Prompt

```text
TASK: M0 tiny CLI skeleton
Milestone: M0
Implements: FR-C1, FR-C2 partial, FR-C3 partial
In scope: CLI path validation, output directory creation, report header, --version, tests.
Out of scope: scanner, .gitignore, inventory, stack detection, SQLite, symbols, Mermaid, JSON, config files, GUI, LLM.

Read first:
- files/01_PRD.md M0 row
- files/02_SRD.md CLI requirements
- files/03_BUILD_PLAN.md Phase 0 / Gate G0
- files/06_SECURITY_MODEL.md hard rules

Build:
1. Implement codelens [flags] <path>.
2. Validate that <path> exists and is a directory.
3. Implement --out <dir>, default ./codelens-out.
4. Implement --version.
5. Create the output directory if missing.
6. Write CODEBASE_REPORT.md with a deterministic title/header only.
7. Return exit code 2 for missing or non-directory paths.

Tests:
- Unit test path validation and output path behavior.
- E2E test codelens ./testdata/sample_empty writes CODEBASE_REPORT.md.
- E2E test invalid path returns exit code 2.
- E2E test --version prints a stable version string.

Safety:
- No target code execution.
- No network access.
- No non-deterministic timestamp in output.

Done when:
CGO_ENABLED=0 go build ./cmd/codelens, go test ./..., go vet ./..., and gofmt are clean.
Stop after M0. Do not start M1.
```

## Section 4 — M1 Ticket Prompt

```text
TASK: M1 scanner and inventory
Milestone: M1
Implements: FR-S1 through FR-S7, NFR-S3, NFR-S4, NFR-S5, NFR-M2
In scope: recursive scan, ignore rules, .gitignore best effort, file inventory, line counts, language detection, SHA-256 hash, binary/large-file skip handling, symlink safety, deterministic ordering.
Out of scope: stack detection, SQLite, symbols, Mermaid, JSON report beyond inventory needs, GUI, LLM.

Read first:
- files/02_SRD.md scanner requirements
- files/03_BUILD_PLAN.md Phase 1 / Gate G1
- files/06_SECURITY_MODEL.md threats and controls

Build:
1. Implement scanner package with typed FileInfo and Scan(root, opts).
2. Exclude default junk directories.
3. Respect root .gitignore on a best-effort basis.
4. Record relative path, extension, size bytes, line count, detected language, SHA-256 content hash, binary flag, skipped flag, and skip reason.
5. Skip binary files and files over the configured cap without reading full content into memory.
6. Never follow symlinks that resolve outside root.
7. Continue after per-file read errors and record warnings.
8. Sort all output deterministically.

Tests:
- Ignore-rule unit tests.
- Size-cap and binary-detection tests.
- Symlink escape test.
- Read-error fail-soft test if practical on the OS.
- Golden inventory test for sample_python and edge_cases fixtures.
- Determinism test: two scans produce identical ordered inventory.

Safety:
- No target code execution.
- No network access.
- No path traversal or out-of-root symlink reads.
- Bounded memory behavior for large files.

Done when:
Gate G1 is green, gofmt/go vet/go test pass, and CGO_ENABLED=0 go build succeeds.
Stop after M1. Do not start M2.
```

## Section 5 — M2 Planning Prompt

```text
You are not implementing M2 yet. Prepare a short plan only.

Goal:
Plan stack detection and structure analysis after M1 is fully complete.

Read:
- files/02_SRD.md FR-D and FR-T
- files/03_BUILD_PLAN.md Phase 2 / Gate G2

Output:
- Proposed fixtures.
- Exact manifest detection rules.
- Entry-point conventions.
- Report sections 1-5 outline.
- Tests required for Gate G2.

Safety:
Do not write code. Do not modify scope. Do not add dependencies unless justified.
```
