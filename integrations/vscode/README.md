# Grove VS Code Extension

Manage Git worktrees directly from VS Code.

## Features

- **Sidebar View**: See all worktrees in the SCM panel
- **Quick Switch**: `Cmd+Shift+G` (Mac) / `Ctrl+Shift+G` (Windows/Linux) to switch worktrees
- **Create Worktrees**: Create new worktrees from the command palette
- **Delete Worktrees**: Remove worktrees with safety checks
- **Terminal Integration**: Open the grove TUI in a terminal

## Commands

| Command | Description |
|---------|-------------|
| `Grove: Open Worktree` | Switch to a different worktree |
| `Grove: Create Worktree` | Create a new worktree |
| `Grove: Delete Worktree` | Delete a worktree |
| `Grove: List Worktrees` | Quick pick worktree list |
| `Grove: Open Grove in Terminal` | Launch grove TUI |

## Installation

### From VSIX

1. Download the `.vsix` file from releases
2. In VS Code, go to Extensions (`Cmd+Shift+X`)
3. Click `...` → `Install from VSIX...`
4. Select the downloaded file

### From Source

```bash
cd integrations/vscode
npm install
npm run compile
# Then press F5 in VS Code to launch Extension Development Host
```

## Usage

### Sidebar

The extension adds a "Worktrees" panel to the SCM sidebar. Click any worktree to open it in a new window.

### Quick Switch

Press `Cmd+Shift+G` (Mac) or `Ctrl+Shift+G` (Windows/Linux) to open the quick pick worktree switcher.

### Create Worktree

1. Open Command Palette (`Cmd+Shift+P`)
2. Type "Grove: Create Worktree"
3. Enter the branch name
4. The worktree will be created and you'll be prompted to open it

### Terminal Integration

For the full grove experience, use `Grove: Open Grove in Terminal` to launch the TUI interface.

## Requirements

- Git 2.5+ (for worktree support)
- grove CLI (optional, for terminal integration)

## Extension Settings

This extension does not add any VS Code settings. Configure grove using `~/.config/grove/config.toml`.
