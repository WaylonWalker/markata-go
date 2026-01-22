package plugins

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/palettes"
)

// PaletteCSSPlugin generates CSS variables from the configured color palette.
// It runs during the Write stage and creates/overwrites css/variables.css
// with the palette's CSS custom properties. It runs after static_assets
// to overwrite the default variables.css with palette-specific values.
//
// The plugin maps palette colors to the theme's expected CSS variable names,
// preserving fonts, spacing, and other non-color variables from the default theme.
type PaletteCSSPlugin struct{}

// NewPaletteCSSPlugin creates a new PaletteCSSPlugin.
func NewPaletteCSSPlugin() *PaletteCSSPlugin {
	return &PaletteCSSPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PaletteCSSPlugin) Name() string {
	return "palette_css"
}

// Write generates CSS from the configured palette and writes it to the output directory.
func (p *PaletteCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Get palette name from config.Extra["theme"]["palette"]
	paletteName := p.getPaletteName(config.Extra)
	if paletteName == "" {
		// No palette configured, skip
		return nil
	}

	// Load the palette
	loader := palettes.NewLoader()
	palette, err := loader.Load(paletteName)
	if err != nil {
		return fmt.Errorf("loading palette %q: %w", paletteName, err)
	}

	// Generate theme-compatible CSS
	css := p.generateThemeCSS(palette)

	// Write to output directory
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}

	cssPath := filepath.Join(cssDir, "variables.css")
	if err := os.WriteFile(cssPath, []byte(css), 0644); err != nil {
		return fmt.Errorf("writing palette CSS: %w", err)
	}

	return nil
}

