package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestFontpackCacheKeyChangesWithRenderedContent(t *testing.T) {
	names := []string{"system", "serif"}
	first := fontpackCacheKey("<p>one</p>", names)
	second := fontpackCacheKey("<p>two</p>", names)
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
	if !fontpackOutputCached(output, "key") {
		t.Fatal("matching marker and stylesheet were not treated as a cache hit")
	}
	if fontpackOutputCached(output, "different") {
		t.Fatal("mismatched marker was treated as a cache hit")
	}
}
