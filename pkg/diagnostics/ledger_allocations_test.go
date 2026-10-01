package diagnostics

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestContentFeedInitialInternerBounds(t *testing.T) {
	a := contentFeedBatchArena{remaining: 256, created: contentFeedInternMin}
	a.internInitialReasons([]string{"custom-0", "second"})
	for i := 0; i < contentFeedInternLists+8; i++ {
		input := []string{"", fmt.Sprintf("custom-%d", i), "second", "second"}
		owned := a.internInitialReasons(input)
		if i < contentFeedInternLists {
			if !reflect.DeepEqual(owned, input[1:3]) || cap(owned) != len(owned) {
				t.Fatal("canonical order or capacity changed")
			}
			input[1] = "mutated"
			if owned[0] == input[1] {
				t.Fatal("caller storage retained")
			}
		} else if owned != nil {
			t.Fatal("interner exceeded list bound")
		}
	}
	if len(a.intern) != contentFeedInternLists ||
		a.internInitialReasons([]string{"a", "b", "c", "d", "e"}) != nil ||
		a.internInitialReasons(make([]string, contentFeedReasonSlabSlots+1)) != nil {
		t.Fatal("interner bounds not enforced")
	}
	sparse := contentFeedBatchArena{remaining: 1}
	sparse.newDisposition("feed", []string{"reason"})
	if sparse.intern != nil {
		t.Fatal("small batch allocated interner")
	}
	unique := contentFeedBatchArena{created: contentFeedInternMin}
	for i := 0; i < 64; i++ {
		if unique.internInitialReasons([]string{fmt.Sprintf("unique-%d", i)}) != nil {
			t.Fatal("unique stream admitted a list")
		}
	}
	if unique.intern != nil || unique.probes != contentFeedInternProbes {
		t.Fatal("unique stream did not stop probing without allocation")
	}
}

func TestRecordFeedBatchInitialSharingDifferential(t *testing.T) {
	got, want := NewContentLedger(), NewContentLedger()
	input := []string{"", "z", "a", "z"}
	rows := make([]ContentFeedObservation, 128)
	for i := range rows {
		rows[i] = ContentFeedObservation{Path: fmt.Sprintf("posts/%d.md", i), Included: i%3 == 0, Reasons: input}
	}
	rows = append(rows,
		ContentFeedObservation{Path: "./posts/100.md", Included: true, Reasons: []string{"later"}},
		ContentFeedObservation{Path: "unknown.png", Reasons: input},
		ContentFeedObservation{Path: ""},
	)
	got.RecordFeedBatch("", rows)
	for _, row := range rows {
		legacyRecordFeed(want, row.Path, "", row.Included, row.Reasons...)
	}
	first := got.entries["posts/80.md"].feeds[""]
	sibling := got.entries["posts/81.md"].feeds[""]
	if &first.Reasons[0] != &sibling.Reasons[0] {
		t.Fatal("repeated initial lists were not shared")
	}
	for _, path := range []string{"posts/80.md", "posts/100.md"} {
		got.RecordFeed(path, "", true, "followup")
		legacyRecordFeed(want, path, "", true, "followup")
	}
	input[1] = "caller mutation"
	if !reflect.DeepEqual(first.Reasons, []string{"z", "a", "followup"}) ||
		!reflect.DeepEqual(sibling.Reasons, []string{"z", "a"}) ||
		!reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
		t.Fatal("shared initial reasons changed ordered observation semantics")
	}
}

// Independent pre-cache snapshot oracle, including raw copying before sorting.
func rawLedgerSnapshot(l *ContentLedger) ContentLedgerSnapshot {
	l.mu.RLock()
	result := ContentLedgerSnapshot{Entries: make([]ContentDisposition, 0, len(l.entries)), TemplateCache: cloneTemplateCacheStats(l.templateCache)}
	for _, entry := range l.entries {
		result.Entries = append(result.Entries, copyContentDisposition(entry))
	}
	l.mu.RUnlock()
	for i := range result.Entries {
		entry := &result.Entries[i]
		entry.Reasons = sortedUnique(entry.Reasons)
		entry.Diagnostics = sortedIssues(entry.Diagnostics)
		slices.SortFunc(entry.Feeds, func(a, b ContentFeedDisposition) int { return strings.Compare(a.Feed, b.Feed) })
		for j := range entry.Feeds {
			feed := &entry.Feeds[j]
			feed.Reasons = sortedUnique(feed.Reasons)
			feed.Reasons = feed.Reasons[:len(feed.Reasons):len(feed.Reasons)]
		}
		result.Summary.Discovered++
		if !entry.Candidate {
			entry.Disposition = DispositionNotCandidate
			continue
		}
		updateContentSummary(&result.Summary, *entry)
		finalizeContentDisposition(&result.Summary, entry)
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Path < result.Entries[j].Path })
	return result
}

