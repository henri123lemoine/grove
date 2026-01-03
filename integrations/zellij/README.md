# Grove Zellij Integration

Use grove as a worktree switcher in Zellij.

**No configuration needed.** Grove auto-detects zellij and handles tab creation automatically.

## Quick Setup

Add to your Zellij config (`~/.config/zellij/config.kdl`):

```kdl
keybinds {
    normal {
        bind "Alt g" {
            Run "grove" {
                close_on_exit true
            }
        }
    }
}
```

That's it! Grove will:
- Create new tabs with `zellij action new-tab --name {branch_short} --cwd {path}`
- Detect existing tabs by name and switch to them

## Optional Configuration

Only configure if you want to change the defaults:

```toml
# ~/.config/grove/config.toml
[open]
# Use panes instead of tabs
command = "zellij action new-pane --cwd {path}"
```
