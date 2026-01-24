// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// categoryUncategorized is the default category name for feeds without a category.
const categoryUncategorized = "Uncategorized"

// BlogrollPlugin fetches and processes external RSS/Atom feeds.
// It runs in the Collect stage to gather external feed entries
// and in the Write stage to generate blogroll and reader pages.
type BlogrollPlugin struct {
	feeds   []*models.ExternalFeed
	entries []*models.ExternalEntry
	mu      sync.RWMutex
}

// NewBlogrollPlugin creates a new BlogrollPlugin.
func NewBlogrollPlugin() *BlogrollPlugin {
	return &BlogrollPlugin{
		feeds:   make([]*models.ExternalFeed, 0),
		entries: make([]*models.ExternalEntry, 0),
	}
}

// Name returns the unique name of the plugin.
func (p *BlogrollPlugin) Name() string {
	return "blogroll"
}

// Priority returns the plugin's priority for a given stage.
func (p *BlogrollPlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageCollect:
		// Run after feeds plugin to not interfere
		return lifecycle.PriorityLate + 10
	case lifecycle.StageWrite:
		// Run after publish_feeds to write blogroll pages
		return lifecycle.PriorityLate + 20
	default:
		return lifecycle.PriorityDefault
	}
}

// Collect fetches and parses configured external feeds.
func (p *BlogrollPlugin) Collect(m *lifecycle.Manager) error {
	config := m.Config()
	blogrollConfig := getBlogrollConfig(config)

	if !blogrollConfig.Enabled {
		return nil
	}

	// Fetch all feeds concurrently
	feeds, entries, err := p.fetchFeeds(blogrollConfig)
	if err != nil {
		return fmt.Errorf("blogroll: %w", err)
	}

	p.mu.Lock()
	p.feeds = feeds
	p.entries = entries
	p.mu.Unlock()

	// Store in cache for templates
	m.Cache().Set("blogroll_feeds", feeds)
	m.Cache().Set("blogroll_entries", entries)
	m.Cache().Set("blogroll_categories", p.groupByCategory(feeds))

	return nil
}

// Write generates the blogroll and reader pages.
func (p *BlogrollPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	blogrollConfig := getBlogrollConfig(config)

	if !blogrollConfig.Enabled {
		return nil
	}

	p.mu.RLock()
	feeds := p.feeds
	entries := p.entries
	p.mu.RUnlock()

	if len(feeds) == 0 {
		return nil
	}

	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	// Generate blogroll page
	if err := p.writeBlogrollPage(m, outputDir, feeds, blogrollConfig); err != nil {
		return fmt.Errorf("blogroll page: %w", err)
	}

	// Generate reader page
	if err := p.writeReaderPage(m, outputDir, entries, blogrollConfig); err != nil {
		return fmt.Errorf("reader page: %w", err)
	}

	return nil
}

