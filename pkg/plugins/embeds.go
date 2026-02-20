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

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
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
	oembed     *oembedResolver
}

// NewEmbedsPlugin creates a new EmbedsPlugin with default settings.
func NewEmbedsPlugin() *EmbedsPlugin {
	config := models.NewEmbedsConfig()
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	return &EmbedsPlugin{
		config:     config,
		httpClient: client,
		oembed:     newOEmbedResolver(config, client),
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

	cfgMap, ok := pluginConfig.(map[string]interface{})
	if !ok {
		return nil
	}

	p.applyEmbedsConfig(cfgMap)
	return nil
}

func (p *EmbedsPlugin) applyEmbedsConfig(cfgMap map[string]interface{}) {
	p.config = models.NewEmbedsConfig()
	applyBool(cfgMap, "enabled", &p.config.Enabled)
	applyString(cfgMap, "internal_card_class", &p.config.InternalCardClass)
	applyString(cfgMap, "external_card_class", &p.config.ExternalCardClass)
	applyBool(cfgMap, "fetch_external", &p.config.FetchExternal)
	applyBool(cfgMap, "oembed_enabled", &p.config.OEmbedEnabled)
	applyString(cfgMap, "resolution_strategy", &p.config.ResolutionStrategy)
	applyString(cfgMap, "cache_dir", &p.config.CacheDir)
	applyInt(cfgMap, "timeout", &p.config.Timeout)
	applyInt(cfgMap, "cache_ttl", &p.config.CacheTTL)
	applyString(cfgMap, "fallback_title", &p.config.FallbackTitle)
	applyBool(cfgMap, "show_image", &p.config.ShowImage)
	applyString(cfgMap, "attachments_prefix", &p.config.AttachmentsPrefix)
	applyBool(cfgMap, "oembed_auto_discover", &p.config.OEmbedAutoDiscover)
	applyString(cfgMap, "default_mode", &p.config.DefaultEmbedMode)
	applyString(cfgMap, "oembed_providers_url", &p.config.OEmbedProvidersURL)

	// Parse providers sub-config
	if providersMap, ok := cfgMap["providers"].(map[string]interface{}); ok {
		p.config.OEmbedProviders = make(map[string]models.OEmbedProviderConfig)
		for name, providerCfg := range providersMap {
			if pm, ok := providerCfg.(map[string]interface{}); ok {
				pc := models.OEmbedProviderConfig{Enabled: true}
				if enabled, ok := pm["enabled"].(bool); ok {
					pc.Enabled = enabled
				}
				if mode, ok := pm["mode"].(string); ok {
					pc.Mode = mode
				}
				p.config.OEmbedProviders[name] = pc
			}
		}
	}

	if p.config.Timeout > 0 {
		p.httpClient.Timeout = time.Duration(p.config.Timeout) * time.Second
	}

	p.configureOEmbedProviders(cfgMap)
	if p.oembed == nil {
		p.oembed = newOEmbedResolver(p.config, p.httpClient)
	} else {
		p.oembed.updateConfig(p.config)
	}
	p.validateResolutionStrategy()
}

func applyBool(cfgMap map[string]interface{}, key string, target *bool) {
	if target == nil {
		return
	}
	if value, ok := cfgMap[key].(bool); ok {
		*target = value
	}
}

func applyString(cfgMap map[string]interface{}, key string, target *string) {
	if target == nil {
		return
	}
	if value, ok := cfgMap[key].(string); ok && value != "" {
		*target = value
	}
}

func applyInt(cfgMap map[string]interface{}, key string, target *int) {
	if target == nil {
		return
	}
	if value, ok := cfgMap[key].(int); ok && value > 0 {
		*target = value
	}
}

func (p *EmbedsPlugin) validateResolutionStrategy() {
	strategy := strings.ToLower(p.config.ResolutionStrategy)
	if strategy == "" {
		strategy = strategyOEmbedFirst
	}

	switch strategy {
	case strategyOEmbedFirst, strategyOGFirst, strategyOEmbedOnly:
		p.config.ResolutionStrategy = strategy
	default:
		p.config.ResolutionStrategy = strategyOEmbedFirst
	}
}

func (p *EmbedsPlugin) configureOEmbedProviders(cfgMap map[string]interface{}) {
	providersRaw, ok := cfgMap["providers"].(map[string]interface{})
	if !ok {
		return
	}

	if p.config.OEmbedProviders == nil {
		p.config.OEmbedProviders = make(map[string]models.OEmbedProviderConfig)
	}

	for name, raw := range providersRaw {
		key := strings.ToLower(name)
		switch value := raw.(type) {
		case bool:
			p.config.OEmbedProviders[key] = models.OEmbedProviderConfig{Enabled: value}
		case map[string]interface{}:
			if enabled, ok := value["enabled"].(bool); ok {
				p.config.OEmbedProviders[key] = models.OEmbedProviderConfig{Enabled: enabled}
			}
		}
	}
}

// Transform processes embed syntax in all post content.
func (p *EmbedsPlugin) Transform(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	// Use the shared PostIndex from the lifecycle manager
	idx := m.PostIndex()
	cache := GetBuildCache(m)

	posts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && post.Content != ""
	})

	// Phase 1: Restore cached results for unchanged posts
	var needProcessing []*models.Post
	if cache != nil {
		for _, post := range posts {
			contentHash := buildcache.ContentHash(post.Content)
			if cached, ok := cache.GetCachedEmbedsContent(post.Path, contentHash); ok {
				post.Content = cached
			} else {
				needProcessing = append(needProcessing, post)
			}
		}
	} else {
		needProcessing = posts
	}

	if len(needProcessing) == 0 {
		return nil
	}

	// Phase 2: Process posts that need updating, concurrently
	return m.ProcessPostsSliceConcurrently(needProcessing, func(post *models.Post) error {
		contentHash := buildcache.ContentHash(post.Content)
		content := p.processAttachmentEmbeds(post.Content)
		content, dependencies := p.processInternalEmbeds(content, idx, post)
		content = p.processExternalEmbeds(content, post)
		post.Content = content

		// Record dependencies for incremental build cache
		for _, dep := range dependencies {
			post.AddDependency(dep)
		}

		if cache != nil {
			cache.CacheEmbedsContent(post.Path, contentHash, content)
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

// embedBracketRegex matches [!embed](url) Obsidian-style embed syntax.
// This is an alternative to ![embed](url) syntax.
// Supports optional options: [!embed](url|class1 class2)
var embedBracketRegex = regexp.MustCompile(`\[!embed\]\(([^)|]+)(?:\|([^)]+))?\)`)

// externalEmbedWithOptionsRegex matches ![embed](url|options) pattern with options.
// Options include: no_title, no_description, no_meta, image_only, center,
// full_width, video, link, rich, hover, card, performance
var externalEmbedWithOptionsRegex = regexp.MustCompile(`!\[embed\]\(([^)|]+)\|([^)]+)\)`)

// externalObsidianEmbedRegex matches Obsidian-style external embeds like ![[https://example.com]]
// with optional display text: ![[https://example.com|Title]].
// Supports optional classes after a second pipe: ![[https://example.com|Title|class1 class2]]
var externalObsidianEmbedRegex = regexp.MustCompile(`!\[\[(https?://[^\]|]+)(?:\|([^\]|]+))?(?:\|([^\]]+))?\]\]`)

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

const (
	strategyOEmbedFirst = "oembed_first"
	strategyOGFirst     = "og_first"
	strategyOEmbedOnly  = "oembed_only"

	schemeHTTP  = "http"
	schemeHTTPS = "https"

	embedModeRich        = "rich"
	embedModeCard        = "card"
	embedModePerformance = "performance"
	embedModeHover       = "hover"
	embedModeImageOnly   = "image_only"
)

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
		if isExternalEmbedURL(slug) {
			return match
		}
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
//
//nolint:gocyclo // multiple regex replacements required, complexity is unavoidable
func (p *EmbedsPlugin) processExternalEmbedsInText(text string, _ *models.Post) string {
	// Process [!embed](url|options) syntax first
	processed := embedBracketRegex.ReplaceAllStringFunc(text, func(match string) string {
		groups := embedBracketRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		rawURL := strings.TrimSpace(groups[1])
		options := parseEmbedOptions(groups[2])

		parsedURL, err := url.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS) {
			return match
		}

		metadata := p.fetchExternalMetadata(rawURL)
		return p.buildExternalEmbedCard(rawURL, parsedURL, metadata, options)
	})

	// Process ![embed](url|options) with options
	processed = externalEmbedWithOptionsRegex.ReplaceAllStringFunc(processed, func(match string) string {
		groups := externalEmbedWithOptionsRegex.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}

		rawURL := strings.TrimSpace(groups[1])
		options := parseEmbedOptions(groups[2])

		parsedURL, err := url.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS) {
			return match
		}

		metadata := p.fetchExternalMetadata(rawURL)
		return p.buildExternalEmbedCard(rawURL, parsedURL, metadata, options)
	})

	// Process ![embed](url) basic syntax
	processed = externalEmbedRegex.ReplaceAllStringFunc(processed, func(match string) string {
		groups := externalEmbedRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		rawURL := strings.TrimSpace(groups[1])

		// Validate URL
		parsedURL, err := url.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS) {
			return match
		}

		metadata := p.fetchExternalMetadata(rawURL)
		return p.buildExternalEmbedCard(rawURL, parsedURL, metadata, EmbedOptions{})
	})

	return externalObsidianEmbedRegex.ReplaceAllStringFunc(processed, func(match string) string {
		groups := externalObsidianEmbedRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		rawURL := strings.TrimSpace(groups[1])
		override := ""
		var options EmbedOptions
		if len(groups) >= 3 && groups[2] != "" {
			override = strings.TrimSpace(groups[2])
		}
		// Check for classes (4th group)
		if len(groups) >= 4 && groups[3] != "" {
			options = parseEmbedOptions(groups[3])
		}

		parsedURL, err := url.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != schemeHTTP && parsedURL.Scheme != schemeHTTPS) {
			return match
		}

		metadata := p.fetchExternalMetadata(rawURL)
		metadata = p.applyExternalTitleOverride(metadata, override)

		return p.buildExternalEmbedCard(rawURL, parsedURL, metadata, options)
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
	Source      string `json:"source"`

	// Provider info for mode selection
	ProviderName string `json:"provider_name"`
	HTML         string `json:"html"` // oEmbed HTML for rich embeds
}

