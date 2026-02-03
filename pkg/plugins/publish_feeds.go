// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
	"github.com/WaylonWalker/markata-go/pkg/themes"
)

// PublishFeedsPlugin writes feeds to multiple output formats during the write stage.
// It also registers synthetic posts in the Configure stage so they can be resolved by wikilinks.
type PublishFeedsPlugin struct {
	// engineCache caches template engines to avoid re-parsing templates for each feed
	engineMu    sync.RWMutex
	engineCache map[string]*templates.Engine
}

// NewPublishFeedsPlugin creates a new PublishFeedsPlugin.
func NewPublishFeedsPlugin() *PublishFeedsPlugin {
	return &PublishFeedsPlugin{
		engineCache: make(map[string]*templates.Engine),
	}
}

// Name returns the unique name of the plugin.
func (p *PublishFeedsPlugin) Name() string {
	return "publish_feeds"
}

// getOrCreateEngine returns a cached template engine, or creates one if not cached.
func (p *PublishFeedsPlugin) getOrCreateEngine(templatesDir, themeName string) (*templates.Engine, error) {
	cacheKey := templatesDir + ":" + themeName

	// Fast path: check cache with read lock
	p.engineMu.RLock()
	if engine, ok := p.engineCache[cacheKey]; ok {
		p.engineMu.RUnlock()
		return engine, nil
	}
	p.engineMu.RUnlock()

	// Slow path: create engine with write lock
	p.engineMu.Lock()
	defer p.engineMu.Unlock()

	// Double-check after acquiring write lock
	if engine, ok := p.engineCache[cacheKey]; ok {
		return engine, nil
	}

	engine, err := templates.NewEngineWithTheme(templatesDir, themeName)
	if err != nil {
		return nil, err
	}

	p.engineCache[cacheKey] = engine
	return engine, nil
}

// Configure registers synthetic posts for feed pages so they can be resolved by wikilinks.
// These posts are marked with Skip: true so they don't interfere with normal rendering.
func (p *PublishFeedsPlugin) Configure(m *lifecycle.Manager) error {
	// Get feed configs from cache (set by FeedsPlugin)
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	if len(feedConfigs) == 0 {
		return nil
	}

	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	// Register synthetic post for each feed
	for i := range feedConfigs {
		fc := &feedConfigs[i]

		// Determine title
		title := fc.Title
		if title == "" {
			title = fc.Slug
		}

		// Create synthetic post
		feedPost := &models.Post{
			Slug:        fc.Slug,
			Title:       strPtr(title),
			Description: strPtr(fc.Description),
			Href:        "/" + fc.Slug + "/",
			Published:   true,
			Skip:        true,
		}
		m.AddPost(feedPost)
	}

	return nil
}

