# ARCHITECTURE_TODO.md — Architecture Explainer (planning doc)

> **Guidance tool — do not overwrite.** Check items off and append notes/dates
> as work happens. Keep it short. KISS.
>
> Prefer a practical first version that shows a readable high-level diagram
> over a complex automatic graph that tries to be perfect.

## 1. Purpose

Add a new native section to `codelens-gui` called **Architecture Explainer**.
After a scan, it shows the scanned project's architecture as shapes and
arrows, so a non-expert can see how the parts connect. Clicking a shape
explains it in plain English. Works on any scanned project — including
CodeLens itself when pointed at its own repo.

Native Fyne only: no HTML, no webview, no browser, no localhost.

## 2. MVP scope

- [x] One new tab in the existing scan-results tab bar (`internal/gui/report.go`)
- [x] Draw a high-level diagram of the scanned project: entry points, top-level
      folders/modules, detected stack, persisted outputs — roughly 10–25 shapes,
      not one per file
- [x] Arrows show the main relationships (entry point → module, module → data)
- [x] Click a shape → a side panel explains it (see §4)
- [x] Reuse the scan data already in `cli.PipelineResult` — no new analysis pass

Out of scope for MVP: per-function graphs, automatic dependency inference
beyond what the pipeline already knows, editing/moving shapes, export.

## 3. Shape legend

| Shape | Meaning |
|---|---|
| Rectangle | File or module |
| Square | Important component |
| Circle | Function, service, or focused unit of logic |
| Oval | User action, entry point, or workflow |
| Cylinder | Database, index, local storage, persisted data |
| Cloud | External service, API, model download, outside dependency |
| Arrow | Dependency, data flow, call flow, or user workflow |

MVP can start with rectangles + ovals + cylinders + arrows (what the pipeline
can reliably identify today) and add the rest as detection improves.

## 4. Click / selection behavior

Selecting a shape shows, in a plain-English panel next to the canvas:

- What file, folder, module, or system part it represents (real path)
- What that part does
- What other parts it connects to
- Why those connections matter
- The connection kind: dependency · data flow · entry point · external service · output

Selection should be visually obvious (highlight border). One shape selected at
a time. Clicking empty canvas deselects.

## 5. Data needed (already available — reuse, don't rebuild)

All of this is in `cli.PipelineResult` after `RunPipeline` (`internal/cli/cli.go`):

| Data | Source | Used for |
|---|---|---|
| `Folders` (path + role) | `internal/structure` | Rectangle shapes + grouping |
| `Entries` (entry points) | `internal/structure` | Oval shapes |
| `Stack` (detected tech) | `internal/stackdetect` | Labels; cloud shapes for external deps |
| `Symbols` (funcs/classes per file) | `internal/symbols` | Circle shapes + "what it does" text |
| `Files` | `internal/scanner` | Drill-down contents, counts |
| Index stats / `codelens.db` | `internal/index` | Cylinder shape (persisted data) |
| `LLM.FileSummaries` (optional) | `internal/llm` / `internal/validate` | Richer plain-English text when `--llm` was on |

Explanation text must be grounded in this data — never invented. Without LLM
summaries, fall back to deterministic text (role, symbol counts, connections).

## 6. Where it fits in the Fyne app

- New tab item in `internal/gui/report.go` next to the existing "Diagram" tab,
  e.g. `container.NewTabItem("Architecture", a.newArchitectureTab(res))`
- New files: `internal/gui/architecture.go` (+ `architecture_test.go`)
- Reuse the existing native-canvas machinery instead of writing a new engine:
  - `DiagramNode` / `DiagramEdge` / `DiagramLayout` and `ComputeDiagramLayout`
    (`internal/gui/diagramlayout.go`) — extend nodes with a shape kind +
    explanation fields rather than forking the types
  - `DiagramNodeWidget` with its `Tapped` handler (`internal/gui/diagramview.go`)
    — already a clickable canvas widget; MVP changes tap behavior from
    drill-down to select-and-explain
  - Existing zoom/legend/breadcrumb patterns from `newDiagramTab`
- The existing "Diagram" tab stays as-is; Architecture Explainer is a separate
  section focused on *explaining*, not navigating

## 7. Implementation steps

1. [x] Define `ShapeKind` (rect/square/circle/oval/cylinder/cloud) and an
       `Explanation` struct (what it is, what it does, connections + kinds)
       — `internal/gui/architecture.go` (`ArchNode`/`ArchEdge` carry the
       explanation fields directly)
2. [x] Write a small builder: `cli.PipelineResult` → nodes/edges with shape
       kinds and grounded explanation text (pure function, easily testable)
       — `BuildArchDiagram`
3. [x] Layout: keep it layered (entry points top → modules middle →
       data/outputs bottom) — `layoutArch`, reusing the Diagram tab's
       spacing constants
4. [x] Render shapes on a Fyne canvas (`canvas.Rectangle`, `canvas.Circle`,
       `canvas.Line` for arrows; cylinders/clouds composed from
       rectangle+ellipse / three ellipses) — `internal/gui/architectureview.go`
5. [x] Selection: highlight tapped shape, fill the explanation side panel
       (tap again or tap empty canvas to deselect)
6. [x] Add the tab in `report.go` ("Architecture", after "Diagram")
7. [x] Sanity-check by scanning CodeLens itself — headless self-scan produces
       16 shapes/15 arrows describing cmd/ → internal/ → index → reports;
       eyeball pass with `make run-gui` still worth doing on a desktop

## 8. Testing notes

- Unit-test the builder (step 2) and layout with fixtures from `testdata/`
  (e.g. `sample_go`, `sample_python`) — assert shape kinds, edges, and that
  explanation text references real paths only
- Follow the existing pattern: layout logic is tested headlessly in
  `diagramlayout_test.go`; do the same, keep GUI-only code thin
- `make lint && make test` must pass; GUI build via `make build-gui` (needs CGO)
- Manual pass: scan a sample project and CodeLens itself; click every shape

## 9. Non-goals

- **No HTML, no webview, no browser, no localhost** — native Fyne only
- No web dashboard
- No complex graph engine, no force-directed/auto-routing layout — simple
  layered layout with straight arrows is fine
- No perfect architecture detection in v1 — grounded and readable beats complete
- Don't overwrite this file — check off, append, date your notes

---

*Progress log (append below, newest first):*

- 2026-07-08 — MVP implemented: `internal/gui/architecture.go` (model +
  builder + layout, pure logic), `internal/gui/architectureview.go` (native
  canvas shapes, arrows with heads, selection + explanation panel), new
  "Architecture" tab in `report.go`. Tests in `architecture_test.go` cover
  builder grounding, layered layout, connections, LLM enrichment, and a
  real-pipeline scan of `testdata/sample_go`. Prerequisite fix: bumped Go to
  1.25 (pinned x/sys and modernc.org/sqlite already required it, so the
  module didn't build at 1.23). Remaining niceties: shape drill-down,
  SVG export of this view, richer per-file circles.
- 2026-07-08 — Document created. No implementation yet.
