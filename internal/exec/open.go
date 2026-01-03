// Package exec handles executing external commands.
package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/henri123lemoine/grove/internal/config"
	"github.com/henri123lemoine/grove/internal/git"
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

	backend := Backend()

	// Check for existing window based on config
	isNewWindow := true
	switch cfg.Open.DetectExisting {
	case "path":
		windowID := backend.FindWindowByPath(wt.Path)
		if windowID != "" {
			err := backend.SwitchToWindow(windowID)
			return false, err
		}
	case "name":
		windowName := wt.BranchShort()
		if cfg.Open.WindowNameStyle == "full" {
			windowName = wt.Branch
		}
		windowID := backend.FindWindowByName(windowName)
		if windowID != "" {
			err := backend.SwitchToWindow(windowID)
			return false, err
		}
	case "none":
		// Always create new window
	}

	// Get the open command - use config if set, otherwise auto-detect
	openCommand := cfg.Open.Command
	if openCommand == "" {
		openCommand = backend.DefaultOpenCommand()
		if openCommand == "" {
			return false, fmt.Errorf("no terminal multiplexer detected and no open command configured")
		}
	}

	// Expand and run the open command
	expanded := expandTemplate(openCommand, wt, repo, cfg)

	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdin = nil

	// Capture stderr for better error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return false, fmt.Errorf("open command failed: %w: %s", err, strings.TrimSpace(errMsg))
		}
		return false, fmt.Errorf("open command failed: %w", err)
	}

	// Apply named layout if provided
	if isNewWindow && layout != nil {
		if err := backend.ApplyNamedLayout(layout, wt, repo, cfg); err != nil {
			return isNewWindow, fmt.Errorf("window opened but layout failed: %w", err)
		}
	}

	return isNewWindow, nil
}

// WindowExistsFor checks if a window already exists for the given worktree
// based on the config's detect_existing setting.
func WindowExistsFor(cfg *config.Config, wt *git.Worktree) bool {
	backend := Backend()
	switch cfg.Open.DetectExisting {
	case "path":
		return backend.FindWindowByPath(wt.Path) != ""
	case "name":
		windowName := wt.BranchShort()
		if cfg.Open.WindowNameStyle == "full" {
			windowName = wt.Branch
		}
		return backend.FindWindowByName(windowName) != ""
	default:
		return false
	}
}

// expandTemplate expands template variables in the command.
func expandTemplate(command string, wt *git.Worktree, repo *git.Repo, cfg *config.Config) string {
	result := command

	branch := wt.Branch
	branchShort := wt.BranchShort()
	repoName := filepath.Base(repo.Root)
	windowName := branchShort
	if cfg != nil && cfg.Open.WindowNameStyle == "full" {
		windowName = branch
	}

	replacements := []struct {
		key   string
		value string
	}{
		{"{path}", shellQuote(wt.Path)},
		{"{branch}", shellQuote(branch)},
		{"{branch_short}", shellQuote(branchShort)},
		{"{repo}", shellQuote(repoName)},
		{"{window_name}", shellQuote(windowName)},
	}

	for _, repl := range replacements {
		result = strings.ReplaceAll(result, repl.key, repl.value)
	}

	return result
}

// shellQuote returns a shell-safe quoted string.
// Uses single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	// If the string has no special characters, no need to quote
	if !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}()&|;<>") {
		return s
	}
	// Use single quotes, escaping embedded single quotes as '\''
	escaped := strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// EchoPath is a simple open command that just echoes the path.
// Useful for shell integration.
func EchoPath(wt *git.Worktree) string {
	return wt.Path
}

// Multiplexer represents the type of terminal multiplexer.
// Deprecated: Use Backend() interface instead for new code.
type Multiplexer int

const (
	MultiplexerNone Multiplexer = iota
	MultiplexerTmux
	MultiplexerZellij
)

// GetMultiplexer detects the current terminal multiplexer.
// Deprecated: Use Backend() instead.
func GetMultiplexer() Multiplexer {
	b := Backend()
	switch b.Name() {
	case "tmux":
		return MultiplexerTmux
	case "zellij":
		return MultiplexerZellij
	default:
		return MultiplexerNone
	}
}

// GetDefaultOpenCommand returns the default open command for the current multiplexer.
func GetDefaultOpenCommand() string {
	return Backend().DefaultOpenCommand()
}

// Name returns a human-readable name for the multiplexer.
func (m Multiplexer) Name() string {
	switch m {
	case MultiplexerTmux:
		return "tmux"
	case MultiplexerZellij:
		return "zellij"
	default:
		return ""
	}
}

// WindowName returns the term used for windows/tabs in this multiplexer.
func (m Multiplexer) WindowName() string {
	switch m {
	case MultiplexerTmux:
		return "window"
	case MultiplexerZellij:
		return "tab"
	default:
		return "window"
	}
}

// FindWindowsForPath finds all windows/tabs that have panes in the given path.
func FindWindowsForPath(path string) []string {
	return Backend().FindWindowsForPath(path)
}

// CloseWindow closes a window/tab by ID.
func CloseWindow(windowID string) error {
	return Backend().CloseWindow(windowID)
}

// InMultiplexer returns true if we're running inside a supported multiplexer.
func InMultiplexer() bool {
	return Backend().Name() != ""
}
