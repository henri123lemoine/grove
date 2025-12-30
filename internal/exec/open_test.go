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
