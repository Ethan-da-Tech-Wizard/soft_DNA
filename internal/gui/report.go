package gui

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"codelens-local/internal/cli"
	"codelens-local/internal/symbols"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// buildReportTabs renders a completed pipeline run as a native tabbed view —
// the interactive, point-and-click equivalent of CODEBASE_REPORT.md, built
// directly from structured data rather than by parsing Markdown.
func (a *App) buildReportTabs(res cli.PipelineResult) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Summary", a.newSummaryTab(res)),
		container.NewTabItem("Tech Stack", newTechStackTab(res)),
		container.NewTabItem("Folders", newFoldersTab(res)),
		container.NewTabItem("Entry Points", newEntryPointsTab(res)),
		container.NewTabItem("Functions & Classes", newSymbolsTab(res.Symbols, "function", "method", "class", "type", "table", "view", "procedure")),
		container.NewTabItem("Imports", newSymbolsTab(res.Symbols, "import")),
		container.NewTabItem("AI Summaries", a.newAISummariesTab(res)),
		container.NewTabItem("Diagram", newDiagramTab(res.Stack, res.Folders, res.Entries)),
		container.NewTabItem("Ask", a.newAskTab(res)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (a *App) newSummaryTab(res cli.PipelineResult) fyne.CanvasObject {
	text := fmt.Sprintf(
		"Scanned %d files.\nDetected stack: %s.\n\nIndex: %s\n  %d files, %d symbols, %d search rows.\n\nMarkdown report and diagram were also written to:\n  %s",
		len(res.Files), res.Stack.Summary,
		res.Index.Path, res.Index.Files, res.Index.Symbols, res.Index.SearchRows,
		res.OutDir,
	)
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord

	openBtn := widget.NewButton("Open Output Folder", func() { a.openFolder(res.OutDir) })
	content := container.NewVBox(label, container.NewHBox(openBtn))
	return container.NewVScroll(container.NewPadded(content))
}

// openFolder asks the OS to open outDir in its default file manager — the
// same "reveal in Finder/Explorer" action any desktop app offers, not a
// network or subprocess operation.
func (a *App) openFolder(outDir string) {
	u, err := url.Parse(storage.NewFileURI(outDir).String())
	if err != nil {
		dialog.ShowError(err, a.win)
		return
	}
	if err := a.fyneApp.OpenURL(u); err != nil {
		dialog.ShowError(err, a.win)
	}
}

func newTechStackTab(res cli.PipelineResult) fyne.CanvasObject {
	rows := make([][]string, 0, len(res.Stack.Tech))
	for _, t := range res.Stack.Tech {
		rows = append(rows, []string{t.Name, t.Category, t.EvidenceFile})
	}
	return newSimpleTable([]string{"Technology", "Category", "Evidence"}, rows)
}

func newFoldersTab(res cli.PipelineResult) fyne.CanvasObject {
	rows := make([][]string, 0, len(res.Folders))
	for _, f := range res.Folders {
		rows = append(rows, []string{f.Path, f.Role, fmt.Sprintf("%d", f.FileCount)})
	}
	return newSimpleTable([]string{"Folder", "Role", "Files"}, rows)
}

func newEntryPointsTab(res cli.PipelineResult) fyne.CanvasObject {
	rows := make([][]string, 0, len(res.Entries))
	for _, e := range res.Entries {
		rows = append(rows, []string{e.Path, e.Reason})
	}
	return newSimpleTable([]string{"Path", "Reason"}, rows)
}

// filterSymbols returns the symbols in all whose name or file path contains
// query (case-insensitive). An empty or whitespace-only query returns all.
func filterSymbols(all []symbols.Symbol, query string) []symbols.Symbol {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return all
	}
	var out []symbols.Symbol
	for _, s := range all {
		if strings.Contains(strings.ToLower(s.Name), query) || strings.Contains(strings.ToLower(s.File), query) {
			out = append(out, s)
		}
	}
	return out
}

