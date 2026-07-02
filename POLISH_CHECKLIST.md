# CodeLens Local — Polish Checklist

A running list of small, concrete imperfections in the app (CLI + desktop
GUI), worked one bit at a time. **Keep it simple, stupid**: every item below
is sized to be one small, independently-verifiable change — a bug fix or a
narrow UX gap, never a redesign. This file is updated after every bit lands
(checked off, or a new one added if found along the way).

Legend: `[ ]` not started · `[~]` in progress · `[x]` done (commit noted)

---

## Desktop GUI (`codelens-gui`)

- [x] **Log panel doesn't auto-scroll** — `widget.Entry` manages its own
      internal scroll privately; no public API lets outside code drive it.
      Fixed by switching the log to a plain `widget.Label` inside a
      `container.Scroll` this code owns, calling `ScrollToBottom()` after
      every append. Verified: scan produced multiple `[LLM Loop]` lines,
      all visible without manual scrolling.
- [x] **Enter key in the path field does nothing** — added
      `pathEntry.OnSubmitted` calling `startScan()`. Verified: typed a path
      and hit Enter with the mouse never touching the Scan button; scan ran
      and results rendered.
- [x] **Ask tab can double-submit** — `ask()` now returns early if
      `send.Disabled()`, checked from both the button tap and
      `question.OnSubmitted`, so a question in flight blocks a second one
      from either entry point.
- [x] **Last project path isn't remembered** — added `LastProject` to
      `Settings`; saved on each scan start, restored into the path field in
      `NewApp`. Verified: settings.json captured the path after a scan, and
      a fresh launch pre-filled it.
- [x] **No indeterminate progress indicator while scanning** — added a
      `widget.NewProgressBarInfinite()` next to the "Scanning…" label,
      explicitly `.Stop()`ped in the completion callback (its animation
      loop doesn't stop just because the widget is discarded). Verified via
      a mid-scan screenshot with a deliberately slowed mock LLM.
- [x] **Settings dialog silently accepts a broken LLM config** — added
      `validateLLMConfig`, a pure function checking the binary/model paths
      are non-empty and exist on disk; the Save handler shows a follow-up
      `dialog.ShowInformation` listing specific problems (settings are
      still saved — this warns, it doesn't block). Verified: enabling LLM
      with empty paths produced a dialog naming both missing paths.
- [x] **No way to jump to the on-disk output** — added an "Open Output
      Folder" button on the Summary tab, calling `fyne.App.OpenURL` on a
      `file://` URI for `res.OutDir` (delegates to the OS's default file
      manager — xdg-open/Explorer/Finder — not a bundled subprocess).
      Verified: button renders under the summary text.
- [x] **Default window/taskbar icon is Fyne's generic icon** — generated a
      small flat geometric mark (dark rounded square, accent-blue inset,
      matching the app's existing primary color) procedurally in Go
      (`internal/gui/assets/gen_icon.go`, `//go:build ignore`, kept in-repo
      for reproducibility), embedded via `//go:embed`, wired into both
      `fyne.App.SetIcon` and `Window.SetIcon`. Regression test decodes the
      embedded PNG and checks it's square and non-trivially sized.
- [x] **Diagram tab has no zoom/fit control** — added Zoom In/Out/Reset
      buttons; `newDiagramView` takes a scale factor and the tab rebuilds
      the canvas at the new scale (Fyne has no built-in transform for
      arbitrary containers, so redrawing at scale is the simplest correct
      option). Verified: two clicks on Zoom In visibly enlarged every box,
      line, and label proportionally.
- [x] **Table columns are not sortable by header click** — clicking a
      header cell now sorts all rows by that column (string comparison,
      case-sensitive — e.g. "App" sorts before "main" since uppercase
      precedes lowercase in ASCII; acceptable for a first pass, noted here
      rather than silently glossed over), toggling ascending/descending on
      repeat clicks, with a ▲/▼ marker on the active column. Verified via
      screenshot before/after clicking "Name" on a multi-symbol fixture.
      **Column drag-resizing was scoped out**: it needs a custom Table
      renderer/drag handler in Fyne, a materially bigger lift than the
      per-column-type fixed widths already in place (140/360px by content
      type) justify for a "polish" item — not implemented.

## CLI / GUI parity

- [x] **GUI hardcoded `LLMMaxFiles: 10`** with no way to change it, while
      the CLI exposes `--llm-max-files` (including `0` = uncapped per the
      v1.2 diagnostics-gate work). Added `LLMMaxFiles` to `Settings`
      (defaults to 10 for a new install, an explicit 0 is preserved across
      save/load as "uncapped"), a "Max files to annotate" field in the
      Settings dialog, and wired it into `runScan`'s `PipelineOptions`
      instead of the hardcoded literal. Verified: dialog shows the field
      pre-filled with 10.

## CLI / shared pipeline

- [ ] *(none currently queued — add here as found)*

---

## Log

_(one line per landed bit, newest first)_

- Batch 3: fixed the CLI/GUI `--llm-max-files` parity gap found while
  re-reading `app.go` for this batch — proof the "keep finding things"
  half of this checklist works, not just the "keep fixing things" half.
- Batch 2: open-output-folder button, a real app icon (generated, not a
  placeholder), diagram zoom in/out/reset, sortable table columns (header
  click, ▲/▼ indicator; drag-resize explicitly scoped out as not worth a
  custom Table renderer for a polish item). All four manually verified on a
  virtual display; `internal/gui` still fully green, headless.
- Batch 1: log auto-scroll, Enter-to-scan, Ask double-submit guard,
  remembered last project path, scan progress indicator, Settings LLM-config
  validation warning. All six manually verified end to end on a virtual
  display plus new/updated unit tests (`internal/gui` still fully green,
  headless).

**Status: all 10 originally-found GUI items done, plus 1 CLI/GUI parity gap
found and fixed along the way. Nothing currently queued — next pass would
need a fresh look at the app (or a real-world project run through it) to
find more.**
