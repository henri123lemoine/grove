# Integrations

Grove works standalone but integrates well with terminal multiplexers and editors.

## tmux

Grove auto-detects tmux. See [integrations/tmux](../integrations/tmux/) for setup.

**Quick start with TPM:**
```tmux
set -g @plugin 'henri123lemoine/grove'
```

## Zellij

Grove auto-detects zellij. See [integrations/zellij](../integrations/zellij/) for setup.

## VS Code

```toml
# ~/.config/grove/config.toml
[open]
command = "code {path}"
```

## Shell Integration

Use `-p` to print the selected path instead of opening:

```bash
cd "$(grove -p)"
```

Or create a function:

```bash
# ~/.bashrc or ~/.zshrc
gw() {
    local path
    path=$(grove -p)
    [ -n "$path" ] && [ -d "$path" ] && cd "$path"
}
```
