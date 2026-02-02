// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/filter"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// MentionsPlugin transforms @handle syntax into HTML links
// by resolving handles against blogroll entries.
type MentionsPlugin struct {
	// cssClass is the CSS class applied to mention links
	cssClass string
}

// NewMentionsPlugin creates a new MentionsPlugin.
func NewMentionsPlugin() *MentionsPlugin {
	return &MentionsPlugin{
		cssClass: "mention",
	}
}

// Name returns the unique name of the plugin.
func (p *MentionsPlugin) Name() string {
	return "mentions"
}

// Configure reads configuration options for the plugin.
func (p *MentionsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// First check the new Mentions config
	mentionsConfig := getMentionsConfig(config)
	if mentionsConfig.CSSClass != "" {
		p.cssClass = mentionsConfig.CSSClass
	}

	// Fall back to legacy Extra config for backwards compatibility
	if config.Extra != nil {
		if cssClass, ok := config.Extra["mentions_css_class"].(string); ok && cssClass != "" {
			p.cssClass = cssClass
		}
	}
	return nil
}

// Priority returns the plugin's priority for a given stage.
// Runs after blogroll has cached feed data in Collect stage.
func (p *MentionsPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageTransform {
		// Run late in transform stage, after blogroll config is available
		return lifecycle.PriorityLate
	}
	return lifecycle.PriorityDefault
}

// Transform processes @mentions in all post content.
func (p *MentionsPlugin) Transform(m *lifecycle.Manager) error {
	// Build handle resolution map from blogroll config
	handleMap := p.buildHandleMap(m)

	log.Printf("mentions: found %d handles in handleMap", len(handleMap))

	if len(handleMap) == 0 {
		// No blogroll entries, nothing to resolve
		log.Printf("mentions: no handle map entries, skipping")
		return nil
	}

	// Fetch metadata for all unique domains
	domainMap := p.fetchAllMetadata(m, handleMap)

	// Store in memory cache for other plugins
	m.Cache().Set("mentions_metadata", domainMap)

	// Attach metadata to mention entries
	p.attachMetadataToEntries(handleMap, domainMap)

	posts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && post.Content != ""
	})

	log.Printf("mentions: processing %d posts", len(posts))

	return m.ProcessPostsSliceConcurrently(posts, func(post *models.Post) error {
		content := p.processMentionsWithMetadata(post.Content, handleMap)
		post.Content = content
		return nil
	})
}

// mentionEntry holds resolved information for a handle.
type mentionEntry struct {
	Handle   string
	SiteURL  string
	Title    string
	Metadata *models.MentionMetadata
}

// registerFeedConfig registers a feed config's handle and aliases in the map.
func (p *MentionsPlugin) registerFeedConfig(feedConfig *models.ExternalFeedConfig, handleMap map[string]*mentionEntry) {
	if !feedConfig.IsActive() {
		return
	}

	// Determine the site URL
	siteURL := feedConfig.SiteURL
	if siteURL == "" {
		siteURL = extractSiteURL(feedConfig.URL)
	}

	if siteURL == "" {
		return
	}

	// Determine the handle
	handle := feedConfig.Handle
	if handle == "" {
		handle = extractHandleFromURL(siteURL)
	}

	if handle == "" {
		return
	}

	handle = strings.ToLower(handle)

	// Create and store the entry
	entry := &mentionEntry{
		Handle:  handle,
		SiteURL: siteURL,
		Title:   feedConfig.Title,
	}

	if _, exists := handleMap[handle]; !exists {
		handleMap[handle] = entry
	}

	// Auto-register domain alias
	domain := extractDomainFromURL(siteURL)
	if domain != "" && domain != handle {
		if _, exists := handleMap[domain]; !exists {
			handleMap[domain] = entry
		}
	}

	// Register manual aliases
	for _, alias := range feedConfig.Aliases {
		p.registerAlias(alias, entry, handleMap)
	}
}

// registerAlias registers an alias in the handle map.
func (p *MentionsPlugin) registerAlias(alias string, entry *mentionEntry, handleMap map[string]*mentionEntry) {
	normalizedAlias := strings.ToLower(alias)
	if normalizedAlias == "" {
		return
	}
	if _, exists := handleMap[normalizedAlias]; exists {
		log.Printf("warning: duplicate alias %q (first entry wins)", normalizedAlias)
		return
	}
	handleMap[normalizedAlias] = entry
}

