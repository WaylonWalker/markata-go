package lsp

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
	"gopkg.in/yaml.v3"
)

// Index maintains an index of all markdown posts in the workspace.
type Index struct {
	logger *log.Logger

	// posts maps slug to PostInfo for quick lookup
	posts map[string]*PostInfo
	mu    sync.RWMutex

	// uriToSlug maps file URI to slug for reverse lookup
	uriToSlug map[string]string

	// mentions maps handle/alias to MentionInfo for quick lookup
	mentions  map[string]*MentionInfo
	mentionMu sync.RWMutex
}

// PostInfo contains indexed information about a post.
type PostInfo struct {
	// URI is the file URI (file:///path/to/file.md)
	URI string

	// Path is the file system path
	Path string

	// Slug is the URL-safe identifier
	Slug string

	// Title is the post title (from frontmatter or filename)
	Title string

	// Description is the post description/excerpt
	Description string

	// Wikilinks contains all wikilinks found in the post
	Wikilinks []WikilinkInfo
}

// MentionInfo contains indexed information about a blogroll mention.
type MentionInfo struct {
	// Handle is the primary handle (e.g., "daverupert")
	Handle string

	// Aliases are alternative handles that resolve to this mention
	Aliases []string

	// Title is the display name (e.g., "Dave Rupert")
	Title string

	// SiteURL is the website URL
	SiteURL string

	// FeedURL is the RSS/Atom feed URL
	FeedURL string

	// Description is a short description
	Description string
}

// WikilinkInfo contains information about a wikilink in a post.
type WikilinkInfo struct {
	// Target is the slug being linked to
	Target string

	// DisplayText is the optional display text
	DisplayText string

	// Line is the 0-based line number
	Line int

	// StartChar is the 0-based character position of [[
	StartChar int

	// EndChar is the 0-based character position after ]]
	EndChar int
}

// NewIndex creates a new post index.
func NewIndex(logger *log.Logger) *Index {
	return &Index{
		logger:    logger,
		posts:     make(map[string]*PostInfo),
		uriToSlug: make(map[string]string),
		mentions:  make(map[string]*MentionInfo),
	}
}

// Build indexes all markdown files in the workspace.
func (idx *Index) Build(rootPath string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear existing index
	idx.posts = make(map[string]*PostInfo)
	idx.uriToSlug = make(map[string]string)

	// Index blogroll mentions from config
	idx.indexBlogrollMentions(rootPath)

	// Walk the directory tree
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories and non-markdown files
		if info.IsDir() {
			// Skip common non-content directories
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "output" || name == ".markata" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Index this file
		if err := idx.indexFile(path); err != nil {
			idx.logger.Printf("Failed to index %s: %v", path, err)
		}

		return nil
	})
}

// indexFile indexes a single markdown file.
func (idx *Index) indexFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return idx.indexContent(path, string(content))
}

// indexContent indexes content for a given path.
func (idx *Index) indexContent(path, content string) error {
	// Parse frontmatter
	metadata, body, err := plugins.ParseFrontmatter(content)
	if err != nil {
		// Still index the file, just without metadata
		metadata = make(map[string]interface{})
		body = content
	}

	// Generate slug
	slug := generateSlug(path, metadata)

	// Get title
	title := plugins.GetString(metadata, "title")
	if title == "" {
		// Derive from filename
		base := filepath.Base(path)
		title = strings.TrimSuffix(base, filepath.Ext(base))
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")
	}

	// Get description
	description := plugins.GetString(metadata, "description")
	if description == "" {
		// Extract first paragraph as description
		description = extractExcerpt(body, 200)
	}

	// Find wikilinks
	wikilinks := findWikilinks(body)

	// Create post info
	uri := pathToURI(path)
	info := &PostInfo{
		URI:         uri,
		Path:        path,
		Slug:        slug,
		Title:       title,
		Description: description,
		Wikilinks:   wikilinks,
	}

	// Store in index
	idx.posts[slug] = info
	idx.uriToSlug[uri] = slug

	return nil
}