// newSymbolsTab lists symbols of the given kinds with a live text filter by
// name or file path.
func newSymbolsTab(symbolList []symbols.Symbol, kinds ...string) fyne.CanvasObject {
	allowed := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		allowed[k] = struct{}{}
	}
	var all []symbols.Symbol
	for _, s := range symbolList {
		if _, ok := allowed[s.Kind]; ok {
			all = append(all, s)
		}
	}

	toRows := func(list []symbols.Symbol) [][]string {
		rows := make([][]string, 0, len(list))
		for _, s := range list {
			rows = append(rows, []string{s.File, s.Kind, s.Name, fmt.Sprintf("%d", s.StartLine), s.Signature})
		}
		return rows
	}

	table, setRows := newLiveTable([]string{"File", "Kind", "Name", "Line", "Signature"}, toRows(all))

	search := widget.NewEntry()
	search.SetPlaceHolder("Filter by name or file…")
	search.OnChanged = func(q string) {
		setRows(toRows(filterSymbols(all, q)))
	}

	if len(all) == 0 {
		return container.NewCenter(widget.NewLabel("No symbols identified."))
	}
	return container.NewBorder(search, nil, nil, nil, table)
}

func (a *App) newAISummariesTab(res cli.PipelineResult) fyne.CanvasObject {
	if a.settings.LLMModel == "" || a.settings.LLMBinary == "" {
		return a.buildSetupAICard()
	}

	openSettings := widget.NewButton("Open Settings", func() { a.showSettingsDialog() })

	if !res.LLM.Enabled {
		var msg string
		if a.settings.LLMEnabled {
			// The user enabled AI but the pass never ran; res.Warnings holds
			// the pipeline's own reason (missing binary/model, hard failure).
			msg = "The AI pass was enabled but did not run:"
			if len(res.Warnings) > 0 {
				msg += "\n\n• " + strings.Join(res.Warnings, "\n• ")
			} else {
				msg += "\n\n(no reason was reported — check the log panel)"
			}
		} else {
			msg = "AI summaries are off.\nEnable the local LLM in Settings and scan again to generate them."
		}
		label := widget.NewLabel(msg)
		label.Wrapping = fyne.TextWrapWord
		return container.NewCenter(container.NewVBox(label, container.NewCenter(openSettings)))
	}

	var items []*widget.AccordionItem
	if res.LLM.ProjectOverview != "" {
		overview := widget.NewLabel(res.LLM.ProjectOverview)
		overview.Wrapping = fyne.TextWrapWord
		items = append(items, widget.NewAccordionItem("Project Overview", overview))
	}

	var files []string
	for f := range res.LLM.FileSummaries {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		fe := res.LLM.FileSummaries[f]
		body := widget.NewLabel(fe.FileSummary)
		body.Wrapping = fyne.TextWrapWord

		content := container.NewVBox(body)
		if len(fe.SymbolExplanations) > 0 {
			rows := make([][]string, 0, len(fe.SymbolExplanations))
			for _, se := range fe.SymbolExplanations {
				rows = append(rows, []string{se.Name, se.Explanation})
			}
			t, _ := newLiveTable([]string{"Symbol", "Explanation"}, rows)
			t.Resize(fyne.NewSize(0, float32(32*(len(rows)+1))))
			content.Add(t)
		}
		items = append(items, widget.NewAccordionItem(f, content))
	}

	if len(res.LLM.Stalled) > 0 {
		stalled := append([]string{}, res.LLM.Stalled...)
		sort.Strings(stalled)
		note := widget.NewLabel("Skipped after repeated validation failures: " + strings.Join(stalled, ", "))
		note.Wrapping = fyne.TextWrapWord
		items = append(items, widget.NewAccordionItem("Skipped Files", note))
	}

	// The pass ran but produced no real content: explain the dominant
	// failure mode using the tallies collected from the diagnostic stream.
	if res.LLM.ProjectOverview == "" && len(res.LLM.FileSummaries) == 0 {
		msg := "The model ran but produced no usable summaries — see the log panel."
		switch {
		case a.scanTimeouts > 0:
			msg = fmt.Sprintf(
				"%d model call(s) timed out (current limit: %s per call).\nRaise \"Timeout per call\" in Settings, or switch to a smaller model.",
				a.scanTimeouts, a.settings.EffectiveLLMTimeout())
		case a.scanFailedTurns > 0:
			msg = fmt.Sprintf(
				"%d model turn(s) failed validation and were skipped — the log panel has each raw output.",
				a.scanFailedTurns)
		}
		label := widget.NewLabel(msg)
		label.Wrapping = fyne.TextWrapWord
		header := container.NewVBox(label, container.NewHBox(openSettings))
		if len(items) == 0 {
			return container.NewCenter(header)
		}
		return container.NewBorder(header, nil, nil, nil, container.NewVScroll(widget.NewAccordion(items...)))
	}

	if len(items) == 0 {
		return container.NewCenter(widget.NewLabel("No AI summaries were generated for this scan."))
	}
	return container.NewVScroll(widget.NewAccordion(items...))
}

