package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalHeadingsFixtureLoadsWithContractValues(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "canonical-headings", "theme.toml"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("shared canonical fixture is unavailable: %v", err)
		}
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
	if motif.Kind != "block-w" || motif.Size != "62px" || motif.Gap != "21px" || motif.RowOffset != .500 || motif.Wobble != .180 || motif.Scatter != .180 || motif.Layer != "over" || motif.Color != "ink" || motif.ColorMix != .010 || motif.URL != "https://waylonwalker.com/w.svg" {
		t.Fatalf("canonical motif did not load: %#v", motif)
	}
	if cfg.Theme.Variables["--content-width"] != "68ch" {
		t.Fatalf("content width did not propagate: %#v", cfg.Theme.Variables)
	}
}

func TestCanonicalHeadingsMarkdownCopyMatchesSharedFixture(t *testing.T) {
	local, err := os.ReadFile(filepath.Join("..", "..", "rendering-fixtures", "test-headings.md"))
	if err != nil {
		t.Fatal(err)
	}
	local = canonicalFixtureBytes(local)
	if got := sha256Digest(local); got != "58bc0ab1e501adbe498c2902f1e0ede5112c8deede94e8b3a278f16613e9cedb" {
		t.Fatalf("local canonical fixture hash drifted: %s", got)
	}
	shared, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "canonical-headings", "test-headings.md"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("shared canonical fixture is unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !bytes.Equal(canonicalFixtureBytes(shared), local) {
		t.Fatal("markata-go headings copy drifted from the shared fixture")
	}
}

func canonicalFixtureBytes(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}
