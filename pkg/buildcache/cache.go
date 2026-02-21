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
	"sync/atomic"
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

	// TagsListingHash caches the tags listing output state
	TagsListingHash string `json:"tags_listing_hash,omitempty"`

	// GardenHash caches the garden view output state
	GardenHash string `json:"garden_hash,omitempty"`

	// Graph tracks dependencies between posts for transitive invalidation
	Graph *DependencyGraph `json:"graph,omitempty"`

	// path to the cache file (not serialized)
	path string

	// dirty tracks whether cache needs saving
	dirty bool

	// skippedCount tracks how many posts were skipped this build (lock-free)
	skippedCount atomic.Int64

	// rebuiltCount tracks how many posts were rebuilt this build (lock-free)
	rebuiltCount atomic.Int64

	// changedSlugs tracks slugs that changed this build (for dependency invalidation)
	changedSlugs map[string]bool

	// changedFeedSlugs tracks slugs with feed-relevant changes this build
	changedFeedSlugs map[string]bool

	// tagsDirty marks whether tag outputs should rebuild this build
	tagsDirty bool

	// gardenDirty marks whether garden outputs should rebuild this build
	gardenDirty bool

	// GlobFiles caches the list of discovered files from glob stage
	GlobFiles []string `json:"glob_files,omitempty"`

	// GlobPatternHash detects when glob patterns change
	GlobPatternHash string `json:"glob_pattern_hash,omitempty"`

	// In-memory caches to avoid repeated per-post disk reads during hot builds.
	// These are populated lazily on first access (backfilled from disk on cache
	// hit) and updated by CacheFullHTML/CacheArticleHTML/CachePostData on writes.
	// Using sync.Map for lock-free concurrent reads from worker pools.

	// fullHTMLMemory maps FullHTMLPath -> HTML string
	fullHTMLMemory sync.Map
	// articleHTMLMemory maps ArticleHTMLPath -> HTML string
	articleHTMLMemory sync.Map
	// postDataMemory maps post cache file path -> *CachedPostData
	postDataMemory sync.Map
	// encryptedHTMLMemory maps EncryptedHTMLPath -> HTML string
	encryptedHTMLMemory sync.Map
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

	// LinkAvatarsHash is a hash of the ArticleHTML input used for link_avatars caching.
	// When ArticleHTML changes, this hash changes and the post is re-processed.
	LinkAvatarsHash string `json:"link_avatars_hash,omitempty"`

	// LinkAvatarsHTML is the cached ArticleHTML output after link_avatars processing.
	LinkAvatarsHTML string `json:"link_avatars_html,omitempty"`

	// EmbedsHash is a hash of the post Content input used for embeds transform caching.
	// When Content changes, this hash changes and the post is re-processed.
	EmbedsHash string `json:"embeds_hash,omitempty"`

	// EmbedsContent is the cached Content output after embeds transform processing.
	EmbedsContent string `json:"embeds_content,omitempty"`

	// GlossaryHash is a hash of the ArticleHTML input + glossary terms state used for caching.
	GlossaryHash string `json:"glossary_hash,omitempty"`

	// GlossaryHTML is the cached ArticleHTML output after glossary processing.
	GlossaryHTML string `json:"glossary_html,omitempty"`

	// EncryptedHash is a hash of inputs for encrypted HTML caching.
	EncryptedHash string `json:"encrypted_hash,omitempty"`

	// EncryptedHTMLPath is the path to cached encrypted HTML wrapper.
	EncryptedHTMLPath string `json:"encrypted_html_path,omitempty"`

	// FeedItemHash is a hash of fields that affect feed output for this post.
	FeedItemHash string `json:"feed_item_hash,omitempty"`

	// TagIndexHash is a hash of fields that affect tag listings for this post.
	TagIndexHash string `json:"tag_index_hash,omitempty"`

	// GardenHash is a hash of fields that affect garden view output for this post.
	GardenHash string `json:"garden_hash,omitempty"`
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
		Version:          CacheVersion,
		Posts:            make(map[string]*PostCache),
		Feeds:            make(map[string]*FeedCache),
		Graph:            NewDependencyGraph(),
		path:             filepath.Join(cacheDir, CacheFileName),
		changedSlugs:     make(map[string]bool),
		changedFeedSlugs: make(map[string]bool),
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
	cache.changedFeedSlugs = make(map[string]bool)

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

	return c.shouldRebuildLocked(sourcePath, inputHash, template)
}

