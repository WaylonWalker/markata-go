package buildcache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	cache := New("")
	if cache == nil {
		t.Fatal("New returned nil")
	}
	if cache.Version != CacheVersion {
		t.Errorf("Version = %d, want %d", cache.Version, CacheVersion)
	}
	if cache.Posts == nil {
		t.Error("Posts map is nil")
	}
	if cache.Graph == nil {
		t.Error("Graph is nil")
	}
	if cache.changedSlugs == nil {
		t.Error("changedSlugs map is nil")
	}
}

func TestCache_SetAndGetDependencies(t *testing.T) {
	cache := New("")

	// Set dependencies
	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})

	// Verify through graph
	deps := cache.Graph.GetDependencies("pages/post-a.md")
	if len(deps) != 2 {
		t.Errorf("GetDependencies returned %d deps, want 2", len(deps))
	}
}

func TestCache_GetAffectedPosts(t *testing.T) {
	cache := New("")

	// Set up dependencies: post-a -> post-b -> post-c
	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	cache.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	// When post-c changes, both post-a and post-b should be affected
	affected := cache.GetAffectedPosts([]string{"post-c"})
	if len(affected) != 2 {
		t.Errorf("GetAffectedPosts returned %d posts, want 2: %v", len(affected), affected)
	}
}

func TestCache_ShouldRebuildWithSlug_DependencyChanged(t *testing.T) {
	cache := New("")

	// Set up: post-a depends on post-b
	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})

	// Mark post-a as previously built
	cache.MarkRebuilt("pages/post-a.md", "hash123", "output/post-a/index.html", "post.html")

	// Mark that post-b changed this build
	cache.MarkSlugChanged("post-b")

	// Even though post-a's hash matches, it should rebuild because post-b changed
	shouldRebuild := cache.ShouldRebuildWithSlug("pages/post-a.md", "post-a", "hash123", "post.html")
	if !shouldRebuild {
		t.Error("ShouldRebuildWithSlug = false, want true (dependency changed)")
	}
}

func TestCache_ShouldRebuildWithSlug_NoDependencyChange(t *testing.T) {
	cache := New("")

	// Set up: post-a depends on post-b
	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})

	// Mark post-a as previously built
	cache.MarkRebuilt("pages/post-a.md", "hash123", "output/post-a/index.html", "post.html")

	// Don't mark post-b as changed

	// post-a should NOT rebuild (hash matches, no dependency changed)
	shouldRebuild := cache.ShouldRebuildWithSlug("pages/post-a.md", "post-a", "hash123", "post.html")
	if shouldRebuild {
		t.Error("ShouldRebuildWithSlug = true, want false (no dependency changed)")
	}
}

func TestCache_GetChangedSlugs(t *testing.T) {
	cache := New("")

	// No changes initially
	changed := cache.GetChangedSlugs()
	if len(changed) != 0 {
		t.Errorf("GetChangedSlugs = %v, want []", changed)
	}

	// Mark some slugs as changed
	cache.MarkSlugChanged("post-a")
	cache.MarkSlugChanged("post-b")

	changed = cache.GetChangedSlugs()
	if len(changed) != 2 {
		t.Errorf("GetChangedSlugs returned %d slugs, want 2", len(changed))
	}
}

func TestCache_ResetStats_ClearsChangedSlugs(t *testing.T) {
	cache := New("")

	cache.MarkSlugChanged("post-a")
	cache.MarkSlugChanged("post-b")

	cache.ResetStats()

	changed := cache.GetChangedSlugs()
	if len(changed) != 0 {
		t.Errorf("GetChangedSlugs after ResetStats = %v, want []", changed)
	}
}

func TestCache_MarkRebuiltWithSlug_TracksChange(t *testing.T) {
	cache := New("")

	cache.MarkRebuiltWithSlug("pages/post-a.md", "post-a", "hash123", "output/post-a/index.html", "post.html")

	changed := cache.GetChangedSlugs()
	if len(changed) != 1 || changed[0] != "post-a" {
		t.Errorf("GetChangedSlugs = %v, want [post-a]", changed)
	}
}

func TestCache_MarkChangedPaths(t *testing.T) {
	cache := New("")

	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	cache.MarkChangedPaths([]string{"pages/post-a.md"})

	changed := cache.GetChangedSlugs()
	if len(changed) != 1 || changed[0] != "post-a" {
		t.Errorf("GetChangedSlugs = %v, want [post-a]", changed)
	}
}

func TestCache_MarkAffectedDependents(t *testing.T) {
	cache := New("")

	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	cache.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	cache.MarkAffectedDependents([]string{"post-c"})

	changed := cache.GetChangedSlugs()
	if len(changed) != 2 {
		t.Fatalf("GetChangedSlugs returned %d slugs, want 2", len(changed))
	}
	want := map[string]bool{"post-a": true, "post-b": true}
	for _, slug := range changed {
		if !want[slug] {
			t.Errorf("unexpected changed slug %q", slug)
		}
	}
}

