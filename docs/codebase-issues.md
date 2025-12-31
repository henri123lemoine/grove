# Grove Codebase Issues Analysis

This document contains a comprehensive analysis of issues found in the Grove codebase, organized by category.

---

## Table of Contents

1. [Error Handling Issues](#1-error-handling-issues)
2. [Race Conditions & Concurrency](#2-race-conditions--concurrency)
3. [Code Duplication](#3-code-duplication)
4. [Test Coverage Gaps](#4-test-coverage-gaps)
5. [API/Interface Design Issues](#5-apiinterface-design-issues)
6. [Performance Issues](#6-performance-issues)
7. [Configuration Validation Gaps](#7-configuration-validation-gaps)
8. [Documentation Issues](#8-documentation-issues)
9. [Maintainability Issues](#9-maintainability-issues)

---

## 1. Error Handling Issues

### Critical: Silent Error Swallowing

| File | Line | Issue |
|------|------|-------|
| `worktree.go` | 201 | `_, _ = runGitInDir(...)` - Git prune fails silently |
| `cache.go` | 93, 107 | `_ = SaveCache(...)` - Cache save failures ignored |
| `multiplexer.go` | 167, 218, 306, 308, 336, 338 | Tmux/Zellij command failures ignored |

### Critical: Parallel Operations Without Error Handling

| File | Line | Issue |
|------|------|-------|
| `worktree.go` | 99 | `wt.IsDirty, wt.DirtyFiles, _ = GetDirtyStatus(...)` |
| `worktree.go` | 115 | `wt.Ahead, wt.Behind, wt.HasUpstream, _ = GetUpstreamStatus(...)` |
| `worktree.go` | 126 | `wt.LastCommitHash, wt.LastCommitMessage, wt.LastCommitTime, _ = GetLastCommit(...)` |
| `worktree.go` | 138-141 | `wt.IsMerged, _` and `wt.UniqueCommits` errors swallowed |

### Inconsistent Error Wrapping

- `repo.go:92-94` - Some errors wrapped with context
- `repo.go:109-122` - Other errors NOT wrapped with context
- `safety.go` - Errors recorded in struct instead of returned

---

## 2. Race Conditions & Concurrency

### CRITICAL: Data Race in Worktree Enrichment

**File**: `worktree.go:67-90`

```go
var wg sync.WaitGroup
for i := range worktrees {
    wt := &worktrees[i]
    // Main thread writes IsCurrent, IsMain (lines 74, 81)
    wg.Add(1)
    go func(wt *Worktree) {
        defer wg.Done()
        enrichWorktree(wt, repo)  // Goroutine writes IsDirty, DirtyFiles
    }(wt)
}
```

Main thread and goroutines write to same struct fields simultaneously.

### CRITICAL: Debug Package Race Condition

**File**: `debug.go:52, 57, 64`

```go
func IsEnabled() bool {
    return enabled      // NO LOCK - RACE!
}

func Log(format string, args ...interface{}) {
    if !enabled {       // NO LOCK - RACE!
        return
    }
    mu.Lock()           // Too late - already read enabled
```

### High: Cache Invalidation Race

**File**: `worktree.go:286`

No file locking for cache.json - multiple Grove instances can corrupt cache.

### High: Unbounded Goroutine Spawning

**File**: `worktree.go:68-89`

No goroutine pool for large worktree lists (1000+ worktrees = 1000+ goroutines).

---

## 3. Code Duplication

### High Priority Duplications

| Pattern | Locations | Frequency |
|---------|-----------|-----------|
| Cursor/scroll management | `app.go:1535-1579` | 2x |
| Visible count calculation | `app.go:1519-1533, 1699-1714` | 2x |
| State reset (delete flow) | `app.go:862-896, 1122-1179` | 8x |
| Find current worktree loop | `app.go:376-382, 675-680, 783-787, 1059-1063` | 4x |
| Y/N confirmation pattern | `app.go:887-898, 1166-1181` | 3x |

### Medium Priority Duplications

| Pattern | Locations | Frequency |
|---------|-----------|-----------|
| Scroll indicators | `render.go:238-259, 460-462, 499-501, 659-661` | 6x |
| Header/divider rendering | `render.go` throughout | 9x |
| Footer help rendering | `render.go` throughout | 7x |
| Stash operations | `stash.go:73-88` | 3x |
| Text input initialization | `app.go:169-183` | 4x |

---

## 4. Test Coverage Gaps

### Packages with Zero Test Coverage

| Package/File | Functions | Status |
|--------------|-----------|--------|
| `git/branch.go` | 10 functions | 0% tested |
| `git/cache.go` | 5 functions | 0% tested |
| `git/status.go` | 4 functions | 0% tested |
| `git/stash.go` | 5 functions | 0% tested |
| `exec/multiplexer.go` | 25+ methods | 0% tested |

### Significantly Under-tested Areas

| File | Functions | Coverage |
|------|-----------|----------|
| `app/app.go` | 60+ functions | ~18% |
| `git/worktree.go` | 17 functions | ~11% |
| `git/repo.go` | 8 functions | ~12% |
| `git/safety.go` | 7 functions | ~14% |

### Untested Error Scenarios

- Git command failures (network, permissions)
- File operations (disk full, read-only)
- Corrupted cache files
- Multiplexer not installed
- Invalid config syntax

---

## 5. API/Interface Design Issues

### Inconsistent Function Signatures

| Issue | Location |
|-------|----------|
| Enrich functions return void vs error | `worktree.go:107-132` |
| Some git operations mutate, some return | `repo.go, worktree.go` |
| CheckSafety() errors recorded in struct, not returned | `safety.go:68-154` |

### Functions Doing Too Much

| Function | Lines | Responsibilities |
|----------|-------|------------------|
| `Update()` | 340 | 16+ message types |
| `CheckSafety()` | 85 | 4 distinct checks |
| `OpenWithConfig()` | 70 | Window detection, templates, open, layout |

### Leaky Abstractions

- `RenderParams` struct has 30+ fields for different states
- `Worktree` exposes lazy-loaded fields that may be uninitialized
- `SafetyInfo` has error list AND boolean fields

### Package Coupling Issues

- `git` package directly calls `ListAndCache()` in `Remove()`
- `git` package imports `debug` package
- No interface abstraction for multiplexer backends

---

## 6. Performance Issues

### High Priority

| Issue | File | Lines |
|-------|------|-------|
| Column widths recalculated every render | `render.go` | 225, 646 |
| ResolvePath() called twice per worktree | `worktree.go` | 74-81 |
| O(n) linear search for detail updates | `app.go` | 504-519 |
| Custom O(n²) string replace | `app.go` | 1499-1516 |

### Medium Priority

| Issue | File | Lines |
|-------|------|-------|
| Branch lookup via linear search | `app.go` | 799-805 |
| Map created per merged branch check | `safety.go` | 205-210 |
| Full worktree copy for upstream status | `app.go` | 1441-1445 |
| Append without pre-allocation | `app.go` | 1211 |

---

## 7. Configuration Validation Gaps

### Completely Unvalidated Config Sections

| Section | Fields |
|---------|--------|
| `GeneralConfig` | DefaultBaseBranch, WorktreeDir, Remote |
| `WorktreeConfig` | CopyPatterns, CopyIgnores |
| `KeysConfig` | All 16 key bindings |

### Partially Validated

| Field | Missing Validation |
|-------|-------------------|
| `Open.Command` | Shell syntax not checked |
| `Open.LayoutCommand` | Shell syntax not checked |
| `Pane.SplitFrom` | Negative values not checked |
| Key bindings | Conflicts not detected |

### Runtime Error Scenarios

1. Invalid WorktreeDir (e.g., `/etc/passwd`) - creates in dangerous location
2. Bad glob patterns in CopyPatterns - silently skipped
3. Invalid key syntax - silently ignored or crashes
4. Negative SplitFrom - potential panic

---

## 8. Documentation Issues

### Missing Godoc Comments

**CRITICAL (Public API):**
- `Render()` in `ui/render.go`
- `EchoPath()`, `GetMultiplexer()`, `FindWindowsForPath()`, `CloseWindow()` in `exec/open.go`
- `Multiplexer` type definition

**HIGH (19 command functions in app.go):**
- `loadWorktrees()`, `refreshWorktrees()`, `checkSafety()`, `createWorktree()`, `deleteWorktree()`, etc.

### Missing Package Documentation

- `internal/exec` - No doc.go
- `internal/git` - No doc.go

### Misleading Comments

- `worktree.go:95` - `enrichWorktree` ignores `_Repo` param without explanation
- Lazy-loaded fields not clearly documented

---

## 9. Maintainability Issues

### God Object

**File**: `app.go:102-164`

`Model` struct has **54 fields** spanning:
- Configuration, List state, Create flow, Delete flow, Filter, Rename flow, Stash flow, Layout flow, UI state

### Very Long Functions

| Function | File | Lines |
|----------|------|-------|
| `Update()` | `app.go` | 340+ |
| `handleListKeys()` | `app.go` | 122 |
| `renderList()` | `render.go` | 105 |
| `renderDelete()` | `render.go` | 77 |
| `renderSelectBase()` | `render.go` | 76 |

### Magic Numbers/Strings

| Value | Location | Purpose |
|-------|----------|---------|
| `5` | `app.go:70` | Sort mode count |
| `250` | `app.go:171` | Branch name char limit |
| `7` | `worktree.go:176` | Short hash length |
| `50`, `70`, `40` | `render.go` | Various widths |

### Deep Nesting

- Safety confirmation logic: 3+ levels (`app.go:318-348`)
- CopyFiles validation: 3+ levels (`worktree.go:365-407`)
- Path traversal checks: complex conditionals (`worktree.go:552-562`)

---

## Summary Statistics

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| Error Handling | 3 | 5 | 8 | 2 |
| Race Conditions | 3 | 3 | 2 | 1 |
| Code Duplication | - | 6 | 10 | 3 |
| Test Coverage | 6 | 5 | 4 | - |
| API Design | - | 4 | 6 | 3 |
| Performance | - | 4 | 4 | 2 |
| Config Validation | 2 | 4 | 3 | 2 |
| Documentation | 2 | 5 | 4 | 3 |
| Maintainability | - | 3 | 8 | 5 |

**Total Issues Identified: ~120+**

---

## Priority Recommendations

### Short-term

1. Add error handling for ignored errors
2. Extract repeated code patterns
3. Add tests for untested functions
4. Document exported functions

### Medium-term

1. Break down Model struct into sub-components
2. Refactor Update() into message handlers
3. Create interfaces for git operations and multiplexers
4. Add comprehensive config validation

### Long-term

1. Extract state machine for UI flows
2. Centralize help text and magic constants
3. Add performance benchmarks
4. Improve package boundaries