// EmbedOptions holds parsing options for embed syntax.
// Supports: no_title, no_description, no_meta, image_only, center,
// full_width, video, link, rich, hover, card, performance
type EmbedOptions struct {
	NoTitle       bool
	NoDescription bool
	NoMeta        bool
	ImageOnly     bool
	Center        bool
	FullWidth     bool
	Video         bool
	Link          bool
	Rich          bool
	Hover         bool
	Card          bool
	Performance   bool
	TitleOverride string
	Classes       []string
}

// parseEmbedOptions parses space-separated classes from embed syntax.
func parseEmbedOptions(optionsStr string) EmbedOptions {
	opts := EmbedOptions{}
	if optionsStr == "" {
		return opts
	}

	parts := strings.Fields(optionsStr)
	opts.Classes = parts

	for _, part := range parts {
		switch strings.ToLower(part) {
		case "no_title":
			opts.NoTitle = true
		case "no_description":
			opts.NoDescription = true
		case "no_meta":
			opts.NoMeta = true
		case embedModeImageOnly:
			opts.ImageOnly = true
		case "center":
			opts.Center = true
		case "full_width":
			opts.FullWidth = true
		case "video":
			opts.Video = true
		case "link":
			opts.Link = true
		case embedModeRich:
			opts.Rich = true
		case embedModeHover:
			opts.Hover = true
		case embedModeCard:
			opts.Card = true
		case embedModePerformance:
			opts.Performance = true
		}
	}

	return opts
}

