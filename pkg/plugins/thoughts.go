package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// ThoughtsPlugin manages microblog content and PESOS functionality.
type ThoughtsPlugin struct {
	// Configuration
	thoughtsDir string
	cacheDir    string
	enabled     bool
	maxItems    int
	sources     map[string]*ThoughtSource
	syndication *ThoughtSyndicationConfig

	// Runtime state
	cachedThoughts []*models.Post
}

// ThoughtSource defines an external source for importing thoughts.
type ThoughtSource struct {
	Type     string `json:"type"` // "mastodon", "twitter", "rss"
	URL      string `json:"url"`
	Handle   string `json:"handle"`
	Active   bool   `json:"active"`
	MaxItems int    `json:"max_items"`
}

// ThoughtSyndicationConfig defines syndication settings for thoughts.
type ThoughtSyndicationConfig struct {
	Enabled   bool                  `json:"enabled"`
	Mastodon  *MastodonSyndication  `json:"mastodon"`
	Twitter   *TwitterSyndication   `json:"twitter"`
	MicroBlog *MicroBlogSyndication `json:"micro_blog"`
}

// MastodonSyndication configures Mastodon posting.
type MastodonSyndication struct {
	AccessToken    string `json:"access_token"`
	InstanceURL    string `json:"instance_url"`
	CharacterLimit int    `json:"character_limit"`
}

// TwitterSyndication configures Twitter posting.
type TwitterSyndication struct {
	APIKey         string `json:"api_key"`
	APISecret      string `json:"api_secret"`
	AccessToken    string `json:"access_token"`
	AccessSecret   string `json:"access_secret"`
	CharacterLimit int    `json:"character_limit"`
}

// MicroBlogSyndication configures Micro.blog posting.
type MicroBlogSyndication struct {
	APIToken       string `json:"api_token"`
	SiteURL        string `json:"site_url"`
	CharacterLimit int    `json:"character_limit"`
}

// NewThoughtsPlugin creates a new thoughts plugin instance.
func NewThoughtsPlugin() *ThoughtsPlugin {
	return &ThoughtsPlugin{
		thoughtsDir: "thoughts",
		cacheDir:    "cache/thoughts",
		enabled:     true,
		maxItems:    200,
		sources:     make(map[string]*ThoughtSource),
		syndication: &ThoughtSyndicationConfig{
			Enabled: false,
		},
	}
}

// Name returns the plugin name.
func (p *ThoughtsPlugin) Name() string {
	return "thoughts"
}

// Configure loads thoughts plugin configuration.
func (p *ThoughtsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Get thoughts configuration from Extra map
	var thoughtsConfig map[string]interface{}
	if config.Extra != nil {
		if thoughts, ok := config.Extra["thoughts"]; ok {
			if thoughtsMap, ok := thoughts.(map[string]interface{}); ok {
				thoughtsConfig = thoughtsMap
			}
		}
	}

	// Apply configuration if found
	if thoughtsConfig != nil {
		if enabled, ok := thoughtsConfig["enabled"].(bool); ok {
			p.enabled = enabled
		}

		if dir, ok := thoughtsConfig["thoughts_dir"].(string); ok {
			p.thoughtsDir = dir
		}

		if cacheDir, ok := thoughtsConfig["cache_dir"].(string); ok {
			p.cacheDir = cacheDir
		}

		if maxItems, ok := thoughtsConfig["max_items"].(int); ok {
			p.maxItems = maxItems
		}

		// Load sources
		if sources, ok := thoughtsConfig["sources"].(map[string]interface{}); ok {
			p.loadSources(sources)
		}

		// Load syndication config
		if syndConfig, ok := thoughtsConfig["syndication"].(map[string]interface{}); ok {
			p.loadSyndicationConfig(syndConfig)
		}
	}

	// Create cache directory if it doesn't exist
	if p.enabled && p.cacheDir != "" {
		if err := os.MkdirAll(p.cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory %s: %w", p.cacheDir, err)
		}
	}

	return nil
}

// loadSources configures external thought sources.
func (p *ThoughtsPlugin) loadSources(sources map[string]interface{}) {
	for name, config := range sources {
		if sourceMap, ok := config.(map[string]interface{}); ok {
			source := &ThoughtSource{
				Type:     "rss",
				Active:   true,
				MaxItems: 50,
			}

			if t, ok := sourceMap["type"].(string); ok {
				source.Type = t
			}
			if url, ok := sourceMap["url"].(string); ok {
				source.URL = url
			}
			if handle, ok := sourceMap["handle"].(string); ok {
				source.Handle = handle
			}
			if active, ok := sourceMap["active"].(bool); ok {
				source.Active = active
			}
			if maxItems, ok := sourceMap["max_items"].(int); ok {
				source.MaxItems = maxItems
			}

			p.sources[name] = source
		}
	}
}

