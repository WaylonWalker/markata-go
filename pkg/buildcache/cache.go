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
	"strings"
	"sync"
	"time"
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

	// AssetsHash is the combined hash of all static assets (JS/CSS)
	AssetsHash string `json:"assets_hash,omitempty"`

	// Posts maps source path to cached post metadata
	Posts map[string]*PostCache `json:"posts"`

	// Feeds maps feed slug to cached feed metadata
	Feeds map[string]*FeedCache `json:"feeds,omitempty"`

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

	// GlobFiles caches the list of discovered files from glob stage
	GlobFiles []string `json:"glob_files,omitempty"`

	// GlobPatternHash detects when glob patterns change
	GlobPatternHash string `json:"glob_pattern_hash,omitempty"`
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

	// ContentHash is the hash of just the markdown content (for render caching)
	ContentHash string `json:"content_hash,omitempty"`

	// ArticleHTMLPath is the path to the cached rendered HTML file
	ArticleHTMLPath string `json:"article_html_path,omitempty"`

	// FullHTMLPath is the path to the cached full page HTML file
	FullHTMLPath string `json:"full_html_path,omitempty"`

	// ModTime is the file modification time (Unix nanoseconds)
	ModTime int64 `json:"mod_time,omitempty"`

	// Slug is the post's slug for dependency tracking
	Slug string `json:"slug,omitempty"`

	// LinkHrefsHash is a hash of the post's article HTML used for link extraction caching
	LinkHrefsHash string `json:"link_hrefs_hash,omitempty"`

	// LinkHrefs caches extracted href values for the post
	LinkHrefs []string `json:"link_hrefs,omitempty"`

	// FeedMembershipHash is the hash of the sorted slugs of feed co-members.
	// When feed membership changes (posts added/removed from a tag), this hash
	// changes and the post is rebuilt with the updated sidebar.
	FeedMembershipHash string `json:"feed_membership_hash,omitempty"`
}

// FeedCache stores cached metadata for a single feed.
type FeedCache struct {
	// Hash is the hash of the feed's content (post slugs, config)
	Hash string `json:"hash"`
}

// New creates a new empty cache.
func New(cacheDir string) *Cache {
	if cacheDir == "" {
		cacheDir = DefaultCacheDir
	}
	return &Cache{
		Version:      CacheVersion,
		Posts:        make(map[string]*PostCache),
		Feeds:        make(map[string]*FeedCache),
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

	// Ensure Feeds is initialized
	if cache.Feeds == nil {
		cache.Feeds = make(map[string]*FeedCache)
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

// SetAssetsHash updates the assets hash and invalidates if changed.
// This ensures pages are rebuilt when JS/CSS files change so they reference new hashed filenames.
func (c *Cache) SetAssetsHash(hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.AssetsHash == hash {
		return false // No change
	}

	// Assets changed - invalidate all posts
	c.AssetsHash = hash
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

// IsFileUnchanged checks if a file's ModTime matches the cached value.
// Returns true if the file has not changed since last build.
// Returns false if file is not in cache or ModTime differs.
func (c *Cache) IsFileUnchanged(sourcePath string, modTime int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return false
	}
	return cached.ModTime == modTime && cached.ModTime != 0
}

// GetCachedPost returns the cached post metadata if the file hasn't changed.
// Returns nil if file is not in cache or has changed.
func (c *Cache) GetCachedPost(sourcePath string) *PostCache {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Posts[sourcePath]
}

// UpdateModTime updates the ModTime for a post.
func (c *Cache) UpdateModTime(sourcePath string, modTime int64, slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.ModTime = modTime
		cached.Slug = slug
	} else {
		c.Posts[sourcePath] = &PostCache{
			ModTime: modTime,
			Slug:    slug,
		}
	}
	c.dirty = true
}

// GetCachedLinkHrefs returns cached hrefs for a post if the article hash matches.
// Returns nil if no matching cache exists.
func (c *Cache) GetCachedLinkHrefs(sourcePath, articleHash string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return nil
	}
	if cached.LinkHrefsHash == "" || cached.LinkHrefsHash != articleHash {
		return nil
	}
	if len(cached.LinkHrefs) == 0 {
		return []string{}
	}

	hrefs := make([]string, len(cached.LinkHrefs))
	copy(hrefs, cached.LinkHrefs)
	return hrefs
}

// CacheLinkHrefs stores extracted hrefs for a post keyed by article hash.
func (c *Cache) CacheLinkHrefs(sourcePath, articleHash string, hrefs []string) {
	if sourcePath == "" || articleHash == "" {
		return
	}

	copyHrefs := make([]string, len(hrefs))
	copy(copyHrefs, hrefs)

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.LinkHrefsHash = articleHash
		cached.LinkHrefs = copyHrefs
	} else {
		c.Posts[sourcePath] = &PostCache{
			LinkHrefsHash: articleHash,
			LinkHrefs:     copyHrefs,
		}
	}
	c.dirty = true
}

