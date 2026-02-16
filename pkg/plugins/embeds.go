// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// EmbedsPlugin processes embed syntax in markdown content.
// It supports two types of embeds:
// - Internal embeds: ![[slug]] or ![[slug|display text]] - embed another post from the same site
// - External embeds: ![embed](https://example.com/article) - embed external content with OG metadata
//
// The plugin runs in the Transform stage, before markdown rendering.
type EmbedsPlugin struct {
	config     models.EmbedsConfig
	httpClient *http.Client
}

// NewEmbedsPlugin creates a new EmbedsPlugin with default settings.
func NewEmbedsPlugin() *EmbedsPlugin {
	config := models.NewEmbedsConfig()
	return &EmbedsPlugin{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// Name returns the unique name of the plugin.
func (p *EmbedsPlugin) Name() string {
	return "embeds"
}

// Priority returns the plugin's priority for a given stage.
// This plugin runs early in the transform stage, before wikilinks.
func (p *EmbedsPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageTransform {
		return lifecycle.PriorityEarly // Run before wikilinks and other transforms
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "embeds" key.
func (p *EmbedsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	pluginConfig, ok := config.Extra["embeds"]
	if !ok {
		return nil
	}

	if cfgMap, ok := pluginConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if internalCardClass, ok := cfgMap["internal_card_class"].(string); ok && internalCardClass != "" {
			p.config.InternalCardClass = internalCardClass
		}
		if externalCardClass, ok := cfgMap["external_card_class"].(string); ok && externalCardClass != "" {
			p.config.ExternalCardClass = externalCardClass
		}
		if fetchExternal, ok := cfgMap["fetch_external"].(bool); ok {
			p.config.FetchExternal = fetchExternal
		}
		if cacheDir, ok := cfgMap["cache_dir"].(string); ok && cacheDir != "" {
			p.config.CacheDir = cacheDir
		}
		if timeout, ok := cfgMap["timeout"].(int); ok && timeout > 0 {
			p.config.Timeout = timeout
			p.httpClient.Timeout = time.Duration(timeout) * time.Second
		}
		if fallbackTitle, ok := cfgMap["fallback_title"].(string); ok && fallbackTitle != "" {
			p.config.FallbackTitle = fallbackTitle
		}
		if showImage, ok := cfgMap["show_image"].(bool); ok {
			p.config.ShowImage = showImage
		}
		if attachmentsPrefix, ok := cfgMap["attachments_prefix"].(string); ok && attachmentsPrefix != "" {
			p.config.AttachmentsPrefix = attachmentsPrefix
		}
	}

	return nil
}

// Transform processes embed syntax in all post content.
func (p *EmbedsPlugin) Transform(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	// Use the shared PostIndex from the lifecycle manager
	idx := m.PostIndex()

	posts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && post.Content != ""
	})

	return m.ProcessPostsSliceConcurrently(posts, func(post *models.Post) error {
		content := p.processAttachmentEmbeds(post.Content)
		content, dependencies := p.processInternalEmbeds(content, idx, post)
		content = p.processExternalEmbeds(content, post)
		post.Content = content

		// Record dependencies for incremental build cache
		for _, dep := range dependencies {
			post.AddDependency(dep)
		}

		return nil
	})
}

// internalEmbedRegex matches ![[slug]] and ![[slug|display text]] patterns.
// This is similar to wikilink syntax but with a leading !
var internalEmbedRegex = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// externalEmbedRegex matches ![embed](url) pattern.
// The alt text must be exactly "embed" to trigger embedding.
var externalEmbedRegex = regexp.MustCompile(`!\[embed\]\(([^)]+)\)`)

// embedsCodeBlockRegex matches fenced code blocks to avoid transforming content inside them.
var embedsCodeBlockRegex = regexp.MustCompile("(?s)(```[^`]*```|~~~[^~]*~~~)")

// attachmentEmbedRegex matches Obsidian-style attachment embeds like ![[file.jpg]] or ![[file.jpg|alt text]].
// Only matches files with extensions (images, PDFs, etc.) to avoid conflicting with post embeds.
var attachmentEmbedRegex = regexp.MustCompile(`!\[\[([^\]|]+\.[a-zA-Z0-9]+)(?:\|([^\]]+))?\]\]`)

// htmlTitleRegex matches the <title> tag in HTML.
var htmlTitleRegex = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)

