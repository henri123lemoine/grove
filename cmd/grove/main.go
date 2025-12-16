package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/henrilemoine/grove/internal/app"
	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Detect git repository
	repo, err := git.GetRepo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "grove must be run from within a git repository.\n")
		os.Exit(1)
	}

	// Create and run the application
	model := app.New(cfg, repo)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if we need to do anything after quitting
	if m, ok := finalModel.(app.Model); ok {
		if m.ShouldQuit() {
			// Normal exit
			os.Exit(0)
		}
	}
}
