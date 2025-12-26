# Grove - Design Specification

A terminal UI for managing Git worktrees with optional terminal multiplexer integration.

## Vision

Grove is a standalone TUI that makes git worktrees as easy to manage as branches in lazygit. It's designed to:

1. **Work anywhere** - Pure terminal app, no dependencies on tmux/zellij
2. **Integrate with anything** - Configurable "open" action for any multiplexer or editor
3. **Be safe** - Smart deletion with merge status and unique commit detection
4. **Be fast** - Single Go binary, instant startup

## Design Decisions

**Scope**: Grove is focused exclusively on worktree management. Full git operations (commit, push, pull, merge, rebase) are out of scope - that's lazygit's job.

**Repo-scoped**: Grove works within a single repository. It doesn't manage worktrees across multiple repos.

**Bare repo support**: Full support for bare repository workflows, unlike lazygit which struggles with these.

---

## Prior Art & Research

### Existing Tools Analyzed

We analyzed 5 existing worktree management tools during design. Here's what we learned:

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

**Limitations:**
- **CLI only** - No TUI, relies on external fzf via shell bindings
- **No worktree listing** - Must use raw `git worktree list`
- **macOS only** - Stated in README
- **Sessions, not windows** - Creates full tmux sessions, not windows
- **No interactive selection** - Requires branch name as argument
- **No confirmation prompts** - Deletes without asking

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

**Limitations:**
- **No worktree switching** - Can open in editor but no terminal/multiplexer integration
- **Settings are view-only** - Must edit JSON manually
- **No fuzzy search** - Arrow navigation only
- TypeScript/Bun dependency (not a single binary)

**Key insight**: Branchlet shows what a polished worktree TUI looks like. The file copying and template variable system is worth adopting. But the lack of switching is a dealbreaker for a "worktree manager".

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

**Limitations:**
- **NO DELETE COMMAND** - Ironically named "manager" but can't delete worktrees
- **No switching** - Shows paths but doesn't help navigate
- **CLI, not TUI** - Rich prompts but not interactive selection
- Python dependency

**Key insight**: Shows what's possible with Rich for beautiful terminal output. The caching and config patterns are solid. But critically incomplete.

---

### 5. worktree-cli (wt)

**Repository**: https://github.com/johnlindquist/worktree-cli
**Stars**: ~200
**Language**: TypeScript
**Maturity**: Production-ready

**What it does well:**

**Safety - The Gold Standard:**

1. **Atomic Operations with Rollback** - Tracks all actions and automatically rolls back on failure
2. **Stash Management** - Uses hash-based stash (`git stash create`) not stack, preventing race conditions
3. **SIGINT Handling** - Restores stashed changes even if interrupted
4. **Merge Safety** - Checks uncommitted changes, requires explicit flags for destructive operations

**Limitations:**
- **No real worktree switching** - Only opens in editor
- **No multiplexer integration** - Editor-focused
- **TypeScript, not binary** - Requires Node.js
- No TUI list view (uses prompts)

**Key insight**: worktree-cli is the most feature-complete tool. The atomic operation pattern and stash handling are worth replicating.

---

## Feature Specification

### Core Features

#### List View

```
┌─ grove ─────────────────────────────────────────── ~/myproject ─┐
│                                                                  │
│  WORKTREES                                                       │
│  ────────────────────────────────────────────────────────────── │
│                                                                  │
│  › main              .               ✓ clean                    │
│    feat/auth         .worktrees/auth ✗ 3 modified   ↑2 ahead   │
│    feat/dashboard    .worktrees/dash ✓ clean        ✓ merged    │
│    fix/login         .worktrees/log  ⚠ 2 unique     📦 2 stash  │
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
- `📦 N stash` - Has N stashed changes

#### Create Worktree

Flow:
1. Select or type branch name (fuzzy search existing, or create new)
2. Branch type indicators show `[worktree]`, `[local]`, `[remote]` prefixes
3. Current branch shown first with "(current)" suffix, then default branch
4. If new branch: select base branch
5. Confirm path (default: `.worktrees/{branch-name}`)
6. Create and optionally open

**File copying**: Automatically copy files matching configured patterns (e.g., `.env*`)

**Post-create hooks**: Run configured commands after creation (e.g., `npm install`)

**Worktree templates**: Pattern-based configuration for different branch types:
```toml
[[worktree.templates]]
pattern = "feature/*"
copy_patterns = [".env.local"]
post_create_cmd = ["npm install", "npm run setup"]
```

#### Delete Worktree

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

#### Open Worktree

Configurable action with template variables:
- `{path}` - Full path to worktree
- `{branch}` - Branch name
- `{branch_short}` - Branch name after last `/` (e.g., `auth` from `feat/auth`)
- `{repo}` - Repository name
- `{window_name}` - Window name based on `window_name_style` config

**Path-based window detection**: Optionally detect existing windows by path instead of name.

**Layout presets**: Apply tmux/zellij layouts after opening (`none`, `dev`, or custom command).

#### Branch Rename

Press `r` to rename the current worktree's branch. Runs `git branch -m old new`.

### Additional Features

#### Fuzzy Search/Filter

Press `/` to enter filter mode with fuzzy matching.

#### Detail Panel

Press `tab` to toggle expanded details for the selected worktree.

#### Fetch All

Press `f` to fetch all remotes (shows progress).

#### Stash on Switch

Optionally auto-stash dirty worktree before opening another.

#### Configurable Keybindings

All keybindings are configurable:
```toml
[keys]
up = "up,k"
down = "down,j"
open = "enter"
new = "n"
delete = "d"
rename = "r"
filter = "/"
fetch = "f"
detail = "tab"
help = "?"
quit = "q,ctrl+c"
```

#### Mouse Support

Click to select worktrees.

#### Theme Support

`auto`, `dark`, or `light` themes.

#### First-run Experience

- Creates default config with comments on first run
- Detects if running in tmux/zellij
- Suggests environment-specific config

#### Config Validation

- Warns on unknown template variables
- Validates enum config values
- Shows warnings on startup

---

## Technical Architecture

### Project Structure

```
grove/
├── cmd/grove/           # Entry point
├── internal/
│   ├── app/            # Main Bubble Tea model, state machine, keybindings
│   ├── ui/             # Pure rendering functions, styles
│   ├── git/            # Git operations (worktree, branch, status, safety)
│   └── config/         # Configuration loading and validation
├── integrations/
│   ├── tmux/           # TPM-compatible plugin
│   └── zellij/         # Zellij config examples
├── docs/               # Documentation
├── Formula/            # Homebrew formula
└── .github/workflows/  # CI/CD
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

### Key Patterns

#### Unique Commits Detection

```bash
# Commits on branch that are not on any remote
git log {branch} --not --remotes --oneline

# If this returns nothing, all commits are "safe" (exist elsewhere)
```

#### Merge Status Check

```bash
# Check if branch is merged into default
git branch --merged {default_branch} | grep -q {branch}
```

---

## References

- [lazygit](https://github.com/jesseduffield/lazygit) - TUI patterns, bubbletea usage
- [twt](https://github.com/todaatsushi/twt) - Tmux session integration patterns
- [branchlet](https://github.com/raghavpillai/branchlet) - TUI design, file copying
- [worktree-cli](https://github.com/johnlindquist/worktree-cli) - Safety patterns, atomic operations
- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework docs
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling docs