// metaPatternCache caches compiled regexes for meta tag extraction.
var metaPatternCache = struct {
	sync.RWMutex
	m map[string][4]*regexp.Regexp // [propertyFirst, contentFirst, nameFirst, contentFirstName]
}{m: make(map[string][4]*regexp.Regexp)}

// getMetaPatterns returns cached regex patterns for a given property.
func getMetaPatterns(property string) [4]*regexp.Regexp {
	metaPatternCache.RLock()
	patterns, ok := metaPatternCache.m[property]
	metaPatternCache.RUnlock()

	if ok {
		return patterns
	}

	// Compile and cache patterns
	escapedProp := regexp.QuoteMeta(property)
	patterns = [4]*regexp.Regexp{
		regexp.MustCompile(`<meta[^>]*property=["']` + escapedProp + `["'][^>]*content=["']([^"']+)["']`),
		regexp.MustCompile(`<meta[^>]*content=["']([^"']+)["'][^>]*property=["']` + escapedProp + `["']`),
		regexp.MustCompile(`<meta[^>]*name=["']` + escapedProp + `["'][^>]*content=["']([^"']+)["']`),
		regexp.MustCompile(`<meta[^>]*content=["']([^"']+)["'][^>]*name=["']` + escapedProp + `["']`),
	}

	metaPatternCache.Lock()
	metaPatternCache.m[property] = patterns
	metaPatternCache.Unlock()

	return patterns
}

// processAttachmentEmbeds replaces Obsidian-style attachment embeds ![[file.jpg]]
// with standard markdown image syntax ![alt](prefix/file.jpg).
// This runs before internal embeds to avoid conflicts - if a post with the
// same slug exists, internal embed takes precedence.
func (p *EmbedsPlugin) processAttachmentEmbeds(content string) string {
	codeBlocks := embedsCodeBlockRegex.FindAllStringIndex(content, -1)

	if len(codeBlocks) == 0 {
		return p.processAttachmentEmbedsInText(content)
	}

	var result strings.Builder
	lastEnd := 0

	for _, block := range codeBlocks {
		start, end := block[0], block[1]

		if start > lastEnd {
			processed := p.processAttachmentEmbedsInText(content[lastEnd:start])
			result.WriteString(processed)
		}

		result.WriteString(content[start:end])
		lastEnd = end
	}

	if lastEnd < len(content) {
		processed := p.processAttachmentEmbedsInText(content[lastEnd:])
		result.WriteString(processed)
	}

	return result.String()
}

// processAttachmentEmbedsInText processes attachment embeds in a text segment.
func (p *EmbedsPlugin) processAttachmentEmbedsInText(text string) string {
	return attachmentEmbedRegex.ReplaceAllStringFunc(text, func(match string) string {
		groups := attachmentEmbedRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		filename := strings.TrimSpace(groups[1])
		altText := filename
		if len(groups) >= 3 && groups[2] != "" {
			altText = strings.TrimSpace(groups[2])
		}

		if filename == "" {
			return match
		}

		src := p.config.AttachmentsPrefix + filename
		alt := html.EscapeString(altText)
		srcEscaped := html.EscapeString(src)

		return fmt.Sprintf("![%s](%s)", alt, srcEscaped)
	})
}

// processInternalEmbeds replaces ![[slug]] syntax with embed cards.
// Returns the processed content and a list of resolved slugs (dependencies).
func (p *EmbedsPlugin) processInternalEmbeds(content string, idx *lifecycle.PostIndex, currentPost *models.Post) (processed string, dependencies []string) {
	// Split content by fenced code blocks to avoid transforming content inside them
	codeBlocks := embedsCodeBlockRegex.FindAllStringIndex(content, -1)

	if len(codeBlocks) == 0 {
		result := p.processInternalEmbedsInText(content, idx, currentPost, &dependencies)
		return result, dependencies
	}

	var result strings.Builder
	lastEnd := 0

	for _, block := range codeBlocks {
		start, end := block[0], block[1]

		if start > lastEnd {
			processed := p.processInternalEmbedsInText(content[lastEnd:start], idx, currentPost, &dependencies)
			result.WriteString(processed)
		}

		result.WriteString(content[start:end])
		lastEnd = end
	}

	if lastEnd < len(content) {
		processed := p.processInternalEmbedsInText(content[lastEnd:], idx, currentPost, &dependencies)
		result.WriteString(processed)
	}

	return result.String(), dependencies
}

