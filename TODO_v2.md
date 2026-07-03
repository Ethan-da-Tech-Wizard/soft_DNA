# TODO — CodeLens Local v2.0 "Working AI & A Diagram Worth Clicking"

Agent-ready task list, written after running the desktop app against a
real project (a multi-folder Python repo) instead of the small test
fixtures. Three things failed that fixtures never exposed:

1. **The AI pass produces nothing** — a model was selected, a scan ran,
   and the AI Summaries tab stayed empty with no visible explanation.
2. **The architecture diagram is unreadable and not interactive** — a
   single endless row of boxes (including `.mypy_cache`, `.pytest_cache`,
   `.ruff_cache` as "folders"), the Project Root node off-screen, every
   role labeled "Project files", edges a flat tangle, nothing clickable.
3. **There is no built-in model story** — the user is expected to procure
   a `llama-cli` binary and wire up two file paths before anything
   AI-related works at all.

Each task below is sized to one commit/PR. **Keep it simple, stupid**:
every task has a "Do NOT" list — the fix for a weak feature is a narrow
strong feature, not a framework.

## Global rules (apply to EVERY task — read first)

- Guardrails are non-negotiable: never execute target code, never make a
  network call during analysis, only the llama.cpp subprocess may be
  spawned. Diagnostics to stderr/log panel only.
- The CLI and GUI share one pipeline (`internal/cli`'s exported
  `RunPipeline`/`Answer`). Fix behavior there once; never fork logic into
  the GUI.
- Determinism: same input ⇒ same output. Sort map-derived data.
- Verification for every task: `go test ./...` green, `gofmt -l .` empty,
  `go vet ./...` clean, `make smoke` passes, and for GUI-visible changes a
  real run on a virtual display (Xvfb) — a screenshot of the feature
  actually working, not just compiling.
- Reference docs in-repo: `CODEBASE_GUIDE.md`, `KISS_LOOP_EQUATION.md`,
  `POLISH_CHECKLIST.md` (all 11 items from the first polish pass are done).

---

# P0 — Make the AI actually produce output

## Diagnosis (verified in code, do not re-derive)

- **D1 — The 60-second timeout kills every 7B call.** The GUI never sets
  `PipelineOptions.LLMTimeout`, so `RunPipeline` applies its 60s default
  to every llama-cli invocation (`internal/cli/cli.go` ~line 258). A
  Qwen2.5 7B Q4_K_M on CPU generates ~3–8 tokens/sec; with a 512-token
  cap plus prompt processing, one call takes 1–4 minutes. Result: every
  call times out, the BRSL loop logs `Health = 0` turns, the stall guard
  force-advances past everything, and Section 11 / the AI tab is empty.
  The only evidence is warning lines in the log panel that scroll past.
- **D2 — Two paths required, half-configured setups fail soft.** The LLM
  pass needs BOTH a llama-cli binary and a .gguf model. If either is
  missing the pipeline soft-skips with one warning. The Settings dialog
  warns on save (added in the polish pass), but the AI Summaries tab
  itself never says why it's empty.
- **D3 — No timeout or GPU control exists in the GUI at all.**

## T1 — Raise and expose the inference timeout ✅ DONE

**Files:** `internal/gui/settings.go`, `internal/gui/settingsdialog.go`,
`internal/gui/app.go`, tests alongside.

**Do:**
1. Add `LLMTimeoutSec int` to `Settings` (JSON-persisted like the rest).
2. Default it by model size at scan time when unset: stat the model file
   once — under 3 GB ⇒ 120s, 3 GB or over ⇒ 420s. One `if`, not a table.
3. Add a "Timeout per call (seconds)" field to the Settings dialog.
4. Pass it into `PipelineOptions.LLMTimeout` in `runScan` and the Ask tab.

**Do NOT:** add adaptive/retry-with-longer-timeout logic; the BRSL loop
already owns retry policy.

**Accept when:** a mock llama-cli that sleeps 90s succeeds with the new
default where it times out today; settings round-trip test covers the
field.

## T2 — The AI tab must explain itself when empty ✅ DONE

**Files:** `internal/gui/report.go`, `internal/gui/app.go`.

