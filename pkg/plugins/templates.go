package plugins

import (
	"fmt"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
	"github.com/example/markata-go/pkg/templates"
)

// TemplatesPlugin wraps rendered markdown content in HTML templates.
// It operates during the render stage, after markdown has been converted to HTML.
type TemplatesPlugin struct {
	engine *templates.Engine
}

// NewTemplatesPlugin creates a new templates plugin.
func NewTemplatesPlugin() *TemplatesPlugin {
	return &TemplatesPlugin{}
}

// Name returns the plugin name.
func (p *TemplatesPlugin) Name() string {
	return "templates"
}

// Configure initializes the template engine from the config.
func (p *TemplatesPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Get templates directory from config
	templatesDir := "templates"
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	// Get theme name from config (default to "default")
	themeName := "default"
	if extra := config.Extra; extra != nil {
		if theme, ok := extra["theme"].(map[string]interface{}); ok {
			if name, ok := theme["name"].(string); ok && name != "" {
				themeName = name
			}
		}
		// Also check for simple theme string
		if name, ok := extra["theme"].(string); ok && name != "" {
			themeName = name
		}
	}

	// Initialize template engine with theme support
	engine, err := templates.NewEngineWithTheme(templatesDir, themeName)
	if err != nil {
		return fmt.Errorf("failed to initialize template engine: %w", err)
	}
	p.engine = engine

	// Store engine in cache for other plugins to use
	m.Cache().Set("templates.engine", engine)

	return nil
}

// Render wraps markdown content in templates.
// This runs after markdown rendering, using post.ArticleHTML as the body.
func (p *TemplatesPlugin) Render(m *lifecycle.Manager) error {
	if p.engine == nil {
		return fmt.Errorf("template engine not initialized")
	}

	// Get config for template context
	config := m.Config()

	// Process each post concurrently
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		// Skip posts marked to skip or without article HTML
		if post.Skip || post.ArticleHTML == "" {
			return nil
		}

		// Determine which template to use
		templateName := post.Template
		if templateName == "" {
			templateName = "post.html"
		}

		// Check if template exists
		if !p.engine.TemplateExists(templateName) {
			// If no template exists, just use the article HTML as final HTML
			post.HTML = post.ArticleHTML
			return nil
		}

		// Create template context
		ctx := templates.NewContext(post, post.ArticleHTML, toModelsConfig(config))
		ctx = ctx.WithCore(m)

		// Render the template
		html, err := p.engine.Render(templateName, ctx)
		if err != nil {
			return fmt.Errorf("failed to render template %q for post %q: %w", templateName, post.Path, err)
		}

		post.HTML = html
		return nil
	})
}

// Priority returns the plugin priority for the given stage.
// Templates should run late in the render stage, after markdown rendering.
func (p *TemplatesPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after markdown rendering
	}
	return lifecycle.PriorityDefault
}

// Engine returns the template engine for use by other plugins.
func (p *TemplatesPlugin) Engine() *templates.Engine {
	return p.engine
}

// toModelsConfig converts lifecycle.Config to models.Config for template context.
func toModelsConfig(config *lifecycle.Config) *models.Config {
	if config == nil {
		return nil
	}
	// Convert lifecycle.Config to models.Config
	return &models.Config{
		OutputDir:    config.OutputDir,
		Title:        getStringFromExtra(config.Extra, "title"),
		URL:          getStringFromExtra(config.Extra, "url"),
		Description:  getStringFromExtra(config.Extra, "description"),
		Author:       getStringFromExtra(config.Extra, "author"),
		TemplatesDir: getStringFromExtra(config.Extra, "templates_dir"),
	}
}

// getStringFromExtra safely gets a string value from the Extra map.
func getStringFromExtra(extra map[string]interface{}, key string) string {
	if extra == nil {
		return ""
	}
	if v, ok := extra[key].(string); ok {
		return v
	}
	return ""
}
