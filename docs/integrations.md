# Integrations

Grove is designed to work with any terminal multiplexer, editor, or workflow. This guide covers common setups.

## tmux

Grove auto-detects tmux and:
- Finds existing windows by checking pane working directories
- Switches to them if found
- Creates new windows with `tmux new-window -n {branch_short} -c {path}` if not

### Popup Integration

Add to your `~/.tmux.conf`:

```bash
# Open grove in a popup (prefix + g)
bind-key g display-popup -E -w 80% -h 80% "grove"
```

### Optional: Use sessions instead of windows

```toml
# ~/.config/grove/config.toml
[open]
command = "tmux new-session -d -s {branch_short} -c {path} 2>/dev/null; tmux switch-client -t {branch_short}"
```

## Zellij

Grove auto-detects zellij and:
- Creates new tabs with `zellij action new-tab --name {branch_short} --cwd {path}`
- Detects existing tabs by name

### Keybinding

Add to your Zellij config (`~/.config/zellij/config.kdl`):

```kdl
keybinds {
    normal {
        bind "Alt g" { Run "grove" }
    }
}
```

### Optional: Use panes instead of tabs

```toml
# ~/.config/grove/config.toml
[open]
command = "zellij action new-pane --cwd {path}"
```

## VS Code

```toml
# ~/.config/grove/config.toml
[open]
command = "code {path}"
exit_after_open = true
```

## Shell Integration

Use the `-p` flag to print the selected worktree path:

```bash
cd "$(grove -p)"
```

Or create a shell function:

```bash
# ~/.bashrc or ~/.zshrc
gw() {
    local path
    path=$(grove -p)
    if [ -n "$path" ] && [ -d "$path" ]; then
        cd "$path"
    fi
}
```

Now `gw` opens grove, and when you select a worktree, it changes to that directory.
