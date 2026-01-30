// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// HeadingAnchorsPlugin adds anchor links to headings in rendered HTML.
// It processes article_html during the render stage, after render_markdown.
type HeadingAnchorsPlugin struct {
	// enabled controls whether the plugin processes headings
	enabled bool

	// minLevel is the minimum heading level to process (1-6, default: 2)
	minLevel int

	// maxLevel is the maximum heading level to process (1-6, default: 4)
	maxLevel int

	// position is where to insert the anchor link ("start" or "end", default: "end")
	position string

	// symbol is the link text for the anchor (default: "#")
	symbol string

	// class is the CSS class for the anchor link (default: "heading-anchor")
	class string
}

// NewHeadingAnchorsPlugin creates a new HeadingAnchorsPlugin with default settings.
func NewHeadingAnchorsPlugin() *HeadingAnchorsPlugin {
	return &HeadingAnchorsPlugin{
		enabled:  true,
		minLevel: 2,
		maxLevel: 4,
		position: PositionEnd,
		symbol:   "#",
		class:    "heading-anchor",
	}
}

// Name returns the unique name of the plugin.
func (p *HeadingAnchorsPlugin) Name() string {
	return "heading_anchors"
}

// Priority returns the plugin's priority for the render stage.
// Returns a positive priority to run after render_markdown.
func (p *HeadingAnchorsPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after render_markdown
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected in config.Extra["heading_anchors"] as a map.
func (p *HeadingAnchorsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Look for heading_anchors configuration
	haConfig, ok := config.Extra["heading_anchors"]
	if !ok {
		return nil
	}

	// Handle map[string]interface{} configuration
	if cfgMap, ok := haConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.enabled = enabled
		}
		if minLevel, ok := cfgMap["min_level"].(int); ok && minLevel >= 1 && minLevel <= 6 {
			p.minLevel = minLevel
		}
		if maxLevel, ok := cfgMap["max_level"].(int); ok && maxLevel >= 1 && maxLevel <= 6 {
			p.maxLevel = maxLevel
		}
		if position, ok := cfgMap["position"].(string); ok && (position == PositionStart || position == PositionEnd) {
			p.position = position
		}
		if symbol, ok := cfgMap["symbol"].(string); ok {
			p.symbol = symbol
		}
		if class, ok := cfgMap["class"].(string); ok {
			p.class = class
		}
	}

	return nil
}

// headingTagRegex matches opening heading tags with optional attributes.
// Captures: Group 1 = level (1-6), Group 2 = attributes (including id), Group 3 = content, Group 4 = closing tag
var headingTagRegex = regexp.MustCompile(`(?i)<h([1-6])([^>]*)>(.*?)</h[1-6]>`)

// idAttrRegex extracts the id attribute value from tag attributes.
var idAttrRegex = regexp.MustCompile(`(?i)\bid=["']([^"']+)["']`)

// Render processes article_html and adds anchor links to headings.
// Posts with Skip=true or empty ArticleHTML are skipped.
func (p *HeadingAnchorsPlugin) Render(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	posts := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.ArticleHTML == "" {
			return false
		}
		return strings.Contains(post.ArticleHTML, "<h")
	})

	return m.ProcessPostsSliceConcurrently(posts, p.processPost)
}

// processPost adds anchor links to headings in a single post's ArticleHTML.
func (p *HeadingAnchorsPlugin) processPost(post *models.Post) error {
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	// Track IDs to handle duplicates
	idCounts := make(map[string]int)

	post.ArticleHTML = headingTagRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		return p.processHeading(match, idCounts)
	})

	return nil
}

// processHeading processes a single heading match and adds an anchor link.
func (p *HeadingAnchorsPlugin) processHeading(match string, idCounts map[string]int) string {
	submatches := headingTagRegex.FindStringSubmatch(match)
	if len(submatches) < 4 {
		return match
	}

	levelStr := submatches[1]
	attrs := submatches[2]
	content := submatches[3]

	// Parse level
	level := int(levelStr[0] - '0')
	if level < p.minLevel || level > p.maxLevel {
		return match
	}

	// Extract or generate ID
	id := p.extractID(attrs)
	if id == "" {
		id = p.generateID(content, idCounts)
		// Add id attribute if not present
		if attrs == "" {
			attrs = fmt.Sprintf(` id=%q`, id)
		} else {
			attrs = fmt.Sprintf(` id=%q%s`, id, attrs)
		}
	} else {
		// Track existing ID for duplicates
		idCounts[id]++
	}

	// Create anchor link
	anchor := p.createAnchor(id)

	// Insert anchor at configured position
	var newContent string
	if p.position == PositionStart {
		newContent = anchor + content
	} else {
		newContent = content + anchor
	}

	return fmt.Sprintf("<h%s%s>%s</h%s>", levelStr, attrs, newContent, levelStr)
}

// extractID extracts the id attribute value from heading attributes.
func (p *HeadingAnchorsPlugin) extractID(attrs string) string {
	matches := idAttrRegex.FindStringSubmatch(attrs)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// generateID creates a URL-safe ID from heading content.
// Handles duplicate IDs by appending numbers.
func (p *HeadingAnchorsPlugin) generateID(content string, idCounts map[string]int) string {
	// Strip HTML tags from content
	text := stripHTMLTags(content)

	// Use the shared Slugify function for consistent slug generation
	id := models.Slugify(text)

	// Handle empty ID
	if id == "" {
		id = "heading"
	}

	// Handle duplicates
	baseID := id
	count := idCounts[baseID]
	idCounts[baseID] = count + 1

	if count > 0 {
		id = fmt.Sprintf("%s-%d", baseID, count)
	}

	return id
}

// stripHTMLRegex matches HTML tags for removal.
var stripHTMLRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes HTML tags from content.
func stripHTMLTags(content string) string {
	return stripHTMLRegex.ReplaceAllString(content, "")
}

// createAnchor creates an anchor link element for the given ID.
func (p *HeadingAnchorsPlugin) createAnchor(id string) string {
	// Add leading space for "end" position, trailing space for "start"
	if p.position == PositionStart {
		return fmt.Sprintf(`<a href="#%s" class=%q>%s</a> `, id, p.class, p.symbol)
	}
	return fmt.Sprintf(` <a href="#%s" class=%q>%s</a>`, id, p.class, p.symbol)
}

// SetEnabled enables or disables the plugin.
func (p *HeadingAnchorsPlugin) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// SetLevelRange sets the minimum and maximum heading levels to process.
func (p *HeadingAnchorsPlugin) SetLevelRange(minLevel, maxLevel int) {
	if minLevel >= 1 && minLevel <= 6 {
		p.minLevel = minLevel
	}
	if maxLevel >= 1 && maxLevel <= 6 && maxLevel >= minLevel {
		p.maxLevel = maxLevel
	}
}

// SetPosition sets the anchor position ("start" or "end").
func (p *HeadingAnchorsPlugin) SetPosition(position string) {
	if position == PositionStart || position == PositionEnd {
		p.position = position
	}
}

// SetSymbol sets the anchor link symbol.
func (p *HeadingAnchorsPlugin) SetSymbol(symbol string) {
	p.symbol = symbol
}

// SetClass sets the anchor link CSS class.
func (p *HeadingAnchorsPlugin) SetClass(class string) {
	p.class = class
}

// Ensure HeadingAnchorsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*HeadingAnchorsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*HeadingAnchorsPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*HeadingAnchorsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*HeadingAnchorsPlugin)(nil)
)