// processInternalEmbedsInText processes internal embeds in a text segment.
// Records successfully resolved slugs in the dependencies slice.
func (p *EmbedsPlugin) processInternalEmbedsInText(text string, idx *lifecycle.PostIndex, currentPost *models.Post, dependencies *[]string) string {
	return internalEmbedRegex.ReplaceAllStringFunc(text, func(match string) string {
		groups := internalEmbedRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		slug := strings.TrimSpace(groups[1])
		displayText := ""
		if len(groups) >= 3 && groups[2] != "" {
			displayText = strings.TrimSpace(groups[2])
		}

		// Look up the target post using the shared index
		targetPost := idx.LookupBySlug(slug)

		if targetPost == nil {
			// Return a warning comment and keep original
			return fmt.Sprintf("<!-- embed not found: %s -->\n%s", slug, match)
		}

		// Don't embed self
		if targetPost.Path == currentPost.Path {
			return fmt.Sprintf("<!-- cannot embed self -->\n%s", match)
		}

		// Record this as a dependency for incremental builds
		*dependencies = append(*dependencies, targetPost.Slug)

		return p.buildInternalEmbedCard(targetPost, displayText)
	})
}

// buildInternalEmbedCard creates HTML for an internal embed card.
func (p *EmbedsPlugin) buildInternalEmbedCard(post *models.Post, displayText string) string {
	var sb strings.Builder

	href := post.Href
	if href == "" {
		href = "/" + post.Slug + "/"
	}

	title := displayText
	if title == "" {
		if post.Title != nil && *post.Title != "" {
			title = *post.Title
		} else {
			title = post.Slug
		}
	}

	description := ""
	if post.Description != nil {
		description = *post.Description
		// Truncate to reasonable length
		if len(description) > 200 {
			description = description[:197] + "..."
		}
	}

	sb.WriteString(`<div class="`)
	sb.WriteString(html.EscapeString(p.config.InternalCardClass))
	sb.WriteString(`">`)
	sb.WriteString("\n")

	sb.WriteString(`  <a href="`)
	sb.WriteString(html.EscapeString(href))
	sb.WriteString(`" class="embed-card-link">`)
	sb.WriteString("\n")

	sb.WriteString(`    <div class="embed-card-content">`)
	sb.WriteString("\n")

	sb.WriteString(`      <div class="embed-card-title">`)
	sb.WriteString(html.EscapeString(title))
	sb.WriteString(`</div>`)
	sb.WriteString("\n")

	if description != "" {
		sb.WriteString(`      <div class="embed-card-description">`)
		sb.WriteString(html.EscapeString(description))
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
	}

	if post.Date != nil {
		sb.WriteString(`      <div class="embed-card-meta">`)
		sb.WriteString(post.Date.Format("Jan 2, 2006"))
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
	}

	sb.WriteString(`    </div>`)
	sb.WriteString("\n")
	sb.WriteString(`  </a>`)
	sb.WriteString("\n")
	sb.WriteString(`</div>`)
	sb.WriteString("\n")

	return sb.String()
}

// processExternalEmbeds replaces ![embed](url) syntax with embed cards.
func (p *EmbedsPlugin) processExternalEmbeds(content string, currentPost *models.Post) string {
	// Split content by fenced code blocks
	codeBlocks := embedsCodeBlockRegex.FindAllStringIndex(content, -1)

	if len(codeBlocks) == 0 {
		return p.processExternalEmbedsInText(content, currentPost)
	}

	var result strings.Builder
	lastEnd := 0

	for _, block := range codeBlocks {
		start, end := block[0], block[1]

		if start > lastEnd {
			processed := p.processExternalEmbedsInText(content[lastEnd:start], currentPost)
			result.WriteString(processed)
		}

		result.WriteString(content[start:end])
		lastEnd = end
	}

	if lastEnd < len(content) {
		processed := p.processExternalEmbedsInText(content[lastEnd:], currentPost)
		result.WriteString(processed)
	}

	return result.String()
}

