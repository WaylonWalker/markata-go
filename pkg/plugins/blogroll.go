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
	neturl "net/url"
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

// Default slug constants for blogroll pages.
const (
	defaultBlogrollSlug = "blogroll"
	defaultReaderSlug   = "reader"
)

// blogrollHTMLTagRegex matches HTML tags for stripping.
var blogrollHTMLTagRegex = regexp.MustCompile(`<[^>]*>`)

// BlogrollPlugin fetches and processes external RSS/Atom feeds.
// It runs in the Configure stage to register synthetic posts for wikilink resolution,
// in the Collect stage to gather external feed entries,
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
	case lifecycle.StageConfigure:
		// Run early to register synthetic posts before wikilinks are processed
		return lifecycle.PriorityDefault
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

// Configure registers synthetic posts for blogroll and reader pages
// so they can be resolved by wikilinks during the Transform stage.
func (p *BlogrollPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	blogrollConfig := getBlogrollConfig(config)

	if !blogrollConfig.Enabled {
		return nil
	}

	// Register synthetic posts for wikilink resolution
	p.registerSyntheticPosts(m, blogrollConfig)

	return nil
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

// registerSyntheticPosts creates synthetic Post objects for blogroll and reader pages
// so they can be resolved by wikilinks. These posts are marked with Skip: true
// so they don't interfere with normal rendering.
func (p *BlogrollPlugin) registerSyntheticPosts(m *lifecycle.Manager, config models.BlogrollConfig) {
	// Get configured slugs with defaults
	blogrollSlug := config.BlogrollSlug
	if blogrollSlug == "" {
		blogrollSlug = defaultBlogrollSlug
	}
	readerSlug := config.ReaderSlug
	if readerSlug == "" {
		readerSlug = defaultReaderSlug
	}

	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	// Register blogroll page
	blogrollPost := &models.Post{
		Slug:        blogrollSlug,
		Title:       strPtr("Blogroll"),
		Description: strPtr("Blogs and feeds I follow"),
		Href:        "/" + blogrollSlug + "/",
		Published:   true,
		Skip:        true,
	}
	m.AddPost(blogrollPost)

	// Register reader page
	readerPost := &models.Post{
		Slug:        readerSlug,
		Title:       strPtr("Reader"),
		Description: strPtr("Latest posts from blogs I follow"),
		Href:        "/" + readerSlug + "/",
		Published:   true,
		Skip:        true,
	}
	m.AddPost(readerPost)
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

	globalMaxEntries := config.MaxEntriesPerFeed
	if globalMaxEntries <= 0 {
		globalMaxEntries = 50
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

			// Use per-feed max_entries if set, otherwise global default
			maxEntries := feedConfig.GetMaxEntries(globalMaxEntries)
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

	// Apply fallback image service for entries without images
	if config.FallbackImageService != "" {
		for _, entry := range allEntries {
			if entry.ImageURL == "" && entry.URL != "" {
				entry.ImageURL = generateFallbackImageURL(config.FallbackImageService, entry.URL)
			}
		}
	}

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

// writeBlogrollPage generates the blogroll page at the configured slug path.
func (p *BlogrollPlugin) writeBlogrollPage(m *lifecycle.Manager, outputDir string, feeds []*models.ExternalFeed, config models.BlogrollConfig) error {
	// Use configured slug or default to "blogroll"
	slug := config.BlogrollSlug
	if slug == "" {
		slug = defaultBlogrollSlug
	}

	// Create output directory
	blogrollDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(blogrollDir, 0o755); err != nil {
		return err
	}

	// Group feeds by category
	categories := p.groupByCategory(feeds)

	// Get reader slug for cross-linking
	readerSlug := config.ReaderSlug
	if readerSlug == "" {
		readerSlug = defaultReaderSlug
	}

	// Build template context with config for theme inheritance
	ctx := map[string]interface{}{
		"title":        "Blogroll",
		"description":  "Blogs and feeds I follow",
		"feeds":        p.feedsToMaps(feeds),
		"categories":   p.categoriesToMaps(categories),
		"feed_count":   len(feeds),
		"config":       p.configToMap(m.Config()),
		"blogroll_url": "/" + slug + "/",
		"reader_url":   "/" + readerSlug + "/",
	}

	// Try to render with template engine
	content, err := p.renderTemplate(m, config.Templates.Blogroll, ctx)
	if err != nil {
		// Fall back to built-in template
		content = p.renderBlogrollFallback(feeds, categories, config)
	}

	// Write the file
	outputFile := filepath.Join(blogrollDir, "index.html")
	return os.WriteFile(outputFile, []byte(content), 0o600)
}

// writeReaderPage generates the paginated /reader pages.
func (p *BlogrollPlugin) writeReaderPage(m *lifecycle.Manager, outputDir string, entries []*models.ExternalEntry, config models.BlogrollConfig) error {
	// Use configured slug or default to "reader"
	slug := config.ReaderSlug
	if slug == "" {
		slug = defaultReaderSlug
	}

	// Create output directory
	readerDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(readerDir, 0o755); err != nil {
		return err
	}

	// Paginate entries
	pages := p.paginateEntries(entries, config, "/reader")

	// If no pages (no entries), generate empty page
	if len(pages) == 0 {
		return p.writeReaderPageFile(m, readerDir, config, models.ReaderPage{
			Number:         1,
			Entries:        []*models.ExternalEntry{},
			TotalPages:     1,
			TotalItems:     0,
			ItemsPerPage:   config.ItemsPerPage,
			PaginationType: config.PaginationType,
			PageURLs:       []string{"/reader/"},
		}, true)
	}

	// Generate each page
	for i := range pages {
		isFirstPage := i == 0
		if err := p.writeReaderPageFile(m, readerDir, config, pages[i], isFirstPage); err != nil {
			return err
		}
	}

	return nil
}

// writeReaderPageFile writes a single reader page (full and partial for HTMX).
func (p *BlogrollPlugin) writeReaderPageFile(m *lifecycle.Manager, readerDir string, config models.BlogrollConfig, page models.ReaderPage, isFirstPage bool) error {
	// Get slugs for cross-linking
	blogrollSlug := config.BlogrollSlug
	if blogrollSlug == "" {
		blogrollSlug = defaultBlogrollSlug
	}
	readerSlug := config.ReaderSlug
	if readerSlug == "" {
		readerSlug = defaultReaderSlug
	}

	// Build template context with config for theme inheritance
	ctx := map[string]interface{}{
		"title":           "Reader",
		"description":     "Latest posts from blogs I follow",
		"entries":         p.entriesToMaps(page.Entries),
		"entry_count":     page.TotalItems,
		"config":          p.configToMap(m.Config()),
		"page":            p.readerPageToMap(page),
		"pagination_type": string(page.PaginationType),
		"blogroll_url":    "/" + blogrollSlug + "/",
		"reader_url":      "/" + readerSlug + "/",
	}

	// Determine output path
	var outputFile string
	var partialDir string
	if isFirstPage {
		outputFile = filepath.Join(readerDir, "index.html")
		partialDir = filepath.Join(readerDir, "partial")
	} else {
		pageDir := filepath.Join(readerDir, "page", fmt.Sprintf("%d", page.Number))
		if err := os.MkdirAll(pageDir, 0o755); err != nil {
			return err
		}
		outputFile = filepath.Join(pageDir, "index.html")
		partialDir = filepath.Join(pageDir, "partial")
	}

	// Try to render with template engine
	content, err := p.renderTemplate(m, config.Templates.Reader, ctx)
	if err != nil {
		// Fall back to built-in template
		content = p.renderReaderFallback(page.Entries, page, config)
	}

	// Write the full page
	if err := os.WriteFile(outputFile, []byte(content), 0o600); err != nil {
		return err
	}

	// For HTMX pagination, also generate partial pages
	if page.PaginationType == models.PaginationHTMX {
		if err := os.MkdirAll(partialDir, 0o755); err != nil {
			return err
		}

		partialContent := p.renderReaderPartial(page.Entries, page)
		partialFile := filepath.Join(partialDir, "index.html")
		if err := os.WriteFile(partialFile, []byte(partialContent), 0o600); err != nil {
			return err
		}
	}

	return nil
}

// paginateEntries divides entries into pages based on ItemsPerPage and OrphanThreshold.
func (p *BlogrollPlugin) paginateEntries(entries []*models.ExternalEntry, config models.BlogrollConfig, baseURL string) []models.ReaderPage {
	if len(entries) == 0 {
		return []models.ReaderPage{}
	}

	itemsPerPage := config.ItemsPerPage
	if itemsPerPage <= 0 {
		itemsPerPage = 50
	}

	orphanThreshold := config.OrphanThreshold
	if orphanThreshold <= 0 {
		orphanThreshold = 3
	}

	paginationType := config.PaginationType
	if paginationType == "" {
		paginationType = models.PaginationManual
	}

	totalEntries := len(entries)
	var pages []models.ReaderPage

	for i := 0; i < totalEntries; i += itemsPerPage {
		end := i + itemsPerPage
		if end > totalEntries {
			end = totalEntries
		}

		// Check orphan threshold: if remaining items are below threshold,
		// add them to the current page instead of creating a new page
		remaining := totalEntries - end
		if remaining > 0 && remaining < orphanThreshold {
			end = totalEntries
		}

		pageNum := len(pages) + 1
		page := models.ReaderPage{
			Number:  pageNum,
			Entries: entries[i:end],
			HasPrev: pageNum > 1,
		}

		pages = append(pages, page)

		if end >= totalEntries {
			break
		}
	}

	totalPages := len(pages)

	// Generate page URLs for numbered navigation
	pageURLs := make([]string, totalPages)
	for i := 0; i < totalPages; i++ {
		if i == 0 {
			pageURLs[i] = baseURL + "/"
		} else {
			pageURLs[i] = baseURL + "/page/" + fmt.Sprintf("%d", i+1) + "/"
		}
	}

	// Set HasNext, URLs, and metadata for each page
	for i := range pages {
		pages[i].HasNext = i < totalPages-1
		pages[i].TotalPages = totalPages
		pages[i].TotalItems = totalEntries
		pages[i].ItemsPerPage = itemsPerPage
		pages[i].PageURLs = pageURLs
		pages[i].PaginationType = paginationType

		if pages[i].HasPrev {
			if i == 1 {
				pages[i].PrevURL = baseURL + "/"
			} else {
				pages[i].PrevURL = baseURL + "/page/" + fmt.Sprintf("%d", i) + "/"
			}
		}

		if pages[i].HasNext {
			pages[i].NextURL = baseURL + "/page/" + fmt.Sprintf("%d", i+2) + "/"
		}
	}

	return pages
}

// readerPageToMap converts a ReaderPage to a template-friendly map.
func (p *BlogrollPlugin) readerPageToMap(page models.ReaderPage) map[string]interface{} {
	return map[string]interface{}{
		"number":          page.Number,
		"has_prev":        page.HasPrev,
		"has_next":        page.HasNext,
		"prev_url":        page.PrevURL,
		"next_url":        page.NextURL,
		"total_pages":     page.TotalPages,
		"total_items":     page.TotalItems,
		"items_per_page":  page.ItemsPerPage,
		"page_urls":       page.PageURLs,
		"pagination_type": string(page.PaginationType),
	}
}

// renderTemplate attempts to render using the template engine.
func (p *BlogrollPlugin) renderTemplate(m *lifecycle.Manager, templateName string, ctx map[string]interface{}) (string, error) {
	// Check if template engine is available
	// The templates plugin stores it as "templates.engine"
	engine, ok := m.Cache().Get("templates.engine")
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
// This mirrors the logic from templates.go:toModelsConfig() to ensure all
// template-accessible config fields are available in blogroll/reader pages.
func (p *BlogrollPlugin) configToMap(config *lifecycle.Config) map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"output_dir": config.OutputDir,
		"lang":       "en", // Default language
	}

	if config.Extra != nil {
		p.extractStringFields(config.Extra, result)
		p.extractNavItems(config.Extra, result)
		p.extractSearchConfig(config.Extra, result)
		p.extractStructConfigs(config.Extra, result)
		p.extractLayoutConfig(config.Extra, result)
		p.extractSEOConfig(config.Extra, result)
		p.extractMiscConfigs(config.Extra, result)
	}

	return result
}

// extractStringFields extracts common string fields from config.Extra.
func (p *BlogrollPlugin) extractStringFields(extra, result map[string]interface{}) {
	stringFields := []string{"title", "description", "url", "author", "templates_dir"}
	for _, field := range stringFields {
		if value, ok := extra[field].(string); ok {
			result[field] = value
		}
	}
}

// extractNavItems converts nav items from config.Extra.
func (p *BlogrollPlugin) extractNavItems(extra, result map[string]interface{}) {
	navItems, ok := extra["nav"].([]models.NavItem)
	if !ok {
		return
	}
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

// extractSearchConfig converts search config from config.Extra.
func (p *BlogrollPlugin) extractSearchConfig(extra, result map[string]interface{}) {
	search, ok := extra["search"].(models.SearchConfig)
	if !ok {
		return
	}
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

// extractStructConfigs extracts simple struct configs from config.Extra.
func (p *BlogrollPlugin) extractStructConfigs(extra, result map[string]interface{}) {
	// Convert components config if available (fixes #316)
	if components, ok := extra["components"].(models.ComponentsConfig); ok {
		result["components"] = components
	}
	if footer, ok := extra["footer"].(models.FooterConfig); ok {
		result["footer"] = footer
	}
	if sidebar, ok := extra["sidebar"].(models.SidebarConfig); ok {
		result["sidebar"] = sidebar
	}
	if toc, ok := extra["toc"].(models.TocConfig); ok {
		result["toc"] = toc
	}
	if header, ok := extra["header"].(models.HeaderLayoutConfig); ok {
		result["header"] = header
	}
	if postFormats, ok := extra["post_formats"].(models.PostFormatsConfig); ok {
		result["post_formats"] = postFormats
	}
	if head, ok := extra["head"].(models.HeadConfig); ok {
		result["head"] = head
	}
}

// extractLayoutConfig converts layout config from config.Extra, handling both pointer and value types.
func (p *BlogrollPlugin) extractLayoutConfig(extra, result map[string]interface{}) {
	switch layoutVal := extra["layout"].(type) {
	case *models.LayoutConfig:
		result["layout"] = *layoutVal
	case models.LayoutConfig:
		result["layout"] = layoutVal
	}
}

// extractSEOConfig converts SEO config from config.Extra, handling both struct and map types.
func (p *BlogrollPlugin) extractSEOConfig(extra, result map[string]interface{}) {
	switch seoVal := extra["seo"].(type) {
	case models.SEOConfig:
		result["seo"] = seoVal
	case map[string]interface{}:
		result["seo"] = models.SEOConfig{
			TwitterHandle: p.getStringFromMap(seoVal, "twitter_handle"),
			DefaultImage:  p.getStringFromMap(seoVal, "default_image"),
			LogoURL:       p.getStringFromMap(seoVal, "logo_url"),
		}
	}
}

// extractMiscConfigs extracts miscellaneous configs that don't need type conversion.
func (p *BlogrollPlugin) extractMiscConfigs(extra, result map[string]interface{}) {
	if webmention, ok := extra["webmention"]; ok {
		result["webmention"] = webmention
	}
	if feeds, ok := extra["feeds"]; ok {
		result["feeds"] = feeds
	}
	// Extract theme config for background-css partial and other theme features
	if theme, ok := extra["theme"].(models.ThemeConfig); ok {
		result["theme"] = p.themeToMap(theme)
	}
}

// themeToMap converts ThemeConfig to a template-friendly map.
func (p *BlogrollPlugin) themeToMap(theme models.ThemeConfig) map[string]interface{} {
	result := map[string]interface{}{
		"name":          theme.Name,
		"palette":       theme.Palette,
		"palette_light": theme.PaletteLight,
		"palette_dark":  theme.PaletteDark,
		"custom_css":    theme.CustomCSS,
	}

	// Convert background config
	bg := theme.Background
	bgEnabled := false
	if bg.Enabled != nil {
		bgEnabled = *bg.Enabled
	}

	// Convert backgrounds array to template-friendly format
	backgrounds := make([]map[string]interface{}, len(bg.Backgrounds))
	for i, bgElem := range bg.Backgrounds {
		backgrounds[i] = map[string]interface{}{
			"html":    bgElem.HTML,
			"z_index": bgElem.ZIndex,
		}
	}

	result["background"] = map[string]interface{}{
		"enabled":        bgEnabled,
		"backgrounds":    backgrounds,
		"scripts":        bg.Scripts,
		"css":            bg.CSS,
		"article_bg":     bg.ArticleBg,
		"article_blur":   bg.ArticleBlur,
		"article_shadow": bg.ArticleShadow,
		"article_border": bg.ArticleBorder,
		"article_radius": bg.ArticleRadius,
	}

	// Convert font config
	result["font"] = map[string]interface{}{
		"family":         theme.Font.Family,
		"heading_family": theme.Font.HeadingFamily,
		"code_family":    theme.Font.CodeFamily,
		"size":           theme.Font.Size,
		"line_height":    theme.Font.LineHeight,
	}

	return result
}

// getStringFromMap safely gets a string value from a map.
func (p *BlogrollPlugin) getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// renderBlogrollFallback generates a basic blogroll page that uses theme CSS if available.
func (p *BlogrollPlugin) renderBlogrollFallback(feeds []*models.ExternalFeed, categories []*models.BlogrollCategory, config models.BlogrollConfig) string {
	var sb strings.Builder

	// Get configured slugs with defaults
	blogrollSlug := config.BlogrollSlug
	if blogrollSlug == "" {
		blogrollSlug = defaultBlogrollSlug
	}
	readerSlug := config.ReaderSlug
	if readerSlug == "" {
		readerSlug = defaultReaderSlug
	}

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
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      max-width: 900px;
      margin: 0 auto;
      padding: 2rem;
      background: var(--color-background);
      color: var(--color-text);
      line-height: 1.6;
    }
    h1 { margin-bottom: 0.5rem; }
    .subtitle { color: var(--color-text-muted); margin-bottom: 2rem; }
    .category { margin-bottom: 2rem; }
    .category h2 { border-bottom: 1px solid var(--color-border); padding-bottom: 0.5rem; }
    .feed-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 1rem;
    }
    .feed-card {
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 8px;
      padding: 1rem;
      display: flex;
      gap: 0.75rem;
    }
    .feed-avatar {
      width: 48px;
      height: 48px;
      border-radius: 6px;
      object-fit: cover;
      flex-shrink: 0;
      background: var(--color-border);
    }
    .feed-avatar-placeholder {
      width: 48px;
      height: 48px;
      border-radius: 6px;
      flex-shrink: 0;
      background: var(--color-border);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 1.25rem;
      color: var(--color-text-muted);
    }
    .feed-content {
      flex: 1;
      min-width: 0;
    }
    .feed-card h3 {
      margin: 0 0 0.5rem 0;
      font-size: 1rem;
    }
    .feed-card h3 a {
      color: var(--color-text);
      text-decoration: none;
    }
    .feed-card h3 a:hover { text-decoration: underline; }
    .feed-card p {
      margin: 0;
      font-size: 0.875rem;
      color: var(--color-text-muted);
    }
    .feed-meta {
      margin-top: 0.5rem;
      font-size: 0.75rem;
      color: var(--color-text-muted);
    }
    .nav-links {
      margin-bottom: 1rem;
    }
    .nav-links a {
      color: var(--color-text);
      margin-right: 1rem;
    }
  </style>
</head>
<body>
  <nav class="blogroll-nav" style="justify-content: flex-start; padding: 1rem 0;">
    <a href="/">Home</a>
    <a href="/` + blogrollSlug + `/">Blogroll</a>
    <a href="/` + readerSlug + `/">Reader</a>
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
			sb.WriteString(`        <article class="blogroll-card feed-card">
`)
			// Feed avatar
			if feed.ImageURL != "" {
				sb.WriteString(fmt.Sprintf(`          <img src=%q alt="" class="feed-avatar" loading="lazy">
`, feed.ImageURL))
			} else {
				// Placeholder with first letter
				initial := "?"
				if feed.Title != "" {
					initial = strings.ToUpper(string([]rune(feed.Title)[0:1]))
				}
				sb.WriteString(fmt.Sprintf(`          <div class="feed-avatar-placeholder">%s</div>
`, initial))
			}
			sb.WriteString(`          <div class="feed-content">
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
				sb.WriteString(fmt.Sprintf(`            <p class="blogroll-card-description">%s</p>
`, html.EscapeString(blogrollTruncateString(feed.Description, 150))))
			}
			sb.WriteString(fmt.Sprintf(`            <footer class="blogroll-card-meta feed-meta">
              <span class="blogroll-card-count">%d posts</span>
            </footer>
          </div>
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
func (p *BlogrollPlugin) renderReaderFallback(entries []*models.ExternalEntry, page models.ReaderPage, config models.BlogrollConfig) string {
	var sb strings.Builder

	// Get configured slugs with defaults
	blogrollSlug := config.BlogrollSlug
	if blogrollSlug == "" {
		blogrollSlug = defaultBlogrollSlug
	}
	readerSlug := config.ReaderSlug
	if readerSlug == "" {
		readerSlug = defaultReaderSlug
	}

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
    .pagination {
      display: flex;
      justify-content: center;
      gap: 1rem;
      margin: 2rem 0;
      padding: 1rem 0;
    }
    .pagination a, .pagination span {
      padding: 0.5rem 1rem;
      border: 1px solid var(--color-border);
      border-radius: 4px;
      text-decoration: none;
      color: var(--color-text);
    }
    .pagination a:hover {
      background: var(--color-surface);
    }
    .pagination .current {
      background: var(--color-primary);
      color: white;
      border-color: var(--color-primary);
    }
    .pagination .disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      max-width: 900px;
      margin: 0 auto;
      padding: 2rem;
      background: var(--color-background);
      color: var(--color-text);
      line-height: 1.6;
    }
    h1 { margin-bottom: 0.5rem; }
    .subtitle { color: var(--color-text-muted); margin-bottom: 2rem; }
    .entry-list { list-style: none; padding: 0; margin: 0; }
    .entry {
      border-bottom: 1px solid var(--color-border);
      padding: 1.5rem 0;
      display: flex;
      gap: 1rem;
    }
    .entry:last-child { border-bottom: none; }
    .entry-image {
      width: 120px;
      height: 80px;
      border-radius: 6px;
      object-fit: cover;
      flex-shrink: 0;
      background: var(--color-border);
    }
    .entry-content {
      flex: 1;
      min-width: 0;
    }
    .entry h2 {
      margin: 0 0 0.5rem 0;
      font-size: 1.25rem;
    }
    .entry h2 a {
      color: var(--color-text);
      text-decoration: none;
    }
    .entry h2 a:hover { text-decoration: underline; }
    .entry-meta {
      font-size: 0.875rem;
      color: var(--color-text-muted);
      margin-bottom: 0.5rem;
    }
    .entry-meta a { color: var(--color-primary); }
    .entry-description {
      color: var(--color-text);
      font-size: 0.9375rem;
    }
    .nav-links {
      margin-bottom: 1rem;
    }
    .nav-links a {
      color: var(--color-text);
      margin-right: 1rem;
    }
    @media (max-width: 600px) {
      .entry {
        flex-direction: column;
      }
      .entry-image {
        width: 100%;
        height: 160px;
      }
    }
  </style>
`)
	// Add HTMX if using that pagination type
	if page.PaginationType == models.PaginationHTMX {
		sb.WriteString(`  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
`)
	}
	sb.WriteString(`</head>
<body>
  <nav class="reader-nav" style="justify-content: flex-start; padding: 1rem 0;">
    <a href="/">Home</a>
    <a href="/` + blogrollSlug + `/">Blogroll</a>
    <a href="/` + readerSlug + `/">Reader</a>
  </nav>
  <div class="reader-page">
    <header class="reader-header" style="text-align: left;">
      <h1>Reader</h1>
      <p class="reader-subtitle">Latest posts from blogs I follow</p>
    </header>
`)

	// Add content container for HTMX
	if page.PaginationType == models.PaginationHTMX {
		sb.WriteString(`    <div id="reader-content">
`)
	}

	sb.WriteString(`    <ul class="reader-entries">
`)

	for _, entry := range entries {
		sb.WriteString(`      <li class="reader-entry entry">
`)
		// Entry image
		if entry.ImageURL != "" {
			sb.WriteString(fmt.Sprintf(`        <img src=%q alt="" class="entry-image" loading="lazy">
`, html.EscapeString(entry.ImageURL)))
		}
		sb.WriteString(`        <div class="entry-content">
          <h2 class="reader-entry-title"><a href="`)
		sb.WriteString(html.EscapeString(entry.URL))
		sb.WriteString(`" target="_blank" rel="noopener">`)
		sb.WriteString(html.EscapeString(entry.Title))
		sb.WriteString(`</a></h2>
          <div class="reader-entry-meta entry-meta">
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
			sb.WriteString(`          <p class="reader-entry-description entry-description">`)
			sb.WriteString(html.EscapeString(blogrollTruncateString(blogrollStripHTML(entry.Description), 200)))
			sb.WriteString(`</p>
`)
		}
		sb.WriteString(`        </div>
      </li>
`)
	}

	sb.WriteString(`    </ul>
`)

	// Render pagination
	sb.WriteString(p.renderPagination(page))

	if page.PaginationType == models.PaginationHTMX {
		sb.WriteString(`    </div>
`)
	}

	sb.WriteString(`  </div>
</body>
</html>`)

	return sb.String()
}

// renderReaderPartial generates just the content portion for HTMX updates.
func (p *BlogrollPlugin) renderReaderPartial(entries []*models.ExternalEntry, page models.ReaderPage) string {
	var sb strings.Builder

	sb.WriteString(`<ul class="reader-entries">
`)

	for _, entry := range entries {
		sb.WriteString(`  <li class="reader-entry">
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
			sb.WriteString(`      <p class="reader-entry-description">`)
			sb.WriteString(html.EscapeString(blogrollTruncateString(blogrollStripHTML(entry.Description), 200)))
			sb.WriteString(`</p>
`)
		}
		sb.WriteString(`    </article>
  </li>
`)
	}

	sb.WriteString(`</ul>
`)

	// Render pagination
	sb.WriteString(p.renderPagination(page))

	return sb.String()
}

// renderPagination generates pagination navigation HTML.
func (p *BlogrollPlugin) renderPagination(page models.ReaderPage) string {
	if page.TotalPages <= 1 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`    <nav class="pagination" aria-label="Page navigation">
`)

	// Previous link
	if page.HasPrev {
		if page.PaginationType == models.PaginationHTMX {
			sb.WriteString(fmt.Sprintf(`      <a href="%s" hx-get="%spartial/" hx-target="#reader-content" hx-swap="innerHTML">&laquo; Previous</a>
`, page.PrevURL, page.PrevURL))
		} else {
			sb.WriteString(fmt.Sprintf(`      <a href="%s">&laquo; Previous</a>
`, page.PrevURL))
		}
	} else {
		sb.WriteString(`      <span class="disabled">&laquo; Previous</span>
`)
	}

	// Page numbers
	for i, url := range page.PageURLs {
		pageNum := i + 1
		if pageNum == page.Number {
			sb.WriteString(fmt.Sprintf(`      <span class="current">%d</span>
`, pageNum))
		} else {
			if page.PaginationType == models.PaginationHTMX {
				sb.WriteString(fmt.Sprintf(`      <a href="%s" hx-get="%spartial/" hx-target="#reader-content" hx-swap="innerHTML">%d</a>
`, url, url, pageNum))
			} else {
				sb.WriteString(fmt.Sprintf(`      <a href="%s">%d</a>
`, url, pageNum))
			}
		}
	}

	// Next link
	if page.HasNext {
		if page.PaginationType == models.PaginationHTMX {
			sb.WriteString(fmt.Sprintf(`      <a href="%s" hx-get="%spartial/" hx-target="#reader-content" hx-swap="innerHTML">Next &raquo;</a>
`, page.NextURL, page.NextURL))
		} else {
			sb.WriteString(fmt.Sprintf(`      <a href="%s">Next &raquo;</a>
`, page.NextURL))
		}
	} else {
		sb.WriteString(`      <span class="disabled">Next &raquo;</span>
`)
	}

	sb.WriteString(`    </nav>
`)

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
	// Use pre-compiled regex for HTML stripping
	s = blogrollHTMLTagRegex.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = html.UnescapeString(s)

	// Normalize whitespace
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// generateFallbackImageURL generates a fallback image URL from a template.
// The template should contain {url} as a placeholder for the URL-encoded entry URL.
func generateFallbackImageURL(template, entryURL string) string {
	encodedURL := neturl.QueryEscape(entryURL)
	return strings.ReplaceAll(template, "{url}", encodedURL)
}

// Ensure BlogrollPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*BlogrollPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*BlogrollPlugin)(nil)
	_ lifecycle.CollectPlugin   = (*BlogrollPlugin)(nil)
	_ lifecycle.WritePlugin     = (*BlogrollPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*BlogrollPlugin)(nil)
)

// CI trigger
