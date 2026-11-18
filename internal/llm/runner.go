// Program: soft_DNA - runner.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Runs the local llama-cli subprocess with a prompt and grammar, enforcing a timeout and an output size cap, and streams the JSON answer text to an optional callback.

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
	"strings"
	"time"
)

// grammarsFS holds the GBNF grammar files compiled into the binary so they ship with it.
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
	TokenCallback  func(string)  // optional callback for streaming tokens in real-time
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

// Write appends p unless doing so would pass the limit; the first overflow is remembered and every later write fails too.
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

	// Build the llama-cli command line; the grammar and GPU flags are only added when configured.
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

	// Wrap the caller's context with a deadline when a timeout is configured.
	runCtx := ctx
	cancel := func() {}
	if cfg.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	}
	defer cancel()

	// Stdout is read through a pipe so output can be streamed; stderr is captured only for error messages.
	cmd := exec.CommandContext(runCtx, cfg.BinaryPath, args...)
	out := &limitedWriter{limit: cfg.MaxOutputBytes}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Read stdout in 1 KB chunks, enforce the size cap, and feed each chunk to the answer streamer when a callback is set.
	buf := make([]byte, 1024)
	state := 0 // 0: searching key, 1: searching start quote, 2: capturing content
	keyBuf := ""
	escaped := false

	for {
		n, err := stdoutPipe.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, werr := out.Write(chunk); werr != nil {
				return nil, werr
			}

			if cfg.TokenCallback != nil {
				parseAnswerStream(chunk, &state, &keyBuf, &escaped, cfg.TokenCallback)
			}
		}
		if err != nil {
			break
		}
	}

	// Report the most specific failure: output overflow first, then timeout, then any other exit error.
	if err := cmd.Wait(); err != nil {
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

// parseAnswerStream is a small state machine over streamed JSON bytes that finds the "answer" key and passes the characters of its string value to tokenCallback, decoding common escapes. state is 0 while looking for the key, 1 while waiting for the opening quote, 2 while copying the value, and 4 once the closing quote is seen.
func parseAnswerStream(chunk []byte, state *int, keyBuf *string, escaped *bool, tokenCallback func(string)) {
	for _, b := range chunk {
		switch *state {
		case 0:
			if b == ':' {
				if strings.Contains(*keyBuf, "answer") {
					*state = 1
				}
			} else if b >= 'a' && b <= 'z' {
				*keyBuf += string(b)
			}
		case 1:
			if b == '"' {
				*state = 2
				*escaped = false
			}
		case 2:
			if *escaped {
				switch b {
				case 'n':
					tokenCallback("\n")
				case 't':
					tokenCallback("\t")
				case '"':
					tokenCallback("\"")
				case '\\':
					tokenCallback("\\")
				default:
					tokenCallback(string(b))
				}
				*escaped = false
			} else {
				if b == '\\' {
					*escaped = true
				} else if b == '"' {
					*state = 4 // Done
				} else {
					tokenCallback(string(b))
				}
			}
		}
	}
}
