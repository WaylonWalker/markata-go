package plugins

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// PaletteCSSPlugin generates CSS variables from the configured color palette.
// It runs during the Write stage and creates/overwrites css/palette.css
// with the palette's CSS custom properties. It runs after static_assets
// to overwrite the default palette.css with palette-specific values.
//
// The plugin supports intelligent light/dark palette mapping:
// - palette = "everforest" will auto-detect everforest-light and everforest-dark
// - palette_light and palette_dark can override the auto-detected variants
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

// Configure generates CSS from the configured palette and registers its hash
// so templates can use the correct hashed filename. This runs in Configure stage
// before templates are rendered in Render stage.
func (p *PaletteCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Get palette configuration from config.Extra["theme"]
	paletteName, paletteLight, paletteDark := p.getPaletteConfig(config.Extra)
	userVariables := p.getThemeVariables(config.Extra)
	if paletteName == "" {
		return nil
	}

	// Load palettes
	loader := palettes.NewLoader()

	// Check if theme switcher is enabled
	switcherEnabled := p.isSwitcherEnabled(config.Extra)

	var css string
	if switcherEnabled {
		css = p.generateMultiPaletteCSS(loader, config.Extra, paletteName, paletteLight, paletteDark, userVariables)
	} else {
		css = p.generateSinglePaletteCSS(loader, paletteName, paletteLight, paletteDark, userVariables)
	}

	// Compute hash of generated CSS
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(css)))[:8]

	// Register hash so templates can use it
	m.SetAssetHash("css/palette.css", hash)
	templates.SetAssetHashes(map[string]string{"css/palette.css": hash})

	log.Printf("[palette_css] Registered hash %s for palette.css", hash)

	return nil
}

// Write generates CSS from the configured palette and writes it to the output directory.
func (p *PaletteCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir
	if config.Extra != nil {
		if fast, ok := config.Extra["fast_mode"].(bool); ok && fast {
			return nil
		}
	}

	// Get palette configuration from config.Extra["theme"]
	paletteName, paletteLight, paletteDark := p.getPaletteConfig(config.Extra)
	userVariables := p.getThemeVariables(config.Extra)
	if paletteName == "" {
		// No palette configured, skip
		log.Printf("[palette_css] No palette configured, skipping CSS generation")
		return nil
	}

	log.Printf("[palette_css] Generating CSS for palette: %s (light: %s, dark: %s)", paletteName, paletteLight, paletteDark)

	// Check if theme switcher is enabled
	switcherEnabled := p.isSwitcherEnabled(config.Extra)

	// Load palettes
	loader := palettes.NewLoader()

	var css string
	if switcherEnabled {
		// Generate CSS for all palettes when switcher is enabled
		css = p.generateMultiPaletteCSS(loader, config.Extra, paletteName, paletteLight, paletteDark, userVariables)
	} else {
		// Generate CSS for just the configured light/dark pair
		css = p.generateSinglePaletteCSS(loader, paletteName, paletteLight, paletteDark, userVariables)
	}

	// Write to output directory
	cssDir := filepath.Join(outputDir, "css")
	cssPath := filepath.Join(cssDir, "palette.css")
	if existing, err := os.ReadFile(cssPath); err == nil {
		if bytes.Equal(existing, []byte(css)) {
			log.Printf("[palette_css] CSS unchanged, skipping write")
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading existing palette CSS: %w", err)
	}

	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}
	//nolint:gosec // G306: palette.css is a public CSS file, 0644 is appropriate
	if err := os.WriteFile(cssPath, []byte(css), 0o644); err != nil {
		return fmt.Errorf("writing palette CSS: %w", err)
	}

	if hash := m.GetAssetHash("css/palette.css"); hash != "" {
		base := strings.TrimSuffix(filepath.Base(cssPath), filepath.Ext(cssPath))
		hashedPath := filepath.Join(cssDir, fmt.Sprintf("%s.%s.css", base, hash))
		//nolint:gosec // G306: hashed palette.css is a public CSS file, 0644 is appropriate
		if err := os.WriteFile(hashedPath, []byte(css), 0o644); err != nil {
			return fmt.Errorf("writing hashed palette CSS: %w", err)
		}
	}

	log.Printf("[palette_css] Wrote %d bytes to %s", len(css), cssPath)

	return nil
}

