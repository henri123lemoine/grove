#!/usr/bin/env bash
# grove.tmux - TPM-compatible plugin for grove integration
#
# Automatically downloads grove binary if not found in PATH.
# Updates binary when plugin is updated via TPM (prefix + U).

set -e

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GROVE_BIN_DIR="$CURRENT_DIR/bin"
GROVE_BIN="$GROVE_BIN_DIR/grove"
VERSION_FILE="$GROVE_BIN_DIR/.version"

# Get tmux option with default
get_tmux_option() {
    local option="$1"
    local default_value="$2"
    local option_value
    option_value=$(tmux show-option -gqv "$option")
    if [ -z "$option_value" ]; then
        echo "$default_value"
    else
        echo "$option_value"
    fi
}

# Detect OS and architecture, output format matching goreleaser
detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) arch="x86_64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            tmux display-message "Grove: Unsupported architecture: $arch"
            return 1
            ;;
    esac

    case "$os" in
        darwin|linux) ;;
        *)
            tmux display-message "Grove: Unsupported OS: $os"
            return 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -sfL "https://api.github.com/repos/henri123lemoine/grove/releases/latest" 2>/dev/null | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    echo "$version"
}

# Download and install grove binary
download_grove() {
    local platform version url tmp_dir

    platform=$(detect_platform) || return 1
    version=$(get_latest_version)

    if [ -z "$version" ]; then
        tmux display-message "Grove: Failed to get latest version from GitHub"
        return 1
    fi

    mkdir -p "$GROVE_BIN_DIR"
    tmp_dir=$(mktemp -d)

    # goreleaser names: grove_0.1.0_darwin_arm64.tar.gz
    local version_num="${version#v}"
    url="https://github.com/henri123lemoine/grove/releases/download/${version}/grove_${version_num}_${platform}.tar.gz"

    tmux display-message "Grove: Downloading $version..."

    if curl -sfL "$url" -o "$tmp_dir/grove.tar.gz" && \
       tar xzf "$tmp_dir/grove.tar.gz" -C "$tmp_dir" && \
       mv "$tmp_dir/grove" "$GROVE_BIN" && \
       chmod +x "$GROVE_BIN"; then
        echo "$version" > "$VERSION_FILE"
        rm -rf "$tmp_dir"
        tmux display-message "Grove: Installed $version"
        return 0
    else
        rm -rf "$tmp_dir"
        tmux display-message "Grove: Download failed. Install manually: brew install henri123lemoine/tap/grove"
        return 1
    fi
}

# Get currently installed version
get_installed_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    fi
}

# Ensure grove is available and up to date
ensure_grove() {
    # If grove is in PATH, prefer that (user-managed installation)
    if command -v grove &> /dev/null; then
        echo "grove"
        return 0
    fi

    local installed_version latest_version

    # Check if we have a local binary
    if [ -x "$GROVE_BIN" ]; then
        installed_version=$(get_installed_version)

        # On TPM update, check if we need to update the binary
        # Only check GitHub if we have network (don't block tmux startup)
        latest_version=$(get_latest_version 2>/dev/null || echo "")

        if [ -n "$latest_version" ] && [ "$installed_version" != "$latest_version" ]; then
            # Update available, download in background
            download_grove &>/dev/null &
        fi

        echo "$GROVE_BIN"
        return 0
    fi

    # No binary found, need to download
    if download_grove; then
        echo "$GROVE_BIN"
        return 0
    fi

    return 1
}

# Main setup
main() {
    local grove_cmd grove_key

    grove_cmd=$(ensure_grove) || exit 1

    # Default key binding (prefix + w)
    grove_key=$(get_tmux_option "@grove-key" "w")

    # Bind the key to launch grove in a popup
    tmux bind-key "$grove_key" display-popup -E -w 80% -h 80% -d "#{pane_current_path}" "$grove_cmd"
}

main
