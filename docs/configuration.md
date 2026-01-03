# Configuration

Grove uses a TOML configuration file located at:

- **macOS/Linux**: `~/.config/grove/config.toml`
- **Windows**: `%APPDATA%\grove\config.toml`

If no config file exists, grove creates a default `config.toml` file with sensible defaults.

## Full Configuration Reference

```toml
[general]
# Default base branch for new worktrees
default_base_branch = "main"

# Directory for worktrees (relative to repo root)
worktree_dir = ".worktrees"

# Default remote name (empty = auto-detect: single remote > "origin" > first)
remote = ""

[open]
# Command to run when opening a worktree (optional - auto-detected for tmux/zellij)
# Only set this to override the default behavior.
# Default for tmux: "tmux new-window -n {branch_short} -c {path}"
# Default for zellij: "zellij action new-tab --name {branch_short} --cwd {path}"
# Template variables: {path}, {branch}, {branch_short}, {repo}, {window_name}
# command = ""

# How to detect existing windows: "path", "name", or "none"
# "path" checks pane working directories (default, most reliable)
# "name" matches window/tab names
# "none" always creates new windows
detect_existing = "path"

# Whether to exit grove after opening a worktree
exit_after_open = true

# Whether to open the worktree after creating it
open_after_create = true

# Window name style: "short" (last segment) or "full" (entire branch)
window_name_style = "short"

# Auto-stash dirty worktree before switching to another
stash_on_switch = false

[delete]
# What to do with terminal window/tab when deleting a worktree: "auto", "ask", "never"
# Works with tmux (windows) and zellij (tabs)
close_window_action = "ask"

# What to do with the branch after deleting a worktree: "ask", "always", "never"
delete_branch_action = "ask"

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

# Default sort order: "default", "name", "name-desc", "dirty", "clean"
default_sort = "default"

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
sort = "o"
help = "?"
quit = "q,ctrl+c"
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

These examples show how to override the defaults for specific use cases.

### tmux - New session per worktree

Use sessions instead of windows:

```toml
[open]
command = "tmux new-session -d -s {branch_short} -c {path} && tmux switch-client -t {branch_short}"
```

### Zellij - New pane instead of tab

```toml
[open]
command = "zellij action new-pane --cwd {path}"
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
worktree_dir = "worktrees"
```

## Disable Safety Features

Not recommended, but if you want to skip confirmations:

```toml
[safety]
confirm_dirty = false
confirm_unmerged = false
require_typing_for_unique = false
```

## Layouts

Define layouts to automatically set up your workspace when opening a worktree. When layouts are defined, grove shows a selector letting you pick which layout to use.

```toml
[[layouts]]
name = "dev"
description = "Editor + terminal"
panes = [
  { command = "nvim" },                                # pane 0: editor (main)
  { split_from = 0, direction = "right", size = 35 }   # pane 1: terminal on right
]

[[layouts]]
name = "full"
description = "Editor + lazygit + system monitor"
panes = [
  { command = "nvim" },                                                            # pane 0: editor
  { split_from = 0, direction = "down", size = 20, command = "git status -sb" },   # pane 1: bottom bar (full width)
  { split_from = 0, direction = "right", size = 35, command = "lazygit" },         # pane 2: right of editor
  { split_from = 2, direction = "down", size = 50, command = "btop" }              # pane 3: below lazygit
]
# Result:
# ┌──────────────┬──────────┐
# │              │ lazygit  │
# │    nvim      ├──────────┤
# │              │  btop    │
# ├──────────────┴──────────┤
# │ git status -sb          │
# └─────────────────────────┘
```

Pane options:
- `command`: Command to run (supports `{path}`, `{branch}`, `{branch_short}`, `{repo}`)
- `split_from`: Which pane to split (0 = main, 1 = first split, etc.)
- `direction`: "right", "down", "left", "up"
- `size`: Percentage of the pane being split (1-99)

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