// Write generates and writes feed files in all configured formats.
// Feed generation is parallelized for better performance with many feeds.
// Uses incremental build cache to skip feeds with unchanged content.
func (p *PublishFeedsPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Get feed configs from cache
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	if len(feedConfigs) == 0 {
		return nil
	}

	// Copy XSL stylesheets to output directory for styled RSS/Atom feeds
	if err := p.copyXSLStylesheets(config, outputDir); err != nil {
		return fmt.Errorf("copying XSL stylesheets: %w", err)
	}

	// Get build cache for incremental builds
	buildCache := GetBuildCache(m)
	var changedSlugs map[string]bool
	if buildCache != nil {
		changedSlugs = make(map[string]bool)
		for _, slug := range buildCache.GetChangedSlugs() {
			changedSlugs[slug] = true
		}
	}

	// Track skipped feeds
	var skippedCount int
	var rebuiltCount int

	// Process feeds concurrently with a worker pool
	// Limit concurrency to avoid overwhelming the system
	const maxConcurrency = 8
	numFeeds := len(feedConfigs)

	// For small numbers of feeds, just process sequentially
	if numFeeds <= 2 {
		for i := range feedConfigs {
			fc := &feedConfigs[i]
			skip, hash := p.shouldSkipFeed(fc, buildCache, changedSlugs, outputDir)
			if skip {
				skippedCount++
				continue
			}
			if err := p.publishFeed(fc, config, outputDir); err != nil {
				return fmt.Errorf("publishing feed %q: %w", fc.Slug, err)
			}
			p.cacheFeedHash(fc, buildCache, hash)
			rebuiltCount++
		}
		return nil
	}

	// Use a semaphore to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	errChan := make(chan error, numFeeds)
	var wg sync.WaitGroup
	var countMu sync.Mutex

	for i := range feedConfigs {
		fc := &feedConfigs[i]
		skip, hash := p.shouldSkipFeed(fc, buildCache, changedSlugs, outputDir)
		if skip {
			skippedCount++
			continue
		}
		wg.Add(1)
		go func(fc *models.FeedConfig, hash string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if err := p.publishFeed(fc, config, outputDir); err != nil {
				errChan <- fmt.Errorf("publishing feed %q: %w", fc.Slug, err)
				return
			}
			p.cacheFeedHash(fc, buildCache, hash)
			countMu.Lock()
			rebuiltCount++
			countMu.Unlock()
		}(fc, hash)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Log incremental stats if any feeds were skipped
	if skippedCount > 0 {
		log.Printf("[publish_feeds] Incremental: %d feeds skipped, %d rebuilt", skippedCount, rebuiltCount)
	}

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// computeFeedHash computes a hash of the feed's content (post slugs and dates).
func (p *PublishFeedsPlugin) computeFeedHash(fc *models.FeedConfig) string {
	h := sha256.New()

	// Hash the sorted post slugs to detect membership changes
	slugs := make([]string, 0, len(fc.Posts))
	for _, post := range fc.Posts {
		slugs = append(slugs, post.Slug)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		h.Write([]byte(slug))
		h.Write([]byte{0}) // separator
	}

	// Also hash the feed config that affects output
	h.Write([]byte(fc.Slug))
	h.Write([]byte(fc.Title))
	fmt.Fprintf(h, "%d", fc.ItemsPerPage)

	return hex.EncodeToString(h.Sum(nil))
}

// shouldSkipFeed checks if a feed can be skipped (incremental build).
// Returns (skip bool, hash string) - hash is returned so callers can reuse it
// for caching without recomputing.
func (p *PublishFeedsPlugin) shouldSkipFeed(fc *models.FeedConfig, cache interface{}, changedSlugs map[string]bool, outputDir string) (skip bool, hash string) {
	// Always compute hash since we return it for caching
	currentHash := p.computeFeedHash(fc)

	if cache == nil {
		return false, currentHash
	}

	bc, ok := cache.(*buildcache.Cache)
	if !ok || bc == nil {
		return false, currentHash
	}

	// Check if any post in this feed has changed
	if len(changedSlugs) > 0 {
		for _, post := range fc.Posts {
			if changedSlugs[post.Slug] {
				return false, currentHash // Need to rebuild
			}
		}
	}

	// Check if feed hash changed (post list membership)
	cachedHash := bc.GetFeedHash(fc.Slug)
	if cachedHash != currentHash {
		return false, currentHash // Need to rebuild
	}

	// Check if output files exist
	feedDir := p.determineFeedDir(outputDir, fc.Slug)
	indexPath := filepath.Join(feedDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return false, currentHash // Need to rebuild
	}

	return true, currentHash // Can skip
}

// cacheFeedHash stores the feed hash in the build cache.
func (p *PublishFeedsPlugin) cacheFeedHash(fc *models.FeedConfig, cache interface{}, hash string) {
	if cache == nil {
		return
	}
	bc, ok := cache.(*buildcache.Cache)
	if !ok || bc == nil {
		return
	}
	bc.SetFeedHash(fc.Slug, hash)
}

// feedFormatPublisher defines how to publish a specific feed format.
type feedFormatPublisher struct {
	name       string // Format name for error messages
	enabled    bool   // Whether this format is enabled
	publish    func() error
	ext        string // File extension for redirect (empty if no redirect needed)
	targetFile string // Target file name for redirect
}

// publishFeed publishes a single feed in all configured formats.
func (p *PublishFeedsPlugin) publishFeed(fc *models.FeedConfig, config *lifecycle.Config, outputDir string) error {
	feedDir := p.determineFeedDir(outputDir, fc.Slug)

	if err := os.MkdirAll(feedDir, 0o755); err != nil {
		return fmt.Errorf("creating feed directory: %w", err)
	}

	// Define all format publishers with their configurations
	publishers := []feedFormatPublisher{
		{name: "HTML", enabled: fc.Formats.HTML, publish: func() error { return p.publishHTMLPages(fc, config, feedDir) }},
		{name: "RSS", enabled: fc.Formats.RSS, publish: func() error { return p.publishRSS(fc, config, feedDir) }},
		{name: "Atom", enabled: fc.Formats.Atom, publish: func() error { return p.publishAtom(fc, config, feedDir) }},
		{name: "JSON", enabled: fc.Formats.JSON, publish: func() error { return p.publishJSON(fc, config, feedDir) }, ext: "json", targetFile: "feed.json"},
		{name: "Markdown", enabled: fc.Formats.Markdown, publish: func() error { return p.publishMarkdown(fc, fc.Slug, outputDir) }, ext: "md", targetFile: ""},
		{name: "Text", enabled: fc.Formats.Text, publish: func() error { return p.publishText(fc, fc.Slug, outputDir) }, ext: "txt", targetFile: ""},
		{name: "Sitemap", enabled: fc.Formats.Sitemap, publish: func() error { return p.publishSitemap(fc, config, feedDir) }},
	}

	for _, pub := range publishers {
		if err := p.publishFormat(pub, fc.Slug, outputDir); err != nil {
			return err
		}
	}

	return nil
}

// determineFeedDir returns the output directory for a feed based on its slug.
func (p *PublishFeedsPlugin) determineFeedDir(outputDir, slug string) string {
	if slug == "" {
		return outputDir
	}
	return filepath.Join(outputDir, slug)
}

// safeWriteFile writes content to a file, removing any existing directory at that path.
// This handles the case where a previous build created a directory where a file should be.
func (p *PublishFeedsPlugin) safeWriteFile(path string, content []byte) error {
	// Check if path exists as a directory
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// Remove the directory so we can write a file
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing directory at file path %s: %w", path, err)
		}
	}

	//nolint:gosec // G306: Output files need 0644 for web serving
	return os.WriteFile(path, content, 0o644)
}

