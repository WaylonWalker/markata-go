// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/htmltotext"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// defaultTxtTemplate is the default template name for txt output.
const defaultTxtTemplate = "default.txt"

// rawTxtTemplate is the template name for raw txt output.
const rawTxtTemplate = "raw.txt"

// rawMdTemplate is the template name for raw markdown output.
const rawMdTemplate = "raw.md"

// specialFiles are slugs that should have their content at /slug.ext rather than /slug/index.ext.
// These are standard web files that are expected at specific root-level locations.
var specialFiles = []string{"robots", "llms", "humans", "security", "ads"}

// isSpecialFile returns true if the slug is a special file that should be served at root level.
func isSpecialFile(slug string) bool {
	for _, special := range specialFiles {
		if slug == special {
			return true
		}
	}
	return false
}

// PublishHTMLPlugin writes individual post HTML files during the write stage.
// It supports multiple output formats: HTML, Markdown source, and OG card HTML.
type PublishHTMLPlugin struct {
	templatesPlugin *TemplatesPlugin
}

// NewPublishHTMLPlugin creates a new PublishHTMLPlugin.
func NewPublishHTMLPlugin() *PublishHTMLPlugin {
	return &PublishHTMLPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PublishHTMLPlugin) Name() string {
	return "publish_html"
}

// Write outputs each post to the configured output directory in enabled formats.
func (p *PublishHTMLPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Get template engine from cache (may be nil if templates plugin not used)
	var engine *templates.Engine
	if cached, ok := m.Cache().Get("templates.engine"); ok && cached != nil {
		if e, ok := cached.(*templates.Engine); ok {
			engine = e
		}
	}

	// Get templates plugin from cache for per-format template resolution
	if cached, ok := m.Cache().Get("templates.plugin"); ok {
		if tp, ok := cached.(*TemplatesPlugin); ok {
			p.templatesPlugin = tp
		}
	}

	// Pre-pass: Identify all posts that changed (hash mismatch) and mark their slugs.
	// This enables dependency-based invalidation in FilterPosts below.
	// Uses ShouldRebuildBatch to acquire the lock once instead of per-post.
	cache := GetBuildCache(m)
	if err := p.markChangedPosts(cache, m, config); err != nil {
		return err
	}

	// Single-pass filter: identify posts that need writing.
	// This replaces the previous two-pass approach (pre-filter + writeHTMLFormat check).
	postsNeedingWrite := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.Draft {
			return false
		}

		// For incremental builds, check if we can skip this post
		if cache != nil && post.InputHash != "" {
			if !cache.ShouldRebuildWithSlug(post.Path, post.Slug, post.InputHash, post.Template) {
				outputPath := filepath.Join(config.OutputDir, post.Slug, "index.html")
				if _, err := os.Stat(outputPath); err == nil {
					cache.MarkSkipped()
					return false
				}
			}
		}

		return true
	})

	// Phase 2: Process only posts that need writing concurrently
	return m.ProcessPostsSliceConcurrently(postsNeedingWrite, func(post *models.Post) error {
		return p.writePost(post, config, engine, m)
	})
}

func (p *PublishHTMLPlugin) markChangedPosts(cache *buildcache.Cache, m *lifecycle.Manager, config *lifecycle.Config) error {
	if cache == nil || m == nil || config == nil {
		return nil
	}

	posts := m.Posts()
	removed := lifecycle.GetServeRemovedPaths(m)
	postFormats := getPostFormatsConfig(config)

	batch := make([]struct{ Path, InputHash, Template string }, 0, len(posts))
	slugByPath := make(map[string]string, len(posts))
	for _, post := range posts {
		if post.Skip || post.Draft || post.InputHash == "" {
			continue
		}
		batch = append(batch, struct{ Path, InputHash, Template string }{
			Path:      post.Path,
			InputHash: post.InputHash,
			Template:  post.Template,
		})
		slugByPath[post.Path] = post.Slug
	}
	changed := cache.ShouldRebuildBatch(batch)
	for path := range changed {
		slug := slugByPath[path]
		cache.MarkSlugChanged(slug)
	}
	for _, removedPath := range removed {
		if slug := cache.Graph.SlugForPath(removedPath); slug != "" {
			cache.MarkSlugChanged(slug)
		}
		if err := p.removePostOutputs(removedPath, config, postFormats, cache); err != nil {
			return err
		}
	}

	return nil
}

