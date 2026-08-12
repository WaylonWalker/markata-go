package renderingrecipe

import (
	"bytes"
	"testing"
)

func TestCompile_CanonicalBundleHasRealHashes(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	one, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	if one.Manifest.SemanticHash == "" || one.Manifest.RecipeHash == "" {
		t.Fatal("missing hashes")
	}
	if one.Manifest.RecipeHash != two.Manifest.RecipeHash {
		t.Fatal("recipe hash is not deterministic")
	}
	for _, asset := range one.Manifest.Assets {
		if len(asset.SHA256) != 64 || !bytes.Equal(one.Assets[asset.Path], two.Assets[asset.Path]) {
			t.Fatalf("asset %s is not deterministic", asset.Path)
		}
	}
}

func TestCanonicalJSON_RejectsDuplicateKeys(t *testing.T) {
	var value map[string]any
	if err := decodeStrict([]byte(`{"a":1,"a":2}`), &value); err == nil {
		t.Fatal("duplicate key accepted")
	}
}

func TestLoadCanonicalTheme_RejectsObsoleteFields(t *testing.T) {
	if _, err := normalize(map[string]any{"theme": map[string]any{"motif": map[string]any{"scale": 1}}}); err == nil {
		t.Fatal("obsolete motif.scale accepted")
	}
}

func TestCanonicalJSON_PreservesZeroAndUnicode(t *testing.T) {
	a, err := canonicalJSON(map[string]any{"zero": 0, "text": "café"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := canonicalJSON(map[string]any{"text": "café", "zero": 0})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) || !bytes.Contains(a, []byte(`"zero":0`)) || !bytes.Contains(a, []byte("café")) {
		t.Fatalf("canonical output = %s", a)
	}
}

func TestHashVectors_AssetAndFontChangesChangeRecipe(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	one, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	if AssetHash([]byte("a")) == AssetHash([]byte("b")) {
		t.Fatal("asset bytes share hash")
	}
	identity := one.Manifest
	identity.RecipeHash = ""
	identity.Fonts[0].SHA256 = "0" + identity.Fonts[0].SHA256[1:]
	encoded, err := canonicalJSON(identity)
	if err != nil {
		t.Fatal(err)
	}
	if digest([]byte(recipeDomain), encoded) == one.Manifest.RecipeHash {
		t.Fatal("font digest did not change recipe identity")
	}
}

func TestHashVectors_CanonicalFixture(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	semantic, err := SemanticHash(theme)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	if semantic != "056a4364f2fb7144074dfb362b1753b354247df03b737d6ed17240c7bc65b753" {
		t.Fatalf("semantic hash = %s", semantic)
	}
	if bundle.Manifest.RecipeHash != "945a7f7d03a466b20f0db233c8834cfaa4c06e354748e2612a9ebbc1c9d31581" {
		t.Fatalf("recipe hash = %s", bundle.Manifest.RecipeHash)
	}
	if AssetHash([]byte(`x`)) != "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881" {
		t.Fatalf("asset vector changed")
	}
}
