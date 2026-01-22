// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// PublishHTMLPlugin writes individual post HTML files during the write stage.
type PublishHTMLPlugin struct{}

// NewPublishHTMLPlugin creates a new PublishHTMLPlugin.
func NewPublishHTMLPlugin() *PublishHTMLPlugin {
	return &PublishHTMLPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PublishHTMLPlugin) Name() string {
	return "publish_html"
}

// Write outputs each post's HTML to the configured output directory.
func (p *PublishHTMLPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Process posts concurrently
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		return p.writePost(post, config)
	})
}

// writePost writes a single post to its output location.
func (p *PublishHTMLPlugin) writePost(post *models.Post, config *lifecycle.Config) error {
	// Skip posts marked as skip
	if post.Skip {
		return nil
	}

	// Skip unpublished posts
	if !post.Published {
		return nil
	}

	// Skip drafts
	if post.Draft {
		return nil
	}

	// Determine output path
	// Use the slug to create: output_dir/slug/index.html
	if post.Slug == "" {
		post.GenerateSlug()
	}

	outputDir := config.OutputDir
	postDir := filepath.Join(outputDir, post.Slug)

	// Create post directory
	if err := os.MkdirAll(postDir, 0755); err != nil {
		return fmt.Errorf("creating post directory %s: %w", postDir, err)
	}

	// Determine HTML content to write
	var htmlContent string
	if post.HTML != "" {
		// Use pre-rendered HTML if available
		htmlContent = post.HTML
	} else if post.ArticleHTML != "" {
		// Wrap ArticleHTML in a basic template
		htmlContent = p.wrapInTemplate(post, config)
	} else {
		// No HTML content available
		return nil
	}

	// Write index.html
	outputPath := filepath.Join(postDir, "index.html")
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
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
		Content:     template.HTML(post.ArticleHTML),
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
