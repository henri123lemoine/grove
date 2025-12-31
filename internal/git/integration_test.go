package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repo for testing.
// Returns the repo path and a cleanup function.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "grove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	// Initialize git repo
	if err := runIn(tmpDir, "git", "init"); err != nil {
		cleanup()
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for commits
	if err := runIn(tmpDir, "git", "config", "user.email", "test@test.com"); err != nil {
		cleanup()
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runIn(tmpDir, "git", "config", "user.name", "Test User"); err != nil {
		cleanup()
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		cleanup()
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := runIn(tmpDir, "git", "add", "."); err != nil {
		cleanup()
		t.Fatalf("git add failed: %v", err)
	}
	if err := runIn(tmpDir, "git", "commit", "-m", "Initial commit"); err != nil {
		cleanup()
		t.Fatalf("git commit failed: %v", err)
	}

	return tmpDir, cleanup
}

// runIn runs a command in a directory.
func runIn(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// TestWorktreeCreateAndDelete tests the full worktree lifecycle.
func TestWorktreeCreateAndDelete(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Change to repo directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	// Verify we can detect the repo
	repo, err := GetRepo()
	if err != nil {
		t.Fatalf("GetRepo failed: %v", err)
	}
	if repo.MainWorktreeRoot != repoDir {
		t.Errorf("MainWorktreeRoot = %q, want %q", repo.MainWorktreeRoot, repoDir)
	}

	// List worktrees - should have 1 (main)
	worktrees, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("Expected 1 worktree, got %d", len(worktrees))
	}

	// Create a new worktree with new branch
	wtPath := filepath.Join(repoDir, ".worktrees", "feature-test")
	if err := Create(wtPath, "feature-test", true, ""); err != nil {
		t.Fatalf("Create worktree failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("Worktree directory was not created")
	}

	// List again - should have 2
	worktrees, err = List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(worktrees))
	}

	// Find the new worktree
	var newWt *Worktree
	for i := range worktrees {
		if worktrees[i].Branch == "feature-test" {
			newWt = &worktrees[i]
			break
		}
	}
	if newWt == nil {
		t.Fatal("Could not find feature-test worktree")
	}
	if newWt.Path != wtPath {
		t.Errorf("Worktree path = %q, want %q", newWt.Path, wtPath)
	}

	// Delete the worktree
	if err := Remove(wtPath, false); err != nil {
		t.Fatalf("Remove worktree failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("Worktree directory still exists after removal")
	}

	// List again - should have 1
	worktrees, err = List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("Expected 1 worktree after delete, got %d", len(worktrees))
	}
}

// TestWorktreeFromExistingBranch tests creating a worktree from an existing branch.
func TestWorktreeFromExistingBranch(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	// Create a branch first
	if err := runIn(repoDir, "git", "branch", "existing-branch"); err != nil {
		t.Fatalf("git branch failed: %v", err)
	}

	// Create worktree from existing branch
	wtPath := filepath.Join(repoDir, ".worktrees", "existing")
	if err := Create(wtPath, "existing-branch", false, ""); err != nil {
		t.Fatalf("Create worktree from existing branch failed: %v", err)
	}

	// Verify
	worktrees, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(worktrees))
	}

	// Cleanup
	_ = Remove(wtPath, false)
}

// TestDirtyStatus tests dirty status detection.
func TestDirtyStatusIntegration(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	// Initially should be clean
	dirty, count, err := GetDirtyStatus(repoDir)
	if err != nil {
		t.Fatalf("GetDirtyStatus failed: %v", err)
	}
	if dirty {
		t.Errorf("Expected clean repo, got dirty with %d changes", count)
	}

	// Create an untracked file
	if err := os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Should be dirty now
	dirty, count, err = GetDirtyStatus(repoDir)
	if err != nil {
		t.Fatalf("GetDirtyStatus failed: %v", err)
	}
	if !dirty {
		t.Error("Expected dirty repo after adding untracked file")
	}
	if count != 1 {
		t.Errorf("Expected 1 change, got %d", count)
	}

	// Modify an existing file
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Modified\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	dirty, count, err = GetDirtyStatus(repoDir)
	if err != nil {
		t.Fatalf("GetDirtyStatus failed: %v", err)
	}
	if !dirty {
		t.Error("Expected dirty repo")
	}
	if count != 2 {
		t.Errorf("Expected 2 changes, got %d", count)
	}
}

