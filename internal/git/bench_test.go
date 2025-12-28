package git

import (
	"flag"
	"os"
	"sync"
	"testing"
	"time"
)

const benchRepoPathEnv = "GROVE_BENCH_PATH"

var benchRepoPathFlag = flag.String("bench-repo-path", "", "path to a git repo with worktrees for performance tests")

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
	benchRepoPath := *benchRepoPathFlag
	if benchRepoPath == "" {
		benchRepoPath = os.Getenv(benchRepoPathEnv)
	}
	if benchRepoPath == "" {
		t.Skip("set GROVE_BENCH_PATH or pass -bench-repo-path (via -args) to run this test")
	}
	if _, err := os.Stat(benchRepoPath); err != nil {
		if os.IsNotExist(err) {
			t.Skip("bench repo not available")
		}
		t.Fatalf("Failed to stat bench repo path: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
		ResetRepo() // Reset cached repo
	}()

	if err := os.Chdir(benchRepoPath); err != nil {
		t.Fatalf("Failed to change to bench repo directory: %v", err)
	}

	// Reset any cached state
	ResetRepo()

	// Test 1: Fresh load (no cache)
	t.Run("FreshLoad", func(t *testing.T) {
		start := time.Now()
		worktrees, err := List()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("List() error: %v", err)
		}

		t.Logf("Fresh load: %d worktrees in %v", len(worktrees), elapsed)
		t.Logf("Per worktree: %v", elapsed/time.Duration(len(worktrees)))
	})

	// Test 2: With cache (no TTL - always uses cache if available)
	t.Run("CachedLoad", func(t *testing.T) {
		// First call populates cache
		repo, _ := GetRepo()
		worktrees, _ := List()
		_ = SaveCache(repo.MainWorktreeRoot, worktrees)

		// Second call should hit cache instantly
		start := time.Now()
		cachedWorktrees, fromCache, err := ListCached()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("ListCached() error: %v", err)
		}

		t.Logf("Cached load: %d worktrees in %v (fromCache=%v)", len(cachedWorktrees), elapsed, fromCache)
		if !fromCache {
			t.Error("Expected cache hit")
		}
		if elapsed > 5*time.Millisecond {
			t.Errorf("Cache should be <5ms, got %v", elapsed)
		}
	})

	// Test 3: Just git status (the bottleneck)
	t.Run("GitStatusOnly", func(t *testing.T) {
		start := time.Now()
		_, _, err := GetDirtyStatus(".")
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("GetDirtyStatus() error: %v", err)
		}

		t.Logf("Single git status: %v", elapsed)
	})

	// Test 4: Parallel git status calls
	t.Run("ParallelGitStatus", func(t *testing.T) {
		worktrees, _ := List()
		paths := make([]string, len(worktrees))
		for i, wt := range worktrees {
			paths[i] = wt.Path
		}

		start := time.Now()
		var wg sync.WaitGroup
		errCh := make(chan error, len(paths))
		for _, path := range paths {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				_, _, err := GetDirtyStatus(p)
				if err != nil {
					errCh <- err
				}
			}(path)
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			t.Fatalf("GetDirtyStatus() error: %v", err)
		}
		elapsed := time.Since(start)

		t.Logf("Parallel git status (%d calls): %v", len(paths), elapsed)
		t.Logf("Speedup vs sequential: %.1fx", float64(len(paths))*50/float64(elapsed.Milliseconds()))
	})
}