// publishFormat publishes a single format if enabled and handles redirects.
func (p *PublishFeedsPlugin) publishFormat(pub feedFormatPublisher, slug, outputDir string) error {
	if !pub.enabled {
		return nil
	}

	if err := pub.publish(); err != nil {
		return fmt.Errorf("publishing %s: %w", pub.name, err)
	}

	// Write redirect for non-root feeds with redirect configuration
	if slug != "" && pub.ext != "" {
		if pub.targetFile == "" {
			// Reversed redirect: /slug/index.ext -> /slug.ext (for Markdown/Text)
			if err := p.writeReversedFeedRedirect(slug, pub.ext, outputDir); err != nil {
				return fmt.Errorf("writing %s redirect: %w", pub.name, err)
			}
		} else {
			// Forward redirect: /slug.ext -> /slug/targetFile (for JSON)
			if err := p.writeFeedFormatRedirect(slug, pub.ext, pub.targetFile, outputDir); err != nil {
				return fmt.Errorf("writing %s redirect: %w", pub.name, err)
			}
		}
	}

	return nil
}

// publishHTMLPages publishes HTML pages for a paginated feed.
func (p *PublishFeedsPlugin) publishHTMLPages(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	for i := range fc.Pages {
		page := &fc.Pages[i]
		// Determine output path
		var pagePath string
		if page.Number == 1 {
			pagePath = filepath.Join(feedDir, "index.html")
		} else {
			pageDir := filepath.Join(feedDir, "page", fmt.Sprintf("%d", page.Number))
			if err := os.MkdirAll(pageDir, 0o755); err != nil {
				return fmt.Errorf("creating page directory: %w", err)
			}
			pagePath = filepath.Join(pageDir, "index.html")
		}

		// Generate HTML content
		html, err := p.generateFeedPageHTML(fc, page, config)
		if err != nil {
			return fmt.Errorf("generating page %d: %w", page.Number, err)
		}

		// Write file
		//nolint:gosec // G306: HTML output files need 0644 for web serving
		if err := os.WriteFile(pagePath, []byte(html), 0o644); err != nil {
			return fmt.Errorf("writing page %d: %w", page.Number, err)
		}
	}

	return nil
}

