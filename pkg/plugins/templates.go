package plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// Output format constants.
const (
	formatHTML     = "html"
	formatTxt      = "txt"
	formatText     = "text"
	formatMarkdown = "markdown"
	formatMD       = "md"
	formatOG       = "og"

	// defaultTemplate is the default template name for posts.
	defaultTemplate = "post.html"
)

// TemplatesPlugin wraps rendered markdown content in HTML templates.
// It operates during the render stage, after markdown has been converted to HTML.
type TemplatesPlugin struct {
	engine       *templates.Engine
	layoutConfig *models.LayoutConfig
	config       *lifecycle.Config
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

	// Store this plugin in cache for other plugins to use per-format resolution
	m.Cache().Set("templates.plugin", p)

	// Get layout config if available
	switch lc := config.Extra["layout"].(type) {
	case *models.LayoutConfig:
		p.layoutConfig = lc
	case models.LayoutConfig:
		p.layoutConfig = &lc
	}

	// Store config reference for template preset resolution
	p.config = config

	return nil
}

// resolveTemplate determines the template to use for a post (HTML format).
// This is a convenience wrapper for resolveTemplateForFormat with "html" format.
// Priority: frontmatter per-format -> preset -> simple template -> layout config -> global default
func (p *TemplatesPlugin) resolveTemplate(post *models.Post) string {
	return p.resolveTemplateForFormat(post, "html")
}

// resolveTemplateForFormat determines the template to use for a post and output format.
// Resolution priority:
// 1. Frontmatter per-format override (templates.html, templates.txt, etc.)
// 2. Frontmatter template preset (template: blog → expand to preset)
// 3. Frontmatter simple template (template: post.html → use for current format)
// 4. Layout config (path/feed-based)
// 5. Global default for format (default_templates.html, etc.)
// 6. Hardcoded default (post.html, default.txt, etc.)
func (p *TemplatesPlugin) resolveTemplateForFormat(post *models.Post, format string) string {
	// 1. Check per-format override in frontmatter
	if post.Templates != nil {
		if tmpl, ok := post.Templates[format]; ok && tmpl != "" {
			return tmpl
		}
	}

	// 2. Check if template is a preset name
	if post.Template != "" && p.config != nil {
		presets := getTemplatePresets(p.config)
		if preset, ok := presets[post.Template]; ok {
			tmpl := preset.TemplateForFormat(format)
			if tmpl != "" {
				return tmpl
			}
		}
	}

	// 3. Use template as explicit file (if has extension) and adapt for format
	if post.Template != "" && strings.Contains(post.Template, ".") {
		return adaptTemplateForFormat(post.Template, format)
	}

	// 4. Use template as-is if it doesn't have an extension
	// This might be a preset name that wasn't found, fall through to layout
	if post.Template != "" {
		// For HTML, use the template directly
		if format == formatHTML {
			return post.Template
		}
		// For other formats, try to adapt it
		return adaptTemplateForFormat(post.Template+".html", format)
	}

	// 5. Use layout configuration to determine template
	if p.layoutConfig != nil {
		// Get feed slug for feed-based layout lookup
		feedSlug := post.PrevNextFeed
		if feedSlug == "" {
			if feed, ok := post.Extra["feed"].(string); ok {
				feedSlug = feed
			}
		}

		// Get post path for path-based layout lookup
		postPath := post.Href
		if postPath == "" {
			postPath = "/" + strings.TrimPrefix(post.Path, "/")
		}

		// Resolve layout based on path and feed
		layout := p.layoutConfig.ResolveLayout(postPath, feedSlug)
		if layout != "" {
			baseTemplate := models.LayoutToTemplate(layout)
			return adaptTemplateForFormat(baseTemplate, format)
		}
	}

	// 6. Check global default templates from config
	if p.config != nil {
		defaultTemplates := getDefaultTemplates(p.config)
		if tmpl, ok := defaultTemplates[format]; ok && tmpl != "" {
			return tmpl
		}
	}

	// 7. Fall back to hardcoded defaults per format
	return getHardcodedDefault(format)
}

// adaptTemplateForFormat adapts a template name for a specific output format.
// For example: post.html → post.txt, post.md, post-og.html
func adaptTemplateForFormat(template, format string) string {
	ext := filepath.Ext(template)
	base := strings.TrimSuffix(template, ext)

	switch format {
	case formatHTML:
		return template
	case formatTxt, formatText:
		return base + ".txt"
	case formatMarkdown, formatMD:
		return base + ".md"
	case formatOG:
		return base + "-og.html"
	default:
		return template
	}
}

// getHardcodedDefault returns the hardcoded default template for a format.
func getHardcodedDefault(format string) string {
	switch format {
	case formatHTML:
		return defaultTemplate
	case formatTxt, formatText:
		return "default.txt"
	case formatMarkdown, formatMD:
		return "raw.txt"
	case formatOG:
		return "og-card.html"
	default:
		return defaultTemplate
	}
}

// getTemplatePresets extracts TemplatePresets from lifecycle.Config.Extra.
func getTemplatePresets(config *lifecycle.Config) map[string]models.TemplatePreset {
	if config.Extra == nil {
		return nil
	}
	if presets, ok := config.Extra["template_presets"].(map[string]models.TemplatePreset); ok {
		return presets
	}
	return nil
}

// getDefaultTemplates extracts DefaultTemplates from lifecycle.Config.Extra.
func getDefaultTemplates(config *lifecycle.Config) map[string]string {
	if config.Extra == nil {
		return nil
	}
	if defaults, ok := config.Extra["default_templates"].(map[string]string); ok {
		return defaults
	}
	return nil
}

