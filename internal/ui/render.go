package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/henrilemoine/grove/internal/git"
)

// State constants (matching app.State)
const (
	StateList = iota
	StateCreate
	StateCreateSelectBase
	StateDelete
	StateFilter
	StateFetching
)

// RenderParams contains all parameters needed for rendering.
type RenderParams struct {
	State             int
	Worktrees         []git.Worktree
	Cursor            int
	Width             int
	Height            int
	Loading           bool
	Err               error
	Repo              *git.Repo
	FilterInput       string
	FilterValue       string
	CreateInput       string
	DeleteWorktree    *git.Worktree
	SafetyInfo        *git.SafetyInfo
	DeleteInput       string
	Branches          []git.Branch
	BaseBranchIndex   int
	CreateBranch      string
}

// Render renders the full UI.
func Render(p RenderParams) string {
	// Ensure minimum dimensions
	if p.Width < 40 {
		p.Width = 80
	}
	if p.Height < 10 {
		p.Height = 24
	}

	var content string

	switch p.State {
	case StateCreate:
		content = renderCreate(p)
	case StateCreateSelectBase:
		content = renderSelectBase(p)
	case StateDelete:
		content = renderDelete(p)
	case StateFilter:
		content = renderFilter(p)
	case StateFetching:
		content = renderFetching(p)
	default:
		content = renderList(p)
	}

	return content
}

