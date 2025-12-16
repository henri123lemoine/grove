// Package exec handles executing external commands.
package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
)

// Open executes the open command for a worktree.
func Open(command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	cfg := config.DefaultConfig()
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
func OpenDetached(command string, wt *git.Worktree) error {
	repo, err := git.GetRepo()
	if err != nil {
		return err
	}

	cfg := config.DefaultConfig()
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
func OpenWithConfig(cfg *config.Config, wt *git.Worktree) (bool, error) {
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

	// Apply layout if new window and layout is configured
	if isNewWindow && cfg.Open.Layout != "none" && cfg.Open.Layout != "" {
		applyLayout(cfg, wt, repo)
	}

	return isNewWindow, nil
}

// findWindowByPath finds a tmux window by pane path.
func findWindowByPath(path string) string {
	// Check if we're in tmux
	if os.Getenv("TMUX") == "" {
		return ""
	}

	// List windows with their pane paths
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_id} #{pane_current_path}")
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

// applyLayout applies the configured layout after creating a new window.
func applyLayout(cfg *config.Config, wt *git.Worktree, repo *git.Repo) error {
	if os.Getenv("TMUX") == "" {
		return nil
	}

	var layoutCmd string
	switch cfg.Open.Layout {
	case "dev":
		// Default dev layout: split horizontally 50/50
		layoutCmd = "tmux split-window -h -p 50 -c " + wt.Path
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
