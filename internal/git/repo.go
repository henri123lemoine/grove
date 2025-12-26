// Package git provides Git operations for worktree management.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Repo holds repository information.
type Repo struct {
	// Root is the current worktree root directory.
	Root string

	// MainWorktreeRoot is the main worktree root (where .worktrees should be created).
	// For bare repos, this is the git directory itself.
	MainWorktreeRoot string

	// GitDir is the path to the .git directory (the common git dir).
	GitDir string

	// IsBare indicates if this is a bare repository.
	IsBare bool

	// DefaultBranch is the default branch (main, master, etc).
	DefaultBranch string
}

// currentRepo caches the current repository info.
var (
	currentRepo *Repo
	repoMu      sync.RWMutex
)

// GetRepo returns the current repository information.
// It caches the result for subsequent calls.
func GetRepo() (*Repo, error) {
	repoMu.RLock()
	if currentRepo != nil {
		defer repoMu.RUnlock()
		return currentRepo, nil
	}
	repoMu.RUnlock()

	repoMu.Lock()
	defer repoMu.Unlock()

	// Double-check after acquiring write lock
	if currentRepo != nil {
		return currentRepo, nil
	}

	repo, err := detectRepo()
	if err != nil {
		return nil, err
	}
	currentRepo = repo
	return repo, nil
}

// ResetRepo clears the cached repository info.
func ResetRepo() {
	repoMu.Lock()
	defer repoMu.Unlock()
	currentRepo = nil
}

// UpdateDefaultBranch updates the repo's default branch using the specified remote.
// This should be called after config is loaded if a specific remote is configured.
func UpdateDefaultBranch(configuredRemote string) {
	repoMu.Lock()
	defer repoMu.Unlock()
	if currentRepo == nil {
		return
	}
	if configuredRemote != "" {
		currentRepo.DefaultBranch = detectDefaultBranchWithRemote(configuredRemote)
	}
}

// detectRepo detects the current Git repository.
func detectRepo() (*Repo, error) {
	// Get the git common directory (the actual .git dir, not worktree's .git file)
	gitDir, err := runGit("rev-parse", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	gitDir = strings.TrimSpace(gitDir)

	// Make gitDir absolute
	if !filepath.IsAbs(gitDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		gitDir = filepath.Join(cwd, gitDir)
	}
	gitDir = filepath.Clean(gitDir)

	// Check if bare repo
	isBareStr, err := runGit("rev-parse", "--is-bare-repository")
	if err != nil {
		return nil, err
	}
	isBare := strings.TrimSpace(isBareStr) == "true"

	// Get current worktree root
	var root string
	if isBare {
		root = gitDir
	} else {
		root, err = runGit("rev-parse", "--show-toplevel")
		if err != nil {
			return nil, err
		}
		root = strings.TrimSpace(root)
	}

	// Get main worktree root (where .worktrees should be created)
	var mainRoot string
	if isBare {
		mainRoot = gitDir
	} else {
		// For normal repos, the main worktree is the parent of the .git directory
		// For worktrees, gitDir points to the common .git dir
		mainRoot = filepath.Dir(gitDir)
	}

	// Get default branch
	defaultBranch := detectDefaultBranch()

	return &Repo{
		Root:             root,
		MainWorktreeRoot: mainRoot,
		GitDir:           gitDir,
		IsBare:           isBare,
		DefaultBranch:    defaultBranch,
	}, nil
}

// GetPrimaryRemote returns the primary remote name.
// If configuredRemote is non-empty, it's used directly.
// Otherwise, it tries to auto-detect:
// 1. If there's only one remote, use it
// 2. If "origin" exists, prefer it
// 3. Otherwise use the first remote alphabetically
func GetPrimaryRemote(configuredRemote string) string {
	if configuredRemote != "" {
		return configuredRemote
	}

	// Get list of remotes
	output, err := runGit("remote")
	if err != nil {
		return "origin" // fallback
	}

	remotes := strings.Fields(strings.TrimSpace(output))
	if len(remotes) == 0 {
		return "origin" // fallback for repos with no remotes
	}

	// If only one remote, use it
	if len(remotes) == 1 {
		return remotes[0]
	}

	// Prefer "origin" if it exists
	for _, r := range remotes {
		if r == "origin" {
			return "origin"
		}
	}

	// Otherwise use first one (already sorted alphabetically by git)
	return remotes[0]
}

// detectDefaultBranch tries to detect the default branch.
func detectDefaultBranch() string {
	return detectDefaultBranchWithRemote("")
}

// detectDefaultBranchWithRemote tries to detect the default branch using a specific remote.
func detectDefaultBranchWithRemote(configuredRemote string) string {
	remote := GetPrimaryRemote(configuredRemote)

	// Try to get from remote HEAD
	output, err := runGit("symbolic-ref", "refs/remotes/"+remote+"/HEAD")
	if err == nil {
		ref := strings.TrimSpace(output)
		prefix := "refs/remotes/" + remote + "/"
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}

	// Try common defaults
	for _, branch := range []string{"main", "master"} {
		_, err := runGit("rev-parse", "--verify", "refs/heads/"+branch)
		if err == nil {
			return branch
		}
	}

	// Fallback
	return "main"
}

// runGit executes a git command and returns the output.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}

// runGitInDir executes a git command in a specific directory.
func runGitInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}
