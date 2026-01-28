// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"strings"

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
	// Build the handle resolution map from blogroll config
	handleMap := p.buildHandleMap(m)

	if len(handleMap) == 0 {
		// No blogroll entries, nothing to resolve
		return nil
	}

	// Process each post
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		content := p.processMentions(post.Content, handleMap)
		post.Content = content

		return nil
	})
}

// mentionEntry holds resolved information for a handle.
type mentionEntry struct {
	Handle  string
	SiteURL string
	Title   string
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

// mentionRegex matches @handle patterns.
// Handles can contain alphanumeric characters, underscores, hyphens, and dots.
// This supports both simple handles like @daverupert and domain-style handles
// like @simonwillison.net. Must start with a letter and not be preceded by
// another @ or word character.
var mentionRegex = regexp.MustCompile(`((?:^|[^@\w])@([a-zA-Z][a-zA-Z0-9_.-]*))([^a-zA-Z0-9_.-]|$)`)

// mentionsCodeBlockRegex matches fenced code blocks to avoid transforming mentions inside them.
var mentionsCodeBlockRegex = regexp.MustCompile("(?s)(```[^`]*```|~~~[^~]*~~~)")

// processMentions replaces @handle syntax with HTML anchor tags.
func (p *MentionsPlugin) processMentions(content string, handleMap map[string]*mentionEntry) string {
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

		// Build the HTML link
		link := fmt.Sprintf(`<a href=%q class=%q>@%s</a>`,
			html.EscapeString(entry.SiteURL),
			html.EscapeString(p.cssClass),
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
