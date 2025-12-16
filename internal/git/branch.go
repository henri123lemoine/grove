package git

import (
	"strings"
)

// Branch represents a Git branch.
type Branch struct {
	Name     string
	IsRemote bool
	IsCurrent bool
}

// ListBranches returns all local branches.
func ListBranches() ([]Branch, error) {
	output, err := runGit("branch", "--format=%(refname:short)\x00%(HEAD)")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) != 2 {
			continue
		}
		branches = append(branches, Branch{
			Name:      parts[0],
			IsRemote:  false,
			IsCurrent: parts[1] == "*",
		})
	}

	return branches, nil
}

// ListRemoteBranches returns all remote branches.
func ListRemoteBranches() ([]Branch, error) {
	output, err := runGit("branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		// Skip HEAD pointers like origin/HEAD
		if strings.HasSuffix(line, "/HEAD") {
			continue
		}
		branches = append(branches, Branch{
			Name:     line,
			IsRemote: true,
		})
	}

	return branches, nil
}

// ListAllBranches returns all local and remote branches.
func ListAllBranches() ([]Branch, error) {
	local, err := ListBranches()
	if err != nil {
		return nil, err
	}

	remote, err := ListRemoteBranches()
	if err != nil {
		return nil, err
	}

	return append(local, remote...), nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch() (string, error) {
	output, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// BranchExists checks if a local branch exists.
func BranchExists(name string) bool {
	_, err := runGit("rev-parse", "--verify", "refs/heads/"+name)
	return err == nil
}

// DeleteBranch deletes a local branch.
func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := runGit("branch", flag, name)
	return err
}

// GetWorktreeBranches returns a set of branches that are checked out in worktrees.
func GetWorktreeBranches() (map[string]bool, error) {
	worktrees, err := List()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, wt := range worktrees {
		if wt.Branch != "" && !strings.Contains(wt.Branch, "(detached)") {
			result[wt.Branch] = true
		}
	}

	return result, nil
}