// writePost writes a single post to its output location in all enabled formats.
// Shadow pages: Unpublished posts are still rendered but not included in feeds.
// This allows sharing draft content via direct URL while keeping it out of public listings.
func (p *PublishHTMLPlugin) writePost(post *models.Post, config *lifecycle.Config, engine *templates.Engine, m *lifecycle.Manager) error {
	// Skip posts marked as skip
	if post.Skip {
		return nil
	}

	// Skip drafts - these are truly private work-in-progress content
	if post.Draft {
		return nil
	}

	// Note: Unpublished posts (published: false) are still rendered as "shadow pages"
	// They won't appear in feeds (which filter by published == True) but can be
	// accessed via direct URL for review/sharing purposes.

	// Determine output path
	// Use slug to create: output_dir/slug/index.html
	if !post.Has("_slug_explicit") && post.Slug == "" {
		post.GenerateSlug()
	}

	outputDir := config.OutputDir
	postDir := filepath.Join(outputDir, post.Slug)

	// Create post directory
	if err := os.MkdirAll(postDir, 0o755); err != nil {
		return fmt.Errorf("creating post directory %s: %w", postDir, err)
	}

	// Get post formats config from Extra
	postFormats := getPostFormatsConfig(config)

	// Get build cache for incremental builds
	cache := GetBuildCache(m)

	// Write HTML format (default)
	if postFormats.IsHTMLEnabled() {
		if err := p.writeHTMLFormat(post, config, postDir, cache); err != nil {
			return err
		}
	}

	// Write Markdown format (raw source)
	// Uses reversed redirect: content at /slug.md, redirect at /slug/index.html
	// Skip redirect if HTML is enabled (index.html already has main content)
	// Skip for private posts to prevent plaintext content leaks
	if postFormats.Markdown && !post.Private {
		mdContent := p.buildFormatContent(post, config, m, "markdown")
		skipRedirect := postFormats.IsHTMLEnabled()
		if err := p.writeReversedFormatOutput(post.Slug, "md", mdContent, config.OutputDir, skipRedirect); err != nil {
			return err
		}
	}

	// Write Text format (plain text)
	// Uses reversed redirect: content at /slug.txt, redirect at /slug/index.html
	// Skip redirect if HTML is enabled (index.html already has main content)
	// Skip for private posts to prevent plaintext content leaks
	if postFormats.Text && !post.Private {
		// Use renderTextContent for txt format to leverage main's sophisticated template resolution
		txtContent := p.renderTextContent(post, config, engine)
		skipRedirect := postFormats.IsHTMLEnabled()
		if err := p.writeReversedFormatOutput(post.Slug, "txt", txtContent, config.OutputDir, skipRedirect); err != nil {
			return err
		}
	}

	// Write OG format (social card HTML)
	// Skip for private posts to prevent metadata leaks in OG cards
	if postFormats.OG && !post.Private {
		if err := p.writeOGFormat(post, config, postDir, engine); err != nil {
			return err
		}
	}

	return nil
}

func (p *PublishHTMLPlugin) removePostOutputs(sourcePath string, config *lifecycle.Config, postFormats models.PostFormatsConfig, cache *buildcache.Cache) error {
	if sourcePath == "" || config == nil {
		return nil
	}
	if cache == nil {
		return nil
	}
	if !lifecycle.IsServeFastModeFromConfig(config) {
		return nil
	}
	postCache := cache.GetCachedPost(sourcePath)
	if postCache == nil || postCache.Slug == "" {
		return nil
	}

	slug := postCache.Slug
	outputDir := config.OutputDir
	postDir := filepath.Join(outputDir, slug)

	if postFormats.IsHTMLEnabled() {
		_ = os.RemoveAll(postDir)
	} else {
		_ = os.RemoveAll(filepath.Join(postDir, "index.md"))
		_ = os.RemoveAll(filepath.Join(postDir, "index.txt"))
		_ = os.RemoveAll(filepath.Join(postDir, "og"))
		_ = os.Remove(filepath.Join(postDir, "index.html"))
	}
	if postFormats.Markdown {
		_ = os.Remove(filepath.Join(outputDir, slug+".md"))
	}
	if postFormats.Text {
		_ = os.Remove(filepath.Join(outputDir, slug+".txt"))
	}
	_ = os.RemoveAll(filepath.Join(postDir, "og"))

	return nil
}

