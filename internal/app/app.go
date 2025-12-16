package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/exec"
	"github.com/henrilemoine/grove/internal/git"
	"github.com/henrilemoine/grove/internal/ui"
)

// State represents the current UI state.
type State int

const (
	StateList State = iota
	StateCreate
	StateCreateSelectBase
	StateDelete
	StateFilter
	StateFetching
	StateHelp
)

// Model is the main application model.
type Model struct {
	// Configuration
	config *config.Config
	repo   *git.Repo

	// Data
	worktrees         []git.Worktree
	filteredWorktrees []git.Worktree
	branches          []git.Branch
	cursor            int

	// State
	state   State
	loading bool
	err     error

	// Create flow
	createInput     textinput.Model
	createBranch    string
	createIsNew     bool
	baseBranchIndex int

	// Delete flow
	deleteWorktree *git.Worktree
	safetyInfo     *git.SafetyInfo
	deleteInput    textinput.Model

	// Filter
	filterInput textinput.Model

	// UI
	width      int
	height     int
	keys       KeyMap
	showDetail bool

	// Exit behavior
	shouldQuit    bool
	openAfterQuit *git.Worktree
}

// New creates a new Model.
func New(cfg *config.Config, repo *git.Repo) Model {
	// Create text inputs
	createInput := textinput.New()
	createInput.Placeholder = "branch-name"
	createInput.CharLimit = 100

	deleteInput := textinput.New()
	deleteInput.Placeholder = "Type 'delete' to confirm"
	deleteInput.CharLimit = 10

	filterInput := textinput.New()
	filterInput.Placeholder = "filter..."
	filterInput.CharLimit = 50

	return Model{
		config:      cfg,
		repo:        repo,
		keys:        DefaultKeyMap(),
		createInput: createInput,
		deleteInput: deleteInput,
		filterInput: filterInput,
		state:       StateList,
		loading:     true,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadWorktrees,
		loadBranches,
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle quit globally
		if key.Matches(msg, m.keys.Quit) && m.state == StateList {
			m.shouldQuit = true
			return m, tea.Quit
		}

		// Delegate to state-specific handler
		return m.handleKeyPress(msg)

	case WorktreesLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.worktrees = msg.Worktrees
		m.applyFilter()
		return m, nil

	case BranchesLoadedMsg:
		if msg.Err != nil {
			// Non-fatal, just continue
			return m, nil
		}
		m.branches = msg.Branches
		return m, nil

	case SafetyCheckedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.state = StateList
			return m, nil
		}
		m.safetyInfo = msg.Info
		// Focus delete input if danger level
		if msg.Info.Level == git.SafetyLevelDanger {
			m.deleteInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case WorktreeCreatedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.state = StateList
		m.createInput.Reset()
		return m, loadWorktrees

	case WorktreeDeletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.state = StateList
		m.deleteInput.Reset()
		m.deleteWorktree = nil
		m.safetyInfo = nil
		return m, loadWorktrees

	case WorktreeOpenedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		if m.config.Open.ExitAfterOpen {
			m.shouldQuit = true
			return m, tea.Quit
		}
		return m, nil

	case FetchCompletedMsg:
		m.state = StateList
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		return m, loadWorktrees
	}

	return m, nil
}

// handleKeyPress handles key presses based on current state.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateList:
		return m.handleListKeys(msg)
	case StateCreate:
		return m.handleCreateKeys(msg)
	case StateCreateSelectBase:
		return m.handleSelectBaseKeys(msg)
	case StateDelete:
		return m.handleDeleteKeys(msg)
	case StateFilter:
		return m.handleFilterKeys(msg)
	case StateHelp:
		return m.handleHelpKeys(msg)
	}
	return m, nil
}

