package models

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Post represents a single markdown post with its metadata and content.
type Post struct {
	// Path is the source file path
	Path string `json:"path" yaml:"path" toml:"path"`

	// Content is the raw markdown content after frontmatter
	Content string `json:"content" yaml:"content" toml:"content"`

	// Slug is the URL-safe identifier
	Slug string `json:"slug" yaml:"slug" toml:"slug"`

	// Href is the relative URL path (e.g., /my-post/)
	Href string `json:"href" yaml:"href" toml:"href"`

	// Title is the optional post title
	Title *string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// Date is the optional publication date
	Date *time.Time `json:"date,omitempty" yaml:"date,omitempty" toml:"date,omitempty"`

	// Published indicates if the post is published
	Published bool `json:"published" yaml:"published" toml:"published"`

	// Draft indicates if the post is a draft
	Draft bool `json:"draft" yaml:"draft" toml:"draft"`

	// Private indicates if the post should be excluded from feeds and search
	// Private posts are rendered but excluded from feeds, sitemaps, and add noindex meta tag
	Private bool `json:"private" yaml:"private" toml:"private"`

	// SecretKey is the name of the encryption key to use for this post.
	// When set along with Private=true, the post content will be encrypted.
	// The actual key is read from environment variable MARKATA_GO_ENCRYPTION_KEY_{SecretKey}
	// Example: secret_key: "blog" reads from MARKATA_GO_ENCRYPTION_KEY_BLOG
	SecretKey string `json:"secret_key,omitempty" yaml:"secret_key,omitempty" toml:"secret_key,omitempty"`

	// Skip indicates if the post should be skipped during processing
	Skip bool `json:"skip" yaml:"skip" toml:"skip"`

	// Tags is a list of tags associated with the post
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty" toml:"tags,omitempty"`

	// Description is the optional post description
	Description *string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`

	// Template is the template file to use for rendering (default: "post.html")
	// Can be a preset name ("blog", "docs") or explicit file ("post.html")
	Template string `json:"template" yaml:"template" toml:"template"`

	// Templates provides per-format template overrides
	// Keys: "html", "txt", "markdown", "og"
	// Values: template file names
	Templates map[string]string `json:"templates,omitempty" yaml:"templates,omitempty" toml:"templates,omitempty"`

	// HTML is the final rendered HTML including template wrapper
	HTML string `json:"html" yaml:"html" toml:"html"`

	// ArticleHTML is the rendered content without template wrapper
	ArticleHTML string `json:"article_html" yaml:"article_html" toml:"article_html"`

	// InputHash is a hash of the post's inputs (content + frontmatter + template)
	// Used for incremental builds to detect changes
	InputHash string `json:"input_hash,omitempty" yaml:"input_hash,omitempty" toml:"input_hash,omitempty"`

	// RawFrontmatter stores the original frontmatter string for hash computation
	// Not serialized to output
	RawFrontmatter string `json:"-" yaml:"-" toml:"-"`

	// Prev is the previous post in the navigation sequence
	Prev *Post `json:"-" yaml:"-" toml:"-"`

	// Next is the next post in the navigation sequence
	Next *Post `json:"-" yaml:"-" toml:"-"`

	// PrevNextFeed is the feed/series slug used for navigation
	PrevNextFeed string `json:"prevnext_feed,omitempty" yaml:"prevnext_feed,omitempty" toml:"prevnext_feed,omitempty"`

	// PrevNextContext contains full navigation context
	PrevNextContext *PrevNextContext `json:"-" yaml:"-" toml:"-"`

	// Hrefs is a list of raw href values from all links in the post
	Hrefs []string `json:"hrefs,omitempty" yaml:"hrefs,omitempty" toml:"hrefs,omitempty"`

	// Inlinks are links pointing TO this post from other posts
	Inlinks []*Link `json:"inlinks,omitempty" yaml:"inlinks,omitempty" toml:"inlinks,omitempty"`

	// Outlinks are links FROM this post to other pages
	Outlinks []*Link `json:"outlinks,omitempty" yaml:"outlinks,omitempty" toml:"outlinks,omitempty"`

	// Dependencies tracks slugs this post depends on (wikilinks, embeds).
	// Used for incremental build cache invalidation.
	// Not persisted to output files.
	Dependencies []string `json:"-" yaml:"-" toml:"-"`

	// Author fields (backward compatible)
	Authors []string `json:"authors,omitempty" yaml:"authors,omitempty" toml:"authors,omitempty"`
	Author  *string  `json:"author,omitempty" yaml:"author,omitempty" toml:"author,omitempty"` // Backward compatibility

	// AuthorRoleOverrides stores per-post role overrides keyed by author ID.
	// Populated when frontmatter uses extended format: authors: [{id: waylon, role: editor}]
	// Not serialized; rebuilt from frontmatter on each load.
	AuthorRoleOverrides map[string]string `json:"-" yaml:"-" toml:"-"`

	// AuthorDetailsOverrides stores per-post details overrides keyed by author ID.
	// Populated when frontmatter uses extended format: authors: [{id: waylon, details: "wrote the intro"}]
	// Not serialized; rebuilt from frontmatter on each load.
	AuthorDetailsOverrides map[string]string `json:"-" yaml:"-" toml:"-"`

	// Extra holds dynamic/unknown fields from frontmatter
	Extra map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty" toml:"extra,omitempty"`

	// Computed fields (not in frontmatter)
	AuthorObjects []Author `json:"-" yaml:"-" toml:"-"`
}

