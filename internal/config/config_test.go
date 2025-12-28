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

func TestLoadPreservesDefaults(t *testing.T) {
	// Create a temp config file with partial config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Only specify some values - others should keep defaults
	tomlContent := `[general]
default_base_branch = "develop"

[open]
command = "custom-command"
`
	if err := os.WriteFile(configPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error: %v", err)
	}

	// Check specified values were loaded
	if cfg.General.DefaultBaseBranch != "develop" {
		t.Errorf("Expected base branch 'develop', got %q", cfg.General.DefaultBaseBranch)
	}

	if cfg.Open.Command != "custom-command" {
		t.Errorf("Expected command 'custom-command', got %q", cfg.Open.Command)
	}

	// Check that non-specified values keep defaults
	if cfg.General.WorktreeDir != ".worktrees" {
		t.Errorf("Expected default worktree dir '.worktrees', got %q", cfg.General.WorktreeDir)
	}

	// IMPORTANT: Check that boolean defaults are preserved when not specified
	if cfg.Open.ExitAfterOpen != true {
		t.Error("Expected ExitAfterOpen to remain true (default) when not specified in config")
	}

	if cfg.Safety.ConfirmDirty != true {
		t.Error("Expected ConfirmDirty to remain true (default) when not specified in config")
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
