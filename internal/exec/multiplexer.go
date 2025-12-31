package exec

import (
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/henri123lemoine/grove/internal/config"
	"github.com/henri123lemoine/grove/internal/git"
)

// MultiplexerBackend defines the interface for terminal multiplexer operations.
type MultiplexerBackend interface {
	// Name returns the human-readable name (e.g., "tmux", "zellij").
	Name() string

	// WindowName returns the term used for windows/tabs (e.g., "window", "tab").
	WindowName() string

	// DefaultOpenCommand returns the default command to open a new window/tab.
	DefaultOpenCommand() string

	// FindWindowByPath finds a window/tab by checking pane paths.
	// Returns window/tab ID or empty string if not found.
	FindWindowByPath(path string) string

	// FindWindowByName finds a window/tab by name.
	// Returns window/tab ID or empty string if not found.
	FindWindowByName(name string) string

	// SwitchToWindow switches to a window/tab by ID.
	SwitchToWindow(windowID string) error

	// FindWindowsForPath finds all windows/tabs that have panes in the given path.
	FindWindowsForPath(path string) []string

	// CloseWindow closes a window/tab by ID.
	CloseWindow(windowID string) error

	// ApplyNamedLayout applies a named layout with multiple panes.
	ApplyNamedLayout(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error
}

// tmuxBackend implements MultiplexerBackend for tmux.
type tmuxBackend struct{}

func (t *tmuxBackend) Name() string {
	return "tmux"
}

func (t *tmuxBackend) WindowName() string {
	return "window"
}

func (t *tmuxBackend) DefaultOpenCommand() string {
	return "tmux new-window -n {branch_short} -c {path}"
}

func (t *tmuxBackend) FindWindowByPath(path string) string {
	cmd := osExec.Command("tmux", "list-panes", "-a", "-F", "#{window_id} #{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	resolvedPath := git.ResolvePath(path)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			windowID := parts[0]
			panePath := git.ResolvePath(parts[1])
			if panePath == resolvedPath || strings.HasPrefix(panePath, resolvedPath+string(filepath.Separator)) {
				return windowID
			}
		}
	}
	return ""
}

