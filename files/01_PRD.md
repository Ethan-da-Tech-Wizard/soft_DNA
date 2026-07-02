# CodeLens Local — Product Requirements Document (PRD)

**Status:** Baseline v1 · **Companion docs:** Charter (00), SRD (02), Build Plan (03)

---

## 1. Overview

CodeLens Local is a command-line tool that ingests a project folder and produces a self-contained Markdown report (plus an optional machine-readable index and an architecture diagram) explaining how the codebase is structured and how it works. It runs offline and never transmits source code.

## 2. Target users

| Persona | Need |
|--------|------|
| **New-team developer** | Onboard onto an unfamiliar repo fast. |
| **Open-source contributor** | Understand a project before making a PR. |
| **The author (you)** | Re-understand an old project; produce documentation; demonstrate engineering skill. |
| **Reviewer / lead** | Get a one-page picture of an unknown project. |

## 3. User stories (v1)

- **US1** — As a developer, I run one command against a folder and get a `CODEBASE_REPORT.md` I can read top-to-bottom.
- **US2** — As a developer, the report tells me the **tech stack** so I know what I'm looking at.
- **US3** — As a developer, the report gives a **folder-by-folder** explanation so I know where things live.
- **US4** — As a developer, the report lists **entry points and important files** so I know where to start reading.
- **US5** — As a developer, the report lists **functions/classes with locations** so I can jump to them.
- **US6** — As a developer, I get an **architecture diagram** (Mermaid) I can paste into docs.
- **US7** — As a developer, I trust the tool ran **offline** and **did not execute** my code.
- **US8** *(later milestone)* — As a developer, I can **ask a question** about the codebase and get a local retrieval answer with file references.

## 4. Feature list (MoSCoW)

**Must have (v1):**
- Recursive folder scan with junk-folder/ignore rules and `.gitignore` awareness.
- File inventory (path, extension, size, line count, detected language).
- Tech-stack detection from manifest files.
- Folder structure summary + entry-point detection.
- Symbol extraction (functions, classes, imports) for the priority languages.
- SQLite index with FTS5 over file/symbol content.
- Markdown report generator with the standard section set.
- Mermaid architecture diagram generation.
- Single-binary build for macOS/Linux/Windows.

**Should have:**
- JSON index output (machine-readable, for tooling/agents).
- Per-file and per-function plain-English summaries (heuristic/template-based, no LLM).
- `--ignore` / `--include` flags and config file.

**Could have (post-v1):**
- Local Q&A over the index: question → FTS5 retrieval → answer **with file references**. A local model can be added later only if it stays offline and optional.
- Dependency graph between internal modules.

**Won't have (v1 — explicitly out):**
- Cloud upload, accounts, multi-user, web UI, VS Code extension, auto-fixing/refactoring, language-server (LSP) integration.

## 5. SCOPE LOCK

This is the contract. Changes require an explicit decision and a charter update; agents must not silently expand scope.

**IN SCOPE (v1):**
1. CLI tool, single binary.
2. Languages for symbol extraction, in priority order: **Python, JavaScript/TypeScript, Go**. (Others: detected & inventoried, but symbol extraction is best-effort.)
3. Outputs: `CODEBASE_REPORT.md` (required), `index.json` (should), `architecture.mmd` Mermaid (required).
4. SQLite + FTS5 index, written to a local file.
5. Offline, static-analysis-only operation.

**OUT OF SCOPE (v1) — deferred, do not build yet:**
1. Any LLM/AI chat or generation (Milestone 5+, behind a flag, local model only).
2. GUI of any kind (Tkinter/PySide/web/desktop).
3. VS Code extension.
4. Multi-language symbol extraction beyond the three priority languages.
5. Anything that executes, builds, installs, or network-fetches the target project.
6. Incremental/watch mode, git history analysis, blame, metrics dashboards.

## 6. Milestones

Each milestone ends in a **demoable, shippable** artifact. Exit criteria are the gate to the next milestone (see Build Plan for the test gates).

| ID | Milestone | Goal | Exit criteria |
|----|-----------|------|---------------|
| **M0** | Tiny CLI skeleton | `codelens <folder>` validates a path, creates the output directory, and writes a near-empty report. | Binary builds; `--version` works; invalid paths return exit code 2; writes a `CODEBASE_REPORT.md` with a header; one unit test and one e2e test pass. No scanner logic yet. |
| **M1** | Scanner + inventory | Recursive scan, ignore rules, file inventory. | Inventory correct on fixture repos; junk folders excluded; large/binary files handled; deterministic ordering. |
| **M2** | Stack detection + structure | Detect stack from manifests; folder summary; entry points. | Correct stack string for every fixture; entry points identified; report sections 1–5 populated. |
| **M3** | Index (SQLite + FTS5) | Persist files/symbols; full-text search works. | DB schema created; FTS5 query returns expected hits on fixtures; index rebuild is idempotent. |
| **M4** | Symbol extraction + full report | Extract functions/classes/imports (Py/JS/TS/Go); Mermaid diagram; complete report. | Symbols extracted with correct line numbers on fixtures; Mermaid renders; all 10 report sections populated; golden-file tests pass. |
| **M5** *(stretch)* | Local Q&A | Ask questions; retrieve from FTS5; write `ANSWER.md` with file references. | Offline; answers cite source files; no network egress (verified). |

## 7. Product-level acceptance criteria

- Running the tool on each fixture repo produces a report whose stack/entry points/structure a reviewer agrees are correct.
- The tool makes **zero network connections** during a run (verifiable).
- The tool never executes target code (no `import`/`exec`/subprocess of the target).
- Re-running on an unchanged folder yields an identical report.
