package app

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	StateDeleteConfirmCloseWindow
	StateFilter
	StateFetching
	StateHelp
	StateRename
	StateStash
	StateSelectLayout
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
	viewOffset        int

	// State
	state   State
	loading bool
	err     error

	// Create flow
	createInput     textinput.Model
	createBranch    string
	createIsNew     bool
	baseBranchIndex int
	baseViewOffset  int

	// Delete flow
	deleteWorktree      *git.Worktree
	safetyInfo          *git.SafetyInfo
	deleteInput         textinput.Model
	pendingWindowsClose []string // Window/tab IDs to potentially close after delete

	// Filter
	filterInput textinput.Model

	// Rename flow
	renameWorktree *git.Worktree
	renameInput    textinput.Model

	// Stash flow
	stashWorktree *git.Worktree
	stashEntries  []git.StashEntry
	stashCursor   int

	// Layout selection flow
	layoutWorktree *git.Worktree
	layoutCursor   int

	// UI
	width          int
	height         int
	keys           KeyMap
	showDetail     bool
	spinner        spinner.Model
	configWarnings []string

	// Exit behavior
	shouldQuit       bool
	openAfterQuit    *git.Worktree
	selectedWorktree *git.Worktree
}

// New creates a new Model.
func New(cfg *config.Config, repo *git.Repo, configWarnings []string) Model {
	// Create text inputs
	createInput := textinput.New()
	createInput.Placeholder = "branch-name"
	createInput.CharLimit = 250 // Git supports up to 255 bytes

	deleteInput := textinput.New()
	deleteInput.Placeholder = "Type 'delete' to confirm"
	deleteInput.CharLimit = 10

	filterInput := textinput.New()
	filterInput.Placeholder = "filter..."
	filterInput.CharLimit = 100

	renameInput := textinput.New()
	renameInput.Placeholder = "new-branch-name"
	renameInput.CharLimit = 250 // Git supports up to 255 bytes

	// Initialize spinner with dots style
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		config:         cfg,
		repo:           repo,
		keys:           KeyMapFromConfig(&cfg.Keys),
		createInput:    createInput,
		deleteInput:    deleteInput,
		filterInput:    filterInput,
		renameInput:    renameInput,
		spinner:        s,
		state:          StateList,
		loading:        true,
		configWarnings: configWarnings,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadWorktrees,
		loadBranchesWithTypes,
		m.spinner.Tick,
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		return m, nil

	case tea.KeyMsg:
		// Clear config warnings on first keypress
		if len(m.configWarnings) > 0 {
			m.configWarnings = nil
			return m, nil
		}

		// Handle quit globally
		if key.Matches(msg, m.keys.Quit) && m.state == StateList {
			m.shouldQuit = true
			return m, tea.Quit
		}

		// Delegate to state-specific handler
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case spinner.TickMsg:
		// Update spinner and continue ticking if we're in a loading state
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.isLoading() {
			return m, cmd
		}
		return m, nil

	case WorktreesCachedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.worktrees = msg.Worktrees
		m.applyFilter()
		m.ensureCursorVisible()
		// If from cache, trigger background refresh + upstream fetch
		if msg.FromCache {
			return m, tea.Batch(refreshWorktrees, loadUpstreamStatus(m.worktrees))
		}
		// Fresh data - just fetch upstream
		return m, loadUpstreamStatus(m.worktrees)

	case WorktreesLoadedMsg:
		// Background refresh completed (or direct load in tests)
		m.loading = false
		if msg.Err != nil {
			// Non-fatal if we already have cached data
			if len(m.worktrees) > 0 {
				return m, nil
			}
			m.err = msg.Err
			return m, nil
		}
		m.worktrees = msg.Worktrees
		m.applyFilter()
		m.ensureCursorVisible()
		// Trigger upstream fetch for fresh data
		return m, loadUpstreamStatus(m.worktrees)

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

		// Check if we can skip confirmation based on config
		skipConfirmation := false
		if msg.Info.Level == git.SafetyLevelSafe {
			// Safe level - check if any confirmation is needed at all
			// Only dirty worktrees need ConfirmDirty, only unmerged need ConfirmUnmerged
			// SafetyLevelSafe means clean and merged, so no confirmation needed
			skipConfirmation = true
		} else if msg.Info.Level == git.SafetyLevelWarning {
			// Warning level - check config flags
			needsDirtyConfirm := msg.Info.HasUncommittedChanges && m.config.Safety.ConfirmDirty
			needsUnmergedConfirm := !msg.Info.IsMerged && m.config.Safety.ConfirmUnmerged
			needsUnpushedConfirm := msg.Info.HasUnpushedCommits // Always warn about unpushed
			skipConfirmation = !needsDirtyConfirm && !needsUnmergedConfirm && !needsUnpushedConfirm
		}

		if skipConfirmation {
			// Proceed with deletion immediately
			force := msg.Info.HasUncommittedChanges
			path := m.deleteWorktree.Path
			m.state = StateList
			m.deleteWorktree = nil
			m.safetyInfo = nil
			return m, deleteWorktree(path, force)
		}

		// Focus delete input only if danger level AND config requires typing
		if msg.Info.Level == git.SafetyLevelDanger && m.config.Safety.RequireTypingForUnique {
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
		// Run post-create operations
		if msg.Err == nil && msg.Path != "" {
			return m, tea.Batch(
				loadWorktrees,
				runPostCreateOperations(m.config, msg.Path, msg.Branch),
			)
		}
		return m, loadWorktrees

	case WorktreeDeletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.state = StateList
			m.deleteInput.Reset()
			m.deleteWorktree = nil
			m.safetyInfo = nil
			return m, loadWorktrees
		}

		// Check for multiplexer windows/tabs to close
		if exec.InMultiplexer() && m.config.Delete.CloseWindowAction != "never" {
			windows := exec.FindWindowsForPath(msg.Path)
			if len(windows) > 0 {
				switch m.config.Delete.CloseWindowAction {
				case "auto":
					// Close windows/tabs immediately
					for _, w := range windows {
						_ = exec.CloseWindow(w)
					}
				case "ask":
					// Store windows and ask user
					m.pendingWindowsClose = windows
					m.state = StateDeleteConfirmCloseWindow
					m.deleteInput.Reset()
					m.deleteWorktree = nil
					m.safetyInfo = nil
					return m, loadWorktrees
				}
			}
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

	case BranchRenamedMsg:
		m.state = StateList
		m.renameInput.Reset()
		m.renameWorktree = nil
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		return m, loadWorktrees

	case FileCopyCompletedMsg:
		if msg.Err != nil {
			// Show error to user with clear context
			m.err = fmt.Errorf("file copy failed: %w", msg.Err)
		}
		return m, nil

	case PostCreateHooksCompletedMsg:
		if msg.Err != nil {
			// Show error to user with clear context
			m.err = fmt.Errorf("post-create hook failed: %w", msg.Err)
		}
		return m, nil

	case PruneCompletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		// Refresh worktrees after pruning
		return m, loadWorktrees

	case StashListLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.state = StateList
			m.stashWorktree = nil
			return m, nil
		}
		m.stashEntries = msg.Entries
		return m, nil

	case StashOperationCompletedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}
		// Return to list and refresh
		m.state = StateList
		m.stashWorktree = nil
		m.stashEntries = nil
		return m, loadWorktrees

	case DetailLoadedMsg:
		// Update worktree with lazy-loaded detail info
		for i := range m.worktrees {
			if m.worktrees[i].Path == msg.Path {
				m.worktrees[i].LastCommitHash = msg.LastCommitHash
				m.worktrees[i].LastCommitMessage = msg.LastCommitMessage
				m.worktrees[i].LastCommitTime = msg.LastCommitTime
				break
			}
		}
		for i := range m.filteredWorktrees {
			if m.filteredWorktrees[i].Path == msg.Path {
				m.filteredWorktrees[i].LastCommitHash = msg.LastCommitHash
				m.filteredWorktrees[i].LastCommitMessage = msg.LastCommitMessage
				m.filteredWorktrees[i].LastCommitTime = msg.LastCommitTime
				break
			}
		}
		return m, nil

	case UpstreamLoadedMsg:
		// Update worktrees with background-loaded upstream status
		for i := range m.worktrees {
			for _, updated := range msg.Worktrees {
				if m.worktrees[i].Path == updated.Path {
					m.worktrees[i].Ahead = updated.Ahead
					m.worktrees[i].Behind = updated.Behind
					m.worktrees[i].HasUpstream = updated.HasUpstream
					break
				}
			}
		}
		// Also update filtered list
		for i := range m.filteredWorktrees {
			for _, updated := range msg.Worktrees {
				if m.filteredWorktrees[i].Path == updated.Path {
					m.filteredWorktrees[i].Ahead = updated.Ahead
					m.filteredWorktrees[i].Behind = updated.Behind
					m.filteredWorktrees[i].HasUpstream = updated.HasUpstream
					break
				}
			}
		}
		return m, nil
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
	case StateDeleteConfirmCloseWindow:
		return m.handleDeleteConfirmCloseWindowKeys(msg)
	case StateFilter:
		return m.handleFilterKeys(msg)
	case StateHelp:
		return m.handleHelpKeys(msg)
	case StateRename:
		return m.handleRenameKeys(msg)
	case StateStash:
		return m.handleStashKeys(msg)
	case StateSelectLayout:
		return m.handleLayoutKeys(msg)
	}
	return m, nil
}