func TestContentFeedSnapshotOrdersAdversarial(t *testing.T) {
	var cache contentFeedSnapshotOrders
	for _, count := range []int{0, 1, 31, 32, 418, 1024, 1025, 32, 32, 32, 32, 32} {
		for layout := 0; layout < 7; layout++ {
			entry := newContentLedgerEntry(fmt.Sprintf("%d-%d.md", count, layout))
			for i := 0; i < count; i++ {
				key := fmt.Sprintf("%d-feed-%04d", layout, i)
				entry.feeds[key] = &ContentFeedDisposition{Feed: key, Reasons: []string{"z", "a", "a"}}
			}
			for _, mismatch := range []bool{false, true, false} {
				key := fmt.Sprintf("%d-feed-%04d", layout, 0)
				if count > 0 {
					entry.feeds[key].Feed = key
					if mismatch {
						entry.feeds[key].Feed = "wrong-label"
					}
				}
				got, sorted := cache.copy(entry)
				want := copyContentDisposition(entry)
				compare := func(a, b ContentFeedDisposition) int { return strings.Compare(a.Feed, b.Feed) }
				if sorted && !slices.IsSortedFunc(got.Feeds, compare) {
					t.Fatal("unverified layout treated as sorted")
				}
				if mismatch && sorted {
					t.Fatal("key/label mismatch used cached layout")
				}
				slices.SortFunc(got.Feeds, compare)
				slices.SortFunc(want.Feeds, compare)
				if !reflect.DeepEqual(got, want) || cache.names > contentFeedOrderNames ||
					cache.admitted > contentFeedOrderAdmissions {
					t.Fatal("cache copy differs or exceeds name budget")
				}
				names := 0
				for _, keys := range cache.layouts {
					names += len(keys)
					if len(keys) > contentFeedOrderMax {
						t.Fatal("cached layout exceeds per-layout limit")
					}
				}
				if names != cache.names {
					t.Fatal("cache name budget accounting differs from retained layouts")
				}
				ledger := NewContentLedger()
				ledger.entries[entry.Path] = entry
				if !reflect.DeepEqual(ledger.Snapshot(), rawLedgerSnapshot(ledger)) {
					t.Fatal("snapshot normalization differs")
				}
			}
		}
	}
}

func TestContentLedgerDenseConcurrentResetSnapshot(t *testing.T) {
	ledger := NewContentLedger()
	rows := make([]ContentFeedObservation, 64)
	for i := range rows {
		rows[i] = ContentFeedObservation{Path: fmt.Sprintf("%d.md", i), Reasons: []string{"custom"}}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for repeat := 0; repeat < 5; repeat++ {
			for i := 0; i < 40; i++ {
				ledger.RecordFeedBatch(fmt.Sprintf("feed-%d", i), rows)
			}
			ledger.Reset()
		}
	}()
	for i := 0; i < 50; i++ {
		snapshot := ledger.Snapshot()
		owned := ledgerSnapshotCopyForTest(snapshot)
		for _, entry := range snapshot.Entries {
			if cap(entry.Reasons) != len(entry.Reasons) {
				t.Fatal("entry reason capacity not clamped")
			}
			for _, feed := range entry.Feeds {
				if cap(feed.Reasons) != len(feed.Reasons) {
					t.Fatal("feed reason capacity not clamped")
				}
			}
		}
		_ = ledger.Snapshot()
		if len(snapshot.Entries) > 0 && !reflect.DeepEqual(snapshot, owned) {
			t.Fatal("concurrent snapshot/reset changed returned storage")
		}
	}
	wg.Wait()
}

func TestContentLedgerCachedSnapshotDeepOwnership(t *testing.T) {
	ledger := benchmarkCompleteLedger(64, 40)
	before := ledger.Snapshot()
	other := ledger.Snapshot()
	expected := ledgerSnapshotCopyForTest(other)
	if !reflect.DeepEqual(before, rawLedgerSnapshot(ledger)) {
		t.Fatal("dense cached snapshot differs from raw-copy oracle")
	}
	for i := range before.Entries {
		entry := &before.Entries[i]
		entry.Reasons[0] = "mutated"
		entry.Diagnostics[0].Message = "mutated"
		for j := range entry.Feeds {
			feed := &entry.Feeds[j]
			feed.Reasons = append(feed.Reasons, "append")
			feed.Reasons[0] = "mutated"
			feed.Feed = "mutated"
		}
		entry.Feeds = append(entry.Feeds, ContentFeedDisposition{Feed: "new"})
	}
	if !reflect.DeepEqual(other, expected) || !reflect.DeepEqual(ledger.Snapshot(), expected) {
		t.Fatal("cached snapshot shared returned storage")
	}
	ledger.Discover([]string{"fresh.md"})
	if !reflect.DeepEqual(other, expected) {
		t.Fatal("rediscovery changed cached snapshot storage")
	}
}
