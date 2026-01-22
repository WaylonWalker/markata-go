package palettes

import "testing"

func TestChromaTheme(t *testing.T) {
	tests := []struct {
		palette string
		want    string
	}{
		// Catppuccin family
		{"catppuccin-latte", "catppuccin-latte"},
		{"catppuccin-frappe", "catppuccin-frappe"},
		{"catppuccin-macchiato", "catppuccin-macchiato"},
		{"catppuccin-mocha", "catppuccin-mocha"},

		// Nord
		{"nord-light", "nord"},
		{"nord-dark", "nord"},

		// Gruvbox
		{"gruvbox-light", "gruvbox-light"},
		{"gruvbox-dark", "gruvbox"},

		// Tokyo Night
		{"tokyo-night", "tokyonight-night"},
		{"tokyo-night-storm", "tokyonight-storm"},
		{"tokyo-night-day", "tokyonight-day"},

		// Rose Pine
		{"rose-pine", "rose-pine"},
		{"rose-pine-moon", "rose-pine-moon"},
		{"rose-pine-dawn", "rose-pine-dawn"},

		// Everforest
		{"everforest-light", "evergarden"},
		{"everforest-dark", "evergarden"},

		// Dracula
		{"dracula", "dracula"},

		// Solarized
		{"solarized-light", "solarized-light"},
		{"solarized-dark", "solarized-dark"},

		// Kanagawa
		{"kanagawa-wave", "vim"},
		{"kanagawa-dragon", "vim"},
		{"kanagawa-lotus", "modus-operandi"},

		// Default themes
		{"default-light", "github"},
		{"default-dark", "github-dark"},

		// Matte black
		{"matte-black", "monokai"},

		// Unknown palette returns empty
		{"unknown-palette", ""},
	}

	for _, tt := range tests {
		t.Run(tt.palette, func(t *testing.T) {
			got := ChromaTheme(tt.palette)
			if got != tt.want {
				t.Errorf("ChromaTheme(%q) = %q, want %q", tt.palette, got, tt.want)
			}
		})
	}
}

func TestChromaThemeForVariant(t *testing.T) {
	tests := []struct {
		variant Variant
		want    string
	}{
		{VariantLight, DefaultChromaThemeLight},
		{VariantDark, DefaultChromaThemeDark},
	}

	for _, tt := range tests {
		t.Run(string(tt.variant), func(t *testing.T) {
			got := ChromaThemeForVariant(tt.variant)
			if got != tt.want {
				t.Errorf("ChromaThemeForVariant(%q) = %q, want %q", tt.variant, got, tt.want)
			}
		})
	}
}

func TestDefaultChromaThemes(t *testing.T) {
	// Verify defaults are sensible
	if DefaultChromaThemeLight != "github" {
		t.Errorf("DefaultChromaThemeLight = %q, want %q", DefaultChromaThemeLight, "github")
	}
	if DefaultChromaThemeDark != "github-dark" {
		t.Errorf("DefaultChromaThemeDark = %q, want %q", DefaultChromaThemeDark, "github-dark")
	}
}

func TestAvailableChromaThemes(t *testing.T) {
	// Verify we have a reasonable number of themes
	if len(AvailableChromaThemes) < 30 {
		t.Errorf("AvailableChromaThemes has %d themes, expected at least 30", len(AvailableChromaThemes))
	}

	// Verify defaults are in the list
	found := make(map[string]bool)
	for _, theme := range AvailableChromaThemes {
		found[theme] = true
	}

	requiredThemes := []string{
		"github", "github-dark", "monokai", "dracula",
		"catppuccin-mocha", "rose-pine", "nord", "gruvbox",
	}

	for _, theme := range requiredThemes {
		if !found[theme] {
			t.Errorf("AvailableChromaThemes missing required theme %q", theme)
		}
	}
}
