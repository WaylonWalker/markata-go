package diagnostics

import (
	"reflect"
	"testing"
)

func TestRecordFeedBatchOversizedRawReasonFallback(t *testing.T) {
	empty := make([]string, 8192)
	duplicates := make([]string, 8192)
	for i := range duplicates {
		if i%3 != 0 {
			duplicates[i] = "only-unique"
		}
	}
	for _, tc := range []struct {
		name    string
		reasons []string
		want    []string
	}{
		{name: "all-empty", reasons: empty},
		{name: "duplicates-and-empties", reasons: duplicates, want: []string{"only-unique"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ledger, oracle := NewContentLedger(), NewContentLedger()
			arena := contentFeedBatchArena{remaining: 1}
			ledger.mu.Lock()
			ledger.recordFeedLocked("post.md", "feed", false, tc.reasons, &arena)
			ledger.mu.Unlock()
			legacyRecordFeed(oracle, "post.md", "feed", false, tc.reasons...)
			disposition := ledger.entries["post.md"].feeds["feed"]
			if arena.reasons != nil {
				t.Fatal("oversized raw input reserved a reason slab")
			}
			if !reflect.DeepEqual(disposition.Reasons, tc.want) || cap(disposition.Reasons) != len(tc.want) {
				t.Fatal("oversized raw input retained storage for duplicate or empty values")
			}
			if !reflect.DeepEqual(ledger.Snapshot(), oracle.Snapshot()) {
				t.Fatal("fallback differs from ordinary standalone recording")
			}
			for i := range tc.reasons {
				tc.reasons[i] = "caller-mutated"
			}
			if !reflect.DeepEqual(ledger.Snapshot(), oracle.Snapshot()) {
				t.Fatal("fallback retained caller storage")
			}
		})
	}
}

func TestContentFeedBatchOversizedFallbackPreservesSlabNeighbors(t *testing.T) {
	arena := contentFeedBatchArena{remaining: 3}
	first := arena.newDisposition("first", []string{"first"})
	addFeedReason(first, "first")
	unused := arena.reasons
	oversized := make([]string, 8192)
	for i := range oversized {
		oversized[i] = "oversized"
	}
	arena.remaining = 2
	second := arena.newDisposition("second", oversized)
	for _, reason := range oversized {
		addFeedReason(second, reason)
	}
	if len(arena.reasons) != len(unused) || &arena.reasons[0] != &unused[0] {
		t.Fatal("fallback discarded or consumed a live slab header")
	}
	arena.remaining = 1
	third := arena.newDisposition("third", []string{"third"})
	addFeedReason(third, "third")
	addFeedReason(first, "first-update")
	addFeedReason(second, "second-update")
	if !reflect.DeepEqual(first.Reasons, []string{"first", "first-update"}) ||
		!reflect.DeepEqual(second.Reasons, []string{"oversized", "second-update"}) ||
		!reflect.DeepEqual(third.Reasons, []string{"third"}) {
		t.Fatal("fallback or append reused/cleared published slab storage")
	}
}