// writeHTMLFormat writes the standard HTML output for a post.
// If incremental build caching is enabled, skips posts that haven't changed.
func (p *PublishHTMLPlugin) writeHTMLFormat(post *models.Post, config *lifecycle.Config, postDir string, cache *buildcache.Cache) error {
	// Determine HTML content to write
	var htmlContent string
	switch {
	case post.HTML != "":
		// Use pre-rendered HTML if available
		htmlContent = post.HTML
	case post.ArticleHTML != "":
		// Wrap ArticleHTML in a basic template
		htmlContent = p.wrapInTemplate(post, config)
	default:
		// No HTML content available
		return nil
	}

	// Write index.html
	outputPath := filepath.Join(postDir, "index.html")

	//nolint:gosec // G306: HTML output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	// Mark as rebuilt in cache (also tracks that this slug changed)
	if cache != nil && post.InputHash != "" {
		cache.MarkRebuiltWithSlug(post.Path, post.Slug, post.InputHash, outputPath, post.Template)
	}

	return nil
}

// buildFormatContent builds content for a specific output format.
// It checks for per-format template override, and if found, renders using the template.
// Otherwise, falls back to the default content builders for that format.
func (p *PublishHTMLPlugin) buildFormatContent(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager, format string) string {
	// Determine the template to use for this format
	templateName := p.resolveTemplateForFormat(post, format)

	// Check if it's a "raw" template (raw.txt means just output raw content)
	if templateName == rawTxtTemplate || templateName == rawMdTemplate {
		return post.Content
	}

	// Try to get the template engine from cache
	var engine *templates.Engine
	if cached, ok := m.Cache().Get("templates.engine"); ok {
		if e, ok := cached.(*templates.Engine); ok {
			engine = e
		}
	}

	// If we have an engine and the template exists, render it
	if engine != nil && templateName != "" && engine.TemplateExists(templateName) {
		// Create template context
		ctx := templates.NewContext(post, post.Content, ToModelsConfig(config))
		ctx = ctx.WithCore(m)

		// Render the template
		result, err := engine.Render(templateName, ctx)
		if err == nil {
			return result
		}
		// On error, fall through to default content builder
	}

	// Fall back to default content builders
	switch format {
	case formatTxt, formatText:
		return p.buildTextContentFallback(post)
	case formatMarkdown, formatMD:
		return p.buildMarkdownContent(post)
	default:
		return post.Content
	}
}

// resolveTemplateForFormat determines the template to use for a post and output format.
// Uses the TemplatesPlugin if available, otherwise falls back to defaults.
func (p *PublishHTMLPlugin) resolveTemplateForFormat(post *models.Post, format string) string {
	// If we have the templates plugin, use its resolution logic
	if p.templatesPlugin != nil {
		return p.templatesPlugin.resolveTemplateForFormat(post, format)
	}

	// Fallback: check per-format override in frontmatter
	if post.Templates != nil {
		if tmpl, ok := post.Templates[format]; ok && tmpl != "" {
			return tmpl
		}
	}

	// Return empty to use default content builder
	return ""
}

