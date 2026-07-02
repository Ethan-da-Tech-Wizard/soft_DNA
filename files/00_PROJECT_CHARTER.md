# CodeLens Local — Project Charter

**Document status:** Baseline / locked for v1
**Owner:** Project lead
**Audience:** Humans and AI build agents

---

## 1. Vision

A developer drops in a project folder and gets back a clear, human-readable explanation of how the whole codebase works — tech stack, structure, entry points, key files, and a diagram — generated **entirely on their own machine**, with no source code ever leaving the device.

One-line description (use this everywhere — recruiter-safe, accurate):

> **A local codebase analysis and documentation generator. Point it at a project folder; it scans the files, detects the tech stack, indexes the source, and produces a plain-English architecture report.**

## 2. Problem statement

Developers regularly inherit unfamiliar codebases (new job, open-source contribution, an old project of their own) and burn hours just figuring out *what is where* and *how it fits together*. Existing tools either require uploading code to a cloud service (a non-starter for private/proprietary code) or are heavyweight IDE features that don't produce a shareable summary. There is room for a small, fast, offline utility that produces a self-contained report.

## 3. Goals

- **G1 — Understandability.** Output explains a codebase in plain English tied to concrete files/functions, not vague AI fluff.
- **G2 — Local & private.** Runs fully offline. Source code never leaves the machine.
- **G3 — Fast & simple.** Single binary, no install ceremony, indexes a typical repo in seconds.
- **G4 — Resume-grade engineering.** Demonstrates file traversal, static analysis, indexing, full-text search, and clean architecture.
- **G5 — Buildable in layers.** Each milestone produces something usable; no big-bang release.

## 4. Non-goals (v1)

- Not a linter, formatter, security scanner, or auto-fixer.
- Not a cloud service, web app, or multi-user system.
- Not an IDE/VS Code extension (possible *future* phase, explicitly out of v1).
- Does not run, build, or execute the analyzed project. **Read and parse only.**

## 5. Success criteria

| # | Criterion | Target |
|---|-----------|--------|
| S1 | Indexes a 1,000+ file repo | < 5 seconds on a normal laptop |
| S2 | Stack detection accuracy on the test fixtures | 100% of fixture projects identified correctly |
| S3 | Report is useful without reading code | A peer can describe the project's purpose & layout from the report alone |
| S4 | Distribution | Single binary per OS, no runtime/toolchain required by the end user |
| S5 | Determinism | Same input folder → byte-identical report (ordering stable) |

## 6. Technical Decision Record (ADR-0001): Language & stack

**Decision:** Build in **Go**, with a **pure-Go, `CGO_ENABLED=0`** dependency set.

**Context.** The brief asks for "simple, clean, efficient code that works perfectly," with an emphasis on speed and easy distribution. Candidate languages were Go, Rust, C++, C#, and Python (the original proposal).

**Key insight that drives the decision.** For a codebase analyzer the dominant cost is **disk I/O and the native parsing/search libraries**, not the host language's arithmetic speed. Real-world pure-Go tools of this exact type index a 1,100+ file repository in well under a second, I/O-bound. Therefore "I need C/C++/Rust for speed" does not apply; runtime differences between these languages are invisible for this workload.

**Why Go wins for the stated goals:**

- Memory-safe and garbage-collected — eliminates an entire class of bugs vs C/C++.
- Compiles to a **single static binary** → solves the packaging/distribution pain that plagued the original Python plan.
- Simple, readable idioms → easy for both a solo developer and AI agents to write correctly.
- Trivial parallelism (goroutines) for the file-scanning phase.
- Fast compile/test loop.

**Why not the others:**

| Option | Reason rejected for v1 |
|--------|------------------------|
| C++ | Manual memory management = least memory-safe; CMake/dependency friction; contradicts "simple + safe." |
| Rust | Excellent and faster, but the learning curve (borrow checker) contradicts "simple, ship fast." Reconsider for a v2 rewrite if desired. |
| C# | Pleasant language, but weaker source-parsing ecosystem and heavier self-contained binaries. |
| Python | Fastest to prototype but distribution/packaging pain (already encountered); not the goal here. |

**Pure-Go stack (no C compiler required):**

- **SQLite + FTS5** via `modernc.org/sqlite` (pure Go; FTS5 supported).
- **Source parsing** starts with lightweight safe extractors for M4. If deeper tree-sitter parsing is needed, the approved parser is **`github.com/odvcencio/gotreesitter`**.
- **Parser exclusion:** do **not** use `github.com/tree-sitter/go-tree-sitter` in v1 unless the pure-Go/no-CGo rule is explicitly changed by ADR. The official binding exposes C-backed allocation patterns and does not match the operational simplicity target.
- **Build:** `CGO_ENABLED=0` → one command cross-compiles to macOS, Linux, Windows (arm64 + amd64).

> Version note for agents: use **Go 1.25 as the minimum supported language version** and build with the **current stable Go 1.26.x toolchain**. Avoid hardcoding patch versions across docs. Resolve dependency versions during implementation, then **pin and commit exact versions** in `go.mod`/`go.sum`; do not leave floating `@latest` instructions in build docs.

## 7. Guiding principles (apply to every decision)

1. **Simplicity first.** When two designs work, choose the one with fewer moving parts. No premature abstraction.
2. **Static analysis only.** Never execute, import, build, or run the target code. Parse and read bytes only.
3. **Local & private by default.** No network calls for core features. Any future LLM feature uses a local model and never transmits source.
4. **Deterministic output.** Stable ordering everywhere so reports diff cleanly and tests are reproducible.
5. **Fail soft.** A single unreadable or malformed file must never crash a scan — log it and continue.
6. **Layered build.** Ship a usable artifact at every milestone; never block on a future feature.