// registerCachedFeed registers a cached feed's handle in the map.
func (p *MentionsPlugin) registerCachedFeed(feed *models.ExternalFeed, handleMap map[string]*mentionEntry) {
	if feed.SiteURL == "" {
		return
	}

	handle := feed.Config.Handle
	if handle == "" {
		handle = extractHandleFromURL(feed.SiteURL)
	}

	if handle == "" {
		return
	}

	handle = strings.ToLower(handle)

	if _, exists := handleMap[handle]; !exists {
		handleMap[handle] = &mentionEntry{
			Handle:  handle,
			SiteURL: feed.SiteURL,
			Title:   feed.Title,
		}
	}
}

// buildHandleMap builds a map of handles to their site URLs from blogroll config
// and internal posts configured via from_posts.
// Resolution order:
// 1. Explicit handle from config
// 2. Auto-generated handle from domain
// 3. Internal posts matching from_posts filters
func (p *MentionsPlugin) buildHandleMap(m *lifecycle.Manager) map[string]*mentionEntry {
	handleMap := make(map[string]*mentionEntry)

	config := m.Config()
	blogrollConfig := getBlogrollConfig(config)
	mentionsConfig := getMentionsConfig(config)

	log.Printf("mentions: blogroll config enabled: %v, feeds count: %d", blogrollConfig.Enabled, len(blogrollConfig.Feeds))

	// Register from blogroll if enabled
	if blogrollConfig.Enabled {
		// Register feed configs
		for i := range blogrollConfig.Feeds {
			p.registerFeedConfig(&blogrollConfig.Feeds[i], handleMap)
		}

		// Register cached feeds
		if cachedFeeds, ok := m.Cache().Get("blogroll_feeds"); ok {
			if feeds, ok := cachedFeeds.([]*models.ExternalFeed); ok {
				for _, feed := range feeds {
					p.registerCachedFeed(feed, handleMap)
				}
			}
		}
	}

	// Register from internal posts (from_posts sources)
	p.registerFromPosts(m, mentionsConfig, handleMap)

	return handleMap
}

// cacheKey generates a cache filename for a domain.
func (p *MentionsPlugin) cacheKey(domain string) string {
	// Normalize domain and replace dots with underscores for safe filenames
	return strings.ReplaceAll(strings.ToLower(domain), ".", "_")
}

// loadFromCache loads mention metadata from the file cache.
// Returns nil if cache doesn't exist, is expired, or has errors.
func (p *MentionsPlugin) loadFromCache(domain, cacheDir string, maxAge time.Duration) *models.MentionMetadata {
	cacheFile := filepath.Join(cacheDir, p.cacheKey(domain)+".json")

	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil // Cache file doesn't exist
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > maxAge {
		return nil
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var metadata models.MentionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil
	}

	// Additional validation
	if metadata.IsExpired(maxAge) {
		return nil
	}

	return &metadata
}