// loadSyndicationConfig configures syndication settings.
func (p *ThoughtsPlugin) loadSyndicationConfig(config map[string]interface{}) {
	if enabled, ok := config["enabled"].(bool); ok {
		p.syndication.Enabled = enabled
	}

	if mastodon, ok := config["mastodon"].(map[string]interface{}); ok {
		p.syndication.Mastodon = &MastodonSyndication{}
		if token, ok := mastodon["access_token"].(string); ok {
			p.syndication.Mastodon.AccessToken = token
		}
		if url, ok := mastodon["instance_url"].(string); ok {
			p.syndication.Mastodon.InstanceURL = url
		}
		if limit, ok := mastodon["character_limit"].(int); ok {
			p.syndication.Mastodon.CharacterLimit = limit
		}
	}

	if twitter, ok := config["twitter"].(map[string]interface{}); ok {
		p.syndication.Twitter = &TwitterSyndication{}
		if key, ok := twitter["api_key"].(string); ok {
			p.syndication.Twitter.APIKey = key
		}
		if secret, ok := twitter["api_secret"].(string); ok {
			p.syndication.Twitter.APISecret = secret
		}
		if token, ok := twitter["access_token"].(string); ok {
			p.syndication.Twitter.AccessToken = token
		}
		if accessSecret, ok := twitter["access_secret"].(string); ok {
			p.syndication.Twitter.AccessSecret = accessSecret
		}
		if limit, ok := twitter["character_limit"].(int); ok {
			p.syndication.Twitter.CharacterLimit = limit
		}
	}
}

// Validate ensures the plugin configuration is valid.
func (p *ThoughtsPlugin) Validate(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Validate thoughts directory exists
	if p.thoughtsDir != "" {
		if _, err := os.Stat(p.thoughtsDir); os.IsNotExist(err) {
			// Create thoughts directory if it doesn't exist
			if err := os.MkdirAll(p.thoughtsDir, 0755); err != nil {
				return fmt.Errorf("failed to create thoughts directory %s: %w", p.thoughtsDir, err)
			}
		}
	}

	// Validate sources
	for name, source := range p.sources {
		if source.Active && source.URL == "" {
			return fmt.Errorf("source %s is active but has no URL", name)
		}
	}

	return nil
}

// Glob adds thought files to the file list.
func (p *ThoughtsPlugin) Glob(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Add thoughts directory to existing glob patterns
	if p.thoughtsDir != "" {
		thoughtPattern := filepath.Join(p.thoughtsDir, "*.md")
		config := m.Config()
		config.GlobPatterns = append(config.GlobPatterns, thoughtPattern)
	}

	return nil
}

// Load processes thought files and adds external thoughts.
func (p *ThoughtsPlugin) Load(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Import thoughts from external sources
	if err := p.importExternalThoughts(m); err != nil {
		return fmt.Errorf("failed to import external thoughts: %w", err)
	}

	return nil
}

// Collect processes thoughts for feed generation and syndication.
func (p *ThoughtsPlugin) Collect(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Process thoughts for syndication if enabled
	if p.syndication.Enabled {
		if err := p.processSyndication(m); err != nil {
			return fmt.Errorf("failed to process syndication: %w", err)
		}
	}

	return nil
}

// importExternalThoughts fetches thoughts from external sources.
func (p *ThoughtsPlugin) importExternalThoughts(m *lifecycle.Manager) error {
	for name, source := range p.sources {
		if !source.Active {
			continue
		}

		switch source.Type {
		case "mastodon", "rss":
			if err := p.importFromRSS(m, source, sourceName); err != nil {
				return fmt.Errorf("failed to import from %s (%s): %w", sourceName, source.URL, err)
			}
		case "json":
			if err := p.importFromJSON(m, source, sourceName); err != nil {
				return fmt.Errorf("failed to import from JSON %s (%s): %w", sourceName, source.URL, err)
			}
		case "twitter":
			if err := p.importFromTwitter(m, source, sourceName); err != nil {
				return fmt.Errorf("failed to import from Twitter %s: %w", sourceName, err)
			}
		default:
			return fmt.Errorf("unsupported source type: %s", source.Type)
		}
		default:
			return fmt.Errorf("unsupported source type: %s", source.Type)
		}

	return nil
}

