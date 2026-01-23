package templates

import (
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/flosch/pongo2/v6"
)

// Context holds data available to templates during rendering.
type Context struct {
	// Post is the current post being rendered
	Post *models.Post

	// Body is the rendered article HTML (markdown content converted to HTML)
	Body string

	// Config is the site configuration
	Config *models.Config

	// Feed is the feed being rendered (for feed templates)
	Feed *models.FeedConfig

	// FeedPage is the current page of paginated feed results
	FeedPage *models.FeedPage

	// Posts is the list of all posts (for index/archive templates)
	Posts []*models.Post

	// Core provides access to the lifecycle manager for filter/map operations
	Core interface{}

	// Extra holds additional context values
	Extra map[string]interface{}
}

// NewContext creates a new template context with the given post, body, and config.
func NewContext(post *models.Post, body string, config *models.Config) Context {
	return Context{
		Post:   post,
		Body:   body,
		Config: config,
		Extra:  make(map[string]interface{}),
	}
}

// NewFeedContext creates a new template context for feed rendering.
func NewFeedContext(feed *models.FeedConfig, page *models.FeedPage, config *models.Config) Context {
	return Context{
		Feed:     feed,
		FeedPage: page,
		Config:   config,
		Posts:    page.Posts,
		Extra:    make(map[string]interface{}),
	}
}

// WithCore returns a copy of the context with the Core field set.
func (c Context) WithCore(core interface{}) Context {
	c.Core = core
	return c
}

// WithPosts returns a copy of the context with the Posts field set.
func (c Context) WithPosts(posts []*models.Post) Context {
	c.Posts = posts
	return c
}

// Set sets an extra value in the context.
func (c *Context) Set(key string, value interface{}) {
	if c.Extra == nil {
		c.Extra = make(map[string]interface{})
	}
	c.Extra[key] = value
}

// Get retrieves an extra value from the context.
func (c *Context) Get(key string) interface{} {
	if c.Extra == nil {
		return nil
	}
	return c.Extra[key]
}

// postToMap converts a Post to a map for template access.
// This handles pointer fields and provides a cleaner interface for pongo2.
func postToMap(p *models.Post) map[string]interface{} {
	if p == nil {
		return nil
	}

	m := map[string]interface{}{
		"path":         p.Path,
		"content":      p.Content,
		"slug":         p.Slug,
		"href":         p.Href,
		"published":    p.Published,
		"draft":        p.Draft,
		"skip":         p.Skip,
		"tags":         p.Tags,
		"template":     p.Template,
		"html":         p.HTML,
		"article_html": p.ArticleHTML,
	}

	// Handle pointer fields - dereference if not nil
	if p.Title != nil {
		m["title"] = *p.Title
	} else {
		m["title"] = nil
	}

	if p.Date != nil {
		m["date"] = *p.Date
	} else {
		m["date"] = nil
	}

	if p.Description != nil {
		m["description"] = *p.Description
	} else {
		m["description"] = nil
	}

	// Add extra fields
	if p.Extra != nil {
		for k, v := range p.Extra {
			// Don't override existing keys
			if _, exists := m[k]; !exists {
				m[k] = v
			}
		}
	}

	return m
}

// configToMap converts a Config to a map for template access.
func configToMap(c *models.Config) map[string]interface{} {
	if c == nil {
		return nil
	}

	// Convert nav items to maps
	navItems := make([]map[string]interface{}, len(c.Nav))
	for i, nav := range c.Nav {
		navItems[i] = map[string]interface{}{
			"label":    nav.Label,
			"url":      nav.URL,
			"external": nav.External,
		}
	}

	// Convert footer to map
	footerMap := map[string]interface{}{
		"text": c.Footer.Text,
	}
	if c.Footer.ShowCopyright != nil {
		footerMap["show_copyright"] = *c.Footer.ShowCopyright
	} else {
		footerMap["show_copyright"] = true // default to true
	}

	// Convert SEO to map
	seoMap := map[string]interface{}{
		"twitter_handle": c.SEO.TwitterHandle,
		"default_image":  c.SEO.DefaultImage,
		"logo_url":       c.SEO.LogoURL,
	}

	// Convert IndieAuth to map
	indieAuthMap := map[string]interface{}{
		"enabled":                c.IndieAuth.Enabled,
		"authorization_endpoint": c.IndieAuth.AuthorizationEndpoint,
		"token_endpoint":         c.IndieAuth.TokenEndpoint,
		"me_url":                 c.IndieAuth.MeURL,
	}

	// Convert Webmention to map
	webmentionMap := map[string]interface{}{
		"enabled":  c.Webmention.Enabled,
		"endpoint": c.Webmention.Endpoint,
	}

	// Convert components to map
	componentsMap := componentsToMap(&c.Components)

	// Convert post_formats to map
	postFormatsMap := postFormatsToMap(&c.PostFormats)

	// Convert head to map
	headMap := headToMap(&c.Head)

	return map[string]interface{}{
		"output_dir":    c.OutputDir,
		"url":           c.URL,
		"title":         c.Title,
		"description":   c.Description,
		"author":        c.Author,
		"assets_dir":    c.AssetsDir,
		"templates_dir": c.TemplatesDir,
		"nav":           navItems,
		"footer":        footerMap,
		"seo":           seoMap,
		"indieauth":     indieAuthMap,
		"webmention":    webmentionMap,
		"components":    componentsMap,
		"post_formats":  postFormatsMap,
		"head":          headMap,
	}
}

