package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalHeadingsFixtureLoadsWithContractValues(t *testing.T) {
	data := readCanonicalFixture(t, "theme.toml")
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
	if motif.Kind != "block-w" || motif.Size != "62px" || motif.Gap != "21px" || motif.RowOffset != .500 || motif.Wobble != .180 || motif.Scatter != .180 || motif.Layer != "over" || motif.Color != "ink" || motif.ColorMix != .010 || motif.URL != "https://waylonwalker.com/w.svg" {
		t.Fatalf("canonical motif did not load: %#v", motif)
	}
	if cfg.Theme.Variables["--content-width"] != "68ch" {
		t.Fatalf("content width did not propagate: %#v", cfg.Theme.Variables)
	}
}

func TestCanonicalHeadingsMarkdownCopyMatchesSharedFixture(t *testing.T) {
	shared := readCanonicalFixture(t, "test-headings.md")
	local, err := os.ReadFile(filepath.Join("..", "..", "rendering-fixtures", "test-headings.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(shared) != string(local) {
		t.Fatal("markata-go headings copy drifted from the shared fixture")
	}
}

func readCanonicalFixture(t *testing.T, name string) []byte {
	t.Helper()
	paths := []string{
		filepath.Join("..", "..", "..", "fixtures", "canonical-headings", name),
		filepath.Join("..", "..", "fixtures", "canonical-headings", name),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return data
		}
		if !os.IsNotExist(err) {
			t.Fatalf("read canonical fixture %s: %v", path, err)
		}
	}
	t.Fatalf("canonical fixture %q not found in shared workspace or repository fixtures", name)
	return nil
}
