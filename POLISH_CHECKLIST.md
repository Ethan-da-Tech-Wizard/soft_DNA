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
- [ ] **No way to jump to the on-disk output** — after a scan, the Markdown
      report/diagram/DB are written to `<project>/codelens-out/` but
      there's no button in the window to reveal that folder for someone who
      wants to copy/share those files.
- [ ] **Default window/taskbar icon is Fyne's generic icon**, not something
      identifying CodeLens.
- [ ] **Diagram tab has no zoom/fit control** — a project with many folders
      and entry points produces a wide canvas with only scrollbars to
      navigate it.
- [ ] **Table columns are fixed-width and not sortable by header click** —
      long file paths or signatures get truncated with no way to widen a
      column or re-sort by it.

## CLI / shared pipeline

- [ ] *(none currently queued — add here as found)*

---

## Log

_(one line per landed bit, newest first)_

- Batch 1: log auto-scroll, Enter-to-scan, Ask double-submit guard,
  remembered last project path, scan progress indicator, Settings LLM-config
  validation warning. All six manually verified end to end on a virtual
  display plus new/updated unit tests (`internal/gui` still fully green,
  headless). 6 of 10 GUI items done; 0 of 0 CLI items queued so far.