// componentsToMap converts a ComponentsConfig to a map for template access.
func componentsToMap(c *models.ComponentsConfig) map[string]interface{} {
	if c == nil {
		return nil
	}

	// Convert nav component
	navEnabled := true
	if c.Nav.Enabled != nil {
		navEnabled = *c.Nav.Enabled
	}
	navItems := make([]map[string]interface{}, len(c.Nav.Items))
	for i, item := range c.Nav.Items {
		navItems[i] = map[string]interface{}{
			"label":    item.Label,
			"url":      item.URL,
			"external": item.External,
		}
	}
	navMap := map[string]interface{}{
		"enabled":  navEnabled,
		"position": c.Nav.Position,
		"style":    c.Nav.Style,
		"items":    navItems,
	}

	// Convert footer component
	footerEnabled := true
	if c.Footer.Enabled != nil {
		footerEnabled = *c.Footer.Enabled
	}
	showCopyright := true
	if c.Footer.ShowCopyright != nil {
		showCopyright = *c.Footer.ShowCopyright
	}
	footerLinks := make([]map[string]interface{}, len(c.Footer.Links))
	for i, link := range c.Footer.Links {
		footerLinks[i] = map[string]interface{}{
			"label":    link.Label,
			"url":      link.URL,
			"external": link.External,
		}
	}
	footerMap := map[string]interface{}{
		"enabled":        footerEnabled,
		"text":           c.Footer.Text,
		"show_copyright": showCopyright,
		"links":          footerLinks,
	}

	// Convert doc_sidebar component
	docSidebarEnabled := false
	if c.DocSidebar.Enabled != nil {
		docSidebarEnabled = *c.DocSidebar.Enabled
	}
	docSidebarMap := map[string]interface{}{
		"enabled":   docSidebarEnabled,
		"position":  c.DocSidebar.Position,
		"width":     c.DocSidebar.Width,
		"min_depth": c.DocSidebar.MinDepth,
		"max_depth": c.DocSidebar.MaxDepth,
	}

	// Convert feed_sidebar component
	feedSidebarEnabled := false
	if c.FeedSidebar.Enabled != nil {
		feedSidebarEnabled = *c.FeedSidebar.Enabled
	}
	feedSidebarMap := map[string]interface{}{
		"enabled":  feedSidebarEnabled,
		"position": c.FeedSidebar.Position,
		"width":    c.FeedSidebar.Width,
		"title":    c.FeedSidebar.Title,
		"feeds":    c.FeedSidebar.Feeds,
	}

	return map[string]interface{}{
		"nav":          navMap,
		"footer":       footerMap,
		"doc_sidebar":  docSidebarMap,
		"feed_sidebar": feedSidebarMap,
	}
}

// postFormatsToMap converts a PostFormatsConfig to a map for template access.
func postFormatsToMap(p *models.PostFormatsConfig) map[string]interface{} {
	if p == nil {
		return nil
	}

	htmlEnabled := true
	if p.HTML != nil {
		htmlEnabled = *p.HTML
	}

	return map[string]interface{}{
		"html":     htmlEnabled,
		"markdown": p.Markdown,
		"og":       p.OG,
	}
}

// headToMap converts a HeadConfig to a map for template access.
func headToMap(h *models.HeadConfig) map[string]interface{} {
	if h == nil {
		return nil
	}

	// Convert meta tags
	metaTags := make([]map[string]interface{}, len(h.Meta))
	for i, meta := range h.Meta {
		metaTags[i] = map[string]interface{}{
			"name":     meta.Name,
			"property": meta.Property,
			"content":  meta.Content,
		}
	}

	// Convert link tags
	linkTags := make([]map[string]interface{}, len(h.Link))
	for i, link := range h.Link {
		linkTags[i] = map[string]interface{}{
			"rel":         link.Rel,
			"href":        link.Href,
			"crossorigin": link.Crossorigin,
		}
	}

	// Convert script tags
	scriptTags := make([]map[string]interface{}, len(h.Script))
	for i, script := range h.Script {
		scriptTags[i] = map[string]interface{}{
			"src": script.Src,
		}
	}

	// Convert alternate feeds
	alternateFeeds := make([]map[string]interface{}, len(h.AlternateFeeds))
	for i, feed := range h.AlternateFeeds {
		alternateFeeds[i] = map[string]interface{}{
			"type":      feed.Type,
			"title":     feed.Title,
			"href":      feed.Href,
			"mime_type": feed.GetMIMEType(),
		}
	}

	return map[string]interface{}{
		"text":            h.Text,
		"meta":            metaTags,
		"link":            linkTags,
		"script":          scriptTags,
		"alternate_feeds": alternateFeeds,
	}
}

