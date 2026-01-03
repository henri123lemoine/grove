# Grove tmux Integration

Use grove as a popup worktree switcher in tmux.

## Quick Setup

Add to your `~/.tmux.conf`:

```tmux
# Open grove in a popup (prefix + g)
bind-key g display-popup -E -w 80% -h 80% "grove"
```

Grove will:
- Find existing windows by checking pane working directories
- Switch to them if found
- Create new windows with `tmux new-window -n {branch_short} -c {path}` if not

## Optional: TPM Plugin

If you prefer TPM, add to `~/.tmux.conf`:

```tmux
set -g @plugin 'henri123lemoine/grove'
```

Then press `prefix + I` to install. This binds `prefix + g` to open grove.

### Custom Key Binding

```tmux
set -g @grove-key "w"  # Use prefix + w instead
```

## Optional Configuration

Only configure if you want to change the defaults:

```toml
# ~/.config/grove/config.toml
[open]
# Use sessions instead of windows
command = "tmux new-session -d -s {branch_short} -c {path}; tmux switch-client -t {branch_short}"
```