// handleListKeys handles key presses in the list view.
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredWorktrees)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Home):
		m.cursor = 0
	case key.Matches(msg, m.keys.End):
		m.cursor = len(m.filteredWorktrees) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.Open):
		if len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := &m.filteredWorktrees[m.cursor]
			return m, openWorktree(m.config.Open.Command, wt)
		}
	case key.Matches(msg, m.keys.New):
		m.state = StateCreate
		m.createInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Delete):
		if len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := &m.filteredWorktrees[m.cursor]
			if wt.IsMain {
				m.err = fmt.Errorf("cannot delete main worktree")
				return m, nil
			}
			m.deleteWorktree = wt
			m.state = StateDelete
			return m, checkSafety(wt.Path, wt.Branch, m.repo.DefaultBranch)
		}
	case key.Matches(msg, m.keys.Filter):
		m.state = StateFilter
		m.filterInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Fetch):
		m.state = StateFetching
		return m, fetchAll
	case key.Matches(msg, m.keys.Help):
		m.state = StateHelp
		return m, nil
	case key.Matches(msg, m.keys.Detail):
		m.showDetail = !m.showDetail
		return m, nil
	}
	return m, nil
}

// handleHelpKeys handles key presses in the help view.
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key closes help
	m.state = StateList
	return m, nil
}

// handleCreateKeys handles key presses in the create flow.
func (m Model) handleCreateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.createInput.Reset()
		return m, nil
	case tea.KeyEnter:
		branchName := m.createInput.Value()
		if branchName == "" {
			return m, nil
		}
		m.createBranch = branchName
		// Check if branch exists
		m.createIsNew = !git.BranchExists(branchName)
		if m.createIsNew {
			m.state = StateCreateSelectBase
			m.baseBranchIndex = 0
			return m, nil
		}
		// Branch exists, create worktree
		return m, createWorktree(m.config, branchName, false, "")
	}

	var cmd tea.Cmd
	m.createInput, cmd = m.createInput.Update(msg)
	return m, cmd
}

// handleSelectBaseKeys handles key presses when selecting base branch.
func (m Model) handleSelectBaseKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.baseBranchIndex > 0 {
			m.baseBranchIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.baseBranchIndex < len(m.branches)-1 {
			m.baseBranchIndex++
		}
	case key.Matches(msg, m.keys.Cancel):
		m.state = StateList
		m.createInput.Reset()
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		baseBranch := ""
		if m.baseBranchIndex < len(m.branches) {
			baseBranch = m.branches[m.baseBranchIndex].Name
		}
		return m, createWorktree(m.config, m.createBranch, true, baseBranch)
	}
	return m, nil
}

// handleDeleteKeys handles key presses in the delete confirmation.
func (m Model) handleDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.safetyInfo == nil {
		// Still loading safety info
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.deleteInput.Reset()
		m.deleteWorktree = nil
		m.safetyInfo = nil
		return m, nil
	case tea.KeyEnter:
		// Check if we need to type "delete"
		if m.safetyInfo.Level == git.SafetyLevelDanger {
			if m.deleteInput.Value() != "delete" {
				return m, nil
			}
		}
		// Proceed with deletion
		force := m.safetyInfo.HasUncommittedChanges
		return m, deleteWorktree(m.deleteWorktree.Path, force)
	}

	// If danger level, handle typing
	if m.safetyInfo.Level == git.SafetyLevelDanger {
		var cmd tea.Cmd
		m.deleteInput, cmd = m.deleteInput.Update(msg)
		return m, cmd
	}

	// For safe/warning, y confirms, n cancels
	if msg.String() == "y" || msg.String() == "Y" {
		force := m.safetyInfo.HasUncommittedChanges
		return m, deleteWorktree(m.deleteWorktree.Path, force)
	}
	if msg.String() == "n" || msg.String() == "N" {
		m.state = StateList
		m.deleteInput.Reset()
		m.deleteWorktree = nil
		m.safetyInfo = nil
		return m, nil
	}

	return m, nil
}

// handleFilterKeys handles key presses in filter mode.
func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.filterInput.Reset()
		m.applyFilter()
		return m, nil
	case tea.KeyEnter:
		m.state = StateList
		m.filterInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

// worktreeSource implements fuzzy.Source for worktree fuzzy matching.
type worktreeSource []git.Worktree

func (w worktreeSource) String(i int) string {
	// Match against both branch name and path for better results
	return w[i].Branch + " " + w[i].Path
}