func (a *App) buildSetupAICard() fyne.CanvasObject {
	title := widget.NewLabel("Local AI Summaries (Offline)")
	title.TextStyle = fyne.TextStyle{Bold: true}

	desc := widget.NewLabel("Generate plain-English project overviews and per-file annotations fully offline using a local model. Nothing leaves your machine.")
	desc.Wrapping = fyne.TextWrapWord

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	progress := widget.NewProgressBar()
	progress.Hide()

	localModels := ScanLocalModels()
	var modelSelect *widget.Select
	var useModelBtn *widget.Button

	var content []fyne.CanvasObject
	content = append(content, title, desc)

	if len(localModels) > 0 {
		modelSelect = widget.NewSelect(localModels, nil)
		modelSelect.PlaceHolder = "Select detected local model..."
		modelSelect.SetSelected(localModels[0])

		useModelBtn = widget.NewButton("Use Selected Model", func() {
			a.settings.LLMModel = modelSelect.Selected
			a.settings.LLMEnabled = true
			a.settings.LLMBinary = autoDetectLlamaCli()
			if err := a.settings.Save(); err != nil {
				dialog.ShowError(err, a.win)
				return
			}
			dialog.ShowInformation("AI Enabled", "Local model selected. Click Scan again to run the AI analysis.", a.win)
			if a.haveResult {
				a.results.Objects = []fyne.CanvasObject{a.buildReportTabs(a.result)}
				a.results.Refresh()
			}
		})
		content = append(content, widget.NewLabel("Detected GGUF Models:"), modelSelect, container.NewHBox(useModelBtn))
	} else {
		content = append(content, widget.NewLabel("No local GGUF models detected in ~/.codelens/models or ~/Downloads."))
	}

	downloadBtn := widget.NewButton("Download Recommended Model (~900 MB)", func() {
		if useModelBtn != nil {
			useModelBtn.Disable()
		}
		progress.Show()
		progress.SetValue(0)
		status.SetText("Downloading recommended model...")

		a.downloadRecommendedModel(progress, status, func() {
			home, _ := os.UserHomeDir()
			a.settings.LLMModel = filepath.Join(home, ".codelens", "models", "qwen2.5-coder-1.5b-instruct-q4_k_m.gguf")
			a.settings.LLMEnabled = true
			a.settings.LLMBinary = autoDetectLlamaCli()
			a.settings.Save()

			if a.settings.LLMBinary == "" {
				a.ensureLlamaCli(status, func() {
					status.SetText("Setup complete! Model and llama-cli ready. Click Scan again.")
					if useModelBtn != nil {
						useModelBtn.Enable()
					}
					if a.haveResult {
						a.results.Objects = []fyne.CanvasObject{a.buildReportTabs(a.result)}
						a.results.Refresh()
					}
				})
			} else {
				status.SetText("Setup complete! Model and llama-cli ready. Click Scan again.")
				if useModelBtn != nil {
					useModelBtn.Enable()
				}
				if a.haveResult {
					a.results.Objects = []fyne.CanvasObject{a.buildReportTabs(a.result)}
					a.results.Refresh()
				}
			}
		})
	})

	openSettings := widget.NewButton("Open Settings", func() { a.showSettingsDialog() })

	content = append(content, container.NewHBox(downloadBtn, openSettings), progress, status)
	return container.NewCenter(container.NewVBox(content...))
}

