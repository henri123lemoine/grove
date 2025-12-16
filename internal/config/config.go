// Package config handles grove configuration.
package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config represents grove configuration.
type Config struct {
	General GeneralConfig `toml:"general"`
	Open    OpenConfig    `toml:"open"`
	Safety  SafetyConfig  `toml:"safety"`
	UI      UIConfig      `toml:"ui"`
}

// GeneralConfig contains general settings.
type GeneralConfig struct {
	// Default base branch for new worktrees
	DefaultBaseBranch string `toml:"default_base_branch"`

	// Directory for worktrees (relative to repo root, or absolute)
	WorktreeDir string `toml:"worktree_dir"`
}

// OpenConfig contains settings for opening worktrees.
type OpenConfig struct {
	// Command to run when opening a worktree
	// Template variables: {path}, {branch}, {branch_short}, {repo}
	Command string `toml:"command"`

	// Whether to exit grove after opening
	ExitAfterOpen bool `toml:"exit_after_open"`
}

// SafetyConfig contains safety settings.
type SafetyConfig struct {
	// Confirm before deleting dirty worktrees
	ConfirmDirty bool `toml:"confirm_dirty"`

	// Confirm before deleting unmerged branches
	ConfirmUnmerged bool `toml:"confirm_unmerged"`

	// Require typing "delete" for worktrees with unique commits
	RequireTypingForUnique bool `toml:"require_typing_for_unique"`
}

// UIConfig contains UI settings.
type UIConfig struct {
	// Show commit info in detail panel
	ShowCommits bool `toml:"show_commits"`

	// Show upstream tracking status
	ShowUpstream bool `toml:"show_upstream"`

	// Color theme: auto, dark, light
	Theme string `toml:"theme"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			DefaultBaseBranch: "main",
			WorktreeDir:       ".worktrees",
		},
		Open: OpenConfig{
			// Try to switch to existing window, otherwise create new one
			Command:       "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}",
			ExitAfterOpen: true,
		},
		Safety: SafetyConfig{
			ConfirmDirty:           true,
			ConfirmUnmerged:        true,
			RequireTypingForUnique: true,
		},
		UI: UIConfig{
			ShowCommits:  true,
			ShowUpstream: true,
			Theme:        "auto",
		},
	}
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	return filepath.Join(configDir, "grove", "config.toml")
}

// Load loads configuration from the config file.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file, use defaults
			return cfg, nil
		}
		return nil, err
	}

	// Parse TOML
	var fileCfg Config
	if err := toml.Unmarshal(data, &fileCfg); err != nil {
		return nil, err
	}

	// Merge with defaults (file config takes precedence)
	mergeConfig(cfg, &fileCfg)

	return cfg, nil
}

// Save saves configuration to the config file.
func Save(cfg *Config) error {
	path := ConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// mergeConfig merges the file config into the base config.
// Non-zero values in file config override base config.
func mergeConfig(base, file *Config) {
	// General
	if file.General.DefaultBaseBranch != "" {
		base.General.DefaultBaseBranch = file.General.DefaultBaseBranch
	}
	if file.General.WorktreeDir != "" {
		base.General.WorktreeDir = file.General.WorktreeDir
	}

	// Open
	if file.Open.Command != "" {
		base.Open.Command = file.Open.Command
	}
	// ExitAfterOpen is a bool, so we always use file value if it's set
	// Since we can't distinguish between "not set" and "false" in TOML,
	// we just use the file value
	base.Open.ExitAfterOpen = file.Open.ExitAfterOpen

	// Safety - same issue with bools, use file values
	base.Safety.ConfirmDirty = file.Safety.ConfirmDirty
	base.Safety.ConfirmUnmerged = file.Safety.ConfirmUnmerged
	base.Safety.RequireTypingForUnique = file.Safety.RequireTypingForUnique

	// UI
	base.UI.ShowCommits = file.UI.ShowCommits
	base.UI.ShowUpstream = file.UI.ShowUpstream
	if file.UI.Theme != "" {
		base.UI.Theme = file.UI.Theme
	}
}
