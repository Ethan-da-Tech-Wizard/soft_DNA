package gui

import (
	"codelens-local/internal/cli"
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

	modelEntry := widget.NewSelectEntry(ScanLocalModels())
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

	timeoutEntry := widget.NewEntry()
	if a.settings.LLMTimeoutSec > 0 {
		timeoutEntry.SetText(strconv.Itoa(a.settings.LLMTimeoutSec))
	}
	timeoutEntry.SetPlaceHolder("auto (by model size)")

	// Test model: one tiny inference against the paths currently typed in
	// this dialog (not the saved settings), so problems surface in seconds
	// instead of after an hours-long scan.
	testStatus := widget.NewLabel("")
	testStatus.Wrapping = fyne.TextWrapWord
	var testBtn *widget.Button
	testBtn = widget.NewButton("Test model", func() {
		threads, err := strconv.Atoi(threadsEntry.Text)
		if err != nil || threads <= 0 {
			threads = a.settings.LLMThreads
		}
		trial := Settings{LLMModel: modelEntry.Text}
		if v, err := strconv.Atoi(strings.TrimSpace(timeoutEntry.Text)); err == nil && v > 0 {
			trial.LLMTimeoutSec = v
		}
		binary, model, timeout := binaryEntry.Text, modelEntry.Text, trial.EffectiveLLMTimeout()

		testBtn.Disable()
		testStatus.SetText("Testing… (a large model can take a minute or two to load)")
		go func() {
			msg, err := cli.TestModel(binary, model, threads, timeout)
			fyne.Do(func() {
				testBtn.Enable()
				if err != nil {
					testStatus.SetText("Failed: " + err.Error())
					return
				}
				testStatus.SetText(msg)
			})
		}()
	})

	form := widget.NewForm(
		widget.NewFormItem("", enabled),
		widget.NewFormItem("llama-cli binary", container.NewBorder(nil, nil, nil, binaryBrowse, binaryEntry)),
		widget.NewFormItem("GGUF model file", container.NewBorder(nil, nil, nil, modelBrowse, modelEntry)),
		widget.NewFormItem("CPU threads", threadsEntry),
		widget.NewFormItem("Max files to annotate", maxFilesEntry),
		widget.NewFormItem("Timeout per call (sec)", timeoutEntry),
		widget.NewFormItem("", container.NewVBox(container.NewHBox(testBtn), testStatus)),
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
		timeoutSec := 0 // empty field = automatic, scaled by model size
		if strings.TrimSpace(timeoutEntry.Text) != "" {
			if v, err := strconv.Atoi(timeoutEntry.Text); err == nil && v > 0 {
				timeoutSec = v
			} else {
				timeoutSec = a.settings.LLMTimeoutSec
			}
		}
		a.settings = Settings{
			LLMEnabled:    enabled.Checked,
			LLMBinary:     binaryEntry.Text,
			LLMModel:      modelEntry.Text,
			LLMThreads:    threads,
			LLMMaxFiles:   maxFiles,
			LLMTimeoutSec: timeoutSec,
			LastProject:   a.settings.LastProject,
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
	d.Resize(fyne.NewSize(520, 340))
	d.Show()
}
