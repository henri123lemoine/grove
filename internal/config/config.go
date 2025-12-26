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
	Delete   DeleteConfig   `toml:"delete"`
	Worktree WorktreeConfig `toml:"worktree"`
	Safety   SafetyConfig   `toml:"safety"`
	UI       UIConfig       `toml:"ui"`
	Keys     KeysConfig     `toml:"keys"`
	Layouts  []LayoutConfig `toml:"layouts"`
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

// DeleteConfig contains settings for worktree deletion.
type DeleteConfig struct {
	// What to do with terminal window/tab when deleting a worktree: "auto", "ask", "never"
	// "auto" - automatically close the window/tab
	// "ask" - prompt before closing
	// "never" - don't close the window/tab
	// Works with tmux (windows) and zellij (tabs)
	CloseWindowAction string `toml:"close_window_action"`
}

// WorktreeConfig contains settings for worktree creation.
type WorktreeConfig struct {
	// File patterns to copy to new worktrees (e.g., ".env*", ".vscode/**")
	CopyPatterns []string `toml:"copy_patterns"`

	// File patterns to ignore when copying
	CopyIgnores []string `toml:"copy_ignores"`

	// Commands to run after creating worktree
	PostCreateCmd []string `toml:"post_create_cmd"`

	// Timeout for post-create hooks in seconds (0 = no timeout)
	HookTimeout int `toml:"hook_timeout"`

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

// PaneConfig defines a pane in a layout.
type PaneConfig struct {
	// Which pane to split from (0 = first/main pane)
	SplitFrom int `toml:"split_from"`

	// Split direction: "right", "down", "left", "up"
	Direction string `toml:"direction"`

	// Size as percentage (1-99)
	Size int `toml:"size"`

	// Command to run in this pane (template vars supported)
	Command string `toml:"command"`
}

// LayoutConfig defines a named layout with multiple panes.
type LayoutConfig struct {
	// Unique name for this layout
	Name string `toml:"name"`

	// Human-readable description
	Description string `toml:"description"`

	// Pane definitions (first pane is the initial window)
	Panes []PaneConfig `toml:"panes"`
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
	Rename string `toml:"rename"`
	Filter string `toml:"filter"`
	Fetch  string `toml:"fetch"`
	Detail string `toml:"detail"`
	Prune  string `toml:"prune"`
	Stash  string `toml:"stash"`
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
			Command:         "tmux new-window -n {branch_short} -c {path}",
			DetectExisting:  "path",
			ExitAfterOpen:   true,
			Layout:          "none",
			LayoutCommand:   "",
			WindowNameStyle: "short",
			StashOnSwitch:   false,
		},
		Delete: DeleteConfig{
			CloseWindowAction: "ask",
		},
		Worktree: WorktreeConfig{
			CopyPatterns:  []string{},
			CopyIgnores:   []string{},
			PostCreateCmd: []string{},
			HookTimeout:   300, // 5 minutes default
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
			Rename: "r",
			Filter: "/",
			Fetch:  "f",
			Detail: "tab",
			Prune:  "P",
			Stash:  "s",
			Help:   "?",
			Quit:   "q,ctrl+c",
		},
		Layouts: []LayoutConfig{},
	}
}

// GetLayoutByName returns the layout with the given name, or nil if not found.
func (c *Config) GetLayoutByName(name string) *LayoutConfig {
	for i := range c.Layouts {
		if c.Layouts[i].Name == name {
			return &c.Layouts[i]
		}
	}
	return nil
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
	return LoadFromPath(ConfigPath())
}

// LoadFromPath loads configuration from a specific path.
func LoadFromPath(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file, use defaults
			return cfg, nil
		}
		return nil, err
	}

	// Unmarshal directly into default config.
	// go-toml/v2 only overwrites fields present in the TOML file,
	// preserving defaults for unspecified fields (including booleans).
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

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

	b.WriteString("[delete]\n")
	b.WriteString("# What to do with terminal window/tab when deleting a worktree\n")
	b.WriteString("# Works with tmux (windows) and zellij (tabs)\n")
	b.WriteString("# \"auto\" - automatically close the window/tab\n")
	b.WriteString("# \"ask\" - prompt before closing (default)\n")
	b.WriteString("# \"never\" - don't close the window/tab\n")
	b.WriteString("close_window_action = \"ask\"\n\n")

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

	// Check delete.close_window_action value
	if c.Delete.CloseWindowAction != "" &&
		c.Delete.CloseWindowAction != "auto" &&
		c.Delete.CloseWindowAction != "ask" &&
		c.Delete.CloseWindowAction != "never" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for delete.close_window_action: %s (expected auto, ask, or never)", c.Delete.CloseWindowAction))
	}

	// Check theme value
	if c.UI.Theme != "" &&
		c.UI.Theme != "auto" &&
		c.UI.Theme != "dark" &&
		c.UI.Theme != "light" {
		warnings = append(warnings, fmt.Sprintf("Invalid value for ui.theme: %s (expected auto, dark, or light)", c.UI.Theme))
	}

	// Validate layouts
	layoutNames := make(map[string]bool)
	validDirections := map[string]bool{"right": true, "down": true, "left": true, "up": true, "": true}
	for _, layout := range c.Layouts {
		// Check for duplicate names
		if layoutNames[layout.Name] {
			warnings = append(warnings, fmt.Sprintf("Duplicate layout name: %s", layout.Name))
		}
		layoutNames[layout.Name] = true

		// Check for empty name
		if layout.Name == "" {
			warnings = append(warnings, "Layout has empty name")
		}

		// Validate panes
		for i, pane := range layout.Panes {
			// Check direction is valid
			if !validDirections[pane.Direction] {
				warnings = append(warnings, fmt.Sprintf("Layout %s pane %d: invalid direction '%s' (expected right, down, left, up)", layout.Name, i, pane.Direction))
			}

			// Check split_from is valid (first pane shouldn't split from anything)
			if i == 0 && pane.SplitFrom != 0 && pane.Direction != "" {
				warnings = append(warnings, fmt.Sprintf("Layout %s pane 0: first pane should not have split_from set", layout.Name))
			}
			if i > 0 && pane.SplitFrom >= i {
				warnings = append(warnings, fmt.Sprintf("Layout %s pane %d: split_from (%d) must reference an earlier pane", layout.Name, i, pane.SplitFrom))
			}

			// Check size is valid (1-99)
			if pane.Size != 0 && (pane.Size < 1 || pane.Size > 99) {
				warnings = append(warnings, fmt.Sprintf("Layout %s pane %d: size must be 1-99, got %d", layout.Name, i, pane.Size))
			}

			// Check template vars in command
			if pane.Command != "" {
				paneVars := extractTemplateVars(pane.Command)
				for _, v := range paneVars {
					found := false
					for _, valid := range validVars {
						if v == valid {
							found = true
							break
						}
					}
					if !found {
						warnings = append(warnings, fmt.Sprintf("Layout %s pane %d: unknown template variable %s", layout.Name, i, v))
					}
				}
			}
		}
	}

	return warnings
}

// extractTemplateVars extracts template variables from a string.
func extractTemplateVars(s string) []string {
	re := regexp.MustCompile(`\{[^}]+\}`)
	return re.FindAllString(s, -1)
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
