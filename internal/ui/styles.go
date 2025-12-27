// Package ui handles terminal UI rendering.
package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents a color theme
type Theme string

const (
	ThemeAuto  Theme = "auto"
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

// ColorPalette holds all theme colors
type ColorPalette struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Danger     lipgloss.Color
	Muted      lipgloss.Color
	Highlight  lipgloss.Color
	Text       lipgloss.Color
	Purple     lipgloss.Color
	Background lipgloss.Color
}

// Dark theme palette
var darkPalette = ColorPalette{
	Primary:    lipgloss.Color("4"),   // Blue
	Secondary:  lipgloss.Color("8"),   // Gray
	Success:    lipgloss.Color("2"),   // Green
	Warning:    lipgloss.Color("3"),   // Yellow
	Danger:     lipgloss.Color("1"),   // Red
	Muted:      lipgloss.Color("245"), // Light gray
	Highlight:  lipgloss.Color("6"),   // Cyan
	Text:       lipgloss.Color("252"), // Light text
	Purple:     lipgloss.Color("5"),   // Purple
	Background: lipgloss.Color("0"),   // Black
}

// Light theme palette
var lightPalette = ColorPalette{
	Primary:    lipgloss.Color("21"),  // Dark blue
	Secondary:  lipgloss.Color("244"), // Dark gray
	Success:    lipgloss.Color("22"),  // Dark green
	Warning:    lipgloss.Color("130"), // Dark yellow/orange
	Danger:     lipgloss.Color("124"), // Dark red
	Muted:      lipgloss.Color("243"), // Dark gray
	Highlight:  lipgloss.Color("30"),  // Dark cyan
	Text:       lipgloss.Color("235"), // Dark text
	Purple:     lipgloss.Color("91"),  // Dark purple
	Background: lipgloss.Color("255"), // White
}

// Current active palette
var activePalette = darkPalette

// Color variables - these reference the active palette
var (
	ColorPrimary   lipgloss.Color
	ColorSecondary lipgloss.Color
	ColorSuccess   lipgloss.Color
	ColorWarning   lipgloss.Color
	ColorDanger    lipgloss.Color
	ColorMuted     lipgloss.Color
	ColorHighlight lipgloss.Color
	ColorText      lipgloss.Color
	ColorPurple    lipgloss.Color
)

// Styles - will be initialized by InitTheme
var (
	BoxStyle         lipgloss.Style
	TitleStyle       lipgloss.Style
	HeaderStyle      lipgloss.Style
	SelectedStyle    lipgloss.Style
	NormalStyle      lipgloss.Style
	BranchStyle      lipgloss.Style
	CleanStyle       lipgloss.Style
	DirtyStyle       lipgloss.Style
	DangerStyle      lipgloss.Style
	MergedStyle      lipgloss.Style
	AheadStyle       lipgloss.Style
	BehindStyle      lipgloss.Style
	UniqueStyle      lipgloss.Style
	StashStyle       lipgloss.Style
	PathStyle        lipgloss.Style
	CommitStyle      lipgloss.Style
	HelpStyle        lipgloss.Style
	InputStyle       lipgloss.Style
	ErrorStyle       lipgloss.Style
	CurrentStyle     lipgloss.Style
	DividerStyle     lipgloss.Style
	WorktreeTagStyle lipgloss.Style
	LocalTagStyle    lipgloss.Style
	RemoteTagStyle   lipgloss.Style
	GitTagStyle      lipgloss.Style // For git tags (not branches)
)

// Symbols
const (
	SymbolCursor  = "›"
	SymbolClean   = "✓"
	SymbolDirty   = "●"
	SymbolAhead   = "↑"
	SymbolBehind  = "↓"
	SymbolMerged  = "✓"
	SymbolUnique  = "!"
	SymbolCurrent = "•"
	SymbolDivider = "─"
	SymbolStash   = "📦"
)

// init initializes styles with default dark theme
func init() {
	InitTheme("auto")
}

// InitTheme initializes styles based on the theme setting
func InitTheme(theme string) {
	switch Theme(theme) {
	case ThemeDark:
		activePalette = darkPalette
	case ThemeLight:
		activePalette = lightPalette
	case ThemeAuto:
		fallthrough
	default:
		activePalette = detectTheme()
	}

	// Set color variables from active palette
	ColorPrimary = activePalette.Primary
	ColorSecondary = activePalette.Secondary
	ColorSuccess = activePalette.Success
	ColorWarning = activePalette.Warning
	ColorDanger = activePalette.Danger
	ColorMuted = activePalette.Muted
	ColorHighlight = activePalette.Highlight
	ColorText = activePalette.Text
	ColorPurple = activePalette.Purple

	// Initialize all styles with new colors
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorMuted)

	SelectedStyle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Bold(true)

	NormalStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	BranchStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	CleanStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	DirtyStyle = lipgloss.NewStyle().
		Foreground(ColorWarning)

	DangerStyle = lipgloss.NewStyle().
		Foreground(ColorDanger)

	MergedStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	AheadStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	BehindStyle = lipgloss.NewStyle().
		Foreground(ColorWarning)

	UniqueStyle = lipgloss.NewStyle().
		Foreground(ColorDanger)

	StashStyle = lipgloss.NewStyle().
		Foreground(ColorPurple)

	PathStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	CommitStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		Faint(true)

	HelpStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	InputStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger)

	CurrentStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	DividerStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	WorktreeTagStyle = lipgloss.NewStyle().
		Foreground(ColorHighlight)

	LocalTagStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	RemoteTagStyle = lipgloss.NewStyle().
		Foreground(ColorWarning)

	GitTagStyle = lipgloss.NewStyle().
		Foreground(ColorPurple)
}

// detectTheme tries to detect whether the terminal has a light or dark background
func detectTheme() ColorPalette {
	// Check COLORFGBG environment variable (set by some terminals)
	// Format: "foreground;background" where light bg is usually >= 7
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			bg := parts[len(parts)-1]
			// Light backgrounds are typically 7, 15, or high numbers
			if bg == "7" || bg == "15" || (len(bg) > 0 && bg[0] >= '1' && bg[0] <= '9') {
				// Check if it's a high number (> 100 usually indicates light)
				if len(bg) >= 3 {
					return lightPalette
				}
				if bg == "7" || bg == "15" {
					return lightPalette
				}
			}
		}
	}

	// Check for common light theme indicators
	colorTerm := os.Getenv("COLORTERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Some terminals set specific vars for light themes
	if strings.Contains(strings.ToLower(colorTerm), "light") {
		return lightPalette
	}

	// macOS Terminal.app - we can't reliably detect its theme,
	// so fall through to dark as the safer default for most terminals
	_ = termProgram // Acknowledge we checked it

	// Default to dark theme (most common in terminals)
	return darkPalette
}
