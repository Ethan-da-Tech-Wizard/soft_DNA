// Program: soft_DNA - logwriter_test.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Tests that lineWriter emits whole lines and holds back a partial line until a later write completes it.

package gui

import "testing"

// TestLineWriterSplitsCompleteLines checks that one write containing two full lines produces two callbacks.
func TestLineWriterSplitsCompleteLines(t *testing.T) {
	var got []string
	w := &lineWriter{onLine: func(line string) { got = append(got, line) }}

	w.Write([]byte("first\nsecond\n"))

	want := []string{"first", "second"}
	if len(got) != len(want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestLineWriterBuffersPartialLineAcrossWrites checks that text without a newline is held back and joined with the next write.
func TestLineWriterBuffersPartialLineAcrossWrites(t *testing.T) {
	var got []string
	w := &lineWriter{onLine: func(line string) { got = append(got, line) }}

	w.Write([]byte("partial "))
	if len(got) != 0 {
		t.Fatalf("onLine fired before newline: %v", got)
	}
	w.Write([]byte("line\nnext\n"))

	want := []string{"partial line", "next"}
	if len(got) != len(want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
