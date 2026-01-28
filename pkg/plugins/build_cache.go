// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"log"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// BuildCachePlugin manages incremental build caching.
// It loads the build cache at the start and saves it at the end.
// Other plugins can use the cache to skip unchanged posts.
type BuildCachePlugin struct {
	cache   *buildcache.Cache
	enabled bool
}

// NewBuildCachePlugin creates a new BuildCachePlugin.
func NewBuildCachePlugin() *BuildCachePlugin {
	return &BuildCachePlugin{
		enabled: true,
	}
}

// Name returns the unique name of the plugin.
func (p *BuildCachePlugin) Name() string {
	return "build_cache"
}

// Priority returns high priority so it runs early in configure and late in cleanup.
func (p *BuildCachePlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageConfigure:
		return lifecycle.PriorityEarly - 100 // Very early in configure
	case lifecycle.StageCleanup:
		return lifecycle.PriorityLate + 100 // Very late in cleanup
	default:
		return lifecycle.PriorityDefault
	}
}

// Configure loads the build cache and checks for global invalidation.
func (p *BuildCachePlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Check if caching is disabled
	if config.Extra != nil {
		if cacheConfig, ok := config.Extra["build_cache"].(map[string]interface{}); ok {
			if enabled, ok := cacheConfig["enabled"].(bool); ok && !enabled {
				p.enabled = false
				return nil
			}
		}
	}

	// Load existing cache
	cacheDir := filepath.Join(config.OutputDir, "..", ".markata")
	if config.Extra != nil {
		if dir, ok := config.Extra["cache_dir"].(string); ok && dir != "" {
			cacheDir = dir
		}
	}

	cache, err := buildcache.Load(cacheDir)
	if err != nil {
		// Non-fatal: start with empty cache
		cache = buildcache.New(cacheDir)
	}
	p.cache = cache

	// Store cache in manager for other plugins to access
	m.Cache().Set("build_cache", cache)

	// Compute and check config hash
	// For now, use a simple hash of the config file if it exists
	configFiles := []string{"markata-go.toml", "markata-go.yaml", "markata-go.json"}
	for _, cf := range configFiles {
		if hash, err := buildcache.HashFile(cf); err == nil {
			if cache.SetConfigHash(hash) {
				// Config changed - cache was invalidated
				log.Printf("[build_cache] Config changed, full rebuild required")
			}
			break
		}
	}

	// Compute and check templates hash
	templatesDir := PluginNameTemplates // default "templates"
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}
	if hash, err := buildcache.HashDirectory(templatesDir, []string{".html", ".txt", ".md"}); err == nil && hash != "" {
		if cache.SetTemplatesHash(hash) {
			// Templates changed - cache was invalidated
			log.Printf("[build_cache] Templates changed, full rebuild required")
		}
	}

	// Reset stats for this build
	cache.ResetStats()

	return nil
}

// Cleanup saves the build cache to disk.
func (p *BuildCachePlugin) Cleanup(m *lifecycle.Manager) error {
	if !p.enabled || p.cache == nil {
		return nil
	}

	// Remove stale entries (posts that no longer exist)
	posts := m.Posts()
	currentPaths := make(map[string]bool, len(posts))
	for _, post := range posts {
		currentPaths[post.Path] = true
	}
	removed := p.cache.RemoveStale(currentPaths)
	if removed > 0 {
		log.Printf("[build_cache] Removed %d stale cache entries", removed)
	}

	// Save cache
	if err := p.cache.Save(); err != nil {
		log.Printf("[build_cache] Failed to save build cache: %v", err)
	}

	// Log stats
	skipped, rebuilt := p.cache.Stats()
	if skipped > 0 || rebuilt > 0 {
		log.Printf("[build_cache] Incremental build: %d skipped, %d rebuilt", skipped, rebuilt)
		m.Cache().Set("build_cache_skipped", skipped)
		m.Cache().Set("build_cache_rebuilt", rebuilt)
	}

	return nil
}

// Cache returns the build cache instance.
func (p *BuildCachePlugin) Cache() *buildcache.Cache {
	return p.cache
}

// Enabled returns whether the build cache is enabled.
func (p *BuildCachePlugin) Enabled() bool {
	return p.enabled && p.cache != nil
}

// GetBuildCache retrieves the build cache from the manager's cache.
// Returns nil if not found or caching is disabled.
func GetBuildCache(m *lifecycle.Manager) *buildcache.Cache {
	if cached, ok := m.Cache().Get("build_cache"); ok {
		if bc, ok := cached.(*buildcache.Cache); ok {
			return bc
		}
	}
	return nil
}

// Ensure BuildCachePlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*BuildCachePlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*BuildCachePlugin)(nil)
	_ lifecycle.CleanupPlugin   = (*BuildCachePlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*BuildCachePlugin)(nil)
)