// saveToCache saves mention metadata to the file cache.
func (p *MentionsPlugin) saveToCache(metadata *models.MentionMetadata, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cacheFile := filepath.Join(cacheDir, p.cacheKey(metadata.Domain)+".json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// fetchMetadata fetches metadata for a single domain, using cache if available.
func (p *MentionsPlugin) fetchMetadata(domain, cacheDir string, maxAge time.Duration, timeout time.Duration) *models.MentionMetadata {
	// Try cache first
	if cached := p.loadFromCache(domain, cacheDir, maxAge); cached != nil {
		return cached
	}

	// Cache miss or expired, fetch from network
	url := "https://" + domain
	metadata := &models.MentionMetadata{
		Domain:      domain,
		URL:         url,
		LastFetched: time.Now(),
	}

	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		metadata.Error = fmt.Sprintf("failed to create request: %v", err)
		p.saveToCache(metadata, cacheDir) // Cache even errors to prevent repeated failed requests
		return metadata
	}

	// Set user agent to be respectful
	req.Header.Set("User-Agent", "markata-go/1.0 mentions-plugin")

	resp, err := client.Do(req)
	if err != nil {
		metadata.Error = fmt.Sprintf("HTTP request failed: %v", err)
		p.saveToCache(metadata, cacheDir)
		return metadata
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metadata.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		p.saveToCache(metadata, cacheDir)
		return metadata
	}

	// Parse HTML to extract metadata
	if err := p.extractMetadataFromHTML(resp, metadata); err != nil {
		metadata.Error = fmt.Sprintf("failed to parse HTML: %v", err)
		p.saveToCache(metadata, cacheDir)
		return metadata
	}

	// Cache successful metadata
	if err := p.saveToCache(metadata, cacheDir); err != nil {
		log.Printf("mentions: failed to cache metadata for %s: %v", domain, err)
	}

	return metadata
}

// extractMetadataFromHTML parses HTML response to extract mention metadata.
func (p *MentionsPlugin) extractMetadataFromHTML(resp *http.Response, metadata *models.MentionMetadata) error {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	htmlContent := string(body)

	// Extract title/name
	if name := p.extractMetaContent(htmlContent, []string{
		`<meta[^>]+property=["']og:site_name["'][^>]+content=["']([^"']+)["']`,
		`<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`,
		`<meta[^>]+name=["']author["'][^>]+content=["']([^"']+)["']`,
		`<title>([^<]+)</title>`,
	}); name != "" {
		metadata.Name = name
	} else {
		metadata.Name = metadata.Domain // fallback to domain
	}

	// Extract description/bio
	if bio := p.extractMetaContent(htmlContent, []string{
		`<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`,
		`<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`,
	}); bio != "" {
		metadata.Bio = bio
	}

	// Extract avatar/image
	if avatar := p.extractMetaContent(htmlContent, []string{
		`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`,
		`<link[^>]+rel=["']icon["'][^>]+href=["']([^"']+)["']`,
		`<link[^>]+rel=["']apple-touch-icon["'][^>]+href=["']([^"']+)["']`,
	}); avatar != "" {
		metadata.Avatar = p.resolveURL(avatar, metadata.URL)
	}

	return nil
}

// extractMetaContent extracts content from HTML using multiple regex patterns.
// Returns the first match from the patterns list.
func (p *MentionsPlugin) extractMetaContent(htmlContent string, patterns []string) string {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(htmlContent)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

// resolveURL makes relative URLs absolute based on the base URL.
func (p *MentionsPlugin) resolveURL(imageURL, baseURL string) string {
	if imageURL == "" {
		return ""
	}

	// If already absolute, return as-is
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}

	// Make relative URLs absolute
	if strings.HasPrefix(imageURL, "/") {
		if parsed, err := url.Parse(baseURL); err == nil {
			return parsed.Scheme + "://" + parsed.Host + imageURL
		}
	}

	// Return original if can't resolve
	return imageURL
}

// fetchAllMetadata fetches metadata for all unique domains concurrently.
func (p *MentionsPlugin) fetchAllMetadata(m *lifecycle.Manager, handleMap map[string]*mentionEntry) map[string]*models.MentionMetadata {
	config := getMentionsConfig(m.Config())

	// Extract unique domains
	domains := p.extractUniqueDomains(handleMap)
	if len(domains) == 0 {
		return make(map[string]*models.MentionMetadata)
	}

	// Parse cache duration
	cacheDuration, err := time.ParseDuration(config.GetCacheDuration())
	if err != nil {
		cacheDuration = 24 * time.Hour // fallback to 24h
	}

	timeout := time.Duration(config.GetTimeout()) * time.Second
	cacheDir := config.GetCacheDir()

	// Concurrent fetching with semaphore
	semaphore := make(chan struct{}, config.GetConcurrentRequests())
	var wg sync.WaitGroup
	metadataMap := make(map[string]*models.MentionMetadata)
	var mu sync.RWMutex

	for _, domain := range domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			metadata := p.fetchMetadata(d, cacheDir, cacheDuration, timeout)
			mu.Lock()
			metadataMap[d] = metadata
			mu.Unlock()
		}(domain)
	}

	wg.Wait()
	return metadataMap
}

// extractUniqueDomains extracts unique domains from handleMap.
func (p *MentionsPlugin) extractUniqueDomains(handleMap map[string]*mentionEntry) []string {
	domainSet := make(map[string]bool)
	for _, entry := range handleMap {
		if entry.SiteURL != "" {
			if parsed, err := url.Parse(entry.SiteURL); err == nil {
				domain := strings.ToLower(parsed.Hostname())
				if domain != "" {
					domainSet[domain] = true
				}
			}
		}
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}
	return domains
}

// attachMetadataToEntries attaches fetched metadata to mention entries.
func (p *MentionsPlugin) attachMetadataToEntries(handleMap map[string]*mentionEntry, domainMap map[string]*models.MentionMetadata) {
	for _, entry := range handleMap {
		if entry.SiteURL != "" {
			if parsed, err := url.Parse(entry.SiteURL); err == nil {
				domain := strings.ToLower(parsed.Hostname())
				if metadata, exists := domainMap[domain]; exists {
					entry.Metadata = metadata
				}
			}
		}
	}
}

