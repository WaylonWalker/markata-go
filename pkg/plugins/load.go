package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// LoadPlugin parses markdown files into Post objects.
type LoadPlugin struct{}

// NewLoadPlugin creates a new LoadPlugin.
func NewLoadPlugin() *LoadPlugin {
	return &LoadPlugin{}
}

// Name returns the plugin identifier.
func (p *LoadPlugin) Name() string {
	return "load"
}

// Load reads and parses all discovered files into Post objects.
func (p *LoadPlugin) Load(m *lifecycle.Manager) error {
	files := m.Files()
	config := m.Config()
	baseDir := config.ContentDir
	if baseDir == "" {
		baseDir = "."
	}

	for _, file := range files {
		// Construct full path
		fullPath := file
		if !filepath.IsAbs(file) {
			fullPath = filepath.Join(baseDir, file)
		}

		// Read file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Parse the file
		post, err := p.parseFile(file, string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}

		m.AddPost(post)
	}

	return nil
}

// parseFile parses a markdown file's content into a Post object.
func (p *LoadPlugin) parseFile(path string, content string) (*models.Post, error) {
	// Parse frontmatter
	metadata, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Create post with defaults
	post := models.NewPost(path)
	post.Content = body

	// Apply metadata to post
	if err := p.applyMetadata(post, metadata); err != nil {
		return nil, err
	}

	// Generate slug if not set
	if post.Slug == "" {
		post.GenerateSlug()
	}

	// Generate href from slug
	post.GenerateHref()

	return post, nil
}

// applyMetadata applies parsed frontmatter metadata to a Post.
func (p *LoadPlugin) applyMetadata(post *models.Post, metadata map[string]interface{}) error {
	// Known fields to extract
	knownFields := map[string]bool{
		"title":       true,
		"date":        true,
		"published":   true,
		"draft":       true,
		"skip":        true,
		"tags":        true,
		"description": true,
		"template":    true,
		"slug":        true,
	}

	// Title
	if title := GetString(metadata, "title"); title != "" {
		post.Title = &title
	}

	// Date - handle various formats
	if dateVal, ok := metadata["date"]; ok {
		date, err := parseDate(dateVal)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
		post.Date = &date
	}

	// Published
	post.Published = GetBool(metadata, "published", false)

	// Draft
	post.Draft = GetBool(metadata, "draft", false)

	// Skip
	post.Skip = GetBool(metadata, "skip", false)

	// Tags
	if tags := GetStringSlice(metadata, "tags"); tags != nil {
		post.Tags = tags
	}

	// Description
	if desc := GetString(metadata, "description"); desc != "" {
		post.Description = &desc
	}

	// Template
	if template := GetString(metadata, "template"); template != "" {
		post.Template = template
	}

	// Slug
	if slug := GetString(metadata, "slug"); slug != "" {
		post.Slug = slug
	}

	// Store unknown fields in Extra
	for key, value := range metadata {
		if !knownFields[key] {
			post.Set(key, value)
		}
	}

	return nil
}

// parseDate attempts to parse a date value from various formats.
func parseDate(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseDateString(v)
	default:
		return time.Time{}, fmt.Errorf("unsupported date type: %T", value)
	}
}

// parseDateString parses a date string using common formats.
func parseDateString(s string) (time.Time, error) {
	// Common date formats to try
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02-01-2006",
		"January 2, 2006",
		"Jan 2, 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