// handleMouse handles mouse events.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.state != StateList {
		return m, nil
	}

	// Handle left mouse button press
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		// Calculate which worktree was clicked
		// Account for header (2 lines) and box padding
		headerHeight := 3

		// Account for "↑ X more above" indicator if scrolled
		if m.viewOffset > 0 {
			headerHeight++
		}

		// Row height depends on whether commits are shown
		rowHeight := 2
		if m.config != nil && m.config.UI.ShowCommits {
			rowHeight = 3
		}

		// Also account for detail panel if shown (adds ~8 lines per selected item)
		// and separator lines between entries (+1 per entry)
		clickedRow := (msg.Y - headerHeight) / (rowHeight + 1) // +1 for separator
		if clickedRow >= 0 && clickedRow < len(m.filteredWorktrees) {
			m.cursor = clickedRow
		}
	}

	return m, nil
}

// handleListKeys handles key presses in the list view.
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear any previous error when user takes action
	m.err = nil

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredWorktrees)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
	case key.Matches(msg, m.keys.Home):
		m.cursor = 0
		m.viewOffset = 0
	case key.Matches(msg, m.keys.End):
		m.cursor = len(m.filteredWorktrees) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	case key.Matches(msg, m.keys.Open):
		if len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := &m.filteredWorktrees[m.cursor]
			m.selectedWorktree = wt

			// If layouts are defined, show layout selector
			if len(m.config.Layouts) > 0 {
				m.layoutWorktree = wt
				m.layoutCursor = 0
				m.state = StateSelectLayout
				return m, nil
			}

			// No layouts, open directly
			var currentWt *git.Worktree
			for i := range m.worktrees {
				if m.worktrees[i].IsCurrent {
					currentWt = &m.worktrees[i]
					break
				}
			}
			return m, openWorktree(m.config, wt, currentWt, nil)
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
	case key.Matches(msg, m.keys.Rename):
		if len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := &m.filteredWorktrees[m.cursor]
			if wt.IsMain {
				m.err = fmt.Errorf("cannot rename main worktree branch")
				return m, nil
			}
			if wt.IsDetached {
				m.err = fmt.Errorf("cannot rename detached HEAD (checkout a branch first)")
				return m, nil
			}
			m.renameWorktree = wt
			m.renameInput.SetValue(wt.Branch)
			m.renameInput.Focus()
			m.state = StateRename
			return m, textinput.Blink
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
		// Lazy-load detail info when toggling on
		if m.showDetail && len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := m.filteredWorktrees[m.cursor]
			if wt.LastCommitHash == "" {
				return m, loadWorktreeDetail(wt.Path)
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Prune):
		return m, pruneWorktrees
	case key.Matches(msg, m.keys.Stash):
		if len(m.filteredWorktrees) > 0 && m.cursor < len(m.filteredWorktrees) {
			wt := &m.filteredWorktrees[m.cursor]
			m.stashWorktree = wt
			m.stashCursor = 0
			m.state = StateStash
			return m, loadStashList(wt.Path)
		}
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

		// Check if this branch already has a worktree
		for i := range m.worktrees {
			if m.worktrees[i].Branch == branchName {
				m.state = StateList
				m.createInput.Reset()
				// Find current worktree for stash_on_switch
				var currentWt *git.Worktree
				for j := range m.worktrees {
					if m.worktrees[j].IsCurrent {
						currentWt = &m.worktrees[j]
						break
					}
				}
				return m, openWorktree(m.config, &m.worktrees[i], currentWt, nil)
			}
		}

		// Check if branch exists
		m.createIsNew = !git.BranchExists(branchName)
		if m.createIsNew {
			m.state = StateCreateSelectBase
			// Pre-select the configured default base branch if it exists in the list
			m.baseBranchIndex = 0
			if m.config.General.DefaultBaseBranch != "" {
				for i, b := range m.branches {
					if b.Name == m.config.General.DefaultBaseBranch {
						m.baseBranchIndex = i
						break
					}
				}
			}
			m.ensureBaseBranchVisible()
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
			m.ensureBaseBranchVisible()
		}
	case key.Matches(msg, m.keys.Down):
		if m.baseBranchIndex < len(m.branches)-1 {
			m.baseBranchIndex++
			m.ensureBaseBranchVisible()
		}
	case key.Matches(msg, m.keys.Cancel):
		m.state = StateList
		m.createInput.Reset()
		m.baseViewOffset = 0
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		baseBranch := ""
		if m.baseBranchIndex < len(m.branches) {
			baseBranch = m.branches[m.baseBranchIndex].Name
		}
		m.baseViewOffset = 0
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

	// Determine if we should require typing "delete" based on config
	requireTyping := m.safetyInfo.Level == git.SafetyLevelDanger &&
		m.config.Safety.RequireTypingForUnique

	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.deleteInput.Reset()
		m.deleteWorktree = nil
		m.safetyInfo = nil
		return m, nil
	case tea.KeyEnter:
		// Check if we need to type "delete"
		if requireTyping {
			if m.deleteInput.Value() != "delete" {
				return m, nil
			}
		}
		// Proceed with deletion
		force := m.safetyInfo.HasUncommittedChanges
		return m, deleteWorktree(m.deleteWorktree.Path, force)
	}

	// If requiring typing, handle text input
	if requireTyping {
		var cmd tea.Cmd
		m.deleteInput, cmd = m.deleteInput.Update(msg)
		return m, cmd
	}

	// For safe/warning (and danger without RequireTypingForUnique), y confirms, n cancels
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

