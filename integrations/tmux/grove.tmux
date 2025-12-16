#!/usr/bin/env bash
# grove.tmux - TPM-compatible plugin for grove integration

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default key binding (prefix + g)
default_key="g"

# Get user-configured key or use default
get_tmux_option() {
    local option="$1"
    local default_value="$2"
    local option_value=$(tmux show-option -gqv "$option")
    if [ -z "$option_value" ]; then
        echo "$default_value"
    else
        echo "$option_value"
    fi
}

grove_key=$(get_tmux_option "@grove-key" "$default_key")

# Bind the key to launch grove
tmux bind-key "$grove_key" display-popup -E -w 80% -h 80% "grove"

# Echo success message
tmux display-message "Grove plugin loaded. Press prefix + $grove_key to open."
