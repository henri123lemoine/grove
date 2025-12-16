// Package git provides Git operations for worktree management.
package git

// Worktree represents a Git worktree with its status.
type Worktree struct {
	Path       string
	Branch     string
	IsCurrent  bool
	IsMain     bool
	IsDirty    bool
	DirtyFiles int

	// Upstream tracking
	Ahead  int
	Behind int

	// Safety info
	IsMerged      bool
	UniqueCommits int // Commits that exist only on this branch

	// Last commit
	LastCommitHash    string
	LastCommitMessage string
	LastCommitTime    string
}

// List returns all worktrees in the current repository.
func List() ([]Worktree, error) {
	// TODO: Implement
	// git worktree list --porcelain
	return nil, nil
}

// Create creates a new worktree.
func Create(path, branch, baseBranch string) error {
	// TODO: Implement
	// git worktree add [-b branch] path [baseBranch]
	return nil
}

// Remove removes a worktree.
func Remove(path string, force bool) error {
	// TODO: Implement
	// git worktree remove [--force] path
	return nil
}
