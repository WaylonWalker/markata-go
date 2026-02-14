// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"strings"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	figure "github.com/mangoumbrella/goldmark-figure"
	"go.abhg.dev/goldmark/anchor"
)

// markdownBufferPool is a sync.Pool for reusing bytes.Buffer instances
// during markdown rendering. This significantly reduces allocations and
// GC pressure when processing many posts.
var markdownBufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate 32KB buffer - typical blog post is 5-20KB HTML
		buf := bytes.NewBuffer(make([]byte, 0, 32*1024))
		return buf
	},
}

// RenderMarkdownPlugin converts markdown content to HTML using goldmark.
type RenderMarkdownPlugin struct {
	md    goldmark.Markdown
	cache *buildcache.Cache // build cache for HTML caching
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
		md: createMarkdownRenderer(palettes.DefaultChromaThemeDark, false, DefaultMarkdownExtensionConfig()),
	}
}

// MarkdownExtensionConfig holds configuration for optional markdown extensions.
type MarkdownExtensionConfig struct {
	TypographerEnabled       bool
	DefinitionListEnabled    bool
	FootnoteEnabled          bool
	CJKEnabled               bool
	FigureEnabled            bool
	AnchorEnabled            bool
	TypographerSubstitutions map[extension.TypographicPunctuation]string
}

// DefaultMarkdownExtensionConfig returns the default configuration with all extensions enabled.
func DefaultMarkdownExtensionConfig() MarkdownExtensionConfig {
	return MarkdownExtensionConfig{
		TypographerEnabled:       true,
		DefinitionListEnabled:    true,
		FootnoteEnabled:          true,
		CJKEnabled:               true,
		FigureEnabled:            true,
		AnchorEnabled:            true,
		TypographerSubstitutions: nil, // nil means use goldmark defaults
	}
}