// shouldRebuildLocked is the lock-free inner implementation of ShouldRebuild.
// Caller must hold at least c.mu.RLock().
func (c *Cache) shouldRebuildLocked(sourcePath, inputHash, template string) bool {
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

// ShouldRebuildBatch checks multiple posts against the cache in a single lock acquisition.
// Returns a map of sourcePath -> true for posts that need rebuilding.
// This is more efficient than calling ShouldRebuild in a loop.
func (c *Cache) ShouldRebuildBatch(posts []struct {
	Path, InputHash, Template string
}) map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]bool, len(posts)/10) // expect ~10% changed
	for _, p := range posts {
		if c.shouldRebuildLocked(p.Path, p.InputHash, p.Template) {
			result[p.Path] = true
		}
	}
	return result
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
	c.rebuiltCount.Add(1)
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
	c.rebuiltCount.Add(1)

	// Track that this slug changed (for dependency invalidation)
	if slug != "" {
		c.changedSlugs[slug] = true
	}
}

// UpdatePostSemanticHashes updates cached per-post hashes for feed/tags/garden.
// Returns which hashes changed compared to cache.
func (c *Cache) UpdatePostSemanticHashes(sourcePath, feedHash, tagHash, gardenHash string) (feedChanged, tagChanged, gardenChanged bool) {
	if sourcePath == "" {
		return false, false, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	cacheEntry, ok := c.Posts[sourcePath]
	if !ok {
		cacheEntry = &PostCache{}
		c.Posts[sourcePath] = cacheEntry
	}

	if cacheEntry.FeedItemHash != feedHash {
		feedChanged = true
		cacheEntry.FeedItemHash = feedHash
	}
	if cacheEntry.TagIndexHash != tagHash {
		tagChanged = true
		cacheEntry.TagIndexHash = tagHash
	}
	if cacheEntry.GardenHash != gardenHash {
		gardenChanged = true
		cacheEntry.GardenHash = gardenHash
	}
	if tagChanged {
		c.tagsDirty = true
	}
	if gardenChanged {
		c.gardenDirty = true
	}

	if feedChanged || tagChanged || gardenChanged {
		c.dirty = true
	}

	return feedChanged, tagChanged, gardenChanged
}

// GetPostSemanticHashes returns the cached feed/tag/garden hashes for a post.
func (c *Cache) GetPostSemanticHashes(sourcePath string) (feedHash, tagHash, gardenHash string) {
	if sourcePath == "" {
		return "", "", ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.Posts[sourcePath]; ok {
		return entry.FeedItemHash, entry.TagIndexHash, entry.GardenHash
	}
	return "", "", ""
}

// TagsDirty reports whether any tag-relevant fields changed this build.
func (c *Cache) TagsDirty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tagsDirty
}

// GardenDirty reports whether any garden-relevant fields changed this build.
func (c *Cache) GardenDirty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gardenDirty
}

// SetTagsListingHash stores the tags listing hash in the cache.
func (c *Cache) SetTagsListingHash(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.TagsListingHash == hash {
		return
	}
	c.TagsListingHash = hash
	c.dirty = true
}

// GetTagsListingHash returns the cached tags listing hash.
func (c *Cache) GetTagsListingHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TagsListingHash
}

// SetGardenHash stores the garden view hash in the cache.
func (c *Cache) SetGardenHash(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.GardenHash == hash {
		return
	}
	c.GardenHash = hash
	c.dirty = true
}

// GetGardenHash returns the cached garden view hash.
func (c *Cache) GetGardenHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GardenHash
}

// MarkSkipped records that a post was skipped (already up to date).
// Uses atomic increment -- no mutex needed.
func (c *Cache) MarkSkipped() {
	c.skippedCount.Add(1)
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

// GetCachedLinkAvatarsHTML returns cached link_avatars output if the article hash matches.
// Returns the cached HTML and true if valid, empty string and false otherwise.
func (c *Cache) GetCachedLinkAvatarsHTML(sourcePath, articleHash string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return "", false
	}
	if cached.LinkAvatarsHash == "" || cached.LinkAvatarsHash != articleHash {
		return "", false
	}
	return cached.LinkAvatarsHTML, true
}

