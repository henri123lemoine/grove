# Grove Zellij Integration

Use grove as a worktree switcher in Zellij.

## Configuration

### Grove Config for Zellij

Create `~/.config/grove/config.toml`:

```toml
[open]
# Create new Zellij tab with branch name
command = "zellij action new-tab --name {branch_short} --cwd {path}"

# Exit grove after opening
exit_after_open = true
```

### Zellij Keybinding

Add to your `~/.config/zellij/config.kdl`:

```kdl
keybinds {
    normal {
        // Open grove with Ctrl+g
        bind "Ctrl g" {
            Run "grove" {
                close_on_exit true
            }
        }
    }
}
```

Or use a floating pane:

```kdl
keybinds {
    normal {
        bind "Ctrl g" {
            LaunchOrFocusPlugin "zellij:strider" {
                floating true
            }
            // Alternative: run grove directly
            // Run "grove" { floating true; close_on_exit true }
        }
    }
}
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
```

## Template Variables

Available in open command:

| Variable | Description | Example |
|----------|-------------|---------|
| `{path}` | Full worktree path | `/home/user/project/.worktrees/feature/auth` |
| `{branch}` | Full branch name | `feature/auth` |
| `{branch_short}` | Short branch name | `auth` |
| `{repo}` | Repository name | `my-project` |
| `{window_name}` | Tab name (respects style config) | `auth` or `feature/auth` |

## Example Config

```toml
[general]
default_base_branch = "main"
worktree_dir = ".worktrees"

[open]
command = "zellij action new-tab --name {branch_short} --cwd {path}"
exit_after_open = true
window_name_style = "short"
```
