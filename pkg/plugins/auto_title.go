// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// AutoTitlePlugin auto-generates human-readable titles for posts that don't have one.
// It derives titles from filenames by replacing hyphens and underscores with spaces
// and applying title case.
type AutoTitlePlugin struct{}

// NewAutoTitlePlugin creates a new AutoTitlePlugin.
func NewAutoTitlePlugin() *AutoTitlePlugin {
	return &AutoTitlePlugin{}
}

// Name returns the unique name of the plugin.
func (p *AutoTitlePlugin) Name() string {
	return "auto_title"
}

// Priority returns the plugin priority for the given stage.
// Auto title should run very early in transform to have title available for other plugins.
func (p *AutoTitlePlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageTransform {
		return lifecycle.PriorityFirst
	}
	return lifecycle.PriorityDefault
}

// Transform generates titles for posts that don't have one.
func (p *AutoTitlePlugin) Transform(m *lifecycle.Manager) error {
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip {
			return nil
		}

		// Skip if title is already set
		if post.Title != nil && *post.Title != "" {
			return nil
		}

		// Generate title from filename
		title := p.generateTitle(post.Path)
		if title != "" {
			post.Title = &title
		}

		return nil
	})
}

// generateTitle creates a human-readable title from a file path.
func (p *AutoTitlePlugin) generateTitle(path string) string {
	// Extract filename without extension
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	if stem == "" {
		return ""
	}

	// Replace hyphens and underscores with spaces
	title := strings.ReplaceAll(stem, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Apply title case
	title = toTitleCase(title)

	return title
}

// toTitleCase converts a string to title case.
// Each word starts with an uppercase letter, rest are lowercase.
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		// Capitalize first rune, lowercase the rest
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// Ensure AutoTitlePlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*AutoTitlePlugin)(nil)
	_ lifecycle.TransformPlugin = (*AutoTitlePlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*AutoTitlePlugin)(nil)
)