**Do:** when the LLM pass was enabled but produced nothing, the AI
Summaries tab shows the actual reason, not a shrug: the collected
`res.Warnings` plus a one-line count of timed-out/stalled turns pulled
from the loop (e.g. "All 11 model calls exceeded the 60s timeout —
raise it in Settings"), and a button that opens Settings directly.
Keep the happy path unchanged.

**Do NOT:** build a diagnostics screen; it's one label + one button on
an existing empty-state.

**Accept when:** enabling LLM with a dead binary path yields a tab that
names the problem; the always-timeout mock yields the timeout message.

## T3 — "Test model" button in Settings ✅ DONE

**Files:** `internal/gui/settingsdialog.go`, small exported helper in
`internal/cli` (reuse `llmRun` seam), tests.

**Do:** one button that runs a single tiny inference (8-token cap, the
configured binary/model/threads/timeout) and reports either
"OK — 4.2 tokens/sec" or the exact error (binary not found / model not
found / timed out after Ns / bad output). This single button eliminates
the entire "selected a model, nothing happens, no idea why" class before
a 10-minute scan is wasted.

**Do NOT:** benchmark suites, warm-up passes, model metadata parsing.

**Accept when:** wired to the mock binary it reports success with a rate;
pointed at a missing file it reports that specific failure, in-dialog.

## T4 — Built-in inference: bundle llama-cli, auto-detect models ✅ DONE

**Files:** `scripts/`, `Makefile`, `internal/gui/settings.go`,
`internal/gui/settingsdialog.go`, README.

The honest constraint first: **the 7B model itself cannot ship inside the
binary or the repo.** Qwen2.5-Coder-7B Q4_K_M is ~4.7 GB; GitHub caps
release assets at 2 GB and stuffing models into git history is a
non-starter. "Built-in" therefore means: *the engine is bundled, the
model is auto-discovered or fetched once, and the user never types a
path.*

**Do:**
1. `make release-gui` packages the platform's llama-cli binary alongside
   codelens-gui in the release archive (the download script already knows
   how to fetch per-OS builds at packaging time).
2. On startup, if `Settings.LLMBinary` is empty, auto-detect in order:
   next to the executable → `~/.codelens/bin/llama-cli` → `$PATH`.
   Fill the setting silently when found.
3. Same for models: if `Settings.LLMModel` is empty, scan
   `~/.codelens/models/*.gguf` and `~/Downloads/*.gguf` (newest first) and
   offer what's found in a dropdown — this picks up the user's
   already-downloaded Qwen 7B with zero typing.
4. First-run "Set up AI" card on the AI tab when nothing is configured:
   [Use detected model ▾] [Download recommended model (~900 MB)] — the
   download button just runs the existing `scripts/download_model.sh`
   flow with a progress bar, only on explicit click (offline-first stays
   the default; analysis never touches the network).

**Do NOT:** embed GGUF bytes in the binary, add a model marketplace,
auto-download anything without an explicit click.

**Accept when:** on a machine with a GGUF in Downloads and llama-cli next
to the binary, a fresh install reaches a working AI scan with zero path
fields typed.

## T5 — Per-task model guidance (small now, big later) ✅ DONE

**Do (now):** when the selected model file is ≥4 GB, show one Settings
hint: "Large model: great for Ask/overview, slow for per-file
annotations (~1–4 min each on CPU). Consider a 0.5–1.5B model for
annotations." Also surface llama-cli's GPU offload as a single optional
"GPU layers" settings field passed through as `-ngl` (0 = CPU, default).

**Do NOT (yet):** two-model routing (small model for file notes, big for
Ask). That's P3 brainstorm — design it only after T1–T4 prove out.

---

# P1 — Architecture Diagram v2: readable, then interactive

## Diagnosis

- Cache/noise directories (`.mypy_cache`, `.pytest_cache`, `.ruff_cache`,
  `.tox`, `.claude`) are not in the scanner's ignore list, so they pollute
  the diagram, the folder table, the index, and stack detection.
- `ComputeDiagramLayout` lays all level-1 nodes in ONE row: 12 folders ⇒
  ~2,450px wide, Project Root centered above ends up off-screen, and the
  root→node edges become a flat unreadable fan.
- `folderRole` only knows ~10 conventional names; everything else is
  "Project files".
- Nothing is clickable. Depth stops at top-level folders.

## T6 — Stop scanning noise directories (quick win, fixes 4 features) ✅ DONE

**Files:** `internal/scanner/scanner.go` (`defaultIgnoredDirs`), test.

**Do:** add `.mypy_cache`, `.pytest_cache`, `.ruff_cache`, `.tox`,
`.eggs`, `.claude`, `.gradle`, `.terraform`, `.svn`, `.hg`.

**Do NOT:** ignore `.github` (it's meaningful CI signal — give it a role
in T9 instead) or make the list user-configurable (the `--ignore` flag
already exists for that).

**Accept when:** a fixture with `.mypy_cache/junk.py` shows no trace of
it in inventory, folders, diagram, or index.

## T7 — Wrapping grid layout with the root always visible ✅ DONE

**Files:** `internal/gui/diagramlayout.go` + tests (pure math — this is
why layout was separated from rendering).

**Do:** wrap level-1 nodes into rows of at most N columns (pick N so a
row fits ~1100px at scale 1.0, i.e. 5); root pinned top-center of the
*visible* grid width; tech row below the last folder row. Edges go
root-bottom → node-top as short segments, not a 2,000px fan.

**Do NOT:** implement force-directed/graphviz layout. A wrapped grid is
the KISS answer and its layout is unit-testable to the pixel.

**Accept when:** layout tests assert 12 folders produce 3 rows of ≤5,
root X within the first row's span, and total width ≤ N·(nodeW+gap).

## T8 — Clickable nodes: drill down one level at a time ✅ DONE

**Files:** `internal/gui/diagramview.go`, new tiny tappable-node widget,
`internal/gui/diagramlayout.go` for the sub-level layout, tests.

**Do:** clicking a folder node re-renders the diagram scoped to that
folder: its immediate subfolders and files (from the already-scanned
inventory — no new I/O), with a breadcrumb ("Project Root › app › core")
and a Back button. Files are leaf nodes; clicking a file does nothing in
this task. Implement the node as a minimal custom widget embedding the
existing rectangle+label and `Tapped()`.

**Do NOT:** hover tooltips (tap targets suffice), animation, multi-select,
or unlimited-depth trees rendered at once — one level per view is the
whole point.

**Accept when:** Xvfb run shows: click `app` → subfolder/file boxes +
breadcrumb; Back returns; empty folders render a placeholder not a crash.

## T9 — Roles worth reading + color coding ✅ DONE

**Files:** `internal/structure/structure.go` (`roleFor`),
`internal/grounding/grounding.go` (`folderRole` — keep the two in sync,
or better: have grounding call structure's), `internal/gui/diagramview.go`.

**Do:** extend role mapping (`.github` ⇒ "CI/CD configuration", `api`,
`models`, `data`, `assets`, `migrations`, `configs`, `web`, `frontend`,
`backend`, `services`, `utils`/`helpers`, `notebooks`…). Tint node fills
by role category (4–5 colors from the existing theme palette) with a
one-line legend under the zoom bar.

**Do NOT:** invent per-language ontologies or user-editable role maps.

**Accept when:** the real-project diagram shows ≤30% "Project files"
where today it shows ~90%, and colors match the legend.

## T10 — Fit-to-window button ✅ DONE

**Files:** `internal/gui/diagramview.go`.

**Do:** a "Fit" button beside Zoom: scale = min(viewportW/layoutW,
viewportH/layoutH, 1.0). One formula.

**Accept when:** a 3-row diagram fits entirely in the tab on click.

---

# P2 — GUI performance on real projects

## T11 — Fix the O(n²) log append ✅ DONE

`appendLog` does `SetText(old + "\n" + line)` — every line copies the
whole log and relayouts the label; a 500-line LLM scan does ~125k line
copies. Keep the last 500 lines in a slice (ring), join once per append,
and note the truncation in-line ("… earlier lines omitted"). File:
`internal/gui/app.go`. Accept: a synthetic 5,000-line append stays
under a second and holds only the tail.

## T12 — Parallel symbol extraction ✅ DONE

`symbols.ExtractAll` walks files serially; it's pure CPU and the output
is sorted afterward anyway, so a worker pool (NumCPU) is safe and
deterministic. File: `internal/symbols/symbols.go`. Accept: existing
tests unchanged (they assert sorted output), plus a benchmark in
`symbols_test.go` showing the speedup; `go test -race` clean.

## T13 — Build report rows off the UI thread ✅ DONE

`buildReportTabs` runs inside `fyne.Do` — thousands of symbol rows are
formatted on the render thread, freezing the window at scan end. Prepare
all `[][]string` row data in the scan goroutine; only construct widgets
in `fyne.Do`. File: `internal/gui/app.go`/`report.go`. Accept: no
behavior change (existing headless tab tests still pass); window stays
responsive at scan-complete on a large repo.

## T14 — Live scan counters instead of a silent bar ✅ DONE

Extend the pipeline's diagnostics writer usage: log one line per N=200
files scanned and one when indexing starts, so big scans show heartbeat
in the log panel between LLM turns. Files: `internal/cli/cli.go`
(RunPipeline), tests. **Do NOT** add a progress-percent API — lines into
the existing writer, that's it.

---

# P3 — Intense ideas (brainstorm — documented, NOT committed)

Each of these graduates into a numbered task only after the P0–P2 list
is done and it still seems worth it. Listed so the thinking isn't lost:

- **Two-model routing** — a small (0.5–1.5B) model for per-file
  annotations, the big 7B for project overview and Ask. The BRSL loop
  already treats every call independently, so routing is one field on
  `llm.Config` away once Settings can hold two model paths.
- **Streaming Ask** — stream llama-cli stdout into the chat bubble
  token-by-token instead of waiting for the full answer. Needs the
  runner to expose a line/chunk callback; grammar-constrained JSON makes
  this trickier than it looks (you'd stream the `answer` field only).
- **Incremental rescan** — `FileInfo.ContentHash` already exists; skip
  re-extraction/re-indexing for unchanged files and re-annotate only
  changed ones. Turns minute-scale rescans into seconds.
- **Watch mode** — rescan-on-save using fsnotify (already an indirect
  Fyne dependency). GUI-only; the CLI stays one-shot.
- **Diagram export** — render the current diagram view to PNG/SVG next
  to `architecture.mmd`, straight from the layout data (no screenshots).
- **"Explain this folder"** — a button on a drilled-in diagram view that
  runs one grounded LLM call scoped to that folder's symbols/excerpts.
  Reuses the Ask plumbing verbatim; the prompt is the only new code.
- **Call-graph edges** — real function-level architecture needs deeper
  parsing (`gotreesitter` is already on the roadmap's "later" list).
  Do not attempt with regexes.
- **GPU offload presets** — beyond T5's `-ngl` field: detect
  CUDA/Metal llama-cli variants at startup and label the Settings hint
  accordingly.
- **Proper installers** — `fyne package` gives .app/.exe/.desktop with
  the icon baked in; releases gain per-OS GUI archives next to the CLI
  ones (blocked on CGO cross-compile, so build natively per-OS in CI
  with a matrix).
- **Perf regression harness** — `make bench` that scans a pinned large
  OSS repo and prints wall-clock per stage, so P2's wins don't silently
  rot.
- **Ask conversation memory** — previously and deliberately rejected
  ("each question answered independently, grounded fresh"). Leaving it
  rejected; recorded here so the decision isn't re-litigated by accident.

---

## Suggested sequencing

| Order | Task | Size | Why |
|---|---|---|---|
| 1 | T6 noise dirs | XS | One list edit fixes diagram, tables, index, stack detection |
| 2 | T1 timeout | S | THE reason AI "generates nothing" with a 7B model |
| 3 | T2 AI tab explains itself | S | Kills the silent-failure experience |
| 4 | T3 Test-model button | S | Config problems caught in 5 seconds, not after a scan |
| 5 | T7 wrapping layout | M | Diagram becomes readable before it becomes interactive |
| 6 | T11 log append | S | Cheapest big perf win |
| 7 | T4 bundled engine + auto-detect | M | "Built-in AI" without shipping 4.7 GB |
| 8 | T8 drill-down nodes | M | The diagram becomes a navigation tool |
| 9 | T9 roles + colors | S | Readability compounding on T7/T8 |
| 10 | T10 fit button | XS | Trivial after T7 |
| 11 | T12–T14 perf batch | M | Big-repo smoothness |
| 12 | T5 model guidance + `-ngl` | S | Rounds out the 7B story |

One task per commit/PR. Update this file's checkboxes as tasks land. If
a task fights you into a bigger design, stop and report back instead of
expanding scope. Keep it simple, stupid.