// processExternalEmbedsInText processes external embeds in a text segment.
func (p *EmbedsPlugin) processExternalEmbedsInText(text string, _ *models.Post) string {
	return externalEmbedRegex.ReplaceAllStringFunc(text, func(match string) string {
		groups := externalEmbedRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		rawURL := strings.TrimSpace(groups[1])

		// Validate URL
		parsedURL, err := url.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return match
		}

		// Fetch OG metadata
		metadata := p.fetchOGMetadata(rawURL)

		return p.buildExternalEmbedCard(rawURL, parsedURL, metadata)
	})
}

// OGMetadata holds Open Graph metadata for external embeds.
type OGMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
	Type        string `json:"type"`
	FetchedAt   int64  `json:"fetched_at"`
}

// fetchOGMetadata fetches Open Graph metadata from a URL.
func (p *EmbedsPlugin) fetchOGMetadata(rawURL string) *OGMetadata {
	if !p.config.FetchExternal {
		return &OGMetadata{Title: p.config.FallbackTitle}
	}

	// Check cache first
	if cached := p.loadCachedMetadata(rawURL); cached != nil {
		return cached
	}

	// Fetch from URL
	metadata := p.fetchMetadataFromURL(rawURL)

	// Cache the result
	p.cacheMetadata(rawURL, metadata)

	return metadata
}

// loadCachedMetadata loads metadata from cache if available and not expired.
func (p *EmbedsPlugin) loadCachedMetadata(rawURL string) *OGMetadata {
	cacheFile := p.getCacheFilePath(rawURL)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var metadata OGMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil
	}

	// Check if cache is expired (7 days)
	if time.Now().Unix()-metadata.FetchedAt > 7*24*60*60 {
		return nil
	}

	return &metadata
}

// cacheMetadata saves metadata to cache.
func (p *EmbedsPlugin) cacheMetadata(rawURL string, metadata *OGMetadata) {
	if p.config.CacheDir == "" {
		return
	}

	metadata.FetchedAt = time.Now().Unix()

	cacheFile := p.getCacheFilePath(rawURL)
	cacheDir := filepath.Dir(cacheFile)

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(cacheFile, data, 0o600) //nolint:errcheck // Best effort cache write
}

// getCacheFilePath returns the cache file path for a URL.
func (p *EmbedsPlugin) getCacheFilePath(rawURL string) string {
	hash := sha256.Sum256([]byte(rawURL))
	hashStr := hex.EncodeToString(hash[:8])
	return filepath.Join(p.config.CacheDir, hashStr+".json")
}

// fetchMetadataFromURL fetches OG metadata from a URL.
func (p *EmbedsPlugin) fetchMetadataFromURL(rawURL string) *OGMetadata {
	metadata := &OGMetadata{
		Title: p.config.FallbackTitle,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", rawURL, http.NoBody)
	if err != nil {
		return metadata
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "markata-go/1.0 (+https://github.com/WaylonWalker/markata-go)")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return metadata
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return metadata
	}

	// Read limited body to avoid memory issues
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return metadata
	}

	htmlContent := string(body)

	// Extract OG metadata using simple regex
	metadata.Title = p.extractMetaContent(htmlContent, "og:title")
	if metadata.Title == "" {
		metadata.Title = p.extractHTMLTitle(htmlContent)
	}
	if metadata.Title == "" {
		metadata.Title = p.config.FallbackTitle
	}

	metadata.Description = p.extractMetaContent(htmlContent, "og:description")
	if metadata.Description == "" {
		metadata.Description = p.extractMetaContent(htmlContent, "description")
	}

	metadata.Image = p.extractMetaContent(htmlContent, "og:image")
	metadata.SiteName = p.extractMetaContent(htmlContent, "og:site_name")
	metadata.Type = p.extractMetaContent(htmlContent, "og:type")

	return metadata
}

