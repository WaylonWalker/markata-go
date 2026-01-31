package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/slugmatch"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// ErrorPagesPlugin generates static error pages (404.html) during build.
// This allows production web servers to serve custom error pages.
type ErrorPagesPlugin struct{}

// Compile-time interface verification.
var _ lifecycle.Plugin = (*ErrorPagesPlugin)(nil)
var _ lifecycle.WritePlugin = (*ErrorPagesPlugin)(nil)

// NewErrorPagesPlugin creates a new ErrorPagesPlugin.
func NewErrorPagesPlugin() *ErrorPagesPlugin {
	return &ErrorPagesPlugin{}
}

// Name returns the plugin name.
func (p *ErrorPagesPlugin) Name() string {
	return "error_pages"
}

// Write generates static error pages during the Write stage.
func (p *ErrorPagesPlugin) Write(m *lifecycle.Manager) error {
	// Get the full config
	lcConfig := m.Config()
	if lcConfig == nil || lcConfig.Extra == nil {
		return nil
	}

	cfg, ok := lcConfig.Extra["models_config"].(*models.Config)
	if !ok || cfg == nil {
		// No config available, skip
		return nil
	}

	// Check if 404 page is enabled
	if !cfg.ErrorPages.Is404Enabled() {
		return nil
	}

	// Generate 404 page
	return p.generate404Page(m, cfg)
}

// generate404Page creates the static 404.html file.
func (p *ErrorPagesPlugin) generate404Page(m *lifecycle.Manager, cfg *models.Config) error {
	// Get posts for suggestions
	posts := m.Posts()
	maxSuggestions := cfg.ErrorPages.MaxSuggestions
	if maxSuggestions <= 0 {
		maxSuggestions = 5
	}

	// Get recent posts for fallback (no slug to match against in static build)
	var recentPosts []*models.Post
	if len(posts) > 0 {
		// Sort by date descending and take top N
		sortedPosts := make([]*models.Post, len(posts))
		copy(sortedPosts, posts)
		sort.Slice(sortedPosts, func(i, j int) bool {
			if sortedPosts[i].Date == nil {
				return false
			}
			if sortedPosts[j].Date == nil {
				return true
			}
			return sortedPosts[i].Date.After(*sortedPosts[j].Date)
		})

		limit := maxSuggestions
		if limit > len(sortedPosts) {
			limit = len(sortedPosts)
		}
		recentPosts = sortedPosts[:limit]
	}

	// Create template engine
	templatesDir := cfg.TemplatesDir
	if templatesDir == "" {
		templatesDir = "templates"
	}
	engine, err := templates.NewEngineWithTheme(templatesDir, cfg.Theme.Name)
	if err != nil {
		return fmt.Errorf("creating template engine for 404 page: %w", err)
	}

	// Create template context
	ctx := templates.NewContext(nil, "", cfg)
	ctx.Set("requested_path", "")                          // Empty in static build
	ctx.Set("suggested_posts", []map[string]interface{}{}) // Empty - we can't know the path ahead of time
	ctx.Set("recent_posts", templates.PostsToMaps(recentPosts))

	// Determine template name
	templateName := cfg.ErrorPages.Custom404Template
	if templateName == "" {
		templateName = "404.html"
	}

	// Render the 404 template
	html, err := engine.Render(templateName, ctx)
	if err != nil {
		return fmt.Errorf("rendering 404 template: %w", err)
	}

	// Write to output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	outputPath := filepath.Join(outputDir, "404.html")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory for 404 page: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("writing 404.html: %w", err)
	}

	return nil
}

// Ensure slugmatch is used (for interface checks during dev mode serving).
var _ = slugmatch.LevenshteinDistance
