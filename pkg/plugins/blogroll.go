// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
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

	"github.com/WaylonWalker/markata-go/pkg/blogroll"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// categoryUncategorized is the default category name for feeds without a category.
const categoryUncategorized = "Uncategorized"

// Default slug constants for blogroll pages.
const (
	defaultBlogrollSlug = "blogroll"
	defaultReaderSlug   = "reader"
)

// Default directory constants.
const (
	defaultOutputDir  = "output"
	blogrollBundleDir = "blogroll"
)

// extractFirstImageFromHTML extracts the first image URL from HTML content.
func extractFirstImageFromHTML(htmlContent string) string {
	// Decode HTML entities first
	decoded := html.UnescapeString(htmlContent)

	// Simple regex to find first img src attribute
	re := regexp.MustCompile(`<img[^>]+src\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(decoded)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// blogrollParsedFeed represents a parsed feed response for blogroll plugin.
type blogrollParsedFeed struct {
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Language     string     `json:"language"`
	SiteURL      string     `json:"site_url"`
	ImageURL     string     `json:"image_url"`
	AvatarURL    string     `json:"avatar_url"`
	AvatarSource string     `json:"avatar_source"`
	LastUpdated  *time.Time `json:"last_updated"`
}

// RSS 2.0 structures for feed parsing.
type rss2Feed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel rss2Channel `xml:"channel"`
}

type rss2Channel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Language    string     `xml:"language"`
	Image       rss2Image  `xml:"image"`
	Items       []rss2Item `xml:"item"`
}

type rss2Image struct {
	URL string `xml:"url"`
}

type rss2Item struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Content     string   `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate     string   `xml:"pubDate"`
	GUID        string   `xml:"guid"`
	Author      string   `xml:"author"`
	Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Categories  []string `xml:"category"`
}

// Atom structures for feed parsing.
type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Link     []atomLink  `xml:"link"`
	Icon     string      `xml:"icon"`
	Logo     string      `xml:"logo"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Link      []atomLink `xml:"link"`
	ID        string     `xml:"id"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
	Summary   string     `xml:"summary"`
	Content   atomText   `xml:"content"`
	Author    atomAuthor `xml:"author"`
	Category  []atomCat  `xml:"category"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomCat struct {
	Term string `xml:"term,attr"`
}

// parseBlogrollFeedResponse parses an HTTP response into a feed structure for blogroll plugin.
func parseBlogrollFeedResponse(resp *http.Response) (*blogrollParsedFeed, []*models.ExternalEntry, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	// Try RSS 2.0 first
	feed, entries, err := parseRSS2Feed(body)
	if err == nil {
		// Successfully parsed as RSS 2.0
		return feed, entries, nil
	}
	rssErr := err

	// Try Atom
	feed, entries, err = parseAtomFeed(body)
	if err == nil {
		// Successfully parsed as Atom
		return feed, entries, nil
	}

	// Both parsers failed - return combined error
	return nil, nil, fmt.Errorf("failed to parse feed: %w", errors.Join(
		fmt.Errorf("rss 2.0: %w", rssErr),
		fmt.Errorf("atom: %w", err),
	))
}

// parseRSS2Feed parses an RSS 2.0 feed.
func parseRSS2Feed(data []byte) (*blogrollParsedFeed, []*models.ExternalEntry, error) {
	var feed rss2Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, nil, err
	}

	if feed.XMLName.Local != "rss" {
		return nil, nil, fmt.Errorf("not an RSS 2.0 feed")
	}

	parsed := &blogrollParsedFeed{
		Title:       feed.Channel.Title,
		Description: feed.Channel.Description,
		Language:    feed.Channel.Language,
		SiteURL:     feed.Channel.Link,
		ImageURL:    feed.Channel.Image.URL,
	}

	entries := make([]*models.ExternalEntry, 0, len(feed.Channel.Items))
	for i := range feed.Channel.Items {
		item := &feed.Channel.Items[i]

		// Parse publish date
		pubDate := parseRSSDate(item.PubDate)
		var pubDatePtr *time.Time
		if !pubDate.IsZero() {
			pubDatePtr = &pubDate
		}

		// Determine content - prefer content:encoded over description
		content := item.Content
		if content == "" {
			content = item.Description
		}

		// Determine author
		author := item.Author
		if author == "" {
			author = item.Creator
		}

		// Determine ID
		id := item.GUID
		if id == "" {
			id = item.Link
		}

		// Extract first image from content
		imageURL := extractFirstImageFromHTML(content)

		// Create entry
		entry := &models.ExternalEntry{
			ID:          id,
			Title:       item.Title,
			URL:         item.Link,
			Published:   pubDatePtr,
			Updated:     pubDatePtr,
			Author:      author,
			Content:     content,
			Description: stripBlogrollHTML(item.Description),
			ImageURL:    imageURL,
			Categories:  item.Categories,
		}

		entries = append(entries, entry)
	}

	return parsed, entries, nil
}

// parseAtomFeed parses an Atom feed.
func parseAtomFeed(data []byte) (*blogrollParsedFeed, []*models.ExternalEntry, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, nil, err
	}

	// Check if it's actually an Atom feed
	if feed.XMLName.Local != "feed" {
		return nil, nil, fmt.Errorf("not an Atom feed")
	}

	// Get site URL from links
	siteURL := ""
	for _, link := range feed.Link {
		if link.Rel == "" || link.Rel == "alternate" {
			siteURL = link.Href
			break
		}
	}

	// Prefer logo over icon
	imageURL := feed.Logo
	if imageURL == "" {
		imageURL = feed.Icon
	}

	parsed := &blogrollParsedFeed{
		Title:       feed.Title,
		Description: feed.Subtitle,
		SiteURL:     siteURL,
		ImageURL:    imageURL,
	}

	entries := make([]*models.ExternalEntry, 0, len(feed.Entries))
	for i := range feed.Entries {
		entry := &feed.Entries[i]

		// Parse dates
		pubDate := parseAtomDate(entry.Published)
		updDate := parseAtomDate(entry.Updated)

		var pubDatePtr, updDatePtr *time.Time
		if !pubDate.IsZero() {
			pubDatePtr = &pubDate
		}
		if !updDate.IsZero() {
			updDatePtr = &updDate
		}

		// Get entry URL
		entryURL := ""
		for _, link := range entry.Link {
			if link.Rel == "" || link.Rel == "alternate" {
				entryURL = link.Href
				break
			}
		}

		// Determine content
		content := entry.Content.Body
		if content == "" {
			content = entry.Summary
		}

		// Get tags
		var tags []string
		for _, cat := range entry.Category {
			if cat.Term != "" {
				tags = append(tags, cat.Term)
			}
		}

		// Extract first image from content
		imageURL := extractFirstImageFromHTML(content)

		// Create entry
		extEntry := &models.ExternalEntry{
			ID:          entry.ID,
			Title:       entry.Title,
			URL:         entryURL,
			Published:   pubDatePtr,
			Updated:     updDatePtr,
			Author:      entry.Author.Name,
			Content:     content,
			Description: stripBlogrollHTML(entry.Summary),
			ImageURL:    imageURL,
			Categories:  tags,
		}

		entries = append(entries, extEntry)
	}

	return parsed, entries, nil
}

// parseRSSDate parses an RSS date string.
func parseRSSDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// RFC822, RFC822Z, RFC1123, RFC1123Z are common RSS date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseAtomDate parses an Atom date string (RFC3339).
func parseAtomDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}
	}

	return t
}

// stripBlogrollHTML strips HTML tags from content.
func stripBlogrollHTML(s string) string {
	// Decode HTML entities
	decoded := html.UnescapeString(s)
	// Remove HTML tags
	stripped := blogrollHTMLTagRegex.ReplaceAllString(decoded, "")
	// Trim whitespace
	return strings.TrimSpace(stripped)
}

// Search config default constants.
const (
	defaultSearchPosition    = "navbar"
	defaultSearchPlaceholder = "Search..."
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
		outputDir = defaultOutputDir
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

	// Sort entries deterministically: date desc, then feed URL, entry ID, title
	sort.SliceStable(allEntries, func(i, j int) bool {
		return compareEntries(allEntries[i], allEntries[j])
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
func updateFeedFromParsed(feed *models.ExternalFeed, parsed *blogrollParsedFeed) {
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
	if feed.AvatarURL == "" && parsed.AvatarURL != "" {
		feed.AvatarURL = parsed.AvatarURL
		feed.AvatarSource = parsed.AvatarSource
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
	parsedFeed, entries, err := parseBlogrollFeedResponse(resp)
	if err != nil {
		feed.Error = fmt.Sprintf("parse: %v", err)
		return feed
	}

	// Update feed with parsed values
	updateFeedFromParsed(feed, parsedFeed)

	// Attempt avatar discovery if no avatar set from config
	// Best-effort: failures don't affect the feed
	if feed.AvatarURL == "" && feed.SiteURL != "" {
		p.discoverFeedAvatar(ctx, feed, config, timeout)
	}

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

func compareEntries(a, b *models.ExternalEntry) bool {
	// Date desc: prefer Published, then Updated
	ai := entryDate(a)
	bj := entryDate(b)
	if ai == nil && bj != nil {
		return false
	}
	if ai != nil && bj == nil {
		return true
	}
	if ai != nil && bj != nil {
		if ai.After(*bj) {
			return true
		}
		if bj.After(*ai) {
			return false
		}
	}

	// Tie-breakers for deterministic ordering
	if a.FeedURL != b.FeedURL {
		return a.FeedURL < b.FeedURL
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return a.Title < b.Title
}

func entryDate(entry *models.ExternalEntry) *time.Time {
	if entry == nil {
		return nil
	}
	if entry.Published != nil {
		return entry.Published
	}
	if entry.Updated != nil {
		return entry.Updated
	}
	return nil
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
	return os.WriteFile(outputFile, []byte(content), 0o644) //nolint:gosec // G306: Public-facing HTML needs 644 permissions
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
	updated := latestEntryDate(page.Entries)
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
		"updated":         updated,
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
	if err := os.WriteFile(outputFile, []byte(content), 0o644); err != nil { //nolint:gosec // G306: Public-facing HTML needs 644 permissions
		return err
	}

	// For HTMX pagination, also generate partial pages
	if page.PaginationType == models.PaginationHTMX {
		if err := os.MkdirAll(partialDir, 0o755); err != nil {
			return err
		}

		partialContent := p.renderReaderPartial(page.Entries, page)
		partialFile := filepath.Join(partialDir, "index.html")
		if err := os.WriteFile(partialFile, []byte(partialContent), 0o644); err != nil { //nolint:gosec // G306: Public-facing HTML needs 644 permissions
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
			"title":         feed.Title,
			"description":   feed.Description,
			"site_url":      feed.SiteURL,
			"feed_url":      feed.FeedURL,
			"image_url":     feed.ImageURL,
			"avatar_url":    feed.AvatarURL,
			"avatar_source": feed.AvatarSource,
			"category":      feed.Category,
			"tags":          feed.Tags,
			"entry_count":   feed.EntryCount,
			"last_fetched":  feed.LastFetched,
			"last_updated":  feed.LastUpdated,
			"error":         feed.Error,
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

func latestEntryDate(entries []*models.ExternalEntry) *time.Time {
	var latest *time.Time
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		candidate := entryDate(entry)
		if candidate == nil {
			continue
		}
		if latest == nil || candidate.After(*latest) {
			t := *candidate
			latest = &t
		}
	}
	return latest
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
// Note: This should eventually be refactored to use ToModelsConfig + configToMap
// from templates package for full consistency.
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
// Sets defaults (enabled=true, position="navbar") when no config is provided.
func (p *BlogrollPlugin) extractSearchConfig(extra, result map[string]interface{}) {
	// Start with defaults
	search := models.NewSearchConfig()

	// Override with config from Extra if available
	if sc, ok := extra["search"].(models.SearchConfig); ok {
		search = sc
	}

	// Determine enabled status (default: true)
	searchEnabled := true
	if search.Enabled != nil {
		searchEnabled = *search.Enabled
	}

	// Default position to "navbar" if not set
	position := search.Position
	if position == "" {
		position = defaultSearchPosition
	}

	// Default placeholder
	placeholder := search.Placeholder
	if placeholder == "" {
		placeholder = defaultSearchPlaceholder
	}

	// Default show_images to true
	showImages := true
	if search.ShowImages != nil {
		showImages = *search.ShowImages
	}

	// Default excerpt length
	excerptLength := search.ExcerptLength
	if excerptLength == 0 {
		excerptLength = 200
	}

	// Convert pagefind config
	// Use the actual pagefind bundle directory (defaults to _pagefind), not the blogroll output dir
	bundleDir := search.Pagefind.BundleDir
	if bundleDir == "" {
		bundleDir = defaultBundleDir
	}

	result["search"] = map[string]interface{}{
		"enabled":        searchEnabled,
		"position":       position,
		"placeholder":    placeholder,
		"show_images":    showImages,
		"excerpt_length": excerptLength,
		"pagefind": map[string]interface{}{
			"bundle_dir": bundleDir,
		},
	}
}

// extractStructConfigs extracts simple struct configs from config.Extra.
// All structs must be converted to maps for pongo2 template access.
func (p *BlogrollPlugin) extractStructConfigs(extra, result map[string]interface{}) {
	// Convert components config if available (fixes #316)
	// Must convert to map for pongo2 template dot-notation access
	if components, ok := extra["components"].(models.ComponentsConfig); ok {
		result["components"] = p.componentsToMap(&components)
	}
	if footer, ok := extra["footer"].(models.FooterConfig); ok {
		result["footer"] = p.footerConfigToMap(&footer)
	}
	if sidebar, ok := extra["sidebar"].(models.SidebarConfig); ok {
		result["sidebar"] = p.sidebarToMap(&sidebar)
	}
	if toc, ok := extra["toc"].(models.TocConfig); ok {
		result["toc"] = p.tocToMap(&toc)
	}
	if header, ok := extra["header"].(models.HeaderLayoutConfig); ok {
		result["header"] = p.headerToMap(&header)
	}
	if postFormats, ok := extra["post_formats"].(models.PostFormatsConfig); ok {
		result["post_formats"] = p.postFormatsToMap(&postFormats)
	}
	if head, ok := extra["head"].(models.HeadConfig); ok {
		result["head"] = p.headToMap(&head)
	}
}

// componentsToMap converts ComponentsConfig to a template-friendly map.
// This is required because pongo2 cannot access Go struct fields directly.
func (p *BlogrollPlugin) componentsToMap(c *models.ComponentsConfig) map[string]interface{} {
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

// footerConfigToMap converts FooterConfig to a template-friendly map.
func (p *BlogrollPlugin) footerConfigToMap(f *models.FooterConfig) map[string]interface{} {
	if f == nil {
		return nil
	}
	showCopyright := true
	if f.ShowCopyright != nil {
		showCopyright = *f.ShowCopyright
	}
	return map[string]interface{}{
		"text":           f.Text,
		"show_copyright": showCopyright,
	}
}

// sidebarToMap converts SidebarConfig to a template-friendly map.
func (p *BlogrollPlugin) sidebarToMap(s *models.SidebarConfig) map[string]interface{} {
	if s == nil {
		return nil
	}
	result := map[string]interface{}{
		"position": s.Position,
		"width":    s.Width,
		"title":    s.Title,
	}
	if s.Enabled != nil {
		result["enabled"] = *s.Enabled
	} else {
		result["enabled"] = true
	}
	if s.Collapsible != nil {
		result["collapsible"] = *s.Collapsible
	}
	if s.DefaultOpen != nil {
		result["default_open"] = *s.DefaultOpen
	}
	return result
}

// tocToMap converts TocConfig to a template-friendly map.
func (p *BlogrollPlugin) tocToMap(t *models.TocConfig) map[string]interface{} {
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
	if t.Enabled != nil {
		result["enabled"] = *t.Enabled
	} else {
		result["enabled"] = true
	}
	if t.Collapsible != nil {
		result["collapsible"] = *t.Collapsible
	}
	if t.DefaultOpen != nil {
		result["default_open"] = *t.DefaultOpen
	}
	return result
}

// headerToMap converts HeaderLayoutConfig to a template-friendly map.
func (p *BlogrollPlugin) headerToMap(h *models.HeaderLayoutConfig) map[string]interface{} {
	if h == nil {
		return nil
	}
	result := map[string]interface{}{
		"style": h.Style,
	}
	if h.Sticky != nil {
		result["sticky"] = *h.Sticky
	} else {
		result["sticky"] = true
	}
	if h.ShowLogo != nil {
		result["show_logo"] = *h.ShowLogo
	} else {
		result["show_logo"] = true
	}
	if h.ShowTitle != nil {
		result["show_title"] = *h.ShowTitle
	} else {
		result["show_title"] = true
	}
	if h.ShowNav != nil {
		result["show_nav"] = *h.ShowNav
	} else {
		result["show_nav"] = true
	}
	if h.ShowSearch != nil {
		result["show_search"] = *h.ShowSearch
	} else {
		result["show_search"] = true
	}
	if h.ShowThemeToggle != nil {
		result["show_theme_toggle"] = *h.ShowThemeToggle
	} else {
		result["show_theme_toggle"] = true
	}
	return result
}

// postFormatsToMap converts PostFormatsConfig to a template-friendly map.
func (p *BlogrollPlugin) postFormatsToMap(pf *models.PostFormatsConfig) map[string]interface{} {
	if pf == nil {
		return nil
	}
	htmlEnabled := true
	if pf.HTML != nil {
		htmlEnabled = *pf.HTML
	}
	return map[string]interface{}{
		"html":     htmlEnabled,
		"markdown": pf.Markdown,
		"text":     pf.Text,
		"og":       pf.OG,
	}
}

// headToMap converts HeadConfig to a template-friendly map.
func (p *BlogrollPlugin) headToMap(h *models.HeadConfig) map[string]interface{} {
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
			AuthorImage:   p.getStringFromMap(seoVal, "author_image"),
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
// This delegates to the shared templates.ThemeToMap to ensure consistency
// across all pages that use base.html.
func (p *BlogrollPlugin) themeToMap(theme models.ThemeConfig) map[string]interface{} {
	return templates.ThemeToMap(&theme)
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
			// Feed avatar - prefer AvatarURL (person/site avatar) over ImageURL (may be article image)
			avatarURL := feed.AvatarURL
			if avatarURL == "" {
				avatarURL = feed.ImageURL
			}
			if avatarURL != "" {
				sb.WriteString(fmt.Sprintf(`          <img src=%q alt="" class="feed-avatar" loading="lazy">
`, avatarURL))
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

// discoverFeedAvatar attempts to discover an avatar for a feed using IndieWeb methods.
// It tries h-card u-photo, WebFinger rel=avatar, and .well-known/avatar.
// This is best-effort and failures are silently ignored.
func (p *BlogrollPlugin) discoverFeedAvatar(ctx context.Context, feed *models.ExternalFeed, config models.ExternalFeedConfig, timeout int) {
	// Create an updater for avatar discovery
	updater := blogroll.NewUpdater(time.Duration(timeout) * time.Second)

	// Determine the resource for WebFinger lookup
	// Use handle if available (e.g., "acct:user@example.com")
	resource := ""
	if config.Handle != "" {
		// If handle looks like an email or acct URI, use it directly
		if strings.Contains(config.Handle, "@") {
			if !strings.HasPrefix(config.Handle, "acct:") {
				resource = "acct:" + config.Handle
			} else {
				resource = config.Handle
			}
		}
	}

	// Attempt avatar discovery
	avatarResult, err := updater.DiscoverAvatar(ctx, feed.SiteURL, resource)
	if err != nil {
		// Silent failure - avatar discovery is best-effort
		return
	}

	if avatarResult != nil && avatarResult.URL != "" {
		feed.AvatarURL = avatarResult.URL
		feed.AvatarSource = string(avatarResult.Source)
	}
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
