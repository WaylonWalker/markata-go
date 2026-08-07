package fontpacks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCatalog(t *testing.T) *Catalog {
	t.Helper()
	return &Catalog{
		SystemStacks: map[string]SystemStack{"sans": {CSS: "system-ui, sans-serif"}, "mono": {CSS: "monospace"}},
		SubsetProfiles: map[string]SubsetProfile{
			"prose-core": {Unicode: []string{"U+0020-007E"}},
			"latin-ext":  {Unicode: []string{"U+0100-024F"}},
		},
		FontSources: map[string]FontSource{"demo": {Provider: "test", Family: "Demo"}},
		FontPacks: map[string]FontPack{
			"system":  {Performance: Performance{Class: "zero-download"}, Roles: map[string]Role{"body": {Stack: "sans"}}},
			"bundled": {Performance: Performance{Class: "bundled"}, Roles: map[string]Role{"body": {Source: "demo", Tier: "prose-core"}}},
		},
		Aliases: map[string]string{"default": "system"},
	}
}

func TestResolvePackAliasAndSystemHasNoAssets(t *testing.T) {
	c := testCatalog(t)
	name, pack, err := c.ResolvePack("default")
	if err != nil || name != "system" || pack.Performance.Class != "zero-download" {
		t.Fatalf("resolve alias = %q, %#v, %v", name, pack, err)
	}
	r, err := c.Resolve("default", t.TempDir(), t.TempDir(), "<p>Hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Assets) != 0 || strings.Contains(r.CSS, "@font-face") {
		t.Fatalf("system resolution emitted assets or font-face: %#v\n%s", r.Assets, r.CSS)
	}
}

func TestRequiredTiersSelectExtendedAndFull(t *testing.T) {
	c := testCatalog(t)
	_, pack, _ := c.ResolvePack("bundled")
	if got := c.RequiredTiers(pack, "<p>Hello</p>")["demo"]; !got["prose-core"] || got["latin-ext"] {
		t.Fatalf("English tiers = %#v", got)
	}
	got := c.RequiredTiers(pack, "<p>Ā</p>")["demo"]
	if !got["latin-ext"] {
		t.Fatalf("extended tiers = %#v", got)
	}
	got = c.RequiredTiers(pack, "<p>Ж</p>")["demo"]
	if !got["full"] || len(got) != 1 {
		t.Fatalf("unsupported tiers = %#v", got)
	}
}

func TestResolveDeduplicatesSourceTierAndCopiesAsset(t *testing.T) {
	c := testCatalog(t)
	c.FontPacks["bundled"] = FontPack{Performance: Performance{Class: "bundled"}, Roles: map[string]Role{
		"body": {Source: "demo", Tier: "prose-core"}, "heading": {Source: "demo", Tier: "prose-core"},
	}}
	root := t.TempDir()
	family := filepath.Join(root, "demo")
	if err := os.MkdirAll(family, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "schema: markata.font/v1\nid: demo\nfamily: Demo\ntiers:\n  prose-core:\n    file: demo.woff2\n    profile: prose-core\n"
	if err := os.WriteFile(filepath.Join(family, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "demo.woff2"), []byte("woff2"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := c.Resolve("bundled", root, t.TempDir(), "<p>Hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Assets) != 1 {
		t.Fatalf("assets = %d, want 1", len(r.Assets))
	}
}

func TestBuiltinFieldNotebookResolvesStableBaseTiers(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "markata-fontpacks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Resolve("field-notebook", filepath.Join("..", "..", "internal", "fontcatalog"), t.TempDir(), "<article><h1>Hello</h1><p>Ordinary prose.</p></article>")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Assets) != 5 {
		t.Fatalf("assets = %d, want five deduplicated family/tier pairs", len(r.Assets))
	}
	if strings.Contains(r.CSS, "full.woff2") {
		t.Fatalf("ordinary English selected full tiers:\n%s", r.CSS)
	}
}

func TestResolveManyGeneratesPerPackRoleSelectors(t *testing.T) {
	c := testCatalog(t)
	root := t.TempDir()
	family := filepath.Join(root, "demo")
	if err := os.MkdirAll(family, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "schema: markata.font/v1\nid: demo\nfamily: Demo\ntiers:\n  prose-core:\n    file: demo.woff2\n    profile: prose-core\n"
	if err := os.WriteFile(filepath.Join(family, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "demo.woff2"), []byte("wOF2"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.FontPacks["alternate"] = FontPack{Performance: Performance{Class: "bundled"}, Roles: map[string]Role{"body": {Source: "demo", Tier: "prose-core"}}}
	resolved, err := c.ResolveMany([]string{"system", "alternate"}, root, "<p>Hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resolved.CSS, `[data-fontpack="system"]`) || !strings.Contains(resolved.CSS, `[data-fontpack="alternate"]`) {
		t.Fatalf("missing per-pack selectors:\n%s", resolved.CSS)
	}
	if len(resolved.Assets) != 1 {
		t.Fatalf("assets = %d, want one deduplicated asset", len(resolved.Assets))
	}
}