// extractMetaContent extracts content from a meta tag.
func (p *EmbedsPlugin) extractMetaContent(htmlContent, property string) string {
	patterns := getMetaPatterns(property)

	// Try property attribute first (og:*)
	if match := patterns[0].FindStringSubmatch(htmlContent); len(match) > 1 {
		return html.UnescapeString(match[1])
	}

	// Try content before property
	if match := patterns[1].FindStringSubmatch(htmlContent); len(match) > 1 {
		return html.UnescapeString(match[1])
	}

	// Try name attribute (description)
	if match := patterns[2].FindStringSubmatch(htmlContent); len(match) > 1 {
		return html.UnescapeString(match[1])
	}

	// Try content before name
	if match := patterns[3].FindStringSubmatch(htmlContent); len(match) > 1 {
		return html.UnescapeString(match[1])
	}

	return ""
}

// extractHTMLTitle extracts the title from HTML.
func (p *EmbedsPlugin) extractHTMLTitle(htmlContent string) string {
	if match := htmlTitleRegex.FindStringSubmatch(htmlContent); len(match) > 1 {
		return html.UnescapeString(strings.TrimSpace(match[1]))
	}
	return ""
}

// buildExternalEmbedCard creates HTML for an external embed card.
func (p *EmbedsPlugin) buildExternalEmbedCard(rawURL string, parsedURL *url.URL, metadata *OGMetadata) string {
	var sb strings.Builder

	domain := parsedURL.Host
	domain = strings.TrimPrefix(domain, "www.")

	sb.WriteString(`<div class="`)
	sb.WriteString(html.EscapeString(p.config.ExternalCardClass))
	sb.WriteString(`">`)
	sb.WriteString("\n")

	sb.WriteString(`  <a href="`)
	sb.WriteString(html.EscapeString(rawURL))
	sb.WriteString(`" class="embed-card-link" target="_blank" rel="noopener noreferrer">`)
	sb.WriteString("\n")

	// Show image if available and enabled
	if p.config.ShowImage && metadata.Image != "" {
		sb.WriteString(`    <div class="embed-card-image">`)
		sb.WriteString("\n")
		sb.WriteString(`      <img src="`)
		sb.WriteString(html.EscapeString(metadata.Image))
		sb.WriteString(`" alt="" loading="lazy">`)
		sb.WriteString("\n")
		sb.WriteString(`    </div>`)
		sb.WriteString("\n")
	}

	sb.WriteString(`    <div class="embed-card-content">`)
	sb.WriteString("\n")

	sb.WriteString(`      <div class="embed-card-title">`)
	sb.WriteString(html.EscapeString(metadata.Title))
	sb.WriteString(`</div>`)
	sb.WriteString("\n")

	if metadata.Description != "" {
		description := metadata.Description
		if len(description) > 200 {
			description = description[:197] + "..."
		}
		sb.WriteString(`      <div class="embed-card-description">`)
		sb.WriteString(html.EscapeString(description))
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
	}

	sb.WriteString(`      <div class="embed-card-meta">`)
	if metadata.SiteName != "" {
		sb.WriteString(html.EscapeString(metadata.SiteName))
		sb.WriteString(` &middot; `)
	}
	sb.WriteString(html.EscapeString(domain))
	sb.WriteString(`</div>`)
	sb.WriteString("\n")

	sb.WriteString(`    </div>`)
	sb.WriteString("\n")
	sb.WriteString(`  </a>`)
	sb.WriteString("\n")
	sb.WriteString(`</div>`)
	sb.WriteString("\n")

	return sb.String()
}

// SetConfig sets the plugin configuration directly.
func (p *EmbedsPlugin) SetConfig(config models.EmbedsConfig) {
	p.config = config
	p.httpClient.Timeout = time.Duration(config.Timeout) * time.Second
}

// Config returns the current plugin configuration.
func (p *EmbedsPlugin) Config() models.EmbedsConfig {
	return p.config
}

// Ensure EmbedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*EmbedsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*EmbedsPlugin)(nil)
	_ lifecycle.TransformPlugin = (*EmbedsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*EmbedsPlugin)(nil)
)
