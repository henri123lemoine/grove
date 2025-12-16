package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/henri123lemoine/grove/internal/config"
)

// KeyMap defines all keybindings.
type KeyMap struct {
	// Navigation
	Up   key.Binding
	Down key.Binding
	Home key.Binding
	End  key.Binding

	// Actions
	Open   key.Binding
	New    key.Binding
	Delete key.Binding
	PR     key.Binding
	Rename key.Binding
	Fetch  key.Binding
	Filter key.Binding
	Detail key.Binding
	Prune  key.Binding
	Stash  key.Binding

	// General
	Confirm key.Binding
	Cancel  key.Binding
	Quit    key.Binding
	Help    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "first"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "last"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		PR: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "create PR"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		Fetch: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fetch"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Detail: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "details"),
		),
		Prune: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "prune"),
		),
		Stash: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stash"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// KeyMapFromConfig creates a KeyMap from config settings.
func KeyMapFromConfig(cfg *config.KeysConfig) KeyMap {
	km := DefaultKeyMap()

	if cfg.Up != "" {
		km.Up = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Up)...),
			key.WithHelp(cfg.Up, "up"),
		)
	}
	if cfg.Down != "" {
		km.Down = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Down)...),
			key.WithHelp(cfg.Down, "down"),
		)
	}
	if cfg.Home != "" {
		km.Home = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Home)...),
			key.WithHelp(cfg.Home, "first"),
		)
	}
	if cfg.End != "" {
		km.End = key.NewBinding(
			key.WithKeys(parseKeys(cfg.End)...),
			key.WithHelp(cfg.End, "last"),
		)
	}
	if cfg.Open != "" {
		km.Open = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Open)...),
			key.WithHelp(cfg.Open, "open"),
		)
	}
	if cfg.New != "" {
		km.New = key.NewBinding(
			key.WithKeys(parseKeys(cfg.New)...),
			key.WithHelp(cfg.New, "new"),
		)
	}
	if cfg.Delete != "" {
		km.Delete = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Delete)...),
			key.WithHelp(cfg.Delete, "delete"),
		)
	}
	if cfg.PR != "" {
		km.PR = key.NewBinding(
			key.WithKeys(parseKeys(cfg.PR)...),
			key.WithHelp(cfg.PR, "create PR"),
		)
	}
	if cfg.Rename != "" {
		km.Rename = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Rename)...),
			key.WithHelp(cfg.Rename, "rename"),
		)
	}
	if cfg.Filter != "" {
		km.Filter = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Filter)...),
			key.WithHelp(cfg.Filter, "filter"),
		)
	}
	if cfg.Fetch != "" {
		km.Fetch = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Fetch)...),
			key.WithHelp(cfg.Fetch, "fetch"),
		)
	}
	if cfg.Detail != "" {
		km.Detail = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Detail)...),
			key.WithHelp(cfg.Detail, "details"),
		)
	}
	if cfg.Prune != "" {
		km.Prune = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Prune)...),
			key.WithHelp(cfg.Prune, "prune"),
		)
	}
	if cfg.Stash != "" {
		km.Stash = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Stash)...),
			key.WithHelp(cfg.Stash, "stash"),
		)
	}
	if cfg.Help != "" {
		km.Help = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Help)...),
			key.WithHelp(cfg.Help, "help"),
		)
	}
	if cfg.Quit != "" {
		km.Quit = key.NewBinding(
			key.WithKeys(parseKeys(cfg.Quit)...),
			key.WithHelp(cfg.Quit, "quit"),
		)
	}

	return km
}

// parseKeys parses a comma-separated list of keys.
func parseKeys(s string) []string {
	parts := strings.Split(s, ",")
	var keys []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keys = append(keys, p)
		}
	}
	return keys
}
