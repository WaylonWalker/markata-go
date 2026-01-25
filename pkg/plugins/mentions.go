// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"strings"

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

// buildHandleMap builds a map of handles to their site URLs from blogroll config.
// Resolution order:
// 1. Explicit handle from config
// 2. Auto-generated handle from domain
func (p *MentionsPlugin) buildHandleMap(m *lifecycle.Manager) map[string]*mentionEntry {
	handleMap := make(map[string]*mentionEntry)

	config := m.Config()
	blogrollConfig := getBlogrollConfig(config)

	if !blogrollConfig.Enabled {
		return handleMap
	}

	for i := range blogrollConfig.Feeds {
		feedConfig := &blogrollConfig.Feeds[i]
		if !feedConfig.IsActive() {
			continue
		}

		// Determine the site URL
		siteURL := feedConfig.SiteURL
		if siteURL == "" {
			// Try to extract site URL from feed URL
			siteURL = extractSiteURL(feedConfig.URL)
		}

		if siteURL == "" {
			// Can't resolve without a site URL
			continue
		}

		// Determine the handle
		handle := feedConfig.Handle
		if handle == "" {
			// Auto-generate handle from domain
			handle = extractHandleFromURL(siteURL)
		}

		if handle == "" {
			continue
		}

		// Normalize handle to lowercase
		handle = strings.ToLower(handle)

		// Create the mention entry
		entry := &mentionEntry{
			Handle:  handle,
			SiteURL: siteURL,
			Title:   feedConfig.Title,
		}

		// Store in map (first entry wins for duplicates)
		if _, exists := handleMap[handle]; !exists {
			handleMap[handle] = entry
		}

		// Register aliases for this handle
		for _, alias := range feedConfig.Aliases {
			normalizedAlias := strings.ToLower(alias)
			if normalizedAlias == "" {
				continue
			}
			if _, exists := handleMap[normalizedAlias]; exists {
				// Log warning for duplicate alias (first entry wins)
				log.Printf("warning: duplicate alias %q (first entry wins)", normalizedAlias)
				continue
			}
			// Alias points to the same entry with the canonical handle
			handleMap[normalizedAlias] = entry
		}
	}

	// Also check for cached feeds that might have site URLs populated from fetching
	if cachedFeeds, ok := m.Cache().Get("blogroll_feeds"); ok {
		if feeds, ok := cachedFeeds.([]*models.ExternalFeed); ok {
			for _, feed := range feeds {
				if feed.SiteURL == "" {
					continue
				}

				// Determine handle
				handle := feed.Config.Handle
				if handle == "" {
					handle = extractHandleFromURL(feed.SiteURL)
				}

				if handle == "" {
					continue
				}

				handle = strings.ToLower(handle)

				// Add to map if not already present
				if _, exists := handleMap[handle]; !exists {
					handleMap[handle] = &mentionEntry{
						Handle:  handle,
						SiteURL: feed.SiteURL,
						Title:   feed.Title,
					}
				}
			}
		}
	}

	return handleMap
}

// mentionRegex matches @handle patterns.
// Handles can contain alphanumeric characters, underscores, hyphens, and dots.
// This supports both simple handles like @daverupert and domain-style handles
// like @simonwillison.net. Must start with a letter and not be preceded by
// another @ or word character.
var mentionRegex = regexp.MustCompile(`((?:^|[^@\w])@([a-zA-Z][a-zA-Z0-9_.-]*))([^a-zA-Z0-9_.-]|$)`)

// processMentions replaces @handle syntax with HTML anchor tags.
func (p *MentionsPlugin) processMentions(content string, handleMap map[string]*mentionEntry) string {
	// Split content by fenced code blocks to avoid transforming mentions inside them
	codeBlockRegex := regexp.MustCompile("(?s)(```[^`]*```|~~~[^~]*~~~)")

	codeBlocks := codeBlockRegex.FindAllStringIndex(content, -1)

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

// Ensure MentionsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*MentionsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*MentionsPlugin)(nil)
	_ lifecycle.TransformPlugin = (*MentionsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*MentionsPlugin)(nil)
)