// isSwitcherEnabled checks if the theme switcher is enabled in config.
func (p *PaletteCSSPlugin) isSwitcherEnabled(extra map[string]interface{}) bool {
	if extra == nil {
		return false
	}
	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Switcher.IsEnabled()
	}
	// Check if it's a map (raw TOML)
	if theme, ok := extra["theme"].(map[string]interface{}); ok {
		if switcher, ok := theme["switcher"].(map[string]interface{}); ok {
			if enabled, ok := switcher["enabled"].(bool); ok {
				return enabled
			}
		}
	}
	return false
}

// getSwitcherConfig gets the theme switcher configuration from Extra.
func (p *PaletteCSSPlugin) getSwitcherConfig(extra map[string]interface{}) models.ThemeSwitcherConfig {
	if extra == nil {
		return models.NewThemeSwitcherConfig()
	}
	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Switcher
	}
	// Return default if not found
	return models.NewThemeSwitcherConfig()
}

// generateSinglePaletteCSS generates CSS for a single light/dark palette pair.
func (p *PaletteCSSPlugin) generateSinglePaletteCSS(loader *palettes.Loader, paletteName, paletteLight, paletteDark string, overrides map[string]string) string {
	// Get effective light and dark palette names
	lightName, darkName := palettes.GetEffectivePalettes(paletteName, paletteLight, paletteDark)

	var lightPalette, darkPalette *palettes.Palette
	var err error

	// Load light palette
	if lightName != "" {
		lightPalette, err = loader.Load(lightName)
		if err != nil {
			// Fall back to base palette, ignore error as we'll handle nil palette later
			//nolint:errcheck // fallback load failure is handled by nil check below
			lightPalette, _ = loader.Load(paletteName)
		}
	}

	// Load dark palette
	if darkName != "" {
		darkPalette, err = loader.Load(darkName)
		if err != nil {
			// Fall back to base palette, ignore error as we'll handle nil palette later
			//nolint:errcheck // fallback load failure is handled by nil check below
			darkPalette, _ = loader.Load(paletteName)
		}
	}

	// If we have no palettes loaded, try to load the base palette
	if lightPalette == nil && darkPalette == nil {
		palette, err := loader.Load(paletteName)
		if err == nil {
			lightPalette = palette
			darkPalette = palette
		}
	}

	// Generate theme-compatible CSS with both variants
	return p.generateThemeCSSWithVariants(lightPalette, darkPalette, lightName, darkName, overrides)
}

// PaletteManifestEntry represents a palette entry in the manifest.
type PaletteManifestEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Variant     string `json:"variant"`
	BaseName    string `json:"baseName"`
}

