package plugins

import (
	"fmt"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// TemplatesPlugin wraps rendered markdown content in HTML templates.
// It operates during the render stage, after markdown has been converted to HTML.
type TemplatesPlugin struct {
	engine       *templates.Engine
	layoutConfig *models.LayoutConfig
}

// NewTemplatesPlugin creates a new templates plugin.
func NewTemplatesPlugin() *TemplatesPlugin {
	return &TemplatesPlugin{}
}

// Name returns the plugin name.
func (p *TemplatesPlugin) Name() string {
	return PluginNameTemplates
}

// Configure initializes the template engine from the config.
func (p *TemplatesPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Get templates directory from config
	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	// Get theme name from config (default to "default")
	themeName := ThemeDefault
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

	// Get layout config if available
	switch lc := config.Extra["layout"].(type) {
	case *models.LayoutConfig:
		p.layoutConfig = lc
	case models.LayoutConfig:
		p.layoutConfig = &lc
	}

	return nil
}

// resolveTemplate determines the template to use for a post.
// Priority: frontmatter template > path-based layout > feed-based layout > global default > "post.html"
func (p *TemplatesPlugin) resolveTemplate(post *models.Post) string {
	// 1. Check for explicit template in frontmatter (highest priority)
	if post.Template != "" {
		return post.Template
	}

	// 2. Use layout configuration to determine template
	if p.layoutConfig != nil {
		// Get feed slug for feed-based layout lookup
		// Check PrevNextFeed first, then look in Extra for feed information
		feedSlug := post.PrevNextFeed
		if feedSlug == "" {
			if feed, ok := post.Extra["feed"].(string); ok {
				feedSlug = feed
			}
		}

		// Get post path for path-based layout lookup
		// Use the Href which represents the URL structure (e.g., /docs/getting-started/)
		postPath := post.Href
		if postPath == "" {
			// Fall back to Path if Href is not set
			postPath = "/" + strings.TrimPrefix(post.Path, "/")
		}

		// Resolve layout based on path and feed
		layout := p.layoutConfig.ResolveLayout(postPath, feedSlug)
		if layout != "" {
			return models.LayoutToTemplate(layout)
		}
	}

	// 3. Fall back to default template
	return "post.html"
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
		templateName := p.resolveTemplate(post)

		// Check if template exists, fall back to post.html if not
		if !p.engine.TemplateExists(templateName) {
			// Template not found, fall back to default post.html
			templateName = "post.html"

			// If even post.html doesn't exist, use article HTML directly
			if !p.engine.TemplateExists(templateName) {
				post.HTML = post.ArticleHTML
				return nil
			}
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
	modelsConfig := &models.Config{
		OutputDir:    config.OutputDir,
		Title:        getStringFromExtra(config.Extra, "title"),
		URL:          getStringFromExtra(config.Extra, "url"),
		Description:  getStringFromExtra(config.Extra, "description"),
		Author:       getStringFromExtra(config.Extra, "author"),
		TemplatesDir: getStringFromExtra(config.Extra, "templates_dir"),
	}

	// Copy nav items if available
	if navItems, ok := config.Extra["nav"].([]models.NavItem); ok {
		modelsConfig.Nav = navItems
	}

	// Copy footer config if available
	if footer, ok := config.Extra["footer"].(models.FooterConfig); ok {
		modelsConfig.Footer = footer
	}

	// Copy layout config if available
	if layout, ok := config.Extra["layout"].(*models.LayoutConfig); ok {
		modelsConfig.Layout = *layout
	} else if layoutVal, ok := config.Extra["layout"].(models.LayoutConfig); ok {
		modelsConfig.Layout = layoutVal
	}

	// Copy sidebar config if available
	if sidebar, ok := config.Extra["sidebar"].(models.SidebarConfig); ok {
		modelsConfig.Sidebar = sidebar
	}

	// Copy toc config if available
	if toc, ok := config.Extra["toc"].(models.TocConfig); ok {
		modelsConfig.Toc = toc
	}

	// Copy header config if available
	if header, ok := config.Extra["header"].(models.HeaderLayoutConfig); ok {
		modelsConfig.Header = header
	}

	// Copy SEO config if available
	switch seoVal := config.Extra["seo"].(type) {
	case models.SEOConfig:
		modelsConfig.SEO = seoVal
	case map[string]interface{}:
		modelsConfig.SEO = models.SEOConfig{
			TwitterHandle: getStringFromMap(seoVal, "twitter_handle"),
			DefaultImage:  getStringFromMap(seoVal, "default_image"),
			LogoURL:       getStringFromMap(seoVal, "logo_url"),
		}
	}

	// Copy Search config if available
	if search, ok := config.Extra["search"].(models.SearchConfig); ok {
		modelsConfig.Search = search
	}

	// Copy Components config if available
	if components, ok := config.Extra["components"].(models.ComponentsConfig); ok {
		modelsConfig.Components = components
	}

	// Copy PostFormats config if available
	if postFormats, ok := config.Extra["post_formats"].(models.PostFormatsConfig); ok {
		modelsConfig.PostFormats = postFormats
	}

	// Copy Head config if available
	if head, ok := config.Extra["head"].(models.HeadConfig); ok {
		modelsConfig.Head = head
	}

	return modelsConfig
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

// getStringFromMap safely gets a string value from a map.
func getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
