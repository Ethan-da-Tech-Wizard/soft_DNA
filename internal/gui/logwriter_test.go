package gui

import "testing"

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
