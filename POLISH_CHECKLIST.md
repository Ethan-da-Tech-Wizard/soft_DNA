# CodeLens Local — Polish Checklist

A running list of small, concrete imperfections in the app (CLI + desktop
GUI), worked one bit at a time. **Keep it simple, stupid**: every item below
is sized to be one small, independently-verifiable change — a bug fix or a
narrow UX gap, never a redesign. This file is updated after every bit lands
(checked off, or a new one added if found along the way).

Legend: `[ ]` not started · `[~]` in progress · `[x]` done (commit noted)

---

## Desktop GUI (`codelens-gui`)

- [ ] **Log panel doesn't auto-scroll** — new `[LLM Loop]` lines append but
      the visible scroll position doesn't follow them during a long scan.
      _(`appendLog` sets `CursorRow` but that doesn't move the containing
      `Scroll`'s viewport.)_
- [ ] **Enter key in the path field does nothing** — you have to reach for
      the mouse to click Scan even after typing/pasting a path.
- [ ] **Ask tab can double-submit** — pressing Enter rapidly (or Enter then
      clicking Send before the first answer returns) fires two concurrent
      questions; the Send button disables but `OnSubmitted` doesn't check
      that guard.
- [ ] **Last project path isn't remembered** — closing and reopening the
      app always starts from an empty path field, even though Settings
      already persists other preferences the same way.
- [ ] **No indeterminate progress indicator while scanning** — the only
      feedback during a scan is the static "Scanning…" label and whatever
      text happens to land in the log; nothing communicates "still working"
      once a long LLM pass goes quiet between files.
- [ ] **Settings dialog silently accepts a broken LLM config** — enabling
      the LLM checkbox with an empty or nonexistent binary/model path saves
      without warning; the user only finds out on the next scan via a log
      warning that easily scrolls past unnoticed.
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