func (w worktreeSource) Len() int {
	return len(w)
}

// applyFilter filters worktrees based on current filter input using fuzzy matching.
func (m *Model) applyFilter() {
	filter := m.filterInput.Value()
	if filter == "" {
		m.filteredWorktrees = m.worktrees
	} else {
		source := worktreeSource(m.worktrees)
		matches := fuzzy.FindFrom(filter, source)

		m.filteredWorktrees = nil
		for _, match := range matches {
			m.filteredWorktrees = append(m.filteredWorktrees, m.worktrees[match.Index])
		}
	}

	// Ensure cursor is in bounds
	if m.cursor >= len(m.filteredWorktrees) {
		m.cursor = len(m.filteredWorktrees) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// View renders the UI.
func (m Model) View() string {
	return ui.Render(ui.RenderParams{
		State:             int(m.state),
		Worktrees:         m.filteredWorktrees,
		Cursor:            m.cursor,
		Width:             m.width,
		Height:            m.height,
		Loading:           m.loading,
		Err:               m.err,
		Repo:              m.repo,
		FilterInput:       m.filterInput.View(),
		FilterValue:       m.filterInput.Value(),
		CreateInput:       m.createInput.View(),
		DeleteWorktree:    m.deleteWorktree,
		SafetyInfo:        m.safetyInfo,
		DeleteInput:       m.deleteInput.View(),
		ShowDetail:        m.showDetail,
		Branches:          m.branches,
		BaseBranchIndex:   m.baseBranchIndex,
		CreateBranch:      m.createBranch,
	})
}

// ShouldQuit returns true if the app should quit.
func (m Model) ShouldQuit() bool {
	return m.shouldQuit
}

// OpenAfterQuit returns the worktree to open after quitting.
func (m Model) OpenAfterQuit() *git.Worktree {
	return m.openAfterQuit
}

// Commands

func loadWorktrees() tea.Msg {
	worktrees, err := git.List()
	return WorktreesLoadedMsg{Worktrees: worktrees, Err: err}
}

func loadBranches() tea.Msg {
	branches, err := git.ListBranches()
	return BranchesLoadedMsg{Branches: branches, Err: err}
}

func checkSafety(path, branch, defaultBranch string) tea.Cmd {
	return func() tea.Msg {
		info, err := git.CheckSafety(path, branch, defaultBranch)
		return SafetyCheckedMsg{Info: info, Err: err}
	}
}

func createWorktree(cfg *config.Config, branch string, isNew bool, baseBranch string) tea.Cmd {
	return func() tea.Msg {
		repo, _ := git.GetRepo()
		path := cfg.General.WorktreeDir + "/" + sanitizePath(branch)
		if repo != nil {
			// Always use MainWorktreeRoot so worktrees are created at the project root
			path = repo.MainWorktreeRoot + "/" + cfg.General.WorktreeDir + "/" + sanitizePath(branch)
		}
		err := git.Create(path, branch, isNew, baseBranch)
		return WorktreeCreatedMsg{Path: path, Err: err}
	}
}

func deleteWorktree(path string, force bool) tea.Cmd {
	return func() tea.Msg {
		err := git.Remove(path, force)
		return WorktreeDeletedMsg{Path: path, Err: err}
	}
}

func openWorktree(command string, wt *git.Worktree) tea.Cmd {
	return func() tea.Msg {
		err := exec.OpenDetached(command, wt)
		return WorktreeOpenedMsg{Err: err}
	}
}

func fetchAll() tea.Msg {
	err := git.FetchAll()
	return FetchCompletedMsg{Err: err}
}

// Helper functions

func sanitizePath(branch string) string {
	// Keep the branch name structure intact (including slashes)
	// Only sanitize truly problematic characters
	result := branch
	for _, c := range []string{"\\", " ", ":"} {
		result = replaceAll(result, c, "-")
	}
	return result
}

func replaceAll(s, old, new string) string {
	for {
		next := replace(s, old, new)
		if next == s {
			return s
		}
		s = next
	}
}

func replace(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

