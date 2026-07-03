#!/usr/bin/env bash
# download_llama.sh — Download and install a pre-built llama-cli binary for CodeLens Local.
#
# Usage:
#   bash scripts/download_llama.sh
#
# Override the installation directory:
#   CODELENS_BIN_DIR=/usr/local/bin bash scripts/download_llama.sh
#
# Pin a specific llama.cpp release (default: query GitHub API for latest):
#   LLAMA_VERSION=b4500 bash scripts/download_llama.sh

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

TARGET_DIR="${CODELENS_BIN_DIR:-${HOME}/.codelens/bin}"
RELEASES_API="https://api.github.com/repos/ggerganov/llama.cpp/releases/latest"
# Fallback tag used when the GitHub API is unreachable (rate-limit / offline).
# Update this to a recent known-good release when bumping the project.
FALLBACK_TAG="b5000"

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------

echo "=== CodeLens Local llama.cpp Downloader ==="
echo ""

# ---------------------------------------------------------------------------
# Detect OS and architecture
# ---------------------------------------------------------------------------

OS="$(uname -s)"
ARCH="$(uname -m)"
echo "  Platform : ${OS} / ${ARCH}"

BINARY_NAME="llama-cli"

case "${OS}" in
    Darwin)
        case "${ARCH}" in
            arm64)  PLATFORM="macos-arm64" ;;
            x86_64) PLATFORM="macos-x64"   ;;
            *)
                echo "Error: unsupported macOS arch '${ARCH}'." >&2
                exit 1
                ;;
        esac
        ;;
    Linux)
        case "${ARCH}" in
            x86_64)         PLATFORM="ubuntu-x64"   ;;
            aarch64|arm64)  PLATFORM="ubuntu-arm64" ;;
            *)
                echo "Error: unsupported Linux arch '${ARCH}'." >&2
                echo "  Build from source: https://github.com/ggerganov/llama.cpp" >&2
                exit 1
                ;;
        esac
        ;;
    MINGW*|CYGWIN*|MSYS*|*NT*)
        PLATFORM="win-avx2-x64"
        BINARY_NAME="llama-cli.exe"
        ;;
    *)
        echo "Error: unsupported OS '${OS}'." >&2
        exit 1
        ;;
esac

echo "  Asset    : ${PLATFORM}"
echo "  Install  : ${TARGET_DIR}/${BINARY_NAME}"
echo ""

# ---------------------------------------------------------------------------
# Check for required tools
# ---------------------------------------------------------------------------

if command -v curl >/dev/null 2>&1; then
    DL_TOOL="curl"
elif command -v wget >/dev/null 2>&1; then
    DL_TOOL="wget"
else
    echo "Error: neither 'curl' nor 'wget' found." >&2
    exit 1
fi

if ! command -v unzip >/dev/null 2>&1; then
    if ! command -v python3 >/dev/null 2>&1 && ! command -v python >/dev/null 2>&1; then
        echo "Error: 'unzip' not found and no Python fallback available." >&2
        echo "  Install unzip: sudo apt install unzip  (Debian/Ubuntu)" >&2
        exit 1
    fi
fi

# ---------------------------------------------------------------------------
# Helper: fetch a URL to stdout
# ---------------------------------------------------------------------------

fetch() {
    local url="$1"
    if [[ "${DL_TOOL}" == "curl" ]]; then
        curl -fsSL "${url}"
    else
        wget -qO- "${url}"
    fi
}

# ---------------------------------------------------------------------------
# Resolve release tag
# ---------------------------------------------------------------------------

if [[ -n "${LLAMA_VERSION:-}" ]]; then
    TAG="${LLAMA_VERSION}"
    echo "  Using pinned version: ${TAG}"
else
    echo "  Fetching latest release from GitHub..."
    API_RESPONSE="$(fetch "${RELEASES_API}" 2>/dev/null || true)"
    # Extract the tag_name value using grep+sed — no jq or python required.
    TAG="$(printf '%s' "${API_RESPONSE}" \
          | grep '"tag_name"' \
          | head -1 \
          | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"

    if [[ -z "${TAG}" ]]; then
        echo "  GitHub API unavailable or rate-limited — using fallback: ${FALLBACK_TAG}"
        TAG="${FALLBACK_TAG}"
    else
        echo "  Latest release: ${TAG}"
    fi
