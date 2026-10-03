package buildcache

import (
	"os"
	"reflect"
	"sync"
	"testing"
)

func TestInvalidateFullHTML_PreservesMetadataAndPersistsRemoval(t *testing.T) {
	dir := t.TempDir()
	cache := New(dir)
	cache.MarkRebuiltWithSlug("private.md", "private", "input", "output/private/index.html", "post.html")
	cache.MarkRebuiltWithSlug("other.md", "other", "input", "output/other/index.html", "post.html")
	cache.UpdatePostSemanticHashes("private.md", "feed", "tag", "garden")
	cache.SetHeadingHighlightRevision("private.md", 1)
	cache.SetLocalPreviewHash("private.md", "preview")
	if err := cache.CacheEncryptedHTML("private.md", "encrypted-hash", "encrypted-wrapper"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"private.md", "other.md"} {
		if err := cache.CacheFullHTML(path, "old full page"); err != nil {
			t.Fatal(err)
		}
	}
	expected := *cache.Posts["private.md"]
	oldPath := expected.FullHTMLPath
	expected.FullHTMLPath = ""
	if cache.GetCachedFullHTML("private.md") != "old full page" {
		t.Fatal("old full page not in memory before invalidation")
	}
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}
	cache.InvalidateFullHTML("missing.md")
	if cache.dirty || cache.Posts["missing.md"] != nil {
		t.Fatal("missing-entry invalidation created or dirtied metadata")
	}
	cache.InvalidateFullHTML("private.md")
	if cache.GetCachedFullHTML("private.md") != "" ||
		!reflect.DeepEqual(*cache.Posts["private.md"], expected) {
		t.Fatal("invalidation did not clear only the full-page reference")
	}
	if cache.GetCachedFullHTML("other.md") != "old full page" {
		t.Fatal("invalidation removed another post's shared cache hit")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("invalidation removed old cache file: %v", err)
	}
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetCachedFullHTML("private.md") != "" ||
		!reflect.DeepEqual(*reloaded.Posts["private.md"], expected) ||
		reloaded.GetCachedEncryptedHTML("private.md", "encrypted-hash") != "encrypted-wrapper" {
		t.Fatal("reload restored old reference or lost other metadata")
	}
	reloaded.InvalidateFullHTML("private.md")
	if reloaded.dirty {
		t.Fatal("already-empty reference dirtied cache")
	}
}

func TestInvalidateFullHTML_ConcurrentGetAndWrite(t *testing.T) {
	cache := New(t.TempDir())
	cache.MarkRebuilt("post.md", "input", "", "post.html")
	if err := cache.CacheFullHTML("post.md", "full page"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.GetCachedFullHTML("post.md")
			}
		}()
	}
	for i := 0; i < 20; i++ {
		cache.InvalidateFullHTML("post.md")
		if err := cache.CacheFullHTML("post.md", "full page"); err != nil {
			t.Error(err)
			break
		}
	}
	wg.Wait()
	cache.InvalidateFullHTML("post.md")
	if cache.GetCachedFullHTML("post.md") != "" {
		t.Fatal("final invalidation restored stale memory cache")
	}
}
