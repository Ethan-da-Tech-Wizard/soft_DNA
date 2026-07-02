#!/usr/bin/env bash
# download_model.sh — Download a quantized GGUF model for CodeLens Local.
#
# Usage:
#   bash scripts/download_model.sh [MODEL_FILENAME]
#
# Examples:
#   bash scripts/download_model.sh
#   bash scripts/download_model.sh qwen2.5-coder-0.5b-instruct-q4_k_m.gguf
#
# Override the target directory:
#   CODELENS_MODEL_DIR=/data/models bash scripts/download_model.sh
#
# Hugging Face token (only needed for gated models):
#   HF_TOKEN=hf_... bash scripts/download_model.sh

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

HF_REPO="Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF"
DEFAULT_MODEL="qwen2.5-coder-1.5b-instruct-q4_k_m.gguf"

MODEL_FILE="${1:-${DEFAULT_MODEL}}"
TARGET_DIR="${CODELENS_MODEL_DIR:-${HOME}/.codelens/models}"
HF_TOKEN="${HF_TOKEN:-}"
TARGET_PATH="${TARGET_DIR}/${MODEL_FILE}"

# ?download=true tells the HF hub to serve the raw bytes instead of a preview page.
MODEL_URL="https://huggingface.co/${HF_REPO}/resolve/main/${MODEL_FILE}?download=true"

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------

echo "=== CodeLens Local Model Downloader ==="
echo "  Model  : ${MODEL_FILE}"
echo "  Repo   : https://huggingface.co/${HF_REPO}"
echo "  Dest   : ${TARGET_PATH}"
echo ""

# ---------------------------------------------------------------------------
# Detect download tool
# ---------------------------------------------------------------------------

if command -v curl >/dev/null 2>&1; then
    DL_TOOL="curl"
elif command -v wget >/dev/null 2>&1; then
    DL_TOOL="wget"
else
    echo "Error: neither 'curl' nor 'wget' found on this system." >&2
    echo "Install one and retry:" >&2
    echo "  Debian/Ubuntu : sudo apt install curl" >&2
    echo "  Fedora/RHEL   : sudo dnf install curl" >&2
    echo "  macOS (Homebrew): brew install curl" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Create destination directory
# ---------------------------------------------------------------------------

mkdir -p "${TARGET_DIR}"

# ---------------------------------------------------------------------------
# Resume notice
# ---------------------------------------------------------------------------

if [[ -f "${TARGET_PATH}" ]]; then
    echo "Found existing file — will attempt to resume partial download..."
fi
echo "Downloading (this may take a while for large models)..."
echo ""

# ---------------------------------------------------------------------------
# Download — direct calls without eval to avoid injection risk
# ---------------------------------------------------------------------------

if [[ "${DL_TOOL}" == "curl" ]]; then
    CURL_ARGS=(
        --location            # follow HuggingFace CDN redirects
        --continue-at -       # resume partial downloads
        --retry 3
        --retry-delay 5
        --retry-all-errors
        --progress-bar
        --output "${TARGET_PATH}"
    )
    if [[ -n "${HF_TOKEN}" ]]; then
        CURL_ARGS+=(--header "Authorization: Bearer ${HF_TOKEN}")
    fi
    curl "${CURL_ARGS[@]}" "${MODEL_URL}"
else
    WGET_ARGS=(
        --continue            # resume partial downloads
        --tries=3
        --waitretry=5
        --show-progress
        --output-document="${TARGET_PATH}"
    )
    if [[ -n "${HF_TOKEN}" ]]; then
        WGET_ARGS+=(--header "Authorization: Bearer ${HF_TOKEN}")
    fi
    wget "${WGET_ARGS[@]}" "${MODEL_URL}"
fi

# ---------------------------------------------------------------------------
# Verify
# ---------------------------------------------------------------------------

if [[ ! -f "${TARGET_PATH}" ]]; then
    echo "" >&2
    echo "Error: download finished but ${TARGET_PATH} is missing." >&2
    exit 1
fi

FINAL_SIZE=$(wc -c <"${TARGET_PATH}")
echo ""
echo "=== Download complete! ==="
echo "  Size      : ${FINAL_SIZE} bytes"
echo "  Full path : ${TARGET_PATH}"
echo ""
echo "Run CodeLens Local with this model:"
echo "  codelens --llm-binary ~/.codelens/bin/llama-cli \\"
echo "           --llm-model \"${TARGET_PATH}\" \\"
echo "           <project-directory>"
echo ""
echo "Or set the path as an environment variable:"
echo "  export CODELENS_MODEL=\"${TARGET_PATH}\""
echo "========================================"
