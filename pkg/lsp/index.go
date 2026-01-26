package lsp

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/plugins"
)

// Index maintains an index of all markdown posts in the workspace.
type Index struct {
	logger *log.Logger

	// posts maps slug to PostInfo for quick lookup
	posts map[string]*PostInfo
	mu    sync.RWMutex

	// uriToSlug maps file URI to slug for reverse lookup
	uriToSlug map[string]string
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
	}
}

// Build indexes all markdown files in the workspace.
func (idx *Index) Build(rootPath string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear existing index
	idx.posts = make(map[string]*PostInfo)
	idx.uriToSlug = make(map[string]string)

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

	// Use filename without extension
	slug := strings.TrimSuffix(base, filepath.Ext(base))
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = slugifyRegex.ReplaceAllString(slug, "")
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return slug
}

// slugifyRegex matches characters that are not alphanumeric, hyphens, or underscores
var slugifyRegex = regexp.MustCompile(`[^a-z0-9\-_]+`)

// multiHyphenRegex matches multiple consecutive hyphens
var multiHyphenRegex = regexp.MustCompile(`-+`)

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
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = slugifyRegex.ReplaceAllString(slug, "")
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
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