// fetchFeeds fetches all configured feeds concurrently.
func (p *BlogrollPlugin) fetchFeeds(config models.BlogrollConfig) ([]*models.ExternalFeed, []*models.ExternalEntry, error) {
	var activeFeeds []models.ExternalFeedConfig
	for i := range config.Feeds {
		if config.Feeds[i].IsActive() {
			activeFeeds = append(activeFeeds, config.Feeds[i])
		}
	}

	if len(activeFeeds) == 0 {
		return nil, nil, nil
	}

	// Parse cache duration
	cacheDuration, err := time.ParseDuration(config.CacheDuration)
	if err != nil {
		cacheDuration = time.Hour
	}

	// Create cache directory
	if config.CacheDir != "" {
		if err := os.MkdirAll(config.CacheDir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create cache dir: %w", err)
		}
	}

	// Fetch feeds concurrently
	concurrency := config.ConcurrentRequests
	if concurrency <= 0 {
		concurrency = 5
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	maxEntries := config.MaxEntriesPerFeed
	if maxEntries <= 0 {
		maxEntries = 50
	}

	semaphore := make(chan struct{}, concurrency)
	resultsCh := make(chan *models.ExternalFeed, len(activeFeeds))
	var wg sync.WaitGroup

	for i := range activeFeeds {
		wg.Add(1)
		go func(feedConfig models.ExternalFeedConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			feed := p.fetchFeed(feedConfig, config.CacheDir, cacheDuration, timeout, maxEntries)
			resultsCh <- feed
		}(activeFeeds[i])
	}

	wg.Wait()
	close(resultsCh)

	// Collect results
	feeds := make([]*models.ExternalFeed, 0, len(activeFeeds))
	var allEntries []*models.ExternalEntry
	for feed := range resultsCh {
		feeds = append(feeds, feed)
		allEntries = append(allEntries, feed.Entries...)
	}

	// Sort feeds by title
	sort.Slice(feeds, func(i, j int) bool {
		return strings.ToLower(feeds[i].Title) < strings.ToLower(feeds[j].Title)
	})

	// Sort entries by published date (newest first)
	sort.Slice(allEntries, func(i, j int) bool {
		ti := allEntries[i].Published
		tj := allEntries[j].Published
		if ti == nil && tj == nil {
			return false
		}
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.After(*tj)
	})

	return feeds, allEntries, nil
}

// initFeedFromConfig creates a new ExternalFeed initialized from configuration.
func initFeedFromConfig(config models.ExternalFeedConfig) *models.ExternalFeed {
	feed := &models.ExternalFeed{
		Config:   config,
		FeedURL:  config.URL,
		Category: config.Category,
		Tags:     config.Tags,
		Entries:  make([]*models.ExternalEntry, 0),
	}

	// Use config values if set
	if config.Title != "" {
		feed.Title = config.Title
	}
	if config.Description != "" {
		feed.Description = config.Description
	}
	if config.SiteURL != "" {
		feed.SiteURL = config.SiteURL
	}
	if config.ImageURL != "" {
		feed.ImageURL = config.ImageURL
	}

	return feed
}

// mergeCachedFeed merges config values into a cached feed.
func mergeCachedFeed(cached *models.ExternalFeed, config models.ExternalFeedConfig) *models.ExternalFeed {
	if config.Title != "" {
		cached.Title = config.Title
	}
	if config.Category != "" {
		cached.Category = config.Category
	}
	cached.Tags = config.Tags
	return cached
}

// updateFeedFromParsed updates feed metadata from parsed feed data.
func updateFeedFromParsed(feed *models.ExternalFeed, parsed *parsedFeed) {
	if feed.Title == "" {
		feed.Title = parsed.Title
	}
	if feed.Description == "" {
		feed.Description = parsed.Description
	}
	if feed.SiteURL == "" {
		feed.SiteURL = parsed.SiteURL
	}
	if feed.ImageURL == "" {
		feed.ImageURL = parsed.ImageURL
	}

	now := time.Now()
	feed.LastFetched = &now
	if parsed.LastUpdated != nil {
		feed.LastUpdated = parsed.LastUpdated
	}
}

// fetchFeed fetches a single feed with caching.
func (p *BlogrollPlugin) fetchFeed(config models.ExternalFeedConfig, cacheDir string, cacheDuration time.Duration, timeout, maxEntries int) *models.ExternalFeed {
	feed := initFeedFromConfig(config)

	// Check cache
	if cacheDir != "" {
		if cached := p.loadFromCache(config.URL, cacheDir, cacheDuration); cached != nil {
			return mergeCachedFeed(cached, config)
		}
	}

	// Fetch the feed
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, http.NoBody)
	if err != nil {
		feed.Error = fmt.Sprintf("create request: %v", err)
		return feed
	}

	req.Header.Set("User-Agent", "markata-go/1.0 (RSS Reader)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		feed.Error = fmt.Sprintf("fetch: %v", err)
		return feed
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		feed.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return feed
	}

	// Parse the feed using simple XML parsing
	parsedFeed, entries, err := parseFeedResponse(resp)
	if err != nil {
		feed.Error = fmt.Sprintf("parse: %v", err)
		return feed
	}

	// Update feed with parsed values
	updateFeedFromParsed(feed, parsedFeed)

	// Limit entries
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	// Add feed info to entries
	for _, entry := range entries {
		entry.FeedURL = config.URL
		entry.FeedTitle = feed.Title
	}

	feed.Entries = entries
	feed.EntryCount = len(entries)

	// Save to cache
	if cacheDir != "" {
		p.saveToCache(feed, cacheDir)
	}

	return feed
}