// buildMarkdownContent builds the markdown content with frontmatter for a post.
// Returns the full markdown string with YAML frontmatter.
func (p *PublishHTMLPlugin) buildMarkdownContent(post *models.Post) string {
	// Reconstruct frontmatter
	var buf strings.Builder
	buf.WriteString("---\n")

	if post.Title != nil {
		buf.WriteString(fmt.Sprintf("title: %q\n", *post.Title))
	}
	if post.Description != nil {
		buf.WriteString(fmt.Sprintf("description: %q\n", *post.Description))
	}
	if post.Date != nil {
		buf.WriteString(fmt.Sprintf("date: %s\n", post.Date.Format("2006-01-02")))
	}
	buf.WriteString(fmt.Sprintf("published: %t\n", post.Published))
	if post.Draft {
		buf.WriteString(fmt.Sprintf("draft: %t\n", post.Draft))
	}
	if len(post.Tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range post.Tags {
			buf.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
	}
	if post.Template != "" && post.Template != defaultTemplate {
		buf.WriteString(fmt.Sprintf("template: %s\n", post.Template))
	}

	buf.WriteString("---\n\n")
	buf.WriteString(post.Content)

	return buf.String()
}

// renderTextContent renders plain text content for a post using templates.
// Template resolution order:
// 1. Frontmatter: template field ending in .txt, or templates.txt field
// 2. Config layout settings (if configured)
// 3. Default: "default.txt"
//
// Falls back to hardcoded format if no template engine is available.
func (p *PublishHTMLPlugin) renderTextContent(post *models.Post, config *lifecycle.Config, engine *templates.Engine) string {
	// If no template engine available, use fallback
	if engine == nil {
		return p.buildTextContentFallback(post)
	}

	// Resolve template name for txt format
	templateName := p.resolveTextTemplate(post, engine)

	// Build template context
	ctx := templates.NewContext(post, post.Content, ToModelsConfig(config))

	// Render the template
	result, err := engine.Render(templateName, ctx)
	if err != nil {
		// If template rendering fails, fall back to hardcoded format
		return p.buildTextContentFallback(post)
	}

	return result
}

// resolveTextTemplate determines which template to use for txt output.
// Resolution order:
// 1. Check post frontmatter for template ending in .txt
// 2. Check post.Extra["templates"]["txt"] for format-specific template
// 3. Check if "raw.txt" should be used for special files (robots.txt, etc.)
// 4. Fall back to "default.txt"
func (p *PublishHTMLPlugin) resolveTextTemplate(post *models.Post, engine *templates.Engine) string {
	// 1. Check if post has explicit txt template in frontmatter
	if post.Template != "" && strings.HasSuffix(post.Template, ".txt") {
		if engine.TemplateExists(post.Template) {
			return post.Template
		}
	}

	// 2. Check for templates.txt in Extra (format-specific template)
	if post.Extra != nil {
		if templatesMap, ok := post.Extra["templates"].(map[string]interface{}); ok {
			if txtTemplate, ok := templatesMap["txt"].(string); ok && txtTemplate != "" {
				if engine.TemplateExists(txtTemplate) {
					return txtTemplate
				}
			}
		}
		// Also check for txt_template shorthand
		if txtTemplate, ok := post.Extra["txt_template"].(string); ok && txtTemplate != "" {
			if engine.TemplateExists(txtTemplate) {
				return txtTemplate
			}
		}
	}

	// 3. Check for special files that should use raw.txt template
	// Files like robots.txt, llms.txt, humans.txt typically just need raw content
	if isSpecialFile(post.Slug) {
		if engine.TemplateExists(rawTxtTemplate) {
			return rawTxtTemplate
		}
	}

	// 4. Fall back to default.txt
	if engine.TemplateExists(defaultTxtTemplate) {
		return defaultTxtTemplate
	}

	// If default.txt doesn't exist, try raw.txt
	if engine.TemplateExists(rawTxtTemplate) {
		return rawTxtTemplate
	}

	// Last resort: return default.txt and let it fail gracefully
	return defaultTxtTemplate
}

// buildTextContentFallback builds plain text content without templates.
// This is the fallback when no template engine is available.
// Returns plain text with title, description, date, and content.
// HTML content is converted to plain text with footnote-style link references.
func (p *PublishHTMLPlugin) buildTextContentFallback(post *models.Post) string {
	var buf strings.Builder

	// Write title as heading
	if post.Title != nil {
		buf.WriteString(*post.Title)
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("=", len(*post.Title)))
		buf.WriteString("\n\n")
	}

	// Write description if present
	if post.Description != nil && *post.Description != "" {
		buf.WriteString(*post.Description)
		buf.WriteString("\n\n")
	}

	// Write date if present
	if post.Date != nil {
		buf.WriteString("Date: ")
		buf.WriteString(post.Date.Format("January 2, 2006"))
		buf.WriteString("\n\n")
	}

	// Convert HTML content to plain text with footnote-style links
	buf.WriteString(htmltotext.Convert(post.Content))

	return buf.String()
}

