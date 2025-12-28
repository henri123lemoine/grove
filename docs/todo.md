# TODO

- [x] Creating a new worktree in the popup should create a new window
    Fixed: Added `open_after_create` config option (default: true) that auto-opens worktrees after creation.
- [x] Associatedly, if I select a worktree that exists but for which a tmux window doesn't exist, it seems to do nothing, rather than creating a tmux window and going there.
    Fixed: Improved error handling to catch and display errors from tmux commands.
- [x] Is the layout functionality working at all?
    Fixed: Layouts were applied to wrong window (grove's window). Now targets new window by name.
- [ ] Rare behaviour where there seems to be a remaining ~cached(?) version of the worktree name right after it's been deleted. like when you run the popup, you might be over testing2 worktree. but if within the popup you delete it, it naturally moves you to another tmux window, and it behaves kinda weirdly, where you esc the popup and it brings you to the main display of grove but with testing2 still there. if you close the popup and reopen it though, it's gone.
