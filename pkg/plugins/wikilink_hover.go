// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// WikilinkHoverPlugin adds hover preview data attributes to wikilinks.
// It runs at the render stage (after wikilinks have been converted to HTML).
// This plugin finds <a class="wikilink"> tags and adds:
// - data-preview: truncated description/content for hover tooltips
// - data-preview-image: featured image URL if available
// - data-preview-screenshot: screenshot URL if service is configured
type WikilinkHoverPlugin struct {
	config  models.WikilinkHoverConfig
	postIdx *lifecycle.PostIndex
	// siteURL is the configured site origin (scheme://host, no trailing
	// slash) so absolute self-links can be previewed like relative ones.
	siteURL string
}

// NewWikilinkHoverPlugin creates a new WikilinkHoverPlugin with default settings.
func NewWikilinkHoverPlugin() *WikilinkHoverPlugin {
	return &WikilinkHoverPlugin{
		config: models.NewWikilinkHoverConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *WikilinkHoverPlugin) Name() string {
	return "wikilink_hover"
}

// Priority returns the plugin's priority for a given stage.
// In the render stage it runs after render_markdown (PriorityDefault) has
// produced ArticleHTML but before templates (PriorityLate) bakes that HTML
// into pages, so the added data attributes reach the final output.
func (p *WikilinkHoverPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate - 10
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "wikilink_hover" key.
func (p *WikilinkHoverPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for wikilink_hover config in Extra
	pluginConfig, ok := config.Extra["wikilink_hover"]
	if !ok {
		return nil
	}

	// Handle map configuration
	if cfgMap, ok := pluginConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if previewLength, ok := parseIntFromInterface(cfgMap["preview_length"]); ok && previewLength > 0 {
			p.config.PreviewLength = previewLength
		}
		if includeImage, ok := cfgMap["include_image"].(bool); ok {
			p.config.IncludeImage = includeImage
		}
		if screenshotService, ok := cfgMap["screenshot_service"].(string); ok {
			p.config.ScreenshotService = screenshotService
		}
		if allInternal, ok := cfgMap["all_internal_links"].(bool); ok {
			p.config.AllInternalLinks = &allInternal
		}
	}

	return nil
}

// Render processes wikilinks in all post HTML to add hover data attributes.
func (p *WikilinkHoverPlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	// Use the shared PostIndex from the lifecycle manager
	p.postIdx = m.PostIndex()
	p.siteURL = siteOrigin(getSiteURL(m.Config()))

	allInternal := p.config.PreviewsAllInternalLinks()
	posts := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.ArticleHTML == "" {
			return false
		}
		if strings.Contains(post.ArticleHTML, `class="wikilink"`) {
			return true
		}
		return allInternal && p.hasInternalHref(post.ArticleHTML)
	})

	return m.ProcessPostsSliceConcurrently(posts, p.processPost)
}

// internalAnchorRegex matches opening anchor tags whose href is site-relative
// or absolute; absolute hrefs are filtered against the site origin later.
var internalAnchorRegex = regexp.MustCompile(`<a\s+[^>]*href="(?:/|https?://)[^"]*"[^>]*>`)

// siteOrigin reduces a configured site URL to scheme://host with no path or
// trailing slash. It returns "" when the URL is empty or unparseable.
func siteOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// hasInternalHref reports whether the HTML contains a link that could resolve
// to a post on this site, used as a cheap pre-filter before regex work.
func (p *WikilinkHoverPlugin) hasInternalHref(htmlContent string) bool {
	if strings.Contains(htmlContent, `href="/`) {
		return true
	}
	return p.siteURL != "" && strings.Contains(htmlContent, `href="`+p.siteURL)
}

// internalPath converts an href to a site-relative path when it points at
// this site, or returns "" when it is external or not a post-like path.
func (p *WikilinkHoverPlugin) internalPath(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		if p.siteURL == "" || !strings.HasPrefix(href, p.siteURL) {
			return ""
		}
		href = href[len(p.siteURL):]
		if href == "" {
			return ""
		}
	}
	if !strings.HasPrefix(href, "/") {
		return ""
	}
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	if href == "/" || strings.Contains(href, ".") {
		return ""
	}
	return href
}

// anchorClassRegex extracts the class attribute of an anchor tag.
var anchorClassRegex = regexp.MustCompile(`class="([^"]*)"`)

// internalLinkSkipClasses lists anchor classes that must not receive preview
// data: wikilinks are handled separately, and chrome links are not content.
var internalLinkSkipClasses = []string{
	"wikilink", "mention", "heading-anchor", "footnote-ref", "footnote-backref",
	"no-preview", "tag", "u-url", "glightbox", "card", "post-nav",
}

