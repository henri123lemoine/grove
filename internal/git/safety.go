package git

// SafetyLevel indicates how risky it is to delete a worktree.
type SafetyLevel int

const (
	// SafetyLevelSafe means the worktree can be deleted without data loss.
	// - Clean working directory
	// - Branch merged to default
	// - No unique commits
	SafetyLevelSafe SafetyLevel = iota

	// SafetyLevelWarning means deletion will lose some work, but it's recoverable.
	// - Has uncommitted changes
	// - Has unpushed commits (but pushed to remote)
	// - Branch not merged (but exists on remote)
	SafetyLevelWarning

	// SafetyLevelDanger means deletion will permanently lose commits.
	// - Has commits that exist ONLY on this branch
	// - Not pushed, not merged anywhere
	SafetyLevelDanger
)

// SafetyInfo contains details about the safety of deleting a worktree.
type SafetyInfo struct {
	Level SafetyLevel

	// Details
	HasUncommittedChanges bool
	UncommittedFileCount  int

	HasUnpushedCommits bool
	UnpushedCommitCount int

	IsMerged bool

	HasUniqueCommits bool
	UniqueCommitCount int
	UniqueCommits     []CommitInfo
}

// CommitInfo represents basic commit information.
type CommitInfo struct {
	Hash    string
	Message string
}

// CheckSafety analyzes a worktree and returns safety information.
func CheckSafety(worktreePath, defaultBranch string) (*SafetyInfo, error) {
	// TODO: Implement
	//
	// 1. Check for uncommitted changes:
	//    git -C {path} status --porcelain
	//
	// 2. Check if branch is merged:
	//    git branch --merged {defaultBranch} | grep {branch}
	//
	// 3. Check for unpushed commits:
	//    git log @{upstream}..HEAD --oneline
	//
	// 4. Check for unique commits (the key safety feature):
	//    git log {branch} --not --remotes --oneline
	//    This shows commits that exist ONLY on this local branch
	//    and are not on ANY remote branch.
	//
	// 5. Determine safety level based on findings

	return nil, nil
}
