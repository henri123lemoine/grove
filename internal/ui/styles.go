// Package ui handles terminal UI rendering.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors - using more subtle, balanced palette
var (
	ColorPrimary   = lipgloss.Color("4")   // Blue
	ColorSecondary = lipgloss.Color("8")   // Gray
	ColorSuccess   = lipgloss.Color("2")   // Green (dimmer)
	ColorWarning   = lipgloss.Color("3")   // Yellow (dimmer)
	ColorDanger    = lipgloss.Color("1")   // Red (dimmer)
	ColorMuted     = lipgloss.Color("245") // Light gray
	ColorHighlight = lipgloss.Color("6")   // Cyan
	ColorText      = lipgloss.Color("252") // Light text
)

// Styles
var (
	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted)

	// Selected item style
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// Normal item style
	NormalStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Branch name style
	BranchStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Status styles - more subtle
	CleanStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	DirtyStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	DangerStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	MergedStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	AheadStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	UniqueStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// Path style - more readable
	PathStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Input style
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	// Current marker style
	CurrentStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Divider style
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)
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
)
