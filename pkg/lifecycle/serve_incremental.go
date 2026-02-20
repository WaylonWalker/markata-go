package lifecycle

import (
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Serve incremental rebuild cache keys.
const (
	CacheKeyServeChangedPaths = "serve.changed_paths"
	CacheKeyServeAffectedPath = "serve.affected_paths"
	CacheKeyServeFullRebuild  = "serve.full_rebuild"
	CacheKeyServeCachedPosts  = "serve.cached_posts"
)

// SetServeChangedPaths stores normalized, relative content paths that changed.
func SetServeChangedPaths(m *Manager, paths []string) {
	if m == nil {
		return
	}
	if len(paths) == 0 {
		m.Cache().Delete(CacheKeyServeChangedPaths)
		return
	}
	clean := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		cp := filepath.Clean(p)
		if _, ok := seen[cp]; ok {
			continue
		}
		seen[cp] = struct{}{}
		clean = append(clean, cp)
	}
	m.Cache().Set(CacheKeyServeChangedPaths, clean)
}

// GetServeChangedPaths returns the changed content paths if present.
func GetServeChangedPaths(m *Manager) []string {
	if m == nil {
		return nil
	}
	if raw, ok := m.Cache().Get(CacheKeyServeChangedPaths); ok {
		if paths, ok := raw.([]string); ok {
			return paths
		}
	}
	return nil
}

// SetServeAffectedPaths stores a lookup map of content paths that should be rebuilt.
func SetServeAffectedPaths(m *Manager, paths map[string]bool) {
	if m == nil {
		return
	}
	if len(paths) == 0 {
		m.Cache().Delete(CacheKeyServeAffectedPath)
		return
	}
	m.Cache().Set(CacheKeyServeAffectedPath, paths)
}

// GetServeAffectedPaths returns a lookup map of paths to rebuild.
func GetServeAffectedPaths(m *Manager) map[string]bool {
	if m == nil {
		return nil
	}
	if raw, ok := m.Cache().Get(CacheKeyServeAffectedPath); ok {
		if paths, ok := raw.(map[string]bool); ok {
			return paths
		}
	}
	return nil
}

// SetServeFullRebuild stores whether the current serve rebuild should be full.
func SetServeFullRebuild(m *Manager, full bool) {
	if m == nil {
		return
	}
	m.Cache().Set(CacheKeyServeFullRebuild, full)
}

// IsServeFullRebuild returns whether the current serve rebuild is full.
func IsServeFullRebuild(m *Manager) bool {
	if m == nil {
		return false
	}
	if raw, ok := m.Cache().Get(CacheKeyServeFullRebuild); ok {
		if full, ok := raw.(bool); ok {
			return full
		}
	}
	return false
}

// IsServeFastMode returns true when fast_mode is enabled in config.
func IsServeFastMode(m *Manager) bool {
	if m == nil {
		return false
	}
	if extra := m.Config().Extra; extra != nil {
		if fast, ok := extra["fast_mode"].(bool); ok {
			return fast
		}
	}
	return false
}

// SetServeCachedPosts stores cached posts for incremental serve rebuilds.
func SetServeCachedPosts(m *Manager, posts map[string]*models.Post) {
	if m == nil {
		return
	}
	if len(posts) == 0 {
		m.Cache().Delete(CacheKeyServeCachedPosts)
		return
	}
	m.Cache().Set(CacheKeyServeCachedPosts, posts)
}

// GetServeCachedPosts returns cached posts for incremental serve rebuilds.
func GetServeCachedPosts(m *Manager) map[string]*models.Post {
	if m == nil {
		return nil
	}
	if raw, ok := m.Cache().Get(CacheKeyServeCachedPosts); ok {
		if posts, ok := raw.(map[string]*models.Post); ok {
			return posts
		}
	}
	return nil
}
