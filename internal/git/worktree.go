package git

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/henri123lemoine/grove/internal/debug"
)

// maxWorkers limits concurrent goroutines for worktree enrichment.
// Uses number of CPUs as a reasonable default.
var maxWorkers = runtime.NumCPU()

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
	defer debug.Timed("git.List")()

	repo, err := GetRepo()
	if err != nil {
		return nil, err
	}

	// Get porcelain output - run from repo root to handle case where CWD was deleted
	output, err := runGitInDir(repo.MainWorktreeRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees := parseWorktreeList(output)

	// Get current working directory to identify current worktree
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	// Phase 1: Enrich with fast, non-git status (single-threaded to avoid races)
	// Resolve cwd once for all comparisons
	cwdPath := ""
	if cwd != "" {
		cwdPath = ResolvePath(cwd)
	}
	mainWorktreePath := ResolvePath(repo.MainWorktreeRoot)

	for i := range worktrees {
		wt := &worktrees[i]
		wtPath := ResolvePath(wt.Path)

		// Check if this is the current worktree (fast, no git call)
		if cwdPath != "" {
			wt.IsCurrent = wtPath == cwdPath || strings.HasPrefix(cwdPath, wtPath+string(filepath.Separator))
		}

		// Check if this is the main worktree (fast, no git call)
		wt.IsMain = wtPath == mainWorktreePath || (repo.IsBare && i == 0)
	}

	// Phase 2: Parallelize git operations (after all single-threaded writes complete)
	// Use a semaphore to limit concurrent goroutines
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for i := range worktrees {
		wt := &worktrees[i]
		wg.Add(1)
		go func(wt *Worktree) {
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			defer wg.Done()
			enrichWorktree(wt, repo)
		}(wt)
	}
	wg.Wait()

	return worktrees, nil
}

// enrichWorktree populates a worktree with essential status information.
// Only fetches dirty status for maximum speed. Other info is lazy-loaded.
func enrichWorktree(wt *Worktree, _ *Repo) {
	// Get dirty status - essential for list view (1 git command)
	wt.IsDirty, wt.DirtyFiles, _ = GetDirtyStatus(wt.Path)

	// NOTE: Upstream, last commit, merged status, and unique commits are
	// fetched on-demand to speed up initial load.
}

// EnrichWorktreesUpstream fetches ahead/behind status for all worktrees.
// Run this in background after initial load for progressive enhancement.
func EnrichWorktreesUpstream(worktrees []Worktree) {
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for i := range worktrees {
		wt := &worktrees[i]
		if wt.Branch != "" && !wt.IsDetached {
			wg.Add(1)
			go func(wt *Worktree) {
				sem <- struct{}{}        // acquire
				defer func() { <-sem }() // release
				defer wg.Done()
				wt.Ahead, wt.Behind, wt.HasUpstream, _ = GetUpstreamStatus(wt.Path, wt.Branch)
			}(wt)
		}
	}
	wg.Wait()
}

// EnrichWorktreeDetail fetches additional info for detail panel display.
// Called lazily when user opens detail view.
func EnrichWorktreeDetail(wt *Worktree) {
	if wt.LastCommitHash == "" {
		wt.LastCommitHash, wt.LastCommitMessage, wt.LastCommitTime, _ = GetLastCommit(wt.Path)
	}
}

// EnrichWorktreeSafety fetches safety-related info for delete operations.
// Called lazily when user initiates delete.
func EnrichWorktreeSafety(wt *Worktree, defaultBranch string) {
	if wt.Branch == "" || wt.IsMain || wt.IsDetached || defaultBranch == "" {
		return
	}

	if wt.Branch != defaultBranch {
		wt.IsMerged, _ = IsBranchMerged(wt.Branch, defaultBranch)
		commits, err := GetUniqueCommits(wt.Branch, defaultBranch)
		if err == nil {
			wt.UniqueCommits = len(commits)
		}
	} else {
		wt.IsMerged = true
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
	repo, err := GetRepo()
	if err != nil {
		return err
	}

	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("path already exists: %s (try a different branch name or delete the existing directory)", path)
	}

	// Ensure the worktree directory is excluded from git tracking
	ensureWorktreeDirExcluded(path, repo)

	// Prune stale worktree entries to avoid conflicts with recently deleted worktrees
	_, _ = runGitInDir(repo.MainWorktreeRoot, "worktree", "prune")

	if err := checkCreateConflicts(path, branch); err != nil {
		return err
	}

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

	_, err = runGitInDir(repo.MainWorktreeRoot, args...)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// checkCreateConflicts validates worktree creation against existing worktrees.
func checkCreateConflicts(path, branch string) error {
	worktrees, err := List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees for preflight: %w", err)
	}

	targetPath := ResolvePath(path)
	for _, wt := range worktrees {
		wtPath := ResolvePath(wt.Path)

		if wtPath == targetPath {
			return fmt.Errorf("worktree path already registered: %s (run prune or delete the existing worktree)", wt.Path)
		}

		if branch != "" && !wt.IsDetached && wt.Branch == branch {
			return fmt.Errorf("branch %q is already checked out at %s", branch, wt.Path)
		}

		if wt.IsMain {
			continue
		}

		if isWithinPath(wtPath, targetPath) {
			return fmt.Errorf("target path %s is inside existing worktree %s", path, wt.Path)
		}

		if isWithinPath(targetPath, wtPath) {
			return fmt.Errorf("target path %s contains existing worktree %s", path, wt.Path)
		}
	}

	return nil
}