// NewPost creates a new Post with the given source file path and default values.
func NewPost(path string) *Post {
	return &Post{
		Path:      path,
		Published: false,
		Draft:     false,
		Private:   false,
		Skip:      false,
		Tags:      []string{},
		Template:  "", // Empty - let templates plugin resolve from layout config
		Extra:     make(map[string]interface{}),
	}
}

// AddDependency records that this post depends on the given slug.
// Used for incremental build cache invalidation. Thread-safe.
// Duplicates are allowed at collection time; deduplication happens when recording to cache.
func (p *Post) AddDependency(slug string) {
	p.Dependencies = append(p.Dependencies, slug)
}

// slugifyRegex matches characters that are not alphanumeric, hyphens, or underscores
var slugifyRegex = regexp.MustCompile(`[^a-z0-9\-_]+`)

// multiHyphenRegex matches multiple consecutive hyphens
var multiHyphenRegex = regexp.MustCompile(`-+`)

// KnownExtensions contains file extensions that should be stripped from filenames.
// Extensions not in this list are treated as part of the filename.
var KnownExtensions = map[string]bool{
	".md": true, ".markdown": true, ".mdown": true, ".mkd": true,
	".html": true, ".htm": true,
	".txt": true, ".text": true,
	".rst": true, ".asciidoc": true, ".adoc": true,
}

// StripKnownExtension removes only recognized file extensions from filenames.
// For example: "post.md" -> "post", but "v1.2.3" -> "v1.2.3"
func StripKnownExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext != "" && KnownExtensions[strings.ToLower(ext)] {
		return strings.TrimSuffix(filename, ext)
	}
	return filename
}

// GenerateSlug generates a URL-safe slug from the title or path.
// If a title is set, it uses the title; otherwise, it derives the slug from the file path.
//
// Special handling for index.md files:
//   - ./index.md → "" (empty slug, becomes homepage)
//   - docs/index.md → "docs"
//   - blog/guides/index.md → "blog/guides"
func (p *Post) GenerateSlug() {
	var source string

	// Check for index.md special case
	base := filepath.Base(p.Path)
	if strings.EqualFold(base, "index.md") {
		// For index.md files, use the directory path as the slug
		dir := filepath.Dir(p.Path)
		// Clean up the directory path
		dir = filepath.Clean(dir)
		// Remove leading ./ or just .
		if dir == "." {
			p.Slug = "" // Root index.md becomes homepage
			return
		}
		// Use directory path as slug (normalized)
		slug := strings.ToLower(dir)
		slug = strings.ReplaceAll(slug, string(filepath.Separator), "/")
		slug = strings.TrimPrefix(slug, "./")
		p.Slug = slug
		return
	}

	// Priority: basename > title
	// Use the filename without known extension as the primary source
	// Only strip recognized file extensions, keeping periods in version numbers etc.
	basename := StripKnownExtension(base)
	if basename != "" {
		source = basename
	} else if p.Title != nil && *p.Title != "" {
		// Fallback to title if basename is somehow empty
		source = *p.Title
	}

	p.Slug = Slugify(source)
}

