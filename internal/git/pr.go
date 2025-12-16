package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckGHAuth checks if gh CLI is installed and authenticated.
func CheckGHAuth() (bool, error) {
	// Check if gh is installed
	_, err := exec.LookPath("gh")
	if err != nil {
		return false, fmt.Errorf("gh CLI not found. Install it from https://cli.github.com")
	}

	// Check auth status
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("gh CLI not authenticated: %s", strings.TrimSpace(string(output)))
	}

	return true, nil
}

// HasUpstream checks if the branch has an upstream tracking branch.
func HasUpstream(worktreePath, branch string) bool {
	_, err := runGitInDir(worktreePath, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	return err == nil
}

// PushBranch pushes the branch to origin with upstream tracking.
func PushBranch(worktreePath, branch string) error {
	_, err := runGitInDir(worktreePath, "push", "-u", "origin", branch)
	if err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}
	return nil
}

// CreatePR creates a pull request using the configured command.
// It runs the command in the worktree directory via shell to properly
// handle quoted arguments like: gh pr create --title "My Title"
func CreatePR(worktreePath, command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty PR command")
	}

	// Run through shell to handle quotes properly
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = worktreePath

	// Run interactively - let gh handle its own I/O
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// RenameBranch renames a branch in the specified worktree.
func RenameBranch(worktreePath, oldName, newName string) error {
	_, err := runGitInDir(worktreePath, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}
	return nil
}

// GetStashCount returns the number of stashed entries for a worktree.
func GetStashCount(worktreePath string) (int, error) {
	output, err := runGitInDir(worktreePath, "stash", "list")
	if err != nil {
		return 0, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return 0, nil
	}

	lines := strings.Split(output, "\n")
	return len(lines), nil
}

// CreateStash creates a stash in the specified worktree.
func CreateStash(worktreePath, message string) (string, error) {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}

	output, err := runGitInDir(worktreePath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create stash: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// PopStash pops the most recent stash in the specified worktree.
func PopStash(worktreePath string) error {
	_, err := runGitInDir(worktreePath, "stash", "pop")
	if err != nil {
		return fmt.Errorf("failed to pop stash: %w", err)
	}
	return nil
}
