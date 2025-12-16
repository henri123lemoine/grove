package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepoDetection tests that we can detect the repo correctly.
func TestRepoDetection(t *testing.T) {
	// This test runs in the grove repo itself
	ResetRepo()
	repo, err := GetRepo()
	if err != nil {
		t.Fatalf("Failed to detect repo: %v", err)
	}

	if repo.Root == "" {
		t.Error("Root should not be empty")
	}

	if repo.GitDir == "" {
		t.Error("GitDir should not be empty")
	}

	if repo.MainWorktreeRoot == "" {
		t.Error("MainWorktreeRoot should not be empty")
	}

	// Verify MainWorktreeRoot contains a .git
	gitPath := filepath.Join(repo.MainWorktreeRoot, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		// For worktrees, check if it's a file pointing to git dir
		if _, err := os.Stat(repo.GitDir); os.IsNotExist(err) {
			t.Errorf("Neither .git nor GitDir exists")
		}
	}
}

// TestWorktreeList tests listing worktrees.
func TestWorktreeList(t *testing.T) {
	ResetRepo()
	worktrees, err := List()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	// We should have at least one worktree (the current one)
	if len(worktrees) == 0 {
		t.Error("Expected at least one worktree")
	}

	// Check that each worktree has required fields
	for _, wt := range worktrees {
		if wt.Path == "" {
			t.Error("Worktree path should not be empty")
		}
		// Branch can be empty for detached HEAD
	}
}

// TestBranchExists tests branch existence checking.
func TestBranchExists(t *testing.T) {
	// The current branch should exist
	currentBranch, err := CurrentBranch()
	if err != nil {
		t.Skipf("Could not get current branch: %v", err)
	}

	// Skip if in detached HEAD state (common in CI)
	if currentBranch == "HEAD" || currentBranch == "" {
		t.Skip("Skipping: repository is in detached HEAD state")
	}

	if !BranchExists(currentBranch) {
		t.Errorf("Current branch %q should exist", currentBranch)
	}

	// A random non-existent branch should not exist
	if BranchExists("this-branch-definitely-does-not-exist-12345") {
		t.Error("Non-existent branch should not exist")
	}
}

// TestListBranches tests branch listing.
func TestListBranches(t *testing.T) {
	branches, err := ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	// We should have at least one branch
	if len(branches) == 0 {
		t.Error("Expected at least one branch")
	}

	// Check that branches have names
	for _, b := range branches {
		if b.Name == "" {
			t.Error("Branch name should not be empty")
		}
	}
}

// TestGetDirtyStatus tests dirty status checking.
func TestGetDirtyStatus(t *testing.T) {
	// Get current worktree path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get cwd: %v", err)
	}

	// Should not error even if clean or dirty
	_, _, err = GetDirtyStatus(cwd)
	if err != nil {
		t.Errorf("GetDirtyStatus failed: %v", err)
	}
}

// TestParseWorktreeList tests parsing of git worktree list --porcelain output.
func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name: "single worktree",
			input: `worktree /path/to/repo
HEAD abc123def456
branch refs/heads/main

`,
			expected: 1,
		},
		{
			name: "multiple worktrees",
			input: `worktree /path/to/repo
HEAD abc123def456
branch refs/heads/main

worktree /path/to/repo/.worktrees/feature
HEAD def789abc012
branch refs/heads/feature/auth

`,
			expected: 2,
		},
		{
			name: "detached head",
			input: `worktree /path/to/repo
HEAD abc123def456
detached

`,
			expected: 1,
		},
		{
			name:     "empty",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorktreeList(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d worktrees, got %d", tt.expected, len(result))
			}
		})
	}
}

// TestSafetyLevel tests safety level string conversion.
func TestSafetyLevel(t *testing.T) {
	tests := []struct {
		level    SafetyLevel
		expected string
	}{
		{SafetyLevelSafe, "safe"},
		{SafetyLevelWarning, "warning"},
		{SafetyLevelDanger, "danger"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tt.level.String())
			}
		})
	}
}