// handleDeleteConfirmCloseWindowKeys handles key presses in the close window confirmation.
func (m Model) handleDeleteConfirmCloseWindowKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel - don't close windows
		m.state = StateList
		m.pendingWindowsClose = nil
		return m, nil
	}

	if msg.String() == "y" || msg.String() == "Y" {
		// Close the windows/tabs
		for _, w := range m.pendingWindowsClose {
			_ = exec.CloseWindow(w)
		}
		m.state = StateList
		m.pendingWindowsClose = nil
		return m, nil
	}
	if msg.String() == "n" || msg.String() == "N" {
		// Don't close windows
		m.state = StateList
		m.pendingWindowsClose = nil
		return m, nil
	}

	return m, nil
}

// handleFilterKeys handles key presses in filter mode.
func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Remember the currently selected worktree before clearing filter
		var selectedPath string
		if m.cursor >= 0 && m.cursor < len(m.filteredWorktrees) {
			selectedPath = m.filteredWorktrees[m.cursor].Path
		}

		m.state = StateList
		m.filterInput.Reset()
		m.applyFilter()

		// Try to restore cursor to the same worktree
		if selectedPath != "" {
			for i, wt := range m.filteredWorktrees {
				if wt.Path == selectedPath {
					m.cursor = i
					break
				}
			}
		}
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

