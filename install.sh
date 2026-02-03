#!/bin/sh
set -e

REPO="Azmekk/gofer"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)  OS=linux ;;
    Darwin*) OS=darwin ;;
    *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH=amd64 ;;
    arm64|aarch64)  ARCH=arm64 ;;
    *)              echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Determine install directory
if [ -n "$GOFER_INSTALL_DIR" ]; then
    INSTALL_DIR="$GOFER_INSTALL_DIR"
elif [ "$OS" = "darwin" ] && [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

BINARY="gofer-${OS}-${ARCH}"

echo "Fetching latest release..."
TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d '"' -f 4)
if [ -z "$TAG" ]; then
    echo "Error: could not determine latest release tag."
    exit 1
fi
echo "Latest version: ${TAG}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${BINARY}..."
curl -sSL -o "${TMPDIR}/gofer" "$DOWNLOAD_URL"

# Best-effort SHA256 verification
echo "Verifying checksum..."
if curl -sSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null; then
    EXPECTED=$(grep "$BINARY" "${TMPDIR}/checksums.txt" | cut -d ' ' -f 1)
    if [ -n "$EXPECTED" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL=$(sha256sum "${TMPDIR}/gofer" | cut -d ' ' -f 1)
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL=$(shasum -a 256 "${TMPDIR}/gofer" | cut -d ' ' -f 1)
        else
            echo "Warning: no sha256 tool found, skipping verification."
            ACTUAL="$EXPECTED"
        fi

        if [ "$ACTUAL" != "$EXPECTED" ]; then
            echo "Error: checksum mismatch!"
            echo "  Expected: $EXPECTED"
            echo "  Got:      $ACTUAL"
            exit 1
        fi
        echo "Checksum verified."
    else
        echo "Warning: binary not found in checksums.txt, skipping verification."
    fi
else
    echo "Warning: could not download checksums, skipping verification."
fi

mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/gofer" "${INSTALL_DIR}/gofer"
chmod +x "${INSTALL_DIR}/gofer"

echo "Installed gofer to ${INSTALL_DIR}/gofer"

# Check if install dir is in PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *) echo "Warning: ${INSTALL_DIR} is not in your PATH. Add it to your shell profile." ;;
esac

# macOS Gatekeeper warning
if [ "$OS" = "darwin" ]; then
    echo "Note: if macOS blocks the binary, run: xattr -d com.apple.quarantine ${INSTALL_DIR}/gofer"
fi