// mentionRegex matches @handle patterns.
// Handles can contain alphanumeric characters, underscores, hyphens, and dots.
// This supports both simple handles like @daverupert and domain-style handles
// like @simonwillison.net. Must start with a letter and not be preceded by
// another @ or word character.
var mentionRegex = regexp.MustCompile(`((?:^|[^@\w])@([a-zA-Z][a-zA-Z0-9_.-]*))([^a-zA-Z0-9_.-]|$)`)

// mentionsCodeBlockRegex matches fenced code blocks to avoid transforming mentions inside them.
var mentionsCodeBlockRegex = regexp.MustCompile("(?s)(```[^`]*```|~~~[^~]*~~~)")

// processMentionsWithMetadata replaces @handle syntax with HTML anchor tags including metadata.
func (p *MentionsPlugin) processMentionsWithMetadata(content string, handleMap map[string]*mentionEntry) string {
	// Split content by fenced code blocks to avoid transforming mentions inside them
	codeBlocks := mentionsCodeBlockRegex.FindAllStringIndex(content, -1)

	if len(codeBlocks) == 0 {
		return p.processMentionsInText(content, handleMap)
	}

	// Process content in segments, skipping code blocks
	var result strings.Builder
	lastEnd := 0

	for _, block := range codeBlocks {
		start, end := block[0], block[1]

		// Process text before this code block
		if start > lastEnd {
			processed := p.processMentionsInText(content[lastEnd:start], handleMap)
			result.WriteString(processed)
		}

		// Keep code block unchanged
		result.WriteString(content[start:end])
		lastEnd = end
	}

	// Process any remaining text after the last code block
	if lastEnd < len(content) {
		processed := p.processMentionsInText(content[lastEnd:], handleMap)
		result.WriteString(processed)
	}

	return result.String()
}

// processMentionsInText processes @mentions in a text segment (not inside code blocks).
func (p *MentionsPlugin) processMentionsInText(text string, handleMap map[string]*mentionEntry) string {
	return mentionRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the handle from the match
		// Groups: [0]=full match, [1]=prefix+@handle, [2]=handle, [3]=suffix
		groups := mentionRegex.FindStringSubmatch(match)
		if len(groups) < 4 {
			return match
		}

		handle := strings.ToLower(groups[2])
		suffix := groups[3]

		// Look up the handle
		entry, found := handleMap[handle]
		if !found {
			// Handle not found, keep original
			return match
		}

		// Determine what prefix was captured (space, newline, etc.)
		prefix := ""
		atPos := strings.Index(match, "@")
		if atPos > 0 {
			prefix = match[:atPos]
		}

		// Build data attributes if metadata is available
		dataAttrs := ""
		if entry.Metadata != nil && entry.Metadata.IsValid() {
			dataAttrs += fmt.Sprintf(` data-name=%q`, html.EscapeString(entry.Metadata.Name))
			if entry.Metadata.Bio != "" {
				dataAttrs += fmt.Sprintf(` data-bio=%q`, html.EscapeString(entry.Metadata.Bio))
			}
			if entry.Metadata.Avatar != "" {
				dataAttrs += fmt.Sprintf(` data-avatar=%q`, html.EscapeString(entry.Metadata.Avatar))
			}
			dataAttrs += fmt.Sprintf(` data-handle=%q`, html.EscapeString("@"+entry.Handle))
		}

		// Build the HTML link
		link := fmt.Sprintf(`<a href=%q class=%q%s>@%s</a>`,
			html.EscapeString(entry.SiteURL),
			html.EscapeString(p.cssClass),
			dataAttrs,
			html.EscapeString(entry.Handle))

		return prefix + link + suffix
	})
}

// extractSiteURL extracts the base site URL from a feed URL.
func extractSiteURL(feedURL string) string {
	if feedURL == "" {
		return ""
	}

	parsed, err := url.Parse(feedURL)
	if err != nil {
		return ""
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	// Return scheme + host
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

// extractHandleFromURL extracts a handle from a URL's domain.
// For example:
// - "https://daverupert.com" -> "daverupert"
// - "https://www.example.com" -> "example"
// - "https://blog.jane.dev" -> "jane"
func extractHandleFromURL(siteURL string) string {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	// Remove common prefixes
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "blog.")

	// Get the first part of the domain (before the TLD)
	parts := strings.Split(host, ".")
	if len(parts) == 0 {
		return ""
	}

	// Take the first part (subdomain or main domain name)
	handle := parts[0]

	// Clean up the handle - only allow alphanumeric, underscores, and hyphens
	cleanHandle := strings.Builder{}
	for _, r := range handle {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			cleanHandle.WriteRune(r)
		}
	}

	return strings.ToLower(cleanHandle.String())
}

