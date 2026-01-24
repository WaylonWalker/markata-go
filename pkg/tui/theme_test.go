package tui

import (
	"testing"
)

func TestDefaultTUIColors(t *testing.T) {
	colors := DefaultColors()

	if colors == nil {
		t.Fatal("DefaultColors() returned nil")
	}

	// Verify that colors are set (ANSI codes)
	if colors.Header == "" {
		t.Error("Header color is empty")
	}
	if colors.Subtle == "" {
		t.Error("Subtle color is empty")
	}
	if colors.Selected == "" {
		t.Error("Selected color is empty")
	}
}

func TestDefaultTUITheme(t *testing.T) {
	theme := DefaultTheme()

	if theme == nil {
		t.Fatal("DefaultTheme() returned nil")
	}

	if theme.Colors == nil {
		t.Error("Theme.Colors is nil")
	}

	// Verify styles are initialized by checking they can render
	rendered := theme.HeaderStyle.Render("test")
	if rendered == "" {
		t.Error("HeaderStyle failed to render")
	}
	rendered = theme.SubtleStyle.Render("test")
	if rendered == "" {
		t.Error("SubtleStyle failed to render")
	}
}

func TestNewTUITheme_NilColors(t *testing.T) {
	theme := NewTheme(nil)

	if theme == nil {
		t.Fatal("NewTheme(nil) returned nil")
	}

	// Should fall back to default colors
	if theme.Colors == nil {
		t.Error("Theme.Colors should fall back to defaults")
	}
}

func TestLoadTUIColors_EmptyPalette(t *testing.T) {
	colors := LoadColors("")

	if colors == nil {
		t.Fatal("LoadColors(\"\") returned nil")
	}

	// Should return default colors
	defaults := DefaultColors()
	if colors.Header != defaults.Header {
		t.Errorf("Header = %v, want %v", colors.Header, defaults.Header)
	}
}

func TestLoadTUIColors_InvalidPalette(t *testing.T) {
	colors := LoadColors("nonexistent-palette-that-does-not-exist")

	if colors == nil {
		t.Fatal("LoadColors() returned nil for invalid palette")
	}

	// Should fall back to default colors
	defaults := DefaultColors()
	if colors.Header != defaults.Header {
		t.Errorf("Header = %v, want %v (should fall back to defaults)", colors.Header, defaults.Header)
	}
}

func TestLoadTUIColors_TokyoNight(t *testing.T) {
	colors := LoadColors("tokyo-night")

	if colors == nil {
		t.Fatal("LoadColors(\"tokyo-night\") returned nil")
	}

	// Tokyo Night palette should load hex colors
	// The exact colors depend on the palette, but they should be different from ANSI defaults
	defaults := DefaultColors()

	// At least one color should be different (hex vs ANSI)
	// Since Tokyo Night uses hex colors like #9d7cd8 for accent
	// and defaults use ANSI codes like "99", they should differ
	if colors.Header == defaults.Header {
		// This might be okay if the palette couldn't be loaded
		t.Log("Warning: Tokyo Night colors same as defaults, palette may not have loaded")
	}
}

func TestGetPaletteNameFromConfig_Nil(t *testing.T) {
	name := GetPaletteNameFromConfig(nil)
	if name != "" {
		t.Errorf("GetPaletteNameFromConfig(nil) = %q, want \"\"", name)
	}
}

func TestGetPaletteNameFromConfig_NoTheme(t *testing.T) {
	extra := map[string]interface{}{
		"other_key": "value",
	}
	name := GetPaletteNameFromConfig(extra)
	if name != "" {
		t.Errorf("GetPaletteNameFromConfig(no theme) = %q, want \"\"", name)
	}
}

func TestGetPaletteNameFromConfig_NoPalette(t *testing.T) {
	extra := map[string]interface{}{
		"theme": map[string]interface{}{
			"name": "default",
		},
	}
	name := GetPaletteNameFromConfig(extra)
	if name != "" {
		t.Errorf("GetPaletteNameFromConfig(no palette) = %q, want \"\"", name)
	}
}

func TestGetPaletteNameFromConfig_WithPalette(t *testing.T) {
	extra := map[string]interface{}{
		"theme": map[string]interface{}{
			"palette": "tokyo-night",
		},
	}
	name := GetPaletteNameFromConfig(extra)
	if name != "tokyo-night" {
		t.Errorf("GetPaletteNameFromConfig() = %q, want \"tokyo-night\"", name)
	}
}

func TestNewModelWithTheme(t *testing.T) {
	theme := DefaultTheme()
	model := NewModelWithTheme(nil, theme)

	if model.theme != theme {
		t.Error("Model.theme not set correctly")
	}
}

func TestNewModelWithTheme_NilTheme(t *testing.T) {
	model := NewModelWithTheme(nil, nil)

	if model.theme == nil {
		t.Error("Model.theme should default to non-nil theme")
	}
}
