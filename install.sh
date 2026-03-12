#!/usr/bin/env bash

set -e

INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/guard-sh"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

echo "=== guard-sh installer ==="
echo ""

# --- Dependencies ---
if ! command -v go &>/dev/null; then
    echo "Error: Go is required. Install it from https://go.dev/dl/"
    exit 1
fi

# --- Build ---
echo "Building guard-sh..."
mkdir -p "$BIN_DIR"
cd "$INSTALL_DIR"
go build -o "$BIN_DIR/guard-sh" .
echo "Binary installed: $BIN_DIR/guard-sh"

if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo "Warning: $BIN_DIR is not in your PATH. Add the following to your shell config:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""

# --- Config ---
mkdir -p "$CONFIG_DIR"

cp "$INSTALL_DIR/config.example.yaml" "$CONFIG_FILE"
chmod 600 "$CONFIG_FILE"
echo "Config created: $CONFIG_FILE"
echo "Edit it and set your api_key before using guard-sh."

cp "$INSTALL_DIR/prompt.txt" "$CONFIG_DIR/prompt.txt"
echo "Prompt copied: $CONFIG_DIR/prompt.txt (edit to customize)"

echo ""
echo "Done! Run: guard-sh check \"<command>\""