// generateMultiPaletteCSS generates CSS for all available palettes when switcher is enabled.
func (p *PaletteCSSPlugin) generateMultiPaletteCSS(loader *palettes.Loader, extra map[string]interface{}, paletteName, paletteLight, paletteDark string, overrides map[string]string) string {
	var buf bytes.Buffer

	buf.WriteString("@layer reset, tokens, base, components, utilities, overrides;\n\n")
	buf.WriteString("@layer tokens {\n")

	// Get all available palettes
	allPalettes, err := loader.Discover()
	if err != nil {
		// Fall back to single palette CSS on error
		return p.generateSinglePaletteCSS(loader, paletteName, paletteLight, paletteDark, overrides)
	}

	// Filter palettes based on switcher config
	switcherConfig := p.getSwitcherConfig(extra)
	filteredPalettes := p.filterPalettes(allPalettes, switcherConfig)

	// Get effective light and dark palette names for the default
	lightName, darkName := palettes.GetEffectivePalettes(paletteName, paletteLight, paletteDark)

	// Header comment
	buf.WriteString("/* CSS Custom Properties - Generated by markata-go */\n")
	buf.WriteString("/* Multi-palette theme switcher enabled */\n")
	buf.WriteString(fmt.Sprintf("/* Default: Light=%s, Dark=%s */\n\n", lightName, darkName))

	// Generate palette manifest for JavaScript
	manifest := p.generatePaletteManifest(filteredPalettes)
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		manifestJSON = []byte("[]")
	}
	// Escape single quotes in the JSON for CSS single-quoted string
	escapedManifest := strings.ReplaceAll(string(manifestJSON), "'", "\\'")

	// Write non-color variables and palette manifest in :root
	buf.WriteString("/* Global configuration and non-color variables */\n")
	buf.WriteString(":root {\n")
	buf.WriteString(fmt.Sprintf("  --palette-light: %q;\n", lightName))
	buf.WriteString(fmt.Sprintf("  --palette-dark: %q;\n", darkName))
	buf.WriteString(fmt.Sprintf("  --palette-manifest: '%s';\n", escapedManifest))
	buf.WriteString("  --palette-switcher-enabled: 1;\n")

	buf.WriteString("}\n\n")

	// Group palettes by base name for display purposes
	palettesByBase := make(map[string][]palettes.PaletteInfo)
	for _, info := range filteredPalettes {
		baseName := getBaseName(info.Name)
		palettesByBase[baseName] = append(palettesByBase[baseName], info)
	}

	// Generate CSS for each palette with data-palette attribute selector
	for _, info := range filteredPalettes {
		palette, err := loader.Load(info.Name)
		if err != nil {
			continue
		}

		// Normalize palette name for CSS selector (lowercase, hyphens)
		selectorName := normalizePaletteName(info.Name)

		buf.WriteString(fmt.Sprintf("/* Palette: %s (%s) */\n", info.Name, info.Variant))
		buf.WriteString(fmt.Sprintf("[data-palette=%q] {\n", selectorName))
		p.writePaletteVariablesIndented(&buf, palette, "  ")
		buf.WriteString("}\n\n")
	}

	// Generate default light mode (using configured light palette)
	// Only applies when no specific data-palette is set
	if lightName != "" {
		lightPalette, err := loader.Load(lightName)
		if err == nil {
			buf.WriteString(fmt.Sprintf("/* Default light mode - %s */\n", lightName))
			buf.WriteString(":root:not([data-palette]),\n")
			buf.WriteString("[data-theme=\"light\"]:not([data-palette]) {\n")
			p.writePaletteVariables(&buf, lightPalette)
			buf.WriteString("}\n\n")
		}
	}

	// Generate default dark mode (using configured dark palette)
	// Only applies when no specific data-palette is set
	if darkName != "" {
		darkPalette, err := loader.Load(darkName)
		if err == nil {
			buf.WriteString(fmt.Sprintf("/* Default dark mode - %s */\n", darkName))
			buf.WriteString("[data-theme=\"dark\"]:not([data-palette]) {\n")
			p.writePaletteVariables(&buf, darkPalette)
			buf.WriteString("}\n\n")

			// Also add prefers-color-scheme media query for auto mode
			buf.WriteString("/* Auto dark mode based on system preference */\n")
			buf.WriteString("@media (prefers-color-scheme: dark) {\n")
			buf.WriteString("  :root:not([data-theme=\"light\"]):not([data-palette]) {\n")
			p.writePaletteVariablesIndented(&buf, darkPalette, "    ")
			buf.WriteString("  }\n")
			buf.WriteString("}\n")
		}
	}

	buf.WriteString("}\n\n")
	p.writeThemeOverrides(&buf, overrides)

	return buf.String()
}

// filterPalettes filters palettes based on switcher configuration.
func (p *PaletteCSSPlugin) filterPalettes(allPalettes []palettes.PaletteInfo, switcherConfig models.ThemeSwitcherConfig) []palettes.PaletteInfo {
	if switcherConfig.IsIncludeAll() {
		// Include all, then exclude specified
		excludeSet := make(map[string]bool)
		for _, name := range switcherConfig.Exclude {
			excludeSet[strings.ToLower(name)] = true
			excludeSet[normalizePaletteName(name)] = true
		}

		var result []palettes.PaletteInfo
		for _, info := range allPalettes {
			lowerName := strings.ToLower(info.Name)
			normalized := normalizePaletteName(info.Name)
			if !excludeSet[lowerName] && !excludeSet[normalized] {
				result = append(result, info)
			}
		}
		return result
	}

	// Include only specified palettes
	includeSet := make(map[string]bool)
	for _, name := range switcherConfig.Include {
		includeSet[strings.ToLower(name)] = true
		includeSet[normalizePaletteName(name)] = true
	}

	var result []palettes.PaletteInfo
	for _, info := range allPalettes {
		lowerName := strings.ToLower(info.Name)
		normalized := normalizePaletteName(info.Name)
		if includeSet[lowerName] || includeSet[normalized] {
			result = append(result, info)
		}
	}
	return result
}

