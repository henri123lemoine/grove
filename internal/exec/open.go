// Package exec handles executing external commands.
package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/henrilemoine/grove/internal/git"
)

// Open executes the open command for a worktree.
func Open(command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	// Expand template variables
	expanded := expandTemplate(command, wt, repo)

	// Execute via shell
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// OpenDetached executes the open command in a detached process.
// This is useful for commands that should outlive grove.
func OpenDetached(command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	// Expand template variables
	expanded := expandTemplate(command, wt, repo)

	// Execute via shell
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process but don't wait for it
	return cmd.Start()
}

// expandTemplate expands template variables in the command.
func expandTemplate(command string, wt *git.Worktree, repo *git.Repo) string {
	result := command

	// {path} - Full path to worktree
	result = strings.ReplaceAll(result, "{path}", wt.Path)

	// {branch} - Full branch name
	result = strings.ReplaceAll(result, "{branch}", wt.Branch)

	// {branch_short} - Short branch name (after last /)
	result = strings.ReplaceAll(result, "{branch_short}", wt.BranchShort())

	// {repo} - Repository name (directory name)
	repoName := filepath.Base(repo.Root)
	result = strings.ReplaceAll(result, "{repo}", repoName)

	return result
}

// EchoPath is a simple open command that just echoes the path.
// Useful for shell integration.
func EchoPath(wt *git.Worktree) string {
	return wt.Path
}
