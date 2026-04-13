#!/bin/bash
set -euo pipefail

BINARY="/usr/local/bin/fpproto"
CONFIG_DIR="$HOME/.fpproto"

if [ ! -f "$BINARY" ]; then
  echo "fpproto is not installed."
  exit 0
fi

echo "Removing $BINARY..."
rm -f "$BINARY"

if [ -d "$CONFIG_DIR" ]; then
  echo "Removing config at $CONFIG_DIR..."
  rm -rf "$CONFIG_DIR"
fi

echo "fpproto has been uninstalled."