// generateThemeCSS generates CSS that maps palette colors to theme variable names.
// It includes the default theme's non-color variables (fonts, spacing, etc.)
// and maps the palette's semantic colors to the theme's expected names.
func (p *PaletteCSSPlugin) generateThemeCSS(palette *palettes.Palette) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("/* CSS Custom Properties - %s Theme */\n", palette.Name))
	buf.WriteString(":root {\n")

	// Map palette semantic colors to theme variable names
	// The theme expects: --color-primary, --color-text, --color-background, etc.
	// The palette provides: text-primary, bg-primary, accent, etc.

	// Primary/accent colors
	if accent := palette.Resolve("accent"); accent != "" {
		buf.WriteString(fmt.Sprintf("  --color-primary: %s;\n", accent))
	}
	if accentHover := palette.Resolve("accent-hover"); accentHover != "" {
		buf.WriteString(fmt.Sprintf("  --color-primary-light: %s;\n", accentHover))
		buf.WriteString(fmt.Sprintf("  --color-primary-dark: %s;\n", accentHover))
	}

	buf.WriteString("\n  /* Semantic colors */\n")

	// Text colors
	if textPrimary := palette.Resolve("text-primary"); textPrimary != "" {
		buf.WriteString(fmt.Sprintf("  --color-text: %s;\n", textPrimary))
	}
	if textMuted := palette.Resolve("text-muted"); textMuted != "" {
		buf.WriteString(fmt.Sprintf("  --color-text-muted: %s;\n", textMuted))
	}

	// Background colors
	if bgPrimary := palette.Resolve("bg-primary"); bgPrimary != "" {
		buf.WriteString(fmt.Sprintf("  --color-background: %s;\n", bgPrimary))
	}
	if bgSurface := palette.Resolve("bg-surface"); bgSurface != "" {
		buf.WriteString(fmt.Sprintf("  --color-surface: %s;\n", bgSurface))
	}

	// Border
	if border := palette.Resolve("border"); border != "" {
		buf.WriteString(fmt.Sprintf("  --color-border: %s;\n", border))
	}

	buf.WriteString("\n  /* Status colors */\n")

	// Status colors
	if success := palette.Resolve("success"); success != "" {
		buf.WriteString(fmt.Sprintf("  --color-success: %s;\n", success))
	}
	if warning := palette.Resolve("warning"); warning != "" {
		buf.WriteString(fmt.Sprintf("  --color-warning: %s;\n", warning))
	}
	if errorColor := palette.Resolve("error"); errorColor != "" {
		buf.WriteString(fmt.Sprintf("  --color-error: %s;\n", errorColor))
	}
	if info := palette.Resolve("info"); info != "" {
		buf.WriteString(fmt.Sprintf("  --color-info: %s;\n", info))
	}

	buf.WriteString("\n  /* Font families */\n")
	buf.WriteString("  --font-body: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;\n")
	buf.WriteString("  --font-heading: var(--font-body);\n")
	buf.WriteString("  --font-mono: ui-monospace, 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;\n")

	buf.WriteString("\n  /* Font sizes (modular scale) */\n")
	buf.WriteString("  --text-xs: 0.75rem;\n")
	buf.WriteString("  --text-sm: 0.875rem;\n")
	buf.WriteString("  --text-base: 1rem;\n")
	buf.WriteString("  --text-lg: 1.125rem;\n")
	buf.WriteString("  --text-xl: 1.25rem;\n")
	buf.WriteString("  --text-2xl: 1.5rem;\n")
	buf.WriteString("  --text-3xl: 1.875rem;\n")
	buf.WriteString("  --text-4xl: 2.25rem;\n")

	buf.WriteString("\n  /* Line heights */\n")
	buf.WriteString("  --leading-tight: 1.25;\n")
	buf.WriteString("  --leading-normal: 1.5;\n")
	buf.WriteString("  --leading-relaxed: 1.75;\n")

	buf.WriteString("\n  /* Spacing scale */\n")
	buf.WriteString("  --space-1: 0.25rem;\n")
	buf.WriteString("  --space-2: 0.5rem;\n")
	buf.WriteString("  --space-3: 0.75rem;\n")
	buf.WriteString("  --space-4: 1rem;\n")
	buf.WriteString("  --space-6: 1.5rem;\n")
	buf.WriteString("  --space-8: 2rem;\n")
	buf.WriteString("  --space-12: 3rem;\n")
	buf.WriteString("  --space-16: 4rem;\n")

	buf.WriteString("\n  /* Layout */\n")
	buf.WriteString("  --content-width: 65ch;\n")
	buf.WriteString("  --page-width: 1200px;\n")
	buf.WriteString("  --radius: 0.375rem;\n")
	buf.WriteString("  --radius-lg: 0.5rem;\n")

	// Add link colors if available
	if link := palette.Resolve("link"); link != "" {
		buf.WriteString(fmt.Sprintf("\n  /* Link colors */\n"))
		buf.WriteString(fmt.Sprintf("  --color-link: %s;\n", link))
		if linkHover := palette.Resolve("link-hover"); linkHover != "" {
			buf.WriteString(fmt.Sprintf("  --color-link-hover: %s;\n", linkHover))
		}
		if linkVisited := palette.Resolve("link-visited"); linkVisited != "" {
			buf.WriteString(fmt.Sprintf("  --color-link-visited: %s;\n", linkVisited))
		}
	}

	// Add code colors if available
	if codeBg := palette.Resolve("code-bg"); codeBg != "" {
		buf.WriteString(fmt.Sprintf("\n  /* Code colors */\n"))
		buf.WriteString(fmt.Sprintf("  --color-code-bg: %s;\n", codeBg))
		if codeText := palette.Resolve("code-text"); codeText != "" {
			buf.WriteString(fmt.Sprintf("  --color-code-text: %s;\n", codeText))
		}
	}

	buf.WriteString("}\n")

	return buf.String()
}

// getPaletteName extracts the palette name from config.Extra.
func (p *PaletteCSSPlugin) getPaletteName(extra map[string]interface{}) string {
	if extra == nil {
		return ""
	}

	// Check [markata-go.theme] section
	if theme, ok := extra["theme"].(map[string]interface{}); ok {
		if palette, ok := theme["palette"].(string); ok && palette != "" {
			return palette
		}
	}

	return ""
}

// Priority returns the plugin priority for the write stage.
// Should run after static_assets so it can overwrite the default variables.css
func (p *PaletteCSSPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityDefault // After static_assets (PriorityEarly)
	}
	return lifecycle.PriorityDefault
}

// Ensure PaletteCSSPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*PaletteCSSPlugin)(nil)
	_ lifecycle.WritePlugin    = (*PaletteCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*PaletteCSSPlugin)(nil)
)
