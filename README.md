# CodeLens Local

CodeLens Local is a source-code exploration project written in Go. It is designed to index files, identify technologies and symbols, and help readers navigate a codebase and its architecture.

**Status:** Developer-tool project. The current build, completeness of indexing, and optional features were not independently verified in this review.

## How it is organized

- [cmd/](cmd/) contains entry points; [internal/](internal/) contains application components.
- [go.mod](go.mod) declares the Go toolchain and dependencies, including Fyne and SQLite.
- [PROJECT_NOTES.md](PROJECT_NOTES.md) retains detailed setup, usage, and architecture information.
- [testdata/](testdata/) contains test inputs; tests do not by themselves establish coverage for every language or codebase.

The project distinguishes deterministic file and symbol indexing from optional local-model summaries. Review the notes and dependency requirements before building or running; runtime behavior and network isolation have not been established for every configuration.

[Project guide](docs/PROJECT_GUIDE.md) · [File guide](docs/FILE_GUIDE.md) · [License review](docs/LICENSE_REVIEW.md)

## License

Project code is distributed under the [MIT License](LICENSE). Dependencies and other assets retain their own terms; see the [license review](docs/LICENSE_REVIEW.md) and [dependency inventory](docs/DEPENDENCIES.csv).
