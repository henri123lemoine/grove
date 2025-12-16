# Grove - Git Worktree Manager

A terminal UI for managing Git worktrees with optional terminal multiplexer integration.

## Vision

Grove is a standalone TUI that makes git worktrees as easy to manage as branches in lazygit. It's designed to:

1. **Work anywhere** - Pure terminal app, no dependencies on tmux/zellij
2. **Integrate with anything** - Configurable "open" action for any multiplexer or editor
3. **Be safe** - Smart deletion with merge status and unique commit detection
4. **Be fast** - Single Go binary, instant startup

---

## Prior Art & Research

### Existing Tools Analyzed

We conducted deep analysis of 5 existing worktree management tools. Here's what we learned:

---

### 1. lazygit (Worktree Support)

**Repository**: https://github.com/jesseduffield/lazygit
**Stars**: ~54,000
**Language**: Go (bubbletea)

**What it does well:**
- Full TUI with worktree view
- Create/delete/switch worktrees
- Integrated with full git workflow
- Excellent UX patterns

**Limitations:**
- Worktrees are a secondary feature, not the focus
- Issues with bare repository workflows (exits instead of showing worktree selector)
- No multiplexer integration (obviously - it's a git tool, not a launcher)
- Can't configure "open in new terminal/window" behavior

**Key insight**: Lazygit proves the TUI model works for worktrees, but it's solving a different problem (git management) not worktree-as-workspace management.

---

### 2. twt (Tmux Worktree)

**Repository**: https://github.com/todaatsushi/twt
**Stars**: 8
**Language**: Go
**Status**: Alpha (v0.0.14)

**What it does well:**
- Tmux session management per worktree
- Post-execution scripts (`common/scripts/go/post.sh`)
- Common files directory for shared assets
- Clean Go architecture with Cobra CLI

**Architecture:**
```
cmd/
├── go.go      # Create/switch to worktree session
├── rm.go      # Remove worktree and session
├── common.go  # Manage shared files
└── check.go   # Validate environment

internal/
├── tmux/      # Session management (new-session, switch, kill, send-keys)
├── git/       # Branch and worktree operations
└── checks/    # Environment validation
```

**Limitations:**
- **CLI only** - No TUI, relies on external fzf via shell bindings
- **No worktree listing** - Must use raw `git worktree list`
- **macOS only** - Stated in README
- **Sessions, not windows** - Creates full tmux sessions, not windows
- **No interactive selection** - Requires branch name as argument
- **No confirmation prompts** - Deletes without asking

**Code quality**: Medium. Well-organized but limited testing, no CI/CD, basic error handling.

**Key insight**: Validates the tmux+worktree concept has demand, but execution is too limited. The "CLI that relies on external fzf" pattern is clunky compared to a proper TUI.

---

### 3. branchlet

**Repository**: https://github.com/raghavpillai/branchlet
**Stars**: ~50
**Language**: TypeScript (Bun + Ink)

**What it does well:**
- **Excellent TUI** - Built with Ink (React for terminals)
- Full interactive navigation (arrows, vim keys, number shortcuts)
- **Safe deletion** - Shows ahead/behind, prevents deleting current/default branch
- **File copying** - Glob patterns for copying .env, configs to new worktrees
- **Post-create hooks** - Run commands after worktree creation
- Template variables: `$BASE_PATH`, `$WORKTREE_PATH`, `$BRANCH_NAME`, `$SOURCE_BRANCH`

**Configuration** (`.branchlet.json`):
```json
{
  "worktreeCopyPatterns": [".env*", ".vscode/**"],
  "worktreeCopyIgnores": ["**/node_modules/**"],
  "worktreePathTemplate": "$BASE_PATH.worktree",
  "postCreateCmd": ["npm install"],
  "terminalCommand": "code $WORKTREE_PATH"
}
```

**Architecture:**
```
src/
├── components/    # React/Ink UI components
├── panels/        # Screens (main menu, create, delete, list, settings)
├── services/      # Business logic (worktree, git, config, files)
└── utils/         # Git commands, error handling
```

**Limitations:**
- **No worktree switching** - Can open in editor but no terminal/multiplexer integration
- **Settings are view-only** - Must edit JSON manually
- **No fuzzy search** - Arrow navigation only
- TypeScript/Bun dependency (not a single binary)

**Code quality**: High. Modern architecture, comprehensive testing (3k+ lines), proper error handling.

**Key insight**: Branchlet shows what a polished worktree TUI looks like. The file copying and template variable system is worth stealing. But the lack of switching is a dealbreaker for a "worktree manager" - you can create and delete but not actually use them efficiently.

---

### 4. git-worktree-manager (gitwm)

**Repository**: https://github.com/JoshYG-TheKey/git-worktree-manager
**Stars**: ~20
**Language**: Python (Rich + Click)

**What it does well:**
- **Beautiful CLI output** - Rich tables, panels, progress indicators
- Diff summaries between branches
- Branch status tracking (ahead/behind)
- Performance caching for git operations
- TOML configuration with environment variable overrides

**Configuration** (`~/.config/git-worktree-manager/config.toml`):
```toml
[worktree]
default_path = "~/worktrees"
auto_cleanup = true

[ui]
theme = "dark"
show_progress = true

[performance]
cache_timeout = 300
```

**Limitations:**
- **NO DELETE COMMAND** - Ironically named "manager" but can't delete worktrees
- **No switching** - Shows paths but doesn't help navigate
- **CLI, not TUI** - Rich prompts but not interactive selection
- Python dependency

**Code quality**: Very high. Excellent typing, testing (7k+ lines), error handling, caching layer.

**Key insight**: Shows what's possible with Rich for beautiful terminal output. The caching and config patterns are solid. But critically incomplete - no deletion is bizarre.

---

### 5. worktree-cli (wt)

**Repository**: https://github.com/johnlindquist/worktree-cli
**Stars**: ~200
**Language**: TypeScript
**Maturity**: Production-ready

**What it does well:**

**Commands:**
- `wt new` - Create worktree from branch
- `wt setup` - Create + run setup scripts
- `wt pr` - Create worktree from GitHub PR or GitLab MR
- `wt open` - Open existing worktree in editor
- `wt list` - List all worktrees
- `wt remove` - Remove worktree (single)
- `wt purge` - Multi-select removal
- `wt merge` - Merge worktree branch into current
- `wt extract` - Extract current branch to worktree
- `wt config` - Manage settings

**Safety - The Gold Standard:**

1. **Atomic Operations with Rollback**
   ```typescript
   // AtomicWorktreeOperation class tracks all actions
   // Automatic rollback on failure:
   // - Removes worktree
   // - Deletes directory
   // - Cleans up branches
   // Rollback runs in reverse order (LIFO)
   ```

2. **Stash Management**
   - Uses hash-based stash (`git stash create`) not stack
   - Prevents race conditions with multiple terminals
   - Auto-restores in `finally` block

3. **SIGINT Handling**
   - Registers cleanup for Ctrl+C
   - Restores stashed changes even if interrupted

4. **Merge Safety**
   - Checks uncommitted changes
   - `--auto-commit` required for dirty commits
   - `--remove` required for destructive cleanup

**PR Integration - Impressive:**
```typescript
// Fetches PR without switching context!
// Uses: git fetch refs/pull/${prNumber}/head:branch
// No dangerous checkout operations
// Supports both GitHub (gh) and GitLab (glab)
// Falls back to REST API if CLI not installed
```

**Path Resolution:**
```typescript
// 1. --path flag (explicit)
// 2. Global worktreepath config (namespaced by repo)
// 3. Sibling directory fallback: /path/repo → /path/repo-branch
```

**Limitations:**
- **No real worktree switching** - Only opens in editor
- **No multiplexer integration** - Editor-focused
- **TypeScript, not binary** - Requires Node.js
- No TUI list view (uses prompts)

**Code quality**: Excellent. Production-ready, comprehensive tests, proper cleanup patterns.

**Key insight**: worktree-cli is the most feature-complete tool. The atomic operation pattern, stash handling, and PR integration are worth studying carefully. If we're building a Go tool, we should replicate these safety patterns.

---

## Feature Specification

### Core Features (MVP)

#### 1. List View

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

**Status indicators:**
- `✓ clean` - No uncommitted changes
- `✗ N modified` - Has uncommitted changes (N files)
- `↑N ahead` - Commits ahead of upstream
- `↓N behind` - Commits behind upstream
- `✓ merged` - Branch is merged into default branch
- `⚠ N unique` - Has N commits that exist ONLY on this branch (not merged anywhere, not pushed)

#### 2. Create Worktree

Flow:
1. Select or type branch name (fuzzy search existing, or create new)
2. If new branch: select base branch
3. Confirm path (default: `.worktrees/{branch-name}`)
4. Create and optionally open

#### 3. Delete Worktree

**Tiered safety based on risk:**

**SAFE** (just confirm):
- Clean working directory
- Branch merged to default
- No unique commits

**WARNING** (show details, confirm):
- Has uncommitted changes (show file count)
- Has unpushed commits (show count)
- Branch not merged (but pushed to remote)

**DANGER** (require typing "delete"):
- Has commits that exist NOWHERE else
- These commits would be permanently lost

```
┌─ Delete fix/login? ─────────────────────────────────────────────┐
│                                                                  │
│  ⚠ WARNING: This worktree has commits that exist NOWHERE ELSE  │
│                                                                  │
│  These 2 commits will be PERMANENTLY LOST:                      │
│                                                                  │
│    a1b2c3d Fix login redirect loop                              │
│    e4f5g6h Add remember me checkbox                             │
│                                                                  │
│  The branch 'fix/login' has not been pushed or merged.          │
│                                                                  │
│  Type "delete" to confirm: _                                    │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

#### 4. Open Worktree

Configurable action. Examples:

```toml
# tmux - new window
open_command = "tmux new-window -n {branch} -c {path}"

# tmux - new session
open_command = "tmux new-session -d -s {branch} -c {path} && tmux switch -t {branch}"

# zellij - new pane
open_command = "zellij action new-pane --cwd {path}"

# VS Code
open_command = "code {path}"

# Just print (for shell integration)
open_command = "echo {path}"
```

Template variables:
- `{path}` - Full path to worktree
- `{branch}` - Branch name
- `{branch_short}` - Branch name after last `/` (e.g., `auth` from `feat/auth`)
- `{repo}` - Repository name

### Additional Features

#### Fuzzy Search/Filter

Press `/` to enter filter mode:
```
┌─ grove ─────────────────────────────────────────────────────────┐
│  Filter: auth_                                                   │
│  ────────────────────────────────────────────────────────────── │
│  › feat/auth         .worktrees/auth ✗ 3 modified              │
│    fix/auth-redirect .worktrees/redi ✓ clean                   │
└──────────────────────────────────────────────────────────────────┘
```

#### Fetch All

Press `f` to fetch all remotes for all worktrees (shows progress).

#### Detail Panel

When hovering over a worktree, show expanded details:
```
┌─ feat/auth ─────────────────────────────────────────────────────┐
│ Path:      /Users/you/project/.worktrees/auth                   │
│ Branch:    feat/auth                                            │
│ Upstream:  origin/feat/auth (2 ahead, 0 behind)                │
│ Status:    3 uncommitted files                                  │
│ Merged:    No                                                   │
│ Last:      a1b2c3d "Add OAuth flow" - 2 hours ago              │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

Location: `~/.config/grove/config.toml`

```toml
[general]
# Default base branch for new worktrees
default_base_branch = "main"

# Where to create worktrees (relative to repo root, or absolute)
worktree_dir = ".worktrees"

[open]
# Command to run when opening a worktree
# Use {path}, {branch}, {branch_short}, {repo}
command = "echo {path}"

# Whether to exit grove after opening
exit_after_open = true

[safety]
# Confirm before deleting worktrees with uncommitted changes
confirm_dirty = true

# Confirm before deleting unmerged branches
confirm_unmerged = true

# Require typing "delete" for worktrees with unique commits
require_typing_for_unique = true

[ui]
# Show commit info in detail panel
show_commits = true

# Show upstream tracking status
show_upstream = true

# Color theme
theme = "auto"  # auto, dark, light
```

---

## Technical Architecture

### Project Structure

```
grove/
├── cmd/
│   └── grove/
│       └── main.go           # Entry point
├── internal/
│   ├── app/
│   │   ├── app.go           # Main application state
│   │   ├── keys.go          # Keybinding definitions
│   │   └── update.go        # Message handling
│   ├── ui/
│   │   ├── list.go          # Worktree list view
│   │   ├── create.go        # Create worktree flow
│   │   ├── delete.go        # Delete confirmation
│   │   ├── detail.go        # Detail panel
│   │   ├── filter.go        # Fuzzy filter
│   │   └── styles.go        # Lipgloss styles
│   ├── git/
│   │   ├── worktree.go      # Worktree operations
│   │   ├── branch.go        # Branch operations
│   │   ├── status.go        # Status checking
│   │   └── safety.go        # Safety checks (unique commits, etc.)
│   ├── config/
│   │   └── config.go        # Configuration loading
│   └── exec/
│       └── open.go          # Execute open command
├── go.mod
├── go.sum
├── README.md
├── SPEC.md                   # This file
└── Makefile
```

### Dependencies

```go
require (
    github.com/charmbracelet/bubbletea  // TUI framework
    github.com/charmbracelet/lipgloss   // Styling
    github.com/charmbracelet/bubbles    // Components (list, textinput, etc.)
    github.com/pelletier/go-toml/v2     // Config parsing
    github.com/sahilm/fuzzy             // Fuzzy matching
)
```

### Key Patterns to Implement

#### 1. Atomic Operations (from worktree-cli)

```go
type AtomicOperation struct {
    rollbackActions []func() error
    committed       bool
}

func (op *AtomicOperation) AddRollback(action func() error) {
    op.rollbackActions = append(op.rollbackActions, action)
}

func (op *AtomicOperation) Rollback() error {
    if op.committed {
        return nil
    }
    // Execute in reverse order
    for i := len(op.rollbackActions) - 1; i >= 0; i-- {
        if err := op.rollbackActions[i](); err != nil {
            // Log but continue
        }
    }
    return nil
}

func (op *AtomicOperation) Commit() {
    op.committed = true
}
```

#### 2. Unique Commits Detection

```bash
# Commits on branch that are not on any remote
git log {branch} --not --remotes --oneline

# If this returns nothing, all commits are "safe" (exist elsewhere)
```

#### 3. Merge Status Check

```bash
# Check if branch is merged into default
git branch --merged {default_branch} | grep -q {branch}
```

#### 4. Safe Stash Pattern (from worktree-cli)

```go
// Use hash-based stash, not stack-based
// This prevents race conditions with multiple terminals

func stashChanges() (string, error) {
    // git stash create returns the stash hash
    hash, err := exec.Command("git", "stash", "create").Output()
    if err != nil {
        return "", err
    }
    if len(hash) == 0 {
        return "", nil // Nothing to stash
    }
    // Store the stash
    exec.Command("git", "stash", "store", "-m", "grove-stash", string(hash)).Run()
    return string(hash), nil
}

func restoreStash(hash string) error {
    if hash == "" {
        return nil
    }
    return exec.Command("git", "stash", "apply", hash).Run()
}
```

---

## Implementation Phases

### Phase 1: Core TUI (MVP)

**Goal**: Basic list, create, delete, open functionality

- [ ] Project scaffolding with bubbletea
- [ ] List view showing all worktrees
- [ ] Basic status (clean/dirty)
- [ ] Create worktree flow
- [ ] Delete with basic confirmation
- [ ] Configurable open command
- [ ] Config file loading

### Phase 2: Safety Features

**Goal**: Smart deletion with risk assessment

- [ ] Unique commits detection
- [ ] Merge status checking
- [ ] Upstream tracking status
- [ ] Tiered confirmation system
- [ ] Atomic operations with rollback

### Phase 3: Polish

**Goal**: Full-featured, pleasant to use

- [ ] Fuzzy search/filter
- [ ] Detail panel
- [ ] Fetch all worktrees
- [ ] Help screen
- [ ] Keyboard shortcut hints
- [ ] Color theme support

### Phase 4: Integration

**Goal**: Ready for real use

- [ ] Update tmux-worktree plugin to use grove
- [ ] Homebrew formula
- [ ] Installation docs
- [ ] Example configs (tmux, zellij, vscode)

---

## Integration with tmux-worktree

After grove is built, the tmux-worktree plugin becomes a thin wrapper:

```bash
# worktree.tmux

# Bind prefix+w to open grove in a popup
tmux bind-key w display-popup -E -w 80% -h 80% "grove"
```

With config:
```toml
# ~/.config/grove/config.toml
[open]
command = "tmux new-window -n {branch_short} -c {path}"
exit_after_open = true
```

The plugin could also ship a default config that gets installed on setup.

---

## Open Questions

1. **Bare repo support**: Should we handle bare repositories specially? (lazygit struggles with this)

2. **File copying**: Should we add branchlet-style file copying for new worktrees? Or keep it simple?

3. **PR integration**: Worth adding GitHub/GitLab PR support like worktree-cli? Or out of scope?

4. **Session vs Window**: For tmux, should the default be new window or new session?

5. **Multiple repos**: Should grove work across multiple repos, or always be repo-scoped?

---

## References

- [lazygit](https://github.com/jesseduffield/lazygit) - TUI patterns, bubbletea usage
- [twt](https://github.com/todaatsushi/twt) - Tmux session integration patterns
- [branchlet](https://github.com/raghavpillai/branchlet) - TUI design, file copying
- [worktree-cli](https://github.com/johnlindquist/worktree-cli) - Safety patterns, atomic operations
- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework docs
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling docs