// Update updates the index for a single file.
func (idx *Index) Update(uri, content string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	path := uriToPath(uri)

	// Remove old entry if exists
	if oldSlug, ok := idx.uriToSlug[uri]; ok {
		delete(idx.posts, oldSlug)
		delete(idx.uriToSlug, uri)
	}

	return idx.indexContent(path, content)
}

// Remove removes a file from the index.
func (idx *Index) Remove(uri string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if slug, ok := idx.uriToSlug[uri]; ok {
		delete(idx.posts, slug)
		delete(idx.uriToSlug, uri)
	}
}

// GetBySlug returns post info for a slug.
func (idx *Index) GetBySlug(slug string) *PostInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Try exact match first
	if info, ok := idx.posts[slug]; ok {
		return info
	}

	// Try case-insensitive match
	normalizedSlug := normalizeSlug(slug)
	for postSlug, info := range idx.posts {
		if normalizeSlug(postSlug) == normalizedSlug {
			return info
		}
	}

	return nil
}

// GetByURI returns post info for a URI.
func (idx *Index) GetByURI(uri string) *PostInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if slug, ok := idx.uriToSlug[uri]; ok {
		return idx.posts[slug]
	}
	return nil
}

// AllPosts returns all indexed posts.
func (idx *Index) AllPosts() []*PostInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	posts := make([]*PostInfo, 0, len(idx.posts))
	for _, info := range idx.posts {
		posts = append(posts, info)
	}
	return posts
}

// SearchPosts returns posts matching a prefix.
func (idx *Index) SearchPosts(prefix string) []*PostInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	prefix = strings.ToLower(prefix)
	var results []*PostInfo

	for slug, info := range idx.posts {
		// Match against slug or title
		if strings.HasPrefix(strings.ToLower(slug), prefix) ||
			strings.Contains(strings.ToLower(info.Title), prefix) {
			results = append(results, info)
		}
	}

	return results
}

// AllMentions returns all indexed mentions (unique by handle).
func (idx *Index) AllMentions() []*MentionInfo {
	idx.mentionMu.RLock()
	defer idx.mentionMu.RUnlock()

	// Deduplicate by handle (aliases point to same MentionInfo)
	seen := make(map[string]bool)
	mentions := make([]*MentionInfo, 0)
	for _, info := range idx.mentions {
		if !seen[info.Handle] {
			seen[info.Handle] = true
			mentions = append(mentions, info)
		}
	}
	return mentions
}

// SearchMentions returns mentions matching a prefix.
func (idx *Index) SearchMentions(prefix string) []*MentionInfo {
	idx.mentionMu.RLock()
	defer idx.mentionMu.RUnlock()

	prefix = strings.ToLower(prefix)

	// Deduplicate results by handle
	seen := make(map[string]bool)
	var results []*MentionInfo

	for key, info := range idx.mentions {
		if seen[info.Handle] {
			continue
		}
		// Match against handle, aliases, or title
		if strings.HasPrefix(strings.ToLower(key), prefix) ||
			strings.Contains(strings.ToLower(info.Title), prefix) {
			seen[info.Handle] = true
			results = append(results, info)
		}
	}

	return results
}

// GetByHandle returns a mention by handle or alias (case-insensitive).
func (idx *Index) GetByHandle(handle string) *MentionInfo {
	idx.mentionMu.RLock()
	defer idx.mentionMu.RUnlock()

	handle = strings.ToLower(handle)
	return idx.mentions[handle]
}

// indexBlogrollMentions reads the markata config and indexes blogroll feeds as mentions.
func (idx *Index) indexBlogrollMentions(rootPath string) {
	idx.mentionMu.Lock()
	defer idx.mentionMu.Unlock()

	idx.mentions = make(map[string]*MentionInfo)

	// Try to find and parse config file
	configPaths := []string{
		filepath.Join(rootPath, "markata-go.toml"),
		filepath.Join(rootPath, "markata.toml"),
		filepath.Join(rootPath, "markata-go.yaml"),
		filepath.Join(rootPath, "markata.yaml"),
	}

	for _, configPath := range configPaths {
		content, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		var feeds []blogrollFeedConfig
		if strings.HasSuffix(configPath, ".toml") {
			feeds = idx.parseBlogrollFromTOML(content)
		} else {
			feeds = idx.parseBlogrollFromYAML(content)
		}

		for _, feed := range feeds {
			idx.indexFeedAsMention(&feed)
		}

		if len(feeds) > 0 {
			idx.logger.Printf("Indexed %d blogroll mentions from %s", len(idx.mentions), configPath)
		}
		return // Only use first config found
	}
}

