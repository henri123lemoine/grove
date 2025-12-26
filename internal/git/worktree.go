package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	HasUpstream bool // True if branch has upstream tracking configured
	Ahead       int
	Behind      int

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

	// Enrich with status information (parallelized for performance)
	var wg sync.WaitGroup
	for i := range worktrees {
		wt := &worktrees[i]

		// Check if this is the current worktree (fast, no git call)
		if cwd != "" {
			wtPath, _ := filepath.Abs(wt.Path)
			cwdPath, _ := filepath.Abs(cwd)
			wt.IsCurrent = wtPath == cwdPath || strings.HasPrefix(cwdPath, wtPath+string(filepath.Separator))
		}

		// Check if this is the main worktree (fast, no git call)
		wt.IsMain = wt.Path == repo.MainWorktreeRoot || (repo.IsBare && i == 0)

		// Parallelize git operations
		wg.Add(1)
		go func(wt *Worktree) {
			defer wg.Done()
			enrichWorktree(wt, repo)
		}(wt)
	}
	wg.Wait()

	return worktrees, nil
}

// enrichWorktree populates a worktree with status information from git.
func enrichWorktree(wt *Worktree, repo *Repo) {
	// Get dirty status
	wt.IsDirty, wt.DirtyFiles, _ = GetDirtyStatus(wt.Path)

	// Get upstream status (skip for detached HEAD - no tracking branch)
	if wt.Branch != "" && !wt.IsDetached {
		wt.Ahead, wt.Behind, wt.HasUpstream, _ = GetUpstreamStatus(wt.Path, wt.Branch)
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

	// Clean up empty parent directories (rmdir-like behavior)
	cleanupEmptyParentDirs(path)

	return nil
}

// cleanupEmptyParentDirs removes empty parent directories up the tree,
// stopping at .worktrees or any non-empty directory.
func cleanupEmptyParentDirs(path string) {
	repo, err := GetRepo()
	if err != nil {
		return
	}

	worktreesDir := filepath.Join(repo.MainWorktreeRoot, ".worktrees")
	parent := filepath.Dir(path)

	for {
		// Stop if we've reached or gone past .worktrees
		if parent == worktreesDir || !strings.HasPrefix(parent, worktreesDir) {
			break
		}

		// Check if directory is empty
		entries, err := os.ReadDir(parent)
		if err != nil {
			break
		}

		if len(entries) > 0 {
			// Directory not empty, stop
			break
		}

		// Remove empty directory
		if err := os.Remove(parent); err != nil {
			break
		}

		// Move to parent
		parent = filepath.Dir(parent)
	}
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
// Note: Commands run without stdin access since grove is a TUI application.
// Use non-interactive commands (e.g., "npm install --yes" instead of "npm install").
// timeoutSeconds of 0 means no timeout.
func RunPostCreateHooks(worktreePath string, commands []string, timeoutSeconds int) error {
	for _, cmdStr := range commands {
		if err := runHookCommand(worktreePath, cmdStr, timeoutSeconds); err != nil {
			return err
		}
	}
	return nil
}

// Prune removes stale worktree entries (worktrees that no longer exist on disk).
// Returns the number of pruned entries.
func Prune() (int, error) {
	// Get current worktrees to count before
	beforeOutput, _ := runGit("worktree", "list", "--porcelain")
	beforeCount := countWorktrees(beforeOutput)

	// Run prune
	_, err := runGit("worktree", "prune")
	if err != nil {
		return 0, fmt.Errorf("failed to prune worktrees: %w", err)
	}

	// Get count after
	afterOutput, _ := runGit("worktree", "list", "--porcelain")
	afterCount := countWorktrees(afterOutput)

	return beforeCount - afterCount, nil
}

// countWorktrees counts the number of worktrees from porcelain output.
func countWorktrees(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			count++
		}
	}
	return count
}

// runHookCommand runs a single hook command with optional timeout.
func runHookCommand(worktreePath, cmdStr string, timeoutSeconds int) error {
	var ctx context.Context
	var cancel context.CancelFunc

	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = worktreePath

	// Capture output for better error messages
	// stdin is nil - interactive commands won't work
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))

		// Check if it was a timeout (context deadline exceeded)
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("post-create command timed out after %ds: %s", timeoutSeconds, cmdStr)
		}

		if outputStr != "" {
			return fmt.Errorf("post-create command failed: %s: %w\nOutput: %s", cmdStr, err, outputStr)
		}
		return fmt.Errorf("post-create command failed: %s: %w", cmdStr, err)
	}
	return nil
}