// handleRenameKeys handles key presses in rename flow.
func (m Model) handleRenameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.renameInput.Reset()
		m.renameWorktree = nil
		return m, nil
	case tea.KeyEnter:
		newName := m.renameInput.Value()
		if newName == "" || newName == m.renameWorktree.Branch {
			m.state = StateList
			m.renameInput.Reset()
			m.renameWorktree = nil
			return m, nil
		}
		return m, renameBranch(m.renameWorktree.Path, m.renameWorktree.Branch, newName)
	}

	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

// handleStashKeys handles key presses in stash management flow.
func (m Model) handleStashKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.stashWorktree = nil
		m.stashEntries = nil
		return m, nil
	case tea.KeyUp:
		if m.stashCursor > 0 {
			m.stashCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.stashCursor < len(m.stashEntries)-1 {
			m.stashCursor++
		}
		return m, nil
	}

	// Check for action keys
	switch msg.String() {
	case "p": // Pop stash
		if len(m.stashEntries) > 0 && m.stashCursor < len(m.stashEntries) {
			entry := m.stashEntries[m.stashCursor]
			return m, popStash(m.stashWorktree.Path, entry.Index)
		}
	case "a": // Apply stash (keep in list)
		if len(m.stashEntries) > 0 && m.stashCursor < len(m.stashEntries) {
			entry := m.stashEntries[m.stashCursor]
			return m, applyStash(m.stashWorktree.Path, entry.Index)
		}
	case "d", "x": // Drop stash
		if len(m.stashEntries) > 0 && m.stashCursor < len(m.stashEntries) {
			entry := m.stashEntries[m.stashCursor]
			return m, dropStash(m.stashWorktree.Path, entry.Index)
		}
	}

	return m, nil
}

