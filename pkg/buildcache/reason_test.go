package buildcache

import "testing"

func TestReasonForRebuild(t *testing.T) {
	cache := New(t.TempDir())
	cache.MarkRebuiltWithSlug("pages/post.md", "post", "hash-a", "output/post/index.html", "post.html")

	tests := []struct {
		name     string
		path     string
		hash     string
		template string
		want     RebuildReason
	}{
		{name: "hit", path: "pages/post.md", hash: "hash-a", template: "post.html", want: RebuildReasonNone},
		{name: "missing", path: "pages/missing.md", hash: "hash-a", template: "post.html", want: RebuildReasonMissingEntry},
		{name: "input changed", path: "pages/post.md", hash: "hash-b", template: "post.html", want: RebuildReasonInputChanged},
		{name: "template changed", path: "pages/post.md", hash: "hash-a", template: "article.html", want: RebuildReasonTemplateChanged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cache.ReasonForRebuild(tt.path, tt.hash, tt.template); got != tt.want {
				t.Fatalf("ReasonForRebuild() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReasonForRebuildNilCache(t *testing.T) {
	var cache *Cache
	if got := cache.ReasonForRebuild("pages/post.md", "hash", "post.html"); got != RebuildReasonMissingEntry {
		t.Fatalf("ReasonForRebuild() = %q, want %q", got, RebuildReasonMissingEntry)
	}
}