// generatePaletteManifest creates a manifest of available palettes for JavaScript.
func (p *PaletteCSSPlugin) generatePaletteManifest(paletteInfos []palettes.PaletteInfo) []PaletteManifestEntry {
	manifest := make([]PaletteManifestEntry, 0, len(paletteInfos))

	for _, info := range paletteInfos {
		entry := PaletteManifestEntry{
			Name:        normalizePaletteName(info.Name),
			DisplayName: info.Name,
			Variant:     string(info.Variant),
			BaseName:    getBaseName(info.Name),
		}
		manifest = append(manifest, entry)
	}

	// Sort by base name, then by variant (light first)
	sort.Slice(manifest, func(i, j int) bool {
		if manifest[i].BaseName != manifest[j].BaseName {
			return manifest[i].BaseName < manifest[j].BaseName
		}
		// Light variants come before dark
		return manifest[i].Variant < manifest[j].Variant
	})

	return manifest
}

// normalizePaletteName converts a palette name to a CSS-safe format.
func normalizePaletteName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// getBaseName extracts the base name from a palette name (removes -light, -dark suffix).
func getBaseName(name string) string {
	lower := strings.ToLower(name)
	lower = strings.TrimSuffix(lower, "-light")
	lower = strings.TrimSuffix(lower, "-dark")
	return lower
}

// generateThemeCSSWithVariants generates CSS with both light and dark palette variants.
// The CSS uses data-theme attribute for switching between variants.
// It preserves all non-color variables (typography, spacing, layout, gradients) from the default theme.
func (p *PaletteCSSPlugin) generateThemeCSSWithVariants(lightPalette, darkPalette *palettes.Palette, lightName, darkName string, overrides map[string]string) string {
	var buf bytes.Buffer

	buf.WriteString("@layer reset, tokens, base, components, utilities, overrides;\n\n")
	buf.WriteString("@layer tokens {\n")

	// Header comment
	buf.WriteString("/* CSS Custom Properties - Generated by markata-go */\n")
	buf.WriteString(fmt.Sprintf("/* Light: %s, Dark: %s */\n\n", lightName, darkName))

	// Generate palette info CSS custom properties for JavaScript
	buf.WriteString("/* Palette configuration for JavaScript theme toggle */\n")
	buf.WriteString(":root {\n")
	// CSS custom property values need the quotes as part of the value for JS to read
	buf.WriteString(fmt.Sprintf("  --palette-light: %q;\n", lightName))
	buf.WriteString(fmt.Sprintf("  --palette-dark: %q;\n", darkName))

	// Write non-color variables (typography, spacing, layout, gradients)
	// These are written once in :root and don't change between light/dark modes

	buf.WriteString("}\n\n")

	// Generate light mode (default) styles
	if lightPalette != nil {
		buf.WriteString(fmt.Sprintf("/* Light mode - %s */\n", lightPalette.Name))
		buf.WriteString(":root,\n")
		buf.WriteString("[data-theme=\"light\"] {\n")
		p.writePaletteVariables(&buf, lightPalette)
		buf.WriteString("}\n\n")
	}

	// Generate dark mode styles
	if darkPalette != nil {
		buf.WriteString(fmt.Sprintf("/* Dark mode - %s */\n", darkPalette.Name))
		buf.WriteString("[data-theme=\"dark\"] {\n")
		p.writePaletteVariables(&buf, darkPalette)
		buf.WriteString("}\n\n")

		// Also add prefers-color-scheme media query for auto mode
		buf.WriteString("/* Auto dark mode based on system preference */\n")
		buf.WriteString("@media (prefers-color-scheme: dark) {\n")
		buf.WriteString("  :root:not([data-theme=\"light\"]) {\n")
		p.writePaletteVariablesIndented(&buf, darkPalette, "    ")
		buf.WriteString("  }\n")
		buf.WriteString("}\n")
	}

	buf.WriteString("}\n\n")
	p.writeThemeOverrides(&buf, overrides)

	return buf.String()
}

// writePaletteVariables writes CSS custom properties for a palette.
func (p *PaletteCSSPlugin) writePaletteVariables(buf *bytes.Buffer, palette *palettes.Palette) {
	p.writePaletteVariablesIndented(buf, palette, "  ")
}

