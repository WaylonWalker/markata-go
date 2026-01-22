// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"regexp"
	"strings"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// TocPlugin extracts headings from markdown content and builds a
// hierarchical table of contents during the transform stage.
type TocPlugin struct {
	// minLevel is the minimum heading level to include (default: 2)
	minLevel int

	// maxLevel is the maximum heading level to include (default: 4)
	maxLevel int
}

// NewTocPlugin creates a new TocPlugin with default settings.
func NewTocPlugin() *TocPlugin {
	return &TocPlugin{
		minLevel: 2,
		maxLevel: 4,
	}
}

// Name returns the unique name of the plugin.
func (p *TocPlugin) Name() string {
	return "toc"
}

// Configure reads configuration options for the plugin.
func (p *TocPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra != nil {
		if minLevel, ok := config.Extra["toc_min_level"].(int); ok && minLevel >= 1 && minLevel <= 6 {
			p.minLevel = minLevel
		}
		if maxLevel, ok := config.Extra["toc_max_level"].(int); ok && maxLevel >= 1 && maxLevel <= 6 {
			p.maxLevel = maxLevel
		}
	}
	return nil
}

// Transform extracts headings from markdown and builds a TOC for each post.
func (p *TocPlugin) Transform(m *lifecycle.Manager) error {
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		toc := p.extractTOC(post.Content)
		if len(toc) > 0 {
			post.Set("toc", toc)
		}

		return nil
	})
}

// TocEntry represents a single entry in the table of contents.
type TocEntry struct {
	// Level is the heading level (1-6)
	Level int `json:"level"`

	// Text is the heading text
	Text string `json:"text"`

	// ID is the anchor ID for the heading
	ID string `json:"id"`

	// Children contains nested headings
	Children []*TocEntry `json:"children,omitempty"`
}

// headingRegex matches ATX-style markdown headings (# Heading).
// Captures: Group 1 = hash marks, Group 2 = heading text
var headingRegex = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)(?:\s*#*)?\s*$`)

// idRegex matches characters that should be removed from heading IDs.
var idRegex = regexp.MustCompile(`[^\w\s-]`)

// extractTOC extracts headings from markdown content and builds a hierarchical TOC.
func (p *TocPlugin) extractTOC(content string) []*TocEntry {
	matches := headingRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	// Extract flat list of headings
	var headings []*TocEntry
	idCounts := make(map[string]int)

	for _, match := range matches {
		level := len(match[1])
		text := strings.TrimSpace(match[2])

		// Skip headings outside our level range
		if level < p.minLevel || level > p.maxLevel {
			continue
		}

		// Generate ID from text
		id := p.generateID(text, idCounts)

		headings = append(headings, &TocEntry{
			Level:    level,
			Text:     text,
			ID:       id,
			Children: make([]*TocEntry, 0),
		})
	}

	if len(headings) == 0 {
		return nil
	}

	// Build hierarchical structure
	return p.buildHierarchy(headings)
}

// generateID creates a URL-safe ID from heading text.
// Handles duplicate IDs by appending numbers.
func (p *TocPlugin) generateID(text string, idCounts map[string]int) string {
	// Convert to lowercase
	id := strings.ToLower(text)

	// Replace spaces with hyphens
	id = strings.ReplaceAll(id, " ", "-")

	// Remove special characters
	id = idRegex.ReplaceAllString(id, "")

	// Collapse multiple hyphens
	id = multiHyphenRegex.ReplaceAllString(id, "-")

	// Trim leading/trailing hyphens
	id = strings.Trim(id, "-")

	// Handle duplicates
	baseID := id
	count := idCounts[baseID]
	idCounts[baseID] = count + 1

	if count > 0 {
		id = strings.ToLower(strings.TrimSpace(id)) + "-" + strings.Repeat("1", count)
	}

	return id
}

// buildHierarchy converts a flat list of headings into a nested structure.
func (p *TocPlugin) buildHierarchy(headings []*TocEntry) []*TocEntry {
	if len(headings) == 0 {
		return nil
	}

	// Find the minimum level to use as root level
	minLevel := 6
	for _, h := range headings {
		if h.Level < minLevel {
			minLevel = h.Level
		}
	}

	var roots []*TocEntry
	var stack []*TocEntry

	for _, heading := range headings {
		// Adjust level relative to minimum
		entry := &TocEntry{
			Level:    heading.Level,
			Text:     heading.Text,
			ID:       heading.ID,
			Children: make([]*TocEntry, 0),
		}

		// Pop stack until we find a parent at a lower level
		for len(stack) > 0 && stack[len(stack)-1].Level >= entry.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			// This is a root-level heading
			roots = append(roots, entry)
		} else {
			// Add as child of the top of stack
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, entry)
		}

		// Push current heading onto stack
		stack = append(stack, entry)
	}

	return roots
}

// SetLevelRange sets the minimum and maximum heading levels to include.
func (p *TocPlugin) SetLevelRange(min, max int) {
	if min >= 1 && min <= 6 {
		p.minLevel = min
	}
	if max >= 1 && max <= 6 && max >= min {
		p.maxLevel = max
	}
}

// Ensure TocPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*TocPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*TocPlugin)(nil)
	_ lifecycle.TransformPlugin = (*TocPlugin)(nil)
)
