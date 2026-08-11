package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalHeadingsFixtureLoadsWithContractValues(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "canonical-headings", "theme.toml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromString(string(data), FormatTOML)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme.ContractVersion != 1 || cfg.Theme.Palette != "ayu-dark" || cfg.Theme.Aesthetic != "balanced" || cfg.Theme.Fontpack != "brush" {
		t.Fatalf("canonical theme identity did not load: %#v", cfg.Theme)
	}
	if cfg.Theme.Texture.Kind != "screenprint" || cfg.Theme.Texture.ColorMix != .211 || cfg.Theme.Texture.Scale != .920 || cfg.Theme.Texture.Scope != "quiet" {
		t.Fatalf("canonical texture did not load: %#v", cfg.Theme.Texture)
	}
	if cfg.Theme.HeadingTexture.Kind != "splatter" || cfg.Theme.HeadingTexture.ColorMix != .290 || cfg.Theme.HeadingTexture.Scale != .250 {
		t.Fatalf("canonical heading texture did not load: %#v", cfg.Theme.HeadingTexture)
	}
	motif := cfg.Theme.Motif
	if motif.Kind != "block-w" || motif.Size != "71px" || motif.Gap != "11px" || motif.RowOffset != .550 || motif.Wobble != .330 || motif.Scatter != .330 || motif.Layer != "over" || motif.Color != "ink" || motif.ColorMix != .010 || motif.URL != "https://waylonwalker.com/w.svg" {
		t.Fatalf("canonical motif did not load: %#v", motif)
	}
	if cfg.Theme.Variables["--content-width"] != "68ch" {
		t.Fatalf("content width did not propagate: %#v", cfg.Theme.Variables)
	}
}

func TestCanonicalHeadingsMarkdownCopyMatchesSharedFixture(t *testing.T) {
	shared, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "canonical-headings", "test-headings.md"))
	if err != nil {
		t.Fatal(err)
	}
	local, err := os.ReadFile(filepath.Join("..", "..", "rendering-fixtures", "test-headings.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(shared) != string(local) {
		t.Fatal("markata-go headings copy drifted from the shared fixture")
	}
}
