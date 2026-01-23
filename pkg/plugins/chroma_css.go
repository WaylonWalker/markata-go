// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
)

// ChromaCSSPlugin generates CSS for syntax highlighting from Chroma themes.
// It runs during the Write stage and creates css/chroma.css with the
// syntax highlighting styles that correspond to the configured theme.
//
// This plugin works in conjunction with RenderMarkdownPlugin which uses
// CSS classes for syntax highlighting (via WithClasses option).
type ChromaCSSPlugin struct {
	chromaTheme string
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

// Configure reads the highlight theme configuration.
func (p *ChromaCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	extra := config.Extra

	// Try to get explicit highlight config from markdown.highlight.theme
	if markdown, ok := extra["markdown"].(map[string]interface{}); ok {
		if highlight, ok := markdown["highlight"].(map[string]interface{}); ok {
			if theme, ok := highlight["theme"].(string); ok && theme != "" {
				p.chromaTheme = theme
				return nil
			}
		}
	}

	// Derive from palette if not explicitly set
	paletteName := p.getPaletteName(extra)
	if paletteName != "" {
		chromaTheme := palettes.ChromaTheme(paletteName)
		if chromaTheme != "" {
			p.chromaTheme = chromaTheme
			return nil
		}

		// Fallback based on variant
		variant := p.getPaletteVariant(paletteName)
		p.chromaTheme = palettes.ChromaThemeForVariant(variant)
	}

	return nil
}

// Write generates the Chroma CSS file.
func (p *ChromaCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Get the style
	style := styles.Get(p.chromaTheme)
	if style == nil {
		style = styles.Fallback
	}

	// Generate CSS using Chroma's formatter
	css, err := p.generateCSS(style)
	if err != nil {
		return fmt.Errorf("generating chroma CSS: %w", err)
	}

	// Write to output directory
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}

	cssPath := filepath.Join(cssDir, "chroma.css")
	//nolint:gosec // G306: chroma.css is a public CSS file, 0644 is appropriate
	if err := os.WriteFile(cssPath, []byte(css), 0o644); err != nil {
		return fmt.Errorf("writing chroma CSS: %w", err)
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