// resolveWithContrast resolves a color from the palette and adjusts it to meet WCAG contrast ratio against a background color.
func (p *PaletteCSSPlugin) resolveWithContrast(palette *palettes.Palette, fgKey string, minRatio float64) string {
	fgHex := palette.Resolve(fgKey)
	if fgHex == "" {
		return ""
	}

	bgHex := palette.Resolve("bg-primary")
	if bgHex == "" {
		return fgHex
	}

	fgColor, errFg := palettes.ParseHexColor(fgHex)
	bgColor, errBg := palettes.ParseHexColor(bgHex)
	if errFg != nil || errBg != nil {
		return fgHex
	}

	adjusted, _ := fgColor.AdjustForContrast(bgColor, minRatio)
	return adjusted.Hex()
}

// writePaletteVariablesIndented writes CSS custom properties with custom indentation.
//
//nolint:gocyclo // complexity is acceptable for a CSS generation function with many rules
func (p *PaletteCSSPlugin) writePaletteVariablesIndented(buf *bytes.Buffer, palette *palettes.Palette, indent string) {
	// Primary/accent colors
	if accent := palette.Resolve("accent"); accent != "" {
		fmt.Fprintf(buf, "%s--color-primary: %s;\n", indent, accent)
	}
	if accentHover := palette.Resolve("accent-hover"); accentHover != "" {
		fmt.Fprintf(buf, "%s--color-primary-light: %s;\n", indent, accentHover)
		fmt.Fprintf(buf, "%s--color-primary-dark: %s;\n", indent, accentHover)
	}

	fmt.Fprintf(buf, "\n%s/* Semantic colors */\n", indent)

	// Text colors
	if textPrimary := p.resolveWithContrast(palette, "text-primary", 4.5); textPrimary != "" {
		fmt.Fprintf(buf, "%s--color-text: %s;\n", indent, textPrimary)
	}
	if textSecondary := p.resolveWithContrast(palette, "text-secondary", 4.5); textSecondary != "" {
		fmt.Fprintf(buf, "%s--color-text-secondary: %s;\n", indent, textSecondary)
	}
	if textMuted := p.resolveWithContrast(palette, "text-muted", 4.5); textMuted != "" {
		fmt.Fprintf(buf, "%s--color-text-muted: %s;\n", indent, textMuted)
	}

	// Background colors
	if bgPrimary := palette.Resolve("bg-primary"); bgPrimary != "" {
		fmt.Fprintf(buf, "%s--color-background: %s;\n", indent, bgPrimary)
	}
	if bgSurface := palette.Resolve("bg-surface"); bgSurface != "" {
		fmt.Fprintf(buf, "%s--color-surface: %s;\n", indent, bgSurface)
	}

	// Border
	if border := palette.Resolve("border"); border != "" {
		fmt.Fprintf(buf, "%s--color-border: %s;\n", indent, border)
	}

	fmt.Fprintf(buf, "\n%s/* Status colors */\n", indent)

	// Status colors
	if success := p.resolveWithContrast(palette, "success", 3.0); success != "" {
		fmt.Fprintf(buf, "%s--color-success: %s;\n", indent, success)
	}
	if warning := p.resolveWithContrast(palette, "warning", 3.0); warning != "" {
		fmt.Fprintf(buf, "%s--color-warning: %s;\n", indent, warning)
	}
	if errorColor := p.resolveWithContrast(palette, "error", 3.0); errorColor != "" {
		fmt.Fprintf(buf, "%s--color-error: %s;\n", indent, errorColor)
	}
	if info := p.resolveWithContrast(palette, "info", 3.0); info != "" {
		fmt.Fprintf(buf, "%s--color-info: %s;\n", indent, info)
	}

	// Add link colors if available
	if link := p.resolveWithContrast(palette, "link", 4.5); link != "" {
		fmt.Fprintf(buf, "\n%s/* Link colors */\n", indent)
		fmt.Fprintf(buf, "%s--color-link: %s;\n", indent, link)
		if linkHover := p.resolveWithContrast(palette, "link-hover", 4.5); linkHover != "" {
			fmt.Fprintf(buf, "%s--color-link-hover: %s;\n", indent, linkHover)
		}
		if linkVisited := p.resolveWithContrast(palette, "link-visited", 4.5); linkVisited != "" {
			fmt.Fprintf(buf, "%s--color-link-visited: %s;\n", indent, linkVisited)
		}
	}

	// Add code colors if available
	if codeBg := palette.Resolve("code-bg"); codeBg != "" {
		fmt.Fprintf(buf, "\n%s/* Code colors */\n", indent)
		fmt.Fprintf(buf, "%s--color-code-bg: %s;\n", indent, codeBg)
		if codeText := palette.Resolve("code-text"); codeText != "" {
			fmt.Fprintf(buf, "%s--color-code-text: %s;\n", indent, codeText)
		}
	}

	// Add admonition colors if available
	admonitionTypes := []string{
		"note", "info", "tip", "hint", "success",
		"warn", "warning", "caution", "important",
		"danger", "error", "bug",
		"example", "quote", "abstract",
		"chat", "chat-reply",
	}

	hasAdmonitions := false
	for _, adType := range admonitionTypes {
		if palette.Resolve("admonition-"+adType+"-bg") != "" ||
			palette.Resolve("admonition-"+adType+"-border") != "" {
			hasAdmonitions = true
			break
		}
	}

	if hasAdmonitions {
		fmt.Fprintf(buf, "\n%s/* Admonition colors */\n", indent)
		for _, adType := range admonitionTypes {
			if bg := palette.Resolve("admonition-" + adType + "-bg"); bg != "" {
				fmt.Fprintf(buf, "%s--admonition-%s-bg: %s;\n", indent, adType, bg)
			}
			if border := palette.Resolve("admonition-" + adType + "-border"); border != "" {
				fmt.Fprintf(buf, "%s--admonition-%s-border: %s;\n", indent, adType, border)
			}
		}
	}

	// Add mark/highlight colors
	// These are used for ==highlighted text== rendered as <mark> elements
	p.writeMarkColors(buf, palette, indent)
}

