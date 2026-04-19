package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestConfigFilesHash_ChangesWhenOverlayChanges(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	overlay := filepath.Join(dir, "tailwind.toml")

	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}
	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = true\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay) error = %v", err)
	}

	first := buildcache.ContentHash(configFilesHash([]string{base, overlay}))

	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = false\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay update) error = %v", err)
	}

	second := buildcache.ContentHash(configFilesHash([]string{base, overlay}))
	if first == second {
		t.Fatal("expected config hash to change when overlay config changes")
	}
}

func TestConfigFilesHash_IsStableAcrossPathFormsAndOrder(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	overlay := filepath.Join(dir, "tailwind.toml")

	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}
	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = true\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay) error = %v", err)
	}

	first := buildcache.ContentHash(configFilesHash([]string{overlay, base, ""}))
	second := buildcache.ContentHash(configFilesHash([]string{filepath.Clean(base), filepath.Clean(overlay)}))
	third := buildcache.ContentHash(configFilesHash([]string{base, overlay}))

	if first != second || second != third {
		t.Fatalf("expected config hash to be stable across path forms and ordering: %q %q %q", first, second, third)
	}
}

func TestConfigHashInput_IsStableForEquivalentConfig(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}

	config := &lifecycle.Config{
		ContentDir:   ".",
		OutputDir:    "output",
		GlobPatterns: []string{"**/*.md"},
		Extra: map[string]interface{}{
			"title":       "Test Site",
			"url":         "https://example.com",
			"config_path": base,
			"config_paths": []string{
				base,
			},
		},
	}

	first := buildcache.ContentHash(configHashInput(config, []string{base}))
	second := buildcache.ContentHash(configHashInput(config, []string{filepath.Clean(base)}))
	if first != second {
		t.Fatalf("expected effective config hash to be stable for equivalent config: %q != %q", first, second)
	}
}