// feedToMap converts a FeedConfig to a map for template access.
func feedToMap(f *models.FeedConfig) map[string]interface{} {
	if f == nil {
		return nil
	}

	return map[string]interface{}{
		"slug":           f.Slug,
		"title":          f.Title,
		"description":    f.Description,
		"filter":         f.Filter,
		"sort":           f.Sort,
		"reverse":        f.Reverse,
		"items_per_page": f.ItemsPerPage,
		"posts":          postsToMaps(f.Posts),
	}
}

// feedPageToMap converts a FeedPage to a map for template access.
func feedPageToMap(p *models.FeedPage) map[string]interface{} {
	if p == nil {
		return nil
	}

	return map[string]interface{}{
		"number":          p.Number,
		"posts":           postsToMaps(p.Posts),
		"has_prev":        p.HasPrev,
		"has_next":        p.HasNext,
		"prev_url":        p.PrevURL,
		"next_url":        p.NextURL,
		"total_pages":     p.TotalPages,
		"total_items":     p.TotalItems,
		"items_per_page":  p.ItemsPerPage,
		"page_urls":       p.PageURLs,
		"pagination_type": string(p.PaginationType),
	}
}

// postsToMaps converts a slice of Posts to a slice of maps.
func postsToMaps(posts []*models.Post) []map[string]interface{} {
	if posts == nil {
		return nil
	}
	result := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		result[i] = postToMap(p)
	}
	return result
}

// ToPongo2 converts the Context to a pongo2.Context for template execution.
func (c Context) ToPongo2() pongo2.Context {
	postMap := postToMap(c.Post)
	configMap := configToMap(c.Config)

	ctx := pongo2.Context{
		"post":   postMap,
		"body":   c.Body,
		"config": configMap,
		"feed":   feedToMap(c.Feed),
		"page":   feedPageToMap(c.FeedPage),
		"posts":  postsToMaps(c.Posts),
		"core":   c.Core,
	}

	// Add post fields directly for convenience (if post exists)
	if postMap != nil {
		ctx["title"] = postMap["title"]
		ctx["date"] = postMap["date"]
		ctx["tags"] = postMap["tags"]
		ctx["slug"] = postMap["slug"]
		ctx["href"] = postMap["href"]
		ctx["published"] = postMap["published"]
		ctx["draft"] = postMap["draft"]
		ctx["description"] = postMap["description"]
		ctx["article_html"] = postMap["article_html"]

		// Add extra fields from post
		if c.Post != nil && c.Post.Extra != nil {
			for k, v := range c.Post.Extra {
				// Don't override existing keys
				if _, exists := ctx[k]; !exists {
					ctx[k] = v
				}
			}
		}
	}

	// Add config fields for convenience (if config exists)
	if c.Config != nil {
		ctx["site_title"] = c.Config.Title
		ctx["site_url"] = c.Config.URL
		ctx["site_description"] = c.Config.Description
		ctx["site_author"] = c.Config.Author
	}

	// Add extra context values
	if c.Extra != nil {
		for k, v := range c.Extra {
			// Don't override existing keys
			if _, exists := ctx[k]; !exists {
				ctx[k] = v
			}
		}
	}

	return ctx
}

// Merge merges another context into this one.
// Values from the other context override existing values.
func (c *Context) Merge(other Context) {
	if other.Post != nil {
		c.Post = other.Post
	}
	if other.Body != "" {
		c.Body = other.Body
	}
	if other.Config != nil {
		c.Config = other.Config
	}
	if other.Feed != nil {
		c.Feed = other.Feed
	}
	if other.FeedPage != nil {
		c.FeedPage = other.FeedPage
	}
	if other.Posts != nil {
		c.Posts = other.Posts
	}
	if other.Core != nil {
		c.Core = other.Core
	}

	if other.Extra != nil {
		if c.Extra == nil {
			c.Extra = make(map[string]interface{})
		}
		for k, v := range other.Extra {
			c.Extra[k] = v
		}
	}
}

// Clone creates a copy of the context.
func (c Context) Clone() Context {
	clone := Context{
		Post:     c.Post,
		Body:     c.Body,
		Config:   c.Config,
		Feed:     c.Feed,
		FeedPage: c.FeedPage,
		Core:     c.Core,
	}

	// Copy Posts slice
	if c.Posts != nil {
		clone.Posts = make([]*models.Post, len(c.Posts))
		copy(clone.Posts, c.Posts)
	}

	// Copy Extra map
	if c.Extra != nil {
		clone.Extra = make(map[string]interface{})
		for k, v := range c.Extra {
			clone.Extra[k] = v
		}
	}

	return clone
}

// TimeValue is a helper type to make time.Time work better in templates.
type TimeValue struct {
	time.Time
}

// String returns the time formatted as RFC3339.
func (t TimeValue) String() string {
	return t.Time.Format(time.RFC3339)
}
