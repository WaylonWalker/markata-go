package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestGlobPlugin_FastBuildRescansMovedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cache := buildcache.New(t.TempDir())

	oldPath := filepath.Join(tmpDir, "old.md")
	newPath := filepath.Join(tmpDir, "new.md")
	if err := os.WriteFile(oldPath, []byte("# old"), 0o600); err != nil {
		t.Fatalf("WriteFile(old.md) error = %v", err)
	}

	first := lifecycle.NewManager()
	first.Config().ContentDir = tmpDir
	first.Config().GlobPatterns = []string{"**/*.md"}
	first.Config().Extra = map[string]any{"fast_mode": true}
	first.Cache().Set("build_cache", cache)

	plugin := NewGlobPlugin()
	if err := plugin.Configure(first); err != nil {
		t.Fatalf("Configure() first build error = %v", err)
	}
	if err := plugin.Glob(first); err != nil {
		t.Fatalf("Glob() first build error = %v", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	second := lifecycle.NewManager()
	second.Config().ContentDir = tmpDir
	second.Config().GlobPatterns = []string{"**/*.md"}
	second.Config().Extra = map[string]any{"fast_mode": true}
	second.Cache().Set("build_cache", cache)

	plugin = NewGlobPlugin()
	if err := plugin.Configure(second); err != nil {
		t.Fatalf("Configure() second build error = %v", err)
	}
	if err := plugin.Glob(second); err != nil {
		t.Fatalf("Glob() second build error = %v", err)
	}

	got := second.Files()
	want := []string{"new.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	if len(got) != 1 || got[0] == "old.md" {
		t.Fatalf("stale glob cache reused after move: %v", got)
	}
}
