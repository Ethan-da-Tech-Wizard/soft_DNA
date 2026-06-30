# CodeLens Local — AI Agent Build Guide

**Status:** Baseline v1 · Read this **before** writing any code.
**Companion docs:** Charter (00), PRD (01), SRD (02), Build Plan (03)

This document tells an AI coding agent how to operate inside this project so the build stays simple, safe, and on-scope.

---

## 1. Read order & ground rules

1. Read **00_PROJECT_CHARTER.md** (why + decisions), then **02_SRD.md** (contracts), **03_BUILD_PLAN.md** (sequence + gates), **05_ADR_LOG.md** (decision history), and **06_SECURITY_MODEL.md** (safety rules).
2. Build **strictly in milestone order** (M0 → M5). Do not start a milestone until the previous **progression gate** is green.
3. Honor the **SCOPE LOCK** in the PRD. If a task seems to require out-of-scope work, **stop and flag it** — do not silently expand scope.
4. When the docs and your instinct conflict, the docs win. If a doc is wrong, propose a doc change first.

## 2. Repository layout (target)

```
codelens-local/
├── cmd/
│   └── codelens/
│       └── main.go          # CLI entry; wires the pipeline
├── internal/
│   ├── scanner/             # FR-S*  file walk, ignore, inventory
│   ├── stackdetect/         # FR-D*  manifest → stack
│   ├── structure/           # FR-T*  folders + entry points
│   ├── symbols/             # FR-Y*  safe symbol extraction (added at M4)
│   ├── index/               # FR-I*  SQLite + FTS5
│   ├── report/              # FR-R*  Markdown report
│   └── diagram/             # FR-R2  Mermaid
├── testdata/                # fixtures (see Build Plan §3)
├── docs/                    # these documents
├── go.mod
├── Makefile                 # build / test / lint / release targets
└── README.md
```

One package per pipeline stage. Packages under `internal/` are not importable externally — keeps the public surface to the CLI only.

## 3. Guardrails (non-negotiable)

- **G-1 Static analysis only.** Never `exec`, `import`, build, or run the target project. Read and parse bytes only. (Tested by the no-exec gate.)
- **G-2 No network in core features.** No HTTP, no telemetry, no package fetching of the *target*. (Tested by the no-egress gate.)
- **G-3 Stay in root.** No path traversal; never follow a symlink that resolves outside the scanned root.
- **G-4 Pure Go.** Keep `CGO_ENABLED=0` building at all times. Prefer stdlib; justify every new dependency. Use `modernc.org/sqlite` for SQLite. If deeper parsing is needed, use `github.com/odvcencio/gotreesitter`; do not use `github.com/tree-sitter/go-tree-sitter` in v1 unless an ADR explicitly changes the pure-Go/no-CGo rule.
- **G-5 Determinism.** Sort every collection that reaches output. Same input ⇒ identical bytes out.
- **G-6 Fail soft.** A single bad file logs a WARN and the run continues; it never panics the process.
- **G-7 Resource caps.** Respect the file-size cap, max-files cap, and parser timeout. Never load a whole repo into memory.
- **G-8 No scope creep.** No GUI, no model-based chat unless explicitly approved and local-only, no VS Code extension, no extra languages beyond Python/JS/TS/Go in v1.

## 4. Coding conventions

- `gofmt` + `go vet` clean before every commit. Idiomatic Go; small functions; explicit errors (wrap with context, don't swallow).
- Logging via `log/slog`; no `fmt.Println` for diagnostics.
- Exported identifiers documented with a comment. Each package has a short package doc.
- Tests live beside code (`_test.go`); fixtures in `testdata/`.
- No global mutable state; pass dependencies explicitly.

## 5. Per-task execution protocol

For each task an agent picks up:

1. **Locate** the task's FR/NFR id and milestone in the SRD/Build Plan.
2. **Confirm scope** — in scope per the PRD Scope Lock? If not, stop and flag.
3. **Ensure a fixture exists** to test against; create one if missing.
4. **Write the test first** (unit + golden/integration as appropriate) describing the expected behavior.
5. **Implement** the smallest change that satisfies the contract.
6. **Run the gates:** `gofmt`, `go vet`, `go build` (CGO off), `go test ./...`, the relevant safety check.
7. **Verify Definition of Done** (Build Plan §6) and the milestone progression gate.
8. **Commit** with a conventional message referencing the FR id. Then stop — do not roll ahead into the next milestone.

For copy-paste-ready task prompts, use **07_AGENT_TASK_PROMPTS.md**. The prompts are intentionally narrow; do not combine multiple sections unless a human explicitly asks.

## 6. How to build, test, verify

```
make build      # CGO_ENABLED=0 go build ./cmd/codelens
make test       # go test ./... (+ -race where applicable) + coverage
make lint       # gofmt -l . ; go vet ./...
make run DIR=./testdata/sample_python
make release    # cross-compile matrix (darwin/linux/windows × amd64/arm64)
```

An agent must run `make lint test build` and see them green before claiming a task is done.

Dependency rule: during implementation, resolve dependency versions intentionally, then pin and commit exact versions in `go.mod` and `go.sum`. Do not leave `@latest` as a durable project instruction.

## 7. Task template (use for every unit of work)

```
TASK: <short title>
Milestone: M<n>
Implements: FR-<id> / NFR-<id>
In scope? : yes  (cite PRD Scope Lock line)
Inputs    : <what the code receives>
Outputs   : <what it must produce, exact shape per SRD>
Fixture   : testdata/<...>
Tests     : <unit + golden/integration + any safety gate>
Acceptance: <copied from the milestone exit criteria>
Done when : lint+vet+build(CGO off)+tests green AND progression gate green
```

## 8. When to ask a human

- A requirement is ambiguous or two docs disagree.
- A task appears to need out-of-scope work.
- A dependency choice would add CGo, network access, or significant surface area.
- A safety guardrail (G-1…G-3) seems to be in the way of a requirement — never work around it silently.

---

### Tagline (for the README)

> **CodeLens Local** — a free, offline codebase explainer. Point it at a project folder; it detects the tech stack, indexes the source, and generates a plain-English architecture report — without your code ever leaving your machine.
