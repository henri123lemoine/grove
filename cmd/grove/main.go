package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/henri123lemoine/grove/internal/app"
	"github.com/henri123lemoine/grove/internal/config"
	"github.com/henri123lemoine/grove/internal/git"
	"github.com/henri123lemoine/grove/internal/ui"
)

var (
	version = "dev"
)

func main() {
	// Parse flags
	printSelected := flag.Bool("print-selected", false, "Print the selected worktree path on exit")
	printPath := flag.Bool("p", false, "Alias for --print-selected")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")
	flag.BoolVar(showHelp, "h", false, "Show help")
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("grove %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Get config validation warnings (will be displayed in TUI)
	configWarnings := cfg.Validate()

	// Initialize theme based on config
	ui.InitTheme(cfg.UI.Theme)

	// Handle first-run experience
	if config.IsFirstRun() {
		if err := config.CreateDefaultConfigFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create default config: %v\n", err)
		}
	}

	// Detect git repository
	repo, err := git.GetRepo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "grove must be run from within a git repository.\n")
		os.Exit(1)
	}

	// Update default branch detection if a specific remote is configured
	if cfg.General.Remote != "" {
		git.UpdateDefaultBranch(cfg.General.Remote)
	}

	// Create and run the application
	model := app.New(cfg, repo, configWarnings)
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if we need to do anything after quitting
	if m, ok := finalModel.(app.Model); ok {
		// Print selected worktree path if requested
		if (*printSelected || *printPath) && m.SelectedWorktree() != nil {
			fmt.Println(m.SelectedWorktree().Path)
		}

		if m.ShouldQuit() {
			os.Exit(0)
		}
	}
}

func printUsage() {
	fmt.Println(`grove - Terminal UI for Git worktrees

Usage:
  grove [flags]

Flags:
  -p, --print-selected  Print the selected worktree path on exit
                        Useful for shell integration: cd "$(grove -p)"
  --version             Show version
  -h, --help            Show this help

Navigation:
  ↑/k          Move up
  ↓/j          Move down
  g/Home       Go to first
  G/End        Go to last
  enter        Open worktree

Actions:
  n            New worktree
  d            Delete worktree
  r            Rename branch
  f            Fetch all remotes
  /            Filter worktrees
  tab          Toggle detail panel

General:
  ?            Show help
  q            Quit
  esc          Cancel

Configuration:
  Config file: ~/.config/grove/config.toml

For more information, see https://github.com/henri123lemoine/grove`)
}
