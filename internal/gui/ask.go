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
	history := container.NewVBox()
	historyScroll := container.NewVScroll(history)

	question := widget.NewEntry()
	question.SetPlaceHolder("Ask a question about this codebase…")

	addBubble := func(who, text string) {
		title := widget.NewLabel(who)
		title.TextStyle = fyne.TextStyle{Bold: true}
		body := widget.NewLabel(text)
		body.Wrapping = fyne.TextWrapWord
		history.Add(container.NewVBox(title, body, widget.NewSeparator()))
		historyScroll.ScrollToBottom()
	}

	send := widget.NewButton("Send", nil)
	ask := func() {
		q := strings.TrimSpace(question.Text)
		if q == "" {
			return
		}
		question.SetText("")
		send.Disable()
		addBubble("You", q)

		go func() {
			ans, err := cli.Answer(cli.AnswerOptions{
				Root:       res.Root,
				OutDir:     res.OutDir,
				Question:   q,
				Entries:    res.Entries,
				LLMEnabled: a.settings.LLMEnabled,
				LLMBinary:  a.settings.LLMBinary,
				LLMModel:   a.settings.LLMModel,
				LLMThreads: a.settings.LLMThreads,
			})
			fyne.Do(func() {
				send.Enable()
				if err != nil {
					addBubble("CodeLens", "Error: "+err.Error())
					return
				}
				addBubble("CodeLens", formatAnswer(ans))
			})
		}()
	}
	send.OnTapped = ask
	question.OnSubmitted = func(string) { ask() }

	inputRow := container.NewBorder(nil, nil, nil, send, question)
	return container.NewBorder(nil, inputRow, nil, nil, historyScroll)
}

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