// generateFeedPageHTML generates HTML for a feed page.
func (p *PublishFeedsPlugin) generateFeedPageHTML(fc *models.FeedConfig, page *models.FeedPage, config *lifecycle.Config) (string, error) {
	// Get templates directory from config
	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	// Get theme name from config (default to "default")
	themeName := ThemeDefault
	if extra := config.Extra; extra != nil {
		// Check for typed ThemeConfig struct (set by core.go)
		if theme, ok := extra["theme"].(models.ThemeConfig); ok {
			if theme.Name != "" {
				themeName = theme.Name
			}
		}
		// Also check for map[string]interface{} (legacy/dynamic config)
		if theme, ok := extra["theme"].(map[string]interface{}); ok {
			if name, ok := theme["name"].(string); ok && name != "" {
				themeName = name
			}
		}
		// Also check for simple theme string
		if name, ok := extra["theme"].(string); ok && name != "" {
			themeName = name
		}
	}

	// Try to use pongo2 template engine with feed.html template (cached)
	engine, err := p.getOrCreateEngine(templatesDir, themeName)
	if err == nil && engine.TemplateExists("feed.html") {
		// Use shared config conversion to ensure all fields are available
		// (search, head, components, theme, etc. - required by base.html)
		modelsConfig := ToModelsConfig(config)

		// Create feed context
		ctx := templates.NewFeedContext(fc, page, modelsConfig)

		// Render with pongo2 template
		html, err := engine.Render("feed.html", ctx)
		if err != nil {
			// Log template rendering errors to help debug issues
			log.Printf("[publish_feeds] Warning: template rendering failed for feed.html: %v (falling back to built-in template)", err)
		} else {
			return html, nil
		}
	}

	// Fallback: Use built-in Go template
	return p.generateFeedPageHTMLFallback(fc, page, config)
}

// generateFeedPageHTMLFallback generates HTML using a built-in Go template.
func (p *PublishFeedsPlugin) generateFeedPageHTMLFallback(fc *models.FeedConfig, page *models.FeedPage, config *lifecycle.Config) (string, error) {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)

	title := fc.Title
	if title == "" {
		title = siteTitle
	}

	// Simple default template with CSS links
	tmplStr := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/css/variables.css">
    <link rel="stylesheet" href="/css/main.css">
    <link rel="stylesheet" href="/css/admonitions.css">
    <link rel="stylesheet" href="/css/code.css">
    <link rel="alternate" type="application/rss+xml" title="RSS Feed" href="rss.xml">
    <link rel="alternate" type="application/atom+xml" title="Atom Feed" href="atom.xml">
    <link rel="alternate" type="application/feed+json" title="JSON Feed" href="feed.json">
