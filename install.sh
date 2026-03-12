#!/usr/bin/env bash

set -e

INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/guard-sh"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
WITH_SHELL=1

for arg in "$@"; do
    [[ "$arg" == "--without-shell" ]] && WITH_SHELL=0
done

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

# --- Shell integration ---
if [[ $WITH_SHELL -eq 1 ]]; then
    SHELL_NAME="$(basename "$SHELL")"
    case "$SHELL_NAME" in
        zsh)
            RC_FILE="$HOME/.zshrc"
            SHELL_SCRIPT="$INSTALL_DIR/shell/guard.zsh"
            ;;
        bash)
            RC_FILE="$HOME/.bashrc"
            SHELL_SCRIPT="$INSTALL_DIR/shell/guard.bash"
            ;;
        *)
            echo "Unsupported shell: $SHELL_NAME (supported: zsh, bash)"
            echo "Manually source the appropriate file from $INSTALL_DIR/shell/"
            exit 0
            ;;
    esac

    SOURCE_LINE="source \"$SHELL_SCRIPT\""
    ON_LINE="guard-sh on"
    if grep -qF "$SOURCE_LINE" "$RC_FILE" 2>/dev/null; then
        if ! grep -qF "$ON_LINE" "$RC_FILE" 2>/dev/null; then
            echo "$ON_LINE" >> "$RC_FILE"
            echo "Shell integration updated in $RC_FILE (added guard-sh on)"
            echo "Restart your shell or run: source $RC_FILE"
        else
            echo "Shell integration already present in $RC_FILE"
        fi
    else
        printf '\n# guard-sh\n%s\n%s\n' "$SOURCE_LINE" "$ON_LINE" >> "$RC_FILE"
        echo "Shell integration added to $RC_FILE"
        echo "Restart your shell or run: source $RC_FILE"
    fi
    echo ""
fi

echo "Done! Run: guard-sh check \"<command>\""
