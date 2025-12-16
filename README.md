# grove

A terminal UI for managing Git worktrees.

```
┌─ grove ─────────────────────────────────────────── ~/myproject ─┐
│                                                                  │
│  WORKTREES                                                       │
│  ────────────────────────────────────────────────────────────── │
│                                                                  │
│  › main              .               ✓ clean                    │
│    feat/auth         .worktrees/auth ✗ 3 modified   ↑2 ahead   │
│    feat/dashboard    .worktrees/dash ✓ clean        ✓ merged    │
│    fix/login         .worktrees/log  ⚠ 2 unique                 │
│                                                                  │
│  ────────────────────────────────────────────────────────────── │
│  [enter] open  [n]ew  [d]elete  [f]etch  [/]filter  [q]uit     │
└──────────────────────────────────────────────────────────────────┘
```

## Why grove?

Git worktrees are powerful but underused. Managing them with raw `git worktree` commands is tedious. Grove makes it easy:

- **See all worktrees** with their status at a glance
- **Create new worktrees** with branch selection and fuzzy search
- **Delete safely** with smart warnings about uncommitted changes and unmerged commits
- **Open anywhere** - configurable action for tmux, zellij, VS Code, or any command

## Installation

```bash
# Homebrew (coming soon)
brew install grove

# From source
go install github.com/henrilemoine/grove@latest
```

## Usage

```bash
# Open the TUI in current repo
grove

# Or in a specific directory
grove /path/to/repo
```

### Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Enter` | Open worktree |
| `n` | Create new worktree |
| `d` | Delete worktree |
| `f` | Fetch all |
| `/` | Filter/search |
| `?` | Help |
| `q` | Quit |

## Configuration

Grove looks for config at `~/.config/grove/config.toml`:

```toml
[open]
# Command to run when opening a worktree
# Variables: {path}, {branch}, {branch_short}, {repo}
command = "tmux new-window -n {branch_short} -c {path}"

[general]
default_base_branch = "main"
worktree_dir = ".worktrees"

[safety]
confirm_dirty = true
confirm_unmerged = true
require_typing_for_unique = true
```

### Example Configurations

**tmux (new window):**
```toml
[open]
command = "tmux new-window -n {branch_short} -c {path}"
```

**tmux (new session):**
```toml
[open]
command = "tmux new-session -d -s {branch_short} -c {path} && tmux switch -t {branch_short}"
```

**zellij:**
```toml
[open]
command = "zellij action new-pane --cwd {path}"
```

**VS Code:**
```toml
[open]
command = "code {path}"
```

## Safety Features

Grove prevents accidental data loss:

- **Uncommitted changes**: Warns before deleting dirty worktrees
- **Unmerged branches**: Shows merge status before deletion
- **Unique commits**: Detects commits that exist ONLY on this branch and requires typing "delete" to confirm

## Integration with tmux-worktree

If you use the [tmux-worktree](https://github.com/henrilemoine/tmux-worktree) plugin, grove can be launched from a keybinding:

```bash
# In your tmux.conf or via the plugin
bind-key w display-popup -E -w 80% -h 80% "grove"
```

## Development

See [SPEC.md](./SPEC.md) for the full specification and research notes.

```bash
# Build
go build -o grove ./cmd/grove

# Run
./grove
```

## License

MIT
