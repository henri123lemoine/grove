# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
make build       # Build the binary to ./grove
make run         # Run directly with go run
make install     # Install to $GOPATH/bin
make test        # Run all tests
make lint        # Run golangci-lint
make fmt         # Format code
```

Run a single test:
```bash
go test -v -run TestName ./internal/git/
```

## Architecture Overview

Grove is a terminal UI for managing Git worktrees, built with the Bubble Tea framework.

### Core Components

**Entry Point**: `cmd/grove/main.go` - Loads config, detects git repo, initializes and runs the Bubble Tea program.

**Application Layer** (`internal/app/`):
- `app.go` - Main Bubble Tea model with state machine (StateList, StateCreate, StateDelete, StateFilter, etc.). Handles all user input via key handlers and coordinates between git operations and UI rendering.
- `messages.go` - Bubble Tea message types for async operations (WorktreesLoadedMsg, SafetyCheckedMsg, etc.)
- `keys.go` - Keybinding definitions

**Git Operations** (`internal/git/`):
- `repo.go` - Repository detection, caches repo info including MainWorktreeRoot (where worktrees are created) and GitDir
- `worktree.go` - List/Create/Remove worktrees with status enrichment (dirty files, ahead/behind, merge status)
- `safety.go` - Delete safety analysis with three levels: Safe, Warning (uncommitted changes), Danger (unique commits that would be lost)
- `status.go` - Git status operations (dirty status, upstream tracking)
- `branch.go` - Branch operations

**UI Rendering** (`internal/ui/`):
- `render.go` - Pure rendering functions for each state (renderList, renderCreate, renderDelete, etc.)
- `styles.go` - Lipgloss style definitions

**Configuration** (`internal/config/config.go`):
- TOML config at `~/.config/grove/config.toml`
- Open command template with variables: `{path}`, `{branch}`, `{branch_short}`, `{repo}`

### Key Design Patterns

- State machine pattern for UI flows (StateList -> StateCreate -> StateCreateSelectBase)
- Async git operations return Bubble Tea messages (Cmd functions return Msg)
- All rendering is delegated to `ui.Render()` with a `RenderParams` struct
- Worktree paths are generated relative to MainWorktreeRoot for consistency across worktrees
- Safety checks run asynchronously before showing delete confirmation

## Workflow for agents

Steps:
1. Read a todo in `docs/todo.md`
2. Complete it and make sure the tests pass and that it is the minimal implementation of the requested feature
3. Ask the user to validate that it behaves as expected
4. If the user says that it is validly made, check the todo and commit the changes
5. Move back to step 1
