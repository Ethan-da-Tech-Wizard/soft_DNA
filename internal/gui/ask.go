// Program: soft_DNA - ask.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Builds the chat-style Ask tab of the GUI and formats cited answers for display.

package gui

import (
	"fmt"
	"strings"

	"codelens-local/internal/cli"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newAskTab builds a chat-style question box over res's index: type a
// question, get a cited answer (grounded by a local LLM if one is
// configured, or plain FTS citations otherwise) appended below as a bubble.
// Every answer is also written to ANSWER.md, matching --ask on the CLI.
func (a *App) newAskTab(res cli.PipelineResult) fyne.CanvasObject {
	// history is the scrolling list of question and answer bubbles.
	history := container.NewVBox()
	historyScroll := container.NewVScroll(history)

	question := widget.NewEntry()
	question.SetPlaceHolder("Ask a question about this codebase…")

	// addBubble appends a bold speaker title and wrapped message text to the history.
	addBubble := func(who, text string) {
		title := widget.NewLabel(who)
		title.TextStyle = fyne.TextStyle{Bold: true}
		body := widget.NewLabel(text)
		body.Wrapping = fyne.TextWrapWord
		history.Add(container.NewVBox(title, body, widget.NewSeparator()))
		historyScroll.ScrollToBottom()
	}

	// The Send button is created first so the ask closure can disable it while a question is running.
	send := widget.NewButton("Send", nil)
	ask := func() {
		if send.Disabled() {
			return // a previous question is still in flight
		}
		q := strings.TrimSpace(question.Text)
		if q == "" {
			return
		}
		question.SetText("")
		send.Disable()
		addBubble("You", q)
		clTitle := widget.NewLabel("CodeLens")
		clTitle.TextStyle = fyne.TextStyle{Bold: true}
		clBody := widget.NewLabel("Thinking…")
		clBody.Wrapping = fyne.TextWrapWord
		history.Add(container.NewVBox(clTitle, clBody, widget.NewSeparator()))
		historyScroll.ScrollToBottom()

		// Run the (possibly slow) answer lookup off the UI thread.
		go func() {
			gotTokens := false
			ans, err := cli.Answer(cli.AnswerOptions{
				Root:         res.Root,
				OutDir:       res.OutDir,
				Question:     q,
				Entries:      res.Entries,
				LLMEnabled:   a.settings.LLMEnabled,
				LLMBinary:    a.settings.LLMBinary,
				LLMModel:     a.settings.LLMModel,
				LLMThreads:   a.settings.LLMThreads,
				LLMTimeout:   a.settings.EffectiveLLMTimeout(),
				LLMGPULayers: a.settings.LLMGPULayers,
				// Stream tokens into the answer bubble as the local model produces them; fyne.Do marshals the update to the UI thread.
				TokenCallback: func(token string) {
					fyne.Do(func() {
						if !gotTokens {
							clBody.SetText("")
							gotTokens = true
						}
						clBody.SetText(clBody.Text + token)
						historyScroll.ScrollToBottom()
					})
				},
			})
			fyne.Do(func() {
				send.Enable()
				if err != nil {
					if !gotTokens {
						clBody.SetText("Error: " + err.Error())
					} else {
						clBody.SetText(clBody.Text + "\n\nError: " + err.Error())
					}
					return
				}
				clBody.SetText(formatAnswer(ans))
				historyScroll.ScrollToBottom()
			})
		}()
	}
	send.OnTapped = ask
	question.OnSubmitted = func(string) { ask() }

	inputRow := container.NewBorder(nil, nil, nil, send, question)
	return container.NewBorder(nil, inputRow, nil, nil, historyScroll)
}

// formatAnswer renders an AnswerResult as plain text: an optional warning, the model answer or a fallback sentence, then the list of source files.
func formatAnswer(ans cli.AnswerResult) string {
	var b strings.Builder
	if ans.LLMWarning != "" {
		fmt.Fprintf(&b, "(%s)\n\n", ans.LLMWarning)
	}
	switch {
	case ans.LLMAnswer != "":
		b.WriteString(ans.LLMAnswer)
	case len(ans.Hits) == 0:
		b.WriteString("No indexed matches were found for this question.")
	default:
		b.WriteString("The most relevant indexed files are:")
	}
	if len(ans.Hits) > 0 {
		b.WriteString("\n\nSources:")
		for _, h := range ans.Hits {
			fmt.Fprintf(&b, "\n  • %s", h.Path)
		}
	}
	return b.String()
}
