package exec

import (
	"testing"

	"github.com/henri123lemoine/grove/internal/config"
	"github.com/henri123lemoine/grove/internal/git"
)

func TestExpandTemplate(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/feature-auth",
		Branch: "feature/auth",
	}
	repo := &git.Repo{
		Root: "/home/user/project",
	}
	cfg := config.DefaultConfig()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "path variable",
			template: "cd {path}",
			expected: "cd /home/user/project/.worktrees/feature-auth",
		},
		{
			name:     "branch variable",
			template: "git checkout {branch}",
			expected: "git checkout feature/auth",
		},
		{
			name:     "branch_short variable",
			template: "tmux new-window -n {branch_short}",
			expected: "tmux new-window -n auth",
		},
		{
			name:     "repo variable",
			template: "echo {repo}",
			expected: "echo project",
		},
		{
			name:     "window_name short style",
			template: "tmux new-window -n {window_name}",
			expected: "tmux new-window -n auth",
		},
		{
			name:     "multiple variables",
			template: "tmux new-window -n {branch_short} -c {path}",
			expected: "tmux new-window -n auth -c /home/user/project/.worktrees/feature-auth",
		},
		{
			name:     "tmux default command",
			template: "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}",
			expected: "tmux select-window -t :auth 2>/dev/null || tmux new-window -n auth -c /home/user/project/.worktrees/feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTemplate(tt.template, wt, repo, cfg)
			if result != tt.expected {
				t.Errorf("expandTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandTemplateFullWindowName(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/feature-auth",
		Branch: "feature/auth",
	}
	repo := &git.Repo{
		Root: "/home/user/project",
	}
	cfg := config.DefaultConfig()
	cfg.Open.WindowNameStyle = "full"

	result := expandTemplate("tmux new-window -n {window_name}", wt, repo, cfg)
	expected := "tmux new-window -n feature/auth"

	if result != expected {
		t.Errorf("expandTemplate() with full window_name = %q, want %q", result, expected)
	}
}

func TestExpandTemplateZellij(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/fix-bug",
		Branch: "fix/bug-123",
	}
	repo := &git.Repo{
		Root: "/home/user/project",
	}
	cfg := config.DefaultConfig()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "zellij new pane",
			template: "zellij action new-pane --cwd {path}",
			expected: "zellij action new-pane --cwd /home/user/project/.worktrees/fix-bug",
		},
		{
			name:     "zellij new tab",
			template: "zellij action new-tab --cwd {path} --name {branch_short}",
			expected: "zellij action new-tab --cwd /home/user/project/.worktrees/fix-bug --name bug-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTemplate(tt.template, wt, repo, cfg)
			if result != tt.expected {
				t.Errorf("expandTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandTemplatePathsWithSpaces(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/My Projects/feature-branch",
		Branch: "feature/test",
	}
	repo := &git.Repo{
		Root: "/home/user/My Projects",
	}
	cfg := config.DefaultConfig()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "path with spaces is quoted",
			template: "cd {path}",
			expected: "cd '/home/user/My Projects/feature-branch'",
		},
		{
			name:     "tmux with path containing spaces",
			template: "tmux new-window -c {path}",
			expected: "tmux new-window -c '/home/user/My Projects/feature-branch'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTemplate(tt.template, wt, repo, cfg)
			if result != tt.expected {
				t.Errorf("expandTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandTemplateQuotesBranchAndRepo(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/project",
		Branch: "feature;rm",
	}
	repo := &git.Repo{
		Root: "/home/user/My Projects",
	}
	cfg := config.DefaultConfig()

	result := expandTemplate("echo {branch} {branch_short} {repo} {window_name}", wt, repo, cfg)
	expected := "echo 'feature;rm' 'feature;rm' 'My Projects' 'feature;rm'"
	if result != expected {
		t.Errorf("expandTemplate() = %q, want %q", result, expected)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars - no quoting",
			input:    "/home/user/project",
			expected: "/home/user/project",
		},
		{
			name:     "path with spaces",
			input:    "/home/user/My Project",
			expected: "'/home/user/My Project'",
		},
		{
			name:     "path with single quote",
			input:    "/home/user/it's here",
			expected: "'/home/user/it'\"'\"'s here'",
		},
		{
			name:     "path with multiple special chars",
			input:    "/home/user/test $VAR",
			expected: "'/home/user/test $VAR'",
		},
		{
			name:     "path with parentheses",
			input:    "/home/user/test (copy)",
			expected: "'/home/user/test (copy)'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// mockBackend is a test implementation of MultiplexerBackend.
type mockBackend struct {
	name              string
	windowName        string
	defaultCmd        string
	windowsByPath     map[string]string
	windowsByName     map[string]string
	allWindowsForPath map[string][]string
	switchCalls       []string
	closeCalls        []string
	layoutCalls       int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		name:              "mock",
		windowName:        "window",
		defaultCmd:        "mock-open {path}",
		windowsByPath:     make(map[string]string),
		windowsByName:     make(map[string]string),
		allWindowsForPath: make(map[string][]string),
	}
}

func (m *mockBackend) Name() string               { return m.name }
func (m *mockBackend) WindowName() string         { return m.windowName }
func (m *mockBackend) DefaultOpenCommand() string { return m.defaultCmd }

func (m *mockBackend) FindWindowByPath(path string) string {
	return m.windowsByPath[path]
}

func (m *mockBackend) FindWindowByName(name string) string {
	return m.windowsByName[name]
}

func (m *mockBackend) SwitchToWindow(windowID string) error {
	m.switchCalls = append(m.switchCalls, windowID)
	return nil
}

func (m *mockBackend) FindWindowsForPath(path string) []string {
	return m.allWindowsForPath[path]
}

func (m *mockBackend) CloseWindow(windowID string) error {
	m.closeCalls = append(m.closeCalls, windowID)
	return nil
}

func (m *mockBackend) ApplyNamedLayout(*config.LayoutConfig, *git.Worktree, *git.Repo, *config.Config) error {
	m.layoutCalls++
	return nil
}

// setMockBackend sets a mock backend for testing and returns a cleanup function.
func setMockBackend(m *mockBackend) func() {
	old := multiplexerBackend
	multiplexerBackend = m
	return func() {
		multiplexerBackend = old
	}
}

func TestWindowExistsFor_ByPath(t *testing.T) {
	mock := newMockBackend()
	mock.windowsByPath["/home/user/project/.worktrees/feature"] = "@1"
	cleanup := setMockBackend(mock)
	defer cleanup()

	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/feature",
		Branch: "feature/test",
	}
	cfg := config.DefaultConfig()
	cfg.Open.DetectExisting = "path"

	if !WindowExistsFor(cfg, wt) {
		t.Error("WindowExistsFor should return true when window exists by path")
	}

	// Test non-existent path
	wt2 := &git.Worktree{
		Path:   "/home/user/project/.worktrees/other",
		Branch: "other",
	}
	if WindowExistsFor(cfg, wt2) {
		t.Error("WindowExistsFor should return false when window doesn't exist")
	}
}

func TestWindowExistsFor_ByName(t *testing.T) {
	mock := newMockBackend()
	mock.windowsByName["test"] = "@2"
	cleanup := setMockBackend(mock)
	defer cleanup()

	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/feature-test",
		Branch: "feature/test",
	}
	cfg := config.DefaultConfig()
	cfg.Open.DetectExisting = "name"

	if !WindowExistsFor(cfg, wt) {
		t.Error("WindowExistsFor should return true when window exists by name")
	}

	// Test with full window name style
	cfg.Open.WindowNameStyle = "full"
	mock.windowsByName["feature/test"] = "@3"
	if !WindowExistsFor(cfg, wt) {
		t.Error("WindowExistsFor should use full branch name when configured")
	}
}

func TestWindowExistsFor_None(t *testing.T) {
	mock := newMockBackend()
	mock.windowsByPath["/some/path"] = "@1"
	mock.windowsByName["test"] = "@2"
	cleanup := setMockBackend(mock)
	defer cleanup()

	wt := &git.Worktree{
		Path:   "/some/path",
		Branch: "feature/test",
	}
	cfg := config.DefaultConfig()
	cfg.Open.DetectExisting = "none"

	if WindowExistsFor(cfg, wt) {
		t.Error("WindowExistsFor should return false when detect_existing is 'none'")
	}
}

func TestFindWindowsForPath_WithMock(t *testing.T) {
	mock := newMockBackend()
	mock.allWindowsForPath["/project"] = []string{"@1", "@2", "@3"}
	cleanup := setMockBackend(mock)
	defer cleanup()

	windows := FindWindowsForPath("/project")
	if len(windows) != 3 {
		t.Errorf("Expected 3 windows, got %d", len(windows))
	}

	// Test empty result
	windows = FindWindowsForPath("/other")
	if len(windows) != 0 {
		t.Errorf("Expected 0 windows for unknown path, got %d", len(windows))
	}
}

func TestCloseWindow_WithMock(t *testing.T) {
	mock := newMockBackend()
	cleanup := setMockBackend(mock)
	defer cleanup()

	if err := CloseWindow("@1"); err != nil {
		t.Errorf("CloseWindow failed: %v", err)
	}
	if err := CloseWindow("@2"); err != nil {
		t.Errorf("CloseWindow failed: %v", err)
	}

	if len(mock.closeCalls) != 2 {
		t.Errorf("Expected 2 close calls, got %d", len(mock.closeCalls))
	}
	if mock.closeCalls[0] != "@1" || mock.closeCalls[1] != "@2" {
		t.Errorf("Close calls = %v, want [@1, @2]", mock.closeCalls)
	}
}

func TestInMultiplexer_WithMock(t *testing.T) {
	// Test with active multiplexer
	mock := newMockBackend()
	mock.name = "tmux"
	cleanup := setMockBackend(mock)

	if !InMultiplexer() {
		t.Error("InMultiplexer should return true when backend has a name")
	}
	cleanup()

	// Test with no multiplexer
	mock2 := newMockBackend()
	mock2.name = ""
	cleanup2 := setMockBackend(mock2)
	defer cleanup2()

	if InMultiplexer() {
		t.Error("InMultiplexer should return false when backend has empty name")
	}
}

func TestGetDefaultOpenCommand_WithMock(t *testing.T) {
	mock := newMockBackend()
	mock.defaultCmd = "custom-cmd {path} {branch}"
	cleanup := setMockBackend(mock)
	defer cleanup()

	cmd := GetDefaultOpenCommand()
	if cmd != "custom-cmd {path} {branch}" {
		t.Errorf("GetDefaultOpenCommand = %q, want %q", cmd, "custom-cmd {path} {branch}")
	}
}

func TestBackendCaching(t *testing.T) {
	// Reset backend
	ResetBackend()
	defer ResetBackend()

	// First call should create backend
	b1 := Backend()
	// Second call should return same instance
	b2 := Backend()

	if b1 != b2 {
		t.Error("Backend() should return cached instance")
	}
}