</head>
<body>
    <header>
        <nav>
            <a href="/">{{.SiteTitle}}</a>
            <a href="/blog/">Blog</a>
        </nav>
    </header>
    <main>
        <h1>{{.Title}}</h1>
        {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
        <div class="post-list">
        {{range .Posts}}
            <article class="post-card">
                <a href="{{.Href}}">
                    <h2>{{if .Title}}{{.Title}}{{else}}{{.Slug}}{{end}}</h2>
                </a>
                {{if .Date}}<time datetime="{{.Date.Format "2006-01-02"}}">{{.Date.Format "January 2, 2006"}}</time>{{end}}
                {{if .Description}}<p>{{.Description}}</p>{{end}}
            </article>
        {{end}}
        </div>
    </main>
    <nav class="pagination">
        {{if .HasPrev}}<a href="{{.PrevURL}}" rel="prev">&laquo; Newer</a>{{end}}
        <span>Page {{.Number}}</span>
        {{if .HasNext}}<a href="{{.NextURL}}" rel="next">Older &raquo;</a>{{end}}
    </nav>
    <footer>
        <p>&copy; {{.SiteTitle}}</p>
    </footer>
</body>
</html>`

	tmpl, err := template.New("feed").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// Prepare template data
	data := struct {
		Title       string
		Description string
		Posts       []*models.Post
		HasPrev     bool
		HasNext     bool
		PrevURL     string
		NextURL     string
		Number      int
		SiteURL     string
		SiteTitle   string
	}{
		Title:       title,
		Description: fc.Description,
		Posts:       page.Posts,
		HasPrev:     page.HasPrev,
		HasNext:     page.HasNext,
		PrevURL:     page.PrevURL,
		NextURL:     page.NextURL,
		Number:      page.Number,
		SiteURL:     siteURL,
		SiteTitle:   siteTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// publishRSS generates and writes an RSS feed.
func (p *PublishFeedsPlugin) publishRSS(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	rss, err := GenerateRSSFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	rssPath := filepath.Join(feedDir, "rss.xml")
	return p.safeWriteFile(rssPath, []byte(rss))
}

// publishAtom generates and writes an Atom feed.
func (p *PublishFeedsPlugin) publishAtom(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	atom, err := GenerateAtomFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	atomPath := filepath.Join(feedDir, "atom.xml")
	return p.safeWriteFile(atomPath, []byte(atom))
}

// publishJSON generates and writes a JSON feed.
func (p *PublishFeedsPlugin) publishJSON(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	jsonFeed, err := GenerateJSONFeedFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(feedDir, "feed.json")
	return p.safeWriteFile(jsonPath, []byte(jsonFeed))
}

// publishMarkdown generates and writes a Markdown feed listing.
// For non-root feeds, content is written to /slug.md (canonical URL).
// For root feeds (slug=""), content is written to /index.md.
// HTML entities in titles and descriptions are decoded to readable characters.
func (p *PublishFeedsPlugin) publishMarkdown(fc *models.FeedConfig, slug, outputDir string) error {
	var sb strings.Builder

	// Title (decode HTML entities for plain text output)
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	sb.WriteString("# " + html.UnescapeString(title) + "\n\n")

	// Description (decode HTML entities)
	if fc.Description != "" {
		sb.WriteString(html.UnescapeString(fc.Description) + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = html.UnescapeString(*post.Title)
		}

		sb.WriteString("- [" + postTitle + "](" + post.Href + ")")

		if post.Date != nil {
			sb.WriteString(" - " + post.Date.Format("2006-01-02"))
		}

		sb.WriteString("\n")
	}

	// Determine output path: /slug.md for non-root, /index.md for root
	var mdPath string
	if slug == "" {
		mdPath = filepath.Join(outputDir, "index.md")
	} else {
		mdPath = filepath.Join(outputDir, slug+".md")
	}
	return p.safeWriteFile(mdPath, []byte(sb.String()))
}

// publishText generates and writes a plain text feed listing.
// For non-root feeds, content is written to /slug.txt (canonical URL).
// For root feeds (slug=""), content is written to /index.txt.
// HTML entities in titles and descriptions are decoded to readable characters.
func (p *PublishFeedsPlugin) publishText(fc *models.FeedConfig, slug, outputDir string) error {
	var sb strings.Builder

	// Title (decode HTML entities for plain text output)
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	title = html.UnescapeString(title)
	sb.WriteString(title + "\n")
	sb.WriteString(strings.Repeat("=", len(title)) + "\n\n")

	// Description (decode HTML entities)
	if fc.Description != "" {
		sb.WriteString(html.UnescapeString(fc.Description) + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = html.UnescapeString(*post.Title)
		}

		if post.Date != nil {
			sb.WriteString(post.Date.Format("2006-01-02") + " - ")
		}

		sb.WriteString(postTitle + "\n")
		sb.WriteString("  " + post.Href + "\n\n")
	}

	// Determine output path: /slug.txt for non-root, /index.txt for root
	var txtPath string
	if slug == "" {
		txtPath = filepath.Join(outputDir, "index.txt")
	} else {
		txtPath = filepath.Join(outputDir, slug+".txt")
	}
	return p.safeWriteFile(txtPath, []byte(sb.String()))
}

// publishSitemap generates and writes a sitemap XML file for feed posts.
func (p *PublishFeedsPlugin) publishSitemap(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	// Get site URL
	siteURL := getSiteURL(config)
	if siteURL == "" {
		siteURL = DefaultSiteURL
	}

	// Build sitemap for this feed's posts
	sitemap := &URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]SitemapURL, 0, len(fc.Posts)),
	}

	// Add all posts in this feed
	for _, post := range fc.Posts {
		if !post.Published || post.Draft || post.Skip || post.Private {
			continue
		}

		url := SitemapURL{
			Loc: siteURL + post.Href,
		}

		// Use post.date for lastmod
		if post.Date != nil {
			url.LastMod = post.Date.Format("2006-01-02")
		}

		// Support frontmatter fields: changefreq and priority
		// Default values from spec
		changefreq := "weekly"
		priority := "0.5"

		if post.Extra != nil {
			if cf, ok := post.Extra["changefreq"].(string); ok && cf != "" {
				changefreq = cf
			}
			if p, ok := post.Extra["priority"].(string); ok && p != "" {
				priority = p
			}
			// Also support float64 for priority from TOML/JSON parsing
			if p, ok := post.Extra["priority"].(float64); ok {
				priority = fmt.Sprintf("%.1f", p)
			}
		}

		url.ChangeFreq = changefreq
		url.Priority = priority

		sitemap.URLs = append(sitemap.URLs, url)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(sitemap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshaling sitemap: %w", err)
	}

	// Add XML declaration
	xmlContent := xml.Header + string(output)

	// Write sitemap.xml
	sitemapPath := filepath.Join(feedDir, "sitemap.xml")
	return p.safeWriteFile(sitemapPath, []byte(xmlContent))
}

// writeFeedFormatRedirect writes a redirect from /slug.ext to /slug/targetFile.
// This creates a file at slug.ext/index.html, which allows the URL /slug.ext
// (without trailing slash) to serve the HTML redirect on most static hosts.
//
// For example, requesting /archive.json will serve the redirect HTML that
// points to /archive/feed.json where the actual JSON content lives.
//
// Note: Web servers serve slug.ext/index.html when /slug.ext is requested,
// without adding a trailing slash redirect (unlike directory-only approaches).
func (p *PublishFeedsPlugin) writeFeedFormatRedirect(slug, ext, targetFile, outputDir string) error {
	// Create redirect HTML that points to the actual file
	targetURL := fmt.Sprintf("/%s/%s", slug, targetFile)
	redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
<title>Redirecting...</title>
</head>
<body>
<p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>`, targetURL, targetURL, targetURL, targetURL)

	// Create directory at /slug.ext/ (e.g., /archive.md/)
	redirectDir := filepath.Join(outputDir, slug+"."+ext)
	if err := os.MkdirAll(redirectDir, 0o755); err != nil {
		return fmt.Errorf("creating redirect directory %s: %w", redirectDir, err)
	}

	// Write index.html inside the directory
	// This allows /slug.ext to be served without trailing slash on most static hosts
	outputPath := filepath.Join(redirectDir, "index.html")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing redirect %s: %w", outputPath, err)
	}

	return nil
}

