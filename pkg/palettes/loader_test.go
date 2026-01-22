package palettes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltin(t *testing.T) {
	// Test loading a known built-in palette
	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes available")
	}

	p, err := LoadBuiltin(names[0])
	if err != nil {
		t.Fatalf("LoadBuiltin(%q) error = %v", names[0], err)
	}

	if p.Name == "" {
		t.Error("Loaded palette has empty name")
	}
	if p.Source != "built-in" {
		t.Errorf("Loaded palette source = %q, want %q", p.Source, "built-in")
	}
}

func TestLoadBuiltin_NotFound(t *testing.T) {
	_, err := LoadBuiltin("nonexistent-palette-xyz")
	if err == nil {
		t.Error("LoadBuiltin() should return error for nonexistent palette")
	}
}

func TestDiscoverBuiltin(t *testing.T) {
	infos := DiscoverBuiltin()

	// We should have at least some built-in palettes
	if len(infos) == 0 {
		t.Skip("No built-in palettes discovered")
	}

	// Verify each info has required fields
	for _, info := range infos {
		if info.Name == "" {
			t.Error("Discovered palette has empty name")
		}
		if info.Variant != VariantLight && info.Variant != VariantDark {
			t.Errorf("Discovered palette %q has invalid variant: %q", info.Name, info.Variant)
		}
		if info.Source != "built-in" {
			t.Errorf("Discovered palette %q source = %q, want %q", info.Name, info.Source, "built-in")
		}
	}
}

func TestHasBuiltin(t *testing.T) {
	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes available")
	}

	// Should find existing palette
	if !HasBuiltin(names[0]) {
		t.Errorf("HasBuiltin(%q) = false, want true", names[0])
	}

	// Should not find nonexistent palette
	if HasBuiltin("nonexistent-palette-xyz") {
		t.Error("HasBuiltin(nonexistent) = true, want false")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary palette file
	tmpDir := t.TempDir()
	paletteFile := filepath.Join(tmpDir, "test-palette.toml")

	content := `
[palette]
name = "Test Palette"
variant = "dark"
author = "Test"

[palette.colors]
red = "#ff0000"
blue = "#0000ff"

[palette.semantic]
primary = "red"
secondary = "blue"
`

	if err := os.WriteFile(paletteFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test palette: %v", err)
	}

	p, err := LoadFromFile(paletteFile)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if p.Name != "Test Palette" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Palette")
	}
	if p.Variant != VariantDark {
		t.Errorf("Variant = %q, want %q", p.Variant, VariantDark)
	}
	if p.Author != "Test" {
		t.Errorf("Author = %q, want %q", p.Author, "Test")
	}
	if len(p.Colors) != 2 {
		t.Errorf("Colors count = %d, want 2", len(p.Colors))
	}
	if len(p.Semantic) != 2 {
		t.Errorf("Semantic count = %d, want 2", len(p.Semantic))
	}
}

func TestLoadFromFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid TOML",
			content: "this is not valid toml [[[",
		},
		{
			name: "missing name",
			content: `
[palette]
variant = "dark"
[palette.colors]
red = "#ff0000"
`,
		},
		{
			name: "invalid variant",
			content: `
[palette]
name = "Test"
variant = "invalid"
[palette.colors]
red = "#ff0000"
`,
		},
		{
			name: "invalid hex color",
			content: `
[palette]
name = "Test"
variant = "dark"
[palette.colors]
bad = "not-a-hex"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paletteFile := filepath.Join(tmpDir, tt.name+".toml")
			if err := os.WriteFile(paletteFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			_, err := LoadFromFile(paletteFile)
			if err == nil {
				t.Errorf("LoadFromFile() should return error for %s", tt.name)
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	// Create temp directory with a palette
	tmpDir := t.TempDir()
	palettesDir := filepath.Join(tmpDir, "palettes")
	if err := os.MkdirAll(palettesDir, 0755); err != nil {
		t.Fatalf("Failed to create palettes dir: %v", err)
	}

	content := `
[palette]
name = "Project Palette"
variant = "light"
[palette.colors]
white = "#ffffff"
[palette.semantic]
bg = "white"
`
	if err := os.WriteFile(filepath.Join(palettesDir, "project.toml"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write palette: %v", err)
	}

	loader := NewLoaderWithPaths([]string{palettesDir})

	p, err := loader.Load("project")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if p.Name != "Project Palette" {
		t.Errorf("Name = %q, want %q", p.Name, "Project Palette")
	}
}

func TestLoader_Discover(t *testing.T) {
	// Create temp directory with palettes
	tmpDir := t.TempDir()
	palettesDir := filepath.Join(tmpDir, "palettes")
	if err := os.MkdirAll(palettesDir, 0755); err != nil {
		t.Fatalf("Failed to create palettes dir: %v", err)
	}

	// Create two test palettes
	for _, name := range []string{"dark", "light"} {
		content := `
[palette]
name = "` + name + ` palette"
variant = "` + name + `"
[palette.colors]
base = "#000000"
`
		if err := os.WriteFile(filepath.Join(palettesDir, name+".toml"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write palette: %v", err)
		}
	}

	loader := NewLoaderWithPaths([]string{palettesDir})
	infos, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find at least our 2 palettes (may include built-ins)
	found := 0
	for _, info := range infos {
		if info.Name == "dark palette" || info.Name == "light palette" {
			found++
		}
	}

	if found != 2 {
		t.Errorf("Found %d test palettes, want 2", found)
	}
}

func TestLoader_DiscoverByVariant(t *testing.T) {
	// Create temp directory with palettes
	tmpDir := t.TempDir()
	palettesDir := filepath.Join(tmpDir, "palettes")
	if err := os.MkdirAll(palettesDir, 0755); err != nil {
		t.Fatalf("Failed to create palettes dir: %v", err)
	}

	// Create dark and light palettes
	darkContent := `
[palette]
name = "Dark Theme"
variant = "dark"
[palette.colors]
base = "#000000"
`
	lightContent := `
[palette]
name = "Light Theme"
variant = "light"
[palette.colors]
base = "#ffffff"
`
	if err := os.WriteFile(filepath.Join(palettesDir, "dark.toml"), []byte(darkContent), 0644); err != nil {
		t.Fatalf("Failed to write palette: %v", err)
	}
	if err := os.WriteFile(filepath.Join(palettesDir, "light.toml"), []byte(lightContent), 0644); err != nil {
		t.Fatalf("Failed to write palette: %v", err)
	}

	loader := NewLoaderWithPaths([]string{palettesDir})

	// Test filtering by dark variant
	darkInfos, err := loader.DiscoverByVariant(VariantDark)
	if err != nil {
		t.Fatalf("DiscoverByVariant(dark) error = %v", err)
	}

	for _, info := range darkInfos {
		if info.Variant != VariantDark {
			t.Errorf("DiscoverByVariant(dark) returned palette with variant %q", info.Variant)
		}
	}
}

func TestNormalizeFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Catppuccin Mocha", "catppuccin-mocha"},
		{"Nord Dark", "nord-dark"},
		{"MY_THEME", "my-theme"},
		{"already-normalized", "already-normalized"},
		{"MixedCase_and Spaces", "mixedcase-and-spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeFileName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeFileName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
