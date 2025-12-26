// Package app contains the main application state and logic.
package app

import (
	"github.com/henrilemoine/grove/internal/git"
)

// Message types for the bubbletea app.

// WorktreesLoadedMsg is sent when worktrees are loaded.
type WorktreesLoadedMsg struct {
	Worktrees []git.Worktree
	Err       error
}

// BranchesLoadedMsg is sent when branches are loaded.
type BranchesLoadedMsg struct {
	Branches []git.Branch
	Err      error
}

// SafetyCheckedMsg is sent when safety info is loaded.
type SafetyCheckedMsg struct {
	Info *git.SafetyInfo
	Err  error
}

// WorktreeCreatedMsg is sent when a worktree is created.
type WorktreeCreatedMsg struct {
	Path   string
	Branch string
	Err    error
}

// WorktreeDeletedMsg is sent when a worktree is deleted.
type WorktreeDeletedMsg struct {
	Path string
	Err  error
}

// WorktreeOpenedMsg is sent when a worktree is opened.
type WorktreeOpenedMsg struct {
	Err         error
	IsNewWindow bool
}

// FetchCompletedMsg is sent when fetch completes.
type FetchCompletedMsg struct {
	Err error
}

// ErrorMsg is a general error message.
type ErrorMsg struct {
	Err error
}

// BranchRenamedMsg is sent when branch rename completes.
type BranchRenamedMsg struct {
	OldName string
	NewName string
	Err     error
}

// StashCreatedMsg is sent when a stash is created.
type StashCreatedMsg struct {
	Err error
}

// StashPoppedMsg is sent when a stash is popped.
type StashPoppedMsg struct {
	Err error
}

// FileCopyCompletedMsg is sent when file copy completes.
type FileCopyCompletedMsg struct {
	Err error
}

// PostCreateHooksCompletedMsg is sent when post-create hooks complete.
type PostCreateHooksCompletedMsg struct {
	Err error
}

// PruneCompletedMsg is sent when worktree pruning completes.
type PruneCompletedMsg struct {
	PrunedCount int
	Err         error
}

// StashListLoadedMsg is sent when stash list is loaded.
type StashListLoadedMsg struct {
	Entries []git.StashEntry
	Err     error
}

// StashOperationCompletedMsg is sent when a stash operation completes.
type StashOperationCompletedMsg struct {
	Operation string // "pop", "apply", or "drop"
	Err       error
}
