// Package exec handles executing external commands.
package exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
)

// Open executes the open command for a worktree.
func Open(cfg *config.Config, command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	// Expand template variables
	expanded := expandTemplate(command, wt, repo, cfg)

	// Execute via shell
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// OpenDetached executes the open command in a detached process.
// This is useful for commands that should outlive grove.
func OpenDetached(cfg *config.Config, command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	// Expand template variables
	expanded := expandTemplate(command, wt, repo, cfg)

	// Execute via shell
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process but don't wait for it
	return cmd.Start()
}

// OpenWithConfig executes the open command with full config support.
// Returns true if a new window was created (vs switching to existing).
func OpenWithConfig(cfg *config.Config, wt *git.Worktree, layout *config.LayoutConfig) (bool, error) {
	repo, err := git.GetRepo()
	if err != nil {
		return false, err
	}

	// Check for existing window based on config
	isNewWindow := true
	switch cfg.Open.DetectExisting {
	case "path":
		windowID := findWindowByPath(wt.Path)
		if windowID != "" {
			// Switch to existing window
			err := switchToWindow(windowID)
			return false, err
		}
	case "name":
		// Default behavior - let the command handle it
		// Most commands like "tmux select-window -t :name || new-window" do this
	case "none":
		// Always create new window
	}

	// Expand and run the open command
	expanded := expandTemplate(cfg.Open.Command, wt, repo, cfg)

	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return false, err
	}

	// Apply named layout if provided
	if isNewWindow && layout != nil {
		// Small delay to ensure tmux window is ready
		time.Sleep(100 * time.Millisecond)
		// Layout errors are non-fatal - we still opened the window successfully
		_ = applyNamedLayout(layout, wt, repo, cfg)
	} else if isNewWindow && cfg.Open.Layout != "none" && cfg.Open.Layout != "" {
		// Fall back to legacy layout system
		_ = applyLayout(cfg, wt, repo)
	}

	return isNewWindow, nil
}

// findWindowByPath finds a tmux window by checking all panes across all windows.
func findWindowByPath(path string) string {
	// Check if we're in tmux
	if os.Getenv("TMUX") == "" {
		return ""
	}

	// List ALL panes across ALL windows
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{window_id} #{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Find window with matching path
	absPath, _ := filepath.Abs(path)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			windowID := parts[0]
			panePath := parts[1]
			// Check for exact match or if pane is within the worktree
			if panePath == absPath || strings.HasPrefix(panePath, absPath+string(filepath.Separator)) {
				return windowID
			}
		}
	}

	return ""
}

// switchToWindow switches to a tmux window by ID.
func switchToWindow(windowID string) error {
	cmd := exec.Command("tmux", "select-window", "-t", windowID)
	return cmd.Run()
}

// applyLayout applies the configured layout after creating a new window (legacy system).
func applyLayout(cfg *config.Config, wt *git.Worktree, repo *git.Repo) error {
	inTmux := os.Getenv("TMUX") != ""
	inZellij := os.Getenv("ZELLIJ") != ""

	// No layout support outside of multiplexers
	if !inTmux && !inZellij {
		return nil
	}

	var layoutCmd string
	switch cfg.Open.Layout {
	case "dev":
		// Default dev layout: split horizontally 50/50
		if inTmux {
			layoutCmd = "tmux split-window -h -p 50 -c " + wt.Path
		} else if inZellij {
			layoutCmd = "zellij action new-pane --direction right --cwd " + wt.Path
		}
	case "custom":
		if cfg.Open.LayoutCommand != "" {
			layoutCmd = expandTemplate(cfg.Open.LayoutCommand, wt, repo, cfg)
		}
	}

	if layoutCmd == "" {
		return nil
	}

	cmd := exec.Command("sh", "-c", layoutCmd)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// applyNamedLayout applies a named layout with multiple panes.
func applyNamedLayout(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error {
	// Only tmux is supported
	if os.Getenv("TMUX") == "" {
		return nil
	}

	if len(layout.Panes) == 0 {
		return nil
	}

	// Track pane IDs as we create them
	// Pane 0 is the initial pane (already exists in the new window)
	paneIDs := make([]string, len(layout.Panes))

	// Get the current pane ID (pane 0)
	cmd := exec.Command("tmux", "display-message", "-p", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current pane ID: %w", err)
	}
	paneIDs[0] = strings.TrimSpace(string(output))

	// Run command for pane 0 if specified
	if len(layout.Panes) > 0 && layout.Panes[0].Command != "" {
		expandedCmd := expandTemplate(layout.Panes[0].Command, wt, repo, cfg)
		sendCmd := exec.Command("tmux", "send-keys", "-t", paneIDs[0], expandedCmd, "Enter")
		_ = sendCmd.Run()
	}

	// Create additional panes
	for i := 1; i < len(layout.Panes); i++ {
		pane := layout.Panes[i]

		// Validate split_from reference
		if pane.SplitFrom < 0 || pane.SplitFrom >= i {
			continue // Skip invalid pane
		}

		targetPane := paneIDs[pane.SplitFrom]
		if targetPane == "" {
			continue // Skip if target pane doesn't exist
		}

		// Build split command
		splitArgs := []string{"split-window"}

		// Direction
		switch pane.Direction {
		case "right":
			splitArgs = append(splitArgs, "-h")
		case "left":
			splitArgs = append(splitArgs, "-hb")
		case "down":
			splitArgs = append(splitArgs, "-v")
		case "up":
			splitArgs = append(splitArgs, "-vb")
		default:
			splitArgs = append(splitArgs, "-h") // Default to right
		}

		// Size (percentage)
		if pane.Size > 0 && pane.Size < 100 {
			splitArgs = append(splitArgs, "-p", fmt.Sprintf("%d", pane.Size))
		}

		// Target pane
		splitArgs = append(splitArgs, "-t", targetPane)

		// Working directory
		splitArgs = append(splitArgs, "-c", wt.Path)

		// Print new pane ID
		splitArgs = append(splitArgs, "-P", "-F", "#{pane_id}")

		// Execute split
		splitCmd := exec.Command("tmux", splitArgs...)
		splitOutput, err := splitCmd.Output()
		if err != nil {
			continue // Skip this pane on error
		}

		newPaneID := strings.TrimSpace(string(splitOutput))
		paneIDs[i] = newPaneID

		// Run command in new pane if specified
		if pane.Command != "" {
			expandedCmd := expandTemplate(pane.Command, wt, repo, cfg)
			sendCmd := exec.Command("tmux", "send-keys", "-t", newPaneID, expandedCmd, "Enter")
			_ = sendCmd.Run()
		}

		// Small delay between pane creations
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// expandTemplate expands template variables in the command.
func expandTemplate(command string, wt *git.Worktree, repo *git.Repo, cfg *config.Config) string {
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

	// {window_name} - Window name based on style config
	windowName := wt.BranchShort()
	if cfg != nil && cfg.Open.WindowNameStyle == "full" {
		windowName = wt.Branch
	}
	result = strings.ReplaceAll(result, "{window_name}", windowName)

	return result
}

// EchoPath is a simple open command that just echoes the path.
// Useful for shell integration.
func EchoPath(wt *git.Worktree) string {
	return wt.Path
}
