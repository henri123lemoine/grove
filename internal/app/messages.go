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
	Path string
	Err  error
}

// WorktreeDeletedMsg is sent when a worktree is deleted.
type WorktreeDeletedMsg struct {
	Path string
	Err  error
}

// WorktreeOpenedMsg is sent when a worktree is opened.
type WorktreeOpenedMsg struct {
	Err error
}

// FetchCompletedMsg is sent when fetch completes.
type FetchCompletedMsg struct {
	Err error
}

// ErrorMsg is a general error message.
type ErrorMsg struct {
	Err error
}
