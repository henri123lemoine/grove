package git

import (
	"fmt"
	"strconv"
)

// PRWorktreeDir is the directory namespace used for PR review worktrees,
// relative to cfg.General.WorktreeDir. Doubles as the local branch prefix.
const PRWorktreeDir = "pr-review"

// PRBranchName returns the local branch name used for a PR review worktree.
func PRBranchName(prNum int) string {
	return PRWorktreeDir + "/" + strconv.Itoa(prNum)
}

// FetchPRHead fetches the head commit of GitHub PR prNum into a local branch
// named pr-review/<num>, force-updating it so re-runs pick up new commits.
// remote defaults to "origin" if empty. Returns the local branch name on success.
func FetchPRHead(prNum int, remote string) (string, error) {
	if prNum <= 0 {
		return "", fmt.Errorf("invalid PR number: %d", prNum)
	}

	repo, err := GetRepo()
	if err != nil {
		return "", err
	}

	if remote == "" {
		remote = "origin"
	}

	branch := PRBranchName(prNum)
	refspec := fmt.Sprintf("+refs/pull/%d/head:refs/heads/%s", prNum, branch)

	if _, err := runGitInDir(repo.MainWorktreeRoot, "fetch", remote, refspec); err != nil {
		return "", fmt.Errorf("failed to fetch PR #%d from %s: %w", prNum, remote, err)
	}

	return branch, nil
}
