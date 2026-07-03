// Package llm runs a local llama.cpp subprocess and returns its JSON output.
// It is the only pipeline component that spawns a child process; all other
// stages are pure in-process computation (G-1 exemption: the target project
// is never run — only the separately-installed inference binary is).
package llm

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

//go:embed grammars/*.gbnf
var grammarsFS embed.FS

// WriteGrammar writes the embedded grammar file to a temporary file on disk
// and returns its path and a cleanup function. Callers must execute the cleanup
// function when finished to remove the temp file.
func WriteGrammar(name string) (string, func(), error) {
	data, err := grammarsFS.ReadFile("grammars/" + name)
	if err != nil {
		return "", nil, fmt.Errorf("llm: read embedded grammar %s: %w", name, err)
	}
	f, err := os.CreateTemp("", "codelens-*.gbnf")
	if err != nil {
		return "", nil, fmt.Errorf("llm: create temp grammar file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("llm: write temp grammar file: %w", err)
	}
	cleanup := func() {
		os.Remove(f.Name())
	}
	return f.Name(), cleanup, nil
}

// Config holds all parameters needed to invoke a llama.cpp llama-cli process.
// Every field is explicit; there are no package-level defaults so callers
// can reason about resource usage at the call site.
type Config struct {
	BinaryPath     string        // absolute path to the llama-cli executable
	ModelPath      string        // absolute path to the .gguf model file
	GrammarPath    string        // optional path to a .gbnf grammar file; empty = no grammar
	Threads        int           // CPU thread count passed to --threads
	MaxTokens      int           // token generation cap passed to --n-predict
	Temperature    float64       // sampling temperature passed to --temp
	Timeout        time.Duration // wall-clock limit for the subprocess; 0 = no timeout
	MaxOutputBytes int64         // byte ceiling on stdout; enforces G-7 resource cap
	GPULayers      int           // GPU layers passed to --gpu-layers / -ngl; 0 = default (no offload)
}

// limitedWriter is an io.Writer that accumulates bytes up to a fixed ceiling.
// Once the ceiling is crossed the write is rejected, the error is stored on
// the struct, and all subsequent writes also fail. This prevents unbounded
// memory growth if an LLM produces runaway output.
type limitedWriter struct {
	buf     bytes.Buffer
	limit   int64
	written int64
	err     error
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.written+int64(len(p)) > w.limit {
		w.err = fmt.Errorf("llm: output exceeded %d-byte limit", w.limit)
		return 0, w.err
	}
	n, err := w.buf.Write(p)
	w.written += int64(n)
	return n, err
}

// Bytes returns the accumulated output collected so far.
func (w *limitedWriter) Bytes() []byte {
	return w.buf.Bytes()
}

// Run invokes the llama-cli binary and returns its stdout as raw bytes.
// The caller is responsible for parsing the returned bytes as JSON.
//
// Stderr is unconditionally discarded: llama.cpp writes progress bars and
// verbose log lines there; they are noise for callers of this function.
//
// If cfg.Timeout is positive, the subprocess is cancelled after that duration.
// If cfg.MaxOutputBytes is exceeded, Run returns an error and discards all output.
func Run(ctx context.Context, cfg Config, prompt string) ([]byte, error) {
	if _, err := os.Stat(cfg.BinaryPath); err != nil {
		return nil, fmt.Errorf("llm: binary not found: %w", err)
	}
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		return nil, fmt.Errorf("llm: model not found: %w", err)
	}

	args := make([]string, 0, 14)
	args = append(args, "--model", cfg.ModelPath)
	if cfg.GrammarPath != "" {
		args = append(args, "--grammar-file", cfg.GrammarPath)
	}
	if cfg.GPULayers > 0 {
		args = append(args, "--gpu-layers", strconv.Itoa(cfg.GPULayers))
	}
	args = append(args,
		"--threads", strconv.Itoa(cfg.Threads),
		"--n-predict", strconv.Itoa(cfg.MaxTokens),
		"--temp", strconv.FormatFloat(cfg.Temperature, 'f', -1, 64),
		"--no-display-prompt",
		"-p", prompt,
	)

	runCtx := ctx
	cancel := func() {}
	if cfg.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, cfg.BinaryPath, args...)
	out := &limitedWriter{limit: cfg.MaxOutputBytes}
	cmd.Stdout = out
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		if out.err != nil {
			return nil, out.err
		}
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("llm: timed out after %s (stderr: %q)", cfg.Timeout, stderrBuf.String())
		}
		return nil, fmt.Errorf("llm: run: %w (stderr: %q)", err, stderrBuf.String())
	}
	if out.err != nil {
		return nil, out.err
	}
	if len(out.Bytes()) == 0 {
		return nil, fmt.Errorf("llm: run returned empty output (stderr: %q)", stderrBuf.String())
	}
	return out.Bytes(), nil
}
