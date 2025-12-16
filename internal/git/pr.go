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

// PushBranch pushes the branch to the specified remote with upstream tracking.
// If remote is empty, it will auto-detect the primary remote.
func PushBranch(worktreePath, branch, remote string) error {
	targetRemote := GetPrimaryRemote(remote)
	_, err := runGitInDir(worktreePath, "push", "-u", targetRemote, branch)
	if err != nil {
		return fmt.Errorf("failed to push branch to %s: %w", targetRemote, err)
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

// StashEntry represents a single stash entry.
type StashEntry struct {
	Index   int    // 0 for stash@{0}, 1 for stash@{1}, etc.
	Message string // The stash message
}

// ListStashes returns all stash entries for a worktree.
func ListStashes(worktreePath string) ([]StashEntry, error) {
	output, err := runGitInDir(worktreePath, "stash", "list", "--format=%gd %gs")
	if err != nil {
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var entries []StashEntry
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		// Parse "stash@{0} message here"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}
		// Extract index from stash@{N}
		indexStr := parts[0]
		var index int
		_, _ = fmt.Sscanf(indexStr, "stash@{%d}", &index)

		msg := ""
		if len(parts) == 2 {
			msg = parts[1]
		}
		entries = append(entries, StashEntry{Index: index, Message: msg})
	}
	return entries, nil
}

// ApplyStash applies a stash without removing it.
func ApplyStash(worktreePath string, index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := runGitInDir(worktreePath, "stash", "apply", stashRef)
	if err != nil {
		return fmt.Errorf("failed to apply stash: %w", err)
	}
	return nil
}

// DropStash removes a stash entry.
func DropStash(worktreePath string, index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := runGitInDir(worktreePath, "stash", "drop", stashRef)
	if err != nil {
		return fmt.Errorf("failed to drop stash: %w", err)
	}
	return nil
}

// PopStashAt pops a specific stash entry.
func PopStashAt(worktreePath string, index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := runGitInDir(worktreePath, "stash", "pop", stashRef)
	if err != nil {
		return fmt.Errorf("failed to pop stash: %w", err)
	}
	return nil
}