// enhanceInternalLinks adds data-title/description/date to plain internal
// links that resolve to a post, so the shared tooltip script can preview them.
func (p *WikilinkHoverPlugin) enhanceInternalLinks(htmlContent string, source *models.Post) string {
	return internalAnchorRegex.ReplaceAllStringFunc(htmlContent, func(tag string) string {
		if strings.Contains(tag, "data-title=") || strings.Contains(tag, "data-preview") {
			return tag
		}
		if cm := anchorClassRegex.FindStringSubmatch(tag); cm != nil {
			for _, cls := range strings.Fields(cm[1]) {
				for _, skip := range internalLinkSkipClasses {
					if cls == skip {
						return tag
					}
				}
			}
		}
		hrefMatches := wikilinkHrefRegex.FindStringSubmatch(tag)
		if len(hrefMatches) < 2 {
			return tag
		}
		href := p.internalPath(hrefMatches[1])
		if href == "" {
			return tag
		}
		target := p.lookupPost(href)
		if target == nil || target.Private {
			if target != nil {
				recordPreviewDependency(source, target)
			}
			recordPreviewPathDependency(source, href)
			return tag
		}
		recordPreviewDependency(source, target)
		recordPreviewPathDependency(source, href)

		var attrs []string
		if target.Title != nil && *target.Title != "" {
			attrs = append(attrs, `data-title="`+html.EscapeString(target.PlainTitle())+`"`)
		}
		if target.Description != nil && *target.Description != "" {
			attrs = append(attrs, `data-description="`+html.EscapeString(truncatePreviewText(*target.Description, p.config.PreviewLength))+`"`)
		}
		if target.Date != nil {
			attrs = append(attrs, `data-date="`+target.Date.Format("2006-01-02")+`"`)
		}
		if len(attrs) == 0 {
			return tag
		}
		attrs = append(attrs, `data-preview="internal"`)
		return tag[:len(tag)-1] + " " + strings.Join(attrs, " ") + ">"
	})
}

// lookupPost resolves an href to a post, tolerating trailing-slash differences.
func (p *WikilinkHoverPlugin) lookupPost(href string) *models.Post {
	if p.postIdx == nil {
		return nil
	}
	if target := p.postIdx.ByHref[href]; target != nil {
		return target
	}
	if target := p.postIdx.ByHref[strings.TrimSuffix(href, "/")]; target != nil {
		return target
	}
	return p.postIdx.ByHref[href+"/"]
}

// wikilinkAnchorRegex matches wikilink anchor tags created by the wikilinks plugin.
// Captures: full tag attributes, href value
var wikilinkAnchorRegex = regexp.MustCompile(
	`<a\s+([^>]*class="[^"]*wikilink[^"]*"[^>]*)>([^<]*)</a>`,
)

// wikilinkHrefRegex extracts href attribute value from tag attributes.
var wikilinkHrefRegex = regexp.MustCompile(`href="([^"]*)"`)

// processPost processes a single post's HTML to add hover data to wikilinks.
func (p *WikilinkHoverPlugin) processPost(post *models.Post) error {
	// Skip posts marked as skip or with no HTML content
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	result := post.ArticleHTML
	if strings.Contains(result, `class="wikilink"`) {
		// Replace wikilink anchors with enhanced versions
		result = wikilinkAnchorRegex.ReplaceAllStringFunc(result, func(match string) string {
			return p.enhanceWikilink(match, post)
		})
	}
	if p.config.PreviewsAllInternalLinks() && p.hasInternalHref(result) {
		result = p.enhanceInternalLinks(result, post)
	}

	post.ArticleHTML = result
	return nil
}

// enhanceWikilink adds data attributes to a wikilink anchor tag.
func (p *WikilinkHoverPlugin) enhanceWikilink(match string, source *models.Post) string {
	// Extract href from the match
	hrefMatches := wikilinkHrefRegex.FindStringSubmatch(match)
	if len(hrefMatches) < 2 {
		return match
	}

	href := hrefMatches[1]

	// Look up the target post using the shared PostIndex
	targetPost := p.postIdx.ByHref[href]
	if targetPost == nil {
		// Try without trailing slash
		targetPost = p.postIdx.ByHref[strings.TrimSuffix(href, "/")]
	}
	if targetPost == nil {
		// Try with trailing slash
		targetPost = p.postIdx.ByHref[href+"/"]
	}
	if targetPost == nil {
		return match
	}
	recordPreviewDependency(source, targetPost)

	// Skip hover data for private posts to prevent metadata leaks
	if targetPost.Private {
		return match
	}

	// Build data attributes
	attrs := p.buildDataAttributes(targetPost)
	if attrs == "" {
		return match
	}

	// Insert data attributes before the closing >
	// Find the position of the first > in the <a ...> tag
	tagEnd := strings.Index(match, ">")
	if tagEnd == -1 {
		return match
	}

	// Insert attributes before the >
	return match[:tagEnd] + " " + attrs + match[tagEnd:]
}

