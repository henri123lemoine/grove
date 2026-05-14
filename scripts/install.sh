#!/usr/bin/env sh
# Install or upgrade grove on macOS / Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/henri123lemoine/grove/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/henri123lemoine/grove/main/scripts/install.sh | sh -s -- v0.2.0
#
# Env vars:
#   GROVE_INSTALL_DIR   Install destination (default: $HOME/.local/bin)
#   GROVE_REPO          Override the source repo (default: henri123lemoine/grove)

set -eu

REPO="${GROVE_REPO:-henri123lemoine/grove}"
DEST="${GROVE_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${1:-latest}"

die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar  >/dev/null 2>&1 || die "tar is required"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    darwin|linux) ;;
    *) die "unsupported OS: $os (try downloading from https://github.com/$REPO/releases)" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64)  arch=x86_64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "unsupported architecture: $arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
        | sed -nE 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/p')
    [ -n "$VERSION" ] || die "could not resolve latest release"
fi

num="${VERSION#v}"
url="https://github.com/$REPO/releases/download/$VERSION/grove_${num}_${os}_${arch}.tar.gz"

mkdir -p "$DEST"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

printf 'Downloading grove %s (%s/%s)...\n' "$VERSION" "$os" "$arch"
curl -fsSL "$url" -o "$tmp/grove.tar.gz" \
    || die "download failed: $url"
tar -xzf "$tmp/grove.tar.gz" -C "$tmp" grove \
    || die "extract failed"
chmod +x "$tmp/grove"
mv "$tmp/grove" "$DEST/grove"

printf 'Installed grove %s → %s/grove\n' "$VERSION" "$DEST"

case ":$PATH:" in
    *":$DEST:"*) ;;
    *) printf '\nNote: %s is not on your PATH. Add this to your shell profile:\n  export PATH="%s:$PATH"\n' "$DEST" "$DEST" ;;
esac

"$DEST/grove" --version
