package git

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	IsDetached bool // True if HEAD is detached (not on a branch)

	// Upstream tracking
	Ahead  int
	Behind int

	// Safety info
	IsMerged      bool
	UniqueCommits int // Commits that exist only on this branch

	// Stash info
	StashCount int

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
		wt.IsMain = wt.Path == repo.MainWorktreeRoot || (repo.IsBare && i == 0)

		// Get dirty status
		wt.IsDirty, wt.DirtyFiles, _ = GetDirtyStatus(wt.Path)

		// Get upstream status (skip for detached HEAD - no tracking branch)
		if wt.Branch != "" && !wt.IsDetached {
			wt.Ahead, wt.Behind, _ = GetUpstreamStatus(wt.Path, wt.Branch)
		}

		// Get last commit
		wt.LastCommitHash, wt.LastCommitMessage, wt.LastCommitTime, _ = GetLastCommit(wt.Path)

		// Get merge status (skip for detached HEAD - use commit hash instead)
		if wt.Branch != "" && wt.Branch != repo.DefaultBranch && !wt.IsDetached {
			wt.IsMerged, _ = IsBranchMerged(wt.Branch, repo.DefaultBranch)
		} else if wt.IsDetached && wt.head != "" {
			// For detached HEAD, check if the commit itself is merged
			wt.IsMerged, _ = IsBranchMerged(wt.head, repo.DefaultBranch)
		}

		// Get unique commits count (skip for detached HEAD)
		if wt.Branch != "" && wt.Branch != repo.DefaultBranch && !wt.IsDetached {
			commits, _ := GetUniqueCommits(wt.Branch, repo.DefaultBranch)
			wt.UniqueCommits = len(commits)
		}

		// Get stash count
		wt.StashCount, _ = GetStashCount(wt.Path)
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
			// Detached HEAD - mark as detached and use short hash for display
			current.IsDetached = true
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

	// Try to make it relative to main worktree root
	relPath, err := filepath.Rel(repo.MainWorktreeRoot, w.Path)
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

// CopyFiles copies files matching patterns from source to dest worktree.
func CopyFiles(sourceDir, destDir string, patterns, ignores []string) error {
	for _, pattern := range patterns {
		// Find files matching pattern
		matches, err := filepath.Glob(filepath.Join(sourceDir, pattern))
		if err != nil {
			continue
		}

		for _, srcPath := range matches {
			// Check if ignored
			relPath, _ := filepath.Rel(sourceDir, srcPath)
			if isIgnored(relPath, ignores) {
				continue
			}

			// Determine destination path
			destPath := filepath.Join(destDir, relPath)

			// Copy file or directory
			info, err := os.Stat(srcPath)
			if err != nil {
				continue
			}

			if info.IsDir() {
				err = copyDir(srcPath, destPath, ignores)
			} else {
				err = copyFile(srcPath, destPath)
			}
			if err != nil {
				return fmt.Errorf("failed to copy %s: %w", relPath, err)
			}
		}
	}
	return nil
}

// isIgnored checks if a path matches any ignore pattern.
func isIgnored(path string, ignores []string) bool {
	for _, pattern := range ignores {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check against base name
		matched, err = filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDir copies a directory recursively.
func copyDir(src, dst string, ignores []string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if isIgnored(entry.Name(), ignores) {
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath, ignores); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RunPostCreateHooks runs post-create commands in the worktree directory.
func RunPostCreateHooks(worktreePath string, commands []string) error {
	for _, cmdStr := range commands {
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = worktreePath
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("post-create command failed: %s: %w", cmdStr, err)
		}
	}
	return nil
}
