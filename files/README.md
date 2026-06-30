# CodeLens Local — Documentation Set

> **CodeLens Local** — a free, offline codebase explainer. Point it at a project folder; it detects the tech stack, indexes the source, and generates a plain-English architecture report — without your code ever leaving your machine.

This folder is the **foundation** for the project: everything an engineer or an AI build agent needs to start building, with nothing built yet. No application code lives here — only the framework, contracts, scope, and plan.

## Headline decisions (locked)

- **Language:** **Go** (minimum 1.25; build with the current stable 1.26.x toolchain), **pure-Go / `CGO_ENABLED=0`** → single static binary, easy cross-compile, memory-safe, simple. *Why not C++/Rust/Python: see ADR-0001 in the Charter — for this tool the bottleneck is disk I/O, not language speed.*
- **Parser rule:** v1 starts with lightweight safe extractors. If deeper tree-sitter parsing is added, use **`github.com/odvcencio/gotreesitter`**. Do **not** use `github.com/tree-sitter/go-tree-sitter` for v1 because it exposes C-backed allocation patterns and conflicts with the pure-Go/no-CGo constraint.
- **MVP:** a **CLI** that turns a folder into `CODEBASE_REPORT.md` + Mermaid diagram (+ optional JSON index). No GUI, no LLM, no VS Code extension in v1.
- **Hard safety rule:** the tool **reads and parses code only — it never executes it**, and makes **no network calls**.

## Read in this order

| # | Document | What it gives you |
|---|----------|-------------------|
| 0 | [00_PROJECT_CHARTER.md](00_PROJECT_CHARTER.md) | Vision, problem, goals/non-goals, success criteria, the tech decision record, guiding principles. |
| 1 | [01_PRD.md](01_PRD.md) | Users, user stories, features (MoSCoW), **Scope Lock**, milestones M0–M5. |
| 2 | [02_SRD.md](02_SRD.md) | Functional + non-functional requirements, architecture, module contracts, **SQLite schema**, CLI/output contracts, pinned stack. |
| 3 | [03_BUILD_PLAN.md](03_BUILD_PLAN.md) | Phased build sequence, **test strategy & progression gates**, CI, Definition of Done. |
| 4 | [04_AGENT_GUIDE.md](04_AGENT_GUIDE.md) | How an AI agent operates in the repo: layout, **guardrails**, task protocol, task template. |
| 5 | [05_ADR_LOG.md](05_ADR_LOG.md) | Short decision log so future agents can change architecture intentionally. |
| 6 | [06_SECURITY_MODEL.md](06_SECURITY_MODEL.md) | Threat model and safety controls for no-exec, no-egress, symlinks, resource caps, and dependencies. |
| 7 | [07_AGENT_TASK_PROMPTS.md](07_AGENT_TASK_PROMPTS.md) | Copy-paste prompts for doc polish, repo skeleton, M0, and M1 work. |

## How to use this with an AI agent

1. Give the agent all five documents.
2. Tell it: *"Build Milestone M0 only, following the Agent Guide. Stop at the M0 progression gate."*
3. Review the gate output; then authorize M1; repeat.

This keeps the build incremental, testable, and on-scope — each milestone is a working, shippable slice.

## The build path at a glance

```
M0 tiny CLI skeleton → M1 scanner → M2 stack+structure
→ M3 SQLite/FTS5 index → M4 symbols + diagram + full report
→ (stretch) M5 local retrieval Q&A
```
