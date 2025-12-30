// Package exec handles executing external commands.
package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		// Check if a window with this name already exists
		windowName := wt.BranchShort()
		if cfg.Open.WindowNameStyle == "full" {
			windowName = wt.Branch
		}
		windowID := findWindowByName(windowName)
		if windowID != "" {
			// Switch to existing window
			err := switchToWindow(windowID)
			return false, err
		}
	case "none":
		// Always create new window
	}

	// Get the open command - use config if set, otherwise auto-detect
	openCommand := cfg.Open.Command
	if openCommand == "" {
		openCommand = GetDefaultOpenCommand()
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

	// Use Run() instead of Start() to wait for completion and catch errors
	// This is safe because tmux/zellij commands complete quickly (they just
	// create the window/tab and return, they don't wait for the shell)
	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return false, fmt.Errorf("open command failed: %w: %s", err, strings.TrimSpace(errMsg))
		}
		return false, fmt.Errorf("open command failed: %w", err)
	}

	// Apply named layout if provided
	if isNewWindow && layout != nil {
		if err := applyNamedLayout(layout, wt, repo, cfg); err != nil {
			return isNewWindow, fmt.Errorf("window opened but layout failed: %w", err)
		}
	} else if isNewWindow && cfg.Open.Layout != "none" && cfg.Open.Layout != "" {
		if err := applyLayout(cfg, wt, repo); err != nil {
			return isNewWindow, fmt.Errorf("window opened but layout failed: %w", err)
		}
	}

	return isNewWindow, nil
}

// findWindowByPath finds a window/tab by checking pane paths.
// Returns window/tab ID or empty string if not found.
func findWindowByPath(path string) string {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		return findTmuxWindowByPath(path)
	case MultiplexerZellij:
		// Zellij doesn't expose pane CWDs, so we can't detect by path.
		// Fall back to name-based detection using directory name.
		return findZellijTabByName(filepath.Base(path))
	default:
		return ""
	}
}

// findTmuxWindowByPath finds a tmux window by checking all panes.
func findTmuxWindowByPath(path string) string {
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{window_id} #{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	resolvedPath := resolvePath(path)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			windowID := parts[0]
			panePath := resolvePath(parts[1])
			if panePath == resolvedPath || strings.HasPrefix(panePath, resolvedPath+string(filepath.Separator)) {
				return windowID
			}
		}
	}
	return ""
}

// switchToWindow switches to a window/tab by ID.
func switchToWindow(windowID string) error {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		cmd := exec.Command("tmux", "select-window", "-t", windowID)
		return cmd.Run()
	case MultiplexerZellij:
		cmd := exec.Command("zellij", "action", "go-to-tab", windowID)
		return cmd.Run()
	default:
		return nil
	}
}

// findWindowByName finds a window/tab by name and returns its ID.
func findWindowByName(name string) string {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		return findTmuxWindowByName(name)
	case MultiplexerZellij:
		return findZellijTabByName(name)
	default:
		return ""
	}
}

// findTmuxWindowByName finds a tmux window by name.
func findTmuxWindowByName(name string) string {
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_id} #{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) == name {
			return parts[0]
		}
	}
	return ""
}

// findZellijTabByName finds a zellij tab by name and returns its 1-based index.
func findZellijTabByName(name string) string {
	cmd := exec.Command("zellij", "action", "query-tab-names")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == name {
			return fmt.Sprintf("%d", i+1) // Zellij uses 1-based indices
		}
	}
	return ""
}

// WindowExistsFor checks if a window already exists for the given worktree
// based on the config's detect_existing setting.
func WindowExistsFor(cfg *config.Config, wt *git.Worktree) bool {
	switch cfg.Open.DetectExisting {
	case "path":
		return findWindowByPath(wt.Path) != ""
	case "name":
		windowName := wt.BranchShort()
		if cfg.Open.WindowNameStyle == "full" {
			windowName = wt.Branch
		}
		return findWindowByName(windowName) != ""
	default:
		return false
	}
}