// createMarkdownRenderer creates a goldmark instance with the specified highlighting options.
func createMarkdownRenderer(chromaTheme string, lineNumbers bool, extConfig MarkdownExtensionConfig) goldmark.Markdown {
	// Use CSS classes instead of inline styles for syntax highlighting.
	// This enables theme customization via external CSS files.
	formatOptions := []chromahtml.Option{
		chromahtml.WithClasses(true),
		chromahtml.WithAllClasses(true),
	}

	if lineNumbers {
		formatOptions = append(formatOptions, chromahtml.WithLineNumbers(true))
	}

	highlightOpts := []highlighting.Option{
		highlighting.WithStyle(chromaTheme),
		highlighting.WithFormatOptions(formatOptions...),
	}

	extensions := []goldmark.Extender{
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
		// Mark extension for ==highlighted text==
		&MarkExtension{},
		// Keys extension for ++Ctrl+Alt+Del++
		&KeysExtension{},
		// Container extension for ::: class
		&ContainerExtension{},
		// Emoji extension for :smile: syntax
		emoji.Emoji,
	}

	// Add CJK extension (Chinese/Japanese/Korean line break support)
	if extConfig.CJKEnabled {
		extensions = append(extensions, extension.NewCJK())
	}

	// Add Figure extension for <figure> elements from images with captions
	if extConfig.FigureEnabled {
		extensions = append(extensions, figure.Figure)
	}

	// Add Anchor extension for heading permalinks
	if extConfig.AnchorEnabled {
		extensions = append(extensions, &anchor.Extender{})
	}

	// Add Typographer extension (smart quotes, dashes, ellipses)
	if extConfig.TypographerEnabled {
		if extConfig.TypographerSubstitutions != nil {
			extensions = append(extensions, extension.NewTypographer(
				extension.WithTypographicSubstitutions(extConfig.TypographerSubstitutions),
			))
		} else {
			extensions = append(extensions, extension.Typographer)
		}
	}

	// Add DefinitionList extension (PHP Markdown Extra style definition lists)
	if extConfig.DefinitionListEnabled {
		extensions = append(extensions, extension.DefinitionList)
	}

	// Add Footnote extension (PHP Markdown Extra style footnotes)
	if extConfig.FootnoteEnabled {
		extensions = append(extensions, extension.Footnote)
	}

	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			// Enable attribute syntax for {.class}, {#id}, {key=value} on block elements
			parser.WithAttribute(),
			// Enable inline attribute syntax for images and links
			parser.WithASTTransformers(
				util.Prioritized(&InlineAttributeTransformer{}, 100),
			),
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
//
// Markdown extension configuration:
// 1. markdown.extensions.typographer - Enable smart quotes (default: true)
// 2. markdown.extensions.definition_list - Enable definition lists (default: true)
// 3. markdown.extensions.footnote - Enable footnotes (default: true)
// 4. markdown.extensions.cjk - Enable CJK line breaks (default: true)
// 5. markdown.extensions.figure - Enable figure from images with captions (default: true)
// 6. markdown.extensions.anchor - Enable heading permalinks (default: true)
func (p *RenderMarkdownPlugin) Configure(m *lifecycle.Manager) error {
	chromaTheme, lineNumbers := p.resolveHighlightConfig(m.Config().Extra)
	extConfig := p.resolveExtensionConfig(m.Config().Extra)

	// Reconfigure the markdown renderer with the resolved theme and extensions
	p.md = createMarkdownRenderer(chromaTheme, lineNumbers, extConfig)

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

// resolveExtensionConfig extracts markdown extension configuration from config.Extra map.
// Returns a MarkdownExtensionConfig with settings for typographer, definition lists, and footnotes.
// All extensions are enabled by default unless explicitly disabled in config.
func (p *RenderMarkdownPlugin) resolveExtensionConfig(extra map[string]interface{}) MarkdownExtensionConfig {
	config := DefaultMarkdownExtensionConfig()

	// Try to get extension config from markdown.extensions
	if markdown, ok := extra["markdown"].(map[string]interface{}); ok {
		if extensions, ok := markdown["extensions"].(map[string]interface{}); ok {
			// Typographer (smart quotes)
			if enabled, ok := extensions["typographer"].(bool); ok {
				config.TypographerEnabled = enabled
			}

			// Definition lists
			if enabled, ok := extensions["definition_list"].(bool); ok {
				config.DefinitionListEnabled = enabled
			}

			// Footnotes
			if enabled, ok := extensions["footnote"].(bool); ok {
				config.FootnoteEnabled = enabled
			}

			// CJK (Chinese/Japanese/Korean line breaks)
			if enabled, ok := extensions["cjk"].(bool); ok {
				config.CJKEnabled = enabled
			}

			// Figure (images with captions to <figure>)
			if enabled, ok := extensions["figure"].(bool); ok {
				config.FigureEnabled = enabled
			}

			// Anchor (heading permalinks)
			if enabled, ok := extensions["anchor"].(bool); ok {
				config.AnchorEnabled = enabled
			}
		}
	}

	return config
}

// Render converts markdown content to HTML for all posts.
// Posts with Skip=true are skipped.
// The rendered HTML is stored in post.ArticleHTML.
// Uses build cache to skip re-rendering unchanged content.
//
// Uses two-phase processing for incremental optimization:
// Phase 1: Quick single-threaded pass to restore cached HTML (no worker overhead)
// Phase 2: Concurrent processing only for posts that need rendering
func (p *RenderMarkdownPlugin) Render(m *lifecycle.Manager) error {
	// Get build cache for HTML caching
	p.cache = GetBuildCache(m)

	// Phase 1: Pre-filter posts and restore cached HTML
	// This avoids worker pool overhead for the ~98% of posts that are cached
	postsNeedingRender := m.FilterPosts(func(post *models.Post) bool {
		// Skip posts marked as skip
		if post.Skip {
			return false
		}

		// Skip posts with no content
		if post.Content == "" {
			post.ArticleHTML = ""
			return false
		}

		// Try to get cached HTML if content hasn't changed
		if p.cache != nil {
			contentHash := buildcache.ContentHash(post.Content)
			if cachedHTML := p.cache.GetCachedArticleHTML(post.Path, contentHash); cachedHTML != "" {
				post.ArticleHTML = cachedHTML
				// Detect CSS requirements from cached HTML
				p.detectCSSRequirements(post)
				return false // Already handled, no concurrent processing needed
			}
		}

		return true // Needs rendering
	})

	// Phase 2: Process only posts that need rendering concurrently
	return m.ProcessPostsSliceConcurrently(postsNeedingRender, p.renderPost)
}

// renderPost renders a single post's markdown content to HTML.
// Uses a buffer pool to reduce allocations and GC pressure.
// Note: Cache lookup is now done in the filter phase, so posts reaching here need rendering.
func (p *RenderMarkdownPlugin) renderPost(post *models.Post) error {
	// Skip posts marked as skip (defensive check, filter should catch this)
	if post.Skip {
		return nil
	}

	// Skip posts with no content (defensive check, filter should catch this)
	if post.Content == "" {
		post.ArticleHTML = ""
		return nil
	}

	// Render the markdown
	renderedHTML, err := p.doRender(post.Content)
	if err != nil {
		return err
	}
	post.ArticleHTML = renderedHTML

	// Cache the result for future incremental builds
	if p.cache != nil {
		contentHash := buildcache.ContentHash(post.Content)
		//nolint:errcheck // caching is best-effort, failures are non-fatal
		p.cache.CacheArticleHTML(post.Path, contentHash, renderedHTML)
	}

	// Detect CSS requirements from rendered HTML
	p.detectCSSRequirements(post)
	return nil
}

// doRender performs the actual markdown to HTML conversion.
func (p *RenderMarkdownPlugin) doRender(content string) (string, error) {
	// Get buffer from pool
	buf, ok := markdownBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buf = new(bytes.Buffer)
	}
	buf.Reset()
	defer markdownBufferPool.Put(buf)

	// Convert markdown to HTML
	if err := p.md.Convert([]byte(content), buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// detectCSSRequirements scans the rendered HTML and sets flags in post.Extra
// to indicate which CSS files are needed. This follows the ImageZoom pattern
// for conditional asset loading.
func (p *RenderMarkdownPlugin) detectCSSRequirements(post *models.Post) {
	if post.ArticleHTML == "" {
		return
	}

	// Initialize Extra map if needed
	if post.Extra == nil {
		post.Extra = make(map[string]interface{})
	}

	// Detect admonitions - check for admonition class
	if strings.Contains(post.ArticleHTML, `class="admonition`) {
		post.Extra["needs_admonitions_css"] = true
	}

	// Detect code blocks - check for chroma syntax highlighting classes or code/pre tags
	if strings.Contains(post.ArticleHTML, `class="chroma"`) ||
		strings.Contains(post.ArticleHTML, `class="highlight"`) ||
		strings.Contains(post.ArticleHTML, "<pre><code") ||
		strings.Contains(post.ArticleHTML, `<code class="language-`) {
		post.Extra["needs_code_css"] = true
	}

	// Detect images that will get GLightbox treatment (image_zoom plugin
	// runs after this, but we can detect existing glightbox markers or
	// images that will be processed).
	if strings.Contains(post.ArticleHTML, `class="glightbox"`) ||
		strings.Contains(post.ArticleHTML, `data-glightbox`) ||
		strings.Contains(post.ArticleHTML, `{data-zoomable}`) ||
		strings.Contains(post.ArticleHTML, `{.zoomable}`) {
		post.Extra["needs_image_zoom"] = true
	}

	// Detect Mermaid diagrams - check for mermaid class or pre tags
	if strings.Contains(post.ArticleHTML, `class="mermaid"`) ||
		strings.Contains(post.ArticleHTML, `<pre class="mermaid"`) ||
		strings.Contains(post.ArticleHTML, `<div class="mermaid"`) {
		post.Extra["has_mermaid"] = true
	}

	// Detect figure elements - check for figure tag (goldmark-figure extension)
	if strings.Contains(post.ArticleHTML, "<figure") {
		post.Extra["has_figure"] = true
	}

	// Detect anchor links - check for anchor class (goldmark-anchor extension)
	if strings.Contains(post.ArticleHTML, `class="anchor"`) {
		post.Extra["has_anchor_links"] = true
	}
}

// Ensure RenderMarkdownPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*RenderMarkdownPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*RenderMarkdownPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*RenderMarkdownPlugin)(nil)
)
