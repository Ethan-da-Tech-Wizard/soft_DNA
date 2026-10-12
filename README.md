# CodeLens Local

An offline codebase explorer that indexes source files, detects stacks, maps symbols, and produces architecture reports.

**Status:** Developer tool  
**Built with:** Go, SQLite, native GUI components

[Project guide](docs/PROJECT_GUIDE.md) · [Every file explained](docs/FILE_GUIDE.md) · [Licenses and dependencies](docs/LICENSE_REVIEW.md)

## Start here

- [PROJECT_NOTES.md](PROJECT_NOTES.md) — Preserved detailed setup, usage, architecture, and development notes from the earlier README.
- [go.mod](go.mod) — Go module identity, toolchain version, and dependency requirements; paired with go.sum for downloaded module checksums.
- [cmd](cmd) — Project entry point and supporting files
- [internal](internal) — Project entry point and supporting files

[Detailed setup, usage, and development notes](PROJECT_NOTES.md) preserve the previous project documentation.

## Repository layout

| Path | Role |
| --- | --- |
| [.github/](.github/) | GitHub workflow and contributor configuration (1 tracked entries). |
| [cmd/](cmd/) | Project-specific cmd components (3 tracked entries). |
| [docs/](docs/) | Project documentation, decisions, and reference material (5 tracked entries). |
| [files/](files/) | Project-specific files components (12 tracked entries). |
| [internal/](internal/) | Project-specific internal components (48 tracked entries). |
| [scripts/](scripts/) | Build, launch, maintenance, and verification commands (2 tracked entries). |
| [testdata/](testdata/) | Project-specific testdata components (30 tracked entries). |

## Scope and current limits

Optional local model-assisted summaries are distinct from deterministic symbol and file indexing. Upstream Go and GUI dependencies retain their own notices.

## License

Original project code retains **MIT** terms in [LICENSE](LICENSE). Third-party libraries, media, data, models, and tools retain their own terms. See the [license review](docs/LICENSE_REVIEW.md) and [dependency inventory](docs/DEPENDENCIES.csv).
