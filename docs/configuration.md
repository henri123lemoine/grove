# Configuration

Grove uses a TOML configuration file located at:

- **macOS/Linux**: `~/.config/grove/config.toml`
- **Windows**: `%APPDATA%\grove\config.toml`

If no config file exists, Grove uses sensible defaults.

## Full Configuration Reference

```toml
[general]
# Default base branch for new worktrees
default_base_branch = "main"

# Directory for worktrees (relative to repo root, or absolute path)
worktree_dir = ".worktrees"

# Default remote name (empty = auto-detect: single remote > "origin" > first)
remote = ""

[open]
# Command to run when opening a worktree
# Template variables: {path}, {branch}, {branch_short}, {repo}, {window_name}
command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"

# How to detect existing windows: "path", "name", or "none"
# "path" is more reliable (won't confuse feat/auth and fix/auth)
detect_existing = "path"

# Whether to exit grove after opening a worktree
exit_after_open = true

# Layout to apply after creating new window: "none", "dev", or "custom"
# "dev" creates a 70/30 horizontal split (works for tmux and zellij)
layout = "none"

# Custom layout command (only used if layout = "custom")
layout_command = ""

# Window name style: "short" (last segment) or "full" (entire branch)
window_name_style = "short"

# Auto-stash dirty worktree before switching to another
stash_on_switch = false

[worktree]
# File patterns to copy to new worktrees (e.g., ".env*")
copy_patterns = []

# File patterns to ignore when copying
copy_ignores = []

[safety]
# Confirm before deleting worktrees with uncommitted changes
confirm_dirty = true

# Confirm before deleting unmerged branches
confirm_unmerged = true

# Require typing "delete" for worktrees with unique commits
require_typing_for_unique = true

[ui]
# Show branch type indicators ([worktree], [local], [remote]) in create flow
show_branch_types = true

# Show last commit info in worktree list
show_commits = true

# Show upstream tracking status (ahead/behind)
show_upstream = true

# Color theme: auto, dark, light
theme = "auto"

[keys]
# All keybindings are configurable (comma-separated for multiple keys)
up = "up,k"
down = "down,j"
home = "home,g"
end = "end,G"
open = "enter"
new = "n"
delete = "d"
rename = "r"
filter = "/"
fetch = "f"
detail = "tab"
prune = "P"
stash = "s"
help = "?"
quit = "q"
```

## Template Variables

When configuring the `open.command`, you can use these template variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{path}` | Full path to the worktree | `/home/user/project/.worktrees/feature` |
| `{branch}` | Full branch name | `feature/auth` |
| `{branch_short}` | Branch name after last `/` | `auth` |
| `{repo}` | Repository name | `myproject` |
| `{window_name}` | Generated window name (based on `window_name_style`) | `auth` or `feature/auth` |

## Example Configurations

### tmux - Switch to existing window or create new

```toml
[open]
command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"
exit_after_open = true
```

### tmux - Always create new window

```toml
[open]
command = "tmux new-window -n {branch_short} -c {path}"
exit_after_open = true
```

### tmux - New session per worktree

```toml
[open]
command = "tmux new-session -d -s {branch_short} -c {path} && tmux switch-client -t {branch_short}"
exit_after_open = true
```

### Zellij - New pane

```toml
[open]
command = "zellij action new-pane --cwd {path}"
exit_after_open = true
```

### Zellij - New tab

```toml
[open]
command = "zellij action new-tab --cwd {path} --name {branch_short}"
exit_after_open = true
```

### VS Code

```toml
[open]
command = "code {path}"
exit_after_open = true
```

### Neovim in Alacritty

```toml
[open]
command = "alacritty --working-directory {path} -e nvim"
exit_after_open = true
```

### Print path only (for shell integration)

```toml
[open]
command = "echo {path}"
exit_after_open = true
```

You can then use this with shell integration:
```bash
cd "$(grove)"
```

### Kitty - New window

```toml
[open]
command = "kitty @ launch --type=window --cwd={path}"
exit_after_open = true
```

### WezTerm - New tab

```toml
[open]
command = "wezterm cli spawn --cwd {path}"
exit_after_open = true
```

## Custom Worktree Directory

By default, worktrees are created in `.worktrees/` relative to the repository root. You can change this:

```toml
[general]
# Relative to repo root
worktree_dir = "worktrees"

# Or use absolute path
worktree_dir = "/tmp/worktrees"
```

## Disable Safety Features

Not recommended, but if you want to skip confirmations:

```toml
[safety]
confirm_dirty = false
confirm_unmerged = false
require_typing_for_unique = false
```

## Layout Presets

The `layout = "dev"` preset creates a split window for development:

```toml
[open]
command = "tmux new-window -n {branch_short} -c {path}"
layout = "dev"  # Creates 70/30 horizontal split
```

For custom layouts:

```toml
[open]
layout = "custom"
layout_command = "tmux split-window -h -p 30 -c {path}"
```

## Auto-Stash on Switch

Automatically stash uncommitted changes when switching worktrees:

```toml
[open]
stash_on_switch = true
```

## Custom Keybindings

Change any keybinding (comma-separated for multiple keys):

```toml
[keys]
# Vim-style navigation
up = "k"
down = "j"

# Alternative delete key
delete = "x,d"

# Disable a keybinding by setting it to empty
prune = ""
```
