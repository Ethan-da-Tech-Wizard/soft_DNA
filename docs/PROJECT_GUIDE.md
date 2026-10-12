# CodeLens Local

An offline codebase explorer that indexes source files, detects stacks, maps symbols, and produces architecture reports.

**Status:** Developer tool  
**Built with:** Go, SQLite, native GUI components

## Reading order

1. Read the root README for project purpose and setup navigation.
2. Follow the entry points below to find the implementation.
3. Use the file guide to locate individual modules, tests, assets, and configuration.
4. Check component licenses before redistributing dependencies or assets.

## Important paths

- [PROJECT_NOTES.md](../PROJECT_NOTES.md) — Preserved detailed setup, usage, architecture, and development notes from the earlier README.
- [go.mod](../go.mod) — Go module identity, toolchain version, and dependency requirements; paired with go.sum for downloaded module checksums.
- [cmd](../cmd) — Project-specific components
- [internal](../internal) — Project-specific components

## Structure

| Path | Role |
| --- | --- |
| [.github/](../.github/) | GitHub workflow and contributor configuration (1 tracked entries). |
| [cmd/](../cmd/) | Project-specific cmd components (3 tracked entries). |
| [docs/](../docs/) | Project documentation, decisions, and reference material (5 tracked entries). |
| [files/](../files/) | Project-specific files components (12 tracked entries). |
| [internal/](../internal/) | Project-specific internal components (48 tracked entries). |
| [scripts/](../scripts/) | Build, launch, maintenance, and verification commands (2 tracked entries). |
| [testdata/](../testdata/) | Project-specific testdata components (30 tracked entries). |

## Scope

Optional local model-assisted summaries are distinct from deterministic symbol and file indexing. Upstream Go and GUI dependencies retain their own notices.

## Maintaining the guide

The file catalog is a source-inspection snapshot, not proof that every runtime path works. Update it when moving files or changing module responsibilities. The 2026-10-12 documentation review did not run every application, deploy services, or validate all historical performance claims.
