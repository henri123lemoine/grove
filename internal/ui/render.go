package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/henrilemoine/grove/internal/config"
	"github.com/henrilemoine/grove/internal/git"
)

// State constants (matching app.State)
const (
	StateList = iota
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

// HelpBinding represents a keybinding for help display.
type HelpBinding struct {
	Keys string
	Desc string
}

// HelpSection represents a section of help bindings.
type HelpSection struct {
	Title    string
	Bindings []HelpBinding
}

// RenderParams contains all parameters needed for rendering.
type RenderParams struct {
	State               int
	Worktrees           []git.Worktree
	Cursor              int
	ViewOffset          int
	VisibleCount        int
	Width               int
	Height              int
	Loading             bool
	Err                 error
	Repo                *git.Repo
	Config              *config.Config
	FilterInput         string
	FilterValue         string
	CreateInput         string
	DeleteWorktree      *git.Worktree
	SafetyInfo          *git.SafetyInfo
	DeleteInput         string
	Branches            []git.Branch
	BaseBranchIndex     int
	BaseViewOffset      int
	VisibleBranchCount  int
	CreateBranch        string
	ShowDetail          bool
	RenameWorktree      *git.Worktree
	RenameInput         string
	StashWorktree       *git.Worktree
	StashEntries        []git.StashEntry
	StashCursor         int
	LayoutWorktree      *git.Worktree
	LayoutCursor        int
	SpinnerFrame        string
	HelpSections        []HelpSection
	PendingWindowsCount int
	PendingWindowsName  string // "window" for tmux, "tab" for zellij
}

// MinWidth is the absolute minimum terminal width we try to support.
const MinWidth = 30

// MinHeight is the absolute minimum terminal height we try to support.
const MinHeight = 8

// Render renders the full UI.
func Render(p RenderParams) string {
	// Graceful degradation for small terminals instead of jumping to arbitrary values.
	// Use actual width but clamp to minimum to prevent rendering issues.
	if p.Width < MinWidth {
		p.Width = MinWidth
	}
	if p.Height < MinHeight {
		p.Height = MinHeight
	}

	switch p.State {
	case StateCreate:
		return renderCreate(p)
	case StateCreateSelectBase:
		return renderSelectBase(p)
	case StateDelete:
		return renderDelete(p)
	case StateDeleteConfirmCloseWindow:
		return renderDeleteConfirmCloseWindow(p)
	case StateFilter:
		return renderFilter(p)
	case StateFetching:
		return renderFetching(p)
	case StateHelp:
		return renderHelp(p)
	case StateRename:
		return renderRename(p)
	case StateStash:
		return renderStash(p)
	case StateSelectLayout:
		return renderSelectLayout(p)
	default:
		return renderList(p)
	}
}

// renderList renders the main worktree list.
func renderList(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4 // Account for box borders and padding

	// Header
	repoName := ""
	if p.Repo != nil {
		repoName = filepath.Base(p.Repo.MainWorktreeRoot)
	}
	header := HeaderStyle.Render("WORKTREES") + "  " + PathStyle.Render(repoName)
	b.WriteString(header + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")

	// Error message if any
	if p.Err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+p.Err.Error()) + "\n\n")
	}

	// Loading state
	if p.Loading {
		b.WriteString("\n" + p.SpinnerFrame + " Loading worktrees...\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Empty state
	if len(p.Worktrees) == 0 {
		b.WriteString("\n" + PathStyle.Render("No worktrees found. Press 'n' to create one.") + "\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Calculate visible range
	startIdx := p.ViewOffset
	endIdx := p.ViewOffset + p.VisibleCount
	if endIdx > len(p.Worktrees) {
		endIdx = len(p.Worktrees)
	}
	if startIdx >= len(p.Worktrees) {
		startIdx = 0
	}

	// Show scroll indicator if items above
	if startIdx > 0 {
		b.WriteString(PathStyle.Render(fmt.Sprintf("  ↑ %d more above", startIdx)) + "\n")
	}

	// Worktree list - only render visible items
	for i := startIdx; i < endIdx; i++ {
		wt := p.Worktrees[i]
		isSelected := i == p.Cursor
		b.WriteString(renderWorktreeEntry(wt, isSelected, contentWidth, p.Config))
		// Show detail panel for selected item if enabled
		if isSelected && p.ShowDetail {
			b.WriteString(renderDetailPanel(wt, contentWidth))
		}
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Show scroll indicator if items below
	if endIdx < len(p.Worktrees) {
		b.WriteString("\n" + PathStyle.Render(fmt.Sprintf("  ↓ %d more below", len(p.Worktrees)-endIdx)))
	}

	// Footer
	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	helpText := compactHelp(
		"enter open • n new • d delete • r rename • f fetch • / filter • tab detail • ? help • q quit",
		"enter•n•d•r•f•/•tab•?•q",
		p.Width,
	)
	b.WriteString(HelpStyle.Render(helpText))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderWorktreeEntry renders a single worktree with full details.
func renderWorktreeEntry(wt git.Worktree, selected bool, width int, cfg *config.Config) string {
	var lines []string

	// Line 1: Cursor + Branch name
	cursor := "  "
	if selected {
		cursor = SelectedStyle.Render("› ")
	} else if wt.IsCurrent {
		cursor = CurrentStyle.Render("• ")
	}

	branch := wt.Branch
	if branch == "" {
		branch = "(detached)"
	}
	if selected {
		branch = SelectedStyle.Render(branch)
	} else {
		branch = BranchStyle.Render(branch)
	}
	lines = append(lines, cursor+branch)

	// Line 2: Path + Status
	indent := "    "
	path := PathStyle.Render(wt.ShortPath())

	// Build status string
	var statusParts []string

	// Dirty indicator
	if wt.IsDirty {
		statusParts = append(statusParts, DirtyStyle.Render(fmt.Sprintf("✗ %d modified", wt.DirtyFiles)))
	} else {
		statusParts = append(statusParts, CleanStyle.Render("✓ clean"))
	}

	// Ahead/Behind with arrows (respects config)
	showUpstream := true
	if cfg != nil {
		showUpstream = cfg.UI.ShowUpstream
	}
	if showUpstream && (wt.Ahead > 0 || wt.Behind > 0) {
		abStr := ""
		if wt.Behind > 0 {
			abStr += fmt.Sprintf("↓%d", wt.Behind)
		}
		if wt.Ahead > 0 {
			if abStr != "" {
				abStr += " "
			}
			abStr += fmt.Sprintf("↑%d", wt.Ahead)
		}
		statusParts = append(statusParts, AheadStyle.Render(abStr))
	}

	// Merged status
	if wt.IsMerged && !wt.IsMain {
		statusParts = append(statusParts, MergedStyle.Render("merged"))
	}

	// Unique/unpushed commits
	if wt.UniqueCommits > 0 {
		statusParts = append(statusParts, UniqueStyle.Render(fmt.Sprintf("%d unpushed", wt.UniqueCommits)))
	}

	status := strings.Join(statusParts, "  ")
	lines = append(lines, indent+path+"  "+status)

	// Line 3: Last commit (respects config)
	showCommits := true
	if cfg != nil {
		showCommits = cfg.UI.ShowCommits
	}
	if showCommits && wt.LastCommitHash != "" {
		commitLine := indent + PathStyle.Render(wt.LastCommitHash)
		msg := wt.LastCommitMessage
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		commitLine += " " + CommitStyle.Render(msg)
		if wt.LastCommitTime != "" {
			commitLine += " " + PathStyle.Render("("+wt.LastCommitTime+")")
		}
		lines = append(lines, commitLine)
	}

	return strings.Join(lines, "\n")
}

// renderDetailPanel renders the expanded detail panel for a worktree.
func renderDetailPanel(wt git.Worktree, width int) string {
	var b strings.Builder
	indent := "      "

	b.WriteString("\n")
	b.WriteString(indent + DividerStyle.Render("┌"+strings.Repeat("─", 50)+"┐") + "\n")

	// Full path
	b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Path:     ") + wt.Path)
	b.WriteString(strings.Repeat(" ", max(0, 49-len(wt.Path)-10)) + DividerStyle.Render("│") + "\n")

	// Branch
	branchLine := fmt.Sprintf("Branch:   %s", wt.Branch)
	b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Branch:   ") + wt.Branch)
	b.WriteString(strings.Repeat(" ", max(0, 49-len(branchLine))) + DividerStyle.Render("│") + "\n")

	// Status
	statusStr := "clean"
	if wt.IsDirty {
		statusStr = fmt.Sprintf("%d uncommitted files", wt.DirtyFiles)
	}
	b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Status:   ") + statusStr)
	b.WriteString(strings.Repeat(" ", max(0, 49-len(statusStr)-10)) + DividerStyle.Render("│") + "\n")

	// Upstream
	upstreamStr := "no upstream"
	if wt.HasUpstream {
		if wt.Ahead > 0 || wt.Behind > 0 {
			upstreamStr = fmt.Sprintf("↑%d ahead, ↓%d behind", wt.Ahead, wt.Behind)
		} else {
			upstreamStr = "up to date"
		}
	}
	b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Upstream: ") + upstreamStr)
	b.WriteString(strings.Repeat(" ", max(0, 49-len(upstreamStr)-10)) + DividerStyle.Render("│") + "\n")

	// Merged status
	mergedStr := "no"
	if wt.IsMerged {
		mergedStr = "yes"
	}
	if wt.IsMain {
		mergedStr = "main worktree"
	}
	b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Merged:   ") + mergedStr)
	b.WriteString(strings.Repeat(" ", max(0, 49-len(mergedStr)-10)) + DividerStyle.Render("│") + "\n")

	// Unique commits
	if wt.UniqueCommits > 0 {
		uniqueStr := fmt.Sprintf("%d commits only on this branch", wt.UniqueCommits)
		b.WriteString(indent + DividerStyle.Render("│") + " " + DangerStyle.Render("Unique:   ") + DangerStyle.Render(uniqueStr))
		b.WriteString(strings.Repeat(" ", max(0, 49-len(uniqueStr)-10)) + DividerStyle.Render("│") + "\n")
	}

	b.WriteString(indent + DividerStyle.Render("└"+strings.Repeat("─", 50)+"┘"))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderCreate renders the create worktree flow.
func renderCreate(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("NEW WORKTREE") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	b.WriteString("Branch name:\n")
	b.WriteString(p.CreateInput + "\n")

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderSelectBase renders the base branch selection.
func renderSelectBase(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("SELECT BASE BRANCH") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	b.WriteString("New branch: " + SelectedStyle.Render(p.CreateBranch) + "\n\n")

	if len(p.Branches) == 0 {
		b.WriteString(PathStyle.Render("No branches found. Press Enter to use HEAD.\n"))
	} else {
		showBranchTypes := true
		if p.Config != nil {
			showBranchTypes = p.Config.UI.ShowBranchTypes
		}

		// Calculate visible range
		startIdx := p.BaseViewOffset
		endIdx := p.BaseViewOffset + p.VisibleBranchCount
		if endIdx > len(p.Branches) {
			endIdx = len(p.Branches)
		}
		if startIdx >= len(p.Branches) {
			startIdx = 0
		}

		// Show scroll indicator if items above
		if startIdx > 0 {
			b.WriteString(PathStyle.Render(fmt.Sprintf("  ↑ %d more above", startIdx)) + "\n")
		}

		for i := startIdx; i < endIdx; i++ {
			branch := p.Branches[i]
			cursor := "  "
			name := branch.Name
			if i == p.BaseBranchIndex {
				cursor = SelectedStyle.Render("› ")
				name = SelectedStyle.Render(name)
			} else {
				name = NormalStyle.Render(name)
			}

			// Add type indicator
			typeIndicator := ""
			if showBranchTypes {
				if branch.IsWorktree {
					typeIndicator = WorktreeTagStyle.Render(" [worktree]")
				} else if branch.IsRemote {
					typeIndicator = RemoteTagStyle.Render(" [remote]")
				} else {
					typeIndicator = LocalTagStyle.Render(" [local]")
				}
			}

			// Add current indicator
			currentIndicator := ""
			if branch.IsCurrent {
				currentIndicator = CurrentStyle.Render(" (current)")
			}

			b.WriteString(cursor + name + typeIndicator + currentIndicator + "\n")
		}

		// Show scroll indicator if items below
		if endIdx < len(p.Branches) {
			b.WriteString(PathStyle.Render(fmt.Sprintf("  ↓ %d more below", len(p.Branches)-endIdx)) + "\n")
		}
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("↑/↓ select • enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderDelete renders the delete confirmation.
func renderDelete(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	if p.DeleteWorktree == nil {
		return ""
	}

	wt := p.DeleteWorktree

	b.WriteString(HeaderStyle.Render("DELETE WORKTREE") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	b.WriteString("Branch: " + SelectedStyle.Render(wt.Branch) + "\n")
	b.WriteString("Path:   " + PathStyle.Render(wt.ShortPath()) + "\n\n")

	if p.SafetyInfo == nil {
		b.WriteString(p.SpinnerFrame + " Checking safety...\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	info := p.SafetyInfo

	switch info.Level {
	case git.SafetyLevelSafe:
		b.WriteString(MergedStyle.Render("✓ Safe to delete") + "\n\n")
		b.WriteString("• Clean working directory\n")
		if info.IsMerged {
			b.WriteString("• Branch merged to default\n")
		}
		b.WriteString("\n" + HelpStyle.Render("y confirm • n cancel"))

	case git.SafetyLevelWarning:
		b.WriteString(DirtyStyle.Render("⚠ Warning") + "\n\n")
		if info.HasUncommittedChanges {
			b.WriteString(fmt.Sprintf("• %d uncommitted changes\n", info.UncommittedFileCount))
		}
		if info.HasUnpushedCommits {
			b.WriteString(fmt.Sprintf("• %d unpushed commits\n", info.UnpushedCommitCount))
		}
		if !info.IsMerged {
			b.WriteString("• Branch not merged\n")
		}
		b.WriteString("\n" + HelpStyle.Render("y confirm • n cancel"))

	case git.SafetyLevelDanger:
		b.WriteString(DangerStyle.Render("⚠ DANGER: Data will be lost!") + "\n\n")
		b.WriteString(fmt.Sprintf("%d commits exist only on this branch:\n\n", info.UniqueCommitCount))
		for i, commit := range info.UniqueCommits {
			if i >= 5 {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", len(info.UniqueCommits)-5))
				break
			}
			msg := commit.Message
			if len(msg) > 50 {
				msg = msg[:47] + "..."
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", PathStyle.Render(commit.Hash), msg))
		}
		b.WriteString("\nType 'delete' to confirm:\n")
		b.WriteString(p.DeleteInput + "\n")
		b.WriteString("\n" + HelpStyle.Render("esc cancel"))
	}

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderDeleteConfirmCloseWindow renders the close window confirmation after delete.
func renderDeleteConfirmCloseWindow(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	// Use the correct term (window/tab) based on multiplexer
	windowName := p.PendingWindowsName
	if windowName == "" {
		windowName = "window"
	}
	windowNamePlural := windowName + "s"
	if p.PendingWindowsCount == 1 {
		windowNamePlural = windowName
	}

	headerText := fmt.Sprintf("CLOSE %s?", strings.ToUpper(windowName))
	b.WriteString(HeaderStyle.Render(headerText) + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	b.WriteString(fmt.Sprintf("Found %d %s with this worktree path.\n\n", p.PendingWindowsCount, windowNamePlural))
	b.WriteString("Would you like to close " + SelectedStyle.Render(fmt.Sprintf("%s", windowNamePlural)) + "?\n\n")
	b.WriteString(HelpStyle.Render("y close • n keep • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderFilter renders the filter mode.
func renderFilter(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("FILTER") + "  ")
	b.WriteString(p.FilterInput + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")

	if len(p.Worktrees) == 0 {
		b.WriteString("\n" + PathStyle.Render("No matches found.") + "\n")
	} else {
		// Calculate visible range
		startIdx := p.ViewOffset
		endIdx := p.ViewOffset + p.VisibleCount
		if endIdx > len(p.Worktrees) {
			endIdx = len(p.Worktrees)
		}
		if startIdx >= len(p.Worktrees) {
			startIdx = 0
		}

		// Show scroll indicator if items above
		if startIdx > 0 {
			b.WriteString(PathStyle.Render(fmt.Sprintf("  ↑ %d more above", startIdx)) + "\n")
		}

		for i := startIdx; i < endIdx; i++ {
			wt := p.Worktrees[i]
			b.WriteString(renderWorktreeEntry(wt, i == p.Cursor, contentWidth, p.Config))
			if i < endIdx-1 {
				b.WriteString("\n")
			}
		}

		// Show scroll indicator if items below
		if endIdx < len(p.Worktrees) {
			b.WriteString("\n" + PathStyle.Render(fmt.Sprintf("  ↓ %d more below", len(p.Worktrees)-endIdx)))
		}
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("enter select • esc clear"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderFetching renders the fetching state.
func renderFetching(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("FETCHING") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")
	b.WriteString(p.SpinnerFrame + " Fetching updates from all remotes...\n")

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderHelp renders the help screen.
func renderHelp(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("HELP") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	// Render each help section from the passed bindings
	for i, section := range p.HelpSections {
		b.WriteString(BranchStyle.Render(section.Title) + "\n")
		b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)) + "\n")
		for _, binding := range section.Bindings {
			// Pad keys to 10 chars for alignment
			keys := binding.Keys
			if len(keys) < 10 {
				keys = keys + strings.Repeat(" ", 10-len(keys))
			}
			b.WriteString(PathStyle.Render("  "+keys) + " " + binding.Desc + "\n")
		}
		if i < len(p.HelpSections)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("Press any key to close"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderRename renders the rename branch flow.
func renderRename(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("RENAME BRANCH") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	if p.RenameWorktree == nil {
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	b.WriteString("Current: " + PathStyle.Render(p.RenameWorktree.Branch) + "\n\n")
	b.WriteString("New name:\n")
	b.WriteString(p.RenameInput + "\n")

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderStash renders the stash management view.
func renderStash(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("STASH MANAGEMENT") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	if p.StashWorktree == nil {
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	b.WriteString("Worktree: " + PathStyle.Render(p.StashWorktree.Branch) + "\n\n")

	if len(p.StashEntries) == 0 {
		b.WriteString(PathStyle.Render("No stashes found.\n"))
	} else {
		for i, entry := range p.StashEntries {
			cursor := "  "
			if i == p.StashCursor {
				cursor = SelectedStyle.Render("› ")
			}
			stashRef := fmt.Sprintf("stash@{%d}", entry.Index)
			msg := entry.Message
			if len(msg) > 50 {
				msg = msg[:47] + "..."
			}
			if i == p.StashCursor {
				b.WriteString(cursor + SelectedStyle.Render(stashRef) + " " + msg + "\n")
			} else {
				b.WriteString(cursor + StashStyle.Render(stashRef) + " " + PathStyle.Render(msg) + "\n")
			}
		}
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("p pop • a apply • d drop • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderSelectLayout renders the layout selection view.
func renderSelectLayout(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("SELECT LAYOUT") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	if p.LayoutWorktree == nil {
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	b.WriteString("Opening: " + SelectedStyle.Render(p.LayoutWorktree.Branch) + "\n\n")

	// List available layouts
	if p.Config == nil || len(p.Config.Layouts) == 0 {
		b.WriteString(PathStyle.Render("No layouts defined.\n"))
	} else {
		for i, layout := range p.Config.Layouts {
			cursor := "  "
			if i == p.LayoutCursor {
				cursor = SelectedStyle.Render("› ")
			}

			name := layout.Name
			desc := layout.Description
			if desc == "" {
				desc = fmt.Sprintf("%d panes", len(layout.Panes))
			}

			if i == p.LayoutCursor {
				b.WriteString(cursor + SelectedStyle.Render(name) + " " + PathStyle.Render(desc) + "\n")
			} else {
				b.WriteString(cursor + BranchStyle.Render(name) + " " + PathStyle.Render(desc) + "\n")
			}
		}

		// "None" option
		noneIdx := len(p.Config.Layouts)
		cursor := "  "
		if p.LayoutCursor == noneIdx {
			cursor = SelectedStyle.Render("› ")
			b.WriteString(cursor + SelectedStyle.Render("None") + " " + PathStyle.Render("Open without layout") + "\n")
		} else {
			b.WriteString(cursor + BranchStyle.Render("None") + " " + PathStyle.Render("Open without layout") + "\n")
		}
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("↑/↓ select • enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// wrapInBox wraps content in a box.
func wrapInBox(content string, width, height int) string {
	boxWidth := width - 2
	// Graceful degradation: use actual width, just ensure minimum for box borders
	if boxWidth < MinWidth-2 {
		boxWidth = MinWidth - 2
	}

	// Don't force height - let content determine size
	style := BoxStyle.Width(boxWidth)

	return style.Render(content)
}

// compactHelp returns a shortened help string for small terminals.
func compactHelp(full, compact string, width int) string {
	// If terminal is wide enough, use full help text
	if width >= 80 {
		return full
	}
	return compact
}