// blogrollFeedConfig is a minimal struct for parsing feed configs.
type blogrollFeedConfig struct {
	URL         string   `toml:"url" yaml:"url"`
	Title       string   `toml:"title" yaml:"title"`
	Description string   `toml:"description" yaml:"description"`
	SiteURL     string   `toml:"site_url" yaml:"site_url"`
	Handle      string   `toml:"handle" yaml:"handle"`
	Aliases     []string `toml:"aliases" yaml:"aliases"`
	Active      *bool    `toml:"active" yaml:"active"`
}

// parseBlogrollFromTOML extracts feed configs from TOML content.
func (idx *Index) parseBlogrollFromTOML(content []byte) []blogrollFeedConfig {
	// Try markata-go.blogroll first (new format)
	var configNew struct {
		MarktaGo struct {
			Blogroll struct {
				Enabled bool                 `toml:"enabled"`
				Feeds   []blogrollFeedConfig `toml:"feeds"`
			} `toml:"blogroll"`
		} `toml:"markata-go"`
	}

	if err := toml.Unmarshal(content, &configNew); err == nil {
		if configNew.MarktaGo.Blogroll.Enabled && len(configNew.MarktaGo.Blogroll.Feeds) > 0 {
			return configNew.MarktaGo.Blogroll.Feeds
		}
	}

	// Fall back to blogroll (old format)
	var configOld struct {
		Blogroll struct {
			Enabled bool                 `toml:"enabled"`
			Feeds   []blogrollFeedConfig `toml:"feeds"`
		} `toml:"blogroll"`
	}

	if err := toml.Unmarshal(content, &configOld); err != nil {
		idx.logger.Printf("Failed to parse TOML config: %v", err)
		return nil
	}

	if !configOld.Blogroll.Enabled {
		return nil
	}

	return configOld.Blogroll.Feeds
}

// parseBlogrollFromYAML extracts feed configs from YAML content.
func (idx *Index) parseBlogrollFromYAML(content []byte) []blogrollFeedConfig {
	var config struct {
		Blogroll struct {
			Enabled bool                 `yaml:"enabled"`
			Feeds   []blogrollFeedConfig `yaml:"feeds"`
		} `yaml:"blogroll"`
	}

	if err := yaml.Unmarshal(content, &config); err != nil {
		idx.logger.Printf("Failed to parse YAML config: %v", err)
		return nil
	}

	if !config.Blogroll.Enabled {
		return nil
	}

	return config.Blogroll.Feeds
}

// indexFeedAsMention creates a MentionInfo from a feed config.
func (idx *Index) indexFeedAsMention(feed *blogrollFeedConfig) {
	// Skip inactive feeds
	if feed.Active != nil && !*feed.Active {
		return
	}

	// Determine handle
	handle := feed.Handle
	if handle == "" {
		handle = extractHandleFromFeedURL(feed.SiteURL)
		if handle == "" {
			handle = extractHandleFromFeedURL(feed.URL)
		}
	}
	if handle == "" {
		return
	}

	handle = strings.ToLower(handle)

	// Create mention info
	info := &MentionInfo{
		Handle:      handle,
		Aliases:     feed.Aliases,
		Title:       feed.Title,
		SiteURL:     feed.SiteURL,
		FeedURL:     feed.URL,
		Description: feed.Description,
	}

	// Index by handle
	if _, exists := idx.mentions[handle]; !exists {
		idx.mentions[handle] = info
	}

	// Index by aliases
	for _, alias := range feed.Aliases {
		alias = strings.ToLower(alias)
		if alias != "" {
			if _, exists := idx.mentions[alias]; !exists {
				idx.mentions[alias] = info
			}
		}
	}

	// Auto-register domain alias
	if feed.SiteURL != "" {
		domain := extractDomainAlias(feed.SiteURL)
		if domain != "" && domain != handle {
			if _, exists := idx.mentions[domain]; !exists {
				idx.mentions[domain] = info
			}
		}
	}
}

