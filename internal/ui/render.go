package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/henrilemoine/grove/internal/config"
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
	StateHelp
	StatePR
	StateRename
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
	Config            *config.Config
	FilterInput       string
	FilterValue       string
	CreateInput       string
	DeleteWorktree    *git.Worktree
	SafetyInfo        *git.SafetyInfo
	DeleteInput       string
	Branches          []git.Branch
	BaseBranchIndex   int
	CreateBranch      string
	ShowDetail        bool
	PRWorktree        *git.Worktree
	PRState           string
	RenameWorktree    *git.Worktree
	RenameInput       string
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
	case StateHelp:
		return renderHelp(p)
	case StatePR:
		return renderPR(p)
	case StateRename:
		return renderRename(p)
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
		isSelected := i == p.Cursor
		b.WriteString(renderWorktreeEntry(wt, isSelected, contentWidth, p.Config))
		// Show detail panel for selected item if enabled
		if isSelected && p.ShowDetail {
			b.WriteString(renderDetailPanel(wt, contentWidth))
		}
		if i < len(p.Worktrees)-1 {
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("enter open • n new • d delete • p PR • r rename • f fetch • / filter • tab detail • ? help • q quit"))

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

	// Stash indicator
	if wt.StashCount > 0 {
		statusParts = append(statusParts, StashStyle.Render(fmt.Sprintf("📦 %d stashed", wt.StashCount)))
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
	if wt.Ahead > 0 || wt.Behind > 0 {
		upstreamStr = fmt.Sprintf("↑%d ahead, ↓%d behind", wt.Ahead, wt.Behind)
	} else if wt.Ahead == 0 && wt.Behind == 0 {
		upstreamStr = "up to date"
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

	// Stash count
	if wt.StashCount > 0 {
		stashStr := fmt.Sprintf("%d stashed changes", wt.StashCount)
		b.WriteString(indent + DividerStyle.Render("│") + " " + PathStyle.Render("Stash:    ") + stashStr)
		b.WriteString(strings.Repeat(" ", max(0, 49-len(stashStr)-10)) + DividerStyle.Render("│") + "\n")
	}

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

		for i, branch := range p.Branches {
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
		b.WriteString(renderWorktreeEntry(wt, i == p.Cursor, contentWidth, p.Config))
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

// renderHelp renders the help screen.
func renderHelp(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("HELP") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	// Navigation section
	b.WriteString(BranchStyle.Render("Navigation") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)) + "\n")
	b.WriteString(PathStyle.Render("  ↑/k      ") + "Move up\n")
	b.WriteString(PathStyle.Render("  ↓/j      ") + "Move down\n")
	b.WriteString(PathStyle.Render("  g/Home   ") + "Go to first\n")
	b.WriteString(PathStyle.Render("  G/End    ") + "Go to last\n")
	b.WriteString(PathStyle.Render("  enter    ") + "Open worktree\n")
	b.WriteString("\n")

	// Actions section
	b.WriteString(BranchStyle.Render("Actions") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)) + "\n")
	b.WriteString(PathStyle.Render("  n        ") + "New worktree\n")
	b.WriteString(PathStyle.Render("  d        ") + "Delete worktree\n")
	b.WriteString(PathStyle.Render("  p        ") + "Create PR\n")
	b.WriteString(PathStyle.Render("  r        ") + "Rename branch\n")
	b.WriteString(PathStyle.Render("  f        ") + "Fetch all remotes\n")
	b.WriteString(PathStyle.Render("  /        ") + "Filter worktrees\n")
	b.WriteString(PathStyle.Render("  tab      ") + "Toggle detail panel\n")
	b.WriteString("\n")

	// General section
	b.WriteString(BranchStyle.Render("General") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", 40)) + "\n")
	b.WriteString(PathStyle.Render("  ?        ") + "Toggle this help\n")
	b.WriteString(PathStyle.Render("  q        ") + "Quit\n")
	b.WriteString(PathStyle.Render("  esc      ") + "Cancel / Close\n")

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("Press any key to close"))

	return wrapInBox(b.String(), p.Width, p.Height)
}

// renderPR renders the PR creation flow.
func renderPR(p RenderParams) string {
	var b strings.Builder
	contentWidth := p.Width - 4

	b.WriteString(HeaderStyle.Render("CREATE PULL REQUEST") + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	if p.PRWorktree == nil {
		return wrapInBox(b.String(), p.Width, p.Height)
	}

	b.WriteString("Branch: " + SelectedStyle.Render(p.PRWorktree.Branch) + "\n\n")

	switch p.PRState {
	case "checking":
		b.WriteString("Checking gh CLI authentication...\n")
	case "pushing":
		b.WriteString("Pushing branch to remote...\n")
	case "creating":
		b.WriteString("Creating pull request...\n")
	default:
		b.WriteString("Preparing...\n")
	}

	b.WriteString("\n" + DividerStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	b.WriteString(HelpStyle.Render("esc cancel"))

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
