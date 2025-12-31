package git

import (
	"fmt"
	"strings"
)

// SafetyLevel indicates how risky it is to delete a worktree.
type SafetyLevel int

const (
	// SafetyLevelSafe means the worktree can be deleted without data loss.
	// - Clean working directory
	// - Branch merged to default, or all commits pushed to remote
	// - No unique local-only commits
	SafetyLevelSafe SafetyLevel = iota

	// SafetyLevelWarning means deletion may lose some work, but it's recoverable.
	// - Has unpushed commits (but branch exists on remote)
	// - Branch not merged (but exists on remote)
	SafetyLevelWarning

	// SafetyLevelDanger means deletion will permanently lose work.
	// - Has uncommitted changes (staged, unstaged, or untracked files)
	// - Has commits that exist ONLY locally (not pushed, not merged)
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

	IsMerged         bool
	MergeStatusKnown bool

	HasUniqueCommits  bool
	UniqueCommitCount int
	UniqueCommits     []CommitInfo

	HasSafetyCheckErrors bool
	SafetyCheckErrors    []string
}

// CommitInfo represents basic commit information.
type CommitInfo struct {
	Hash    string
	Message string
}

// isDetachedBranch checks if the branch string represents a detached HEAD.
func isDetachedBranch(branch string) bool {
	return strings.HasSuffix(branch, "(detached)")
}

// remoteBranchExists checks if a remote branch exists (e.g., "origin/my-branch").
func remoteBranchExists(remoteBranch string) bool {
	_, err := runGit("rev-parse", "--verify", remoteBranch)
	return err == nil
}

// CheckSafety analyzes a worktree and returns safety information.
func CheckSafety(worktreePath, branch, defaultBranch string) (*SafetyInfo, error) {
	info := &SafetyInfo{
		Level: SafetyLevelSafe,
	}

	isDetached := isDetachedBranch(branch)
	recordError := func(format string, args ...interface{}) {
		info.HasSafetyCheckErrors = true
		info.SafetyCheckErrors = append(info.SafetyCheckErrors, fmt.Sprintf(format, args...))
		if info.Level < SafetyLevelWarning {
			info.Level = SafetyLevelWarning
		}
	}

	if defaultBranch == "" {
		recordError("default branch could not be detected")
	}

	// 1. Check for uncommitted changes (staged, unstaged, untracked)
	// These are truly unrecoverable, so this is Danger level
	isDirty, count, err := GetDirtyStatus(worktreePath)
	if err != nil {
		recordError("could not check uncommitted changes: %v", err)
	} else if isDirty {
		info.HasUncommittedChanges = true
		info.UncommittedFileCount = count
		info.Level = SafetyLevelDanger
	}

	// 2. Check if branch is merged to default
	// For detached HEAD, extract the commit hash and check if it's merged
	if isDetached {
		// Extract hash from "abc1234 (detached)"
		commitHash := strings.TrimSuffix(branch, " (detached)")
		if commitHash != "" && defaultBranch != "" {
			merged, err := IsBranchMerged(commitHash, defaultBranch)
			if err != nil {
				recordError("could not verify merge status: %v", err)
			} else {
				info.IsMerged = merged
				info.MergeStatusKnown = true
			}
		}
	} else if branch != "" && branch != defaultBranch && defaultBranch != "" {
		merged, err := IsBranchMerged(branch, defaultBranch)
		if err != nil {
			recordError("could not verify merge status: %v", err)
		} else {
			info.IsMerged = merged
			info.MergeStatusKnown = true
		}
	} else if defaultBranch != "" {
		// Default branch is always considered "merged"
		info.IsMerged = true
		info.MergeStatusKnown = true
	}

	// 3. Check for unpushed commits (skip for detached HEAD - no tracking branch)
	if branch != "" && !isDetached {
		ahead, _, hasUpstream, err := GetUpstreamStatus(worktreePath, branch)
		if err == nil && hasUpstream && ahead > 0 {
			info.HasUnpushedCommits = true
			info.UnpushedCommitCount = ahead
			if info.Level < SafetyLevelWarning {
				info.Level = SafetyLevelWarning
			}
		}
	}

	// 4. Check for unique commits (the key safety feature)
	// These are commits that exist ONLY on this branch and not on default
	// For detached HEAD, we can't determine unique commits easily, so skip
	if branch != "" && branch != defaultBranch && !isDetached && defaultBranch != "" {
		commits, err := GetUniqueCommits(branch, defaultBranch)
		if err != nil {
			recordError("could not verify unique commits: %v", err)
		} else if len(commits) > 0 {
			info.HasUniqueCommits = true
			info.UniqueCommitCount = len(commits)
			info.UniqueCommits = commits
			info.Level = SafetyLevelDanger
		}
	}

	return info, nil
}

// GetUniqueCommits returns commits that exist only on this branch.
// These are commits not on the default branch AND not pushed to the remote.
// If pushed to the remote, they're recoverable even if we delete the local branch.
func GetUniqueCommits(branch, defaultBranch string) ([]CommitInfo, error) {
	// First, check if there's a remote tracking branch
	remoteBranch := "origin/" + branch
	hasRemote := remoteBranchExists(remoteBranch)

	// git log {branch} --not {defaultBranch} [--not origin/{branch}]
	// Shows commits on this branch that aren't on the default branch (or remote)
	args := []string{"log", branch, "--not", defaultBranch}
	if hasRemote {
		args = append(args, "--not", remoteBranch)
	}
	args = append(args, "--format=%h %s")

	output, err := runGit(args...)
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
		// Split on first space: hash message
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 1 {
			msg := ""
			if len(parts) == 2 {
				msg = parts[1]
			}
			commits = append(commits, CommitInfo{
				Hash:    parts[0],
				Message: msg,
			})
		}
	}

	return commits, nil
}

// IsBranchMerged checks if a branch is merged into another branch.
func IsBranchMerged(branch, intoBranch string) (bool, error) {
	merged, err := GetMergedBranches(intoBranch)
	if err != nil {
		return false, err
	}
	return merged[branch], nil
}

// GetMergedBranches returns a set of all branches merged into the given branch.
// Call this once and reuse the result to avoid repeated git calls.
func GetMergedBranches(intoBranch string) (map[string]bool, error) {
	output, err := runGit("branch", "--merged", intoBranch)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "* ")
		if name != "" {
			merged[name] = true
		}
	}

	return merged, nil
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
