package gui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showSettingsDialog lets the user toggle the local LLM pass and point it at
// a llama-cli binary and a .gguf model file, both loaded from local disk.
// Nothing here ever performs a network call. Saved settings persist to the
// OS user config directory so they survive between runs.
func (a *App) showSettingsDialog() {
	enabled := widget.NewCheck("Enable local LLM summaries and grounded Ask", nil)
	enabled.SetChecked(a.settings.LLMEnabled)

	binaryEntry := widget.NewEntry()
	binaryEntry.SetText(a.settings.LLMBinary)
	binaryBrowse := widget.NewButton("Browse…", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			defer r.Close()
			binaryEntry.SetText(r.URI().Path())
		}, a.win)
	})

	modelEntry := widget.NewEntry()
	modelEntry.SetText(a.settings.LLMModel)
	modelBrowse := widget.NewButton("Browse…", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			defer r.Close()
			modelEntry.SetText(r.URI().Path())
		}, a.win)
	})

	threadsEntry := widget.NewEntry()
	threadsEntry.SetText(strconv.Itoa(a.settings.LLMThreads))

	maxFilesEntry := widget.NewEntry()
	maxFilesEntry.SetText(strconv.Itoa(a.settings.LLMMaxFiles))
	maxFilesEntry.SetPlaceHolder("0 = no limit")

	form := widget.NewForm(
		widget.NewFormItem("", enabled),
		widget.NewFormItem("llama-cli binary", container.NewBorder(nil, nil, nil, binaryBrowse, binaryEntry)),
		widget.NewFormItem("GGUF model file", container.NewBorder(nil, nil, nil, modelBrowse, modelEntry)),
		widget.NewFormItem("CPU threads", threadsEntry),
		widget.NewFormItem("Max files to annotate", maxFilesEntry),
	)

	d := dialog.NewCustomConfirm("Settings", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		threads, err := strconv.Atoi(threadsEntry.Text)
		if err != nil || threads <= 0 {
			threads = a.settings.LLMThreads
		}
		maxFiles, err := strconv.Atoi(maxFilesEntry.Text)
		if err != nil || maxFiles < 0 {
			maxFiles = a.settings.LLMMaxFiles
		}
		a.settings = Settings{
			LLMEnabled:  enabled.Checked,
			LLMBinary:   binaryEntry.Text,
			LLMModel:    modelEntry.Text,
			LLMThreads:  threads,
			LLMMaxFiles: maxFiles,
			LastProject: a.settings.LastProject,
		}
		if err := a.settings.Save(); err != nil {
			dialog.ShowError(err, a.win)
		}
		if problems := validateLLMConfig(a.settings); len(problems) > 0 {
			dialog.ShowInformation("LLM settings need attention",
				"Settings were saved, but the local LLM pass will be skipped until this is fixed:\n\n"+strings.Join(problems, "\n"),
				a.win)
		}
	}, a.win)
	d.Resize(fyne.NewSize(520, 300))
	d.Show()
}
