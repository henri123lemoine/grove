# grove

A terminal UI for managing Git worktrees.

<!-- TODO: Add screenshot -->
![grove screenshot](./docs/screenshot.png)

## Features

- **See all worktrees** with their status at a glance
- **Create new worktrees** with branch selection
- **Delete safely** with smart warnings about uncommitted changes and unique commits
- **Open anywhere** - configurable action for tmux, zellij, VS Code, or any command
- **Switch to existing** - reuses tmux windows instead of creating duplicates

## Installation

```bash
# From source
go install github.com/henrilemoine/grove/cmd/grove@latest

# Or build locally
git clone https://github.com/henrilemoine/grove
cd grove
go build -o grove ./cmd/grove
```

## Usage

```bash
grove
```

### Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Enter` | Open worktree |
| `n` | New worktree |
| `d` | Delete worktree |
| `f` | Fetch all |
| `/` | Filter |
| `q` | Quit |

## Configuration

Config location: `~/.config/grove/config.toml`

```toml
[open]
# Command to run when opening a worktree
# Variables: {path}, {branch}, {branch_short}, {repo}
command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"
exit_after_open = true

[general]
default_base_branch = "main"
worktree_dir = ".worktrees"

[safety]
confirm_dirty = true
confirm_unmerged = true
require_typing_for_unique = true
```

### Open Command Examples

**tmux (switch or create window):**
```toml
command = "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}"
```

**zellij:**
```toml
command = "zellij action new-pane --cwd {path}"
```

**VS Code:**
```toml
command = "code {path}"
```

## Safety

Grove prevents accidental data loss:

- **Uncommitted changes**: Warns before deleting dirty worktrees
- **Unmerged branches**: Shows merge status
- **Unique commits**: Commits that exist ONLY on this branch require typing "delete" to confirm

## Development

```bash
make build
make run
make test
```

See [SPEC.md](./SPEC.md) for the full specification.

## License

MIT