// CacheLinkAvatarsHTML stores the link_avatars processed HTML keyed by article hash.
func (c *Cache) CacheLinkAvatarsHTML(sourcePath, articleHash, html string) {
	if sourcePath == "" || articleHash == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.LinkAvatarsHash = articleHash
		cached.LinkAvatarsHTML = html
	} else {
		c.Posts[sourcePath] = &PostCache{
			LinkAvatarsHash: articleHash,
			LinkAvatarsHTML: html,
		}
	}
	c.dirty = true
}

// GetCachedEmbedsContent returns cached embeds transform output if the content hash matches.
// Returns the cached content and true if valid, empty string and false otherwise.
func (c *Cache) GetCachedEmbedsContent(sourcePath, contentHash string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return "", false
	}
	if cached.EmbedsHash == "" || cached.EmbedsHash != contentHash {
		return "", false
	}
	return cached.EmbedsContent, true
}

// CacheEmbedsContent stores the embeds transform output keyed by content hash.
func (c *Cache) CacheEmbedsContent(sourcePath, contentHash, content string) {
	if sourcePath == "" || contentHash == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.EmbedsHash = contentHash
		cached.EmbedsContent = content
	} else {
		c.Posts[sourcePath] = &PostCache{
			EmbedsHash:    contentHash,
			EmbedsContent: content,
		}
	}
	c.dirty = true
}

// GetCachedGlossaryHTML returns cached glossary output if the combined hash matches.
// Returns the cached HTML and true if valid, empty string and false otherwise.
func (c *Cache) GetCachedGlossaryHTML(sourcePath, combinedHash string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.Posts[sourcePath]
	if !ok {
		return "", false
	}
	if cached.GlossaryHash == "" || cached.GlossaryHash != combinedHash {
		return "", false
	}
	return cached.GlossaryHTML, true
}

// CacheGlossaryHTML stores the glossary processed HTML keyed by combined hash.
func (c *Cache) CacheGlossaryHTML(sourcePath, combinedHash, html string) {
	if sourcePath == "" || combinedHash == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.Posts[sourcePath]; ok {
		cached.GlossaryHash = combinedHash
		cached.GlossaryHTML = html
	} else {
		c.Posts[sourcePath] = &PostCache{
			GlossaryHash: combinedHash,
			GlossaryHTML: html,
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
	return int(c.skippedCount.Load()), int(c.rebuiltCount.Load())
}

// CacheStats holds build statistics.
type CacheStats struct {
	Skipped int
	Rebuilt int
}

// GetStats returns build statistics as a struct.
func (c *Cache) GetStats() CacheStats {
	return CacheStats{
		Skipped: int(c.skippedCount.Load()),
		Rebuilt: int(c.rebuiltCount.Load()),
	}
}

// ResetStats resets the build statistics for a new build.
func (c *Cache) ResetStats() {
	c.skippedCount.Store(0)
	c.rebuiltCount.Store(0)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changedSlugs = make(map[string]bool)
	c.changedFeedSlugs = make(map[string]bool)
	c.tagsDirty = false
	c.gardenDirty = false
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
			c.Graph.RemoveSource(path)
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

// GetChangedFeedSlugs returns slugs that changed in feed-relevant ways this build.
func (c *Cache) GetChangedFeedSlugs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.changedFeedSlugs) == 0 {
		return nil
	}
	result := make([]string, 0, len(c.changedFeedSlugs))
	for slug := range c.changedFeedSlugs {
		result = append(result, slug)
	}
	sort.Strings(result)
	return result
}

// ClearChangedFeedSlugs resets the feed change tracking for the current build.
func (c *Cache) ClearChangedFeedSlugs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changedFeedSlugs = make(map[string]bool)
}

// MarkSlugChanged records that a slug changed this build.
// Used for dependency invalidation.
func (c *Cache) MarkSlugChanged(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changedSlugs[slug] = true
}

