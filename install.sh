#!/bin/bash
set -euo pipefail

REPO="fieldpulse-prototypes/fpproto"
INSTALL_DIR="/usr/local/bin"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) BINARY="fpproto-darwin-arm64" ;;
  x86_64)        BINARY="fpproto-darwin-amd64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release URL
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*$BINARY" | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
  echo "Failed to find latest release."
  exit 1
fi

echo "Downloading fpproto..."
curl -fsSL "$LATEST" -o "$INSTALL_DIR/fpproto"
chmod +x "$INSTALL_DIR/fpproto"

echo "fpproto installed to $INSTALL_DIR/fpproto"
fpproto --version
