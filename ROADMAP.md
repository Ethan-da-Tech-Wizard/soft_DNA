# Roadmap

CodeLens Local is a small offline codebase explainer. The path stays intentionally simple: scan, detect, index, report, diagram, local retrieval Q&A, and optional local-LLM summarization.

## v1.0 Complete

- CLI workflow
- Scanner and inventory
- Stack detection from manifests
- Folder structure and entry-point detection
- SQLite/FTS5 index
- Lightweight symbol extraction
- Mermaid architecture diagram
- Local retrieval answer with file citations
- Cross-platform release builds with checksums
- C++ and CMake project support
- Optional local model summarization, offline only (hybrid LLM pass)

## v1.2 Complete (Diagnostics & Integrity Gate)

- Real-time BRSL loop diagnostics (`[LLM Loop]` coverage/throttle/health/stall lines; see `KISS_LOOP_EQUATION.md`)
- Validation-failure retries at half chunk, with an atomic-block stall guard
- Bounded source excerpts grounding the per-file LLM prompts
- Project overview covered by the diagnostics loop; `--llm-max-files 0` = uncapped
- Grounded `--ask`: FTS retrieval + one local LLM call = cited plain-English answer, with FTS-only fallback
- Hardened non-Go symbol extraction (multi-line signatures, decorators/attributes, C/C++ statement filter)
- Manifest detection fixes (Makefile no longer implies C without `.c` sources)

## v1.3 Complete (Native Desktop App)

- `codelens-gui`: a real, native desktop window (Fyne) — no browser, no
  localhost, no server for the user to run or think about
- Same pipeline as the CLI end to end (`internal/cli`'s exported
  `RunPipeline`/`WriteMarkdownReport`/`Answer`); nothing reimplemented
- Native tabbed report view (Summary, Tech Stack, Folders, Entry Points, a
  filterable Functions & Classes table, Imports, AI Summaries as cards),
  built directly from structured data rather than by parsing Markdown
  or rendering it in a browser
- Architecture diagram drawn on a native canvas (no Mermaid renderer, no
  image export step)
- Chat-style Ask tab over the grounded `--ask` pipeline
- Settings dialog for the local LLM binary/model/threads, persisted as
  JSON under the OS user config directory — never uploaded anywhere

## Next Polish

- Improve report wording on larger real projects.
- Improve setup guesses for common Go, Node, and Python layouts.
- Keep CI green on every push.

## Later, Only If Needed

- Deeper parsing with `github.com/odvcencio/gotreesitter`.
- Better ranking for `--ask` results.
- Future language extensions — see [Language Support Roadmap](files/08_LANGUAGE_SUPPORT_ROADMAP.md).

## Not Planned

- Cloud upload
- Web/browser-based dashboard (a native desktop GUI shipped in v1.3 instead)
- VS Code extension
- Multi-turn conversational memory across Ask questions (each question is
  answered independently, grounded fresh each time)
- Any target-code execution
