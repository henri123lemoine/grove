package git

import (
	"fmt"
	"strconv"
	"strings"
)

// StashEntry represents a single git stash entry.
type StashEntry struct {
	Index   int
	Message string
}

// ListStashes returns the list of stashes for the repository.
func ListStashes(worktreePath string) ([]StashEntry, error) {
	output, err := runGitInDir(worktreePath, "stash", "list")
	if err != nil {
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return []StashEntry{}, nil
	}

	lines := strings.Split(output, "\n")
	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		ref := parts[0]
		msg := ""
		if len(parts) > 1 {
			msg = parts[1]
		}

		start := strings.Index(ref, "{")
		end := strings.Index(ref, "}")
		if start == -1 || end == -1 || end <= start+1 {
			continue
		}
		idx, err := strconv.Atoi(ref[start+1 : end])
		if err != nil {
			continue
		}

		entries = append(entries, StashEntry{Index: idx, Message: msg})
	}

	return entries, nil
}

// CreateStash saves current changes to a new stash entry.
func CreateStash(worktreePath, message string) (string, error) {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}

	output, err := runGitInDir(worktreePath, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// PopStashAt applies and drops the stash entry at the given index.
func PopStashAt(worktreePath string, index int) error {
	_, err := runGitInDir(worktreePath, "stash", "pop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// ApplyStash applies the stash entry at the given index without dropping it.
func ApplyStash(worktreePath string, index int) error {
	_, err := runGitInDir(worktreePath, "stash", "apply", fmt.Sprintf("stash@{%d}", index))
	return err
}

// DropStash removes the stash entry at the given index.
func DropStash(worktreePath string, index int) error {
	_, err := runGitInDir(worktreePath, "stash", "drop", fmt.Sprintf("stash@{%d}", index))
	return err
}
