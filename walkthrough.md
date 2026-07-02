# CodeLens Local v1.0.0 — Developer Walkthrough

**Status:** release candidate compiled, archived, and hashed. Ready for distribution.

---

## What v1.0.0 Includes

| Area | Capability |
|---|---|
| Scanner | Walks a directory tree; skips binaries, ignores, and oversized files |
| Stack detection | Identifies language + toolchain from manifest files (Go, Python, Node, Rust, Ruby, C, C++, R, Swift, SQL) |
| Node.js lockfiles | Distinguishes npm / Yarn / pnpm / Bun from `package.json` companion files |
| Symbol extraction | Regex-based extraction of functions, types, classes, imports — no compiler |
| SQLite/FTS5 index | Persistent local index for sub-millisecond keyword search |
| `--ask` mode | Answers a natural-language question with FTS citations; no model required |
| Mermaid diagram | Writes `architecture.mmd` — paste into any Mermaid renderer |
| Offline LLM pass | `--llm` flag drives a local llama.cpp subprocess; project overview + per-file annotations |
| GBNF-constrained output | Sampler-level grammar enforcement; no hallucinated JSON keys |
| Post-generation validation | Symbol allowlist check; sanitized, truncated prose; fail-soft recovery |
| Cross-platform builds | Six static binaries (macOS Intel/ARM, Linux x86-64/ARM64, Windows x86-64/ARM64) |

---

## Release Artifacts

`make release VERSION=v1.0.0` produces `dist/`:

```
codelens-v1.0.0-darwin-amd64.tar.gz
codelens-v1.0.0-darwin-arm64.tar.gz
codelens-v1.0.0-linux-amd64.tar.gz
codelens-v1.0.0-linux-arm64.tar.gz
codelens-v1.0.0-windows-amd64.zip
codelens-v1.0.0-windows-arm64.zip
checksums.txt
```

Each archive contains a single binary at its root (`codelens` or `codelens.exe`).
`checksums.txt` contains SHA-256 hashes for all six archives; it does not hash itself.

### Verifying checksums

```sh
cd dist

# Linux / WSL
sha256sum -c checksums.txt

# macOS (no sha256sum by default)
shasum -a 256 -c checksums.txt
```

All six lines should read `OK`.

---

## Step 1 — Get the Binary

**From a release archive (recommended):**

```sh
# Linux x86-64
tar xzf codelens-v1.0.0-linux-amd64.tar.gz
chmod +x codelens
sudo mv codelens /usr/local/bin/
codelens --version   # codelens v1.0.0
```

**Build from source:**

```sh
git clone <repo>
cd soft_DNA-main
make build VERSION=v1.0.0
# → bin/codelens
```

---

## Step 2 — Run a Basic Scan

```sh
codelens ./my-project
```

Writes to `./codelens-out/`:

| File | Contents |
|---|---|
| `CODEBASE_REPORT.md` | Full architecture report in Markdown |
| `architecture.mmd` | Mermaid diagram source |
| `codelens.db` | SQLite index (keep for repeated `--ask` queries) |

Open `CODEBASE_REPORT.md` in any Markdown viewer for the formatted report.

### Ask a question

```sh
codelens --ask "where is the database connection opened?" ./my-project
# Writes codelens-out/ANSWER.md
```

No model is needed. The answer uses FTS5 full-text search over the indexed source.

### Change the output directory

```sh
codelens --out /tmp/report ./my-project
```

### Skip large files

```sh
codelens --max-file-size 524288 ./my-project   # 512 KB limit
```

---

## Step 3 — Enable Offline LLM Summaries (Optional)

The `--llm` flag activates Section 11 of the report ("AI-Assisted Summaries").
It requires two one-time setup steps. Nothing is sent over the network.

### 3a. Install llama-cli

```sh
bash scripts/download_llama.sh
```

The script auto-detects your OS and CPU architecture, downloads the latest
llama.cpp binary release from GitHub, and installs `llama-cli` to
`~/.codelens/bin/`. Override with `CODELENS_BIN_DIR`.

Verify:

```sh
~/.codelens/bin/llama-cli --version
```

### 3b. Download a model

```sh
bash scripts/download_model.sh
```

Downloads `Qwen2.5-Coder-1.5B-Instruct-Q4_K_M.gguf` (~900 MB) from Hugging Face
to `~/.codelens/models/`. Override directory with `CODELENS_MODEL_DIR`.

For a lighter model (~300 MB):

```sh
bash scripts/download_model.sh qwen2.5-coder-0.5b-instruct-q4_k_m.gguf
```

Private or gated models require `HF_TOKEN`:

```sh
HF_TOKEN=hf_xxx bash scripts/download_model.sh
```

### 3c. Run with `--llm`

```sh
codelens \
  --llm \
  --llm-binary ~/.codelens/bin/llama-cli \
  --llm-model  ~/.codelens/models/qwen2.5-coder-1.5b-instruct-q4_k_m.gguf \
  ./my-project
```

Additional LLM tuning flags:

| Flag | Default | Notes |
|---|---|---|
| `--llm-threads` | CPU count | Lower to reduce heat on laptops |
| `--llm-timeout` | `60s` | Increase for large files on slow machines |
| `--llm-max-files` | `10` | Cap how many files receive per-file annotations |

If any LLM call fails (timeout, parse error, hallucinated symbol), the
deterministic report is still written and a `warning:` line appears on stderr.
The `--llm` flag never prevents the run from completing.

---

## What the LLM Pass Does

1. **Grounding**: Reads verified facts from the SQLite index (file list, stack,
   entry points, top symbols by reference count).
2. **Project overview**: A single llama-cli call with a structured prompt asking
   for a plain-English project description. Output constrained by `project_overview.gbnf`.
3. **Per-file annotation**: For each of the top N files (by symbol count, up to
   `--llm-max-files`), a second call generates a file summary and per-symbol
   explanations. Output constrained by `file_explain.gbnf`.
4. **Validation**: Symbol names are checked against the allowlist from the index.
   Unknown names are discarded. All prose is sanitized (backticks, URLs, and
   absolute paths stripped) and truncated to safe lengths.
5. **Report merge**: LLM results are merged into Section 11 of
   `CODEBASE_REPORT.md`. Sections 1–10 are identical to the deterministic path.

GBNF grammars are embedded in the binary — no grammar files need to be
distributed alongside the release archives.

---

## Safety Properties

- No code from the scanned project is executed or imported
- No network calls during analysis; the LLM pass calls a local subprocess only
- Symlink traversal is bounded to the scanned root
- `CGO_ENABLED=0` — fully static binary, no shared library dependencies
- Fail-soft throughout: a single unreadable file or a failed LLM call logs a
  warning and the run continues

---

## Development Cheatsheet

```sh
make test          # go test ./...  (all packages)
make lint          # gofmt -l + go vet
make build         # CGO_ENABLED=0 → bin/codelens
make smoke         # build + basic end-to-end against sample_python fixture
make release VERSION=v1.0.0
```

Test fixtures live in `testdata/`:

```
sample_python   sample_node    sample_rust
sample_ruby     sample_cpp     sample_c
sample_r        sample_swift   sample_sql
sample_go       sample_empty   edge_cases
```
