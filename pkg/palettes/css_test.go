package palettes

import (
	"strings"
	"testing"
)

func TestPalette_GenerateCSS(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red":  "#ff0000",
			"blue": "#0000ff",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
		Components: map[string]string{
			"button-bg": "primary",
		},
	}

	css := p.GenerateCSS()

	// Should contain :root
	if !strings.Contains(css, ":root") {
		t.Error("CSS should contain :root selector")
	}

	// Should contain raw colors
	if !strings.Contains(css, "--palette-red:") || !strings.Contains(css, "#ff0000") {
		t.Error("CSS should contain raw color definitions")
	}

	// Should contain semantic colors with var() references
	if !strings.Contains(css, "--color-primary:") {
		t.Error("CSS should contain semantic color definitions")
	}

	// Should contain component colors
	if !strings.Contains(css, "--button-bg:") {
		t.Error("CSS should contain component color definitions")
	}
}

func TestPalette_GenerateCSSWithFormat(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red": "#ff0000",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
	}

	t.Run("minified", func(t *testing.T) {
		format := CSSFormat{
			UseReferences:   true,
			RawPrefix:       "palette",
			SemanticPrefix:  "color",
			IncludeRaw:      true,
			IncludeSemantic: true,
			Minify:          true,
		}

		css := p.GenerateCSSWithFormat(format)

		// Should not have newlines (except possibly at the end)
		lines := strings.Split(strings.TrimSpace(css), "\n")
		if len(lines) > 2 {
			t.Errorf("Minified CSS should have few lines, got %d", len(lines))
		}
	})

	t.Run("custom prefix", func(t *testing.T) {
		format := CSSFormat{
			UseReferences:   true,
			RawPrefix:       "custom",
			SemanticPrefix:  "theme",
			IncludeRaw:      true,
			IncludeSemantic: true,
		}

		css := p.GenerateCSSWithFormat(format)

		if !strings.Contains(css, "--custom-red:") {
			t.Error("CSS should use custom raw prefix")
		}
		if !strings.Contains(css, "--theme-primary:") {
			t.Error("CSS should use custom semantic prefix")
		}
	})

	t.Run("resolved values", func(t *testing.T) {
		format := CSSFormat{
			UseReferences:   false, // Resolved hex values
			RawPrefix:       "palette",
			SemanticPrefix:  "color",
			IncludeRaw:      true,
			IncludeSemantic: true,
		}

		css := p.GenerateCSSWithFormat(format)

		// Semantic colors should have hex values, not var() references
		if strings.Contains(css, "var(--palette-red)") {
			t.Error("CSS with UseReferences=false should not contain var() references")
		}
	})

	t.Run("exclude sections", func(t *testing.T) {
		format := CSSFormat{
			IncludeRaw:        false,
			IncludeSemantic:   true,
			IncludeComponents: false,
		}

		css := p.GenerateCSSWithFormat(format)

		if strings.Contains(css, "/* Raw colors */") {
			t.Error("CSS should not contain raw colors section")
		}
	})
}

func TestPalette_GenerateSCSS(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red":  "#ff0000",
			"blue": "#0000ff",
		},
		Semantic: map[string]string{
			"primary":   "red",
			"secondary": "blue",
		},
		Components: map[string]string{
			"button-bg": "primary",
		},
	}

	scss := p.GenerateSCSS()

	// Should contain SCSS variable syntax
	if !strings.Contains(scss, "$palette-red:") {
		t.Error("SCSS should contain palette variables")
	}
	if !strings.Contains(scss, "$color-primary:") {
		t.Error("SCSS should contain semantic variables")
	}
	if !strings.Contains(scss, "$button-bg:") {
		t.Error("SCSS should contain component variables")
	}

	// Should reference other SCSS variables
	if !strings.Contains(scss, "$palette-red;") {
		t.Error("SCSS semantic should reference palette variables")
	}
}

