package palettes

// ChromaTheme returns the Chroma syntax highlighting theme name that best matches
// the given palette name. If no specific mapping exists, it returns a sensible
// default based on the palette's variant (light/dark).
//
// This mapping allows the site's color palette to automatically drive the code
// highlighting theme, providing a cohesive visual experience.
func ChromaTheme(paletteName string) string {
	if theme, ok := paletteToChroma[paletteName]; ok {
		return theme
	}
	// Default fallback - will be determined by variant in the caller
	return ""
}

// ChromaThemeForVariant returns a default Chroma theme for the given variant.
// Used as a fallback when no specific palette mapping exists.
func ChromaThemeForVariant(variant Variant) string {
	if variant == VariantLight {
		return DefaultChromaThemeLight
	}
	return DefaultChromaThemeDark
}

// DefaultChromaThemeLight is the default Chroma theme for light palettes.
const DefaultChromaThemeLight = "github"

// DefaultChromaThemeDark is the default Chroma theme for dark palettes.
const DefaultChromaThemeDark = "github-dark"

// paletteToChroma maps site palette names to their corresponding Chroma themes.
// The mappings are chosen based on:
// 1. Exact matches where Chroma has the same theme (catppuccin, rose-pine, etc.)
// 2. Closest visual match for themes without exact Chroma equivalents
var paletteToChroma = map[string]string{
	// Catppuccin family - exact matches in Chroma
	"catppuccin-latte":     "catppuccin-latte",
	"catppuccin-frappe":    "catppuccin-frappe",
	"catppuccin-macchiato": "catppuccin-macchiato",
	"catppuccin-mocha":     "catppuccin-mocha",

	// Nord - Chroma has "nord" which works for both variants
	"nord-light": "nord",
	"nord-dark":  "nord",

	// Gruvbox - Chroma has both variants
	"gruvbox-light": "gruvbox-light",
	"gruvbox-dark":  "gruvbox",

	// Tokyo Night - Chroma has all variants (note: Chroma uses "tokyonight" not "tokyo-night")
	"tokyo-night":       "tokyonight-night",
	"tokyo-night-storm": "tokyonight-storm",
	"tokyo-night-day":   "tokyonight-day",

	// Rose Pine - exact matches in Chroma
	"rose-pine":      "rose-pine",
	"rose-pine-moon": "rose-pine-moon",
	"rose-pine-dawn": "rose-pine-dawn",

	// Everforest - Chroma has "evergarden" which is similar
	"everforest-light": "evergarden",
	"everforest-dark":  "evergarden",

	// Dracula - exact match
	"dracula": "dracula",

	// Solarized - exact matches
	"solarized-light": "solarized-light",
	"solarized-dark":  "solarized-dark",

	// Kanagawa - no exact Chroma match, use vim (similar Japanese aesthetic)
	// or nord for its muted tones
	"kanagawa-wave":   "vim",
	"kanagawa-dragon": "vim",
	"kanagawa-lotus":  "modus-operandi", // Light variant, use clean light theme

	// Default themes
	"default-light": "github",
	"default-dark":  "github-dark",

	// Matte black - use monokai for its dark, contrasty appearance
	"matte-black": "monokai",
}

// AvailableChromaThemes returns a list of all Chroma themes that are known to work.
// This is a subset of all Chroma styles, focusing on popular, well-maintained themes.
var AvailableChromaThemes = []string{
	// Light themes
	"github",
	"gruvbox-light",
	"solarized-light",
	"catppuccin-latte",
	"rose-pine-dawn",
	"tokyonight-day",
	"modus-operandi",
	"evergarden",
	"xcode",
	"vs",
	"autumn",
	"friendly",
	"tango",
	"trac",
	"perldoc",
	"lovelace",
	"paraiso-light",
	"algol",
	"emacs",

	// Dark themes
	"github-dark",
	"gruvbox",
	"solarized-dark",
	"catppuccin-frappe",
	"catppuccin-macchiato",
	"catppuccin-mocha",
	"rose-pine",
	"rose-pine-moon",
	"tokyonight-night",
	"tokyonight-storm",
	"tokyonight-moon",
	"nord",
	"dracula",
	"monokai",
	"onedark",
	"doom-one",
	"doom-one2",
	"vim",
	"native",
	"fruity",
	"xcode-dark",
	"modus-vivendi",
	"aura-theme-dark",
	"base16-snazzy",
	"witchhazel",
	"paraiso-dark",
	"hrdark",
	"vulcan",
	"rrt",
}