// buildDataAttributes creates the data attribute string for hover previews.
func (p *WikilinkHoverPlugin) buildDataAttributes(post *models.Post) string {
	var attrs []string

	// Get preview text from description or content
	previewText := p.getPreviewText(post)
	if previewText != "" {
		attrs = append(attrs, `data-preview="`+html.EscapeString(previewText)+`"`)
	}

	// Add preview image if configured and available
	if p.config.IncludeImage {
		imageURL := p.getPostImage(post)
		if imageURL != "" {
			attrs = append(attrs, `data-preview-image="`+html.EscapeString(imageURL)+`"`)
		}
	}

	// Add screenshot URL if service is configured
	if p.config.ScreenshotService != "" && post.Href != "" {
		screenshotURL := p.config.ScreenshotService + post.Href
		attrs = append(attrs, `data-preview-screenshot="`+html.EscapeString(screenshotURL)+`"`)
	}

	return strings.Join(attrs, " ")
}

// recordPreviewDependency records a resolved preview target even when the
// target currently produces no preview attributes (for example, while it is
// private). A later publication or metadata change can then invalidate the
// linking post's cached page.
func recordPreviewDependency(source, target *models.Post) {
	if source == nil || target == nil || target.Slug == "" {
		return
	}
	source.AddDependency(target.Slug)
}

const internalHrefDependencyPrefix = "internal-href:"

// internalHrefDependencyKey returns the stable dependency token for an
// internal URL. It deliberately ignores a trailing slash so links written as
// /target and /target/ invalidate from the same target post.
func internalHrefDependencyKey(href string) string {
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	if !strings.HasPrefix(href, "/") || href == "/" {
		return ""
	}
	href = strings.TrimSuffix(href, "/")
	if href == "" || strings.Contains(href, ".") {
		return ""
	}
	return internalHrefDependencyPrefix + href
}

// recordPreviewPathDependency keeps unresolved internal links connected to the
// target path. When a new post later claims that path, the source page must be
// rebuilt so the preview attributes can be added.
func recordPreviewPathDependency(source *models.Post, href string) {
	if source == nil {
		return
	}
	if key := internalHrefDependencyKey(href); key != "" {
		source.AddDependency(key)
	}
}

// getPreviewText extracts preview text from post description or content.
func (p *WikilinkHoverPlugin) getPreviewText(post *models.Post) string {
	// First try description
	if post.Description != nil && *post.Description != "" {
		return truncatePreviewText(*post.Description, p.config.PreviewLength)
	}

	// Fall back to content (strip HTML if any)
	if post.Content != "" {
		// Use ArticleHTML if available (already rendered), otherwise raw content
		text := post.Content
		if post.ArticleHTML != "" {
			text = stripHTML(post.ArticleHTML)
		}
		return truncatePreviewText(text, p.config.PreviewLength)
	}

	return ""
}

// getPostImage finds a featured image URL for the post.
func (p *WikilinkHoverPlugin) getPostImage(post *models.Post) string {
	if post.Extra == nil {
		return ""
	}

	// Check common image field names
	imageFields := []string{"image", "featured_image", "cover_image", "og_image", "thumbnail"}
	for _, field := range imageFields {
		if img, ok := post.Extra[field].(string); ok && img != "" {
			return img
		}
	}

	return ""
}

// truncatePreviewText truncates text to maxLen characters, adding ellipsis if needed.
// It tries to break at word boundaries.
func truncatePreviewText(text string, maxLen int) string {
	// Clean up whitespace
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	// Collapse multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}

	// Truncate to maxLen runes
	runes := []rune(text)
	truncated := string(runes[:maxLen])

	// Try to break at last space
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(truncated) + "..."
}

// stripHTML removes HTML tags from text.
func stripHTML(s string) string {
	// Simple regex-based HTML stripping
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	result := tagRegex.ReplaceAllString(s, "")

	// Decode common HTML entities
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", `"`)
	result = strings.ReplaceAll(result, "&#39;", "'")

	return result
}

// SetConfig sets the plugin configuration directly.
// This is useful for testing or programmatic configuration.
func (p *WikilinkHoverPlugin) SetConfig(config models.WikilinkHoverConfig) {
	p.config = config
}

// Config returns the current plugin configuration.
func (p *WikilinkHoverPlugin) Config() models.WikilinkHoverConfig {
	return p.config
}

// Ensure WikilinkHoverPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*WikilinkHoverPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*WikilinkHoverPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*WikilinkHoverPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*WikilinkHoverPlugin)(nil)
)