// writeReversedFormatOutput writes content for .txt and .md formats.
//
// For special files (robots, llms, humans, security, ads):
// - /slug.<ext> - actual content (e.g., /robots.txt)
// - /slug/index.<ext> - redirect to /slug.<ext> (e.g., /robots/index.txt -> /robots.txt)
// - /slug/index.html - redirect to /slug.<ext> if HTML is disabled
//
// For regular files:
// - /slug.<ext> - actual content (e.g., /test.txt)
// - /slug/index.<ext> - redirect to /slug.<ext> (e.g., /test/index.txt -> /test.txt)
// - /slug/index.html - redirect to /slug.<ext> if HTML is disabled
//
// If skipSlugRedirect is true, the /slug/index.html redirect is skipped
// (used when HTML format is also enabled, since /slug/index.html has the main HTML content).
//
// Fixes: https://github.com/WaylonWalker/markata-go/issues/465
func (p *PublishHTMLPlugin) writeReversedFormatOutput(slug, ext, content, outputDir string, skipSlugRedirect bool) error {
	// Special files get content at root level (e.g., /robots.txt)
	if isSpecialFile(slug) {
		return p.writeSpecialFileOutput(slug, ext, content, outputDir, skipSlugRedirect)
	}

	// Regular files get content at /slug.<ext> (e.g., /test.txt)
	return p.writeRegularFormatOutput(slug, ext, content, outputDir, skipSlugRedirect)
}

// writeSpecialFileOutput writes output for special files like robots.txt, llms.txt, etc.
// Content is placed at /slug.<ext> with redirects from /slug/index.<ext>.
func (p *PublishHTMLPlugin) writeSpecialFileOutput(slug, ext, content, outputDir string, skipSlugRedirect bool) error {
	// Write actual content at /slug.<ext> (e.g., /robots.txt)
	contentPath := filepath.Join(outputDir, slug+"."+ext)
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(contentPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", contentPath, err)
	}

	// Ensure slug directory exists for redirects
	slugDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(slugDir, 0o755); err != nil {
		return fmt.Errorf("creating slug directory %s: %w", slugDir, err)
	}

	// Create redirect HTML that points to /slug.<ext>
	targetURL := fmt.Sprintf("/%s.%s", slug, ext)
	redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
<title>Redirecting...</title>
</head>
<body>
<p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>`, targetURL, targetURL, targetURL, targetURL)

	// Create HTML redirect at /slug/index.<ext>/index.html (e.g., /robots/index.txt/index.html)
	// This handles requests to /robots/index.txt by serving the redirect HTML
	extRedirectDir := filepath.Join(slugDir, "index."+ext)
	if err := os.MkdirAll(extRedirectDir, 0o755); err != nil {
		return fmt.Errorf("creating format redirect directory %s: %w", extRedirectDir, err)
	}

	extRedirectPath := filepath.Join(extRedirectDir, "index.html")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(extRedirectPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing format redirect %s: %w", extRedirectPath, err)
	}

	// Create HTML redirect at /slug/index.html if HTML is not enabled
	if !skipSlugRedirect {
		htmlRedirectPath := filepath.Join(slugDir, "index.html")
		//nolint:gosec // G306: Output files need 0644 for web serving
		if err := os.WriteFile(htmlRedirectPath, []byte(redirectHTML), 0o644); err != nil {
			return fmt.Errorf("writing HTML redirect %s: %w", htmlRedirectPath, err)
		}
	}

	return nil
}

// writeRegularFormatOutput writes output for regular (non-special) files.
// Content is placed at /slug.<ext> with redirects from /slug.<ext>/index.html.
func (p *PublishHTMLPlugin) writeRegularFormatOutput(slug, ext, content, outputDir string, skipSlugRedirect bool) error {
	// Write actual content at /slug.<ext> (e.g., /test.txt)
	contentPath := filepath.Join(outputDir, slug+"."+ext)
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(contentPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", contentPath, err)
	}

	// Create redirect HTML that points to /slug.<ext>
	targetURL := fmt.Sprintf("/%s.%s", slug, ext)
	redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
<title>Redirecting...</title>
</head>
<body>
<p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>`, targetURL, targetURL, targetURL, targetURL)

	// Create redirect at /slug/index.html if HTML is not enabled
	// (when HTML is enabled, /slug/index.html already has the main HTML content)
	if !skipSlugRedirect {
		slugDir := filepath.Join(outputDir, slug)
		if err := os.MkdirAll(slugDir, 0o755); err != nil {
			return fmt.Errorf("creating slug directory %s: %w", slugDir, err)
		}

		redirectPath := filepath.Join(slugDir, "index.html")
		//nolint:gosec // G306: Output files need 0644 for web serving
		if err := os.WriteFile(redirectPath, []byte(redirectHTML), 0o644); err != nil {
			return fmt.Errorf("writing redirect %s: %w", redirectPath, err)
		}
	}

	// Create redirect at /slug/index.<ext> to /slug.<ext>
	slugExtRedirectDir := filepath.Join(outputDir, slug, "index."+ext)
	if err := os.MkdirAll(slugExtRedirectDir, 0o755); err != nil {
		return fmt.Errorf("creating slug index redirect directory %s: %w", slugExtRedirectDir, err)
	}

	slugExtRedirectPath := filepath.Join(slugExtRedirectDir, "index.html")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(slugExtRedirectPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing slug index redirect %s: %w", slugExtRedirectPath, err)
	}

	return nil
}