// extractDomainFromURL extracts the full domain from a URL.
// For example:
// - "https://daverupert.com" -> "daverupert.com"
// - "https://www.example.com" -> "www.example.com"
// - "https://blog.jane.dev" -> "blog.jane.dev"
func extractDomainFromURL(siteURL string) string {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	return strings.ToLower(host)
}

// registerFromPosts registers handles from internal posts matching from_posts sources.
func (p *MentionsPlugin) registerFromPosts(m *lifecycle.Manager, config models.MentionsConfig, handleMap map[string]*mentionEntry) {
	if len(config.FromPosts) == 0 {
		return
	}

	posts := m.Posts()

	for _, source := range config.FromPosts {
		if source.Filter == "" {
			continue
		}

		// Parse and apply the filter
		f, err := filter.Parse(source.Filter)
		if err != nil {
			log.Printf("mentions: error parsing from_posts filter %q: %v", source.Filter, err)
			continue
		}

		matchedPosts := f.MatchAll(posts)

		for _, post := range matchedPosts {
			p.registerPostAsHandle(post, source, handleMap)
		}
	}
}

// registerPostAsHandle registers a single post as a handle source.
func (p *MentionsPlugin) registerPostAsHandle(post *models.Post, source models.MentionPostSource, handleMap map[string]*mentionEntry) {
	// Get the handle from the specified field or fall back to slug
	handle := p.getHandleFromPost(post, source.HandleField)
	if handle == "" {
		return
	}

	handle = strings.ToLower(handle)

	// Get the title for the entry
	title := ""
	if post.Title != nil {
		title = *post.Title
	}

	// Create the entry with the post's Href as the URL
	entry := &mentionEntry{
		Handle:  handle,
		SiteURL: post.Href,
		Title:   title,
	}

	// Register the handle (first entry wins)
	if _, exists := handleMap[handle]; !exists {
		handleMap[handle] = entry
	}

	// Register aliases if configured
	if source.AliasesField != "" {
		aliases := p.getAliasesFromPost(post, source.AliasesField)
		for _, alias := range aliases {
			p.registerAlias(alias, entry, handleMap)
		}
	}
}

// getHandleFromPost extracts a handle from a post's frontmatter field or falls back to slug.
func (p *MentionsPlugin) getHandleFromPost(post *models.Post, fieldName string) string {
	// If no field specified, use slug
	if fieldName == "" {
		return post.Slug
	}

	// Try to get the field from Extra
	if post.Extra == nil {
		return post.Slug
	}

	value, ok := post.Extra[fieldName]
	if !ok {
		return post.Slug
	}

	// Handle string value
	if str, ok := value.(string); ok && str != "" {
		return str
	}

	// Fall back to slug
	return post.Slug
}

// getAliasesFromPost extracts aliases from a post's frontmatter field.
func (p *MentionsPlugin) getAliasesFromPost(post *models.Post, fieldName string) []string {
	if fieldName == "" || post.Extra == nil {
		return nil
	}

	value, ok := post.Extra[fieldName]
	if !ok {
		return nil
	}

	// Handle []string
	if aliases, ok := value.([]string); ok {
		return aliases
	}

	// Handle []interface{} (common from YAML/JSON parsing)
	if aliases, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			if str, ok := alias.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}

	return nil
}

// getMentionsConfig retrieves mentions configuration from the manager config.
func getMentionsConfig(config *lifecycle.Config) models.MentionsConfig {
	if config.Extra == nil {
		return models.NewMentionsConfig()
	}

	// First check for typed MentionsConfig in Extra
	if mentions, ok := config.Extra["mentions"]; ok {
		if mc, ok := mentions.(models.MentionsConfig); ok {
			return mc
		}
	}

	return models.NewMentionsConfig()
}

// Ensure MentionsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*MentionsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*MentionsPlugin)(nil)
	_ lifecycle.TransformPlugin = (*MentionsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*MentionsPlugin)(nil)
)