func (t *tmuxBackend) FindWindowByName(name string) string {
	cmd := osExec.Command("tmux", "list-windows", "-F", "#{window_id} #{window_name}")
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

func (t *tmuxBackend) SwitchToWindow(windowID string) error {
	cmd := osExec.Command("tmux", "select-window", "-t", windowID)
	return cmd.Run()
}

func (t *tmuxBackend) FindWindowsForPath(path string) []string {
	cmd := osExec.Command("tmux", "list-panes", "-a", "-F", "#{window_id} #{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	resolvedPath := git.ResolvePath(path)
	windowsMap := make(map[string]bool)

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			windowID := parts[0]
			panePath := git.ResolvePath(parts[1])
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

func (t *tmuxBackend) CloseWindow(windowID string) error {
	cmd := osExec.Command("tmux", "kill-window", "-t", windowID)
	return cmd.Run()
}

func (t *tmuxBackend) ApplyNamedLayout(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error {
	if len(layout.Panes) == 0 {
		return nil
	}

	// Determine window name to target (the newly created window)
	windowName := wt.BranchShort()
	if cfg.Open.WindowNameStyle == "full" {
		windowName = wt.Branch
	}

	// Track pane IDs as we create them
	paneIDs := make([]string, len(layout.Panes))

	// Get the pane ID of the newly created window
	cmd := osExec.Command("tmux", "list-panes", "-t", windowName, "-F", "#{pane_id}")
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
	if layout.Panes[0].Command != "" {
		expandedCmd := expandTemplate(layout.Panes[0].Command, wt, repo, cfg)
		sendCmd := osExec.Command("tmux", "send-keys", "-t", paneIDs[0], expandedCmd, "Enter")
		_ = sendCmd.Run()
	}

	// Create additional panes
	for i := 1; i < len(layout.Panes); i++ {
		pane := layout.Panes[i]

		if pane.SplitFrom < 0 || pane.SplitFrom >= i {
			continue
		}

		targetPane := paneIDs[pane.SplitFrom]
		if targetPane == "" {
			continue
		}

		splitArgs := []string{"split-window"}

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
			splitArgs = append(splitArgs, "-h")
		}

		if pane.Size > 0 && pane.Size < 100 {
			splitArgs = append(splitArgs, "-p", fmt.Sprintf("%d", pane.Size))
		}

		splitArgs = append(splitArgs, "-t", targetPane)
		splitArgs = append(splitArgs, "-c", wt.Path)
		splitArgs = append(splitArgs, "-P", "-F", "#{pane_id}")

		splitCmd := osExec.Command("tmux", splitArgs...)
		splitOutput, err := splitCmd.Output()
		if err != nil {
			continue
		}

		newPaneID := strings.TrimSpace(string(splitOutput))
		paneIDs[i] = newPaneID

		if pane.Command != "" {
			expandedCmd := expandTemplate(pane.Command, wt, repo, cfg)
			sendCmd := osExec.Command("tmux", "send-keys", "-t", newPaneID, expandedCmd, "Enter")
			_ = sendCmd.Run()
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// zellijBackend implements MultiplexerBackend for zellij.
type zellijBackend struct{}

func (z *zellijBackend) Name() string {
	return "zellij"
}

func (z *zellijBackend) WindowName() string {
	return "tab"
}

func (z *zellijBackend) DefaultOpenCommand() string {
	return "zellij action new-tab --name {branch_short} --cwd {path}"
}

func (z *zellijBackend) FindWindowByPath(path string) string {
	// Zellij doesn't expose pane CWDs, so fall back to name-based detection
	return z.FindWindowByName(filepath.Base(path))
}

func (z *zellijBackend) FindWindowByName(name string) string {
	cmd := osExec.Command("zellij", "action", "query-tab-names")
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

func (z *zellijBackend) SwitchToWindow(windowID string) error {
	cmd := osExec.Command("zellij", "action", "go-to-tab", windowID)
	return cmd.Run()
}

func (z *zellijBackend) FindWindowsForPath(path string) []string {
	dirName := filepath.Base(path)

	cmd := osExec.Command("zellij", "action", "query-tab-names")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var tabs []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		tabName := strings.TrimSpace(line)
		if tabName == dirName {
			tabs = append(tabs, fmt.Sprintf("%d", i+1))
		}
	}
	return tabs
}

func (z *zellijBackend) CloseWindow(tabIndex string) error {
	goCmd := osExec.Command("zellij", "action", "go-to-tab", tabIndex)
	if err := goCmd.Run(); err != nil {
		return err
	}
	closeCmd := osExec.Command("zellij", "action", "close-tab")
	return closeCmd.Run()
}

func (z *zellijBackend) ApplyNamedLayout(layout *config.LayoutConfig, wt *git.Worktree, repo *git.Repo, cfg *config.Config) error {
	if len(layout.Panes) == 0 {
		return nil
	}

	// Run command for pane 0 if specified
	if layout.Panes[0].Command != "" {
		expandedCmd := expandTemplate(layout.Panes[0].Command, wt, repo, cfg)
		writeCmd := osExec.Command("zellij", "action", "write-chars", expandedCmd)
		_ = writeCmd.Run()
		enterCmd := osExec.Command("zellij", "action", "write", "10")
		_ = enterCmd.Run()
	}

	// Create additional panes
	// Note: Zellij doesn't support split_from like tmux
	for i := 1; i < len(layout.Panes); i++ {
		pane := layout.Panes[i]

		direction := "right"
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

		newPaneCmd := osExec.Command("zellij", "action", "new-pane", "--direction", direction, "--cwd", wt.Path)
		if err := newPaneCmd.Run(); err != nil {
			continue
		}

		if pane.Command != "" {
			expandedCmd := expandTemplate(pane.Command, wt, repo, cfg)
			writeCmd := osExec.Command("zellij", "action", "write-chars", expandedCmd)
			_ = writeCmd.Run()
			enterCmd := osExec.Command("zellij", "action", "write", "10")
			_ = enterCmd.Run()
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// noneBackend is a no-op implementation for when no multiplexer is detected.
type noneBackend struct{}

func (n *noneBackend) Name() string                       { return "" }
func (n *noneBackend) WindowName() string                 { return "window" }
func (n *noneBackend) DefaultOpenCommand() string         { return "" }
func (n *noneBackend) FindWindowByPath(string) string     { return "" }
func (n *noneBackend) FindWindowByName(string) string     { return "" }
func (n *noneBackend) SwitchToWindow(string) error        { return nil }
func (n *noneBackend) FindWindowsForPath(string) []string { return nil }
func (n *noneBackend) CloseWindow(string) error           { return nil }
func (n *noneBackend) ApplyNamedLayout(*config.LayoutConfig, *git.Worktree, *git.Repo, *config.Config) error {
	return nil
}

// Backend returns the MultiplexerBackend for the current environment.
// The backend is cached for the lifetime of the process.
var multiplexerBackend MultiplexerBackend

func Backend() MultiplexerBackend {
	if multiplexerBackend != nil {
		return multiplexerBackend
	}

	// Check for IDE terminals first - they inherit env vars but aren't interactive
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "vscode" {
		multiplexerBackend = &noneBackend{}
		return multiplexerBackend
	}
	if strings.HasPrefix(os.Getenv("TERMINAL_EMULATOR"), "JetBrains") {
		multiplexerBackend = &noneBackend{}
		return multiplexerBackend
	}

	if os.Getenv("TMUX") != "" {
		multiplexerBackend = &tmuxBackend{}
	} else if os.Getenv("ZELLIJ") != "" {
		multiplexerBackend = &zellijBackend{}
	} else {
		multiplexerBackend = &noneBackend{}
	}

	return multiplexerBackend
}

// ResetBackend resets the cached backend (useful for testing).
func ResetBackend() {
	multiplexerBackend = nil
}
