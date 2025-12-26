package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
)

func TestNewModel(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)

	if model.state != StateList {
		t.Errorf("Expected initial state StateList, got %d", model.state)
	}

	if !model.loading {
		t.Error("Expected loading to be true initially")
	}

	if model.config != cfg {
		t.Error("Config not set correctly")
	}

	if model.repo != repo {
		t.Error("Repo not set correctly")
	}
}

func TestStateTransitions(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{
		{Path: "/test/repo", Branch: "main", IsMain: true},
		{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
	}
	model.filteredWorktrees = model.worktrees

	// Test transitions from StateList

	// Press 'n' for new
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := newModel.(Model)
	if m.state != StateCreate {
		t.Errorf("Expected StateCreate after 'n', got %d", m.state)
	}

	// Press 'esc' to go back
	backModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = backModel.(Model)
	if m.state != StateList {
		t.Errorf("Expected StateList after 'esc', got %d", m.state)
	}

	// Press '?' for help
	helpModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = helpModel.(Model)
	if m.state != StateHelp {
		t.Errorf("Expected StateHelp after '?', got %d", m.state)
	}

	// Press any key to close help
	closeModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = closeModel.(Model)
	if m.state != StateList {
		t.Errorf("Expected StateList after closing help, got %d", m.state)
	}

	// Press '/' for filter
	filterModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = filterModel.(Model)
	if m.state != StateFilter {
		t.Errorf("Expected StateFilter after '/', got %d", m.state)
	}

	// Press 'esc' to exit filter
	exitFilterModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = exitFilterModel.(Model)
	if m.state != StateList {
		t.Errorf("Expected StateList after exiting filter, got %d", m.state)
	}
}

func TestCursorNavigation(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{
		{Path: "/test/repo", Branch: "main"},
		{Path: "/test/repo/.worktrees/feature1", Branch: "feature1"},
		{Path: "/test/repo/.worktrees/feature2", Branch: "feature2"},
	}
	model.filteredWorktrees = model.worktrees
	model.cursor = 0

	// Move down
	downModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := downModel.(Model)
	if m.cursor != 1 {
		t.Errorf("Expected cursor 1 after down, got %d", m.cursor)
	}

	// Move down again
	downModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = downModel.(Model)
	if m.cursor != 2 {
		t.Errorf("Expected cursor 2 after 'j', got %d", m.cursor)
	}

	// Can't move down past last item
	downModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = downModel.(Model)
	if m.cursor != 2 {
		t.Errorf("Expected cursor 2 (clamped), got %d", m.cursor)
	}

	// Move up
	upModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = upModel.(Model)
	if m.cursor != 1 {
		t.Errorf("Expected cursor 1 after up, got %d", m.cursor)
	}

	// Move up with 'k'
	upModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = upModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Expected cursor 0 after 'k', got %d", m.cursor)
	}

	// Can't move up past first item
	upModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = upModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Expected cursor 0 (clamped), got %d", m.cursor)
	}

	// Go to end
	m.cursor = 0
	endModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = endModel.(Model)
	if m.cursor != 2 {
		t.Errorf("Expected cursor 2 after 'G', got %d", m.cursor)
	}

	// Go to home
	homeModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = homeModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Expected cursor 0 after 'g', got %d", m.cursor)
	}
}

func TestFuzzyFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{
		{Path: "/test/repo", Branch: "main"},
		{Path: "/test/repo/.worktrees/feature-auth", Branch: "feature-auth"},
		{Path: "/test/repo/.worktrees/feature-payment", Branch: "feature-payment"},
		{Path: "/test/repo/.worktrees/fix-bug", Branch: "fix-bug"},
	}
	model.filteredWorktrees = model.worktrees

	// Filter by "feature"
	model.filterInput.SetValue("feature")
	model.applyFilter()

	if len(model.filteredWorktrees) != 2 {
		t.Errorf("Expected 2 filtered worktrees, got %d", len(model.filteredWorktrees))
	}

	// Filter by "auth"
	model.filterInput.SetValue("auth")
	model.applyFilter()

	if len(model.filteredWorktrees) != 1 {
		t.Errorf("Expected 1 filtered worktree, got %d", len(model.filteredWorktrees))
	}

	// Clear filter
	model.filterInput.SetValue("")
	model.applyFilter()

	if len(model.filteredWorktrees) != 4 {
		t.Errorf("Expected 4 worktrees after clearing filter, got %d", len(model.filteredWorktrees))
	}
}

func TestWindowSizeMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(Model)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

func TestWorktreesLoadedMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = true

	worktrees := []git.Worktree{
		{Path: "/test/repo", Branch: "main"},
		{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
	}

	newModel, _ := model.Update(WorktreesLoadedMsg{Worktrees: worktrees, Err: nil})
	m := newModel.(Model)

	if m.loading {
		t.Error("Expected loading to be false after WorktreesLoadedMsg")
	}

	if len(m.worktrees) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(m.worktrees))
	}
}

func TestDeleteFlowCannotDeleteMain(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{
		{Path: "/test/repo", Branch: "main", IsMain: true},
	}
	model.filteredWorktrees = model.worktrees
	model.cursor = 0

	// Try to delete main worktree
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m := newModel.(Model)

	// Should stay in list state and have an error
	if m.state != StateList {
		t.Errorf("Expected to stay in StateList, got %d", m.state)
	}

	if m.err == nil {
		t.Error("Expected error when trying to delete main worktree")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/auth", "feature/auth"},
		{"feature\\windows", "feature-windows"},
		{"my branch", "my-branch"},
		{"test:colon", "test-colon"},
		{"normal-branch", "normal-branch"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestKeyMapFromConfig(t *testing.T) {
	keysConfig := &config.KeysConfig{
		Up:   "up,k,w",
		Down: "down,j,s",
		Open: "enter,o",
		Quit: "q,ctrl+c,esc",
	}

	km := KeyMapFromConfig(keysConfig)

	// Check that custom keys work
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}, km.Up) {
		t.Error("Expected 'w' to match Up binding")
	}

	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, km.Down) {
		t.Error("Expected 's' to match Down binding")
	}

	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}, km.Open) {
		t.Error("Expected 'o' to match Open binding")
	}
}

func TestDetailToggle(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{
		{Path: "/test/repo", Branch: "main"},
	}
	model.filteredWorktrees = model.worktrees

	if model.showDetail {
		t.Error("Expected showDetail to be false initially")
	}

	// Toggle detail on
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := newModel.(Model)

	if !m.showDetail {
		t.Error("Expected showDetail to be true after tab")
	}

	// Toggle detail off
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)

	if m.showDetail {
		t.Error("Expected showDetail to be false after second tab")
	}
}

func TestShouldQuit(t *testing.T) {
	cfg := config.DefaultConfig()
	repo := &git.Repo{
		Root:             "/test/repo",
		GitDir:           "/test/repo/.git",
		MainWorktreeRoot: "/test/repo",
		DefaultBranch:    "main",
	}

	model := New(cfg, repo)
	model.loading = false
	model.worktrees = []git.Worktree{}
	model.filteredWorktrees = model.worktrees

	if model.ShouldQuit() {
		t.Error("ShouldQuit should be false initially")
	}

	// Press 'q' to quit
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(Model)

	if !m.ShouldQuit() {
		t.Error("ShouldQuit should be true after 'q'")
	}
}
