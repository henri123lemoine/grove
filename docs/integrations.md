# Integrations

Grove is designed to work with any terminal multiplexer, editor, or workflow. This guide covers common setups.

## tmux

### Popup Integration

Add to your `~/.tmux.conf`:

```bash
# Open grove in a popup (prefix + w)
bind-key w display-popup -E -w 80% -h 80% "grove"
```

### Configuration

```toml
# ~/.config/grove/config.toml
[open]
command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"
exit_after_open = true
```

This will:
1. Try to switch to an existing window named after the branch
2. If no window exists, create a new one
3. Exit grove after opening

### With tmux-sessionizer

If you use tmux-sessionizer or similar tools, you might prefer:

```toml
[open]
command = "tmux new-session -d -s {branch_short} -c {path} 2>/dev/null; tmux switch-client -t {branch_short}"
```

## Zellij

### Keybinding

Add to your Zellij config (`~/.config/zellij/config.kdl`):

```kdl
keybinds {
    normal {
        bind "Alt w" { Run "grove" }
    }
}
```

### Configuration

```toml
# ~/.config/grove/config.toml
[open]
command = "zellij action new-pane --cwd {path}"
exit_after_open = true
```

Or to open in a new tab:

```toml
[open]
command = "zellij action new-tab --cwd {path} --name {branch_short}"
```

## VS Code

### From Terminal

```toml
# ~/.config/grove/config.toml
[open]
command = "code {path}"
exit_after_open = true
```

### From VS Code Integrated Terminal

Grove works in VS Code's integrated terminal. When you open a worktree, it will open a new VS Code window for that path.

## Neovim

### With External Terminal

```toml
# Alacritty
[open]
command = "alacritty --working-directory {path} -e nvim"

# Kitty
[open]
command = "kitty @ launch --type=window --cwd={path} nvim"

# WezTerm
[open]
command = "wezterm cli spawn --cwd {path} -- nvim"
```

### With tmux

Most Neovim users run inside tmux. Use the tmux configuration above.

## Shell Integration

### cd to worktree

If you just want to change directory:

```toml
[open]
command = "echo {path}"
exit_after_open = true
```

Then create a shell function:

```bash
# ~/.bashrc or ~/.zshrc
gw() {
    local path
    path=$(grove 2>/dev/null)
    if [ -n "$path" ] && [ -d "$path" ]; then
        cd "$path"
    fi
}
```

Now `gw` opens grove, and when you select a worktree, it changes to that directory.

### Fuzzy finder integration

You can combine grove with fzf for a non-TUI workflow:

```bash
# List worktrees and select with fzf
git worktree list | fzf | awk '{print $1}' | xargs -I {} code {}
```

But grove's built-in TUI is usually more convenient.

## Bare Repository Workflow

Grove fully supports bare repositories. Create your repo as bare:

```bash
git clone --bare git@github.com:user/repo.git repo
cd repo
```

Then use grove normally. Worktrees will be created at the configured location (default: `.worktrees/`).

## CI/Automation

Grove is interactive and requires a TTY. For scripts, use git directly:

```bash
# Create worktree
git worktree add .worktrees/feature feature-branch

# Remove worktree
git worktree remove .worktrees/feature
```

## Multiple Monitors

If you work with multiple monitors and want each worktree on a different screen:

### macOS with yabai

```toml
[open]
command = "alacritty --working-directory {path} & sleep 0.5 && yabai -m window --display 2"
```

### Linux with i3/sway

```toml
[open]
command = "i3-msg 'workspace 2; exec alacritty --working-directory {path}'"
```

## Troubleshooting

### Command not found

Make sure grove is in your PATH:
```bash
which grove
```

### Worktrees not showing

Check that you're in a git repository:
```bash
git worktree list
```

### tmux window not switching

The window name must match exactly. Check with:
```bash
tmux list-windows
```

If using special characters in branch names, they may be sanitized. Use `{branch_short}` for simpler names.
