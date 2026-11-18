// Program: soft_DNA - logwriter.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Provides lineWriter, an io.Writer adapter that turns a byte stream into one callback per completed line for the GUI log.

package gui

import "bytes"

// lineWriter is an io.Writer adapter that invokes onLine once per completed
// line written to it, buffering any trailing partial line until the next
// Write completes it. RunPipeline's diagnostics writer calls Write with
// whole formatted lines, but lineWriter makes no assumption about that.
type lineWriter struct {
	onLine func(string)
	buf    bytes.Buffer
}

// Write buffers p and calls onLine for every complete line; it always reports the full length as written.
func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Incomplete line: ReadString already drained it from buf, so
			// put it back and wait for the rest on the next Write.
			w.buf.Reset()
			w.buf.WriteString(line)
			return len(p), nil
		}
		w.onLine(line[:len(line)-1])
	}
}
