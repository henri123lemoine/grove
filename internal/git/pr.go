package git

import (
	"fmt"
	"strconv"
	"strings"
)

// PRWorktreeDir is the directory namespace for PR worktrees, relative to
// cfg.General.WorktreeDir. Doubles as the local branch prefix (pr/<num>).
const PRWorktreeDir = "pr"

// PRBranchName returns the local branch name used for a PR worktree.
func PRBranchName(prNum int) string {
	return PRWorktreeDir + "/" + strconv.Itoa(prNum)
}

// PRNumberFromBranch returns the PR number encoded in a branch named "pr/<n>",
// or 0 if the branch is not a PR worktree branch.
func PRNumberFromBranch(branch string) int {
	rest, ok := strings.CutPrefix(branch, PRWorktreeDir+"/")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// FetchPRHead fetches the head commit of GitHub PR prNum into a local branch
// named pr/<num>, force-updating it so re-runs pick up new commits.
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
