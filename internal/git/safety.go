package git

import (
	"strings"
)

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

	HasUnpushedCommits  bool
	UnpushedCommitCount int

	IsMerged bool

	HasUniqueCommits  bool
	UniqueCommitCount int
	UniqueCommits     []CommitInfo
}

// CommitInfo represents basic commit information.
type CommitInfo struct {
	Hash    string
	Message string
}

// CheckSafety analyzes a worktree and returns safety information.
func CheckSafety(worktreePath, branch, defaultBranch string) (*SafetyInfo, error) {
	info := &SafetyInfo{
		Level: SafetyLevelSafe,
	}

	// 1. Check for uncommitted changes
	isDirty, count, err := GetDirtyStatus(worktreePath)
	if err == nil && isDirty {
		info.HasUncommittedChanges = true
		info.UncommittedFileCount = count
		info.Level = SafetyLevelWarning
	}

	// 2. Check if branch is merged to default
	if branch != "" && branch != defaultBranch {
		merged, err := IsBranchMerged(branch, defaultBranch)
		if err == nil {
			info.IsMerged = merged
		}
	} else {
		// Default branch is always considered "merged"
		info.IsMerged = true
	}

	// 3. Check for unpushed commits
	if branch != "" {
		ahead, _, err := GetUpstreamStatus(worktreePath, branch)
		if err == nil && ahead > 0 {
			info.HasUnpushedCommits = true
			info.UnpushedCommitCount = ahead
			if info.Level < SafetyLevelWarning {
				info.Level = SafetyLevelWarning
			}
		}
	}

	// 4. Check for unique commits (the key safety feature)
	// These are commits that exist ONLY on this branch and nowhere else
	if branch != "" {
		commits, err := GetUniqueCommits(branch)
		if err == nil && len(commits) > 0 {
			info.HasUniqueCommits = true
			info.UniqueCommitCount = len(commits)
			info.UniqueCommits = commits
			info.Level = SafetyLevelDanger
		}
	}

	return info, nil
}

// GetUniqueCommits returns commits that exist only on this branch.
// These are commits that are not on any remote branch.
func GetUniqueCommits(branch string) ([]CommitInfo, error) {
	// git log {branch} --not --remotes --oneline
	// Shows commits that are on this branch but not on ANY remote
	output, err := runGit("log", branch, "--not", "--remotes", "--oneline", "--format=%h\x00%s")
	if err != nil {
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var commits []CommitInfo
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) == 2 {
			commits = append(commits, CommitInfo{
				Hash:    parts[0],
				Message: parts[1],
			})
		}
	}

	return commits, nil
}

// IsBranchMerged checks if a branch is merged into another branch.
func IsBranchMerged(branch, intoBranch string) (bool, error) {
	// git branch --merged {intoBranch}
	output, err := runGit("branch", "--merged", intoBranch)
	if err != nil {
		return false, err
	}

	// Check if our branch is in the list
	for _, line := range strings.Split(output, "\n") {
		// Remove leading spaces and asterisk
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "* ")
		if name == branch {
			return true, nil
		}
	}

	return false, nil
}

// String returns a human-readable string for the safety level.
func (s SafetyLevel) String() string {
	switch s {
	case SafetyLevelSafe:
		return "safe"
	case SafetyLevelWarning:
		return "warning"
	case SafetyLevelDanger:
		return "danger"
	default:
		return "unknown"
	}
}
