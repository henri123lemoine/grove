package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

	// Internal
	head string // The HEAD commit
}

// List returns all worktrees in the current repository.
func List() ([]Worktree, error) {
	repo, err := GetRepo()
	if err != nil {
		return nil, err
	}

	// Get porcelain output
	output, err := runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees := parseWorktreeList(output)

	// Get current working directory to identify current worktree
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	// Enrich with status information
	for i := range worktrees {
		wt := &worktrees[i]

		// Check if this is the current worktree
		if cwd != "" {
			wtPath, _ := filepath.Abs(wt.Path)
			cwdPath, _ := filepath.Abs(cwd)
			wt.IsCurrent = wtPath == cwdPath || strings.HasPrefix(cwdPath, wtPath+string(filepath.Separator))
		}

		// Check if this is the main worktree
		wt.IsMain = wt.Path == repo.Root || (repo.IsBare && i == 0)

		// Get dirty status
		wt.IsDirty, wt.DirtyFiles, _ = GetDirtyStatus(wt.Path)

		// Get upstream status
		if wt.Branch != "" {
			wt.Ahead, wt.Behind, _ = GetUpstreamStatus(wt.Path, wt.Branch)
		}

		// Get last commit
		wt.LastCommitHash, wt.LastCommitMessage, wt.LastCommitTime, _ = GetLastCommit(wt.Path)

		// Get merge status
		if wt.Branch != "" && wt.Branch != repo.DefaultBranch {
			wt.IsMerged, _ = IsBranchMerged(wt.Branch, repo.DefaultBranch)
		}

		// Get unique commits count
		if wt.Branch != "" {
			commits, _ := GetUniqueCommits(wt.Branch)
			wt.UniqueCommits = len(commits)
		}
	}

	return worktrees, nil
}

// parseWorktreeList parses the porcelain output of git worktree list.
func parseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			branch := strings.TrimPrefix(line, "branch ")
			// Remove refs/heads/ prefix
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		} else if line == "bare" && current != nil {
			// Bare repo worktree - no branch
		} else if line == "detached" && current != nil {
			// Detached HEAD - use short hash as "branch"
			if current.head != "" && len(current.head) >= 7 {
				current.Branch = current.head[:7] + " (detached)"
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

// Create creates a new worktree.
func Create(path, branch string, isNewBranch bool, baseBranch string) error {
	// Build command arguments
	args := []string{"worktree", "add"}

	if isNewBranch {
		args = append(args, "-b", branch, path)
		if baseBranch != "" {
			args = append(args, baseBranch)
		}
	} else {
		args = append(args, path, branch)
	}

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// Remove removes a worktree.
func Remove(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// ShortPath returns a shortened path relative to the repo root.
func (w *Worktree) ShortPath() string {
	repo, err := GetRepo()
	if err != nil {
		return w.Path
	}

	// Try to make it relative to repo root
	relPath, err := filepath.Rel(repo.Root, w.Path)
	if err != nil {
		return w.Path
	}

	// If the path starts with .., use absolute path
	if strings.HasPrefix(relPath, "..") {
		return w.Path
	}

	// Use . for the main worktree
	if relPath == "." {
		return "."
	}

	return relPath
}

// BranchShort returns the short branch name (after last /).
func (w *Worktree) BranchShort() string {
	if w.Branch == "" {
		return ""
	}
	parts := strings.Split(w.Branch, "/")
	return parts[len(parts)-1]
}
