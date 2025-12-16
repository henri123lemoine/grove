package exec

import (
	"testing"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
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

func TestExpandTemplateVSCode(t *testing.T) {
	wt := &git.Worktree{
		Path:   "/home/user/project/.worktrees/feature",
		Branch: "feature/new-feature",
	}
	repo := &git.Repo{
		Root: "/home/user/project",
	}
	cfg := config.DefaultConfig()

	result := expandTemplate("code {path}", wt, repo, cfg)
	expected := "code /home/user/project/.worktrees/feature"

	if result != expected {
		t.Errorf("expandTemplate() for VS Code = %q, want %q", result, expected)
	}
}
