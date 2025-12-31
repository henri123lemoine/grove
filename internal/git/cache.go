package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// WorktreeCache represents cached worktree data.
type WorktreeCache struct {
	RepoRoot  string     `json:"repo_root"`
	Worktrees []Worktree `json:"worktrees"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// getCachePath returns the cache file path for the current repo.
func getCachePath(repoRoot string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	// Use hash of repo path to avoid conflicts
	safeKey := filepath.Base(repoRoot)
	return filepath.Join(cacheDir, "grove", safeKey+".json")
}

// LoadCache attempts to load cached worktree data.
// Returns nil if cache doesn't exist or is for a different repo.
// Always returns cached data regardless of age - caller decides whether to refresh.
func LoadCache(repoRoot string) *WorktreeCache {
	path := getCachePath(repoRoot)

	// Acquire shared (read) lock - blocks if exclusive lock is held
	fileLock := flock.New(path + ".lock")
	if err := fileLock.RLock(); err != nil {
		return nil
	}
	defer fileLock.Unlock()

	// Read and parse
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cache WorktreeCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}

	// Check if cache is for the right repo
	if cache.RepoRoot != repoRoot {
		return nil
	}

	return &cache
}

// SaveCache saves worktree data to cache.
func SaveCache(repoRoot string, worktrees []Worktree) error {
	cache := WorktreeCache{
		RepoRoot:  repoRoot,
		Worktrees: worktrees,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	path := getCachePath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	// Acquire exclusive lock - blocks until lock is available
	fileLock := flock.New(path + ".lock")
	if err := fileLock.Lock(); err != nil {
		return err
	}
	defer fileLock.Unlock()

	// Write atomically: write to temp file then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// ListCached returns worktrees from cache if available, otherwise fetches fresh.
// Always returns fromCache=true if cache exists (caller should always refresh in background).
func ListCached() ([]Worktree, bool, error) {
	repo, err := GetRepo()
	if err != nil {
		return nil, false, err
	}

	// Try cache first - use it regardless of age for instant startup
	if cache := LoadCache(repo.MainWorktreeRoot); cache != nil {
		// Always indicate cache hit so caller triggers background refresh
		return cache.Worktrees, true, nil
	}

	// Cache miss - fetch fresh (only happens on first run)
	worktrees, err := List()
	if err != nil {
		return nil, false, err
	}

	// Save to cache (ignore errors)
	_ = SaveCache(repo.MainWorktreeRoot, worktrees)

	return worktrees, false, nil
}

// ListAndCache fetches fresh worktrees and saves to cache.
func ListAndCache() ([]Worktree, error) {
	worktrees, err := List()
	if err != nil {
		return nil, err
	}

	repo, err := GetRepo()
	if err == nil {
		_ = SaveCache(repo.MainWorktreeRoot, worktrees)
	}

	return worktrees, nil
}
