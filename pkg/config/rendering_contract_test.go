package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRenderingContract_LegacyMigratesWithCanonicalPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "markata-go.toml")
	contents := `[markata-go]
texture_strength = "10%"
heading_texture_strength = "20%"
motif_color_distance = "1%"

[markata-go.theme]
contract_version = 1
palette = "ayu-dark"
fontpack = "brush-poster"

[markata-go.theme.texture]
kind = "screenprint"
color_mix = 0.9
scale = 1.0
scope = "all"
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := loadResolvedConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Theme.Texture.ColorMix != .9 {
		t.Fatalf("color_mix = %v", config.Theme.Texture.ColorMix)
	}
	if config.Theme.Fontpack != "brush-poster" {
		t.Fatalf("fontpack = %q", config.Theme.Fontpack)
	}
	warnings, ok := config.Extra["theme_migration_warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Fatal("expected migration warning")
	}
}

func TestLoadRenderingContract_LegacyOnlyValuesSurviveDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "markata-go.toml")
	if err := os.WriteFile(path, []byte("[markata-go]\ntexture_strength = \"10%\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := loadResolvedConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Theme.Texture.ColorMix != .1 {
		t.Fatalf("legacy color_mix = %v, want .1", config.Theme.Texture.ColorMix)
	}
}