// TestSafetyCheck tests the safety check for worktree deletion.
func TestSafetyCheckIntegration(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	repo, err := GetRepo()
	if err != nil {
		t.Fatalf("GetRepo failed: %v", err)
	}

	// Create a worktree with a new branch
	wtPath := filepath.Join(repoDir, ".worktrees", "feature")
	if err := Create(wtPath, "feature", true, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() { _ = Remove(wtPath, true) }()

	// Safety check on clean worktree with no unique commits - should be safe
	safety, err := CheckSafety(wtPath, "feature", repo.DefaultBranch)
	if err != nil {
		t.Fatalf("CheckSafety failed: %v", err)
	}
	if safety.Level != SafetyLevelSafe {
		t.Errorf("Expected SafetyLevelSafe, got %s", safety.Level)
	}

	// Add uncommitted changes - this is danger level (unrecoverable)
	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	safety, err = CheckSafety(wtPath, "feature", repo.DefaultBranch)
	if err != nil {
		t.Fatalf("CheckSafety failed: %v", err)
	}
	if safety.Level != SafetyLevelDanger {
		t.Errorf("Expected SafetyLevelDanger for dirty worktree, got %s", safety.Level)
	}
	if !safety.HasUncommittedChanges {
		t.Error("Expected HasUncommittedChanges=true")
	}

	// Commit the change - still danger level (unique unpushed commits)
	if err := runIn(wtPath, "git", "add", "."); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runIn(wtPath, "git", "commit", "-m", "Feature commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	safety, err = CheckSafety(wtPath, "feature", repo.DefaultBranch)
	if err != nil {
		t.Fatalf("CheckSafety failed: %v", err)
	}
	if safety.Level != SafetyLevelDanger {
		t.Errorf("Expected SafetyLevelDanger for unique commits, got %s", safety.Level)
	}
	if len(safety.UniqueCommits) != 1 {
		t.Errorf("Expected 1 unique commit, got %d", len(safety.UniqueCommits))
	}
}

// TestBranchOperations tests branch-related operations.
func TestBranchOperations(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	// List branches - should have main/master
	branches, err := ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("Expected 1 branch, got %d", len(branches))
	}

	// Create a new branch
	if err := runIn(repoDir, "git", "branch", "new-branch"); err != nil {
		t.Fatalf("git branch failed: %v", err)
	}

	// Should now have 2 branches
	branches, err = ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("Expected 2 branches, got %d", len(branches))
	}

	// BranchExists should work
	if !BranchExists("new-branch") {
		t.Error("new-branch should exist")
	}
	if BranchExists("nonexistent") {
		t.Error("nonexistent should not exist")
	}

	// Delete the branch
	if err := DeleteBranch("new-branch", false); err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// Should be back to 1 branch
	branches, err = ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("Expected 1 branch after delete, got %d", len(branches))
	}
}

// TestCacheOperations tests cache save/load.
func TestCacheOperations(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo()
	}()
	ResetRepo()

	// Get fresh list
	worktrees, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Save to cache
	if err := SaveCache(repoDir, worktrees); err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// Load from cache
	cache := LoadCache(repoDir)
	if cache == nil {
		t.Fatal("LoadCache returned nil")
	}
	if len(cache.Worktrees) != len(worktrees) {
		t.Errorf("Cache has %d worktrees, expected %d", len(cache.Worktrees), len(worktrees))
	}
	if cache.RepoRoot != repoDir {
		t.Errorf("Cache RepoRoot = %q, want %q", cache.RepoRoot, repoDir)
	}

	// Verify cache data matches original
	if len(cache.Worktrees) > 0 && len(worktrees) > 0 {
		if cache.Worktrees[0].Path != worktrees[0].Path {
			t.Errorf("Cached worktree path mismatch: %q vs %q", cache.Worktrees[0].Path, worktrees[0].Path)
		}
	}
}
