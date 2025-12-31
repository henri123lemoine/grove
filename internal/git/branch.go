package git

import (
	"sort"
	"strings"
)

// Branch represents a Git branch or tag.
type Branch struct {
	Name       string
	IsRemote   bool
	IsCurrent  bool
	IsWorktree bool // Branch is checked out in a worktree
	IsTag      bool // This is a tag, not a branch
}

// ListBranches returns all local branches.
func ListBranches() ([]Branch, error) {
	// Use --list to get branches with current indicator
	output, err := runGit("branch", "--list", "--format=%(HEAD)%(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		isCurrent := strings.HasPrefix(line, "*")
		name := strings.TrimPrefix(line, "*")
		name = strings.TrimPrefix(name, " ")
		branches = append(branches, Branch{
			Name:      name,
			IsRemote:  false,
			IsCurrent: isCurrent,
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

// ListTags returns all tags.
func ListTags() ([]Branch, error) {
	output, err := runGit("tag", "--list", "--sort=-creatordate")
	if err != nil {
		return nil, err
	}

	var tags []Branch
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		tags = append(tags, Branch{
			Name:  line,
			IsTag: true,
		})
	}

	return tags, nil
}

// ListAllBranchesWithWorktreeStatus returns all branches and tags with worktree status.
// Sorted: current first, default branch, worktrees, local branches, remote branches, then tags.
func ListAllBranchesWithWorktreeStatus() ([]Branch, error) {
	// Get all branches
	local, err := ListBranches()
	if err != nil {
		return nil, err
	}

	remote, err := ListRemoteBranches()
	if err != nil {
		return nil, err
	}

	// Get tags (ignore errors - repo might not have tags)
	tags, _ := ListTags()

	// Get worktree branches
	worktreeBranches, err := GetWorktreeBranches()
	if err != nil {
		worktreeBranches = make(map[string]bool)
	}

	// Get repo for default branch
	repo, _ := GetRepo()
	defaultBranch := "main"
	if repo != nil && repo.DefaultBranch != "" {
		defaultBranch = repo.DefaultBranch
	}

	// Mark worktree status
	for i := range local {
		local[i].IsWorktree = worktreeBranches[local[i].Name]
	}
	for i := range remote {
		// Extract branch name without remote prefix (origin/main -> main)
		parts := strings.SplitN(remote[i].Name, "/", 2)
		if len(parts) == 2 {
			remote[i].IsWorktree = worktreeBranches[parts[1]]
		}
	}

	// Combine all branches and tags
	allBranches := append(local, remote...)
	allBranches = append(allBranches, tags...)

	// Sort: current first, default, worktrees, local, remote, then tags
	sort.SliceStable(allBranches, func(i, j int) bool {
		bi, bj := allBranches[i], allBranches[j]

		// Tags come last
		if bi.IsTag != bj.IsTag {
			return !bi.IsTag
		}

		// Current branch first
		if bi.IsCurrent != bj.IsCurrent {
			return bi.IsCurrent
		}

		// Default branch second
		iDefault := bi.Name == defaultBranch
		jDefault := bj.Name == defaultBranch
		if iDefault != jDefault {
			return iDefault
		}

		// Worktree branches third
		if bi.IsWorktree != bj.IsWorktree {
			return bi.IsWorktree
		}

		// Local before remote
		if bi.IsRemote != bj.IsRemote {
			return !bi.IsRemote
		}

		// Alphabetical
		return bi.Name < bj.Name
	})

	return allBranches, nil
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
	repo, err := GetRepo()
	if err != nil {
		return err
	}

	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err = runGitInDir(repo.MainWorktreeRoot, "branch", flag, name)
	return err
}

// RenameBranch renames a branch within the given worktree.
func RenameBranch(worktreePath, oldName, newName string) error {
	_, err := runGitInDir(worktreePath, "branch", "-m", oldName, newName)
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