// handleLayoutKeys handles key presses in layout selection.
func (m Model) handleLayoutKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Number of options: layouts + "None" option
	numOptions := len(m.config.Layouts) + 1

	switch msg.Type {
	case tea.KeyEsc:
		m.state = StateList
		m.layoutWorktree = nil
		return m, nil
	case tea.KeyUp:
		if m.layoutCursor > 0 {
			m.layoutCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.layoutCursor < numOptions-1 {
			m.layoutCursor++
		}
		return m, nil
	case tea.KeyEnter:
		// Find current worktree for stash_on_switch
		var currentWt *git.Worktree
		for i := range m.worktrees {
			if m.worktrees[i].IsCurrent {
				currentWt = &m.worktrees[i]
				break
			}
		}

		// Determine selected layout (nil = "None" option)
		var selectedLayout *config.LayoutConfig
		if m.layoutCursor < len(m.config.Layouts) {
			selectedLayout = &m.config.Layouts[m.layoutCursor]
		}

		wt := m.layoutWorktree
		m.state = StateList
		m.layoutWorktree = nil
		return m, openWorktree(m.config, wt, currentWt, selectedLayout)
	}

	return m, nil
}

// worktreeSource implements fuzzy.Source for worktree fuzzy matching.
type worktreeSource []git.Worktree

func (w worktreeSource) String(i int) string {
	// Match against both branch name and short path for better results
	// Using ShortPath avoids matching user home directory in absolute paths
	return w[i].Branch + " " + w[i].ShortPath()
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
		State:               int(m.state),
		Worktrees:           m.filteredWorktrees,
		Cursor:              m.cursor,
		ViewOffset:          m.viewOffset,
		VisibleCount:        m.visibleItemCount(),
		Width:               m.width,
		Height:              m.height,
		Loading:             m.loading,
		Err:                 m.err,
		Repo:                m.repo,
		Config:              m.config,
		FilterInput:         m.filterInput.View(),
		FilterValue:         m.filterInput.Value(),
		CreateInput:         m.createInput.View(),
		DeleteWorktree:      m.deleteWorktree,
		SafetyInfo:          m.safetyInfo,
		DeleteInput:         m.deleteInput.View(),
		ShowDetail:          m.showDetail,
		Branches:            m.branches,
		BaseBranchIndex:     m.baseBranchIndex,
		BaseViewOffset:      m.baseViewOffset,
		VisibleBranchCount:  m.visibleBranchCount(),
		CreateBranch:        m.createBranch,
		RenameWorktree:      m.renameWorktree,
		RenameInput:         m.renameInput.View(),
		StashWorktree:       m.stashWorktree,
		StashEntries:        m.stashEntries,
		StashCursor:         m.stashCursor,
		LayoutWorktree:      m.layoutWorktree,
		LayoutCursor:        m.layoutCursor,
		SpinnerFrame:        m.spinner.View(),
		HelpSections:        m.keys.HelpSections(),
		PendingWindowsCount: len(m.pendingWindowsClose),
		PendingWindowsName:  exec.GetMultiplexer().WindowName(),
		ConfigWarnings:      m.configWarnings,
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

// SelectedWorktree returns the selected worktree (for --print-selected).
func (m Model) SelectedWorktree() *git.Worktree {
	return m.selectedWorktree
}

// isLoading returns true if the app is in any loading state.
func (m Model) isLoading() bool {
	return m.loading ||
		m.state == StateFetching ||
		(m.state == StateDelete && m.safetyInfo == nil)
}

// Commands

func loadWorktrees() tea.Msg {
	worktrees, fromCache, err := git.ListCached()
	return WorktreesCachedMsg{Worktrees: worktrees, FromCache: fromCache, Err: err}
}

func refreshWorktrees() tea.Msg {
	worktrees, err := git.ListAndCache()
	return WorktreesLoadedMsg{Worktrees: worktrees, Err: err}
}

func loadBranchesWithTypes() tea.Msg {
	branches, err := git.ListAllBranchesWithWorktreeStatus()
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
		path := filepath.Join(cfg.General.WorktreeDir, sanitizePath(branch))
		if repo != nil {
			// Always use MainWorktreeRoot so worktrees are created at the project root
			path = filepath.Join(repo.MainWorktreeRoot, cfg.General.WorktreeDir, sanitizePath(branch))
		}
		err := git.Create(path, branch, isNew, baseBranch)
		return WorktreeCreatedMsg{Path: path, Branch: branch, Err: err}
	}
}

