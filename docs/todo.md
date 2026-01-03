# TODO

- [ ] Look into branch pruning options and features
- [ ] Fix `truncateMsg()` panic with small `maxLen`
- [ ] Fix `replace()` infinite loop with empty `old` string
- [ ] Add `".."` filtering to `sanitizePath()`
- [ ] Add synchronization to `multiplexerBackend` global
- [ ] Replace custom `replaceAll()` with `strings.ReplaceAll()`
- [ ] Add error handling for silently ignored errors
- [ ] Extract repeated code patterns (state reset, scroll indicators)
- [ ] Add tests for `ui/` package
- [ ] Break down `Model` struct into sub-components
- [ ] Extract `Update()` message handlers into methods
- [ ] Create interfaces for git operations for testability
- [ ] Fix UTF-8 truncation to preserve valid characters
- [ ] Improve test coverage for all packages
- [ ] Centralize magic numbers as named constants
- [ ] Remove dead code and deprecated types
- [ ] Add package-level documentation
