// Package buildcache provides incremental build caching for markata-go.
//
// The cache tracks input hashes (content + frontmatter + template) for each post
// and allows skipping rebuild when inputs haven't changed. It also tracks global
// hashes (templates, config) to invalidate the entire cache when needed.
//
// # Cache File Structure
//
// The cache is stored in .markata/build-cache.json with the structure:
//
//	{
//	  "version": 1,
//	  "config_hash": "abc123",
//	  "templates_hash": "def456",
//	  "posts": {
//	    "path/to/post.md": {
//	      "input_hash": "xyz789",
//	      "output_path": "output/post/index.html",
//	      "template": "post.html"
//	    }
//	  },
//	  "graph": {
//	    "dependencies": {"path/to/post.md": ["linked-slug"]},
//	    "path_to_slug": {"path/to/post.md": "post-slug"}
//	  }
//	}
package buildcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// CacheVersion is incremented when the cache format changes.
const CacheVersion = 1

// DefaultCacheDir is the directory for cache files.
const DefaultCacheDir = ".markata"

// CacheFileName is the name of the build cache file.
const CacheFileName = "build-cache.json"

// Cache manages incremental build state.
type Cache struct {
	mu sync.RWMutex

	// Version of the cache format
	Version int `json:"version"`

	// ConfigHash is the hash of the config file contents
	ConfigHash string `json:"config_hash"`

	// TemplatesHash is the combined hash of all template files
	TemplatesHash string `json:"templates_hash"`

	// Posts maps source path to cached post metadata
	Posts map[string]*PostCache `json:"posts"`

	// Graph tracks dependencies between posts for transitive invalidation
	Graph *DependencyGraph `json:"graph,omitempty"`

	// path to the cache file (not serialized)
	path string

	// dirty tracks whether cache needs saving
	dirty bool

	// skippedCount tracks how many posts were skipped this build
	skippedCount int

	// rebuiltCount tracks how many posts were rebuilt this build
	rebuiltCount int

	// changedSlugs tracks slugs that changed this build (for dependency invalidation)
	changedSlugs map[string]bool
}

// PostCache stores cached metadata for a single post.
type PostCache struct {
	// InputHash is the hash of content + frontmatter + template
	InputHash string `json:"input_hash"`

	// OutputPath is the primary output file path
	OutputPath string `json:"output_path"`

	// Template is the template name used for rendering
	Template string `json:"template"`

	// OutputHash is the hash of the rendered output (optional, for verification)
	OutputHash string `json:"output_hash,omitempty"`
}

// New creates a new empty cache.
func New(cacheDir string) *Cache {
	if cacheDir == "" {
		cacheDir = DefaultCacheDir
	}
	return &Cache{
		Version:      CacheVersion,
		Posts:        make(map[string]*PostCache),
		Graph:        NewDependencyGraph(),
		path:         filepath.Join(cacheDir, CacheFileName),
		changedSlugs: make(map[string]bool),
	}
}

// Load reads the cache from disk. Returns a new empty cache if file doesn't exist.
func Load(cacheDir string) (*Cache, error) {
	cache := New(cacheDir)

	data, err := os.ReadFile(cache.path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil // Empty cache is fine
		}
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	if err := json.Unmarshal(data, cache); err != nil {
		// Corrupt cache, start fresh
		return New(cacheDir), nil
	}

	// Version mismatch, start fresh
	if cache.Version != CacheVersion {
		return New(cacheDir), nil
	}

	// Ensure Graph is initialized (might be nil from old cache versions)
	if cache.Graph == nil {
		cache.Graph = NewDependencyGraph()
	} else {
		// Rebuild the reverse index (Dependents) from persisted Dependencies
		cache.Graph.RebuildReverse()
	}

	// Ensure changedSlugs is initialized
	cache.changedSlugs = make(map[string]bool)

	return cache, nil
}

// Save writes the cache to disk.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	//nolint:gosec // G306: Cache file needs 0644 for sharing across builds
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	c.dirty = false
	return nil
}

// SetConfigHash updates the config hash and invalidates if changed.
func (c *Cache) SetConfigHash(hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ConfigHash == hash {
		return false // No change
	}

	// Config changed - invalidate all posts
	c.ConfigHash = hash
	c.Posts = make(map[string]*PostCache)
	c.dirty = true
	return true
}

// SetTemplatesHash updates the templates hash and invalidates if changed.
func (c *Cache) SetTemplatesHash(hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.TemplatesHash == hash {
		return false // No change
	}

	// Templates changed - invalidate all posts
	c.TemplatesHash = hash
	c.Posts = make(map[string]*PostCache)
	c.dirty = true
	return true
}