// Render wraps markdown content in templates.
// This runs after markdown rendering, using post.ArticleHTML as the body.
// Skips posts that don't need rebuilding (incremental builds).
func (p *TemplatesPlugin) Render(m *lifecycle.Manager) error {
	if p.engine == nil {
		return fmt.Errorf("template engine not initialized")
	}

	// Get config for template context
	config := m.Config()

	// Get build cache to check if posts need rebuilding
	cache := GetBuildCache(m)
	changedSlugs := getChangedSlugsMap(cache)

	// Collect private paths for robots.txt template variable
	privatePaths := collectPrivatePaths(m.Posts())

	// Process each post concurrently
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		// Skip posts marked to skip or without article HTML
		if post.Skip || post.ArticleHTML == "" {
			return nil
		}

		// Try to use cached HTML for unchanged posts
		if cachedHTML := p.tryGetCachedHTML(post, cache, changedSlugs); cachedHTML != "" {
			post.HTML = cachedHTML
			return nil
		}

		// Render the template
		html, err := p.renderPost(post, config, m, privatePaths)
		if err != nil {
			return err
		}
		post.HTML = html

		// Cache the full HTML for future incremental builds
		if cache != nil && post.InputHash != "" {
			//nolint:errcheck // caching is best-effort, failures are non-fatal
			cache.CacheFullHTML(post.Path, html)
		}

		return nil
	})
}

// getChangedSlugsMap returns a map of slugs that changed in this build.
func getChangedSlugsMap(cache *buildcache.Cache) map[string]bool {
	if cache == nil {
		return nil
	}
	changedSlugs := make(map[string]bool)
	for _, slug := range cache.GetChangedSlugs() {
		changedSlugs[slug] = true
	}
	return changedSlugs
}

// tryGetCachedHTML checks if a post can use cached HTML.
// Returns cached HTML if available and valid, empty string otherwise.
func (p *TemplatesPlugin) tryGetCachedHTML(post *models.Post, cache *buildcache.Cache, changedSlugs map[string]bool) string {
	if cache == nil || post.InputHash == "" {
		return ""
	}

	// Check if post itself changed
	if cache.ShouldRebuild(post.Path, post.InputHash, post.Template) {
		return ""
	}

	// Check if any dependency changed
	if len(changedSlugs) > 0 {
		for _, dep := range post.Dependencies {
			if changedSlugs[dep] {
				return ""
			}
		}
		// Check if this post's slug is in changedSlugs
		if changedSlugs[post.Slug] {
			return ""
		}
	}

	// Try to load cached HTML
	return cache.GetCachedFullHTML(post.Path)
}

// renderPost renders a single post using the appropriate template.
func (p *TemplatesPlugin) renderPost(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager, privatePaths []string) (string, error) {
	// Determine which template to use
	templateName := p.resolveTemplate(post)

	// Check if template exists, fall back to post.html if not
	if !p.engine.TemplateExists(templateName) {
		templateName = "post.html"
		if !p.engine.TemplateExists(templateName) {
			return post.ArticleHTML, nil
		}
	}

	// Create template context
	ctx := templates.NewContext(post, post.ArticleHTML, ToModelsConfig(config))
	ctx = ctx.WithCore(m)
	ctx.Set("private_paths", privatePaths)

	// Render the template
	html, err := p.engine.Render(templateName, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to render template %q for post %q: %w", templateName, post.Path, err)
	}

	return html, nil
}

// collectPrivatePaths returns a list of paths (hrefs) for all private posts.
// These paths are used in robots.txt templates to add Disallow directives.
// Includes all format variants (.txt, .md, .og) and excludes the robots post itself.
func collectPrivatePaths(posts []*models.Post) []string {
	var paths []string
	for _, post := range posts {
		if post.Private && !post.Draft && !post.Skip {
			// Skip the robots post itself to avoid self-reference
			if post.Slug == "robots" {
				continue
			}
			// Add base href (e.g., /slug/)
			paths = append(paths, post.Href)
			// Add format variants
			// For regular posts: /slug/index.txt, /slug/index.md, /slug.og/
			if post.Slug != "" {
				paths = append(paths,
					"/"+post.Slug+"/index.txt",
					"/"+post.Slug+"/index.md",
					"/"+post.Slug+".og/",
				)
			}
		}
	}
	return paths
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

// ToModelsConfig converts lifecycle.Config to models.Config for template context.
// This is exported for use by other plugins that need to build template contexts
// with the full config (e.g., publish_feeds, blogroll).
func ToModelsConfig(config *lifecycle.Config) *models.Config {
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
	switch layoutVal := config.Extra["layout"].(type) {
	case *models.LayoutConfig:
		modelsConfig.Layout = *layoutVal
	case models.LayoutConfig:
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

	// Copy Search config if available, otherwise use defaults
	// This ensures search is enabled by default with position "navbar"
	if search, ok := config.Extra["search"].(models.SearchConfig); ok {
		modelsConfig.Search = search
	} else {
		modelsConfig.Search = models.NewSearchConfig()
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

	// Copy Theme config if available
	if theme, ok := config.Extra["theme"].(models.ThemeConfig); ok {
		modelsConfig.Theme = theme
	}

	// Copy the entire Extra map so templates can access dynamic plugin config
	// (e.g., glightbox_enabled, glightbox_options set by image_zoom plugin)
	if config.Extra != nil {
		modelsConfig.Extra = make(map[string]any)
		for k, v := range config.Extra {
			modelsConfig.Extra[k] = v
		}
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
