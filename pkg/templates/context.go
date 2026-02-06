package templates

import (
	"reflect"
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

	// SidebarItems holds the resolved sidebar navigation items for the current page
	SidebarItems []models.SidebarNavItem

	// SidebarTitle holds the title for the current page's sidebar
	SidebarTitle string

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

// WithSidebar returns a copy of the context with the SidebarItems and SidebarTitle fields set.
func (c Context) WithSidebar(items []models.SidebarNavItem, title string) Context {
	c.SidebarItems = items
	c.SidebarTitle = title
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

// postToMap converts a Post to a map for template access using the global cache.
// This handles pointer fields and provides a cleaner interface for pongo2.
func postToMap(p *models.Post) map[string]interface{} {
	return GetPostMap(p)
}

// postToMapUncached converts a Post to a map without caching.
// This is the actual conversion logic used by the cache.
func postToMapUncached(p *models.Post) map[string]interface{} {
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
		"private":      p.Private,
		"skip":         p.Skip,
		"tags":         p.Tags,
		"template":     p.Template,
		"templateKey":  p.Template, // Alias for backwards compatibility
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
		// Also expose Extra as a nested map for template access like post.Extra.key
		extraMap := make(map[string]interface{})
		for k, v := range p.Extra {
			extraMap[k] = v
			// Don't override existing keys (flatten to top level for backwards compat)
			if _, exists := m[k]; !exists {
				// Special handling for structured_data
				if k == "structured_data" {
					if sd, ok := v.(*models.StructuredData); ok {
						m[k] = structuredDataToMap(sd)
						extraMap[k] = structuredDataToMap(sd)
						continue
					}
				}
				// Special handling for toc entries (from toc plugin)
				if k == "toc" {
					m[k] = tocEntriesToMaps(v)
					extraMap[k] = tocEntriesToMaps(v)
					continue
				}
				m[k] = v
			}
		}
		m["Extra"] = extraMap
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
		"author_image":   c.SEO.AuthorImage,
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

	// Convert search to map
	searchMap := searchToMap(&c.Search)

	// Convert WebSub to map
	webSubMap := webSubToMap(&c.WebSub)

	// Convert layout to map
	layoutMap := layoutToMap(&c.Layout)

	// Convert sidebar to map
	sidebarMap := sidebarToMap(&c.Sidebar)

	// Convert toc to map
	tocMap := tocToMap(&c.Toc)

	// Convert header to map
	headerMap := headerToMap(&c.Header)

	// Convert theme to map
	themeMap := ThemeToMap(&c.Theme)

	result := map[string]interface{}{
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
		"search":        searchMap,
		"websub":        webSubMap,
		"layout":        layoutMap,
		"sidebar":       sidebarMap,
		"toc":           tocMap,
		"header":        headerMap,
		"theme":         themeMap,
	}

	// Add Extra map for plugin configs (e.g., glightbox_enabled, glightbox_options)
	if c.Extra != nil {
		result["Extra"] = c.Extra
	}

	return result
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

	// Convert card_router component - use MergedMappings to include defaults
	cardRouterMap := map[string]interface{}{
		"mappings": c.CardRouter.MergedMappings(),
	}

	return map[string]interface{}{
		"nav":          navMap,
		"footer":       footerMap,
		"doc_sidebar":  docSidebarMap,
		"feed_sidebar": feedSidebarMap,
		"card_router":  cardRouterMap,
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
		"text":     p.Text,
		"og":       p.OG,
	}
}

// webSubToMap converts a WebSubConfig to a map for template access.
func webSubToMap(w *models.WebSubConfig) map[string]interface{} {
	if w == nil {
		return nil
	}

	enabled := false
	if w.Enabled != nil {
		enabled = *w.Enabled
	}

	return map[string]interface{}{
		"enabled": enabled,
		"hubs":    append([]string{}, w.Hubs...),
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

// searchToMap converts a SearchConfig to a map for template access.
func searchToMap(s *models.SearchConfig) map[string]interface{} {
	if s == nil {
		return nil
	}

	// Determine enabled status (default: true)
	enabled := true
	if s.Enabled != nil {
		enabled = *s.Enabled
	}

	// Determine show_images status (default: true)
	showImages := true
	if s.ShowImages != nil {
		showImages = *s.ShowImages
	}

	// Defaults for optional fields
	position := s.Position
	if position == "" {
		position = "navbar"
	}

	placeholder := s.Placeholder
	if placeholder == "" {
		placeholder = "Search…"
	}

	excerptLength := s.ExcerptLength
	if excerptLength == 0 {
		excerptLength = 200
	}

	// Convert pagefind config
	bundleDir := s.Pagefind.BundleDir
	if bundleDir == "" {
		bundleDir = "_pagefind"
	}

	pagefindMap := map[string]interface{}{
		"bundle_dir":        bundleDir,
		"exclude_selectors": s.Pagefind.ExcludeSelectors,
		"root_selector":     s.Pagefind.RootSelector,
	}

	// Convert feed-specific search configs
	feedConfigs := make([]map[string]interface{}, len(s.Feeds))
	for i, feed := range s.Feeds {
		feedConfigs[i] = map[string]interface{}{
			"name":        feed.Name,
			"filter":      feed.Filter,
			"position":    feed.Position,
			"placeholder": feed.Placeholder,
		}
	}

	return map[string]interface{}{
		"enabled":        enabled,
		"position":       position,
		"placeholder":    placeholder,
		"show_images":    showImages,
		"excerpt_length": excerptLength,
		"pagefind":       pagefindMap,
		"feeds":          feedConfigs,
	}
}

// layoutToMap converts a LayoutConfig to a map for template access.
func layoutToMap(l *models.LayoutConfig) map[string]interface{} {
	if l == nil {
		return nil
	}

	// Convert docs layout config
	docsMap := map[string]interface{}{
		"sidebar_position":  l.Docs.SidebarPosition,
		"sidebar_width":     l.Docs.SidebarWidth,
		"toc_position":      l.Docs.TocPosition,
		"toc_width":         l.Docs.TocWidth,
		"content_max_width": l.Docs.ContentMaxWidth,
		"header_style":      l.Docs.HeaderStyle,
		"footer_style":      l.Docs.FooterStyle,
	}
	if l.Docs.SidebarCollapsible != nil {
		docsMap["sidebar_collapsible"] = *l.Docs.SidebarCollapsible
	}
	if l.Docs.SidebarDefaultOpen != nil {
		docsMap["sidebar_default_open"] = *l.Docs.SidebarDefaultOpen
	}
	if l.Docs.TocCollapsible != nil {
		docsMap["toc_collapsible"] = *l.Docs.TocCollapsible
	}
	if l.Docs.TocDefaultOpen != nil {
		docsMap["toc_default_open"] = *l.Docs.TocDefaultOpen
	}

	// Convert blog layout config
	blogMap := map[string]interface{}{
		"content_max_width": l.Blog.ContentMaxWidth,
		"toc_position":      l.Blog.TocPosition,
		"toc_width":         l.Blog.TocWidth,
		"header_style":      l.Blog.HeaderStyle,
		"footer_style":      l.Blog.FooterStyle,
	}
	if l.Blog.ShowToc != nil {
		blogMap["show_toc"] = *l.Blog.ShowToc
	}
	if l.Blog.ShowAuthor != nil {
		blogMap["show_author"] = *l.Blog.ShowAuthor
	}
	if l.Blog.ShowDate != nil {
		blogMap["show_date"] = *l.Blog.ShowDate
	}
	if l.Blog.ShowTags != nil {
		blogMap["show_tags"] = *l.Blog.ShowTags
	}
	if l.Blog.ShowReadingTime != nil {
		blogMap["show_reading_time"] = *l.Blog.ShowReadingTime
	}
	if l.Blog.ShowPrevNext != nil {
		blogMap["show_prev_next"] = *l.Blog.ShowPrevNext
	}

	// Convert landing layout config
	landingMap := map[string]interface{}{
		"content_max_width": l.Landing.ContentMaxWidth,
		"header_style":      l.Landing.HeaderStyle,
		"footer_style":      l.Landing.FooterStyle,
	}
	if l.Landing.HeaderSticky != nil {
		landingMap["header_sticky"] = *l.Landing.HeaderSticky
	}
	if l.Landing.HeroEnabled != nil {
		landingMap["hero_enabled"] = *l.Landing.HeroEnabled
	}

	// Convert bare layout config
	bareMap := map[string]interface{}{
		"content_max_width": l.Bare.ContentMaxWidth,
	}

	// Convert defaults
	defaultsMap := map[string]interface{}{
		"content_max_width": l.Defaults.ContentMaxWidth,
	}
	if l.Defaults.HeaderSticky != nil {
		defaultsMap["header_sticky"] = *l.Defaults.HeaderSticky
	}
	if l.Defaults.FooterSticky != nil {
		defaultsMap["footer_sticky"] = *l.Defaults.FooterSticky
	}

	return map[string]interface{}{
		"name":     l.Name,
		"paths":    l.Paths,
		"feeds":    l.Feeds,
		"docs":     docsMap,
		"blog":     blogMap,
		"landing":  landingMap,
		"bare":     bareMap,
		"defaults": defaultsMap,
	}
}

// sidebarToMap converts a SidebarConfig to a map for template access.
func sidebarToMap(s *models.SidebarConfig) map[string]interface{} {
	if s == nil {
		return nil
	}

	// Convert nav items
	navItems := sidebarItemsToMaps(s.Nav)

	result := map[string]interface{}{
		"position": s.Position,
		"width":    s.Width,
		"title":    s.Title,
		"nav":      navItems,
	}

	// Handle pointer fields with defaults
	if s.Enabled != nil {
		result["enabled"] = *s.Enabled
	} else {
		result["enabled"] = true // default for docs layout
	}
	if s.Collapsible != nil {
		result["collapsible"] = *s.Collapsible
	}
	if s.DefaultOpen != nil {
		result["default_open"] = *s.DefaultOpen
	}

	return result
}

// tocToMap converts a TocConfig to a map for template access.
func tocToMap(t *models.TocConfig) map[string]interface{} {
	if t == nil {
		return nil
	}

	result := map[string]interface{}{
		"position":  t.Position,
		"width":     t.Width,
		"min_depth": t.MinDepth,
		"max_depth": t.MaxDepth,
		"title":     t.Title,
	}

	// Handle pointer fields with defaults
	if t.Enabled != nil {
		result["enabled"] = *t.Enabled
	} else {
		result["enabled"] = true // default for docs layout
	}
	if t.Collapsible != nil {
		result["collapsible"] = *t.Collapsible
	}
	if t.DefaultOpen != nil {
		result["default_open"] = *t.DefaultOpen
	}

	return result
}

// headerToMap converts a HeaderLayoutConfig to a map for template access.
func headerToMap(h *models.HeaderLayoutConfig) map[string]interface{} {
	if h == nil {
		return nil
	}

	result := map[string]interface{}{
		"style": h.Style,
	}

	// Handle pointer fields with defaults
	if h.Sticky != nil {
		result["sticky"] = *h.Sticky
	} else {
		result["sticky"] = true // default
	}
	if h.ShowLogo != nil {
		result["show_logo"] = *h.ShowLogo
	} else {
		result["show_logo"] = true // default
	}
	if h.ShowTitle != nil {
		result["show_title"] = *h.ShowTitle
	} else {
		result["show_title"] = true // default
	}
	if h.ShowNav != nil {
		result["show_nav"] = *h.ShowNav
	} else {
		result["show_nav"] = true // default
	}
	if h.ShowSearch != nil {
		result["show_search"] = *h.ShowSearch
	} else {
		result["show_search"] = true // default
	}
	if h.ShowThemeToggle != nil {
		result["show_theme_toggle"] = *h.ShowThemeToggle
	} else {
		result["show_theme_toggle"] = true // default
	}

	return result
}

// ThemeToMap converts a ThemeConfig to a map for template access.
// This is exported for use by plugins that need consistent theme config rendering.
func ThemeToMap(t *models.ThemeConfig) map[string]interface{} {
	if t == nil {
		return nil
	}

	backgroundMap := BackgroundToMap(&t.Background)
	fontMap := FontToMap(&t.Font)
	switcherMap := SwitcherToMap(&t.Switcher)

	return map[string]interface{}{
		"name":          t.Name,
		"palette":       t.Palette,
		"palette_light": t.PaletteLight,
		"palette_dark":  t.PaletteDark,
		"variables":     t.Variables,
		"custom_css":    t.CustomCSS,
		"background":    backgroundMap,
		"font":          fontMap,
		"switcher":      switcherMap,
	}
}

// SwitcherToMap converts a ThemeSwitcherConfig to a map for template access.
// This is exported for use by plugins that need consistent switcher config rendering.
func SwitcherToMap(s *models.ThemeSwitcherConfig) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{
			"enabled":     false,
			"include_all": true,
			"position":    "header",
		}
	}

	return map[string]interface{}{
		"enabled":     s.IsEnabled(),
		"include_all": s.IsIncludeAll(),
		"include":     s.Include,
		"exclude":     s.Exclude,
		"position":    s.Position,
	}
}

// BackgroundToMap converts a BackgroundConfig to a map for template access.
// This is exported for use by plugins that need consistent background config rendering.
func BackgroundToMap(b *models.BackgroundConfig) map[string]interface{} {
	if b == nil {
		return nil
	}

	backgroundElements := make([]map[string]interface{}, len(b.Backgrounds))
	for i, bg := range b.Backgrounds {
		backgroundElements[i] = map[string]interface{}{
			"html":    bg.HTML,
			"z_index": bg.ZIndex,
		}
	}

	result := map[string]interface{}{
		"backgrounds":          backgroundElements,
		"scripts":              b.Scripts,
		"css":                  b.CSS,
		"article_bg":           b.ArticleBg,
		"article_blur_enabled": b.IsArticleBlurEnabled(),
		"article_blur":         b.ArticleBlur,
		"article_shadow":       b.ArticleShadow,
		"article_border":       b.ArticleBorder,
		"article_radius":       b.ArticleRadius,
	}

	if b.Enabled != nil {
		result["enabled"] = *b.Enabled
	} else {
		result["enabled"] = false
	}

	return result
}

// FontToMap converts a FontConfig to a map for template access.
// This is exported for use by plugins that need consistent font config rendering.
func FontToMap(f *models.FontConfig) map[string]interface{} {
	if f == nil {
		return nil
	}

	return map[string]interface{}{
		"family":         f.Family,
		"heading_family": f.HeadingFamily,
		"code_family":    f.CodeFamily,
		"size":           f.Size,
		"line_height":    f.LineHeight,
		"google_fonts":   f.GoogleFonts,
		"custom_urls":    f.CustomURLs,
	}
}

// tocEntriesToMaps converts TOC entries (from the toc plugin) to template-friendly maps.
// It uses reflection to avoid import cycles with the plugins package.
func tocEntriesToMaps(entries interface{}) []map[string]interface{} {
	if entries == nil {
		return nil
	}

	// Use reflection to handle the []*plugins.TocEntry type
	v := reflect.ValueOf(entries)
	if v.Kind() != reflect.Slice {
		return nil
	}

	result := make([]map[string]interface{}, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		entry := v.Index(i)
		if entry.Kind() == reflect.Ptr {
			entry = entry.Elem()
		}
		if entry.Kind() != reflect.Struct {
			continue
		}

		// Extract fields by name
		levelField := entry.FieldByName("Level")
		textField := entry.FieldByName("Text")
		idField := entry.FieldByName("ID")
		childrenField := entry.FieldByName("Children")

		entryMap := map[string]interface{}{
			"level": 0,
			"text":  "",
			"id":    "",
		}

		if levelField.IsValid() {
			entryMap["level"] = int(levelField.Int())
		}
		if textField.IsValid() {
			entryMap["text"] = textField.String()
		}
		if idField.IsValid() {
			entryMap["id"] = idField.String()
		}

		// Recursively convert children
		if childrenField.IsValid() && !childrenField.IsNil() {
			children := tocEntriesToMaps(childrenField.Interface())
			if len(children) > 0 {
				entryMap["children"] = children
			}
		}

		result = append(result, entryMap)
	}

	return result
}

// structuredDataToMap converts a StructuredData to a map for template access.
func structuredDataToMap(sd *models.StructuredData) map[string]interface{} {
	if sd == nil {
		return nil
	}

	// Convert OpenGraph tags
	opengraphTags := make([]map[string]interface{}, len(sd.OpenGraph))
	for i, og := range sd.OpenGraph {
		opengraphTags[i] = map[string]interface{}{
			"property": og.Property,
			"content":  og.Content,
		}
	}

	// Convert Twitter tags
	twitterTags := make([]map[string]interface{}, len(sd.Twitter))
	for i, tw := range sd.Twitter {
		twitterTags[i] = map[string]interface{}{
			"name":    tw.Name,
			"content": tw.Content,
		}
	}

	return map[string]interface{}{
		"jsonld":    sd.JSONLD,
		"opengraph": opengraphTags,
		"twitter":   twitterTags,
	}
}

// feedToMap converts a FeedConfig to a map for template access.
func feedToMap(f *models.FeedConfig) map[string]interface{} {
	if f == nil {
		return nil
	}

	formats := map[string]interface{}{
		"html":     f.Formats.HTML,
		"rss":      f.Formats.RSS,
		"atom":     f.Formats.Atom,
		"json":     f.Formats.JSON,
		"markdown": f.Formats.Markdown,
		"text":     f.Formats.Text,
		"sitemap":  f.Formats.Sitemap,
	}

	// Compute base_url from slug (e.g., "archive" -> "/archive")
	baseURL := "/" + f.Slug

	return map[string]interface{}{
		"slug":           f.Slug,
		"base_url":       baseURL,
		"title":          f.Title,
		"description":    f.Description,
		"filter":         f.Filter,
		"sort":           f.Sort,
		"reverse":        f.Reverse,
		"items_per_page": f.ItemsPerPage,
		"posts":          PostsToMaps(f.Posts),
		"formats":        formats,
	}
}

// feedPageToMap converts a FeedPage to a map for template access.
func feedPageToMap(p *models.FeedPage) map[string]interface{} {
	if p == nil {
		return nil
	}

	return map[string]interface{}{
		"number":          p.Number,
		"posts":           PostsToMaps(p.Posts),
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

// PostsToMaps converts a slice of Posts to a slice of maps.
// Exported for use by plugins that need to add posts to template context.
func PostsToMaps(posts []*models.Post) []map[string]interface{} {
	if posts == nil {
		return nil
	}
	result := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		result[i] = postToMap(p)
	}
	return result
}

// sidebarItemsToMaps converts a slice of SidebarNavItems to a slice of maps.
func sidebarItemsToMaps(items []models.SidebarNavItem) []map[string]interface{} {
	if items == nil {
		return nil
	}
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		itemMap := map[string]interface{}{
			"title":        item.Title,
			"href":         item.Href,
			"has_children": len(item.Children) > 0,
		}
		if len(item.Children) > 0 {
			itemMap["children"] = sidebarItemsToMaps(item.Children)
		}
		result[i] = itemMap
	}
	return result
}

// ToPongo2 converts the Context to a pongo2.Context for template execution.
func (c Context) ToPongo2() pongo2.Context {
	postMap := postToMap(c.Post)
	configMap := GetConfigMap(c.Config)

	ctx := pongo2.Context{
		"post":          postMap,
		"body":          c.Body,
		"config":        configMap,
		"feed":          feedToMap(c.Feed),
		"page":          feedPageToMap(c.FeedPage),
		"posts":         PostsToMaps(c.Posts),
		"core":          c.Core,
		"sidebar_items": sidebarItemsToMaps(c.SidebarItems),
		"sidebar_title": c.SidebarTitle,
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
		ctx["private"] = postMap["private"]
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

	// Add feed fields directly for convenience (if feed exists)
	if c.Feed != nil {
		ctx["feed_slug"] = c.Feed.Slug
		ctx["feed_title"] = c.Feed.Title
		ctx["feed_description"] = c.Feed.Description
	}

	// Add extra context values
	if c.Extra != nil {
		for k, v := range c.Extra {
			// Don't override existing keys
			if _, exists := ctx[k]; !exists {
				// Convert Post types to maps for template access
				switch typed := v.(type) {
				case []*models.Post:
					ctx[k] = PostsToMaps(typed)
				case *models.Post:
					ctx[k] = postToMap(typed)
				default:
					ctx[k] = v
				}
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
	if other.SidebarItems != nil {
		c.SidebarItems = other.SidebarItems
	}
	if other.SidebarTitle != "" {
		c.SidebarTitle = other.SidebarTitle
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
		Post:         c.Post,
		Body:         c.Body,
		Config:       c.Config,
		Feed:         c.Feed,
		FeedPage:     c.FeedPage,
		Core:         c.Core,
		SidebarTitle: c.SidebarTitle,
	}

	// Copy Posts slice
	if c.Posts != nil {
		clone.Posts = make([]*models.Post, len(c.Posts))
		copy(clone.Posts, c.Posts)
	}

	// Copy SidebarItems slice
	if c.SidebarItems != nil {
		clone.SidebarItems = make([]models.SidebarNavItem, len(c.SidebarItems))
		copy(clone.SidebarItems, c.SidebarItems)
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