// extractHandleFromFeedURL extracts a handle from a URL's domain.
func extractHandleFromFeedURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	// Simple extraction - find the domain part
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "www.")
	urlStr = strings.TrimPrefix(urlStr, "blog.")

	// Get first part before /
	if idx := strings.Index(urlStr, "/"); idx > 0 {
		urlStr = urlStr[:idx]
	}

	// Get first part before .
	parts := strings.Split(urlStr, ".")
	if len(parts) > 0 {
		return strings.ToLower(parts[0])
	}

	return ""
}

// extractDomainAlias extracts the full domain as an alias.
func extractDomainAlias(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Get domain before /
	if idx := strings.Index(urlStr, "/"); idx > 0 {
		urlStr = urlStr[:idx]
	}

	return strings.ToLower(urlStr)
}

// generateSlug generates a slug from path and metadata.
func generateSlug(path string, metadata map[string]interface{}) string {
	// Check for slug in metadata
	if slug := plugins.GetString(metadata, "slug"); slug != "" {
		return slug
	}

	// Generate from filename
	base := filepath.Base(path)

	// Handle index.md special case
	if strings.EqualFold(base, "index.md") {
		dir := filepath.Dir(path)
		dir = filepath.Clean(dir)
		if dir == "." {
			return "" // Root index.md becomes homepage
		}
		slug := strings.ToLower(dir)
		slug = strings.ReplaceAll(slug, string(filepath.Separator), "/")
		slug = strings.TrimPrefix(slug, "./")
		return slug
	}

	// Use filename without known extension (consistent with models.Post)
	basename := models.StripKnownExtension(base)
	return models.Slugify(basename)
}

// wikilinkRegex matches [[slug]] and [[slug|display text]] patterns.
var wikilinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// findWikilinks finds all wikilinks in content.
func findWikilinks(content string) []WikilinkInfo {
	var results []WikilinkInfo
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		matches := wikilinkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}

			// match[0:2] is the full match position
			// match[2:4] is group 1 (slug)
			// match[4:6] is group 2 (display text) if present
			fullMatch := line[match[0]:match[1]]
			groups := wikilinkRegex.FindStringSubmatch(fullMatch)

			target := strings.TrimSpace(groups[1])
			displayText := ""
			if len(groups) > 2 && groups[2] != "" {
				displayText = strings.TrimSpace(groups[2])
			}

			results = append(results, WikilinkInfo{
				Target:      target,
				DisplayText: displayText,
				Line:        lineNum,
				StartChar:   match[0],
				EndChar:     match[1],
			})
		}
	}

	return results
}

// extractExcerpt extracts the first paragraph as an excerpt.
func extractExcerpt(content string, maxLen int) string {
	// Remove leading whitespace and headers
	content = strings.TrimSpace(content)

	// Skip leading headers
	lines := strings.Split(content, "\n")
	textLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			if len(textLines) > 0 {
				break // End of first paragraph
			}
			continue
		}
		textLines = append(textLines, line)
	}

	if len(textLines) == 0 {
		return ""
	}

	excerpt := strings.Join(textLines, " ")
	if len(excerpt) > maxLen {
		excerpt = excerpt[:maxLen-3] + "..."
	}

	return excerpt
}

// normalizeSlug normalizes a slug for comparison.
func normalizeSlug(slug string) string {
	return models.Slugify(slug)
}

// pathToURI converts a file system path to a file URI.
func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	// Normalize separators for URI
	absPath = filepath.ToSlash(absPath)
	return "file://" + absPath
}

// uriToPath converts a file URI to a file system path.
func uriToPath(uri string) string {
	path := strings.TrimPrefix(uri, "file://")
	// Handle Windows paths
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}
