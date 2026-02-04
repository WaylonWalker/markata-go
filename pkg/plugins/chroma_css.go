// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// ChromaCSSPlugin generates CSS for syntax highlighting from Chroma themes.
// It runs during the Write stage and creates css/chroma.css with the
// syntax highlighting styles that correspond to the configured theme.
//
// This plugin works in conjunction with RenderMarkdownPlugin which uses
// CSS classes for syntax highlighting (via WithClasses option).
type ChromaCSSPlugin struct {
	chromaTheme string
	chromaCSS   string // Pre-generated CSS content (computed in Configure)
	chromaHash  string // Content hash for cache busting
}

// NewChromaCSSPlugin creates a new ChromaCSSPlugin.
func NewChromaCSSPlugin() *ChromaCSSPlugin {
	return &ChromaCSSPlugin{
		chromaTheme: palettes.DefaultChromaThemeDark,
	}
}

// Name returns the unique name of the plugin.
func (p *ChromaCSSPlugin) Name() string {
	return "chroma_css"
}

// Configure reads the highlight theme configuration and pre-generates the CSS.
// This runs before templates are rendered, allowing us to register the hash
// for cache busting (css/chroma.css -> css/chroma.abc12345.css).
func (p *ChromaCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	extra := config.Extra

	configuredTheme := ""

	// Try to get explicit highlight config from markdown.highlight.theme
	if markdown, ok := extra["markdown"].(map[string]interface{}); ok {
		if highlight, ok := markdown["highlight"].(map[string]interface{}); ok {
			if theme, ok := highlight["theme"].(string); ok && theme != "" {
				configuredTheme = theme
			}
		}
	}

	// Derive from palette if not explicitly set
	if configuredTheme == "" {
		paletteName := p.getPaletteName(extra)
		if paletteName != "" {
			chromaTheme := palettes.ChromaTheme(paletteName)
			if chromaTheme != "" {
				configuredTheme = chromaTheme
			} else {
				// Fallback based on variant
				variant := p.getPaletteVariant(paletteName)
				configuredTheme = palettes.ChromaThemeForVariant(variant)
			}
		}
	}

	// Use configured theme if found, otherwise keep default
	if configuredTheme != "" {
		p.chromaTheme = configuredTheme
	}

	// Pre-generate the CSS content so we can hash it for cache busting
	style := styles.Get(p.chromaTheme)
	if style == nil {
		style = styles.Fallback
	}

	css, err := p.generateCSS(style)
	if err != nil {
		return fmt.Errorf("generating chroma CSS: %w", err)
	}
	p.chromaCSS = css

	// Compute hash for cache busting (first 8 chars of SHA-256)
	hash := sha256.Sum256([]byte(css))
	p.chromaHash = fmt.Sprintf("%x", hash[:4]) // 4 bytes = 8 hex chars

	// Register the hash with the template engine for theme_asset_hashed filter
	assetHashes := map[string]string{
		"css/chroma.css": p.chromaHash,
	}
	templates.SetAssetHashes(assetHashes)

	// Also register with the lifecycle manager for tracking
	m.SetAssetHash("css/chroma.css", p.chromaHash)

	return nil
}

// Write generates the Chroma CSS file.
// The CSS content was already generated in Configure() for cache busting.
func (p *ChromaCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// CSS was pre-generated in Configure for hash computation
	css := p.chromaCSS
	if css == "" {
		// Fallback: generate now if Configure didn't run (shouldn't happen)
		style := styles.Get(p.chromaTheme)
		if style == nil {
			style = styles.Fallback
		}
		var err error
		css, err = p.generateCSS(style)
		if err != nil {
			return fmt.Errorf("generating chroma CSS: %w", err)
		}
	}

	// Write to output directory
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}

	// Write original filename (css/chroma.css)
	cssPath := filepath.Join(cssDir, "chroma.css")
	//nolint:gosec // G306: chroma.css is a public CSS file, 0644 is appropriate
	if err := os.WriteFile(cssPath, []byte(css), 0o644); err != nil {
		return fmt.Errorf("writing chroma CSS: %w", err)
	}

	// Write hashed version (css/chroma.abc12345.css) for cache busting
	if p.chromaHash != "" {
		hashedFilename := fmt.Sprintf("chroma.%s.css", p.chromaHash)
		hashedPath := filepath.Join(cssDir, hashedFilename)
		//nolint:gosec // G306: chroma CSS is a public file, 0644 is appropriate
		if err := os.WriteFile(hashedPath, []byte(css), 0o644); err != nil {
			return fmt.Errorf("writing hashed chroma CSS: %w", err)
		}
	}

	return nil
}

// generateCSS creates CSS from a Chroma style.
func (p *ChromaCSSPlugin) generateCSS(style *chroma.Style) (string, error) {
	formatter := chromahtml.New(chromahtml.WithClasses(true), chromahtml.WithAllClasses(true))

	var sb strings.Builder
	sb.WriteString("/* Syntax highlighting - generated from Chroma theme: ")
	sb.WriteString(p.chromaTheme)
	sb.WriteString(" */\n\n")

	// Write the CSS for the style
	if err := formatter.WriteCSS(&sb, style); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// getPaletteName extracts the palette name from config.Extra.
func (p *ChromaCSSPlugin) getPaletteName(extra map[string]interface{}) string {
	if extra == nil {
		return ""
	}

	if theme, ok := extra["theme"].(map[string]interface{}); ok {
		if palette, ok := theme["palette"].(string); ok && palette != "" {
			return palette
		}
	}

	return ""
}

// getPaletteVariant determines the variant (light/dark) of a palette by name.
func (p *ChromaCSSPlugin) getPaletteVariant(paletteName string) palettes.Variant {
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

// Priority returns the plugin priority for the write stage.
// Should run after static_assets so it can add to the css directory.
func (p *ChromaCSSPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityDefault // After static_assets (PriorityEarly)
	}
	return lifecycle.PriorityDefault
}

// Ensure ChromaCSSPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*ChromaCSSPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*ChromaCSSPlugin)(nil)
	_ lifecycle.WritePlugin     = (*ChromaCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*ChromaCSSPlugin)(nil)
)
