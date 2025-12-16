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
func GetUpstreamStatus(worktreePath, branch string) (ahead, behind int, err error) {
	// Check if there's an upstream
	_, err = runGitInDir(worktreePath, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil {
		// No upstream configured
		return 0, 0, nil
	}

	// Get count
	output, err := runGitInDir(worktreePath, "rev-list", "--left-right", "--count", branch+"@{upstream}..."+branch)
	if err != nil {
		return 0, 0, nil
	}

	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 2 {
		return 0, 0, nil
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])

	return ahead, behind, nil
}

// GetLastCommit returns information about the last commit in a worktree.
func GetLastCommit(worktreePath string) (hash, message, relTime string, err error) {
	// Format: short hash, subject, relative time
	output, err := runGitInDir(worktreePath, "log", "-1", "--format=%h\x00%s\x00%cr")
	if err != nil {
		return "", "", "", err
	}

	parts := strings.Split(strings.TrimSpace(output), "\x00")
	if len(parts) != 3 {
		return "", "", "", nil
	}

	return parts[0], parts[1], parts[2], nil
}

// FetchAll fetches updates for all remotes.
func FetchAll() error {
	_, err := runGit("fetch", "--all", "--prune")
	return err
}
