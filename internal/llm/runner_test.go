package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// ---- limitedWriter unit tests ----

func TestLimitedWriterBelowLimit(t *testing.T) {
	w := &limitedWriter{limit: 100}
	payload := []byte("hello world")
	n, err := w.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Errorf("n = %d, want %d", n, len(payload))
	}
	if got := string(w.Bytes()); got != "hello world" {
		t.Errorf("Bytes() = %q, want %q", got, "hello world")
	}
}

func TestLimitedWriterExceedsLimitInOneWrite(t *testing.T) {
	w := &limitedWriter{limit: 5}
	_, err := w.Write([]byte("hello world")) // 11 bytes > limit 5
	if err == nil {
		t.Fatal("Write: want error when exceeding limit, got nil")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Errorf("err = %q, want to contain 'exceeded'", err.Error())
	}
	if !strings.Contains(err.Error(), "5-byte limit") {
		t.Errorf("err = %q, want to contain the limit value", err.Error())
	}
}

func TestLimitedWriterExactLimitThenOneByte(t *testing.T) {
	// Writing exactly `limit` bytes must succeed; the next byte must fail.
	w := &limitedWriter{limit: 5}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("Write at exact limit: %v", err)
	}
	if _, err := w.Write([]byte("!")); err == nil {
		t.Fatal("Write: want error for byte beyond limit, got nil")
	}
}

func TestLimitedWriterAccumulatesAcrossMultipleWrites(t *testing.T) {
	w := &limitedWriter{limit: 11}
	for _, chunk := range []string{"hello", " ", "world"} {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatalf("Write(%q): %v", chunk, err)
		}
	}
	if got := string(w.Bytes()); got != "hello world" {
		t.Errorf("Bytes() = %q, want %q", got, "hello world")
	}
	// 12th byte pushes past the limit.
	if _, err := w.Write([]byte("!")); err == nil {
		t.Fatal("Write: want error beyond accumulated limit, got nil")
	}
}

func TestLimitedWriterErrorSticksAfterFirstFailure(t *testing.T) {
	// Once the limit is crossed, every subsequent Write must also fail.
	w := &limitedWriter{limit: 3}
	w.Write([]byte("toolong")) //nolint:errcheck // deliberately ignoring here
	_, err1 := w.Write([]byte("a"))
	_, err2 := w.Write([]byte("b"))
	if err1 == nil || err2 == nil {
		t.Errorf("Write after limit breach: err1=%v err2=%v, both want non-nil", err1, err2)
	}
}

func TestLimitedWriterZeroLimit(t *testing.T) {
	// A zero-byte limit must reject every write, including empty slices
	// (well, an empty slice has len=0 and 0+0 > 0 is false, so it passes).
	w := &limitedWriter{limit: 0}
	_, err := w.Write([]byte("x"))
	if err == nil {
		t.Fatal("Write with limit=0: want error, got nil")
	}
}

// ---- Run() unit tests ----

func TestRunMissingBinary(t *testing.T) {
	cfg := Config{
		BinaryPath:     "/nonexistent/llama-cli",
		ModelPath:      "/nonexistent/model.gguf",
		Threads:        4,
		MaxTokens:      256,
		Temperature:    0.0,
		Timeout:        5 * time.Second,
		MaxOutputBytes: 65536,
	}
	_, err := Run(context.Background(), cfg, "hello")
	if err == nil {
		t.Fatal("Run: want error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "llm: binary not found") {
		t.Errorf("err = %q, want 'llm: binary not found'", err.Error())
	}
}

func TestRunMissingModel(t *testing.T) {
	// /bin/sh is guaranteed to exist on any POSIX system; we use it only to
	// satisfy the BinaryPath stat check. The function returns before attempting
	// to actually exec the command because ModelPath stat fails first.
	cfg := Config{
		BinaryPath:     "/bin/sh",
		ModelPath:      "/nonexistent/model.gguf",
		Threads:        4,
		MaxTokens:      256,
		Temperature:    0.0,
		Timeout:        5 * time.Second,
		MaxOutputBytes: 65536,
	}
	_, err := Run(context.Background(), cfg, "hello")
	if err == nil {
		t.Fatal("Run: want error for missing model, got nil")
	}
	if !strings.Contains(err.Error(), "llm: model not found") {
		t.Errorf("err = %q, want 'llm: model not found'", err.Error())
	}
}

func TestRunMissingBinaryTakesPrecedenceOverMissingModel(t *testing.T) {
	// BinaryPath is checked before ModelPath; both missing → binary error wins.
	cfg := Config{
		BinaryPath:     "/nonexistent/llama-cli",
		ModelPath:      "/also/nonexistent/model.gguf",
		Timeout:        5 * time.Second,
		MaxOutputBytes: 65536,
	}
	_, err := Run(context.Background(), cfg, "hello")
	if err == nil {
		t.Fatal("Run: want error, got nil")
	}
	if !strings.Contains(err.Error(), "llm: binary not found") {
		t.Errorf("err = %q, want 'llm: binary not found' (not model error)", err.Error())
	}
}

func TestRunZeroTimeoutDoesNotBlock(t *testing.T) {
	// Timeout=0 disables the deadline. This test only checks the stat path
	// (binary missing), so it returns instantly regardless of Timeout value.
	cfg := Config{
		BinaryPath:     "/nonexistent/llama-cli",
		ModelPath:      "/nonexistent/model.gguf",
		Timeout:        0, // no deadline
		MaxOutputBytes: 1024,
	}
	_, err := Run(context.Background(), cfg, "hi")
	if err == nil || !strings.Contains(err.Error(), "llm: binary not found") {
		t.Errorf("err = %v, want binary-not-found error", err)
	}
}

func TestWriteGrammar(t *testing.T) {
	path, cleanup, err := WriteGrammar("file_explain.gbnf")
	if err != nil {
		t.Fatalf("WriteGrammar: %v", err)
	}
	defer cleanup()

	if path == "" {
		t.Fatal("WriteGrammar: expected path to temp file, got empty string")
	}

	// Verify file was written.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat temp grammar file: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(content), "root ::= \"{\" ws") {
		t.Errorf("temp grammar file contents do not look right:\n%s", string(content))
	}
}