// SetFeedMembershipHash stores the feed membership hash for a post.
func (c *Cache) SetFeedMembershipHash(sourcePath, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.Posts[sourcePath]; ok {
		cached.FeedMembershipHash = hash
	}
	c.dirty = true
}

// GetFeedMembershipHash returns the cached feed membership hash for a post.
func (c *Cache) GetFeedMembershipHash(sourcePath string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if cached, ok := c.Posts[sourcePath]; ok {
		return cached.FeedMembershipHash
	}
	return ""
}

// ComputeFeedMembershipHash computes a hash of the sorted slugs of feed co-members.
// Returns empty string if the slug list is empty.
func ComputeFeedMembershipHash(slugs []string) string {
	if len(slugs) == 0 {
		return ""
	}
	sorted := make([]string, len(slugs))
	copy(sorted, slugs)
	sort.Strings(sorted)
	return HashContent(strings.Join(sorted, "\x00"))
}

// Stats returns build statistics.
func (c *Cache) Stats() (skipped, rebuilt int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.skippedCount, c.rebuiltCount
}

// CacheStats holds build statistics.
type CacheStats struct {
	Skipped int
	Rebuilt int
}

// GetStats returns build statistics as a struct.
func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		Skipped: c.skippedCount,
		Rebuilt: c.rebuiltCount,
	}
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

// HashAssetMap computes a combined hash from a map of asset paths to content hashes.
// This is used to detect when any JS/CSS file changes so we can invalidate cached HTML.
func HashAssetMap(assetHashes map[string]string) string {
	if len(assetHashes) == 0 {
		return ""
	}

	// Get sorted list of paths for deterministic ordering
	paths := make([]string, 0, len(assetHashes))
	for path := range assetHashes {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Combine path+hash pairs
	h := sha256.New()
	for _, path := range paths {
		h.Write([]byte(path))
		h.Write([]byte(assetHashes[path]))
	}

	return hex.EncodeToString(h.Sum(nil))
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

// GetGlobCache returns the cached glob file list and pattern hash.
func (c *Cache) GetGlobCache() (files []string, patternHash string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GlobFiles, c.GlobPatternHash
}

// SetGlobCache stores the glob file list and pattern hash.
func (c *Cache) SetGlobCache(files []string, patternHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.GlobFiles = files
	c.GlobPatternHash = patternHash
	c.dirty = true
}

// GraphSize returns the number of posts with dependencies tracked.
func (c *Cache) GraphSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Graph.Size()
}

// HTMLCacheDir is the subdirectory for cached rendered HTML files.
const HTMLCacheDir = "html-cache"

// GetCachedArticleHTML returns the cached rendered HTML for a post if available.
// Returns empty string if not cached or cache is stale.
func (c *Cache) GetCachedArticleHTML(sourcePath, contentHash string) string {
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	c.mu.RUnlock()

	if !ok || cached.ContentHash != contentHash || cached.ArticleHTMLPath == "" {
		return ""
	}

	// Read cached HTML file
	data, err := os.ReadFile(cached.ArticleHTMLPath)
	if err != nil {
		return ""
	}

	return string(data)
}

// CacheArticleHTML stores rendered HTML for a post.
func (c *Cache) CacheArticleHTML(sourcePath, contentHash, articleHTML string) error {
	// Get cache directory from path
	cacheDir := filepath.Dir(c.path)
	htmlCacheDir := filepath.Join(cacheDir, HTMLCacheDir)

	// Ensure html-cache directory exists
	if err := os.MkdirAll(htmlCacheDir, 0o755); err != nil {
		return fmt.Errorf("creating html cache dir: %w", err)
	}

	// Use content hash as filename (first 16 chars for shorter paths)
	hashPrefix := contentHash
	if len(hashPrefix) > 16 {
		hashPrefix = hashPrefix[:16]
	}
	htmlPath := filepath.Join(htmlCacheDir, hashPrefix+".html")

	// Write HTML to cache file
	//nolint:gosec // G306: cache files need 0644 for reading by other processes
	if err := os.WriteFile(htmlPath, []byte(articleHTML), 0o644); err != nil {
		return fmt.Errorf("writing html cache: %w", err)
	}

	// Update cache metadata
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.ContentHash = contentHash
		cached.ArticleHTMLPath = htmlPath
	} else {
		c.Posts[sourcePath] = &PostCache{
			ContentHash:     contentHash,
			ArticleHTMLPath: htmlPath,
		}
	}
	c.dirty = true

	return nil
}

// ContentHash computes a hash of just the markdown content.
func ContentHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// GetFeedHash returns the cached hash for a feed, or empty string if not cached.
func (c *Cache) GetFeedHash(slug string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.Feeds[slug]; ok {
		return cached.Hash
	}
	return ""
}

// SetFeedHash stores the hash for a feed in the cache.
func (c *Cache) SetFeedHash(slug, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Feeds[slug] = &FeedCache{Hash: hash}
	c.dirty = true
}

// FullHTMLCacheDir is the subdirectory for cached full page HTML files.
const FullHTMLCacheDir = "fullhtml-cache"

