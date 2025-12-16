# Grove tmux Integration

Use grove as a popup worktree switcher in tmux.

## Installation

### Using TPM (Tmux Plugin Manager)

Add to your `~/.tmux.conf`:

```tmux
set -g @plugin 'henrilemoine/grove'
```

Then press `prefix + I` to install.

### Manual Installation

Clone the repo and source the plugin:

```bash
git clone https://github.com/henrilemoine/grove ~/.tmux/plugins/grove
```

Add to `~/.tmux.conf`:

```tmux
run-shell ~/.tmux/plugins/grove/integrations/tmux/grove.tmux
```

## Usage

Press `prefix + g` to open grove in a popup window.

## Configuration

### Custom Key Binding

Change the trigger key in `~/.tmux.conf`:

```tmux
set -g @grove-key "w"  # Use prefix + w instead
```

### Grove Config for tmux

Create `~/.config/grove/config.toml`:

```toml
[open]
# Create new tmux window with branch name
command = "tmux new-window -n {branch_short} -c {path}"

# Or switch to existing window if it exists
# command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"

# Exit grove after opening (recommended for popup)
exit_after_open = true

# Detect existing windows by path
detect_existing = "path"

# Apply dev layout (split 50/50)
# layout = "dev"
```

### Advanced: Custom Layout

Apply a custom layout after creating a new window:

```toml
[open]
command = "tmux new-window -n {branch_short} -c {path}"
layout = "custom"
layout_command = "tmux split-window -h -p 30 -c {path} && tmux select-pane -L"
```

## Shell Integration

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
# Quick worktree switch with cd
gw() {
    local path
    path=$(grove -p)
    if [ -n "$path" ]; then
        cd "$path"
    fi
}

# Bind to Ctrl+G
bind '"\C-g": "gw\n"'  # bash
# bindkey -s '^g' 'gw\n'  # zsh
```

## Template Variables

Available in open command:

| Variable | Description | Example |
|----------|-------------|---------|
| `{path}` | Full worktree path | `/home/user/project/.worktrees/feature/auth` |
| `{branch}` | Full branch name | `feature/auth` |
| `{branch_short}` | Short branch name | `auth` |
| `{repo}` | Repository name | `my-project` |
| `{window_name}` | Window name (respects style config) | `auth` or `feature/auth` |
