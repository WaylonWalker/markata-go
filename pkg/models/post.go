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

	// Skip indicates if the post should be skipped during processing
	Skip bool `json:"skip" yaml:"skip" toml:"skip"`

	// Tags is a list of tags associated with the post
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty" toml:"tags,omitempty"`

	// Description is the optional post description
	Description *string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`

	// Template is the template file to use for rendering (default: "post.html")
	Template string `json:"template" yaml:"template" toml:"template"`

	// HTML is the final rendered HTML including template wrapper
	HTML string `json:"html" yaml:"html" toml:"html"`

	// ArticleHTML is the rendered content without template wrapper
	ArticleHTML string `json:"article_html" yaml:"article_html" toml:"article_html"`

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

	// Extra holds dynamic/unknown fields from frontmatter
	Extra map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty" toml:"extra,omitempty"`
}

// NewPost creates a new Post with the given source file path and default values.
func NewPost(path string) *Post {
	return &Post{
		Path:      path,
		Published: false,
		Draft:     false,
		Skip:      false,
		Tags:      []string{},
		Template:  "", // Empty - let templates plugin resolve from layout config
		Extra:     make(map[string]interface{}),
	}
}

// slugifyRegex matches characters that are not alphanumeric, hyphens, or underscores
var slugifyRegex = regexp.MustCompile(`[^a-z0-9\-_]+`)

// multiHyphenRegex matches multiple consecutive hyphens
var multiHyphenRegex = regexp.MustCompile(`-+`)

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
	// Use the filename without extension as the primary source
	basename := strings.TrimSuffix(base, filepath.Ext(base))
	if basename != "" {
		source = basename
	} else if p.Title != nil && *p.Title != "" {
		// Fallback to title if basename is somehow empty
		source = *p.Title
	}

	// Convert to lowercase
	slug := strings.ToLower(source)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove invalid characters
	slug = slugifyRegex.ReplaceAllString(slug, "")

	// Collapse multiple hyphens
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	p.Slug = slug
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