// fetchOGMetadata fetches Open Graph metadata from a URL.
func (p *EmbedsPlugin) fetchOGMetadata(rawURL string) *OGMetadata {
	if !p.config.FetchExternal {
		return &OGMetadata{Title: p.config.FallbackTitle}
	}

	metadata := p.fetchCachedMetadata(rawURL, "og")
	if metadata != nil {
		return metadata
	}

	// Fetch from URL
	metadata = p.fetchMetadataFromURL(rawURL)

	// Cache the result
	p.cacheMetadata(rawURL, "og", metadata)

	return metadata
}

// fetchExternalMetadata resolves external metadata using the configured strategy.
func (p *EmbedsPlugin) fetchExternalMetadata(rawURL string) *OGMetadata {
	strategy := strings.ToLower(p.config.ResolutionStrategy)
	if strategy == "" {
		strategy = strategyOEmbedFirst
	}

	if strategy == strategyOEmbedFirst || strategy == strategyOEmbedOnly {
		if !p.config.OEmbedEnabled {
			strategy = strategyOGFirst
		}
	}

	tryOEmbed := func() (*OGMetadata, bool) {
		return p.resolveOEmbedMetadata(rawURL)
	}

	tryCached := func() *OGMetadata {
		if cached := p.fetchCachedMetadata(rawURL, "oembed"); cached != nil {
			return cached
		}
		return p.fetchCachedMetadata(rawURL, "og")
	}

	tryOG := func() *OGMetadata {
		return p.fetchOGMetadata(rawURL)
	}

	switch strategy {
	case strategyOEmbedOnly:
		if metadata, _ := tryOEmbed(); metadata != nil {
			return metadata
		}
		if cached := tryCached(); cached != nil {
			return cached
		}
		return &OGMetadata{Title: p.config.FallbackTitle}
	case strategyOGFirst:
		metadata := tryOG()
		if metadata != nil && metadata.Title != p.config.FallbackTitle {
			return metadata
		}
		if oembed, _ := tryOEmbed(); oembed != nil {
			return oembed
		}
		if cached := tryCached(); cached != nil {
			return cached
		}
		return metadata
	default:
		if oembed, matched := tryOEmbed(); oembed != nil {
			return oembed
		} else if matched {
			// Provider matched but failed; fall back if allowed
			if p.config.FetchExternal {
				return tryOG()
			}
			return &OGMetadata{Title: p.config.FallbackTitle}
		}

		metadata := tryOG()
		if metadata != nil && metadata.Title != p.config.FallbackTitle {
			return metadata
		}
		if cached := tryCached(); cached != nil {
			return cached
		}
		return metadata
	}
}

