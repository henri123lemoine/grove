package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CacheTTL is how long cached data is considered fresh.
const CacheTTL = 30 * time.Second

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
// Returns nil if cache doesn't exist, is stale, or is for a different repo.
func LoadCache(repoRoot string) *WorktreeCache {
	path := getCachePath(repoRoot)
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

	// Check if cache is still fresh
	if time.Since(cache.UpdatedAt) > CacheTTL {
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ListCached returns worktrees from cache if available, otherwise fetches fresh.
// Also returns whether the data came from cache.
func ListCached() ([]Worktree, bool, error) {
	repo, err := GetRepo()
	if err != nil {
		return nil, false, err
	}

	// Try cache first
	if cache := LoadCache(repo.MainWorktreeRoot); cache != nil {
		return cache.Worktrees, true, nil
	}

	// Cache miss - fetch fresh
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
