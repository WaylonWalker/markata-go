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
)

// defaultTemplate is the default template name for posts.
const defaultTemplate = "post.html"

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

	// Process posts concurrently
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		return p.writePost(post, config)
	})
}

// writePost writes a single post to its output location in all enabled formats.
// Shadow pages: Unpublished posts are still rendered but not included in feeds.
// This allows sharing draft content via direct URL while keeping it out of public listings.
func (p *PublishHTMLPlugin) writePost(post *models.Post, config *lifecycle.Config) error {
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
	if postFormats.Markdown {
		if err := p.writeMarkdownFormat(post, postDir); err != nil {
			return err
		}
		// Write redirect from /slug.md to /slug/index.md
		if err := p.writeFormatRedirect(post.Slug, "md", config.OutputDir); err != nil {
			return err
		}
	}

	// Write Text format (plain text)
	if postFormats.Text {
		if err := p.writeTextFormat(post, postDir); err != nil {
			return err
		}
		// Write redirect from /slug.txt to /slug/index.txt
		if err := p.writeFormatRedirect(post.Slug, "txt", config.OutputDir); err != nil {
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

// writeMarkdownFormat writes the raw markdown source with frontmatter.
func (p *PublishHTMLPlugin) writeMarkdownFormat(post *models.Post, postDir string) error {
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

	// Write index.md
	outputPath := filepath.Join(postDir, "index.md")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

// writeTextFormat writes the plain text version of a post.
// This outputs just the content without frontmatter or formatting.
func (p *PublishHTMLPlugin) writeTextFormat(post *models.Post, postDir string) error {
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

	// Write index.txt
	outputPath := filepath.Join(postDir, "index.txt")
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

// writeFormatRedirect writes a redirect file from /slug.ext to /slug/index.ext.
// This allows cleaner URLs like /my-post.md instead of /my-post/index.md.
// The redirect uses HTTP meta refresh for maximum compatibility.
func (p *PublishHTMLPlugin) writeFormatRedirect(slug, ext, outputDir string) error {
	// Create redirect HTML that points to the actual file
	targetURL := fmt.Sprintf("/%s/index.%s", slug, ext)
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

	// Write the redirect file at /slug.ext
	outputPath := filepath.Join(outputDir, slug+"."+ext)
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(outputPath, []byte(redirectHTML), 0o644); err != nil {
		return fmt.Errorf("writing redirect %s: %w", outputPath, err)
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
