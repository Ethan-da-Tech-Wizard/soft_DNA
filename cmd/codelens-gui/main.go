// codelens-gui is the native desktop counterpart to codelens: a single
// window, no browser, no localhost, fully offline. It shares 100% of the
// scanning/indexing/LLM pipeline in internal/cli with the CLI — this binary
// only adds a Fyne front end on top of it.
package main

import "codelens-local/internal/gui"

func main() {
	gui.NewApp().ShowAndRun()
}
