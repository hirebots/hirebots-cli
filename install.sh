#!/bin/sh
#
# HireBots CLI installer — one-command install:
#
#   curl -fsSL https://hirebots.ai/install.sh | sh
#
# Detects OS + architecture, downloads the correct binary, installs to PATH.
#

set -e

# --- Detect OS ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)  OS="linux"  ;;
    darwin) OS="darwin" ;;
    *)
        echo "Error: unsupported OS '$(uname -s)'. Only Linux and macOS are supported." >&2
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)  ARCH="amd64"  ;;
    aarch64|arm64) ARCH="arm64"  ;;
    *)
        echo "Error: unsupported architecture '$ARCH'. Only amd64 and arm64 are supported." >&2
        exit 1
        ;;
esac

BINARY="hirebots-${OS}-${ARCH}"
URL="https://hirebots.ai/downloads/${BINARY}"

# --- Determine install directory ---
# Prefer /usr/local/bin (system-wide), fall back to ~/.local/bin (user-local)
INSTALL_DIR=""
for dir in /usr/local/bin "$HOME/.local/bin"; do
    if [ -d "$dir" ] && [ -w "$dir" ]; then
        INSTALL_DIR="$dir"
        break
    fi
done

if [ -z "$INSTALL_DIR" ]; then
    # Create ~/.local/bin if nothing else works
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# --- Download ---
TMPFILE=$(mktemp)
echo "Downloading HireBots CLI (${OS}/${ARCH})..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMPFILE" || {
        echo "Error: download failed from $URL" >&2
        rm -f "$TMPFILE"
        exit 1
    }
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMPFILE" "$URL" || {
        echo "Error: download failed from $URL" >&2
        rm -f "$TMPFILE"
        exit 1
    }
else
    echo "Error: neither curl nor wget found." >&2
    rm -f "$TMPFILE"
    exit 1
fi

# --- Install ---
chmod +x "$TMPFILE"
mv "$TMPFILE" "${INSTALL_DIR}/hirebots"

echo ""
echo "✓ HireBots CLI installed to ${INSTALL_DIR}/hirebots"

# --- Check PATH ---
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo ""
        echo "⚠ ${INSTALL_DIR} is not in your PATH."
        echo "  Add this line to your shell profile (~/.bashrc or ~/.zshrc):"
        echo ""
        echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        ;;
esac

# --- Verify ---
echo "Verifying..."
"${INSTALL_DIR}/hirebots" --help >/dev/null 2>&1 && echo "✓ Ready! Run 'hirebots --help' to get started." || echo "✓ Installed. Run '${INSTALL_DIR}/hirebots --help' to verify."