// loadFromCache loads a feed from cache if valid.
func (p *BlogrollPlugin) loadFromCache(url, cacheDir string, maxAge time.Duration) *models.ExternalFeed {
	cacheFile := filepath.Join(cacheDir, p.cacheKey(url)+".json")

	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil
	}

	// Check if cache is still valid
	if time.Since(info.ModTime()) > maxAge {
		return nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var feed models.ExternalFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil
	}

	return &feed
}

// saveToCache saves a feed to cache.
// Cache writes are best-effort; errors are silently ignored.
func (p *BlogrollPlugin) saveToCache(feed *models.ExternalFeed, cacheDir string) {
	cacheFile := filepath.Join(cacheDir, p.cacheKey(feed.FeedURL)+".json")

	data, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return
	}

	//nolint:errcheck // Cache writes are best-effort
	os.WriteFile(cacheFile, data, 0o600)
}

// cacheKey generates a cache key from a URL.
func (p *BlogrollPlugin) cacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:8])
}

// groupByCategory groups feeds by their category.
func (p *BlogrollPlugin) groupByCategory(feeds []*models.ExternalFeed) []*models.BlogrollCategory {
	categoryMap := make(map[string]*models.BlogrollCategory)

	for _, feed := range feeds {
		category := feed.Category
		if category == "" {
			category = categoryUncategorized
		}

		cat, ok := categoryMap[category]
		if !ok {
			cat = &models.BlogrollCategory{
				Name: category,
				Slug: blogrollSlugify(category),
			}
			categoryMap[category] = cat
		}
		cat.Feeds = append(cat.Feeds, feed)
	}

	// Convert to slice and sort
	categories := make([]*models.BlogrollCategory, 0, len(categoryMap))
	for _, cat := range categoryMap {
		categories = append(categories, cat)
	}

	sort.Slice(categories, func(i, j int) bool {
		// Put "Uncategorized" last
		if categories[i].Name == categoryUncategorized {
			return false
		}
		if categories[j].Name == categoryUncategorized {
			return true
		}
		return categories[i].Name < categories[j].Name
	})

	return categories
}

// writeBlogrollPage generates the /blogroll page.
func (p *BlogrollPlugin) writeBlogrollPage(m *lifecycle.Manager, outputDir string, feeds []*models.ExternalFeed, config models.BlogrollConfig) error {
	// Create output directory
	blogrollDir := filepath.Join(outputDir, "blogroll")
	if err := os.MkdirAll(blogrollDir, 0o755); err != nil {
		return err
	}

	// Group feeds by category
	categories := p.groupByCategory(feeds)

	// Build template context with config for theme inheritance
	ctx := map[string]interface{}{
		"title":       "Blogroll",
		"description": "Blogs and feeds I follow",
		"feeds":       p.feedsToMaps(feeds),
		"categories":  p.categoriesToMaps(categories),
		"feed_count":  len(feeds),
		"config":      p.configToMap(m.Config()),
	}

	// Try to render with template engine
	content, err := p.renderTemplate(m, config.Templates.Blogroll, ctx)
	if err != nil {
		// Fall back to built-in template
		content = p.renderBlogrollFallback(feeds, categories)
	}

	// Write the file
	outputFile := filepath.Join(blogrollDir, "index.html")
	return os.WriteFile(outputFile, []byte(content), 0o600)
}