// GetCachedFullHTML returns the cached full page HTML for a post if available.
func (c *Cache) GetCachedFullHTML(sourcePath string) string {
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	c.mu.RUnlock()

	if !ok || cached.FullHTMLPath == "" {
		return ""
	}

	data, err := os.ReadFile(cached.FullHTMLPath)
	if err != nil {
		return ""
	}

	return string(data)
}

// CacheFullHTML stores the full page HTML for a post.
func (c *Cache) CacheFullHTML(sourcePath, fullHTML string) error {
	cacheDir := filepath.Dir(c.path)
	htmlCacheDir := filepath.Join(cacheDir, FullHTMLCacheDir)

	if err := os.MkdirAll(htmlCacheDir, 0o755); err != nil {
		return fmt.Errorf("creating fullhtml cache dir: %w", err)
	}

	// Use input hash as filename
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	var hashPrefix string
	if ok && cached.InputHash != "" {
		hashPrefix = cached.InputHash
		if len(hashPrefix) > 16 {
			hashPrefix = hashPrefix[:16]
		}
	} else {
		// Generate hash from path
		h := sha256.New()
		h.Write([]byte(sourcePath))
		hashPrefix = hex.EncodeToString(h.Sum(nil))[:16]
	}
	c.mu.RUnlock()

	htmlPath := filepath.Join(htmlCacheDir, hashPrefix+".html")

	//nolint:gosec // G306: cache files need 0644 for reading by other processes
	if err := os.WriteFile(htmlPath, []byte(fullHTML), 0o644); err != nil {
		return fmt.Errorf("writing fullhtml cache: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.FullHTMLPath = htmlPath
	} else {
		c.Posts[sourcePath] = &PostCache{
			FullHTMLPath: htmlPath,
		}
	}
	c.dirty = true

	return nil
}

// PostCacheDir is the subdirectory for cached parsed post JSON files.
const PostCacheDir = "post-cache"

// CachedPostData holds the serializable parts of a Post for caching.
// This excludes rendered HTML which is cached separately.
type CachedPostData struct {
	Path           string            `json:"path"`
	Content        string            `json:"content"`
	Slug           string            `json:"slug"`
	Href           string            `json:"href"`
	Title          *string           `json:"title,omitempty"`
	Date           *time.Time        `json:"date,omitempty"`
	Published      bool              `json:"published"`
	Draft          bool              `json:"draft"`
	Private        bool              `json:"private"`
	Skip           bool              `json:"skip"`
	Tags           []string          `json:"tags,omitempty"`
	Description    *string           `json:"description,omitempty"`
	Template       string            `json:"template"`
	Templates      map[string]string `json:"templates,omitempty"`
	RawFrontmatter string            `json:"raw_frontmatter"`
	InputHash      string            `json:"input_hash"`
	Extra          map[string]any    `json:"extra,omitempty"`
}

// GetCachedPostData returns cached post data if ModTime matches.
// Returns nil if post is not cached or file has changed.
func (c *Cache) GetCachedPostData(sourcePath string, modTime int64) *CachedPostData {
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	c.mu.RUnlock()

	if !ok || cached.ModTime != modTime || cached.ModTime == 0 {
		return nil
	}

	// Try to load from disk cache
	cacheDir := filepath.Dir(c.path)
	postCacheDir := filepath.Join(cacheDir, PostCacheDir)

	// Use path hash as filename
	h := sha256.Sum256([]byte(sourcePath))
	hashPrefix := hex.EncodeToString(h[:])[:16]
	postPath := filepath.Join(postCacheDir, hashPrefix+".json")

	data, err := os.ReadFile(postPath)
	if err != nil {
		return nil
	}

	var postData CachedPostData
	if err := json.Unmarshal(data, &postData); err != nil {
		return nil
	}

	return &postData
}

// CachePostData stores parsed post data to disk.
func (c *Cache) CachePostData(sourcePath string, modTime int64, postData *CachedPostData) error {
	cacheDir := filepath.Dir(c.path)
	postCacheDir := filepath.Join(cacheDir, PostCacheDir)

	if err := os.MkdirAll(postCacheDir, 0o755); err != nil {
		return fmt.Errorf("creating post cache dir: %w", err)
	}

	h := sha256.Sum256([]byte(sourcePath))
	hashPrefix := hex.EncodeToString(h[:])[:16]
	postPath := filepath.Join(postCacheDir, hashPrefix+".json")

	data, err := json.Marshal(postData)
	if err != nil {
		return fmt.Errorf("marshaling post data: %w", err)
	}

	//nolint:gosec // G306: cache files need 0644 for reading
	if err := os.WriteFile(postPath, data, 0o644); err != nil {
		return fmt.Errorf("writing post cache: %w", err)
	}

	// Update ModTime in cache
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.ModTime = modTime
		cached.Slug = postData.Slug
	} else {
		c.Posts[sourcePath] = &PostCache{
			ModTime: modTime,
			Slug:    postData.Slug,
		}
	}
	c.dirty = true

	return nil
}
