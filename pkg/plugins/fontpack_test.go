package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/fontpacks"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestConfiguredFontpackName_UsesCanonicalTheme(t *testing.T) {
	configured := &models.Config{}
	configured.Theme.Fontpack = "brush-poster"
	if got := configuredFontpackName(map[string]any{"models_config": configured, "fontpack": "system"}); got != "brush" {
		t.Fatalf("fontpack = %q", got)
	}
}

func TestMarkHTMLFontpackReplacesExactlyOneAttribute(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"exact", "<html>", `<html data-fontpack="field-notebook">`},
		{"attributes", `<html lang="en">`, `<html lang="en" data-fontpack="field-notebook">`},
		{"uppercase", `<HTML class="site">`, `<HTML class="site" data-fontpack="field-notebook">`},
		{"replace", `<html data-fontpack="old" lang="en">`, `<html data-fontpack="field-notebook" lang="en">`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := markHTMLFontpack(tt.in, "field-notebook")
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if strings.Count(strings.ToLower(got), "data-fontpack=") != 1 {
				t.Fatalf("duplicate data-fontpack attribute: %q", got)
			}
		})
	}
}

func TestMarkPostFontpackKeepsHashedStylesheet(t *testing.T) {
	html := `<html><head><link rel="stylesheet" href="/css/fonts.12345678.css"></head><body></body></html>`
	got := markPostFontpack(html, "brush")
	if strings.Count(got, "fonts.") != 1 || strings.Contains(got, `href="/css/fonts.css"`) {
		t.Fatalf("duplicate font stylesheet: %s", got)
	}
	if !strings.Contains(got, `data-fontpack="brush"`) {
		t.Fatalf("missing resolved fontpack: %s", got)
	}
}

func TestFontpackCacheKeyChangesWithRenderedContent(t *testing.T) {
	names := []string{"system", "serif"}
	catalog, err := fontpacks.BuiltinSource()
	if err != nil {
		t.Fatal(err)
	}
	first := fontpackCacheKey("<p>one</p>", names, catalog.Catalog)
	second := fontpackCacheKey("<p>two</p>", names, catalog.Catalog)
	if first == second {
		t.Fatal("fontpack cache key did not change with rendered content")
	}
}

func TestFontpackOutputCachedRequiresMatchingKeyAndStylesheet(t *testing.T) {
	output := t.TempDir()
	if fontpackOutputCached(output, "key") {
		t.Fatal("missing marker was treated as a cache hit")
	}

	if err := os.WriteFile(filepath.Join(output, fontpackCacheFile), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	if fontpackOutputCached(output, "key") {
		t.Fatal("missing stylesheet was treated as a cache hit")
	}
	if err := os.Mkdir(filepath.Join(output, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "css", "fonts.css"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(output, "assets", "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "assets", "fonts", ".markata-fonts.json"), []byte(`{"files":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if !fontpackOutputCached(output, "key") {
		t.Fatal("matching marker and stylesheet were not treated as a cache hit")
	}
	if fontpackOutputCached(output, "different") {
		t.Fatal("mismatched marker was treated as a cache hit")
	}
}

func TestFontpackPreloadURLs(t *testing.T) {
	assets := []fontpacks.Asset{
		{Source: "body", Tier: "prose-core", URL: "/assets/fonts/body-core.woff2"},
		{Source: "body", Tier: "latin-ext", URL: "/assets/fonts/body-ext.woff2"},
		{Source: "display", Tier: "full", URL: "/assets/fonts/display-full.woff2"},
		{Source: "code", Tier: "full", URL: "/assets/fonts/code-full.woff2"},
	}

	pack := fontpacks.FontPack{Roles: map[string]fontpacks.Role{
		"body": {Source: "body"}, "display": {Source: "display"},
		"heading": {Source: "body"}, "code": {Source: "code"},
	}}
	got := fontpackPreloadURLs(pack, assets)
	want := []string{"/assets/fonts/body-core.woff2", "/assets/fonts/display-full.woff2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("preloads = %v, want %v", got, want)
	}
	if got := fontpackPreloadURLs(fontpacks.FontPack{Roles: map[string]fontpacks.Role{"body": {Stack: "sans"}}}, assets); len(got) != 0 {
		t.Fatalf("system pack preloads = %v", got)
	}
	assets = append(assets, fontpacks.Asset{Source: "body", Tier: "full", URL: "/assets/fonts/body-full.woff2"})
	if got := fontpackPreloadURLs(pack, assets); got[0] != "/assets/fonts/body-full.woff2" {
		t.Fatalf("full tier must supersede subsets: %v", got)
	}
}

func TestValidFontpackPreloadCache(t *testing.T) {
	output := t.TempDir()
	catalog := &fontpacks.Catalog{FontPacks: map[string]fontpacks.FontPack{"system": {}}}
	cache := fontpackPreloadCache{Hash: "1234abcd", URLs: map[string][]string{"system": {}}}
	if !validFontpackPreloadCache(output, []string{"system"}, catalog, cache) {
		t.Fatal("empty system pack should have a valid cached preload list")
	}
	cache.URLs["system"] = []string{"/assets/fonts/missing.woff2"}
	if validFontpackPreloadCache(output, []string{"system"}, catalog, cache) {
		t.Fatal("missing font file accepted")
	}
	cache.URLs["system"] = []string{"/assets/fonts/../bad.woff2"}
	if validFontpackPreloadCache(output, []string{"system"}, catalog, cache) {
		t.Fatal("unsafe font path accepted")
	}
	cache.URLs["system"] = nil
	cache.Hash = "invalid!"
	if validFontpackPreloadCache(output, []string{"system"}, catalog, cache) {
		t.Fatal("invalid hash accepted")
	}
}