func deleteWorktree(path string, force bool) tea.Cmd {
	return func() tea.Msg {
		err := git.Remove(path, force)
		return WorktreeDeletedMsg{Path: path, Err: err}
	}
}

func openWorktree(cfg *config.Config, wt *git.Worktree, currentWt *git.Worktree, layout *config.LayoutConfig) tea.Cmd {
	return func() tea.Msg {
		// Handle stash_on_switch: stash current worktree if dirty
		if cfg.Open.StashOnSwitch && currentWt != nil && currentWt.IsDirty && currentWt.Path != wt.Path {
			_, err := git.CreateStash(currentWt.Path, "grove: auto-stash before switching")
			if err != nil {
				return WorktreeOpenedMsg{Err: fmt.Errorf("failed to stash changes: %w", err), IsNewWindow: false}
			}
		}

		isNew, err := exec.OpenWithConfig(cfg, wt, layout)
		return WorktreeOpenedMsg{Err: err, IsNewWindow: isNew}
	}
}

func fetchAll() tea.Msg {
	err := git.FetchAll()
	return FetchCompletedMsg{Err: err}
}

func pruneWorktrees() tea.Msg {
	count, err := git.Prune()
	return PruneCompletedMsg{PrunedCount: count, Err: err}
}

func renameBranch(worktreePath, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		err := git.RenameBranch(worktreePath, oldName, newName)
		return BranchRenamedMsg{OldName: oldName, NewName: newName, Err: err}
	}
}

func loadStashList(worktreePath string) tea.Cmd {
	return func() tea.Msg {
		entries, err := git.ListStashes(worktreePath)
		return StashListLoadedMsg{Entries: entries, Err: err}
	}
}

func loadWorktreeDetail(worktreePath string) tea.Cmd {
	return func() tea.Msg {
		hash, msg, time, _ := git.GetLastCommit(worktreePath)
		return DetailLoadedMsg{
			Path:              worktreePath,
			LastCommitHash:    hash,
			LastCommitMessage: msg,
			LastCommitTime:    time,
		}
	}
}

func loadUpstreamStatus(worktrees []git.Worktree) tea.Cmd {
	return func() tea.Msg {
		// Make a copy to avoid race conditions
		wtCopy := make([]git.Worktree, len(worktrees))
		copy(wtCopy, worktrees)
		git.EnrichWorktreesUpstream(wtCopy)
		return UpstreamLoadedMsg{Worktrees: wtCopy}
	}
}

func popStash(worktreePath string, index int) tea.Cmd {
	return func() tea.Msg {
		err := git.PopStashAt(worktreePath, index)
		return StashOperationCompletedMsg{Operation: "pop", Err: err}
	}
}

