// Package config handles grove configuration.
package config

import (
	"os"
	"path/filepath"
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
			Command:       "echo {path}",
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
	// TODO: Implement
	// 1. Check if config file exists
	// 2. Parse TOML
	// 3. Merge with defaults
	return DefaultConfig(), nil
}

// Save saves configuration to the config file.
func Save(cfg *Config) error {
	// TODO: Implement
	return nil
}