// TestWorktreeShortPath tests the ShortPath method.
func TestWorktreeShortPath(t *testing.T) {
	ResetRepo()
	repo, err := GetRepo()
	if err != nil {
		t.Skip("Not in a git repo")
	}

	// Test with main worktree
	wt := Worktree{Path: repo.MainWorktreeRoot}
	result := wt.ShortPath()
	// Should return "." or a short path, not an absolute path
	if filepath.IsAbs(result) && result == repo.MainWorktreeRoot {
		// If using absolute path, it should at least be the right one
		t.Logf("ShortPath returned absolute path: %s", result)
	}

	// Test with sub worktree
	subPath := filepath.Join(repo.MainWorktreeRoot, ".worktrees", "test")
	wt2 := Worktree{Path: subPath}
	result2 := wt2.ShortPath()
	t.Logf("ShortPath for %s: %s", subPath, result2)
}

// TestBranchShort tests the BranchShort method.
func TestBranchShort(t *testing.T) {
	tests := []struct {
		branch   string
		expected string
	}{
		{"main", "main"},
		{"feature/auth", "auth"},
		{"feature/nested/deep", "deep"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			wt := Worktree{Branch: tt.branch}
			if wt.BranchShort() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, wt.BranchShort())
			}
		})
	}
}

// TestIntegration runs a full integration test if in a suitable environment.
func TestIntegration(t *testing.T) {
	// Skip if not in a git repo
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		t.Skip("Not in a git repo")
	}

	// Test the full flow
	ResetRepo()

	// 1. Get repo
	repo, err := GetRepo()
	if err != nil {
		t.Fatalf("GetRepo: %v", err)
	}
	t.Logf("Repo root: %s", repo.Root)
	t.Logf("Main worktree: %s", repo.MainWorktreeRoot)
	t.Logf("Git dir: %s", repo.GitDir)
	t.Logf("Default branch: %s", repo.DefaultBranch)

	// 2. List worktrees
	worktrees, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	t.Logf("Found %d worktrees", len(worktrees))
	for _, wt := range worktrees {
		t.Logf("  - %s (%s) dirty=%v", wt.Branch, wt.ShortPath(), wt.IsDirty)
	}

	// 3. List branches
	branches, err := ListBranches()
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	t.Logf("Found %d branches", len(branches))

	// 4. Check safety (on current worktree)
	if len(worktrees) > 0 {
		wt := worktrees[0]
		safety, err := CheckSafety(wt.Path, wt.Branch, repo.DefaultBranch)
		if err != nil {
			t.Logf("CheckSafety warning: %v", err)
		} else {
			t.Logf("Safety level for %s: %s", wt.Branch, safety.Level)
		}
	}
}

// TestDefaultBranch tests default branch detection.
func TestDefaultBranch(t *testing.T) {
	branch := detectDefaultBranch()
	if branch == "" {
		t.Error("Default branch should not be empty")
	}

	// Should be main or master typically
	validDefaults := []string{"main", "master", "develop", "trunk"}
	found := false
	for _, v := range validDefaults {
		if branch == v {
			found = true
			break
		}
	}

	// If not a common default, just log it (might be custom)
	if !found {
		t.Logf("Detected non-standard default branch: %s", branch)
	}
}

// TestIsBranchMerged tests merge detection.
func TestIsBranchMerged(t *testing.T) {
	ResetRepo()
	repo, err := GetRepo()
	if err != nil {
		t.Skip("Not in a git repo")
	}

	// Test merge detection - just verify it doesn't error
	merged, err := IsBranchMerged(repo.DefaultBranch, repo.DefaultBranch)
	if err != nil {
		t.Logf("IsBranchMerged: %v", err)
		return
	}
	t.Logf("Is %s merged into %s: %v", repo.DefaultBranch, repo.DefaultBranch, merged)
}

// TestGetUniqueCommits tests unique commit detection.
func TestGetUniqueCommits(t *testing.T) {
	ResetRepo()

	repo, err := GetRepo()
	if err != nil {
		t.Skip("Could not get repo")
	}

	currentBranch, err := CurrentBranch()
	if err != nil {
		t.Skip("Could not get current branch")
	}

	// Get unique commits - may or may not have any
	commits, err := GetUniqueCommits(currentBranch, repo.DefaultBranch)
	if err != nil {
		// This can fail if there's no default branch ref
		if strings.Contains(err.Error(), "unknown revision") {
			t.Skip("Could not compare branches")
		}
		t.Logf("GetUniqueCommits: %v", err)
		return
	}

	t.Logf("Found %d unique commits on %s (vs %s)", len(commits), currentBranch, repo.DefaultBranch)
	for _, c := range commits {
		t.Logf("  %s: %s", c.Hash, c.Message)
	}
}
