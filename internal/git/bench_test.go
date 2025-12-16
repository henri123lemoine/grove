package git

import (
	"os"
	"testing"
	"time"
)

func BenchmarkListWorktrees(b *testing.B) {
	// Skip if not in a git repo
	if _, err := GetRepo(); err != nil {
		b.Skip("Not in a git repo")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := List()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestListPerformance(t *testing.T) {
	// Test with linuxbench if available
	linuxbenchPath := "/Users/henrilemoine/Documents/Work/AI/AI_Control/EquiStamp/linuxbench"
	if _, err := os.Stat(linuxbenchPath); os.IsNotExist(err) {
		t.Skip("linuxbench not available")
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(linuxbenchPath); err != nil {
		t.Fatalf("Failed to change to linuxbench directory: %v", err)
	}

	start := time.Now()
	worktrees, err := List()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	t.Logf("Found %d worktrees in %v", len(worktrees), elapsed)
	t.Logf("Average time per worktree: %v", elapsed/time.Duration(len(worktrees)))

	// Performance threshold: should complete within 10 seconds for 50+ worktrees
	if len(worktrees) > 0 && elapsed > 10*time.Second {
		t.Errorf("Performance too slow: %v for %d worktrees", elapsed, len(worktrees))
	}
}