// writeMarkColors generates CSS variables for mark/highlight elements.
// If mark-bg/mark-text are not defined in the palette, computes them from the warning color.
func (p *PaletteCSSPlugin) writeMarkColors(buf *bytes.Buffer, palette *palettes.Palette, indent string) {
	fmt.Fprintf(buf, "\n%s/* Mark/highlight colors */\n", indent)

	// Try explicit mark colors first
	markBg := palette.Resolve("mark-bg")
	markText := palette.Resolve("mark-text")

	// If not defined, compute from warning color
	if markBg == "" {
		warningHex := p.resolveWithContrast(palette, "warning", 3.0)
		if warningHex != "" {
			warningColor, err := palettes.ParseHexColor(warningHex)
			if err == nil {
				// Light palettes: lighten warning for subtle background
				// Dark palettes: darken warning for subtle background
				if palette.Variant == palettes.VariantLight {
					markBg = warningColor.Lighten(0.75).Hex()
				} else {
					markBg = warningColor.Darken(0.6).Hex()
				}
			}
		}
	}

	// Write mark-bg if we have it
	if markBg != "" {
		fmt.Fprintf(buf, "%s--color-mark-bg: %s;\n", indent, markBg)
	}

	// Compute mark-text with contrast checking
	if markText == "" && markBg != "" {
		bgColor, err := palettes.ParseHexColor(markBg)
		if err == nil {
			// Get the text color from the palette
			textHex := p.resolveWithContrast(palette, "text-primary", 4.5)
			if textHex == "" {
				// Fallback: use black for light bg, white for dark bg
				if bgColor.RelativeLuminance() > 0.5 {
					textHex = "#1a1a1a"
				} else {
					textHex = "#f5f5f5"
				}
			}

			textColor, err := palettes.ParseHexColor(textHex)
			if err == nil {
				// Ensure WCAG AA contrast (4.5:1 for normal text)
				adjustedText, ok := textColor.AdjustForContrast(bgColor, 4.5)
				if ok {
					markText = adjustedText.Hex()
				} else {
					// Couldn't adjust, use high contrast fallback
					if bgColor.RelativeLuminance() > 0.5 {
						markText = "#000000"
					} else {
						markText = "#ffffff"
					}
				}
			}
		}
	}

	// Write mark-text if we have it
	if markText != "" {
		fmt.Fprintf(buf, "%s--color-mark-text: %s;\n", indent, markText)
	}

	// Add selection colors (for user text selection)
	// These default to mark colors but can be overridden
	p.writeSelectionColors(buf, palette, indent)
}

