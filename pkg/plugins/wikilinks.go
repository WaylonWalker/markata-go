// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// WikilinksPlugin transforms [[slug]] and [[slug|text]] wikilink syntax
// into HTML anchor tags during the transform stage.
type WikilinksPlugin struct {
	// warnOnBroken controls whether to warn about broken links
	warnOnBroken bool
}

// NewWikilinksPlugin creates a new WikilinksPlugin.
func NewWikilinksPlugin() *WikilinksPlugin {
	return &WikilinksPlugin{
		warnOnBroken: true,
	}
}

// Name returns the unique name of the plugin.
func (p *WikilinksPlugin) Name() string {
	return "wikilinks"
}

// Configure reads configuration options for the plugin.
func (p *WikilinksPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra != nil {
		if warnOnBroken, ok := config.Extra["wikilinks_warn_broken"].(bool); ok {
			p.warnOnBroken = warnOnBroken
		}
	}
	return nil
}

// Transform processes wikilinks in all post content.
func (p *WikilinksPlugin) Transform(m *lifecycle.Manager) error {
	// Use the shared post index instead of building our own maps
	postIndex := m.PostIndex()

	// Process each post
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		content, warnings := p.processWikilinks(post.Content, postIndex)
		post.Content = content

		// Store warnings if any
		if len(warnings) > 0 && p.warnOnBroken {
			existingWarnings, _ := post.Extra["wikilink_warnings"].([]string) //nolint:errcheck // type assertion ok to fail
			post.Set("wikilink_warnings", append(existingWarnings, warnings...))
		}

		return nil
	})
}

// wikilinkRegex matches [[slug]] and [[slug|display text]] patterns.
// It captures:
// - Group 1: The slug
// - Group 2: Optional display text (after the pipe)
var wikilinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// wikilinksCodeBlockRegex matches fenced code blocks to avoid transforming wikilinks inside them.
var wikilinksCodeBlockRegex = regexp.MustCompile("(?s)(```[^`]*```|~~~[^~]*~~~)")

// processWikilinks replaces wikilink syntax with HTML anchor tags.
// Returns the processed content and any warnings about broken links.
// Wikilinks inside fenced code blocks are preserved and not transformed.
func (p *WikilinksPlugin) processWikilinks(content string, postIndex *lifecycle.PostIndex) (processed string, warnings []string) {
	// Split content by fenced code blocks to avoid transforming wikilinks inside them
	// Match ``` or ~~~ fenced code blocks (with optional language identifier)

	// Find all code blocks and their positions
	codeBlocks := wikilinksCodeBlockRegex.FindAllStringIndex(content, -1)

	// If no code blocks, process the entire content
	if len(codeBlocks) == 0 {
		processed = p.processWikilinksInText(content, postIndex, &warnings)
		return processed, warnings
	}

	// Process content in segments, skipping code blocks
	var result strings.Builder
	lastEnd := 0

	for _, block := range codeBlocks {
		start, end := block[0], block[1]

		// Process text before this code block
		if start > lastEnd {
			processed := p.processWikilinksInText(content[lastEnd:start], postIndex, &warnings)
			result.WriteString(processed)
		}

		// Keep code block unchanged
		result.WriteString(content[start:end])
		lastEnd = end
	}

	// Process any remaining text after the last code block
	if lastEnd < len(content) {
		processed := p.processWikilinksInText(content[lastEnd:], postIndex, &warnings)
		result.WriteString(processed)
	}

	return result.String(), warnings
}

// processWikilinksInText processes wikilinks in a text segment (not inside code blocks).
// Uses the shared PostIndex for O(1) lookups.
func (p *WikilinksPlugin) processWikilinksInText(text string, postIndex *lifecycle.PostIndex, warnings *[]string) string {
	return wikilinkRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract groups from the match
		groups := wikilinkRegex.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}

		slug := strings.TrimSpace(groups[1])
		displayText := ""
		if len(groups) >= 3 && groups[2] != "" {
			displayText = strings.TrimSpace(groups[2])
		}

		// Use the shared PostIndex for lookup
		targetPost := postIndex.LookupBySlug(slug)

		if targetPost == nil {
			// Target post not found - warn and keep original syntax
			*warnings = append(*warnings, fmt.Sprintf("broken wikilink: [[%s]]", slug))
			return match
		}

		// Determine the display text
		if displayText == "" {
			// Use post title if available, otherwise use slug
			if targetPost.Title != nil && *targetPost.Title != "" {
				displayText = *targetPost.Title
			} else {
				displayText = targetPost.Slug
			}
		}

		// Generate HTML anchor tag with wikilink class for styling
		href := targetPost.Href
		if href == "" {
			href = "/" + targetPost.Slug + "/"
		}

		// Build data attributes for tooltip
		dataAttrs := ""
		if targetPost.Title != nil && *targetPost.Title != "" {
			dataAttrs += fmt.Sprintf(` data-title=%q`, html.EscapeString(*targetPost.Title))
		}
		if targetPost.Description != nil && *targetPost.Description != "" {
			dataAttrs += fmt.Sprintf(` data-description=%q`, html.EscapeString(*targetPost.Description))
		}
		if targetPost.Date != nil {
			dataAttrs += fmt.Sprintf(` data-date=%q`, targetPost.Date.Format("2006-01-02"))
		}

		return fmt.Sprintf(`<a href=%q class="wikilink"%s>%s</a>`,
			html.EscapeString(href),
			dataAttrs,
			html.EscapeString(displayText))
	})
}

// SetWarnOnBroken enables or disables warnings for broken links.
func (p *WikilinksPlugin) SetWarnOnBroken(warn bool) {
	p.warnOnBroken = warn
}

// Ensure WikilinksPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*WikilinksPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*WikilinksPlugin)(nil)
	_ lifecycle.TransformPlugin = (*WikilinksPlugin)(nil)
)