// Remove removes a worktree.
func Remove(path string, force bool) error {
	repo, err := GetRepo()
	if err != nil {
		return err
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	_, err = runGitInDir(repo.MainWorktreeRoot, args...)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Clean up empty parent directories (rmdir-like behavior)
	cleanupEmptyParentDirs(path)

	// Immediately update the cache so it reflects the deletion.
	// This is important because if the window is closed after deletion,
	// we want the next grove instance to see the correct state.
	_, _ = ListAndCache()

	return nil
}

// ensureWorktreeDirExcluded adds the worktree directory to .git/info/exclude
// if it's not already there. This prevents worktrees from showing as untracked files.
func ensureWorktreeDirExcluded(worktreePath string, repo *Repo) {
	// Get the worktree directory (parent of the actual worktree path)
	// e.g., if path is "/repo/.worktrees/feature", we want ".worktrees"
	relPath, err := filepath.Rel(repo.MainWorktreeRoot, worktreePath)
	if err != nil {
		return
	}

	// Get the first directory component (the worktree container dir)
	parts := strings.SplitN(relPath, string(filepath.Separator), 2)
	if len(parts) == 0 || parts[0] == "" || parts[0] == "." {
		return
	}
	worktreeDir := parts[0]

	// Don't add if it starts with ".." (worktrees outside repo)
	if strings.HasPrefix(worktreeDir, "..") {
		return
	}

	excludePath := filepath.Join(repo.GitDir, "info", "exclude")

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	// Check if already excluded (with or without trailing slash)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == worktreeDir || trimmed == worktreeDir+"/" {
			return // Already excluded
		}
	}

	// Ensure info directory exists
	infoDir := filepath.Join(repo.GitDir, "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return
	}

	// Append to exclude file
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	// Add newline before entry if file doesn't end with one
	entry := worktreeDir + "/\n"
	if len(content) > 0 && content[len(content)-1] != '\n' {
		entry = "\n" + entry
	}

	_, _ = f.WriteString(entry)
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

	// Clean up empty parent directories up to .worktrees
	for parent != worktreesDir && strings.HasPrefix(parent, worktreesDir) {
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
	// Resolve source directory for path traversal validation
	resolvedSource := ResolvePath(sourceDir)

	for _, pattern := range patterns {
		// Find files matching pattern
		matches, err := filepath.Glob(filepath.Join(sourceDir, pattern))
		if err != nil {
			continue
		}

		for _, srcPath := range matches {
			// Validate path is within source directory (prevent path traversal)
			resolvedSrc := ResolvePath(srcPath)
			if !isWithinPath(resolvedSource, resolvedSrc) {
				continue
			}

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
		// Handle ** patterns (e.g., node_modules/**)
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator)) {
				return true
			}
		}

		// Standard filepath.Match against full path
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
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

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

// Prune removes stale worktree entries (worktrees that no longer exist on disk).
// Returns the number of pruned entries.
func Prune() (int, error) {
	repo, err := GetRepo()
	if err != nil {
		return 0, err
	}

	// Get current worktrees to count before
	beforeOutput, _ := runGitInDir(repo.MainWorktreeRoot, "worktree", "list", "--porcelain")
	beforeCount := countWorktrees(beforeOutput)

	// Run prune
	_, err = runGitInDir(repo.MainWorktreeRoot, "worktree", "prune")
	if err != nil {
		return 0, fmt.Errorf("failed to prune worktrees: %w", err)
	}

	// Get count after
	afterOutput, _ := runGitInDir(repo.MainWorktreeRoot, "worktree", "list", "--porcelain")
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

// ResolvePath returns the absolute path with symlinks resolved.
// Falls back to absolute path if symlink resolution fails.
func ResolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

// isWithinPath reports whether child is inside parent (or equal).
func isWithinPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
