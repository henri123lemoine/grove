# TODO

- [x] Create `Multiplexer` interface for tmux/zellij abstraction (replace 10+ switch statements in `open.go`)
- [ ] Extract state from Model into flow-specific structs (create, delete, filter, rename, stash, layout)
- [ ] Cache column widths and divider strings in `render.go` to avoid recalculation on every render
- [x] Add package documentation to all internal packages
- [ ] Setup homebrew installation
- [ ] Look into branch pruning options and features.
