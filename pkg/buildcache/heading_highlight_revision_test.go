package buildcache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestHeadingHighlightRevision_LegacyMetadata(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"version":3,"posts":{"legacy.md":{"input_hash":"unchanged","output_path":"legacy/index.html","template":"post.html"}}}`
	if err := os.WriteFile(filepath.Join(dir, CacheFileName), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cache.GetHeadingHighlightRevision("legacy.md") != 0 || cache.ShouldRebuild("legacy.md", "unchanged", "post.html") {
		t.Fatal("additive metadata invalidated a legacy post")
	}
}

func TestHeadingHighlightRevision_ConcurrentAccess(t *testing.T) {
	cache := New(t.TempDir())
	cache.Posts["post.md"] = &PostCache{}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				cache.SetHeadingHighlightRevision("post.md", 1)
				if cache.GetHeadingHighlightRevision("post.md") != 1 {
					t.Error("concurrent getter did not observe recorded revision")
				}
			}
		})
	}
	workers.Wait()
}

func TestHeadingHighlightRevision_LiveMetadataAndPersistence(t *testing.T) {
	dir := t.TempDir()
	cache := New(dir)
	cache.SetHeadingHighlightRevision("missing.md", 1)
	if len(cache.Posts) != 0 || cache.dirty || cache.GetHeadingHighlightRevision("missing.md") != 0 {
		t.Fatal("missing entry must not be created or dirty the cache")
	}
	cache.Posts["post.md"] = &PostCache{InputHash: "input"}
	cache.SetHeadingHighlightRevision("post.md", 1)
	if !cache.dirty || cache.GetHeadingHighlightRevision("post.md") != 1 {
		t.Fatal("changed live revision was not recorded")
	}
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}
	cache.SetHeadingHighlightRevision("post.md", 1)
	if cache.dirty {
		t.Fatal("unchanged revision dirtied cache")
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != CacheVersion || loaded.GetHeadingHighlightRevision("post.md") != 1 {
		t.Fatal("additive revision did not survive existing cache format")
	}
	loaded.SetNavPreviewHash("new navigation")
	loaded.dirty = false
	loaded.SetHeadingHighlightRevision("post.md", 2)
	if loaded.GetHeadingHighlightRevision("post.md") != 0 || loaded.dirty || loaded.stalePosts["post.md"].HeadingHighlightRevision != 1 {
		t.Fatal("revision setter modified stale ownership metadata")
	}
}
