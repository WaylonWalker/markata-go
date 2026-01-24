package palettes

import "testing"

func TestExtractBaseName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"light suffix", "everforest-light", "everforest"},
		{"dark suffix", "everforest-dark", "everforest"},
		{"day suffix", "tokyo-night-day", "tokyo-night"},
		{"night suffix", "nord-night", "nord"},
		{"storm suffix", "tokyo-night-storm", "tokyo-night"},
		{"moon suffix", "rose-pine-moon", "rose-pine"},
		{"dawn suffix", "rose-pine-dawn", "rose-pine"},
		{"no suffix", "dracula", "dracula"},
		{"multiple hyphens", "nord-aurora-light", "nord-aurora"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBaseName(tt.input)
			if got != tt.expected {
				t.Errorf("extractBaseName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetKnownVariants(t *testing.T) {
	tests := []struct {
		name        string
		paletteName string
		expectLight string
		expectDark  string
		expectNil   bool
	}{
		{"catppuccin-latte", "catppuccin-latte", "catppuccin-latte", "catppuccin-mocha", false},
		{"catppuccin-mocha", "catppuccin-mocha", "catppuccin-latte", "catppuccin-mocha", false},
		{"rose-pine-dawn", "rose-pine-dawn", "rose-pine-dawn", "rose-pine", false},
		{"rose-pine", "rose-pine", "rose-pine-dawn", "rose-pine", false},
		{"tokyo-night", "tokyo-night", "tokyo-night-day", "tokyo-night", false},
		{"tokyo-night-day", "tokyo-night-day", "tokyo-night-day", "tokyo-night", false},
		{"dracula (dark only)", "dracula", "", "dracula", false},
		{"unknown palette", "unknown-palette", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variants := getKnownVariants(tt.paletteName)
			if tt.expectNil {
				if variants != nil {
					t.Errorf("getKnownVariants(%q) = %+v, want nil", tt.paletteName, variants)
				}
				return
			}
			if variants == nil {
				t.Fatalf("getKnownVariants(%q) = nil, want non-nil", tt.paletteName)
			}
			if variants.Light != tt.expectLight {
				t.Errorf("getKnownVariants(%q).Light = %q, want %q", tt.paletteName, variants.Light, tt.expectLight)
			}
			if variants.Dark != tt.expectDark {
				t.Errorf("getKnownVariants(%q).Dark = %q, want %q", tt.paletteName, variants.Dark, tt.expectDark)
			}
		})
	}
}

func TestDetectVariants(t *testing.T) {
	// Test with standard palettes (light/dark suffix pattern)
	tests := []struct {
		name        string
		palette     string
		expectLight string
		expectDark  string
	}{
		{"everforest base", "everforest-light", "everforest-light", "everforest-dark"},
		{"everforest dark", "everforest-dark", "everforest-light", "everforest-dark"},
		{"default light", "default-light", "default-light", "default-dark"},
		{"default dark", "default-dark", "default-light", "default-dark"},
		// Known mappings
		{"catppuccin-latte", "catppuccin-latte", "catppuccin-latte", "catppuccin-mocha"},
		{"catppuccin-mocha", "catppuccin-mocha", "catppuccin-latte", "catppuccin-mocha"},
		{"dracula (dark only)", "dracula", "", "dracula"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variants := DetectVariants(tt.palette)
			if variants.Light != tt.expectLight {
				t.Errorf("DetectVariants(%q).Light = %q, want %q", tt.palette, variants.Light, tt.expectLight)
			}
			if variants.Dark != tt.expectDark {
				t.Errorf("DetectVariants(%q).Dark = %q, want %q", tt.palette, variants.Dark, tt.expectDark)
			}
		})
	}
}

func TestGetEffectivePalettes(t *testing.T) {
	tests := []struct {
		name         string
		palette      string
		paletteLight string
		paletteDark  string
		wantLight    string
		wantDark     string
	}{
		{
			name:      "auto-detect from base name",
			palette:   "everforest-light",
			wantLight: "everforest-light",
			wantDark:  "everforest-dark",
		},
		{
			name:         "explicit overrides",
			palette:      "default-light",
			paletteLight: "nord-light",
			paletteDark:  "dracula",
			wantLight:    "nord-light",
			wantDark:     "dracula",
		},
		{
			name:         "only light override",
			palette:      "everforest-dark",
			paletteLight: "nord-light",
			wantLight:    "nord-light",
			wantDark:     "everforest-dark",
		},
		{
			name:        "only dark override",
			palette:     "everforest-light",
			paletteDark: "dracula",
			wantLight:   "everforest-light",
			wantDark:    "dracula",
		},
		{
			name:      "catppuccin auto-mapping",
			palette:   "catppuccin-mocha",
			wantLight: "catppuccin-latte",
			wantDark:  "catppuccin-mocha",
		},
		{
			name:      "dracula dark-only fallback",
			palette:   "dracula",
			wantLight: "dracula",
			wantDark:  "dracula",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLight, gotDark := GetEffectivePalettes(tt.palette, tt.paletteLight, tt.paletteDark)
			if gotLight != tt.wantLight {
				t.Errorf("GetEffectivePalettes() light = %q, want %q", gotLight, tt.wantLight)
			}
			if gotDark != tt.wantDark {
				t.Errorf("GetEffectivePalettes() dark = %q, want %q", gotDark, tt.wantDark)
			}
		})
	}
}

func TestPaletteVariantsStruct(t *testing.T) {
	v := PaletteVariants{
		Light: "test-light",
		Dark:  "test-dark",
		Base:  "test",
	}

	if v.Light != "test-light" {
		t.Errorf("Light = %q, want %q", v.Light, "test-light")
	}
	if v.Dark != "test-dark" {
		t.Errorf("Dark = %q, want %q", v.Dark, "test-dark")
	}
	if v.Base != "test" {
		t.Errorf("Base = %q, want %q", v.Base, "test")
	}
}
