package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
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
func (p *LoadPlugin) parseFile(path, content string) (*models.Post, error) {
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

	// Generate slug if not explicitly set in frontmatter
	// If slug was explicitly set (even to empty string), respect it
	if !post.Has("_slug_explicit") && post.Slug == "" {
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

	// Slug - support custom slugs including explicit empty string for homepage
	if slugVal, exists := metadata["slug"]; exists {
		slug := normalizeCustomSlug(GetString(metadata, "slug"))
		post.Slug = slug
		// Mark that slug was explicitly set (prevents auto-generation)
		post.Set("_slug_explicit", true)
		_ = slugVal // Exists check used, value handled via GetString
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
	// Normalize the date string first
	s = normalizeDateString(s)

	// Common date formats to try (most specific first)
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"02-01-2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2 January 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// normalizeDateString normalizes a date string to handle common variations.
// It handles:
// - Single-digit hours/minutes (e.g., "1:00:00" -> "01:00:00")
// - Malformed time components (e.g., "8:011:00" -> "8:11:00")
// - Extra whitespace
func normalizeDateString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Early return if no time component
	if !strings.Contains(s, ":") {
		return s
	}

	// Fix malformed time components like "8:011:00" -> "08:11:00"
	// This regex finds time components with potentially extra leading zeros
	timeFixRegex := regexp.MustCompile(`(\d{1,2}):0*(\d{1,2}):0*(\d{1,2})`)
	s = timeFixRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := timeFixRegex.FindStringSubmatch(match)
		if len(parts) == 4 {
			h, err := strconv.Atoi(parts[1])
			if err != nil {
				return match
			}
			m, err := strconv.Atoi(parts[2])
			if err != nil {
				return match
			}
			sec, err := strconv.Atoi(parts[3])
			if err != nil {
				return match
			}
			return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
		}
		return match
	})

	// Normalize single-digit hours in time component
	// Match patterns like " 1:00" or "T1:00" and pad the hour
	singleDigitHourRegex := regexp.MustCompile(`([ T])(\d):(\d{2})`)
	s = singleDigitHourRegex.ReplaceAllString(s, "${1}0${2}:${3}")

	// Handle time at start of string or after date with space
	if matched, err := regexp.MatchString(`^\d:\d{2}`, s); err == nil && matched {
		s = "0" + s
	}

	return s
}

// normalizeCustomSlug normalizes a slug from frontmatter.
// It handles:
//   - "/" or "" -> "" (homepage)
//   - "/docs/page" -> "docs/page" (strip leading slash)
//   - "docs/page/" -> "docs/page" (strip trailing slash)
//   - Preserves internal structure for nested paths
func normalizeCustomSlug(slug string) string {
	// Trim whitespace
	slug = strings.TrimSpace(slug)

	// "/" means homepage (empty slug)
	if slug == "/" {
		return ""
	}

	// Strip leading and trailing slashes
	slug = strings.Trim(slug, "/")

	return slug
}
