// Program: soft_DNA - main.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Entry point for the desktop GUI binary; it builds the Fyne app from internal/gui and runs its window loop.

// codelens-gui is the native desktop counterpart to codelens: a single
// window, no browser, no localhost, fully offline. It shares 100% of the
// scanning/indexing/LLM pipeline in internal/cli with the CLI — this binary
// only adds a Fyne front end on top of it.
package main

import "codelens-local/internal/gui"

// main creates the GUI application and blocks until its window is closed.
func main() {
	gui.NewApp().ShowAndRun()
}