// importFromRSS imports thoughts from RSS/Atom feeds (including Mastodon).
func (p *ThoughtsPlugin) importFromRSS(m *lifecycle.Manager, source *ThoughtSource, sourceName string) error {
	// Check cache first
	cacheKey := fmt.Sprintf("thoughts_feed_%s", sourceName)
	if cached := p.getCachedEntries(cacheKey); cached != nil {
		for i, entry := range cached {
			if i >= source.MaxItems {
				break
			}
			post := p.convertExternalEntryToPost(entry, source, sourceName)
			if post != nil {
				m.AddPost(post)
			}
		}
		return nil
	}

	resp, err := http.Get(source.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	_, entries, err := parseFeedResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// Cache entries
	if err := p.cacheEntries(cacheKey, entries); err != nil {
		// Log error but don't fail import
		fmt.Printf("Warning: failed to cache entries for %s: %v\n", sourceName, err)
	}

	// Convert entries to posts and add to manager
	limit := source.MaxItems
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	for i := 0; i < limit; i++ {
		entry := entries[i]
		post := p.convertExternalEntryToPost(entry, source, sourceName)
		if post != nil {
			m.AddPost(post)
		}
	}

	return nil
}

// importFromJSON imports thoughts from JSON API (like thoughts.waylonwalker.com).
func (p *ThoughtsPlugin) importFromJSON(m *lifecycle.Manager, source *ThoughtSource, sourceName string) error {
	// Check cache first
	cacheKey := fmt.Sprintf("thoughts_feed_%s", sourceName)
	if cached := p.getCachedJSONEntries(cacheKey); cached != nil {
		for i, entry := range cached {
			if i >= source.MaxItems {
				break
			}
			post := p.convertJSONEntryToPost(entry, source, sourceName)
			if post != nil {
				m.AddPost(post)
			}
		}
		return nil
	}

	// Fetch from JSON API
	resp, err := http.Get(source.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch JSON feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JSON API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON array
	var jsonData []map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to external entries
	entries := make([]*models.ExternalEntry, 0, len(jsonData))
	for _, item := range jsonData {
		entry := p.convertJSONToExternalEntry(item)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	// Cache entries
	if err := p.cacheJSONEntries(cacheKey, entries); err != nil {
		fmt.Printf("Warning: failed to cache JSON entries for %s: %v\n", sourceName, err)
	}

	// Convert entries to posts and add to manager
	limit := source.MaxItems
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	for i := 0; i < limit; i++ {
		entry := entries[i]
		post := p.convertJSONEntryToPost(entry, source, sourceName)
		if post != nil {
			m.AddPost(post)
		}
	}

	return nil
}

// convertJSONToExternalEntry converts JSON item to ExternalEntry.
func (p *ThoughtsPlugin) convertJSONToExternalEntry(item map[string]interface{}) *models.ExternalEntry {
	entry := &models.ExternalEntry{}

	// Extract title
	if title, ok := item["title"].(string); ok {
		entry.Title = title
	}

	// Extract URL
	if url, ok := item["link"].(string); ok {
		entry.URL = url
	}

	// Extract content/message
	if message, ok := item["message"].(string); ok {
		entry.Content = message
		entry.Description = p.truncateText(message, 200)
	}

	// Extract ID
	if id, ok := item["id"].(float64); ok {
		entry.ID = fmt.Sprintf("%.0f", id)
	}

	// Extract date
	if dateStr, ok := item["date"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
			entry.Published = &parsed
		}
	}

	// Extract author
	if author, ok := item["author"].(map[string]interface{}); ok {
		if username, ok := author["username"].(string); ok {
			entry.Author = username
		}
	}

	// Extract tags
	if tags, ok := item["tags"].(string); ok {
		if tags != "" {
			entry.Categories = strings.Split(tags, ",")
			for i := range entry.Categories {
				entry.Categories[i] = strings.TrimSpace(entry.Categories[i])
			}
		}
	}

	return entry
}

// truncateText truncates text to specified length with smart truncation.
func (p *ThoughtsPlugin) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	
	// Try to truncate at word boundary
	truncated := text[:maxLen]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
		return truncated[:lastSpace] + "..."
	}
	
	return truncated + "..."
}
			post := p.convertExternalEntryToPost(entry, source, sourceName)
			if post != nil {
				m.AddPost(post)
			}
		}
		return nil
	}

	resp, err := http.Get(source.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	_, entries, err := parseFeedResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// Cache the entries
	if err := p.cacheEntries(cacheKey, entries); err != nil {
		// Log error but don't fail the import
		fmt.Printf("Warning: failed to cache entries for %s: %v\n", sourceName, err)
	}

	// Convert entries to posts and add to manager
	limit := source.MaxItems
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	for i := 0; i < limit; i++ {
		entry := entries[i]
		post := p.convertExternalEntryToPost(entry, source, sourceName)
		if post != nil {
			m.AddPost(post)
		}
	}

	return nil
}