func applyStash(worktreePath string, index int) tea.Cmd {
	return func() tea.Msg {
		err := git.ApplyStash(worktreePath, index)
		return StashOperationCompletedMsg{Operation: "apply", Err: err}
	}
}

func dropStash(worktreePath string, index int) tea.Cmd {
	return func() tea.Msg {
		err := git.DropStash(worktreePath, index)
		return StashOperationCompletedMsg{Operation: "drop", Err: err}
	}
}

func runPostCreateOperations(cfg *config.Config, path, branch string) tea.Cmd {
	return func() tea.Msg {
		// Check for template match
		template := cfg.GetTemplateForBranch(branch)

		// Determine patterns to use
		copyPatterns := cfg.Worktree.CopyPatterns
		postCreateCmd := cfg.Worktree.PostCreateCmd

		if template != nil {
			if len(template.CopyPatterns) > 0 {
				copyPatterns = template.CopyPatterns
			}
			if len(template.PostCreateCmd) > 0 {
				postCreateCmd = template.PostCreateCmd
			}
		}

		// Copy files
		if len(copyPatterns) > 0 {
			repo, _ := git.GetRepo()
			if repo != nil {
				err := git.CopyFiles(repo.MainWorktreeRoot, path, copyPatterns, cfg.Worktree.CopyIgnores)
				if err != nil {
					return PostCreateHooksCompletedMsg{Err: err}
				}
			}
		}

		// Run post-create commands
		if len(postCreateCmd) > 0 {
			err := git.RunPostCreateHooks(path, postCreateCmd, cfg.Worktree.HookTimeout)
			if err != nil {
				return PostCreateHooksCompletedMsg{Err: err}
			}
		}

		return PostCreateHooksCompletedMsg{Err: nil}
	}
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

// visibleItemCount returns how many worktree items can fit in the viewport.
func (m Model) visibleItemCount() int {
	// Account for UI chrome:
	// - Header: 2 lines (title + divider)
	// - Footer: 2 lines (divider + help)
	// - Box borders: 2 lines
	// Total overhead: 6 lines
	const overhead = 6

	// Each worktree entry takes 2-3 lines (depending on ShowCommits)
	// Plus 1 line for the separator between entries
	linesPerItem := 2
	if m.config != nil && m.config.UI.ShowCommits {
		linesPerItem = 3
	}
	// Account for separator line between entries
	linesPerItem++

	availableLines := m.height - overhead
	if availableLines < linesPerItem {
		return 1
	}
	return availableLines / linesPerItem
}

// ensureCursorVisible adjusts viewOffset to keep cursor in visible area.
func (m *Model) ensureCursorVisible() {
	visible := m.visibleItemCount()
	if visible <= 0 {
		visible = 1
	}

	// If cursor is above the visible area, scroll up
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}

	// If cursor is below the visible area, scroll down
	if m.cursor >= m.viewOffset+visible {
		m.viewOffset = m.cursor - visible + 1
	}

	// Ensure viewOffset doesn't go negative
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
}

// ensureBaseBranchVisible adjusts baseViewOffset to keep baseBranchIndex in visible area.
func (m *Model) ensureBaseBranchVisible() {
	visible := m.visibleBranchCount()
	if visible <= 0 {
		visible = 1
	}

	// If cursor is above the visible area, scroll up
	if m.baseBranchIndex < m.baseViewOffset {
		m.baseViewOffset = m.baseBranchIndex
	}

	// If cursor is below the visible area, scroll down
	if m.baseBranchIndex >= m.baseViewOffset+visible {
		m.baseViewOffset = m.baseBranchIndex - visible + 1
	}

	// Ensure baseViewOffset doesn't go negative
	if m.baseViewOffset < 0 {
		m.baseViewOffset = 0
	}
}

// visibleBranchCount returns how many branch items can fit in the viewport.
func (m Model) visibleBranchCount() int {
	// Account for UI chrome:
	// - Header: 2 lines (title + divider)
	// - "New branch: X" line + blank: 2 lines
	// - Footer: 2 lines (divider + help)
	// - Box borders: 2 lines
	// Total overhead: 8 lines
	const overhead = 8

	// Each branch entry takes 1 line
	availableLines := m.height - overhead
	if availableLines < 1 {
		return 1
	}
	return availableLines
}
