// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// defaultTemplate is the default template name for posts.
const defaultTemplate = "post.html"

// defaultTxtTemplate is the default template name for txt output.
const defaultTxtTemplate = "default.txt"

// PublishHTMLPlugin writes individual post HTML files during the write stage.
// It supports multiple output formats: HTML, Markdown source, and OG card HTML.
type PublishHTMLPlugin struct{}

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

	// Process posts concurrently
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		return p.writePost(post, config, engine)
	})
}

// writePost writes a single post to its output location in all enabled formats.
// Shadow pages: Unpublished posts are still rendered but not included in feeds.
// This allows sharing draft content via direct URL while keeping it out of public listings.
func (p *PublishHTMLPlugin) writePost(post *models.Post, config *lifecycle.Config, engine *templates.Engine) error {
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
	// Use the slug to create: output_dir/slug/index.html
	if post.Slug == "" {
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

	// Write HTML format (default)
	if postFormats.IsHTMLEnabled() {
		if err := p.writeHTMLFormat(post, config, postDir); err != nil {
			return err
		}
	}

	// Write Markdown format (raw source)
	// Uses reversed redirect: content at /slug.md, redirect at /slug/index.md
	if postFormats.Markdown {
		mdContent := p.buildMarkdownContent(post)
		if err := p.writeReversedFormatOutput(post.Slug, "md", mdContent, config.OutputDir); err != nil {
			return err
		}
	}

	// Write Text format (plain text)
	// Uses reversed redirect: content at /slug.txt, redirect at /slug/index.txt
	if postFormats.Text {
		txtContent := p.renderTextContent(post, config, engine)
		if err := p.writeReversedFormatOutput(post.Slug, "txt", txtContent, config.OutputDir); err != nil {
			return err
		}
	}

	// Write OG format (social card HTML)
	if postFormats.OG {
		if err := p.writeOGFormat(post, config, postDir); err != nil {
			return err
		}
	}

	return nil
}

// writeHTMLFormat writes the standard HTML output for a post.
func (p *PublishHTMLPlugin) writeHTMLFormat(post *models.Post, config *lifecycle.Config, postDir string) error {
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

	return nil
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
	ctx := templates.NewContext(post, post.Content, toModelsConfig(config))

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
	specialFiles := []string{"robots", "llms", "humans", "security", "ads"}
	for _, special := range specialFiles {
		if post.Slug == special {
			if engine.TemplateExists("raw.txt") {
				return "raw.txt"
			}
			break
		}
	}

	// 4. Fall back to default.txt
	if engine.TemplateExists(defaultTxtTemplate) {
		return defaultTxtTemplate
	}

	// If default.txt doesn't exist, try raw.txt
	if engine.TemplateExists("raw.txt") {
		return "raw.txt"
	}

	// Last resort: return default.txt and let it fail gracefully
	return defaultTxtTemplate
}

// buildTextContentFallback builds plain text content without templates.
// This is the fallback when no template engine is available.
// Returns plain text with title, description, date, and content.
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

	// Write the raw content (without markdown processing)
	buf.WriteString(post.Content)

	return buf.String()
}

// writeReversedFormatOutput writes the primary content at /slug.ext (canonical URL)
// and creates a backwards-compatible redirect at /slug/index.ext.
//
// This is the REVERSED approach for txt/md files:
// - Canonical content: /robots.txt, /llms.txt (standard web file locations)
// - Redirect: /robots/index.txt → /robots.txt (backwards compatibility)
//
// This allows standard web txt files (robots.txt, llms.txt, humans.txt) to be
// served at their expected locations while maintaining the directory-based
// URL structure for backward compatibility.
func (p *PublishHTMLPlugin) writeReversedFormatOutput(slug, ext, content, outputDir string) error {
	// Write primary content at /slug.ext (e.g., /robots.txt)
	primaryPath := filepath.Join(outputDir, slug+"."+ext)
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(primaryPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", primaryPath, err)
	}

	// Create redirect HTML that points from /slug/index.ext to /slug.ext
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

	// Create directory at /slug/ for the redirect (e.g., /robots/)
	redirectDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(redirectDir, 0o755); err != nil {
		return fmt.Errorf("creating redirect directory %s: %w", redirectDir, err)
	}

	// Write index.ext redirect inside the directory (e.g., /robots/index.txt)
	// This redirects to the canonical /slug.ext location
	redirectPath := filepath.Join(redirectDir, "index."+ext)
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(redirectPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing redirect %s: %w", redirectPath, err)
	}

	return nil
}

// writeOGFormat writes the OpenGraph card HTML for social image generation.
func (p *PublishHTMLPlugin) writeOGFormat(post *models.Post, config *lifecycle.Config, postDir string) error {
	// Create og subdirectory
	ogDir := filepath.Join(postDir, "og")
	if err := os.MkdirAll(ogDir, 0o755); err != nil {
		return fmt.Errorf("creating og directory %s: %w", ogDir, err)
	}

	// Generate OG HTML
	ogHTML := p.generateOGHTML(post, config)

	// Write og/index.html
	outputPath := filepath.Join(ogDir, "index.html")
	//nolint:gosec // G306: HTML output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(ogHTML), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

// generateOGHTML generates OpenGraph card HTML optimized for 1200x630 screenshots.
func (p *PublishHTMLPlugin) generateOGHTML(post *models.Post, config *lifecycle.Config) string {
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
    <main>
        <article>
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
