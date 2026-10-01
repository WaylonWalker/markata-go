package buildcache

import (
	"sync"
	"testing"
)

func TestCache_RebuildReasonBoolAndBatch(t *testing.T) {
	cache := New(t.TempDir())
	cache.MarkRebuilt("match", "hash", "", "post.html")
	cache.MarkRebuilt("partial", "", "", "post.html")
	cache.MarkRebuilt("", "root-hash", "", "post.html")
	tests := []struct {
		path, hash, template string
		want                 RebuildReason
	}{
		{"missing", "hash", "post.html", RebuildReasonMissingEntry},
		{"match", "hash", "post.html", RebuildReasonNone},
		{"match", "changed", "changed.html", RebuildReasonInputChanged},
		{"match", "hash", "changed.html", RebuildReasonTemplateChanged},
		{"partial", "hash", "changed.html", RebuildReasonInputChanged},
		{"partial", "", "post.html", RebuildReasonNone},
		{"", "root-hash", "post.html", RebuildReasonNone},
	}
	for _, test := range tests {
		t.Run(test.path+"/"+test.hash+"/"+test.template, func(t *testing.T) {
			if got := cache.ReasonForRebuild(test.path, test.hash, test.template); got != test.want {
				t.Fatalf("reason = %v, want %v", got, test.want)
			}
			want := test.want != RebuildReasonNone
			if got := cache.ShouldRebuild(test.path, test.hash, test.template); got != want {
				t.Fatalf("bool = %v, want %v", got, want)
			}
			batch := cache.ShouldRebuildBatch([]struct{ Path, InputHash, Template string }{
				{test.path, test.hash, test.template},
			})
			if batch[test.path] != want {
				t.Fatalf("batch = %v, want %v", batch, want)
			}
		})
	}
}

func TestCache_RebuildReasonConcurrent(t *testing.T) {
	cache := New(t.TempDir())
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				cache.MarkRebuilt("post.md", "hash", "", "post.html")
				_ = cache.ReasonForRebuild("post.md", "hash", "post.html")
				_ = cache.ShouldRebuild("post.md", "hash", "post.html")
				_ = cache.ShouldRebuildBatch([]struct{ Path, InputHash, Template string }{
					{"post.md", "hash", "post.html"},
				})
			}
		}()
	}
	wg.Wait()
	if cache.ReasonForRebuild("post.md", "hash", "post.html") != RebuildReasonNone {
		t.Fatal("matching entry should be reusable")
	}
}