// Slugify converts a string to a URL-safe slug.
// It converts to lowercase, replaces non-alphanumeric characters with hyphens,
// collapses multiple hyphens, and trims leading/trailing hyphens.
func Slugify(s string) string {
	// Convert to lowercase
	slug := strings.ToLower(s)

	// Replace spaces with hyphens first (before the regex to preserve intent)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Replace invalid characters with hyphens (not remove them)
	slug = slugifyRegex.ReplaceAllString(slug, "-")

	// Collapse multiple hyphens
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// GenerateHref generates the relative URL path from the slug.
// The href follows the pattern /{slug}/ or / for empty slug (homepage).
// Note: This method assumes the slug is already set. Use GenerateSlug() first
// if automatic slug generation is needed.
func (p *Post) GenerateHref() {
	// Empty slug means homepage
	if p.Slug == "" {
		p.Href = "/"
		return
	}
	p.Href = "/" + p.Slug + "/"
}

// Get retrieves a value from the Extra map by key.
// Returns nil if the key does not exist.
func (p *Post) Get(key string) interface{} {
	if p.Extra == nil {
		return nil
	}
	return p.Extra[key]
}

// Set sets a value in the Extra map.
// Initializes the Extra map if it is nil.
func (p *Post) Set(key string, value interface{}) {
	if p.Extra == nil {
		p.Extra = make(map[string]interface{})
	}
	p.Extra[key] = value
}

// Has checks if a key exists in the Extra map.
func (p *Post) Has(key string) bool {
	if p.Extra == nil {
		return false
	}
	_, exists := p.Extra[key]
	return exists
}

// GetAuthors returns a list of author IDs for this post.
// Prefers the Authors array if available, falls back to single Author field.
func (p *Post) GetAuthors() []string {
	if len(p.Authors) > 0 {
		return p.Authors
	}
	if p.Author != nil && *p.Author != "" {
		return []string{*p.Author}
	}
	return []string{}
}

// HasAuthor checks if the post has the specified author ID.
func (p *Post) HasAuthor(authorID string) bool {
	for _, id := range p.GetAuthors() {
		if id == authorID {
			return true
		}
	}
	return false
}

// SetAuthors sets the authors for this post.
// Accepts either a single string (for backward compatibility), an array of strings,
// or a mixed array where items can be strings or maps with "id" and optional "role".
// Per-post role overrides from map entries are stored in AuthorRoleOverrides.
func (p *Post) SetAuthors(authors interface{}) {
	switch v := authors.(type) {
	case string:
		p.Author = &v
		p.Authors = nil
		p.AuthorRoleOverrides = nil
		p.AuthorDetailsOverrides = nil
	case []string:
		p.Authors = v
		p.Author = nil
		p.AuthorRoleOverrides = nil
		p.AuthorDetailsOverrides = nil
	case []interface{}:
		authorIDs, roleOverrides, detailsOverrides := parseAuthorItems(v)
		p.Authors = authorIDs
		p.Author = nil
		p.AuthorRoleOverrides = roleOverrides
		p.AuthorDetailsOverrides = detailsOverrides
	}
}

// parseAuthorItems processes a mixed-format author slice where items can be
// strings or maps with "id", optional "role", and optional "details".
// Returns the collected author IDs and any per-author overrides.
func parseAuthorItems(items []interface{}) (ids []string, roles, details map[string]string) {
	ids = make([]string, 0, len(items))
	roles = make(map[string]string)
	details = make(map[string]string)
	for _, item := range items {
		switch entry := item.(type) {
		case string:
			ids = append(ids, entry)
		case map[string]interface{}:
			addAuthorEntry(entry, &ids, roles, details)
		case map[interface{}]interface{}:
			// YAML sometimes produces map[interface{}]interface{} instead of map[string]interface{}
			normalized := make(map[string]interface{}, len(entry))
			for k, v := range entry {
				if ks, ok := k.(string); ok {
					normalized[ks] = v
				}
			}
			addAuthorEntry(normalized, &ids, roles, details)
		}
	}
	if len(roles) == 0 {
		roles = nil
	}
	if len(details) == 0 {
		details = nil
	}
	return ids, roles, details
}

// addAuthorEntry extracts id, role, and details from a map-format author entry.
// Appends the author ID to ids and populates role/details override maps as needed.
func addAuthorEntry(entry map[string]interface{}, ids *[]string, roles, details map[string]string) {
	id, ok := entry["id"].(string)
	if !ok || id == "" {
		return
	}
	*ids = append(*ids, id)
	if role, ok := entry["role"].(string); ok && role != "" {
		roles[id] = role
	}
	if detail, ok := entry["details"].(string); ok && detail != "" {
		details[id] = detail
	}
}
