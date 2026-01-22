// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
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
//
// The initial configuration uses a default theme. The Configure() method will
// reconfigure the markdown renderer with the appropriate theme based on the
// site's palette configuration.
func NewRenderMarkdownPlugin() *RenderMarkdownPlugin {
	return &RenderMarkdownPlugin{
		md: createMarkdownRenderer(palettes.DefaultChromaThemeDark, false),
	}
}

// createMarkdownRenderer creates a goldmark instance with the specified highlighting options.
func createMarkdownRenderer(chromaTheme string, _ bool) goldmark.Markdown {
	highlightOpts := []highlighting.Option{
		highlighting.WithStyle(chromaTheme),
		highlighting.WithFormatOptions(),
	}

	return goldmark.New(
		goldmark.WithExtensions(
			// GFM extensions
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.TaskList,
			// Syntax highlighting with chroma
			highlighting.NewHighlighting(highlightOpts...),
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
}

// Name returns the unique name of the plugin.
func (p *RenderMarkdownPlugin) Name() string {
	return "render_markdown"
}

// Configure initializes the plugin during the configure stage.
// It reads the highlight theme from configuration, falling back to the theme
// derived from the site's color palette if not explicitly set.
//
// Configuration priority:
// 1. markdown.highlight.theme in config (explicit override)
// 2. Theme derived from theme.palette (automatic matching)
// 3. Default theme based on palette variant (github/github-dark)
func (p *RenderMarkdownPlugin) Configure(m *lifecycle.Manager) error {
	chromaTheme, lineNumbers := p.resolveHighlightConfig(m.Config().Extra)

	// Reconfigure the markdown renderer with the resolved theme
	p.md = createMarkdownRenderer(chromaTheme, lineNumbers)

	return nil
}

// resolveHighlightConfig extracts highlight configuration from the config.Extra map.
// Returns the Chroma theme name and whether line numbers should be shown.
func (p *RenderMarkdownPlugin) resolveHighlightConfig(extra map[string]interface{}) (string, bool) {
	var chromaTheme string
	var lineNumbers bool
	var paletteVariant = palettes.VariantDark

	// Try to get explicit highlight config from markdown.highlight
	if markdown, ok := extra["markdown"].(map[string]interface{}); ok {
		if highlight, ok := markdown["highlight"].(map[string]interface{}); ok {
			// Check if highlighting is disabled
			if enabled, ok := highlight["enabled"].(bool); ok && !enabled {
				// Return empty theme to effectively disable highlighting
				return "monokailight", false // Use a neutral theme, highlighting is still applied
			}

			// Get explicit theme
			if theme, ok := highlight["theme"].(string); ok && theme != "" {
				chromaTheme = theme
			}

			// Get line numbers setting
			if ln, ok := highlight["line_numbers"].(bool); ok {
				lineNumbers = ln
			}
		}
	}

	// If no explicit theme, derive from palette
	if chromaTheme == "" {
		paletteName := p.getPaletteName(extra)
		if paletteName != "" {
			// Try to get theme from palette mapping
			chromaTheme = palettes.ChromaTheme(paletteName)

			// Determine palette variant for fallback
			paletteVariant = p.getPaletteVariant(paletteName)
		}
	}

	// Final fallback based on palette variant
	if chromaTheme == "" {
		chromaTheme = palettes.ChromaThemeForVariant(paletteVariant)
	}

	return chromaTheme, lineNumbers
}

// getPaletteName extracts the palette name from config.Extra.
func (p *RenderMarkdownPlugin) getPaletteName(extra map[string]interface{}) string {
	if theme, ok := extra["theme"].(map[string]interface{}); ok {
		if palette, ok := theme["palette"].(string); ok && palette != "" {
			return palette
		}
	}
	return ""
}

// getPaletteVariant determines the variant (light/dark) of a palette by name.
// Uses naming conventions to infer variant when palette data isn't loaded.
func (p *RenderMarkdownPlugin) getPaletteVariant(paletteName string) palettes.Variant {
	// Check common light palette name patterns
	lightPatterns := []string{
		"-light", "-latte", "-dawn", "-day", "-lotus",
	}
	for _, pattern := range lightPatterns {
		if strings.Contains(paletteName, pattern) {
			return palettes.VariantLight
		}
	}
	return palettes.VariantDark
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
