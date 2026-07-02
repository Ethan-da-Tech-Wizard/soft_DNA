package gui

import (
	"path/filepath"

	"codelens-local/internal/cli"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// App is the CodeLens Local desktop application: one native window, no
// browser, no localhost, no server the user ever has to think about.
// Any hybrid-LLM subprocess it launches is spawned and owned internally by
// internal/cli's pipeline, exactly as it is for the command-line tool.
type App struct {
	fyneApp  fyne.App
	win      fyne.Window
	settings Settings

	pathEntry *widget.Entry
	scanBtn   *widget.Button
	logView   *widget.Label
	logScroll *container.Scroll
	results   *fyne.Container // swapped between a placeholder and the report tabs

	result     cli.PipelineResult
	haveResult bool

	scanProgress *widget.ProgressBarInfinite // non-nil only while a scan is running
}

// NewApp builds the application and its main window, loading any previously
// saved settings from the local config directory.
func NewApp() *App {
	a := &App{
		fyneApp:  fyneapp.NewWithID("com.codelens.local"),
		settings: LoadSettings(),
	}
	a.fyneApp.SetIcon(appIcon)
	a.win = a.fyneApp.NewWindow("CodeLens Local")
	a.win.SetIcon(appIcon)
	a.win.Resize(fyne.NewSize(1100, 750))
	a.win.SetContent(a.buildContent())
	a.pathEntry.SetText(a.settings.LastProject)
	return a
}

// ShowAndRun displays the window and blocks until it is closed.
func (a *App) ShowAndRun() {
	a.win.ShowAndRun()
}

func (a *App) buildContent() fyne.CanvasObject {
	a.pathEntry = widget.NewEntry()
	a.pathEntry.SetPlaceHolder("Choose a project folder to scan…")
	a.pathEntry.OnSubmitted = func(string) { a.startScan() }

	browseBtn := widget.NewButtonWithIcon("Browse…", nil, func() {
		dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil || u == nil {
				return
			}
			a.pathEntry.SetText(u.Path())
		}, a.win)
	})

	a.scanBtn = widget.NewButton("Scan", a.startScan)

	settingsBtn := widget.NewButton("Settings", func() {
		a.showSettingsDialog()
	})

	top := container.NewBorder(nil, nil,
		widget.NewLabel("Project:"), container.NewHBox(browseBtn, a.scanBtn, settingsBtn),
		a.pathEntry,
	)

	// A widget.Entry manages its own internal scrolling, which can't be
	// driven from outside the widget — so a plain Label wrapped in a Scroll
	// this code fully owns is what makes appendLog's ScrollToBottom work.
	a.logView = widget.NewLabel("")
	a.logView.Wrapping = fyne.TextWrapWord
	a.logScroll = container.NewVScroll(a.logView)
	a.logScroll.SetMinSize(fyne.NewSize(0, 160))

	placeholder := container.NewCenter(widget.NewLabel(
		"Choose a project folder and click Scan.\nAnalysis runs fully offline — nothing leaves this machine.",
	))
	a.results = container.NewStack(placeholder)

	center := container.NewBorder(nil, container.NewVBox(widget.NewLabel("Log"), a.logScroll), nil, nil, a.results)

	return container.NewBorder(top, nil, nil, nil, center)
}

func (a *App) appendLog(line string) {
	if a.logView.Text != "" {
		a.logView.SetText(a.logView.Text + "\n" + line)
	} else {
		a.logView.SetText(line)
	}
	a.logScroll.ScrollToBottom()
}

func (a *App) startScan() {
	root := a.pathEntry.Text
	if root == "" {
		dialog.ShowInformation("No folder selected", "Choose a project folder first.", a.win)
		return
	}

	a.scanBtn.Disable()
	a.logView.SetText("")
	a.scanProgress = widget.NewProgressBarInfinite()
	a.results.Objects = []fyne.CanvasObject{container.NewCenter(container.NewVBox(
		widget.NewLabel("Scanning…"), a.scanProgress,
	))}
	a.results.Refresh()

	if root != a.settings.LastProject {
		a.settings.LastProject = root
		if err := a.settings.Save(); err != nil {
			a.appendLog("warning: failed to remember this project path: " + err.Error())
		}
	}

	go a.runScan(root)
}

func (a *App) runScan(root string) {
	diagOut := &lineWriter{onLine: func(line string) {
		fyne.Do(func() { a.appendLog(line) })
	}}

	opts := cli.PipelineOptions{
		Root:        root,
		OutDir:      filepath.Join(root, "codelens-out"),
		LLMEnabled:  a.settings.LLMEnabled,
		LLMBinary:   a.settings.LLMBinary,
		LLMModel:    a.settings.LLMModel,
		LLMThreads:  a.settings.LLMThreads,
		LLMMaxFiles: a.settings.LLMMaxFiles,
	}

	res, err := cli.RunPipeline(opts, diagOut)

	fyne.Do(func() {
		a.scanBtn.Enable()
		if a.scanProgress != nil {
			a.scanProgress.Stop()
			a.scanProgress = nil
		}
		if err != nil {
			a.results.Objects = []fyne.CanvasObject{container.NewCenter(widget.NewLabel("Scan failed. See error dialog."))}
			a.results.Refresh()
			dialog.ShowError(err, a.win)
			return
		}
		for _, w := range res.Warnings {
			a.appendLog("warning: " + w)
		}
		if werr := cli.WriteMarkdownReport(res); werr != nil {
			a.appendLog("warning: failed to write CODEBASE_REPORT.md: " + werr.Error())
		}

		a.result = res
		a.haveResult = true
		a.results.Objects = []fyne.CanvasObject{a.buildReportTabs(res)}
		a.results.Refresh()
	})
}