// writeReaderPage generates the /reader page.
func (p *BlogrollPlugin) writeReaderPage(m *lifecycle.Manager, outputDir string, entries []*models.ExternalEntry, config models.BlogrollConfig) error {
	// Create output directory
	readerDir := filepath.Join(outputDir, "reader")
	if err := os.MkdirAll(readerDir, 0o755); err != nil {
		return err
	}

	// Limit entries to 50 for the page
	displayEntries := entries
	if len(displayEntries) > 50 {
		displayEntries = displayEntries[:50]
	}

	// Build template context with config for theme inheritance
	ctx := map[string]interface{}{
		"title":       "Reader",
		"description": "Latest posts from blogs I follow",
		"entries":     p.entriesToMaps(displayEntries),
		"entry_count": len(entries),
		"config":      p.configToMap(m.Config()),
	}

	// Try to render with template engine
	content, err := p.renderTemplate(m, config.Templates.Reader, ctx)
	if err != nil {
		// Fall back to built-in template
		content = p.renderReaderFallback(entries)
	}

	// Write the file
	outputFile := filepath.Join(readerDir, "index.html")
	return os.WriteFile(outputFile, []byte(content), 0o600)
}

// renderTemplate attempts to render using the template engine.
func (p *BlogrollPlugin) renderTemplate(m *lifecycle.Manager, templateName string, ctx map[string]interface{}) (string, error) {
	// Check if template engine is available
	engine, ok := m.Cache().Get("template_engine")
	if !ok {
		return "", fmt.Errorf("template engine not available")
	}

	// Try to use the engine
	if eng, ok := engine.(interface {
		RenderToString(string, map[string]interface{}) (string, error)
	}); ok {
		return eng.RenderToString(templateName, ctx)
	}

	return "", fmt.Errorf("template engine does not support RenderToString")
}

// feedsToMaps converts feeds to template-friendly maps.
func (p *BlogrollPlugin) feedsToMaps(feeds []*models.ExternalFeed) []map[string]interface{} {
	result := make([]map[string]interface{}, len(feeds))
	for i, feed := range feeds {
		result[i] = map[string]interface{}{
			"title":        feed.Title,
			"description":  feed.Description,
			"site_url":     feed.SiteURL,
			"feed_url":     feed.FeedURL,
			"image_url":    feed.ImageURL,
			"category":     feed.Category,
			"tags":         feed.Tags,
			"entry_count":  feed.EntryCount,
			"last_fetched": feed.LastFetched,
			"last_updated": feed.LastUpdated,
			"error":        feed.Error,
		}
	}
	return result
}

// entriesToMaps converts entries to template-friendly maps.
func (p *BlogrollPlugin) entriesToMaps(entries []*models.ExternalEntry) []map[string]interface{} {
	result := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = map[string]interface{}{
			"feed_url":     entry.FeedURL,
			"feed_title":   entry.FeedTitle,
			"id":           entry.ID,
			"url":          entry.URL,
			"title":        entry.Title,
			"description":  entry.Description,
			"content":      entry.Content,
			"author":       entry.Author,
			"published":    entry.Published,
			"updated":      entry.Updated,
			"categories":   entry.Categories,
			"image_url":    entry.ImageURL,
			"reading_time": entry.ReadingTime,
		}
	}
	return result
}

// categoriesToMaps converts categories to template-friendly maps.
func (p *BlogrollPlugin) categoriesToMaps(categories []*models.BlogrollCategory) []map[string]interface{} {
	result := make([]map[string]interface{}, len(categories))
	for i, cat := range categories {
		result[i] = map[string]interface{}{
			"name":  cat.Name,
			"slug":  cat.Slug,
			"feeds": p.feedsToMaps(cat.Feeds),
		}
	}
	return result
}

