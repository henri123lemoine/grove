# TODO

- [x] Shell-escape `open.command` template variables to prevent injection/broken commands.
- [x] Improve default-branch detection and safety behavior (treat detection/merge/unique-commit errors as warnings that require confirmation).
- [x] Consolidate default config generation to read values from `DefaultConfig()` (single source of truth).
- [x] Add preflight checks for worktree creation conflicts (branch already checked out elsewhere, path nesting under existing worktree, clearer errors).