func TestCache_SaveLoad_PreservesGraph(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".markata")

	// Create and populate cache
	cache := New(cacheDir)
	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})
	cache.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})
	cache.MarkRebuilt("pages/post-a.md", "hash-a", "output/post-a/index.html", "post.html")

	// Save
	if err := cache.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into new cache
	loaded, err := Load(cacheDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify graph was preserved
	deps := loaded.Graph.GetDependencies("pages/post-a.md")
	if len(deps) != 2 {
		t.Errorf("Loaded cache has %d deps for post-a, want 2", len(deps))
	}

	// Verify PathToSlug was preserved (needed for transitive lookups)
	slug := loaded.Graph.PathToSlug["pages/post-a.md"]
	if slug != "post-a" {
		t.Errorf("PathToSlug[post-a.md] = %q, want %q", slug, "post-a")
	}

	// Verify Dependents was rebuilt
	if !loaded.Graph.HasDependents("post-c") {
		t.Error("post-c should have dependents after load")
	}

	// Verify affected posts calculation works after load
	affected := loaded.GetAffectedPosts([]string{"post-c"})
	if len(affected) != 2 {
		t.Errorf("GetAffectedPosts after load returned %d posts, want 2: %v", len(affected), affected)
	}
}

func TestCache_LoadMissingGraph(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".markata")

	// Create cache file manually without graph field
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	cacheFile := filepath.Join(cacheDir, CacheFileName)
	data := `{"version":1,"config_hash":"abc","templates_hash":"def","posts":{}}`
	//nolint:gosec // G306: Test file, 0644 is fine
	if err := os.WriteFile(cacheFile, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Load - should initialize graph
	loaded, err := Load(cacheDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Graph == nil {
		t.Error("Graph should be initialized when missing from cache file")
	}
}

func TestCache_GraphSize(t *testing.T) {
	cache := New("")

	if cache.GraphSize() != 0 {
		t.Errorf("GraphSize = %d, want 0", cache.GraphSize())
	}

	cache.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	cache.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	if cache.GraphSize() != 2 {
		t.Errorf("GraphSize = %d, want 2", cache.GraphSize())
	}
}

func TestComputeFeedMembershipHash_Empty(t *testing.T) {
	hash := ComputeFeedMembershipHash(nil)
	if hash != "" {
		t.Errorf("ComputeFeedMembershipHash(nil) = %q, want empty", hash)
	}
	hash = ComputeFeedMembershipHash([]string{})
	if hash != "" {
		t.Errorf("ComputeFeedMembershipHash([]) = %q, want empty", hash)
	}
}

func TestComputeFeedMembershipHash_Deterministic(t *testing.T) {
	// Same slugs in different order should produce the same hash
	hash1 := ComputeFeedMembershipHash([]string{"post-c", "post-a", "post-b"})
	hash2 := ComputeFeedMembershipHash([]string{"post-a", "post-b", "post-c"})
	hash3 := ComputeFeedMembershipHash([]string{"post-b", "post-c", "post-a"})

	if hash1 == "" {
		t.Fatal("ComputeFeedMembershipHash returned empty for non-empty input")
	}
	if hash1 != hash2 {
		t.Errorf("hash mismatch: %q != %q (different order should produce same hash)", hash1, hash2)
	}
	if hash1 != hash3 {
		t.Errorf("hash mismatch: %q != %q (different order should produce same hash)", hash1, hash3)
	}
}

func TestComputeFeedMembershipHash_ChangesOnMembershipChange(t *testing.T) {
	hash1 := ComputeFeedMembershipHash([]string{"post-a", "post-b"})
	hash2 := ComputeFeedMembershipHash([]string{"post-a", "post-b", "post-c"})
	hash3 := ComputeFeedMembershipHash([]string{"post-a"})

	if hash1 == hash2 {
		t.Error("hash should change when a member is added")
	}
	if hash1 == hash3 {
		t.Error("hash should change when a member is removed")
	}
}

func TestComputeFeedMembershipHash_DoesNotMutateInput(t *testing.T) {
	input := []string{"post-c", "post-a", "post-b"}
	original := make([]string, len(input))
	copy(original, input)

	ComputeFeedMembershipHash(input)

	for i, v := range input {
		if v != original[i] {
			t.Errorf("input[%d] = %q, want %q (input was mutated)", i, v, original[i])
		}
	}
}

func TestCache_SetGetFeedMembershipHash(t *testing.T) {
	cache := New("")

	// Getting hash for non-existent post returns empty string
	hash := cache.GetFeedMembershipHash("pages/post-a.md")
	if hash != "" {
		t.Errorf("GetFeedMembershipHash for missing post = %q, want empty", hash)
	}

	// Create a post entry first
	cache.MarkRebuilt("pages/post-a.md", "hash123", "output/post-a/index.html", "post.html")

	// Set and get the membership hash
	cache.SetFeedMembershipHash("pages/post-a.md", "membership-hash-abc")
	hash = cache.GetFeedMembershipHash("pages/post-a.md")
	if hash != "membership-hash-abc" {
		t.Errorf("GetFeedMembershipHash = %q, want %q", hash, "membership-hash-abc")
	}

	// Update the membership hash
	cache.SetFeedMembershipHash("pages/post-a.md", "membership-hash-def")
	hash = cache.GetFeedMembershipHash("pages/post-a.md")
	if hash != "membership-hash-def" {
		t.Errorf("GetFeedMembershipHash after update = %q, want %q", hash, "membership-hash-def")
	}
}

func TestCache_FeedMembershipHash_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".markata")

	// Create cache with feed membership hash
	cache := New(cacheDir)
	cache.MarkRebuilt("pages/post-a.md", "hash-a", "output/post-a/index.html", "post.html")
	cache.SetFeedMembershipHash("pages/post-a.md", "membership-hash-123")

	// Save
	if err := cache.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into new cache
	loaded, err := Load(cacheDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify feed membership hash was preserved
	hash := loaded.GetFeedMembershipHash("pages/post-a.md")
	if hash != "membership-hash-123" {
		t.Errorf("GetFeedMembershipHash after load = %q, want %q", hash, "membership-hash-123")
	}
}
