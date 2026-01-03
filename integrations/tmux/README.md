# Grove tmux Integration

Use grove as a popup worktree switcher in tmux.

## TPM (Recommended)

Add to `~/.tmux.conf`:

```tmux
set -g @plugin 'henri123lemoine/grove'
```

Press `prefix + I` to install. Binds `prefix + w` to open grove.

### Custom Key Binding

```tmux
set -g @grove-key "g"  # Use prefix + g instead
```

## Without TPM

If you don't use TPM, install grove separately then add a keybinding:

```bash
brew install henri123lemoine/tap/grove
```

```tmux
# ~/.tmux.conf
bind-key w display-popup -E -w 80% -h 80% "grove"
```

## How It Works

Grove auto-detects tmux and:
- Finds existing windows by checking pane working directories
- Switches to them if found
- Creates new windows with `tmux new-window -n {branch_short} -c {path}` if not

## Optional Configuration

Override defaults in `~/.config/grove/config.toml`:

```toml
[open]
# Use sessions instead of windows
command = "tmux new-session -d -s {branch_short} -c {path}; tmux switch-client -t {branch_short}"
```
