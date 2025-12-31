package git

import (
	"strconv"
	"strings"
)

// GetDirtyStatus checks if a worktree has uncommitted changes.
func GetDirtyStatus(worktreePath string) (isDirty bool, count int, err error) {
	output, err := runGitInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, 0, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return false, 0, nil
	}

	// Count lines
	lines := strings.Split(output, "\n")
	return true, len(lines), nil
}

// GetUpstreamStatus returns how many commits a branch is ahead/behind its upstream.
// Returns hasUpstream=false if no upstream tracking is configured.
func GetUpstreamStatus(worktreePath, branch string) (ahead, behind int, hasUpstream bool, err error) {
	// Try to get count directly - if no upstream, this will fail
	output, err := runGitInDir(worktreePath, "rev-list", "--left-right", "--count", branch+"@{upstream}..."+branch)
	if err != nil {
		// No upstream configured (or other error)
		return 0, 0, false, nil
	}

	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 2 {
		return 0, 0, true, nil
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])

	return ahead, behind, true, nil
}

// GetLastCommit returns information about the last commit in a worktree.
func GetLastCommit(worktreePath string) (hash, message, relTime string, err error) {
	// Get all info in one call using a delimiter unlikely to appear in commit messages
	const delim = "\x00"
	output, err := runGitInDir(worktreePath, "log", "-1", "--format=%h"+delim+"%s"+delim+"%cr")
	if err != nil {
		return "", "", "", err
	}

	parts := strings.Split(strings.TrimSpace(output), delim)
	if len(parts) >= 1 {
		hash = parts[0]
	}
	if len(parts) >= 2 {
		message = parts[1]
	}
	if len(parts) >= 3 {
		relTime = parts[2]
	}

	return hash, message, relTime, nil
}

// FetchAll fetches updates for all remotes.
func FetchAll() error {
	repo, err := GetRepo()
	if err != nil {
		return err
	}
	_, err = runGitInDir(repo.MainWorktreeRoot, "fetch", "--all", "--prune")
	return err
}