func TestPalette_ExportJSON(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Author:  "Test Author",
		Colors: map[string]string{
			"red": "#ff0000",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
	}

	t.Run("without resolved", func(t *testing.T) {
		data, err := p.ExportJSON(false)
		if err != nil {
			t.Fatalf("ExportJSON() error = %v", err)
		}

		json := string(data)

		if !strings.Contains(json, `"name": "Test"`) {
			t.Error("JSON should contain name")
		}
		if !strings.Contains(json, `"variant": "dark"`) {
			t.Error("JSON should contain variant")
		}
		if strings.Contains(json, `"resolved"`) {
			t.Error("JSON without includeResolved should not have resolved field")
		}
	})

	t.Run("with resolved", func(t *testing.T) {
		data, err := p.ExportJSON(true)
		if err != nil {
			t.Fatalf("ExportJSON() error = %v", err)
		}

		json := string(data)

		if !strings.Contains(json, `"resolved"`) {
			t.Error("JSON with includeResolved should have resolved field")
		}
	})
}

func TestPalette_GenerateTailwind(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red":  "#ff0000",
			"blue": "#0000ff",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
	}

	tw := p.GenerateTailwind()

	// Should be valid JS module
	if !strings.Contains(tw, "module.exports") {
		t.Error("Tailwind config should export module")
	}

	// Should have colors under theme.extend.colors
	if !strings.Contains(tw, "theme:") {
		t.Error("Tailwind config should have theme section")
	}
	if !strings.Contains(tw, "extend:") {
		t.Error("Tailwind config should have extend section")
	}
	if !strings.Contains(tw, "colors:") {
		t.Error("Tailwind config should have colors section")
	}

	// Should have palette namespace
	if !strings.Contains(tw, "palette:") {
		t.Error("Tailwind config should have palette namespace")
	}

	// Should contain color values
	if !strings.Contains(tw, "#ff0000") {
		t.Error("Tailwind config should contain hex values")
	}
}

func TestGenerateDarkModeCSS(t *testing.T) {
	light := &Palette{
		Name:    "Light",
		Variant: VariantLight,
		Colors: map[string]string{
			"bg": "#ffffff",
		},
		Semantic: map[string]string{
			"bg-primary": "bg",
		},
	}

	dark := &Palette{
		Name:    "Dark",
		Variant: VariantDark,
		Colors: map[string]string{
			"bg": "#000000",
		},
		Semantic: map[string]string{
			"bg-primary": "bg",
		},
	}

	css := GenerateDarkModeCSS(light, dark)

	// Should have light mode (default)
	if !strings.Contains(css, "Light mode (default)") {
		t.Error("CSS should contain light mode section")
	}

	// Should have dark mode media query
	if !strings.Contains(css, "@media (prefers-color-scheme: dark)") {
		t.Error("CSS should contain dark mode media query")
	}

	// Should have both palettes' colors
	if !strings.Contains(css, "#ffffff") {
		t.Error("CSS should contain light palette colors")
	}
	if !strings.Contains(css, "#000000") {
		t.Error("CSS should contain dark palette colors")
	}
}

func TestDefaultCSSFormat(t *testing.T) {
	format := DefaultCSSFormat()

	if !format.UseReferences {
		t.Error("Default format should use references")
	}
	if format.RawPrefix != "palette" {
		t.Errorf("Default RawPrefix = %q, want %q", format.RawPrefix, "palette")
	}
	if format.SemanticPrefix != "color" {
		t.Errorf("Default SemanticPrefix = %q, want %q", format.SemanticPrefix, "color")
	}
	if !format.IncludeRaw {
		t.Error("Default format should include raw colors")
	}
	if !format.IncludeSemantic {
		t.Error("Default format should include semantic colors")
	}
	if !format.IncludeComponents {
		t.Error("Default format should include component colors")
	}
	if format.Minify {
		t.Error("Default format should not minify")
	}
}

func TestCssVarName(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		suffix string
		want   string
	}{
		{"palette", "red", "", "--palette-red"},
		{"color", "primary", "", "--color-primary"},
		{"", "button", "", "--button"},
		{"theme", "bg", "light", "--theme-bg-light"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := cssVarName(tt.prefix, tt.name, tt.suffix)
			if got != tt.want {
				t.Errorf("cssVarName(%q, %q, %q) = %q, want %q",
					tt.prefix, tt.name, tt.suffix, got, tt.want)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]string{
		"zebra":  "z",
		"apple":  "a",
		"mango":  "m",
		"banana": "b",
	}

	keys := sortedKeys(m)

	expected := []string{"apple", "banana", "mango", "zebra"}
	if len(keys) != len(expected) {
		t.Fatalf("sortedKeys() returned %d keys, want %d", len(keys), len(expected))
	}

	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("sortedKeys()[%d] = %q, want %q", i, k, expected[i])
		}
	}
}