// renderList renders the main worktree list.
func renderList(p RenderParams) string {
	var b strings.Builder

	// Title line
	repoName := ""
	if p.Repo != nil {
		repoName = filepath.Base(p.Repo.MainWorktreeRoot)
	}
	title := TitleStyle.Render("grove")
	if repoName != "" {
		title += " " + PathStyle.Render(repoName)
	}
	b.WriteString(title + "\n\n")

	// Error message if any
	if p.Err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+p.Err.Error()) + "\n\n")
	}

	// Loading state
	if p.Loading {
		b.WriteString("Loading worktrees...\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Empty state
	if len(p.Worktrees) == 0 {
		b.WriteString(NormalStyle.Render("No worktrees found.") + "\n")
		b.WriteString(HelpStyle.Render("Press 'n' to create one.") + "\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Calculate column widths based on content
	maxBranch := 0
	maxPath := 0
	for _, wt := range p.Worktrees {
		if len(wt.Branch) > maxBranch {
			maxBranch = len(wt.Branch)
		}
		shortPath := wt.ShortPath()
		if len(shortPath) > maxPath {
			maxPath = len(shortPath)
		}
	}
	// Add some padding
	branchWidth := maxBranch + 2
	if branchWidth < 15 {
		branchWidth = 15
	}
	if branchWidth > 30 {
		branchWidth = 30
	}
	pathWidth := maxPath + 2
	if pathWidth < 10 {
		pathWidth = 10
	}
	if pathWidth > 25 {
		pathWidth = 25
	}

	// Worktree list
	for i, wt := range p.Worktrees {
		line := renderWorktreeLine(wt, i == p.Cursor, branchWidth, pathWidth)
		b.WriteString(line + "\n")
	}

	// Spacer to push help to bottom
	contentHeight := lipgloss.Height(b.String())
	// Account for box padding (2 top + 2 bottom) and border (2)
	availableHeight := p.Height - 6
	if contentHeight < availableHeight-2 {
		b.WriteString(strings.Repeat("\n", availableHeight-contentHeight-2))
	}

	// Footer
	b.WriteString("\n" + renderHelp())

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderWorktreeLine renders a single worktree line.
func renderWorktreeLine(wt git.Worktree, selected bool, branchWidth, pathWidth int) string {
	// Cursor indicator
	cursor := "  "
	if selected {
		cursor = SelectedStyle.Render(SymbolCursor + " ")
	} else if wt.IsCurrent {
		cursor = CurrentStyle.Render(SymbolCurrent + " ")
	}

	// Branch name
	branch := wt.Branch
	if branch == "" {
		branch = "(detached)"
	}
	if selected {
		branch = SelectedStyle.Render(padRight(branch, branchWidth))
	} else {
		branch = NormalStyle.Render(padRight(branch, branchWidth))
	}

	// Path (shortened)
	path := padRight(wt.ShortPath(), pathWidth)
	path = PathStyle.Render(path)

	// Status indicators - more compact
	var status []string

	if wt.IsDirty {
		status = append(status, DirtyStyle.Render(fmt.Sprintf("%d modified", wt.DirtyFiles)))
	}

	if wt.Ahead > 0 {
		status = append(status, AheadStyle.Render(fmt.Sprintf("%s%d", SymbolAhead, wt.Ahead)))
	}
	if wt.Behind > 0 {
		status = append(status, AheadStyle.Render(fmt.Sprintf("%s%d", SymbolBehind, wt.Behind)))
	}

	if wt.UniqueCommits > 0 {
		status = append(status, UniqueStyle.Render(fmt.Sprintf("%d unpushed", wt.UniqueCommits)))
	} else if wt.IsMerged && !wt.IsMain {
		status = append(status, MergedStyle.Render("merged"))
	}

	statusStr := ""
	if len(status) > 0 {
		statusStr = strings.Join(status, " ")
	}

	return cursor + branch + path + statusStr
}

// renderCreate renders the create worktree flow.
func renderCreate(p RenderParams) string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("New Worktree") + "\n\n")
	b.WriteString("Branch name:\n")
	b.WriteString(p.CreateInput + "\n")

	// Spacer
	contentHeight := lipgloss.Height(b.String())
	availableHeight := p.Height - 6
	if contentHeight < availableHeight-2 {
		b.WriteString(strings.Repeat("\n", availableHeight-contentHeight-2))
	}

	b.WriteString("\n" + HelpStyle.Render("enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderSelectBase renders the base branch selection.
func renderSelectBase(p RenderParams) string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Select Base Branch") + "\n\n")
	b.WriteString("New branch: " + SelectedStyle.Render(p.CreateBranch) + "\n\n")

	if len(p.Branches) == 0 {
		b.WriteString(NormalStyle.Render("No branches found. Press Enter to use HEAD.\n"))
	} else {
		for i, branch := range p.Branches {
			cursor := "  "
			name := branch.Name
			if i == p.BaseBranchIndex {
				cursor = SelectedStyle.Render(SymbolCursor + " ")
				name = SelectedStyle.Render(name)
			} else {
				name = NormalStyle.Render(name)
			}
			b.WriteString(cursor + name + "\n")
		}
	}

	// Spacer
	contentHeight := lipgloss.Height(b.String())
	availableHeight := p.Height - 6
	if contentHeight < availableHeight-2 {
		b.WriteString(strings.Repeat("\n", availableHeight-contentHeight-2))
	}

	b.WriteString("\n" + HelpStyle.Render("↑/↓ select • enter confirm • esc cancel"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderDelete renders the delete confirmation.
func renderDelete(p RenderParams) string {
	var b strings.Builder

	if p.DeleteWorktree == nil {
		return ""
	}

	wt := p.DeleteWorktree

	b.WriteString(TitleStyle.Render("Delete Worktree") + "\n\n")
	b.WriteString("Branch: " + SelectedStyle.Render(wt.Branch) + "\n")
	b.WriteString("Path: " + PathStyle.Render(wt.ShortPath()) + "\n\n")

	if p.SafetyInfo == nil {
		b.WriteString("Checking safety...\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	info := p.SafetyInfo

	switch info.Level {
	case git.SafetyLevelSafe:
		b.WriteString(MergedStyle.Render("Safe to delete") + "\n\n")
		b.WriteString("• Clean working directory\n")
		if info.IsMerged {
			b.WriteString("• Branch merged to default\n")
		}
		b.WriteString("\n" + HelpStyle.Render("y confirm • n cancel"))

	case git.SafetyLevelWarning:
		b.WriteString(DirtyStyle.Render("Warning") + "\n\n")
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
		b.WriteString(DangerStyle.Render("DANGER: Data will be lost!") + "\n\n")
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

// renderFilter renders the filter mode.
func renderFilter(p RenderParams) string {
	var b strings.Builder

	b.WriteString("Filter: " + p.FilterInput + "\n\n")

	// Calculate column widths
	maxBranch := 15
	maxPath := 10
	for _, wt := range p.Worktrees {
		if len(wt.Branch) > maxBranch {
			maxBranch = len(wt.Branch)
		}
		if len(wt.ShortPath()) > maxPath {
			maxPath = len(wt.ShortPath())
		}
	}
	if maxBranch > 30 {
		maxBranch = 30
	}
	if maxPath > 25 {
		maxPath = 25
	}

	for i, wt := range p.Worktrees {
		line := renderWorktreeLine(wt, i == p.Cursor, maxBranch+2, maxPath+2)
		b.WriteString(line + "\n")
	}

	if len(p.Worktrees) == 0 {
		b.WriteString(HelpStyle.Render("No matches found.\n"))
	}

	// Spacer
	contentHeight := lipgloss.Height(b.String())
	availableHeight := p.Height - 6
	if contentHeight < availableHeight-2 {
		b.WriteString(strings.Repeat("\n", availableHeight-contentHeight-2))
	}

	b.WriteString("\n" + HelpStyle.Render("enter select • esc clear"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderFetching renders the fetching state.
func renderFetching(p RenderParams) string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Fetching") + "\n\n")
	b.WriteString("Fetching updates from all remotes...\n")

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderHelp renders the help footer.
func renderHelp() string {
	return HelpStyle.Render("enter open • n new • d delete • f fetch • / filter • q quit")
}

// wrapInBox wraps content in a box that fills the terminal.
func wrapInBox(content string, width, height int) string {
	boxWidth := width - 2
	if boxWidth < 40 {
		boxWidth = 78
	}

	boxHeight := height - 2
	if boxHeight < 8 {
		boxHeight = 22
	}

	style := BoxStyle.
		Width(boxWidth).
		Height(boxHeight)

	return style.Render(content)
}

// padRight pads a string to the right.
func padRight(s string, width int) string {
	visibleLen := lipgloss.Width(s)
	if visibleLen >= width {
		// Truncate if too long
		if len(s) > width-1 {
			return s[:width-1] + "…"
		}
		return s
	}
	return s + strings.Repeat(" ", width-visibleLen)
}