// writeOGFormat writes the OpenGraph card HTML for social image generation.
func (p *PublishHTMLPlugin) writeOGFormat(post *models.Post, config *lifecycle.Config, postDir string, engine *templates.Engine) error {
	// Create og subdirectory
	ogDir := filepath.Join(postDir, "og")
	if err := os.MkdirAll(ogDir, 0o755); err != nil {
		return fmt.Errorf("creating og directory %s: %w", ogDir, err)
	}

	// Generate OG HTML
	ogHTML := p.generateOGHTML(post, config, engine)

	// Write og/index.html
	outputPath := filepath.Join(ogDir, "index.html")
	//nolint:gosec // G306: HTML output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(ogHTML), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

// generateOGHTML generates OpenGraph card HTML optimized for 1200x630 screenshots.
// It first tries to use a theme template (og-card.html), falling back to a built-in template.
func (p *PublishHTMLPlugin) generateOGHTML(post *models.Post, config *lifecycle.Config, engine *templates.Engine) string {
	// Try theme template first if engine is available
	if engine != nil && engine.TemplateExists("og-card.html") {
		return p.renderOGWithThemeTemplate(post, config, engine)
	}

	// Fall back to built-in template
	return p.renderOGWithBuiltinTemplate(post, config)
}

// renderOGWithThemeTemplate renders the OG card using the theme's og-card.html template.
func (p *PublishHTMLPlugin) renderOGWithThemeTemplate(post *models.Post, config *lifecycle.Config, engine *templates.Engine) string {
	// Build context for pongo2 template
	ctx := templates.NewContext(post, "", ToModelsConfig(config))

	result, err := engine.Render("og-card.html", ctx)
	if err != nil {
		// Log error and fall back to built-in template
		fmt.Printf("Warning: failed to render og-card.html template: %v, falling back to built-in\n", err)
		return p.renderOGWithBuiltinTemplate(post, config)
	}

	return result
}

// renderOGWithBuiltinTemplate renders the OG card using the built-in Go template.
func (p *PublishHTMLPlugin) renderOGWithBuiltinTemplate(post *models.Post, config *lifecycle.Config) string {
	siteTitle := getSiteTitle(config)
	siteURL := getSiteURL(config)

	title := post.Slug
	if post.Title != nil {
		title = *post.Title
	}

	description := ""
	if post.Description != nil {
		description = *post.Description
	}

	dateStr := ""
	if post.Date != nil {
		dateStr = post.Date.Format("January 2, 2006")
	}

	// Build canonical URL for the original post
	canonicalURL := siteURL + "/" + post.Slug + "/"

	// Built-in OG card template (1200x630 optimized for social images)
	ogTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=1200, height=630">
    <link rel="canonical" href="{{.CanonicalURL}}">
    <meta name="robots" content="noindex, nofollow">
    <title>{{.Title}} - OG Card</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        html, body {
            width: 1200px;
            height: 630px;
            overflow: hidden;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 60px;
        }
        .og-card {
            background: white;
            border-radius: 20px;
            padding: 60px;
            width: 100%;
            height: 100%;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
        }
        .og-content {
            flex: 1;
            display: flex;
            flex-direction: column;
            justify-content: center;
        }
        h1 {
            font-size: 56px;
            font-weight: 700;
            line-height: 1.2;
            color: #1a202c;
            margin-bottom: 24px;
            overflow: hidden;
            display: -webkit-box;
            -webkit-line-clamp: 3;
            -webkit-box-orient: vertical;
        }
        .description {
            font-size: 28px;
            color: #4a5568;
            line-height: 1.5;
            overflow: hidden;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
        }
        .og-footer {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding-top: 30px;
            border-top: 2px solid #e2e8f0;
        }
        .site-name {
            font-size: 24px;
            font-weight: 600;
            color: #667eea;
        }
        .date {
            font-size: 20px;
            color: #718096;
        }
        {{if .Tags}}
        .tags {
            display: flex;
            gap: 12px;
            margin-top: 20px;
            flex-wrap: wrap;
        }
        .tag {
            background: #edf2f7;
            color: #4a5568;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 18px;
        }
        {{end}}
    </style>
</head>
<body>
    <div class="og-card">
        <div class="og-content">
            <h1>{{.Title}}</h1>
            {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
            {{if .Tags}}
            <div class="tags">
                {{range .TagsDisplay}}
                <span class="tag">{{.}}</span>
                {{end}}
            </div>
            {{end}}
        </div>
        <div class="og-footer">
            <span class="site-name">{{.SiteTitle}}</span>
            {{if .DateStr}}<span class="date">{{.DateStr}}</span>{{end}}
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("og").Parse(ogTemplate)
	if err != nil {
		// Return minimal HTML on template error
		return fmt.Sprintf("<html><body><h1>%s</h1></body></html>",
			template.HTMLEscapeString(title))
	}

	// Limit tags to first 3 for display
	tagsDisplay := post.Tags
	if len(tagsDisplay) > 3 {
		tagsDisplay = tagsDisplay[:3]
	}

	data := struct {
		Title        string
		Description  string
		DateStr      string
		Tags         []string
		TagsDisplay  []string
		SiteTitle    string
		CanonicalURL string
	}{
		Title:        title,
		Description:  description,
		DateStr:      dateStr,
		Tags:         post.Tags,
		TagsDisplay:  tagsDisplay,
		SiteTitle:    siteTitle,
		CanonicalURL: canonicalURL,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Return minimal HTML on execution error
		return fmt.Sprintf("<html><body><h1>%s</h1></body></html>",
			template.HTMLEscapeString(title))
	}

	return buf.String()
}

// wrapInTemplate wraps post content in a basic HTML template.
func (p *PublishHTMLPlugin) wrapInTemplate(post *models.Post, config *lifecycle.Config) string {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)

	title := post.Slug
	if post.Title != nil {
		title = *post.Title
	}

	description := ""
	if post.Description != nil {
		description = *post.Description
	}

	dateStr := ""
	dateISO := ""
	if post.Date != nil {
		dateStr = post.Date.Format("January 2, 2006")
		dateISO = post.Date.Format("2006-01-02")
	}

	// Simple default template
	tmplStr := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} | {{.SiteTitle}}</title>
    {{if .Description}}<meta name="description" content="{{.Description}}">{{end}}
    <link rel="canonical" href="{{.SiteURL}}{{.Href}}">
    <link rel="stylesheet" href="/css/variables.css">
  <link rel="stylesheet" href="/css/palette.css">
    <link rel="stylesheet" href="/css/palette.css">
    <link rel="stylesheet" href="/css/main.css">
    <link rel="stylesheet" href="/css/code.css">
    <link rel="stylesheet" href="/css/admonitions.css">
</head>
<body>
    <header>
        <nav>
            <a href="{{.SiteURL}}">{{.SiteTitle}}</a>
        </nav>
    </header>
    <div class="article-progress" role="presentation" aria-hidden="true"><span class="article-progress__indicator"></span></div>
    <main>
        <article class="post">
            <header>
                <h1>{{.Title}}</h1>
                {{if .DateStr}}<time datetime="{{.DateISO}}">{{.DateStr}}</time>{{end}}
            </header>
            <div class="content">
                {{.Content}}
            </div>
            {{if .Tags}}
            <footer>
                <ul class="tags">
                {{range .Tags}}
                    <li><a href="/tags/{{.}}/">{{.}}</a></li>
                {{end}}
                </ul>
            </footer>
            {{end}}
        </article>
    </main>
    <script>
      (function initArticleProgressIndicator() {
        var indicator = document.querySelector('.article-progress__indicator');
        var article = document.querySelector('article.post');
        if (!indicator || !article) {
          return;
        }

        var raf = window.requestAnimationFrame ? window.requestAnimationFrame.bind(window) : function(cb) { return window.setTimeout(cb, 16); };
        var scheduled = false;
        var lastValue = -1;

        function clamp(value, min, max) {
          if (value < min) return min;
          if (value > max) return max;
          return value;
        }

        function refresh() {
          scheduled = false;
          var rect = article.getBoundingClientRect();
          var articleTop = window.scrollY + rect.top;
          var articleHeight = rect.height;
          var viewportHeight = window.innerHeight;
          var scrollableDistance = articleHeight - viewportHeight;
          var ratio;
          if (scrollableDistance <= 0) {
            ratio = 1;
          } else {
            ratio = clamp((window.scrollY - articleTop) / scrollableDistance, 0, 1);
          }
          if (Math.abs(ratio - lastValue) < 0.001) {
            return;
          }
          lastValue = ratio;
          indicator.style.transform = 'scaleX(' + ratio + ')';
        }

        function schedule() {
          if (scheduled) {
            return;
          }
          scheduled = true;
          raf(refresh);
        }

        window.addEventListener('scroll', schedule, { passive: true });
        window.addEventListener('resize', schedule);
        schedule();
      })();
    </script>
    <footer>
        <p><a href="{{.SiteURL}}">{{.SiteTitle}}</a></p>
    </footer>
</body>
</html>`

	tmpl, err := template.New("post").Parse(tmplStr)
	if err != nil {
		// Return basic HTML on template error
		return fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>",
			template.HTMLEscapeString(title), post.ArticleHTML)
	}

	data := struct {
		Title       string
		Description string
		Content     template.HTML
		DateStr     string
		DateISO     string
		Tags        []string
		Href        string
		SiteURL     string
		SiteTitle   string
	}{
		Title:       title,
		Description: description,
		Content:     template.HTML(post.ArticleHTML), //nolint:gosec // G203: ArticleHTML is sanitized markdown output
		DateStr:     dateStr,
		DateISO:     dateISO,
		Tags:        post.Tags,
		Href:        post.Href,
		SiteURL:     siteURL,
		SiteTitle:   siteTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Return basic HTML on execution error
		return fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>",
			template.HTMLEscapeString(title), post.ArticleHTML)
	}

	return buf.String()
}

// Ensure PublishHTMLPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin      = (*PublishHTMLPlugin)(nil)
	_ lifecycle.WritePlugin = (*PublishHTMLPlugin)(nil)
)

// getPostFormatsConfig extracts PostFormatsConfig from lifecycle.Config.Extra.
func getPostFormatsConfig(config *lifecycle.Config) models.PostFormatsConfig {
	if config.Extra != nil {
		if pf, ok := config.Extra["post_formats"].(models.PostFormatsConfig); ok {
			return pf
		}
	}
	return models.NewPostFormatsConfig()
}
