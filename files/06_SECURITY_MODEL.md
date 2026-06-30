# CodeLens Local — Security Model

**Status:** Baseline v1
**Audience:** Humans and AI build agents

This project analyzes untrusted source folders. Treat every target repo as hostile input. The tool must read bytes, parse safely, and produce reports without executing code or sending data anywhere.

## 1. Assets to protect

- User source code and file paths.
- Local machine integrity.
- Deterministic report output.
- User trust that the tool is offline and static-analysis-only.

## 2. Hard rules

1. **No execution.** Never run, import, build, eval, install, shell out to, or load code from the target repo.
2. **No egress.** Core operation makes no network calls, telemetry, update checks, package downloads, or API requests.
3. **Stay inside root.** Never follow symlinks or resolved paths outside the selected root.
4. **Bound resources.** Enforce file-size caps, max-file caps, parser timeouts, and bounded worker pools.
5. **Fail soft.** Bad files produce warnings and flags in output; they do not crash the scan.
6. **Pin dependencies.** Resolve dependencies intentionally and commit exact versions. Do not leave build instructions that depend on floating `@latest`.

## 3. Threats and controls

| Threat | Control | Test |
|---|---|---|
| Import side effects | Read bytes only; no target execution APIs. | Fixture file writes a sentinel if executed; assert sentinel never appears. |
| Network leakage | No HTTP clients or telemetry in core path. | Run no-egress test around CLI execution. |
| Symlink escape | Resolve real paths before reading; reject out-of-root targets. | Fixture symlink points outside root; assert it is skipped. |
| Huge file or repo exhaustion | Default 2 MB per-file cap, max-file cap, bounded workers. | Edge fixture with huge file and many files. |
| Malformed source | M4 extraction is recoverable and must not panic. | Malformed fixture completes. |
| Non-deterministic output | Stable sorting; no default timestamps. | Run twice; compare bytes. |
| Unsafe dependency drift | Exact `go.mod`/`go.sum`; ADR for risky deps. | CI checks `CGO_ENABLED=0 go build` and dependency review. |

## 4. Dependency policy

- Prefer Go stdlib.
- Accept `modernc.org/sqlite` for SQLite/FTS5 because it supports the no-CGo distribution goal.
- Accept lightweight safe extractors for M4. Accept `github.com/odvcencio/gotreesitter` later if deeper parsing is justified.
- Reject `github.com/tree-sitter/go-tree-sitter` for v1 unless an ADR supersedes ADR-0002.
- Any dependency that adds CGo, network behavior, native binaries, code generation at user runtime, or broad transitive risk needs a new ADR before use.

## 5. Security acceptance gates

Every milestone must preserve:

- `CGO_ENABLED=0 go build` succeeds.
- No target code execution.
- No network egress during CLI runs.
- Out-of-root symlinks are not followed.
- Generated outputs are deterministic for unchanged inputs.

M1 adds symlink and resource-cap tests. M4 adds malformed-source tests.
