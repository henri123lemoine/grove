// Package app provides the main Bubble Tea application model for Grove.
//
// It manages the UI state machine, handles user input, and coordinates
// between git operations and UI rendering. The package implements states
// for listing worktrees, creating/deleting worktrees, filtering, renaming
// branches, managing stashes, and selecting layouts.
//
// The main type is Model, which implements the Bubble Tea interface
// (Init, Update, View) and manages all application state.
package app