// importFromTwitter imports thoughts from Twitter (placeholder).
func (p *ThoughtsPlugin) importFromTwitter(m *lifecycle.Manager, source *ThoughtSource, sourceName string) error {
	// TODO: Implement Twitter API integration
	return fmt.Errorf("Twitter import not yet implemented")
}

// convertExternalEntryToPost converts an external entry to a Post model.
func (p *ThoughtsPlugin) convertExternalEntryToPost(entry *models.ExternalEntry, source *ThoughtSource, sourceName string) *models.Post {
	// Normalize content for thoughts
	content := p.normalizeThoughtContent(entry.Content, source.Type)
	if content == "" {
		content = p.normalizeThoughtContent(entry.Description, source.Type)
	}
	if content == "" {
		content = entry.Title
	}

	// Create title from content or entry title
	title := entry.Title
	if title == "" {
		// Use first 50 chars of content as title
		if len(content) > 50 {
			title = content[:47] + "..."
		} else {
			title = content
		}
	}

	// Generate a thought-specific slug
	slug := p.generateThoughtSlug(entry, sourceName)
	post := models.NewPost(fmt.Sprintf("thoughts/%s.md", slug))

	// Set basic fields
	post.Title = &title
	post.Description = &entry.Description
	post.Template = "thought.html"

	// Set published date
	if entry.Published != nil {
		post.Date = entry.Published
		post.Published = true
	} else {
		now := time.Now()
		post.Date = &now
		post.Published = true
	}

	// Set content
	post.Content = content

	// Add thought-specific metadata
	post.Set("thought_source", sourceName)
	post.Set("thought_type", source.Type)
	post.Set("original_url", entry.URL)
	post.Set("external_id", entry.ID)

	if entry.Author != "" {
		post.Set("author", entry.Author)
	}

	// Add source handle if available
	if source.Handle != "" {
		post.Set("source_handle", source.Handle)
	}

	// Add categories as tags, including the source type
	post.Tags = append(post.Tags, source.Type)
	if len(entry.Categories) > 0 {
		post.Tags = append(post.Tags, entry.Categories...)
	}

	// Add syndication metadata
	post.Set("syndicate_to", []string{}) // Empty by default for imported content
	post.Set("syndication_urls", make(map[string]string))

	// Add external metadata for templates
	post.Set("is_external_thought", true)
	if entry.ImageURL != "" {
		post.Set("image_url", entry.ImageURL)
	}

	// Generate slug and href
	post.GenerateSlug()
	post.GenerateHref()

	return post
}

// processSyndication handles posting thoughts to external platforms.
func (p *ThoughtsPlugin) processSyndication(m *lifecycle.Manager) error {
	posts := m.Posts()

	for _, post := range posts {
		// Only process thoughts
		if post.Template != "thought.html" && !strings.HasPrefix(post.Path, "thoughts/") {
			continue
		}

		// Skip if already syndicated
		if post.Has("syndicated") && post.Get("syndicated").(bool) {
			continue
		}

		// Get syndication targets
		targets := []string{}
		if post.Has("syndicate_to") {
			if t, ok := post.Get("syndicate_to").([]string); ok {
				targets = t
			}
		}

		// Syndicate to each target
		for _, target := range targets {
			if err := p.syndicateThought(post, target); err != nil {
				// Log error but continue with other syndications
				fmt.Printf("Failed to syndicate thought %s to %s: %v\n", post.Slug, target, err)
			}
		}

		// Mark as syndicated
		post.Set("syndicated", true)
	}

	return nil
}

// syndicateThought posts a thought to the specified platform.
func (p *ThoughtsPlugin) syndicateThought(post *models.Post, platform string) error {
	switch platform {
	case "mastodon":
		return p.postToMastodon(post)
	case "twitter":
		return p.postToTwitter(post)
	case "micro_blog":
		return p.postToMicroBlog(post)
	default:
		return fmt.Errorf("unsupported syndication platform: %s", platform)
	}
}

// postToMastodon posts a thought to Mastodon.
func (p *ThoughtsPlugin) postToMastodon(post *models.Post) error {
	if p.syndication.Mastodon == nil {
		return fmt.Errorf("Mastodon syndication not configured")
	}

	// TODO: Implement Mastodon API integration
	return fmt.Errorf("Mastodon syndication not yet implemented")
}

