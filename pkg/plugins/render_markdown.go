// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// RenderMarkdownPlugin converts markdown content to HTML using goldmark.
type RenderMarkdownPlugin struct {
	md goldmark.Markdown
}

// NewRenderMarkdownPlugin creates a new RenderMarkdownPlugin with goldmark configured.
// The goldmark instance is configured with:
// - GFM extensions (tables, strikethrough, autolinks, task lists)
// - Syntax highlighting using chroma
// - HTML rendering with unsafe mode (allow raw HTML)
// - Auto heading IDs
// - Custom admonition support
func NewRenderMarkdownPlugin() *RenderMarkdownPlugin {
	md := goldmark.New(
		goldmark.WithExtensions(
			// GFM extensions
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.TaskList,
			// Syntax highlighting with chroma
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(),
			),
			// Custom admonition extension
			&AdmonitionExtension{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			// Allow raw HTML in markdown
			html.WithUnsafe(),
		),
	)

	return &RenderMarkdownPlugin{
		md: md,
	}
}

// Name returns the unique name of the plugin.
func (p *RenderMarkdownPlugin) Name() string {
	return "render_markdown"
}

// Configure initializes the plugin during the configure stage.
// This can be used to apply configuration options to the markdown renderer.
func (p *RenderMarkdownPlugin) Configure(m *lifecycle.Manager) error {
	// Configuration could be read from m.Config().Extra if needed
	// For example, to change the highlighting style or enable/disable extensions
	return nil
}

// Render converts markdown content to HTML for all posts.
// Posts with Skip=true are skipped.
// The rendered HTML is stored in post.ArticleHTML.
func (p *RenderMarkdownPlugin) Render(m *lifecycle.Manager) error {
	return m.ProcessPostsConcurrently(p.renderPost)
}

// renderPost renders a single post's markdown content to HTML.
func (p *RenderMarkdownPlugin) renderPost(post *models.Post) error {
	// Skip posts marked as skip
	if post.Skip {
		return nil
	}

	// Skip posts with no content
	if post.Content == "" {
		post.ArticleHTML = ""
		return nil
	}

	// Convert markdown to HTML
	var buf bytes.Buffer
	if err := p.md.Convert([]byte(post.Content), &buf); err != nil {
		return err
	}

	post.ArticleHTML = buf.String()
	return nil
}

// Ensure RenderMarkdownPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*RenderMarkdownPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*RenderMarkdownPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*RenderMarkdownPlugin)(nil)
)