// writeSelectionColors generates CSS variables for user text selection.
// Defaults to mark colors for consistency, but allows explicit override via palette.
func (p *PaletteCSSPlugin) writeSelectionColors(buf *bytes.Buffer, palette *palettes.Palette, indent string) {
	// Check for explicit selection colors
	selectionBg := palette.Resolve("selection-bg")
	selectionText := palette.Resolve("selection-text")

	// Only write if explicitly set (CSS fallback handles the mark color default)
	if selectionBg != "" || selectionText != "" {
		fmt.Fprintf(buf, "\n%s/* Selection colors */\n", indent)
		if selectionBg != "" {
			fmt.Fprintf(buf, "%s--color-selection-bg: %s;\n", indent, selectionBg)
		}
		if selectionText != "" {
			fmt.Fprintf(buf, "%s--color-selection-text: %s;\n", indent, selectionText)
		}
	}
}

// getPaletteConfig extracts palette configuration from config.Extra.
// Returns the base palette name and optional light/dark overrides.
func (p *PaletteCSSPlugin) getPaletteConfig(extra map[string]interface{}) (palette, paletteLight, paletteDark string) {
	if extra == nil {
		return "", "", ""
	}

	if modelsConfig, ok := extra["models_config"].(*models.Config); ok {
		if modelsConfig.Theme.Palette != "" || modelsConfig.Theme.PaletteLight != "" || modelsConfig.Theme.PaletteDark != "" {
			return modelsConfig.Theme.Palette, modelsConfig.Theme.PaletteLight, modelsConfig.Theme.PaletteDark
		}
	}

	// Check if theme is a models.ThemeConfig (from core.go)
	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Palette, themeConfig.PaletteLight, themeConfig.PaletteDark
	}

	// Check if theme is a map[string]interface{} (from benchmark.go or raw TOML)
	theme, ok := extra["theme"].(map[string]interface{})
	if !ok {
		return "", "", ""
	}

	// Get base palette
	if pal, ok := theme["palette"].(string); ok && pal != "" {
		palette = pal
	}

	// Get optional light override
	if pal, ok := theme["palette_light"].(string); ok && pal != "" {
		paletteLight = pal
	}

	// Get optional dark override
	if pal, ok := theme["palette_dark"].(string); ok && pal != "" {
		paletteDark = pal
	}

	return palette, paletteLight, paletteDark
}

func (p *PaletteCSSPlugin) getThemeVariables(extra map[string]interface{}) map[string]string {
	if extra == nil {
		return nil
	}

	if modelsConfig, ok := extra["models_config"].(*models.Config); ok {
		if len(modelsConfig.Theme.Variables) > 0 {
			return normalizeThemeVariables(modelsConfig.Theme.Variables)
		}
	}

	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return normalizeThemeVariables(themeConfig.Variables)
	}

	theme, ok := extra["theme"].(map[string]interface{})
	if !ok {
		return nil
	}

	if vars, ok := theme["variables"].(map[string]string); ok {
		return normalizeThemeVariables(vars)
	}

	if vars, ok := theme["variables"].(map[string]interface{}); ok {
		converted := make(map[string]string, len(vars))
		for key, value := range vars {
			str, ok := value.(string)
			if !ok {
				continue
			}
			converted[key] = str
		}
		return normalizeThemeVariables(converted)
	}

	return nil
}

func normalizeThemeVariables(vars map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(vars))
	for key, value := range vars {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if !strings.HasPrefix(key, "--") {
			continue
		}
		normalized[key] = value
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func (p *PaletteCSSPlugin) writeThemeOverrides(buf *bytes.Buffer, overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buf.WriteString("@layer overrides {\n")
	buf.WriteString("  /* Theme variable overrides */\n")
	buf.WriteString("  :root {\n")
	for _, key := range keys {
		fmt.Fprintf(buf, "    %s: %s;\n", key, overrides[key])
	}
	buf.WriteString("  }\n")
	buf.WriteString("}\n")
}

// Priority returns the plugin priority for the write stage.
// Should run after static_assets so it can overwrite the default palette.css
func (p *PaletteCSSPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityDefault // After static_assets (PriorityEarly)
	}
	return lifecycle.PriorityDefault
}

// Ensure PaletteCSSPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*PaletteCSSPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*PaletteCSSPlugin)(nil)
	_ lifecycle.WritePlugin     = (*PaletteCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*PaletteCSSPlugin)(nil)
)