// writeReversedFeedRedirect writes a redirect from /slug/index.ext to /slug.ext.
// This is the "reversed" direction from writeFeedFormatRedirect - content is at the
// short URL (/slug.ext) and the redirect points there from the long URL (/slug/index.ext).
//
// Creates a file at /slug/index.ext/index.html, which allows the URL /slug/index.ext
// (without trailing slash) to serve the HTML redirect on most static hosts.
//
// For example, requesting /archive/index.md will serve the redirect HTML that
// points to /archive.md where the actual markdown content lives.
func (p *PublishFeedsPlugin) writeReversedFeedRedirect(slug, ext, outputDir string) error {
	// Create redirect HTML that points to the canonical short URL
	targetURL := fmt.Sprintf("/%s.%s", slug, ext)
	redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
<title>Redirecting...</title>
</head>
<body>
<p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>`, targetURL, targetURL, targetURL, targetURL)

	// Create directory at /slug/index.ext/ (e.g., /archive/index.md/)
	redirectDir := filepath.Join(outputDir, slug, "index."+ext)
	if err := os.MkdirAll(redirectDir, 0o755); err != nil {
		return fmt.Errorf("creating redirect directory %s: %w", redirectDir, err)
	}

	// Write index.html inside the directory
	// This allows /slug/index.ext to be served without trailing slash on most static hosts
	outputPath := filepath.Join(redirectDir, "index.html")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing redirect %s: %w", outputPath, err)
	}

	return nil
}

// copyXSLStylesheets copies XSL stylesheets to the output directory for styled RSS/Atom feeds.
// It searches for XSL files in the following order:
// 1. User's templates directory (if configured)
// 2. Embedded default theme templates (fallback)
func (p *PublishFeedsPlugin) copyXSLStylesheets(config *lifecycle.Config, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Get templates directory from config
	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	// List of XSL files to copy
	xslFiles := []string{"rss.xsl", "atom.xsl"}

	for _, xslFile := range xslFiles {
		var content []byte
		var err error

		// First, try to read from user's templates directory
		srcPath := filepath.Join(templatesDir, xslFile)
		if _, statErr := os.Stat(srcPath); statErr == nil {
			// User has their own XSL file, use it
			content, err = os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("reading XSL file %s: %w", srcPath, err)
			}
		} else if os.IsNotExist(statErr) {
			// No user file, try embedded templates as fallback
			content, err = themes.ReadTemplate(xslFile)
			if err != nil {
				// XSL file doesn't exist in embedded templates either, skip it
				continue
			}
		} else {
			return fmt.Errorf("checking XSL file %s: %w", srcPath, statErr)
		}

		// Write to output directory
		dstPath := filepath.Join(outputDir, xslFile)
		//nolint:gosec // G306: XSL files need 0644 for web serving
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return fmt.Errorf("writing XSL file %s: %w", dstPath, err)
		}
	}

	return nil
}

// Ensure PublishFeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*PublishFeedsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*PublishFeedsPlugin)(nil)
	_ lifecycle.WritePlugin     = (*PublishFeedsPlugin)(nil)
)
