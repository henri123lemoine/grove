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
	if p.Width < 40 {
		p.Width = 80
	}
	if p.Height < 10 {
		p.Height = 24
	}

	switch p.State {
	case StateCreate:
		return renderCreate(p)
	case StateCreateSelectBase:
		return renderSelectBase(p)
	case StateDelete:
		return renderDelete(p)
	case StateFilter:
		return renderFilter(p)
	case StateFetching:
		return renderFetching(p)
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
		b.WriteString("\nLoading...\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Empty state
	if len(p.Worktrees) == 0 {
		b.WriteString("\n" + PathStyle.Render("No worktrees found. Press 'n' to create one.") + "\n")
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	// Worktree list - each entry shows multiple lines of info
	for i, wt := range p.Worktrees {
		b.WriteString(renderWorktreeEntry(wt, i == p.Cursor, contentWidth))
		if i < len(p.Worktrees)-1 {
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("enter open • n new • d delete • f fetch • / filter • q quit"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderWorktreeEntry renders a single worktree with full details.
func renderWorktreeEntry(wt git.Worktree, selected bool, width int) string {
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

	// Ahead/Behind with arrows
	if wt.Ahead > 0 || wt.Behind > 0 {
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

	// Line 3: Last commit
	if wt.LastCommitHash != "" {
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
		for i, branch := range p.Branches {
			cursor := "  "
			name := branch.Name
			if i == p.BaseBranchIndex {
				cursor = SelectedStyle.Render("› ")
				name = SelectedStyle.Render(name)
			} else {
				name = NormalStyle.Render(name)
			}
			b.WriteString(cursor + name + "\n")
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
		b.WriteString("Checking safety...\n")
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

// renderFilter renders the filter mode.
func renderFilter(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("FILTER") + "  ")
	b.WriteString(p.FilterInput + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")

	for i, wt := range p.Worktrees {
		b.WriteString(renderWorktreeEntry(wt, i == p.Cursor, contentWidth))
		if i < len(p.Worktrees)-1 {
			b.WriteString("\n")
		}
	}

	if len(p.Worktrees) == 0 {
		b.WriteString("\n" + PathStyle.Render("No matches found.") + "\n")
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
	b.WriteString("Fetching updates from all remotes...\n")

	return wrapInBox(b.String(), p.Width, p.Height)
}

// wrapInBox wraps content in a box.
func wrapInBox(content string, width, height int) string {
	boxWidth := width - 2
	if boxWidth < 40 {
		boxWidth = 78
	}

	// Don't force height - let content determine size
	style := BoxStyle.Width(boxWidth)

	return style.Render(content)
}

// padRight pads a string to the right.
func padRight(s string, width int) string {
	visibleLen := lipgloss.Width(s)
	if visibleLen >= width {
		if len(s) > width-1 {
			return s[:width-1] + "…"
		}
		return s
	}
	return s + strings.Repeat(" ", width-visibleLen)
}
