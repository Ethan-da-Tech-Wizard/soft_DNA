# CodeLens Local

> **A free, offline codebase explainer.** Point it at a project folder; it detects the tech stack, indexes the source, and generates a plain-English architecture report — without your code ever leaving your machine.

**Current release: v1.2.0** · Diagnostics & integrity gate complete (BRSL loop, source-grounded summaries, grounded `--ask`, hardened symbol extraction).

---

## Features

- **Stack detection** from manifest files (Go, Python, Node, Rust, Ruby, C/C++, R, Swift, SQL, and more)
- **Folder structure + entry-point mapping** by naming convention
- **Symbol extraction** (functions, classes, types, imports) via regex — no compiler required
- **SQLite/FTS5 index** for fast local keyword search
- **Mermaid architecture diagram** written to `architecture.mmd`
- **`--ask` mode** — answer a question about the codebase using indexed citations, no model needed
- **Grounded `--ask`** (optional, offline) — combine `--ask` with `--llm` for a plain-English answer generated from retrieved source snippets, always cited, falling back to citations-only if the model fails
- **Hybrid LLM summaries** (optional, offline) — project overview + per-file annotations grounded in verified symbols and bounded source excerpts, with real-time loop diagnostics on stderr
- **Cross-platform** — static CGO-free binary for macOS, Linux, and Windows

---

## Quick Start (Installation & Execution)

Download the correct release package for your operating system from the [GitHub Releases](https://github.com/Ethan-da-Tech-Wizard/soft_DNA/releases) page.

### 🐧 Linux
```sh
# Extract and run
tar xzf codelens-v1.2.0-linux-amd64.tar.gz
./codelens ./my-project
```

### 🍎 macOS
```sh
# Extract and run (use arm64 for Apple Silicon, amd64 for Intel)
tar xzf codelens-v1.2.0-darwin-arm64.tar.gz
./codelens ./my-project
```

### 🪟 Windows
1. Extract `codelens-v1.2.0-windows-amd64.zip`
2. Open command prompt (`cmd`) or PowerShell, navigate to the folder, and run:
```cmd
codelens.exe C:\path\to\my-project
```

### Outputs
Analysis writes the following to `./codelens-out/`:
* `CODEBASE_REPORT.md` — full formatted architecture report
* `architecture.mmd` — Mermaid diagram source
* `codelens.db` — SQLite index database (cached for fast Q&A)

### Ask a question

```sh
codelens --ask "where does the request routing happen?" ./my-project
# Writes: codelens-out/ANSWER.md  (FTS citations, no model call)

# With a local model: grounded plain-English answer, printed to stdout
# and written to ANSWER.md with sources (falls back to citations on failure)
codelens --ask "where does the request routing happen?" \
  --llm --llm-binary ./llama-cli --llm-model ./model.gguf ./my-project
```

---

## LLM Quick-Start (Offline AI Summaries)

CodeLens Local supports an optional hybrid summarization pass using a local
llama.cpp inference engine. **Nothing leaves your machine.**

Section 11 of the generated report ("AI-Assisted Summaries") is populated only
when `--llm` is enabled. Without it the report is purely deterministic.

### Step 1 — Install llama-cli

```sh
bash scripts/download_llama.sh
# Auto-detects your OS/arch, downloads the latest llama.cpp release,
# and installs llama-cli to ~/.codelens/bin/
```

Override the install directory:

```sh
CODELENS_BIN_DIR=/usr/local/bin bash scripts/download_llama.sh
```

### Step 2 — Download a model

```sh
bash scripts/download_model.sh
# Downloads Qwen2.5-Coder-1.5B-Instruct-Q4_K_M.gguf (~900 MB)
# to ~/.codelens/models/
```

Use a smaller or larger variant by passing the filename:

```sh
bash scripts/download_model.sh qwen2.5-coder-0.5b-instruct-q4_k_m.gguf
```

Override the model directory:

```sh
CODELENS_MODEL_DIR=/data/models bash scripts/download_model.sh
```

### Step 3 — Run with `--llm`

```sh
codelens \
  --llm \
  --llm-binary ~/.codelens/bin/llama-cli \
  --llm-model  ~/.codelens/models/qwen2.5-coder-1.5b-instruct-q4_k_m.gguf \
  ./my-project
```

### All `--llm` flags

| Flag | Default | Description |
|---|---|---|
| `--llm` | `false` | Enable the offline LLM summarization pass |
| `--llm-binary` | *(required when --llm)* | Path to the `llama-cli` executable |
| `--llm-model` | *(required when --llm)* | Path to the `.gguf` model file |
| `--llm-threads` | CPU count | Thread count passed to llama-cli `--threads` |
| `--llm-timeout` | `60s` | Per-call subprocess deadline |
| `--llm-max-files` | `10` | Maximum number of source files to annotate |

The LLM pass is **fail-soft**: if inference fails for any file (timeout, parse
error, hallucinated symbol name) the deterministic report is still written and a
`warning:` line is printed to stderr. The `--llm` flag never blocks the run.

---

## All Flags

```
codelens [flags] <directory>

  --out            string    Output directory (default: ./codelens-out)
  --ask            string    FTS question; writes ANSWER.md
  --max-file-size  int64     Skip files larger than this (bytes, default 2 MB)
  --version                  Print version and exit

  --llm                      Enable offline LLM summarization pass
  --llm-binary     string    Path to llama-cli executable
  --llm-model      string    Path to .gguf model file
  --llm-threads    int       Inference thread count (default: CPU count)
  --llm-timeout    duration  Per-call timeout (default: 60s)
  --llm-max-files  int       Max files to annotate (default: 10)
```

---

## Development

```sh
make lint          # gofmt + go vet
make test          # go test ./...
make build         # CGO_ENABLED=0 go build → bin/codelens
make smoke         # build + end-to-end smoke against sample fixtures
make run DIR=./testdata/sample_python
```

Sample fixtures in `testdata/`:

```
testdata/sample_python/    testdata/sample_node/     testdata/sample_rust/
testdata/sample_ruby/      testdata/sample_cpp/      testdata/sample_c/
testdata/sample_r/         testdata/sample_swift/    testdata/sample_sql/
```

---

## Release Build

```sh
make release VERSION=v1.2.0
```

Produces archives in `dist/` for the full cross-platform matrix:

| File | Platform |
|---|---|
| `codelens-v1.2.0-darwin-amd64.tar.gz` | macOS Intel |
| `codelens-v1.2.0-darwin-arm64.tar.gz` | macOS Apple Silicon |
| `codelens-v1.2.0-linux-amd64.tar.gz` | Linux x86-64 |
| `codelens-v1.2.0-linux-arm64.tar.gz` | Linux ARM64 |
| `codelens-v1.2.0-windows-amd64.zip` | Windows x86-64 |
| `codelens-v1.2.0-windows-arm64.zip` | Windows ARM64 |
| `checksums.txt` | SHA-256 for all archives |

Verify on Linux/macOS:

```sh
cd dist && sha256sum -c checksums.txt
# or on macOS without sha256sum:
cd dist && shasum -a 256 -c checksums.txt
```

---

## Safety Guardrails

CodeLens Local never:

- Executes or imports code from the scanned project
- Makes network calls during analysis
- Follows symlinks outside the scanned root
- Uses CGo (builds with `CGO_ENABLED=0`)

The optional LLM pass runs a separately-installed local binary
(`llama-cli`) as a subprocess — the scanned project's code is never passed
to any remote endpoint.

---

## Language Support

Detected automatically from file extensions and manifest files:

Go · Python · JavaScript/TypeScript · Rust · Ruby · C · C++ · R · Swift · SQL