fi

EXT="tar.gz"
if [[ "${PLATFORM}" == win-* ]]; then
    EXT="zip"
fi

DOWNLOAD_URL="https://github.com/ggerganov/llama.cpp/releases/download/${TAG}/llama-${TAG}-bin-${PLATFORM}.${EXT}"
echo "  URL      : ${DOWNLOAD_URL}"
echo ""

# ---------------------------------------------------------------------------
# Download to a temp directory (cleaned up on any exit)
# ---------------------------------------------------------------------------

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

ZIP_FILE="${WORK_DIR}/llama.${EXT}"
EXTRACT_DIR="${WORK_DIR}/extracted"
mkdir -p "${EXTRACT_DIR}"

echo "==> Downloading..."
if [[ "${DL_TOOL}" == "curl" ]]; then
    curl \
        --location \
        --retry 3 \
        --retry-delay 5 \
        --retry-all-errors \
        --progress-bar \
        --output "${ZIP_FILE}" \
        "${DOWNLOAD_URL}"
else
    wget \
        --tries=3 \
        --waitretry=5 \
        --show-progress \
        --output-document="${ZIP_FILE}" \
        "${DOWNLOAD_URL}"
fi

# ---------------------------------------------------------------------------
# Extract
# ---------------------------------------------------------------------------

echo ""
echo "==> Extracting..."

if [[ "${EXT}" == "zip" ]]; then
    if command -v unzip >/dev/null 2>&1; then
        # Full (non-flat) extraction so we can find the binary at any nesting level.
        unzip -q "${ZIP_FILE}" -d "${EXTRACT_DIR}"
    elif command -v python3 >/dev/null 2>&1; then
        python3 -m zipfile -e "${ZIP_FILE}" "${EXTRACT_DIR}"
    else
        python -m zipfile -e "${ZIP_FILE}" "${EXTRACT_DIR}"
    fi
else
    tar -xzf "${ZIP_FILE}" -C "${EXTRACT_DIR}"
fi

# ---------------------------------------------------------------------------
# Locate llama-cli inside the extracted tree
# ---------------------------------------------------------------------------

BIN_SRC="$(find "${EXTRACT_DIR}" -type f -name "${BINARY_NAME}" | head -1)"

if [[ -z "${BIN_SRC}" ]]; then
    echo "Error: '${BINARY_NAME}' not found in the downloaded archive." >&2
    echo "Files in archive:" >&2
    find "${EXTRACT_DIR}" -type f | head -20 >&2
    echo "" >&2
    echo "The release asset naming may have changed. Try:" >&2
    echo "  LLAMA_VERSION=<tag> bash scripts/download_llama.sh" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------

mkdir -p "${TARGET_DIR}"
cp "${BIN_SRC}" "${TARGET_DIR}/${BINARY_NAME}"
chmod +x "${TARGET_DIR}/${BINARY_NAME}"

SRC_DIR="$(dirname "${BIN_SRC}")"
# Also copy all shared libraries next to the binary, preserving symlinks
cp -d "${SRC_DIR}"/*.so* "${TARGET_DIR}/" 2>/dev/null || true
cp -d "${SRC_DIR}"/*.dylib* "${TARGET_DIR}/" 2>/dev/null || true
cp -d "${SRC_DIR}"/*.dll* "${TARGET_DIR}/" 2>/dev/null || true

# ---------------------------------------------------------------------------
# Verify
# ---------------------------------------------------------------------------

echo ""
echo "=== Installation complete ==="
echo "  Installed : ${TARGET_DIR}/${BINARY_NAME}"
echo ""
echo "  Version check:"
"${TARGET_DIR}/${BINARY_NAME}" --version 2>&1 | head -3 || true
echo ""
echo "  Add to PATH if needed:"
echo "    export PATH=\"${TARGET_DIR}:\$PATH\""
echo ""
echo "  Then run CodeLens Local with:"
echo "    codelens --llm-binary ${TARGET_DIR}/${BINARY_NAME} \\"
echo "             --llm-model ~/.codelens/models/<model.gguf> \\"
echo "             <project-directory>"
echo "==========================================="
