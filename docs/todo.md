# TODO

These TODOs are to be prioritized roughly in the order they are in the page.

## Misc

- [ ] For some reason when you filder with /, it doesn't filter everything? I don't really know how the filtering works.
- [ ] When I'm in a window and want to go to a worktree that already exists, it creates a new one instead of going to the one that already exists

## Edge Cases

- [ ] **Empty WorktreeDir**: Creates paths like `/repo//branch-name`
- [ ] **Symlinked worktrees**: Path comparisons may fail with symlinks
- [ ] **Worktree creation path conflicts**: Better error message when sanitized path already exists
- [ ] **Stash list shows global stashes**: May be confusing since `git stash` is repo-wide, not worktree-specific

## Performance

- [ ] **Combine GetLastCommit git calls**: Currently makes 3 separate git calls, could be one with `--format=%h|%s|%cr`
- [ ] **Repository cache thread safety**: `currentRepo` global accessed without synchronization (low risk in practice)

## UX

- [ ] Instead of the info being packed to the right, it should look more like a table (with all the times being aligned nicely)
- [ ] All that has colour-appropriate stuff (like the commit LOC changes) should be coloured appropriately
- [ ] **Adaptive detail panel width**: Currently fixed at 50 chars
- [ ] **Consistent commit message truncation**: Different lengths in different views (60, 50, 47 chars)
- [ ] **Config warnings display**: Currently printed to stderr before TUI starts, disappear immediately

## Code Quality

- [ ] **Template pattern matching edge cases**: `feature/*` vs `feature/foo/bar` matching
- [ ] **`isIgnored` nested path matching**: `node_modules/package/file` won't match `node_modules/**`
- [ ] **IsBranchMerged for commit hashes**: Uses `git branch --merged` which lists branches, not commits

## Missing Features

- [ ] **Cancel long-running operations**: During fetch, ESC sets state but doesn't cancel the underlying git operation
- [ ] **Pagination for large worktree lists**: With many worktrees, items get cut off without scrolling
- [ ] **Refresh/reload shortcut**: No way to refresh worktree list without restarting or fetching
- [ ] **Create stash from UI**: Can only view/pop/apply/drop stashes, not create new ones
- [ ] **Confirmation before prune**: Prune runs immediately without confirmation
- [ ] **Delete branch after worktree deletion**: Option to also delete the branch when removing worktree
- [ ] **Sorting options for worktree list**: Sort by name, date, dirty status, etc.
- [ ] **Visual feedback after prune**: Show how many entries were pruned
- [ ] **Create worktree at specific commit/tag**: Currently only supports branches
- [ ] **Shell quoting for paths with spaces**: Template expansion doesn't quote paths

