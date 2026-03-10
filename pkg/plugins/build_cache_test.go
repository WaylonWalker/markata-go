package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
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