func (p *EmbedsPlugin) resolveOEmbedMetadata(rawURL string) (*OGMetadata, bool) {
	if !p.config.OEmbedEnabled || p.oembed == nil {
		return nil, false
	}

	if cached := p.fetchCachedMetadata(rawURL, "oembed"); cached != nil {
		return cached, true
	}

	response, matched, err := p.oembed.Resolve(rawURL)
	if err != nil || !matched || response == nil {
		return nil, matched
	}

	metadata := &OGMetadata{
		Title:        response.Title,
		Image:        response.ThumbnailURL,
		SiteName:     response.ProviderName,
		ProviderName: response.ProviderName,
		Type:         response.Type,
		HTML:         response.HTML,
		FetchedAt:    time.Now().Unix(),
		Source:       "oembed",
	}

	if metadata.Image == "" {
		metadata.Image = response.URL
	}

	if metadata.Title == "" {
		metadata.Title = p.config.FallbackTitle
	}

	p.cacheMetadata(rawURL, "oembed", metadata)

	return metadata, true
}

// loadCachedMetadata loads metadata from cache if available and not expired.
func (p *EmbedsPlugin) fetchCachedMetadata(rawURL, suffix string) *OGMetadata {
	cacheFile := p.getCacheFilePath(rawURL, suffix)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if suffix == "og" {
			legacyFile := p.getLegacyCacheFilePath(rawURL)
			if legacyData, legacyErr := os.ReadFile(legacyFile); legacyErr == nil {
				data = legacyData
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	var metadata OGMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil
	}

	cacheTTL := p.config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 7 * 24 * 60 * 60
	}

	// Check if cache is expired
	if time.Now().Unix()-metadata.FetchedAt > int64(cacheTTL) {
		return nil
	}

	return &metadata
}

