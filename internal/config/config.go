// Package config handles grove configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config represents grove configuration.
type Config struct {
	General  GeneralConfig  `toml:"general"`
	Open     OpenConfig     `toml:"open"`
	PR       PRConfig       `toml:"pr"`
	Worktree WorktreeConfig `toml:"worktree"`
	Safety   SafetyConfig   `toml:"safety"`
	UI       UIConfig       `toml:"ui"`
	Keys     KeysConfig     `toml:"keys"`
}

// GeneralConfig contains general settings.
type GeneralConfig struct {
	// Default base branch for new worktrees
	DefaultBaseBranch string `toml:"default_base_branch"`

	// Directory for worktrees (relative to repo root, or absolute)
	WorktreeDir string `toml:"worktree_dir"`

	// Default remote name (empty = auto-detect)
	Remote string `toml:"remote"`
}

// OpenConfig contains settings for opening worktrees.
type OpenConfig struct {
	// Command to run when opening a worktree
	// Template variables: {path}, {branch}, {branch_short}, {repo}, {window_name}
	Command string `toml:"command"`

	// How to detect existing windows: "path", "name", or "none"
	DetectExisting string `toml:"detect_existing"`

	// Whether to exit grove after opening
	ExitAfterOpen bool `toml:"exit_after_open"`

	// Layout to apply after creating new window: "none", "dev", or "custom"
	Layout string `toml:"layout"`

	// Custom layout command (only if layout = "custom")
	LayoutCommand string `toml:"layout_command"`

	// Window name style: "short" or "full"
	WindowNameStyle string `toml:"window_name_style"`

	// Stash dirty worktree before switching
	StashOnSwitch bool `toml:"stash_on_switch"`
}

// PRConfig contains settings for PR creation.
type PRConfig struct {
	// Command to create PR (e.g., "gh pr create" or "glab mr create")
	Command string `toml:"command"`

	// Auto-push branch if no upstream
	AutoPush bool `toml:"auto_push"`
}

// WorktreeConfig contains settings for worktree creation.
type WorktreeConfig struct {
	// File patterns to copy to new worktrees (e.g., ".env*", ".vscode/**")
	CopyPatterns []string `toml:"copy_patterns"`

	// File patterns to ignore when copying
	CopyIgnores []string `toml:"copy_ignores"`

	// Commands to run after creating worktree
	PostCreateCmd []string `toml:"post_create_cmd"`

	// Templates for different branch patterns
	Templates []TemplateConfig `toml:"templates"`
}

// TemplateConfig defines a template for specific branch patterns.
type TemplateConfig struct {
	// Pattern to match branch names (glob-style)
	Pattern string `toml:"pattern"`

	// File patterns to copy for this template
	CopyPatterns []string `toml:"copy_patterns"`

	// Commands to run after creating worktree
	PostCreateCmd []string `toml:"post_create_cmd"`
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
	// Show branch type indicators in create flow
	ShowBranchTypes bool `toml:"show_branch_types"`

	// Show commit info in detail panel
	ShowCommits bool `toml:"show_commits"`

	// Show upstream tracking status
	ShowUpstream bool `toml:"show_upstream"`

	// Color theme: auto, dark, light
	Theme string `toml:"theme"`
}

