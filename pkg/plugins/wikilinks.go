// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
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
	posts := m.Posts()

	// Build a map of slug -> post for quick lookup
	postMap := make(map[string]*models.Post)
	for _, post := range posts {
		if post.Slug != "" {
			postMap[post.Slug] = post
		}
	}

	// Process each post
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		content, warnings := p.processWikilinks(post.Content, postMap)
		post.Content = content

		// Store warnings if any
		if len(warnings) > 0 && p.warnOnBroken {
			existingWarnings, _ := post.Extra["wikilink_warnings"].([]string)
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

// processWikilinks replaces wikilink syntax with HTML anchor tags.
// Returns the processed content and any warnings about broken links.
func (p *WikilinksPlugin) processWikilinks(content string, postMap map[string]*models.Post) (string, []string) {
	var warnings []string

	result := wikilinkRegex.ReplaceAllStringFunc(content, func(match string) string {
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

		// Normalize slug for lookup
		normalizedSlug := normalizeSlug(slug)

		// Look up the target post
		targetPost, found := postMap[normalizedSlug]
		if !found {
			// Try case-insensitive lookup
			for postSlug, post := range postMap {
				if strings.EqualFold(postSlug, normalizedSlug) {
					targetPost = post
					found = true
					break
				}
			}
		}

		if !found {
			// Target post not found - warn and keep original syntax
			warnings = append(warnings, fmt.Sprintf("broken wikilink: [[%s]]", slug))
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

		// Generate HTML anchor tag
		href := targetPost.Href
		if href == "" {
			href = "/" + targetPost.Slug + "/"
		}

		return fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(href),
			html.EscapeString(displayText))
	})

	return result, warnings
}

// normalizeSlug normalizes a slug for lookup by converting to lowercase
// and replacing spaces with hyphens.
func normalizeSlug(slug string) string {
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = slugifyRegex.ReplaceAllString(slug, "")
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
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