// cacheMetadata saves metadata to cache.
func (p *EmbedsPlugin) cacheMetadata(rawURL, suffix string, metadata *OGMetadata) {
	if p.config.CacheDir == "" {
		return
	}

	metadata.FetchedAt = time.Now().Unix()

	cacheFile := p.getCacheFilePath(rawURL, suffix)
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
func (p *EmbedsPlugin) getCacheFilePath(rawURL, suffix string) string {
	hash := sha256.Sum256([]byte(rawURL))
	hashStr := hex.EncodeToString(hash[:8])

	if suffix == "" {
		suffix = "og"
	}

	return filepath.Join(p.config.CacheDir, hashStr+"-"+suffix+".json")
}

func (p *EmbedsPlugin) getLegacyCacheFilePath(rawURL string) string {
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
	metadata.Source = "og"
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

// defaultModeByType returns the default mode for an oEmbed type.
func defaultModeByType(oembedType string) string {
	switch strings.ToLower(oembedType) {
	case "photo":
		return embedModeImageOnly
	case "video":
		return embedModeRich
	case "rich":
		return embedModeRich
	default:
		return embedModeCard
	}
}

// effectiveEmbedMode determines the effective embed mode considering:
// 1. Explicit options from embed syntax
// 2. Provider-specific config
// 3. Default mode based on oEmbed type
// 4. Global default mode
func (p *EmbedsPlugin) effectiveEmbedMode(opts EmbedOptions, metadata *OGMetadata) EmbedOptions {
	// If explicit mode is set via options, use that
	if opts.Rich || opts.Performance || opts.Hover || opts.Card || opts.ImageOnly {
		return opts
	}

	// Check provider-specific config
	providerName := strings.ToLower(metadata.ProviderName)
	if providerName != "" {
		if providerCfg, ok := p.config.OEmbedProviders[providerName]; ok && providerCfg.Mode != "" {
			mode := strings.ToLower(providerCfg.Mode)
			switch mode {
			case embedModeRich:
				opts.Rich = true
			case embedModeCard:
				opts.Card = true
			case embedModePerformance:
				opts.Performance = true
			case embedModeHover:
				opts.Hover = true
			case embedModeImageOnly:
				opts.ImageOnly = true
			}
			return opts
		}
	}

	// Fall back to default mode based on oEmbed type or global default
	oembedType := strings.ToLower(metadata.Type)
	defaultMode := p.config.DefaultEmbedMode
	if defaultMode == "" {
		defaultMode = defaultModeByType(oembedType)
	}

	switch defaultMode {
	case embedModeRich:
		opts.Rich = true
	case embedModePerformance:
		opts.Performance = true
	case embedModeHover:
		opts.Hover = true
	case embedModeImageOnly:
		opts.ImageOnly = true
	}

	return opts
}

// buildExternalEmbedCard creates HTML for an external embed card.
// It respects the EmbedOptions to control what elements are displayed.
//
//nolint:gocyclo // multiple rendering modes require conditional branches
func (p *EmbedsPlugin) buildExternalEmbedCard(rawURL string, parsedURL *url.URL, metadata *OGMetadata, opts EmbedOptions) string {
	// Determine effective mode based on config and oEmbed type
	opts = p.effectiveEmbedMode(opts, metadata)

	var sb strings.Builder

	domain := parsedURL.Host
	domain = strings.TrimPrefix(domain, "www.")

	// Build class list
	classes := []string{p.config.ExternalCardClass}
	if opts.Center {
		classes = append(classes, "embed-card-center")
	}
	if opts.FullWidth {
		classes = append(classes, "embed-card-full-width")
	}
	if opts.ImageOnly {
		classes = append(classes, "embed-card-image-only")
	}
	if opts.Hover {
		classes = append(classes, "embed-card-hover")
	}
	if metadata.ProviderName != "" {
		classes = append(classes, "embed-card-provider-"+strings.ToLower(metadata.ProviderName))
	}
	classes = append(classes, opts.Classes...)

	sb.WriteString(`<div class="`)
	sb.WriteString(html.EscapeString(strings.Join(classes, " ")))
	sb.WriteString(`">`)
	sb.WriteString("\n")

	// Handle rich embed (iframe) mode
	if opts.Rich && metadata.HTML != "" {
		sb.WriteString(`  <div class="embed-card-rich">`)
		sb.WriteString("\n")
		sb.WriteString(metadata.HTML)
		sb.WriteString("\n")
		sb.WriteString(`  </div>`)
		sb.WriteString("\n")
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
		return sb.String()
	}

	// Handle hover mode - shows image, swaps to embed on hover
	if opts.Hover && metadata.HTML != "" {
		sb.WriteString(`  <a href="`)
		sb.WriteString(html.EscapeString(rawURL))
		sb.WriteString(`" class="embed-card-link" target="_blank" rel="noopener noreferrer">`)
		sb.WriteString("\n")
		if p.config.ShowImage && metadata.Image != "" {
			sb.WriteString(`    <div class="embed-card-image embed-card-lazy">`)
			sb.WriteString("\n")
			sb.WriteString(`      <img src="`)
			sb.WriteString(html.EscapeString(metadata.Image))
			sb.WriteString(`" alt="" loading="lazy">`)
			sb.WriteString("\n")
			sb.WriteString(`      <div class="embed-card-hover-overlay">`)
			sb.WriteString("\n")
			sb.WriteString(`        <span class="embed-card-hover-text">Click to load embed</span>`)
			sb.WriteString("\n")
			sb.WriteString(`      </div>`)
			sb.WriteString("\n")
			sb.WriteString(`    </div>`)
			sb.WriteString("\n")
		}
		if !opts.NoTitle && metadata.Title != "" {
			sb.WriteString(`    <div class="embed-card-title">`)
			sb.WriteString(html.EscapeString(metadata.Title))
			sb.WriteString(`</div>`)
			sb.WriteString("\n")
		}
		sb.WriteString(`  </a>`)
		sb.WriteString("\n")
		sb.WriteString(`  <div class="embed-card-hover-embed" data-embed-html="`)
		sb.WriteString(html.EscapeString(metadata.HTML))
		sb.WriteString(`">`)
		sb.WriteString("\n")
		sb.WriteString(`  </div>`)
		sb.WriteString("\n")
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
		return sb.String()
	}

	// Handle link-only mode
	if opts.Link {
		linkTitle := metadata.Title
		if linkTitle == "" {
			linkTitle = p.config.FallbackTitle
		}
		if linkTitle == "" {
			linkTitle = domain
		}

		sb.WriteString(`  <a href="`)
		sb.WriteString(html.EscapeString(rawURL))
		sb.WriteString(`" class="embed-card-link-only" target="_blank" rel="noopener noreferrer">`)
		sb.WriteString(html.EscapeString(linkTitle))
		sb.WriteString(`</a>`)
		sb.WriteString("\n")
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
		return sb.String()
	}

	// Handle performance/image_only mode - just show image, no text
	if opts.Performance || opts.ImageOnly {
		if metadata.Image != "" {
			sb.WriteString(`  <a href="`)
			sb.WriteString(html.EscapeString(rawURL))
			sb.WriteString(`" target="_blank" rel="noopener noreferrer">`)
			sb.WriteString("\n")
			sb.WriteString(`    <img src="`)
			sb.WriteString(html.EscapeString(metadata.Image))
			sb.WriteString(`" alt="`)
			if metadata.Title != "" {
				sb.WriteString(html.EscapeString(metadata.Title))
			} else {
				sb.WriteString(html.EscapeString(domain))
			}
			sb.WriteString(`" loading="lazy" class="embed-card-img">`)
			sb.WriteString("\n")
			sb.WriteString(`  </a>`)
			sb.WriteString("\n")
		}
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
		return sb.String()
	}

	// Standard card mode (default)
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

	// Show title unless disabled
	if !opts.NoTitle && metadata.Title != "" {
		sb.WriteString(`      <div class="embed-card-title">`)
		sb.WriteString(html.EscapeString(metadata.Title))
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
	}

	// Show description unless disabled
	if !opts.NoDescription && metadata.Description != "" {
		description := metadata.Description
		if len(description) > 200 {
			description = description[:197] + "..."
		}
		sb.WriteString(`      <div class="embed-card-description">`)
		sb.WriteString(html.EscapeString(description))
		sb.WriteString(`</div>`)
		sb.WriteString("\n")
	}

	// Show meta (site name + domain) unless disabled
	if !opts.NoMeta {
		sb.WriteString(`      <div class="embed-card-meta">`)
		if metadata.SiteName != "" {
			sb.WriteString(html.EscapeString(metadata.SiteName))
			sb.WriteString(` &middot; `)
		}
		sb.WriteString(html.EscapeString(domain))
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

func (p *EmbedsPlugin) applyExternalTitleOverride(metadata *OGMetadata, override string) *OGMetadata {
	if override == "" || metadata == nil {
		return metadata
	}

	cloned := *metadata
	cloned.Title = override

	return &cloned
}

func isExternalEmbedURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}

	return parsed.Scheme == schemeHTTP || parsed.Scheme == schemeHTTPS
}

// SetConfig sets the plugin configuration directly.
func (p *EmbedsPlugin) SetConfig(config models.EmbedsConfig) {
	p.config = config
	p.httpClient.Timeout = time.Duration(config.Timeout) * time.Second
	if p.oembed == nil {
		p.oembed = newOEmbedResolver(config, p.httpClient)
	} else {
		p.oembed.updateConfig(config)
	}
	p.validateResolutionStrategy()
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
