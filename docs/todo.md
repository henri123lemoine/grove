# TODO

## Immediate (Before Release)

- [ ] Fix O(n²) loops in `app.go:491-533` - Convert nested worktree update loops to O(n) using map-based lookups
- [ ] Fix stash cursor bounds issue (`app.go:998-1002`) - End key can set cursor to -1 with empty stash list
- [ ] Add path validation in `worktree.go:355-390` - Prevent path traversal via `copy_patterns` like `../*`
- [ ] Fix file permissions in `config.go:310-333` and `worktree.go:420,452` - Use 0700 for dirs, 0600 for files instead of 0755/0644

## High Priority

- [ ] Add test coverage for `CheckSafety()` with detached HEAD, empty default branch, git errors
- [ ] Add test coverage for all stash operations in `internal/git/stash.go`
- [ ] Extract duplicate `resolvePath()` function from `worktree.go:514-526` and `open.go:685-697` to shared package
- [ ] Add validation for `layout_command` when `layout = "custom"` is set
- [ ] Document complex safety check logic in `app.go:303-330`

## Medium Priority

- [ ] Create `Multiplexer` interface for tmux/zellij abstraction (replace 10+ switch statements in `open.go`)
- [ ] Extract state from Model into flow-specific structs (create, delete, filter, rename, stash, layout)
- [ ] Cache column widths and divider strings in `render.go` to avoid recalculation on every render
- [ ] Add package documentation to all internal packages

## Backlog

- [ ] Setup homebrew installation