// applyLayout applies the configured layout after creating a new window (legacy system).
func applyLayout(cfg *config.Config, wt *git.Worktree, repo *git.Repo) error {
	inTmux := os.Getenv("TMUX") != ""
	inZellij := os.Getenv("ZELLIJ") != ""

	// No layout support outside of multiplexers
	if !inTmux && !inZellij {
		return nil
	}

	// Determine window name to target (the newly created window)
	windowName := wt.BranchShort()
	if cfg.Open.WindowNameStyle == "full" {
		windowName = wt.Branch
	}

	var layoutCmd string
	switch cfg.Open.Layout {
	case "dev":
		// Default dev layout: split horizontally 50/50
		if inTmux {
			// Target the new window by name to split the correct window
			layoutCmd = "tmux split-window -h -p 50 -t " + shellQuote(windowName) + " -c " + shellQuote(wt.Path)
		} else if inZellij {
			layoutCmd = "zellij action new-pane --direction right --cwd " + shellQuote(wt.Path)
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
	if len(layout.Panes) == 0 {
		return nil
	}

	switch GetMultiplexer() {
	case MultiplexerTmux:
		return applyNamedLayoutTmux(layout, wt, repo, cfg)
	case MultiplexerZellij:
		return applyNamedLayoutZellij(layout, wt, repo, cfg)
	default:
		return nil
	}
}

// applyNamedLayoutTmux applies a named layout in tmux.
func applyNamedLayoutTmux(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error {
	// Determine window name to target (the newly created window)
	windowName := wt.BranchShort()
	if cfg.Open.WindowNameStyle == "full" {
		windowName = wt.Branch
	}

	// Track pane IDs as we create them
	// Pane 0 is the initial pane (already exists in the new window)
	paneIDs := make([]string, len(layout.Panes))

	// Get the pane ID of the newly created window (target by name)
	// We need to find the first pane in the window with the matching name
	cmd := exec.Command("tmux", "list-panes", "-t", windowName, "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get pane ID for window %s: %w", windowName, err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return fmt.Errorf("no panes found in window %s", windowName)
	}
	paneIDs[0] = lines[0]

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

		// Working directory (use unquoted path here since it's passed directly to tmux, not through shell)
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

// applyNamedLayoutZellij applies a named layout in zellij.
func applyNamedLayoutZellij(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error {
	// Run command for pane 0 if specified
	if len(layout.Panes) > 0 && layout.Panes[0].Command != "" {
		expandedCmd := expandTemplate(layout.Panes[0].Command, wt, repo, cfg)
		// Use write-chars to type the command, then write 0d (Enter key)
		writeCmd := exec.Command("zellij", "action", "write-chars", expandedCmd)
		_ = writeCmd.Run()
		enterCmd := exec.Command("zellij", "action", "write", "10") // 10 = newline
		_ = enterCmd.Run()
	}

	// Create additional panes
	// Note: Zellij doesn't support split_from like tmux - it always splits the focused pane.
	// For complex layouts with split_from references, results may differ from tmux.
	for i := 1; i < len(layout.Panes); i++ {
		pane := layout.Panes[i]

		// Map direction to zellij direction
		direction := "right" // default
		switch pane.Direction {
		case "right":
			direction = "right"
		case "left":
			direction = "left"
		case "down":
			direction = "down"
		case "up":
			direction = "up"
		}

		// Create new pane
		// Note: Zellij doesn't support percentage-based sizing in new-pane command
		newPaneCmd := exec.Command("zellij", "action", "new-pane", "--direction", direction, "--cwd", wt.Path)
		if err := newPaneCmd.Run(); err != nil {
			continue // Skip this pane on error
		}

		// Run command in new pane if specified
		if pane.Command != "" {
			expandedCmd := expandTemplate(pane.Command, wt, repo, cfg)
			writeCmd := exec.Command("zellij", "action", "write-chars", expandedCmd)
			_ = writeCmd.Run()
			enterCmd := exec.Command("zellij", "action", "write", "10")
			_ = enterCmd.Run()
		}

		// Small delay between pane creations
		time.Sleep(50 * time.Millisecond)
	}

	return nil
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
	// This closes the single quote, adds an escaped single quote, and reopens
	escaped := strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// EchoPath is a simple open command that just echoes the path.
// Useful for shell integration.
func EchoPath(wt *git.Worktree) string {
	return wt.Path
}

// Multiplexer represents the type of terminal multiplexer.
type Multiplexer int

const (
	MultiplexerNone Multiplexer = iota
	MultiplexerTmux
	MultiplexerZellij
)

// GetMultiplexer detects the current terminal multiplexer.
// Returns MultiplexerNone for IDE terminals (VSCode, JetBrains) even if
// multiplexer env vars are set, since those are inherited but not interactive.
func GetMultiplexer() Multiplexer {
	// Check for IDE terminals first - they inherit env vars but aren't interactive
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "vscode" {
		return MultiplexerNone
	}
	// JetBrains IDEs use TERMINAL_EMULATOR
	if strings.HasPrefix(os.Getenv("TERMINAL_EMULATOR"), "JetBrains") {
		return MultiplexerNone
	}

	if os.Getenv("TMUX") != "" {
		return MultiplexerTmux
	}
	if os.Getenv("ZELLIJ") != "" {
		return MultiplexerZellij
	}
	return MultiplexerNone
}

// GetDefaultOpenCommand returns the default open command for the current multiplexer.
// Returns empty string if no multiplexer is detected.
func GetDefaultOpenCommand() string {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		return "tmux new-window -n {branch_short} -c {path}"
	case MultiplexerZellij:
		return "zellij action new-tab --name {branch_short} --cwd {path}"
	default:
		return ""
	}
}

// MultiplexerName returns a human-readable name for the multiplexer.
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
// Returns a list of window/tab IDs.
func FindWindowsForPath(path string) []string {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		return findTmuxWindowsForPath(path)
	case MultiplexerZellij:
		return findZellijTabsForPath(path)
	default:
		return nil
	}
}

// CloseWindow closes a window/tab by ID.
func CloseWindow(windowID string) error {
	switch GetMultiplexer() {
	case MultiplexerTmux:
		return closeTmuxWindow(windowID)
	case MultiplexerZellij:
		return closeZellijTab(windowID)
	default:
		return nil
	}
}

// InMultiplexer returns true if we're running inside a supported multiplexer.
func InMultiplexer() bool {
	return GetMultiplexer() != MultiplexerNone
}

// findTmuxWindowsForPath finds all tmux windows that have panes in the given path.
func findTmuxWindowsForPath(path string) []string {
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{window_id} #{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Resolve symlinks for reliable comparison
	resolvedPath := resolvePath(path)
	windowsMap := make(map[string]bool)

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			windowID := parts[0]
			panePath := resolvePath(parts[1])
			// Check for exact match or if pane is within the worktree
			if panePath == resolvedPath || strings.HasPrefix(panePath, resolvedPath+string(filepath.Separator)) {
				windowsMap[windowID] = true
			}
		}
	}

	windows := make([]string, 0, len(windowsMap))
	for w := range windowsMap {
		windows = append(windows, w)
	}
	return windows
}

// closeTmuxWindow closes a tmux window by ID.
func closeTmuxWindow(windowID string) error {
	cmd := exec.Command("tmux", "kill-window", "-t", windowID)
	return cmd.Run()
}

// findZellijTabsForPath finds zellij tabs that might be in the given path.
// Zellij doesn't have a direct way to query pane CWDs, so we use the tab name
// to find tabs that match the worktree's branch name pattern.
func findZellijTabsForPath(path string) []string {
	// Get the directory name which is typically the branch name
	dirName := filepath.Base(path)

	// Query tab names
	cmd := exec.Command("zellij", "action", "query-tab-names")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var tabs []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		tabName := strings.TrimSpace(line)
		// Check if tab name matches the directory name (branch name)
		if tabName == dirName {
			// Zellij uses 1-based tab indices
			tabs = append(tabs, fmt.Sprintf("%d", i+1))
		}
	}
	return tabs
}

// closeZellijTab closes a zellij tab by index.
func closeZellijTab(tabIndex string) error {
	// First go to the tab, then close it
	goCmd := exec.Command("zellij", "action", "go-to-tab", tabIndex)
	if err := goCmd.Run(); err != nil {
		return err
	}
	closeCmd := exec.Command("zellij", "action", "close-tab")
	return closeCmd.Run()
}

// resolvePath returns the absolute path with symlinks resolved.
// Falls back to absolute path if symlink resolution fails.
func resolvePath(path string) string {
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