// postToTwitter posts a thought to Twitter.
func (p *ThoughtsPlugin) postToTwitter(post *models.Post) error {
	if p.syndication.Twitter == nil {
		return fmt.Errorf("Twitter syndication not configured")
	}

	// TODO: Implement Twitter API integration
	return fmt.Errorf("Twitter syndication not yet implemented")
}

// postToMicroBlog posts a thought to Micro.blog.
func (p *ThoughtsPlugin) postToMicroBlog(post *models.Post) error {
	if p.syndication.MicroBlog == nil {
		return fmt.Errorf("Micro.blog syndication not configured")
	}

	// TODO: Implement Micro.blog API integration
	return fmt.Errorf("Micro.blog syndication not yet implemented")
}

// getCachedEntries retrieves cached entries for a feed.
func (p *ThoughtsPlugin) getCachedEntries(cacheKey string) []*models.ExternalEntry {
	if p.cacheDir == "" {
		return nil
	}

	cacheFile := filepath.Join(p.cacheDir, cacheKey+".json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var entries []*models.ExternalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	return entries
}

// cacheEntries stores entries in cache.
func (p *ThoughtsPlugin) cacheEntries(cacheKey string, entries []*models.ExternalEntry) error {
	if p.cacheDir == "" {
		return nil
	}

	cacheFile := filepath.Join(p.cacheDir, cacheKey+".json")
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// getCachedJSONEntries retrieves cached JSON entries for a feed.
func (p *ThoughtsPlugin) getCachedJSONEntries(cacheKey string) []*models.ExternalEntry {
	if p.cacheDir == "" {
		return nil
	}

	cacheFile := filepath.Join(p.cacheDir, cacheKey+"_json.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var entries []*models.ExternalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	return entries
}

// cacheJSONEntries stores JSON entries in cache.
func (p *ThoughtsPlugin) cacheJSONEntries(cacheKey string, entries []*models.ExternalEntry) error {
	if p.cacheDir == "" {
		return nil
	}

	cacheFile := filepath.Join(p.cacheDir, cacheKey+"_json.json")
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// convertJSONEntryToPost converts a JSON thought entry to a Post model.
func (p *ThoughtsPlugin) convertJSONEntryToPost(entry *models.ExternalEntry, source *ThoughtSource, sourceName string) *models.Post {
	// Use the same conversion logic as RSS entries
	return p.convertExternalEntryToPost(entry, source, sourceName)
}

// normalizeThoughtContent processes content for better display.
func (p *ThoughtsPlugin) normalizeThoughtContent(content string, sourceType string) string {
	// Remove HTML tags for plain text thoughts
	if sourceType == "mastodon" || sourceType == "twitter" {
		// Simple HTML tag removal
		content = strings.ReplaceAll(content, "<p>", "")
		content = strings.ReplaceAll(content, "</p>", "\n\n")
		content = strings.ReplaceAll(content, "<br>", "\n")
		content = strings.ReplaceAll(content, "<br/>", "\n")
		content = strings.ReplaceAll(content, "<br />", "\n")
	}

	// Trim whitespace
	content = strings.TrimSpace(content)

	// Limit content length for micro thoughts
	if len(content) > 500 {
		content = content[:497] + "..."
	}

	return content
}

// generateThoughtSlug creates a unique slug for a thought.
func (p *ThoughtsPlugin) generateThoughtSlug(entry *models.ExternalEntry, sourceName string) string {
	// Create base slug from title or content
	base := entry.Title
	if base == "" {
		base = entry.Description
	}
	if base == "" {
		base = entry.Content
	}

	// Truncate and normalize
	if len(base) > 50 {
		base = base[:50]
	}

	// Create slug
	slug := strings.ToLower(base)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens
	slug = strings.Trim(slug, "-")

	// Add source and timestamp for uniqueness
	timestamp := ""
	if entry.Published != nil {
		timestamp = fmt.Sprintf("-%d", entry.Published.Unix())
	}

	return fmt.Sprintf("%s-%s%s", slug, sourceName, timestamp)
}

// Ensure the plugin implements the required interfaces
var (
	_ lifecycle.Plugin          = (*ThoughtsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*ThoughtsPlugin)(nil)
	_ lifecycle.ValidatePlugin  = (*ThoughtsPlugin)(nil)
	_ lifecycle.GlobPlugin      = (*ThoughtsPlugin)(nil)
	_ lifecycle.LoadPlugin      = (*ThoughtsPlugin)(nil)
	_ lifecycle.CollectPlugin   = (*ThoughtsPlugin)(nil)
)
