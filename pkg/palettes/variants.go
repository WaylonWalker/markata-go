package palettes

import "strings"

// PaletteVariants holds the light and dark variants for a palette.
type PaletteVariants struct {
	// Light is the light variant palette name (empty if not found)
	Light string `json:"light"`
	// Dark is the dark variant palette name (empty if not found)
	Dark string `json:"dark"`
	// Base is the base palette name without variant suffix
	Base string `json:"base"`
}

// DetectVariants detects light and dark palette variants for a given palette name.
// It uses intelligent mapping rules:
// 1. If palette ends with "-light" or "-dark", extract base and find pair
// 2. If palette has no suffix, try appending "-light" and "-dark"
// 3. For named variants (like catppuccin-latte), use known mappings
//
// Examples:
//   - "everforest" -> light: "everforest-light", dark: "everforest-dark"
//   - "everforest-light" -> light: "everforest-light", dark: "everforest-dark"
//   - "catppuccin-mocha" -> light: "catppuccin-latte", dark: "catppuccin-mocha"
//   - "dracula" -> light: "", dark: "dracula" (no light variant)
func DetectVariants(name string) PaletteVariants {
	loader := NewLoader()
	return detectVariantsWithLoader(name, loader)
}

// detectVariantsWithLoader is the internal implementation that accepts a loader.
func detectVariantsWithLoader(name string, loader *Loader) PaletteVariants {
	result := PaletteVariants{Base: name}

	// Check if the palette exists
	palette, err := loader.Load(name)
	if err != nil {
		// Palette doesn't exist, return empty
		return result
	}

	// Determine the base name and current variant
	baseName := extractBaseName(name)
	result.Base = baseName

	// Check for known palette families with non-standard naming
	if variants := getKnownVariants(name); variants != nil {
		result.Light = variants.Light
		result.Dark = variants.Dark
		return result
	}

	// Standard naming: base-light / base-dark
	lightName := baseName + "-light"
	darkName := baseName + "-dark"

	// Check if light variant exists
	if _, err := loader.Load(lightName); err == nil {
		result.Light = lightName
	}

	// Check if dark variant exists
	if _, err := loader.Load(darkName); err == nil {
		result.Dark = darkName
	}

	// If the current palette is already a variant, use it
	if palette.Variant == VariantLight && result.Light == "" {
		result.Light = name
	} else if palette.Variant == VariantDark && result.Dark == "" {
		result.Dark = name
	}

	// If only one variant exists, use the current palette for the other
	if result.Light == "" && result.Dark != "" && palette.Variant == VariantLight {
		result.Light = name
	} else if result.Dark == "" && result.Light != "" && palette.Variant == VariantDark {
		result.Dark = name
	}

	return result
}

// extractBaseName extracts the base palette name without variant suffixes.
func extractBaseName(name string) string {
	// Common suffixes to strip
	suffixes := []string{"-light", "-dark", "-day", "-night", "-storm", "-moon", "-dawn"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}

	return name
}

// knownVariantMappings maps palette names to their light/dark variants.
// This handles palettes with non-standard naming conventions.
var knownVariantMappings = map[string]PaletteVariants{
	// Catppuccin family - latte is light, mocha/macchiato/frappe are dark
	"catppuccin-latte":     {Base: "catppuccin", Light: "catppuccin-latte", Dark: "catppuccin-mocha"},
	"catppuccin-mocha":     {Base: "catppuccin", Light: "catppuccin-latte", Dark: "catppuccin-mocha"},
	"catppuccin-frappe":    {Base: "catppuccin", Light: "catppuccin-latte", Dark: "catppuccin-frappe"},
	"catppuccin-macchiato": {Base: "catppuccin", Light: "catppuccin-latte", Dark: "catppuccin-macchiato"},

	// Rose Pine family - dawn is light, main and moon are dark
	"rose-pine":      {Base: "rose-pine", Light: "rose-pine-dawn", Dark: "rose-pine"},
	"rose-pine-dawn": {Base: "rose-pine", Light: "rose-pine-dawn", Dark: "rose-pine"},
	"rose-pine-moon": {Base: "rose-pine", Light: "rose-pine-dawn", Dark: "rose-pine-moon"},

	// Tokyo Night family - day is light, main and storm are dark
	"tokyo-night":       {Base: "tokyo-night", Light: "tokyo-night-day", Dark: "tokyo-night"},
	"tokyo-night-day":   {Base: "tokyo-night", Light: "tokyo-night-day", Dark: "tokyo-night"},
	"tokyo-night-storm": {Base: "tokyo-night", Light: "tokyo-night-day", Dark: "tokyo-night-storm"},

	// Kanagawa family - lotus is light, wave and dragon are dark
	"kanagawa-wave":   {Base: "kanagawa", Light: "kanagawa-lotus", Dark: "kanagawa-wave"},
	"kanagawa-lotus":  {Base: "kanagawa", Light: "kanagawa-lotus", Dark: "kanagawa-wave"},
	"kanagawa-dragon": {Base: "kanagawa", Light: "kanagawa-lotus", Dark: "kanagawa-dragon"},

	// Single-variant palettes (dark only)
	"dracula":     {Base: "dracula", Light: "", Dark: "dracula"},
	"matte-black": {Base: "matte-black", Light: "", Dark: "matte-black"},
}

// getKnownVariants returns known variant mappings for special palette families.
func getKnownVariants(name string) *PaletteVariants {
	if variants, ok := knownVariantMappings[name]; ok {
		return &variants
	}
	return nil
}

// GetEffectivePalettes returns the actual palette names to use for light and dark modes.
// It respects explicit overrides while providing intelligent defaults.
//
// Parameters:
//   - palette: the base palette from config (e.g., "everforest")
//   - paletteLight: explicit light variant override (optional)
//   - paletteDark: explicit dark variant override (optional)
//
// Returns:
//   - light: the palette name to use for light mode
//   - dark: the palette name to use for dark mode
func GetEffectivePalettes(palette, paletteLight, paletteDark string) (light, dark string) {
	if palette == "generated" {
		light = "generated-light"
		dark = "generated-dark"
		if paletteLight != "" {
			light = paletteLight
		}
		if paletteDark != "" {
			dark = paletteDark
		}
		return light, dark
	}

	// If both are explicitly set, use them directly
	if paletteLight != "" && paletteDark != "" {
		return paletteLight, paletteDark
	}

	// Detect variants for the base palette
	variants := DetectVariants(palette)

	// Use explicit override or detected variant
	light = paletteLight
	if light == "" {
		light = variants.Light
	}

	dark = paletteDark
	if dark == "" {
		dark = variants.Dark
	}

	// Fallback to the base palette if no variant found
	if light == "" && dark == "" {
		// Single palette mode - use for both
		return palette, palette
	}

	// If only one variant exists, use the base for the other
	if light == "" {
		light = palette
	}
	if dark == "" {
		dark = palette
	}

	return light, dark
}