func (a *App) downloadRecommendedModel(progress *widget.ProgressBar, status *widget.Label, onDone func()) {
	go func() {
		home, err := os.UserHomeDir()
		if err != nil {
			fyne.Do(func() { status.SetText("Failed: " + err.Error()) })
			return
		}

		targetDir := filepath.Join(home, ".codelens", "models")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			fyne.Do(func() { status.SetText("Failed to create folder: " + err.Error()) })
			return
		}

		targetPath := filepath.Join(targetDir, "qwen2.5-coder-1.5b-instruct-q4_k_m.gguf")
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			fyne.Do(func() { status.SetText("Failed to create file: " + err.Error()) })
			return
		}
		defer out.Close()

		url := "https://huggingface.co/Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF/resolve/main/qwen2.5-coder-1.5b-instruct-q4_k_m.gguf?download=true"
		resp, err := http.Get(url)
		if err != nil {
			fyne.Do(func() { status.SetText("Download failed: " + err.Error()) })
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fyne.Do(func() { status.SetText(fmt.Sprintf("Server returned status %d", resp.StatusCode)) })
			return
		}

		total := resp.ContentLength
		var written int64

		buf := make([]byte, 32*1024)
		for {
			nr, er := resp.Body.Read(buf)
			if nr > 0 {
				nw, ew := out.Write(buf[0:nr])
				if nw > 0 {
					written += int64(nw)
				}
				if ew != nil {
					fyne.Do(func() { status.SetText("Write failed: " + ew.Error()) })
					return
				}
				if nr != nw {
					fyne.Do(func() { status.SetText("Short write error") })
					return
				}
			}
			if er != nil {
				if er == io.EOF {
					break
				}
				fyne.Do(func() { status.SetText("Read error: " + er.Error()) })
				return
			}

			if total > 0 {
				val := float64(written) / float64(total)
				fyne.Do(func() {
					progress.SetValue(val)
					status.SetText(fmt.Sprintf("Downloading... %.1f%% (%s / %s)", val*100, formatBytes(written), formatBytes(total)))
				})
			}
		}

		fyne.Do(func() {
			progress.SetValue(1.0)
			status.SetText("Download complete! Model saved.")
			onDone()
		})
	}()
}

func (a *App) ensureLlamaCli(status *widget.Label, onDone func()) {
	if a.settings.LLMBinary != "" {
		onDone()
		return
	}
	a.settings.LLMBinary = autoDetectLlamaCli()
	if a.settings.LLMBinary != "" {
		onDone()
		return
	}

	go func() {
		home, err := os.UserHomeDir()
		if err != nil {
			fyne.Do(func() { status.SetText("Failed to locate home dir: " + err.Error()) })
			return
		}
		scriptPath := "scripts/download_llama.sh"
		if _, err := os.Stat(scriptPath); err == nil {
			fyne.Do(func() { status.SetText("Downloading llama-cli binary...") })
			cmd := exec.Command("bash", scriptPath)
			if err := cmd.Run(); err != nil {
				fyne.Do(func() { status.SetText("Failed to run download_llama.sh: " + err.Error()) })
				return
			}
			a.settings.LLMBinary = filepath.Join(home, ".codelens", "bin", "llama-cli")
			if runtime.GOOS == "windows" {
				a.settings.LLMBinary += ".exe"
			}
			fyne.Do(onDone)
			return
		}

		fyne.Do(func() {
			status.SetText("llama-cli binary not found. Please install llama-cli or set its path in Settings.")
		})
	}()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
