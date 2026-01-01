# TODO

## Critical: Race Conditions
- [ ] Fix data race in worktree enrichment (`worktree.go:67-90`) - main thread and goroutines write to same struct fields simultaneously
- [ ] Fix debug package race condition (`debug.go:52,57,64`) - `enabled` variable read without lock

## Critical: Error Handling
- [ ] Add error handling for silent failures in `worktree.go:201` (git prune), `cache.go:93,107` (cache save), `multiplexer.go` (tmux/zellij commands)
- [ ] Handle errors from parallel worktree enrichment operations (`worktree.go:99,115,126,138-141`)

## High: Concurrency & Caching
- [ ] Add file locking for `cache.json` to prevent corruption from multiple Grove instances
- [ ] Add goroutine pool to limit concurrent goroutines for large worktree lists (currently unbounded)

## High: Test Coverage
- [ ] Add tests for `git/branch.go` (10 functions, 0% coverage)
- [ ] Add tests for `git/cache.go` (5 functions, 0% coverage)
- [ ] Add tests for `git/status.go` (4 functions, 0% coverage)
- [ ] Add tests for `git/stash.go` (5 functions, 0% coverage)
- [ ] Add tests for `exec/multiplexer.go` (25+ methods, 0% coverage)

## High: Configuration Validation
- [ ] Add validation for `GeneralConfig` (DefaultBaseBranch, WorktreeDir, Remote)
- [ ] Add validation for `WorktreeConfig` (CopyPatterns, CopyIgnores - bad globs silently skipped)
- [ ] Add validation for `KeysConfig` (16 key bindings - conflicts not detected, invalid syntax ignored)
- [ ] Validate `Pane.SplitFrom` for negative values (potential panic)

## High: Refactoring
- [ ] Extract state from Model into flow-specific structs (create, delete, filter, rename, stash, layout)
- [ ] Refactor `Update()` function (340+ lines, 16+ message types) into smaller message handlers
- [ ] Extract duplicated cursor/scroll management logic (`app.go:1535-1579`)
- [ ] Extract duplicated "find current worktree" loops (`app.go:376-382, 675-680, 783-787, 1059-1063`)

## Medium: Performance
- [ ] Cache column widths and divider strings in `render.go` to avoid recalculation on every render
- [ ] Fix `ResolvePath()` being called twice per worktree (`worktree.go:74-81`)
- [ ] Replace O(n) linear search for detail updates (`app.go:504-519`) with map lookup

## Features
- [ ] Look into branch pruning options and features

## Completed
- [x] Create `Multiplexer` interface for tmux/zellij abstraction (replace 10+ switch statements in `open.go`)
- [x] Add package documentation to all internal packages
- [x] Setup homebrew installation
