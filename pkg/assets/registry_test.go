package assets

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	assets := Registry()
	if len(assets) == 0 {
		t.Error("expected non-empty registry")
	}

	// Verify it returns a copy
	assets[0].Name = "modified"
	original := Registry()
	if original[0].Name == "modified" {
		t.Error("Registry should return a copy, not the original slice")
	}
}

func TestGetAsset(t *testing.T) {
	tests := []struct {
		name      string
		assetName string
		wantNil   bool
	}{
		{"existing asset", "glightbox-js", false},
		{"existing css", "glightbox-css", false},
		{"htmx", "htmx", false},
		{"mermaid", "mermaid", false},
		{"non-existent", "non-existent-asset", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := GetAsset(tt.assetName)
			if tt.wantNil && asset != nil {
				t.Errorf("expected nil for %s", tt.assetName)
			}
			if !tt.wantNil && asset == nil {
				t.Errorf("expected non-nil for %s", tt.assetName)
			}
			if asset != nil && asset.Name != tt.assetName {
				t.Errorf("expected Name %s, got %s", tt.assetName, asset.Name)
			}
		})
	}
}

func TestGetAssetsByType(t *testing.T) {
	jsAssets := GetAssetsByType("js")
	if len(jsAssets) == 0 {
		t.Error("expected at least one JS asset")
	}
	for _, a := range jsAssets {
		if a.Type != "js" {
			t.Errorf("expected type js, got %s", a.Type)
		}
	}

	cssAssets := GetAssetsByType("css")
	if len(cssAssets) == 0 {
		t.Error("expected at least one CSS asset")
	}
	for _, a := range cssAssets {
		if a.Type != "css" {
			t.Errorf("expected type css, got %s", a.Type)
		}
	}
}

func TestAssetGroups(t *testing.T) {
	groups := AssetGroups()
	if len(groups) == 0 {
		t.Error("expected non-empty groups")
	}

	// Check for expected groups
	expectedGroups := []string{"glightbox", "htmx", "mermaid", "chartjs", "cal-heatmap", "d3", "popper"}
	for _, name := range expectedGroups {
		if _, ok := groups[name]; !ok {
			t.Errorf("expected group %s to exist", name)
		}
	}

	// Verify glightbox has both CSS and JS
	glightbox := groups["glightbox"]
	if len(glightbox) != 2 {
		t.Errorf("expected 2 glightbox assets, got %d", len(glightbox))
	}
}

func TestAssetNames(t *testing.T) {
	names := AssetNames()
	if len(names) == 0 {
		t.Error("expected non-empty names list")
	}

	// Check for expected names
	expectedNames := map[string]bool{
		"glightbox-js":  true,
		"glightbox-css": true,
		"htmx":          true,
		"mermaid":       true,
		"chartjs":       true,
	}

	for _, name := range names {
		delete(expectedNames, name)
	}

	if len(expectedNames) > 0 {
		for name := range expectedNames {
			t.Errorf("expected asset name %s not found", name)
		}
	}
}

func TestAssetURLsAreValid(t *testing.T) {
	for _, asset := range Registry() {
		if asset.URL == "" {
			t.Errorf("asset %s has empty URL", asset.Name)
		}
		if asset.LocalPath == "" {
			t.Errorf("asset %s has empty LocalPath", asset.Name)
		}
		if asset.Type == "" {
			t.Errorf("asset %s has empty Type", asset.Name)
		}
	}
}