// ShouldRebuild checks if a post needs rebuilding based on input hash.
// Returns true if the post should be rebuilt (hash mismatch or not in cache).
// Also returns true if any post this one depends on has changed.
func (c *Cache) ShouldRebuild(sourcePath, inputHash, template string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return true // Not in cache
	}

	// Check if hash matches and template is the same
	if cached.InputHash != inputHash || cached.Template != template {
		return true // Changed
	}

	return false
}

// ShouldRebuildWithSlug checks if a post needs rebuilding based on input hash
// and also checks if any of the posts it depends on have changed this build.
// The slug is used for dependency tracking.
func (c *Cache) ShouldRebuildWithSlug(sourcePath, _, inputHash, template string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return true // Not in cache
	}

	// Check if hash matches and template is the same
	if cached.InputHash != inputHash || cached.Template != template {
		return true // Changed
	}

	// Check if any dependency has changed this build
	if deps := c.Graph.GetDependencies(sourcePath); len(deps) > 0 {
		for _, dep := range deps {
			if c.changedSlugs[dep] {
				return true // A dependency changed
			}
		}
	}

	return false
}

// MarkRebuilt records that a post was rebuilt with the given hash.
// The slug is used for dependency invalidation tracking.
func (c *Cache) MarkRebuilt(sourcePath, inputHash, outputPath, template string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Posts[sourcePath] = &PostCache{
		InputHash:  inputHash,
		OutputPath: outputPath,
		Template:   template,
	}
	c.dirty = true
	c.rebuiltCount++
}

// MarkRebuiltWithSlug records that a post was rebuilt with the given hash.
// Also records that this slug changed, for dependency invalidation.
func (c *Cache) MarkRebuiltWithSlug(sourcePath, slug, inputHash, outputPath, template string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Posts[sourcePath] = &PostCache{
		InputHash:  inputHash,
		OutputPath: outputPath,
		Template:   template,
	}
	c.dirty = true
	c.rebuiltCount++

	// Track that this slug changed (for dependency invalidation)
	if slug != "" {
		c.changedSlugs[slug] = true
	}
}

// MarkSkipped records that a post was skipped (already up to date).
func (c *Cache) MarkSkipped() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skippedCount++
}

// Stats returns build statistics.
func (c *Cache) Stats() (skipped, rebuilt int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.skippedCount, c.rebuiltCount
}

// ResetStats resets the build statistics for a new build.
func (c *Cache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skippedCount = 0
	c.rebuiltCount = 0
	c.changedSlugs = make(map[string]bool)
}

// RemoveStale removes cache entries for posts that no longer exist.
// Returns the number of entries removed.
func (c *Cache) RemoveStale(currentPaths map[string]bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for path := range c.Posts {
		if !currentPaths[path] {
			delete(c.Posts, path)
			removed++
			c.dirty = true
		}
	}
	return removed
}

// HashContent computes a SHA256 hash of the given content.
func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// HashFile computes a SHA256 hash of a file's contents.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashDirectory computes a combined hash of all files in a directory.
// Files are processed in sorted order for deterministic output.
func HashDirectory(dir string, extensions []string) (string, error) {
	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[ext] = true
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if len(extMap) == 0 || extMap[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Directory doesn't exist, empty hash
		}
		return "", err
	}

	// Sort for deterministic ordering
	sort.Strings(files)

	h := sha256.New()
	for _, path := range files {
		// Include relative path in hash (so renames are detected)
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = path // fallback to absolute path
		}
		h.Write([]byte(relPath))

		// Include file contents
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		h.Write(content)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputePostInputHash computes the input hash for a post.
// This combines: content + frontmatter fields that affect output + template name.
func ComputePostInputHash(content, frontmatter, template string) string {
	h := sha256.New()
	h.Write([]byte(content))
	h.Write([]byte("\x00")) // Separator
	h.Write([]byte(frontmatter))
	h.Write([]byte("\x00"))
	h.Write([]byte(template))
	return hex.EncodeToString(h.Sum(nil))
}

// SetDependencies records what targets a source post links to.
// This delegates to the underlying DependencyGraph.
func (c *Cache) SetDependencies(sourcePath, sourceSlug string, targets []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Graph.SetDependencies(sourcePath, sourceSlug, targets)
	c.dirty = true
}

// GetAffectedPosts returns all posts that need rebuilding when the given
// slugs change. This performs transitive closure via the dependency graph.
func (c *Cache) GetAffectedPosts(changedSlugs []string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Graph.GetAffectedPosts(changedSlugs)
}

// GetChangedSlugs returns the slugs that changed during this build.
func (c *Cache) GetChangedSlugs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, 0, len(c.changedSlugs))
	for slug := range c.changedSlugs {
		result = append(result, slug)
	}
	sort.Strings(result)
	return result
}

// MarkSlugChanged records that a slug changed this build.
// Used for dependency invalidation.
func (c *Cache) MarkSlugChanged(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changedSlugs[slug] = true
}

// GraphSize returns the number of posts with dependencies tracked.
func (c *Cache) GraphSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Graph.Size()
}
