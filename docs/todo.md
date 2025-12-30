# TODO

- [x] Shell-escape `open.command` template variables to prevent injection/broken commands.
- [ ] Improve default-branch detection and safety behavior (treat detection/merge/unique-commit errors as warnings that require confirmation).
- [ ] Align config defaults between runtime and generated config file (`open.detect_existing` mismatch).
- [ ] Add preflight checks for worktree creation conflicts (branch already checked out elsewhere, path nesting under existing worktree, clearer errors).