// MarkFeedSlugChanged records that a slug changed in feed-relevant ways.
func (c *Cache) MarkFeedSlugChanged(slug string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changedFeedSlugs[slug] = true
}

// SetPostSlug records the slug for a source path in the dependency graph.
func (c *Cache) SetPostSlug(sourcePath, slug string) {
	if sourcePath == "" || slug == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Graph.PathToSlug[sourcePath] = slug
}

// MarkChangedPaths records slugs for the provided source paths if known in the cache.
// This is used to seed dependency invalidation from filesystem change events.
func (c *Cache) MarkChangedPaths(paths []string) {
	if len(paths) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range paths {
		if path == "" {
			continue
		}
		if slug := c.Graph.PathToSlug[path]; slug != "" {
			c.changedSlugs[slug] = true
		}
	}
}

// MarkAffectedDependents records all dependents (transitive) of changed slugs as changed.
func (c *Cache) MarkAffectedDependents(changedSlugs []string) {
	if len(changedSlugs) == 0 {
		return
	}

	c.mu.RLock()
	affectedPaths := c.Graph.GetAffectedPosts(changedSlugs)
	c.mu.RUnlock()
	if len(affectedPaths) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range affectedPaths {
		if slug := c.Graph.PathToSlug[path]; slug != "" {
			c.changedSlugs[slug] = true
		}
	}
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

// EncryptedHTMLCacheDir is the subdirectory for cached encrypted HTML files.
const EncryptedHTMLCacheDir = "encrypted-html-cache"

// GetCachedArticleHTML returns the cached rendered HTML for a post if available.
// Returns empty string if not cached or cache is stale.
func (c *Cache) GetCachedArticleHTML(sourcePath, contentHash string) string {
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	c.mu.RUnlock()

	if !ok || cached.ContentHash != contentHash || cached.ArticleHTMLPath == "" {
		return ""
	}

	// Check in-memory cache first (populated by preloadCaches)
	if val, ok := c.articleHTMLMemory.Load(cached.ArticleHTMLPath); ok {
		if html, ok := val.(string); ok {
			return html
		}
	}

	// Fallback to disk read
	data, err := os.ReadFile(cached.ArticleHTMLPath)
	if err != nil {
		return ""
	}

	html := string(data)
	// Store in memory for future calls
	c.articleHTMLMemory.Store(cached.ArticleHTMLPath, html)
	return html
}

// CacheArticleHTML stores rendered HTML for a post.
func (c *Cache) CacheArticleHTML(sourcePath, contentHash, articleHTML string) error {
	htmlPath, err := c.cacheHTMLFile(HTMLCacheDir, contentHash, articleHTML, &c.articleHTMLMemory)
	if err != nil {
		return err
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

	// Check in-memory cache first (populated by preloadCaches)
	if val, ok := c.fullHTMLMemory.Load(cached.FullHTMLPath); ok {
		if html, ok := val.(string); ok {
			return html
		}
	}

	// Fallback to disk read (shouldn't happen in normal hot builds)
	data, err := os.ReadFile(cached.FullHTMLPath)
	if err != nil {
		return ""
	}

	html := string(data)
	// Store in memory for future calls
	c.fullHTMLMemory.Store(cached.FullHTMLPath, html)
	return html
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

	// Store in memory cache for future reads within this build
	c.fullHTMLMemory.Store(htmlPath, fullHTML)

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

// GetCachedEncryptedHTML returns cached encrypted HTML if hash matches.
func (c *Cache) GetCachedEncryptedHTML(sourcePath, encryptedHash string) string {
	c.mu.RLock()
	cached, ok := c.Posts[sourcePath]
	c.mu.RUnlock()
	if !ok || cached.EncryptedHash != encryptedHash || cached.EncryptedHTMLPath == "" {
		return ""
	}

	if val, ok := c.encryptedHTMLMemory.Load(cached.EncryptedHTMLPath); ok {
		if html, ok := val.(string); ok {
			return html
		}
	}

	data, err := os.ReadFile(cached.EncryptedHTMLPath)
	if err != nil {
		return ""
	}

	html := string(data)
	c.encryptedHTMLMemory.Store(cached.EncryptedHTMLPath, html)
	return html
}

// CacheEncryptedHTML stores encrypted HTML for a post.
func (c *Cache) CacheEncryptedHTML(sourcePath, encryptedHash, html string) error {
	htmlPath, err := c.cacheHTMLFile(EncryptedHTMLCacheDir, encryptedHash, html, &c.encryptedHTMLMemory)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.Posts[sourcePath]; ok {
		cached.EncryptedHash = encryptedHash
		cached.EncryptedHTMLPath = htmlPath
	} else {
		c.Posts[sourcePath] = &PostCache{
			EncryptedHash:     encryptedHash,
			EncryptedHTMLPath: htmlPath,
		}
	}
	c.dirty = true
	return nil
}

func (c *Cache) cacheHTMLFile(cacheDirName, hash, html string, memory *sync.Map) (string, error) {
	cacheDir := filepath.Dir(c.path)
	cacheSubDir := filepath.Join(cacheDir, cacheDirName)
	if err := os.MkdirAll(cacheSubDir, 0o755); err != nil {
		return "", fmt.Errorf("creating html cache dir: %w", err)
	}

	hashPrefix := hash
	if len(hashPrefix) > 16 {
		hashPrefix = hashPrefix[:16]
	}
	htmlPath := filepath.Join(cacheSubDir, hashPrefix+".html")

	//nolint:gosec // G306: cache files need 0644 for reading by other processes
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("writing html cache: %w", err)
	}

	if memory != nil {
		memory.Store(htmlPath, html)
	}

	return htmlPath, nil
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
	Modified       *time.Time        `json:"modified,omitempty"`
	Published      bool              `json:"published"`
	Draft          bool              `json:"draft"`
	Private        bool              `json:"private"`
	Skip           bool              `json:"skip"`
	SecretKey      string            `json:"secret_key,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Description    *string           `json:"description,omitempty"`
	Template       string            `json:"template"`
	Templates      map[string]string `json:"templates,omitempty"`
	RawFrontmatter string            `json:"raw_frontmatter"`
	InputHash      string            `json:"input_hash"`
	Authors        []string          `json:"authors,omitempty"`
	Author         *string           `json:"author,omitempty"`
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

	// Compute the post cache file path
	cacheDir := filepath.Dir(c.path)
	postCacheDir := filepath.Join(cacheDir, PostCacheDir)
	h := sha256.Sum256([]byte(sourcePath))
	hashPrefix := hex.EncodeToString(h[:])[:16]
	postPath := filepath.Join(postCacheDir, hashPrefix+".json")

	// Check in-memory cache first (populated by preloadCaches)
	if val, ok := c.postDataMemory.Load(postPath); ok {
		if pd, ok := val.(*CachedPostData); ok {
			return pd
		}
	}

	// Fallback to disk read
	data, err := os.ReadFile(postPath)
	if err != nil {
		return nil
	}

	var postData CachedPostData
	if err := json.Unmarshal(data, &postData); err != nil {
		return nil
	}

	// Store in memory for future calls
	c.postDataMemory.Store(postPath, &postData)
	return &postData
}

// GetCachedPostDataLatest returns cached post data without checking ModTime.
// Use only when external change detection is trusted.
func (c *Cache) GetCachedPostDataLatest(sourcePath string) *CachedPostData {
	if sourcePath == "" {
		return nil
	}

	// Compute the post cache file path
	cacheDir := filepath.Dir(c.path)
	postCacheDir := filepath.Join(cacheDir, PostCacheDir)
	h := sha256.Sum256([]byte(sourcePath))
	hashPrefix := hex.EncodeToString(h[:])[:16]
	postPath := filepath.Join(postCacheDir, hashPrefix+".json")

	// Check in-memory cache first
	if val, ok := c.postDataMemory.Load(postPath); ok {
		if pd, ok := val.(*CachedPostData); ok {
			return pd
		}
	}

	data, err := os.ReadFile(postPath)
	if err != nil {
		return nil
	}

	var postData CachedPostData
	if err := json.Unmarshal(data, &postData); err != nil {
		return nil
	}

	c.postDataMemory.Store(postPath, &postData)
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

	// Store in memory cache for future reads within this build
	c.postDataMemory.Store(postPath, postData)

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