// KeysConfig contains keybinding settings.
type KeysConfig struct {
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Home   string `toml:"home"`
	End    string `toml:"end"`
	Open   string `toml:"open"`
	New    string `toml:"new"`
	Delete string `toml:"delete"`
	PR     string `toml:"pr"`
	Rename string `toml:"rename"`
	Filter string `toml:"filter"`
	Fetch  string `toml:"fetch"`
	Detail string `toml:"detail"`
	Help   string `toml:"help"`
	Quit   string `toml:"quit"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			DefaultBaseBranch: "main",
			WorktreeDir:       ".worktrees",
		},
		Open: OpenConfig{
			// Smart default: detect by path, then create new window
			Command:         "tmux select-window -t :{branch_short} 2>/dev/null || tmux new-window -n {branch_short} -c {path}",
			DetectExisting:  "name",
			ExitAfterOpen:   true,
			Layout:          "none",
			LayoutCommand:   "",
			WindowNameStyle: "short",
			StashOnSwitch:   false,
		},
		PR: PRConfig{
			Command:  "gh pr create",
			AutoPush: true,
		},
		Worktree: WorktreeConfig{
			CopyPatterns:  []string{},
			CopyIgnores:   []string{},
			PostCreateCmd: []string{},
			Templates:     []TemplateConfig{},
		},
		Safety: SafetyConfig{
			ConfirmDirty:           true,
			ConfirmUnmerged:        true,
			RequireTypingForUnique: true,
		},
		UI: UIConfig{
			ShowBranchTypes: true,
			ShowCommits:     true,
			ShowUpstream:    true,
			Theme:           "auto",
		},
		Keys: KeysConfig{
			Up:     "up,k",
			Down:   "down,j",
			Home:   "home,g",
			End:    "end,G",
			Open:   "enter",
			New:    "n",
			Delete: "d",
			PR:     "p",
			Rename: "r",
			Filter: "/",
			Fetch:  "f",
			Detail: "tab",
			Help:   "?",
			Quit:   "q,ctrl+c",
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

// IsFirstRun returns true if no config file exists.
func IsFirstRun() bool {
	_, err := os.Stat(ConfigPath())
	return os.IsNotExist(err)
}

// DetectEnvironment detects the terminal multiplexer environment.
func DetectEnvironment() string {
	if os.Getenv("TMUX") != "" {
		return "tmux"
	}
	if os.Getenv("ZELLIJ") != "" {
		return "zellij"
	}
	return "generic"
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

// CreateDefaultConfigFile creates a default config file with comments.
func CreateDefaultConfigFile() error {
	path := ConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	env := DetectEnvironment()

	content := generateDefaultConfigContent(env)
	return os.WriteFile(path, []byte(content), 0644)
}

// generateDefaultConfigContent generates a commented config file.
func generateDefaultConfigContent(env string) string {
	var b strings.Builder

	b.WriteString("# Grove Configuration\n")
	b.WriteString("# See https://github.com/henrilemoine/grove for documentation\n\n")

	b.WriteString("[general]\n")
	b.WriteString("# Default base branch for new worktrees\n")
	b.WriteString("default_base_branch = \"main\"\n")
	b.WriteString("# Directory for worktrees (relative to repo root)\n")
	b.WriteString("worktree_dir = \".worktrees\"\n\n")

	b.WriteString("[open]\n")
	b.WriteString("# Command to run when opening a worktree\n")
	b.WriteString("# Template variables: {path}, {branch}, {branch_short}, {repo}, {window_name}\n")
	if env == "tmux" {
		b.WriteString("command = \"tmux new-window -n {branch_short} -c {path}\"\n")
	} else if env == "zellij" {
		b.WriteString("command = \"zellij action new-tab --name {branch_short} --cwd {path}\"\n")
	} else {
		b.WriteString("# command = \"tmux new-window -n {branch_short} -c {path}\"\n")
	}
	b.WriteString("# How to detect existing windows: \"path\", \"name\", or \"none\"\n")
	b.WriteString("detect_existing = \"name\"\n")
	b.WriteString("# Whether to exit grove after opening\n")
	b.WriteString("exit_after_open = true\n")
	b.WriteString("# Layout to apply after creating new window: \"none\", \"dev\", or \"custom\"\n")
	b.WriteString("layout = \"none\"\n")
	b.WriteString("# Custom layout command (only if layout = \"custom\")\n")
	b.WriteString("# layout_command = \"tmux split-window -h -p 50 -c {path}\"\n")
	b.WriteString("# Window name style: \"short\" or \"full\"\n")
	b.WriteString("window_name_style = \"short\"\n")
	b.WriteString("# Stash dirty worktree before switching\n")
	b.WriteString("stash_on_switch = false\n\n")

	b.WriteString("[pr]\n")
	b.WriteString("# Command to create PR\n")
	b.WriteString("command = \"gh pr create\"\n")
	b.WriteString("# Auto-push branch if no upstream\n")
	b.WriteString("auto_push = true\n\n")

	b.WriteString("[worktree]\n")
	b.WriteString("# File patterns to copy to new worktrees\n")
	b.WriteString("# copy_patterns = [\".env*\", \".vscode/**\"]\n")
	b.WriteString("# File patterns to ignore when copying\n")
	b.WriteString("# copy_ignores = [\"node_modules/**\"]\n")
	b.WriteString("# Commands to run after creating worktree\n")
	b.WriteString("# post_create_cmd = [\"npm install\"]\n\n")

	b.WriteString("# Template example:\n")
	b.WriteString("# [[worktree.templates]]\n")
	b.WriteString("# pattern = \"feature/*\"\n")
	b.WriteString("# copy_patterns = [\".env.local\"]\n")
	b.WriteString("# post_create_cmd = [\"npm install\", \"npm run setup\"]\n\n")

	b.WriteString("[safety]\n")
	b.WriteString("# Confirm before deleting dirty worktrees\n")
	b.WriteString("confirm_dirty = true\n")
	b.WriteString("# Confirm before deleting unmerged branches\n")
	b.WriteString("confirm_unmerged = true\n")
	b.WriteString("# Require typing \"delete\" for worktrees with unique commits\n")
	b.WriteString("require_typing_for_unique = true\n\n")

	b.WriteString("[ui]\n")
	b.WriteString("# Show branch type indicators in create flow\n")
	b.WriteString("show_branch_types = true\n")
	b.WriteString("# Show commit info\n")
	b.WriteString("show_commits = true\n")
	b.WriteString("# Show upstream tracking status\n")
	b.WriteString("show_upstream = true\n")
	b.WriteString("# Color theme: \"auto\", \"dark\", or \"light\"\n")
	b.WriteString("theme = \"auto\"\n\n")

	b.WriteString("[keys]\n")
	b.WriteString("# Keybindings (comma-separated for multiple keys)\n")
	b.WriteString("# up = \"up,k\"\n")
	b.WriteString("# down = \"down,j\"\n")
	b.WriteString("# open = \"enter\"\n")
	b.WriteString("# new = \"n\"\n")
	b.WriteString("# delete = \"d\"\n")
	b.WriteString("# pr = \"p\"\n")
	b.WriteString("# rename = \"r\"\n")
	b.WriteString("# filter = \"/\"\n")
	b.WriteString("# fetch = \"f\"\n")
	b.WriteString("# detail = \"tab\"\n")
	b.WriteString("# help = \"?\"\n")
	b.WriteString("# quit = \"q,ctrl+c\"\n")

	return b.String()
}

// Validate validates the configuration and returns warnings.
func (c *Config) Validate() []string {
	var warnings []string

	// Check template variables in command
	validVars := []string{"{path}", "{branch}", "{branch_short}", "{repo}", "{window_name}"}
	vars := extractTemplateVars(c.Open.Command)
	for _, v := range vars {
		found := false
		for _, valid := range validVars {
			if v == valid {
				found = true
				break
			}
		}
		if !found {
			warnings = append(warnings, fmt.Sprintf("Unknown template variable in open.command: %s", v))
		}
	}

	// Check layout command vars too
	if c.Open.LayoutCommand != "" {
		layoutVars := extractTemplateVars(c.Open.LayoutCommand)
		for _, v := range layoutVars {
			found := false
			for _, valid := range validVars {
				if v == valid {
					found = true
					break
				}
			}
			if !found {
				warnings = append(warnings, fmt.Sprintf("Unknown template variable in open.layout_command: %s", v))
			}
		}
	}

	// Check detect_existing value
	if c.Open.DetectExisting != "" &&
		c.Open.DetectExisting != "path" &&
		c.Open.DetectExisting != "name" &&
		c.Open.DetectExisting != "none" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for open.detect_existing: %s (expected path, name, or none)", c.Open.DetectExisting))
	}

	// Check layout value
	if c.Open.Layout != "" &&
		c.Open.Layout != "none" &&
		c.Open.Layout != "dev" &&
		c.Open.Layout != "custom" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for open.layout: %s (expected none, dev, or custom)", c.Open.Layout))
	}

	// Warn if layout is set but command doesn't look like tmux
	if c.Open.Layout != "" && c.Open.Layout != "none" {
		if !strings.Contains(c.Open.Command, "tmux") {
			warnings = append(warnings, "Layout is configured but open.command doesn't appear to use tmux")
		}
	}

	// Check window_name_style value
	if c.Open.WindowNameStyle != "" &&
		c.Open.WindowNameStyle != "short" &&
		c.Open.WindowNameStyle != "full" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for open.window_name_style: %s (expected short or full)", c.Open.WindowNameStyle))
	}

	// Check theme value
	if c.UI.Theme != "" &&
		c.UI.Theme != "auto" &&
		c.UI.Theme != "dark" &&
		c.UI.Theme != "light" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for ui.theme: %s (expected auto, dark, or light)", c.UI.Theme))
	}

	return warnings
}

// extractTemplateVars extracts template variables from a string.
func extractTemplateVars(s string) []string {
	re := regexp.MustCompile(`\{[^}]+\}`)
	return re.FindAllString(s, -1)
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
	if file.Open.DetectExisting != "" {
		base.Open.DetectExisting = file.Open.DetectExisting
	}
	// ExitAfterOpen is a bool, always use file value
	base.Open.ExitAfterOpen = file.Open.ExitAfterOpen
	if file.Open.Layout != "" {
		base.Open.Layout = file.Open.Layout
	}
	if file.Open.LayoutCommand != "" {
		base.Open.LayoutCommand = file.Open.LayoutCommand
	}
	if file.Open.WindowNameStyle != "" {
		base.Open.WindowNameStyle = file.Open.WindowNameStyle
	}
	base.Open.StashOnSwitch = file.Open.StashOnSwitch

	// PR
	if file.PR.Command != "" {
		base.PR.Command = file.PR.Command
	}
	base.PR.AutoPush = file.PR.AutoPush

	// Worktree
	if len(file.Worktree.CopyPatterns) > 0 {
		base.Worktree.CopyPatterns = file.Worktree.CopyPatterns
	}
	if len(file.Worktree.CopyIgnores) > 0 {
		base.Worktree.CopyIgnores = file.Worktree.CopyIgnores
	}
	if len(file.Worktree.PostCreateCmd) > 0 {
		base.Worktree.PostCreateCmd = file.Worktree.PostCreateCmd
	}
	if len(file.Worktree.Templates) > 0 {
		base.Worktree.Templates = file.Worktree.Templates
	}

	// Safety
	base.Safety.ConfirmDirty = file.Safety.ConfirmDirty
	base.Safety.ConfirmUnmerged = file.Safety.ConfirmUnmerged
	base.Safety.RequireTypingForUnique = file.Safety.RequireTypingForUnique

	// UI
	base.UI.ShowBranchTypes = file.UI.ShowBranchTypes
	base.UI.ShowCommits = file.UI.ShowCommits
	base.UI.ShowUpstream = file.UI.ShowUpstream
	if file.UI.Theme != "" {
		base.UI.Theme = file.UI.Theme
	}

	// Keys
	if file.Keys.Up != "" {
		base.Keys.Up = file.Keys.Up
	}
	if file.Keys.Down != "" {
		base.Keys.Down = file.Keys.Down
	}
	if file.Keys.Home != "" {
		base.Keys.Home = file.Keys.Home
	}
	if file.Keys.End != "" {
		base.Keys.End = file.Keys.End
	}
	if file.Keys.Open != "" {
		base.Keys.Open = file.Keys.Open
	}
	if file.Keys.New != "" {
		base.Keys.New = file.Keys.New
	}
	if file.Keys.Delete != "" {
		base.Keys.Delete = file.Keys.Delete
	}
	if file.Keys.PR != "" {
		base.Keys.PR = file.Keys.PR
	}
	if file.Keys.Rename != "" {
		base.Keys.Rename = file.Keys.Rename
	}
	if file.Keys.Filter != "" {
		base.Keys.Filter = file.Keys.Filter
	}
	if file.Keys.Fetch != "" {
		base.Keys.Fetch = file.Keys.Fetch
	}
	if file.Keys.Detail != "" {
		base.Keys.Detail = file.Keys.Detail
	}
	if file.Keys.Help != "" {
		base.Keys.Help = file.Keys.Help
	}
	if file.Keys.Quit != "" {
		base.Keys.Quit = file.Keys.Quit
	}
}

// GetTemplateForBranch returns the template that matches the branch name.
func (c *Config) GetTemplateForBranch(branch string) *TemplateConfig {
	for i := range c.Worktree.Templates {
		t := &c.Worktree.Templates[i]
		if matchGlobPattern(t.Pattern, branch) {
			return t
		}
	}
	return nil
}

// matchGlobPattern matches a branch name against a glob pattern.
func matchGlobPattern(pattern, name string) bool {
	// Convert glob to regex
	regexStr := "^"
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// ** matches anything including /
				regexStr += ".*"
				i++
			} else {
				// * matches anything except /
				regexStr += "[^/]*"
			}
		case '?':
			regexStr += "[^/]"
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			regexStr += "\\" + string(pattern[i])
		default:
			regexStr += string(pattern[i])
		}
	}
	regexStr += "$"

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return false
	}
	return re.MatchString(name)
}
