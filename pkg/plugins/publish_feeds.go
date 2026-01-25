// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// PublishFeedsPlugin writes feeds to multiple output formats during the write stage.
type PublishFeedsPlugin struct{}

// NewPublishFeedsPlugin creates a new PublishFeedsPlugin.
func NewPublishFeedsPlugin() *PublishFeedsPlugin {
	return &PublishFeedsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PublishFeedsPlugin) Name() string {
	return "publish_feeds"
}

// Write generates and writes feed files in all configured formats.
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

	for i := range feedConfigs {
		if err := p.publishFeed(&feedConfigs[i], config, outputDir); err != nil {
			return fmt.Errorf("publishing feed %q: %w", feedConfigs[i].Slug, err)
		}
	}

	return nil
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
		{name: "Markdown", enabled: fc.Formats.Markdown, publish: func() error { return p.publishMarkdown(fc, feedDir) }, ext: "md", targetFile: "index.md"},
		{name: "Text", enabled: fc.Formats.Text, publish: func() error { return p.publishText(fc, feedDir) }, ext: "txt", targetFile: "index.txt"},
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
		if err := p.writeFeedFormatRedirect(slug, pub.ext, pub.targetFile, outputDir); err != nil {
			return fmt.Errorf("writing %s redirect: %w", pub.name, err)
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
	themeName := "default"
	if extra := config.Extra; extra != nil {
		if theme, ok := extra["theme"].(map[string]interface{}); ok {
			if name, ok := theme["name"].(string); ok && name != "" {
				themeName = name
			}
		}
		if name, ok := extra["theme"].(string); ok && name != "" {
			themeName = name
		}
	}

	// Try to use pongo2 template engine with feed.html template
	engine, err := templates.NewEngineWithTheme(templatesDir, themeName)
	if err == nil && engine.TemplateExists("feed.html") {
		// Build config for template context
		modelsConfig := &models.Config{
			OutputDir:   config.OutputDir,
			Title:       getStringFromExtra(config.Extra, "title"),
			URL:         getStringFromExtra(config.Extra, "url"),
			Description: getStringFromExtra(config.Extra, "description"),
			Author:      getStringFromExtra(config.Extra, "author"),
		}

		// Copy nav items if available
		if navItems, ok := config.Extra["nav"].([]models.NavItem); ok {
			modelsConfig.Nav = navItems
		}

		// Copy footer config if available
		if footer, ok := config.Extra["footer"].(models.FooterConfig); ok {
			modelsConfig.Footer = footer
		}

		// Create feed context
		ctx := templates.NewFeedContext(fc, page, modelsConfig)

		// Render with pongo2 template
		html, err := engine.Render("feed.html", ctx)
		if err == nil {
			return html, nil
		}
		// If pongo2 rendering fails, fall back to built-in template
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
	//nolint:gosec // G306: Feed files need 0644 for web serving
	return os.WriteFile(rssPath, []byte(rss), 0o644)
}

// publishAtom generates and writes an Atom feed.
func (p *PublishFeedsPlugin) publishAtom(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	atom, err := GenerateAtomFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	atomPath := filepath.Join(feedDir, "atom.xml")
	//nolint:gosec // G306: Feed files need 0644 for web serving
	return os.WriteFile(atomPath, []byte(atom), 0o644)
}

// publishJSON generates and writes a JSON feed.
func (p *PublishFeedsPlugin) publishJSON(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	jsonFeed, err := GenerateJSONFeedFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(feedDir, "feed.json")
	//nolint:gosec // G306: Feed files need 0644 for web serving
	return os.WriteFile(jsonPath, []byte(jsonFeed), 0o644)
}

// publishMarkdown generates and writes a Markdown feed listing.
func (p *PublishFeedsPlugin) publishMarkdown(fc *models.FeedConfig, feedDir string) error {
	var sb strings.Builder

	// Title
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	sb.WriteString("# " + title + "\n\n")

	// Description
	if fc.Description != "" {
		sb.WriteString(fc.Description + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = *post.Title
		}

		sb.WriteString("- [" + postTitle + "](" + post.Href + ")")

		if post.Date != nil {
			sb.WriteString(" - " + post.Date.Format("2006-01-02"))
		}

		sb.WriteString("\n")
	}

	mdPath := filepath.Join(feedDir, "index.md")
	//nolint:gosec // G306: Feed files need 0644 for web serving
	return os.WriteFile(mdPath, []byte(sb.String()), 0o644)
}

// publishText generates and writes a plain text feed listing.
func (p *PublishFeedsPlugin) publishText(fc *models.FeedConfig, feedDir string) error {
	var sb strings.Builder

	// Title
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	sb.WriteString(title + "\n")
	sb.WriteString(strings.Repeat("=", len(title)) + "\n\n")

	// Description
	if fc.Description != "" {
		sb.WriteString(fc.Description + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = *post.Title
		}

		if post.Date != nil {
			sb.WriteString(post.Date.Format("2006-01-02") + " - ")
		}

		sb.WriteString(postTitle + "\n")
		sb.WriteString("  " + post.Href + "\n\n")
	}

	txtPath := filepath.Join(feedDir, "index.txt")
	//nolint:gosec // G306: Feed files need 0644 for web serving
	return os.WriteFile(txtPath, []byte(sb.String()), 0o644)
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
		if !post.Published || post.Draft || post.Skip {
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
	//nolint:gosec // G306: Sitemap files need 0644 for web serving
	return os.WriteFile(sitemapPath, []byte(xmlContent), 0o644)
}

// writeFeedFormatRedirect writes a redirect from /slug.ext to /slug/targetFile.
// This creates a file at slug.ext/index.html, which allows the URL /slug.ext
// (without trailing slash) to serve the HTML redirect on most static hosts.
//
// For example, requesting /archive.md will serve the redirect HTML that
// points to /archive/index.md where the actual markdown content lives.
// Similarly, /archive.json redirects to /archive/feed.json.
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

// Ensure PublishFeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin      = (*PublishFeedsPlugin)(nil)
	_ lifecycle.WritePlugin = (*PublishFeedsPlugin)(nil)
)
