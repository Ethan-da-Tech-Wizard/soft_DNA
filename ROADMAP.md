# Roadmap

CodeLens Local is a small offline codebase explainer. The current v1 path is intentionally simple: scan, detect, index, report, diagram, and local retrieval Q&A.

## v1 Complete

- CLI workflow
- Scanner and inventory
- Stack detection from manifests
- Folder structure and entry-point detection
- SQLite/FTS5 index
- Lightweight symbol extraction
- Mermaid architecture diagram
- Local retrieval answer with file citations
- Cross-platform release builds

## Next Polish

- Improve report wording on larger real projects.
- Improve setup guesses for common Go, Node, and Python layouts.
- Add checksums for release binaries.
- Keep CI green on every push.

## Later, Only If Needed

- Deeper parsing with `github.com/odvcencio/gotreesitter`.
- Better ranking for `--ask` results.
- Optional local model summarization, offline only.

## Not Planned For v1

- Cloud upload
- GUI
- VS Code extension
- Model-based chat by default
- Any target-code execution