// configToMap converts config to a template-friendly map with essential fields.
func (p *BlogrollPlugin) configToMap(config *lifecycle.Config) map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"output_dir": config.OutputDir,
		"lang":       "en", // Default language
	}

	// Extract common fields from Extra
	if config.Extra != nil {
		if title, ok := config.Extra["title"].(string); ok {
			result["title"] = title
		}
		if description, ok := config.Extra["description"].(string); ok {
			result["description"] = description
		}
		if url, ok := config.Extra["url"].(string); ok {
			result["url"] = url
		}
		if author, ok := config.Extra["author"].(string); ok {
			result["author"] = author
		}

		// Convert nav items if available
		if navItems, ok := config.Extra["nav"].([]models.NavItem); ok {
			navMaps := make([]map[string]interface{}, len(navItems))
			for i, nav := range navItems {
				navMaps[i] = map[string]interface{}{
					"label":    nav.Label,
					"url":      nav.URL,
					"external": nav.External,
				}
			}
			result["nav"] = navMaps
		}

		// Convert search config if available
		if search, ok := config.Extra["search"].(models.SearchConfig); ok {
			searchEnabled := true
			if search.Enabled != nil {
				searchEnabled = *search.Enabled
			}
			result["search"] = map[string]interface{}{
				"enabled":     searchEnabled,
				"position":    search.Position,
				"placeholder": search.Placeholder,
			}
		}
	}

	return result
}

// renderBlogrollFallback generates a basic blogroll page that uses theme CSS if available.
func (p *BlogrollPlugin) renderBlogrollFallback(feeds []*models.ExternalFeed, categories []*models.BlogrollCategory) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Blogroll</title>
  <!-- Theme CSS - uses site's configured palette if available -->
  <link rel="stylesheet" href="/css/variables.css">
  <link rel="stylesheet" href="/css/main.css">
  <link rel="stylesheet" href="/css/components.css">
  <style>
    /* Fallback styles if theme CSS is not available */
    :root {
      --color-background: #ffffff;
      --color-text: #1a1a1a;
      --color-text-muted: #666666;
      --color-border: #e0e0e0;
      --color-surface: #f8f9fa;
      --color-primary: #3b82f6;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --color-background: #1a1a1a;
        --color-text: #f0f0f0;
        --color-text-muted: #999999;
        --color-border: #333333;
        --color-surface: #2a2a2a;
        --color-primary: #60a5fa;
      }
    }
  </style>
</head>
<body>
  <nav class="blogroll-nav" style="justify-content: flex-start; padding: 1rem 0;">
    <a href="/">Home</a>
    <a href="/blogroll/">Blogroll</a>
    <a href="/reader/">Reader</a>
  </nav>
  <div class="blogroll-page">
    <header class="blogroll-header" style="text-align: left;">
      <h1>Blogroll</h1>
      <p class="blogroll-subtitle">`)
	sb.WriteString(fmt.Sprintf("%d blogs and feeds I follow", len(feeds)))
	sb.WriteString(`</p>
    </header>
`)

	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf(`    <section class="blogroll-category">
      <h2>%s</h2>
      <div class="blogroll-grid">
`, html.EscapeString(cat.Name)))

		for _, feed := range cat.Feeds {
			sb.WriteString(`        <article class="blogroll-card">
          <h3 class="blogroll-card-title">`)
			if feed.SiteURL != "" {
				sb.WriteString(fmt.Sprintf(`<a href=%q target="_blank" rel="noopener">%s</a>`,
					feed.SiteURL,
					html.EscapeString(feed.Title)))
			} else {
				sb.WriteString(html.EscapeString(feed.Title))
			}
			sb.WriteString(`</h3>
`)
			if feed.Description != "" {
				sb.WriteString(fmt.Sprintf(`          <p class="blogroll-card-description">%s</p>
`, html.EscapeString(blogrollTruncateString(feed.Description, 150))))
			}
			sb.WriteString(fmt.Sprintf(`          <footer class="blogroll-card-meta">
            <span class="blogroll-card-count">%d posts</span>
          </footer>
        </article>
`, feed.EntryCount))
		}

		sb.WriteString(`      </div>
    </section>
`)
	}

	sb.WriteString(`  </div>
