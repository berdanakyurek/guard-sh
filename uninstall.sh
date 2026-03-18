#!/usr/bin/env bash

set -e

BIN_DIR="${HOME}/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/guard-sh"
KEEP_CONFIG=0

for arg in "$@"; do
    [[ "$arg" == "--keep-config" ]] && KEEP_CONFIG=1
done

echo "=== guard-sh uninstaller ==="
echo ""

# --- Shell integration ---
for rc in "$HOME/.bashrc" "$HOME/.zshrc" "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish"; do
    [[ -f "$rc" ]] || continue
    if grep -qE "guard\.(bash|zsh|fish)|^# guard-sh$|^guard-sh on$" "$rc" 2>/dev/null; then
        grep -vE "guard\.(bash|zsh|fish)|^# guard-sh$|^guard-sh on$" "$rc" > "${rc}.guardtmp" \
            && mv "${rc}.guardtmp" "$rc"
        echo "Shell integration removed from $rc"
    fi
done

echo ""

# --- Config dir ---
if [[ $KEEP_CONFIG -eq 0 ]]; then
    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
        echo "Config removed: $CONFIG_DIR"
    fi
else
    echo "Config kept: $CONFIG_DIR (--keep-config)"
fi

# --- Binary ---
if [[ -f "$BIN_DIR/guard-sh" ]]; then
    rm -f "$BIN_DIR/guard-sh"
    echo "Binary removed: $BIN_DIR/guard-sh"
fi

echo ""
echo "Done! Restart your shell to complete the uninstallation."
