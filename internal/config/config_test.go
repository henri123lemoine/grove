package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.General.DefaultBaseBranch != "main" {
		t.Errorf("Expected default base branch 'main', got %q", cfg.General.DefaultBaseBranch)
	}

	if cfg.General.WorktreeDir != ".worktrees" {
		t.Errorf("Expected worktree dir '.worktrees', got %q", cfg.General.WorktreeDir)
	}

	if cfg.Open.ExitAfterOpen != true {
		t.Error("Expected ExitAfterOpen to be true")
	}

	if cfg.PR.Command != "gh pr create" {
		t.Errorf("Expected PR command 'gh pr create', got %q", cfg.PR.Command)
	}

	if cfg.PR.AutoPush != true {
		t.Error("Expected AutoPush to be true")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantWarning bool
	}{
		{
			name:        "default config is valid",
			config:      DefaultConfig(),
			wantWarning: false,
		},
		{
			name: "invalid template variable",
			config: &Config{
				Open: OpenConfig{
					Command: "tmux new-window -c {invalid_var}",
				},
			},
			wantWarning: true,
		},
		{
			name: "invalid detect_existing",
			config: &Config{
				Open: OpenConfig{
					DetectExisting: "invalid",
				},
			},
			wantWarning: true,
		},
		{
			name: "invalid layout",
			config: &Config{
				Open: OpenConfig{
					Layout: "invalid",
				},
			},
			wantWarning: true,
		},
		{
			name: "invalid window_name_style",
			config: &Config{
				Open: OpenConfig{
					WindowNameStyle: "invalid",
				},
			},
			wantWarning: true,
		},
		{
			name: "invalid theme",
			config: &Config{
				UI: UIConfig{
					Theme: "invalid",
				},
			},
			wantWarning: true,
		},
		{
			name: "valid template variables",
			config: &Config{
				Open: OpenConfig{
					Command: "tmux new-window -n {branch_short} -c {path}",
				},
			},
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.config.Validate()
			hasWarnings := len(warnings) > 0
			if hasWarnings != tt.wantWarning {
				t.Errorf("Validate() hasWarnings = %v, want %v. Warnings: %v", hasWarnings, tt.wantWarning, warnings)
			}
		})
	}
}

func TestMergeConfig(t *testing.T) {
	base := DefaultConfig()
	file := &Config{
		General: GeneralConfig{
			DefaultBaseBranch: "develop",
		},
		Open: OpenConfig{
			Command: "custom-command",
		},
		PR: PRConfig{
			Command:  "glab mr create",
			AutoPush: false,
		},
	}

	mergeConfig(base, file)

	if base.General.DefaultBaseBranch != "develop" {
		t.Errorf("Expected merged base branch 'develop', got %q", base.General.DefaultBaseBranch)
	}

	if base.Open.Command != "custom-command" {
		t.Errorf("Expected merged command 'custom-command', got %q", base.Open.Command)
	}

	if base.PR.Command != "glab mr create" {
		t.Errorf("Expected merged PR command 'glab mr create', got %q", base.PR.Command)
	}

	// Check that non-specified values keep defaults
	if base.General.WorktreeDir != ".worktrees" {
		t.Errorf("Expected default worktree dir '.worktrees', got %q", base.General.WorktreeDir)
	}
}

func TestGetTemplateForBranch(t *testing.T) {
	cfg := &Config{
		Worktree: WorktreeConfig{
			Templates: []TemplateConfig{
				{Pattern: "feature/*", CopyPatterns: []string{".env.local"}},
				{Pattern: "fix/*", CopyPatterns: []string{".env.test"}},
				{Pattern: "release/**", PostCreateCmd: []string{"npm run build"}},
			},
		},
	}

	tests := []struct {
		branch   string
		wantNil  bool
		patterns []string
	}{
		{"feature/auth", false, []string{".env.local"}},
		{"feature/payment", false, []string{".env.local"}},
		{"fix/bug-123", false, []string{".env.test"}},
		{"release/v1/patch", false, nil},
		{"main", true, nil},
		{"develop", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			template := cfg.GetTemplateForBranch(tt.branch)
			if tt.wantNil {
				if template != nil {
					t.Errorf("Expected nil template for branch %q", tt.branch)
				}
			} else {
				if template == nil {
					t.Errorf("Expected template for branch %q, got nil", tt.branch)
					return
				}
				if tt.patterns != nil && len(template.CopyPatterns) != len(tt.patterns) {
					t.Errorf("Expected %d patterns, got %d", len(tt.patterns), len(template.CopyPatterns))
				}
			}
		})
	}
}

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"feature/*", "feature/auth", true},
		{"feature/*", "feature/payment", true},
		{"feature/*", "main", false},
		{"feature/*", "feature/nested/deep", false}, // * doesn't match /
		{"feature/**", "feature/nested/deep", true}, // ** matches /
		{"fix-*", "fix-123", true},
		{"fix-*", "fix-abc", true},
		{"*.go", "main.go", true},
		{"*.go", "main.txt", false},
		{"release/?", "release/a", true},
		{"release/?", "release/ab", false}, // ? matches single char
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.name, func(t *testing.T) {
			got := matchGlobPattern(tt.pattern, tt.name)
			if got != tt.want {
				t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestDetectEnvironment(t *testing.T) {
	// Save original env
	origTmux := os.Getenv("TMUX")
	origZellij := os.Getenv("ZELLIJ")
	defer func() {
		os.Setenv("TMUX", origTmux)
		os.Setenv("ZELLIJ", origZellij)
	}()

	// Test tmux detection
	os.Setenv("TMUX", "/tmp/tmux-123/default,12345,0")
	os.Setenv("ZELLIJ", "")
	if env := DetectEnvironment(); env != "tmux" {
		t.Errorf("Expected 'tmux', got %q", env)
	}

	// Test zellij detection
	os.Setenv("TMUX", "")
	os.Setenv("ZELLIJ", "123")
	if env := DetectEnvironment(); env != "zellij" {
		t.Errorf("Expected 'zellij', got %q", env)
	}

	// Test generic
	os.Setenv("TMUX", "")
	os.Setenv("ZELLIJ", "")
	if env := DetectEnvironment(); env != "generic" {
		t.Errorf("Expected 'generic', got %q", env)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath should not return empty string")
	}

	// Should end with grove/config.toml
	if filepath.Base(path) != "config.toml" {
		t.Errorf("Expected config.toml, got %q", filepath.Base(path))
	}

	dir := filepath.Dir(path)
	if filepath.Base(dir) != "grove" {
		t.Errorf("Expected grove dir, got %q", filepath.Base(dir))
	}
}

func TestExtractTemplateVars(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"tmux new-window -n {branch_short} -c {path}", []string{"{branch_short}", "{path}"}},
		{"no vars here", nil},
		{"{a} {b} {c}", []string{"{a}", "{b}", "{c}"}},
		{"{}", nil}, // Empty braces are not valid template vars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractTemplateVars(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("extractTemplateVars(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