</body>
</html>`)

	return sb.String()
}

// renderReaderFallback generates a basic reader page that uses theme CSS if available.
func (p *BlogrollPlugin) renderReaderFallback(entries []*models.ExternalEntry) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reader</title>
  <!-- Theme CSS - uses site's configured palette if available -->
  <link rel="stylesheet" href="/css/variables.css">
  <link rel="stylesheet" href="/css/main.css">
  <link rel="stylesheet" href="/css/components.css">
  <style>
    /* Fallback styles if theme CSS is not available */
    :root {
      --color-background: #ffffff;
      --color-text: #1a1a1a;
      --color-text-muted: #666666;
      --color-border: #e0e0e0;
      --color-surface: #f8f9fa;
      --color-primary: #3b82f6;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --color-background: #1a1a1a;
        --color-text: #f0f0f0;
        --color-text-muted: #999999;
        --color-border: #333333;
        --color-surface: #2a2a2a;
        --color-primary: #60a5fa;
      }
    }
  </style>
</head>
<body>
  <nav class="reader-nav" style="justify-content: flex-start; padding: 1rem 0;">
    <a href="/">Home</a>
    <a href="/blogroll/">Blogroll</a>
    <a href="/reader/">Reader</a>
  </nav>
  <div class="reader-page">
    <header class="reader-header" style="text-align: left;">
      <h1>Reader</h1>
      <p class="reader-subtitle">Latest posts from blogs I follow</p>
    </header>
    <ul class="reader-entries">
`)

	// Limit to 50 entries for the page
	displayEntries := entries
	if len(displayEntries) > 50 {
		displayEntries = displayEntries[:50]
	}

	for _, entry := range displayEntries {
		sb.WriteString(`      <li class="reader-entry">
        <article>
          <h2 class="reader-entry-title"><a href="`)
		sb.WriteString(html.EscapeString(entry.URL))
		sb.WriteString(`" target="_blank" rel="noopener">`)
		sb.WriteString(html.EscapeString(entry.Title))
		sb.WriteString(`</a></h2>
          <div class="reader-entry-meta">
            <span class="reader-entry-source">`)
		sb.WriteString(html.EscapeString(entry.FeedTitle))
		sb.WriteString(`</span>`)

		if entry.Published != nil {
			sb.WriteString(`<time>`)
			sb.WriteString(entry.Published.Format("Jan 2, 2006"))
			sb.WriteString(`</time>`)
		}

		sb.WriteString(`
          </div>
`)
		if entry.Description != "" {
			sb.WriteString(`          <p class="reader-entry-description">`)
			sb.WriteString(html.EscapeString(blogrollTruncateString(blogrollStripHTML(entry.Description), 200)))
			sb.WriteString(`</p>
`)
		}
		sb.WriteString(`        </article>
      </li>
`)
	}

	sb.WriteString(`    </ul>
  </div>
</body>
</html>`)

	return sb.String()
}

// getBlogrollConfig retrieves blogroll configuration from the manager config.
func getBlogrollConfig(config *lifecycle.Config) models.BlogrollConfig {
	if config.Extra == nil {
		return models.NewBlogrollConfig()
	}

	if blogroll, ok := config.Extra["blogroll"]; ok {
		if bc, ok := blogroll.(models.BlogrollConfig); ok {
			return bc
		}
	}

	return models.NewBlogrollConfig()
}

// blogrollSlugify converts a string to a URL-safe slug.
func blogrollSlugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove multiple consecutive hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim leading/trailing hyphens
	return strings.Trim(slug, "-")
}

// blogrollTruncateString truncates a string to the specified length.
func blogrollTruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// blogrollStripHTML removes HTML tags from a string.
func blogrollStripHTML(s string) string {
	// Simple regex-based HTML stripping
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = html.UnescapeString(s)

	// Normalize whitespace
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// Ensure BlogrollPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*BlogrollPlugin)(nil)
	_ lifecycle.CollectPlugin  = (*BlogrollPlugin)(nil)
	_ lifecycle.WritePlugin    = (*BlogrollPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*BlogrollPlugin)(nil)
)
