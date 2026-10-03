package diagnostics

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

// Keep the pre-batch implementation as an independent ordered-write oracle.
func legacyRecordFeed(l *ContentLedger, path, feed string, included bool, reasons ...string) {
	if l == nil {
		return
	}
	path = normalizeContentPath(path)
	if path == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entryLocked(path)
	if !entry.Candidate {
		return
	}
	if entry.feeds == nil {
		entry.feeds = make(map[string]*ContentFeedDisposition)
	}
	disposition := entry.feeds[feed]
	if disposition == nil {
		disposition = &ContentFeedDisposition{Feed: feed}
		entry.feeds[feed] = disposition
	}
	disposition.Included = included
	for _, reason := range reasons {
		addFeedReason(disposition, reason)
		if reason != "" && !included {
			addReasonLocked(entry, reason)
		}
	}
}

func TestRecordFeedBatchDifferential(t *testing.T) {
	rows := []ContentFeedObservation{
		{Path: "a.md", Reasons: []string{"", "arbitrary", "arbitrary", ReasonFeedOffset}},
		{Path: "./a.md", Included: true, Reasons: []string{ReasonFeedLimit}},
		{Path: "dir/../a.md", Included: false},
		{Path: `dir\b.md`, Included: false, Reasons: []string{ReasonContentDraft}},
		{Path: "dir/b.md", Included: true},
		{Path: "", Reasons: []string{"ignored"}},
		{Path: ".", Reasons: []string{"ignored"}},
		{Path: "/absolute/a.md", Reasons: []string{"external"}},
		{Path: normalizeContentPath("/absolute/a.md"), Included: true},
		{Path: "../a.md", Included: true},
		{Path: "C:\\secret\\a.md", Reasons: []string{"external"}},
		{Path: "unknown.txt", Reasons: []string{"ignored"}},
		{Path: "unknown.png", Reasons: []string{"ignored-noncandidate"}},
		{Path: "unknown.md", Reasons: []string{ReasonContentFiltered}},
	}
	for _, feed := range []string{"", "new", "prior"} {
		t.Run(feed, func(t *testing.T) {
			got, want := &ContentLedger{}, &ContentLedger{}
			for _, ledger := range []*ContentLedger{got, want} {
				ledger.Discover([]string{"a.md", "dir/b.md", "discovered.txt"})
			}
			got.RecordFeed("a.md", "prior", false, "prior-reason")
			got.RecordFeed("only-prior.md", "prior", true)
			legacyRecordFeed(want, "a.md", "prior", false, "prior-reason")
			legacyRecordFeed(want, "only-prior.md", "prior", true)
			for repeat := 0; repeat < 3; repeat++ {
				got.RecordFeedBatch(feed, rows)
				for _, row := range rows {
					legacyRecordFeed(want, row.Path, feed, row.Included, row.Reasons...)
				}
				if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
					t.Fatalf("repeat %d snapshots differ", repeat)
				}
			}
			for _, reset := range []func(*ContentLedger){
				func(l *ContentLedger) { l.Reset() },
				func(l *ContentLedger) { l.Discover([]string{"a.md"}) },
			} {
				reset(got)
				reset(want)
				got.RecordFeedBatch(feed, rows)
				for _, row := range rows {
					legacyRecordFeed(want, row.Path, feed, row.Included, row.Reasons...)
				}
				if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
					t.Fatal("snapshot differs after reset/discover")
				}
			}
		})
	}
	var nilLedger *ContentLedger
	nilLedger.RecordFeedBatch("", rows)
	ledger := &ContentLedger{}
	ledger.RecordFeedBatch("", nil)
	ledger.RecordFeedBatch("", []ContentFeedObservation{})
	if len(ledger.Snapshot().Entries) != 0 {
		t.Fatal("empty batch is not a no-op")
	}
}

func TestRecordFeedBatchOwnsInputs(t *testing.T) {
	reasons := []string{"one", "two", "two"}
	rows := []ContentFeedObservation{{Path: "a.md", Reasons: reasons[:2]}, {Path: "b.md", Reasons: reasons[1:]}}
	ledger := NewContentLedger()
	ledger.RecordFeedBatch("feed", rows)
	before := ledger.Snapshot()
	reasons[0], reasons[1] = "changed", "changed"
	rows[0] = ContentFeedObservation{Path: "changed.md"}
	if !reflect.DeepEqual(before, ledger.Snapshot()) {
		t.Fatal("ledger retained caller storage")
	}
}

func TestRecordFeedBatchConcurrentAtomic(t *testing.T) {
	ledger := NewContentLedger()
	rows := []ContentFeedObservation{{Path: "a.md"}, {Path: "b.md"}}
	ledger.RecordFeedBatch("feed", rows)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			rows[0].Included, rows[1].Included = i%2 == 0, i%2 == 0
			ledger.RecordFeedBatch("feed", rows)
			ledger.RecordFeed("other.md", "other", true)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			snapshot := ledger.Snapshot()
			if snapshot.Entries[0].Feeds[0].Included != snapshot.Entries[1].Feeds[0].Included {
				t.Error("snapshot observed partial batch")
				return
			}
		}
	}()
	wg.Wait()
}

func TestRecordFeedBatchMixedDifferential(t *testing.T) {
	got, want := &ContentLedger{}, &ContentLedger{}
	random := rand.New(rand.NewSource(1475)) //nolint:gosec // Deterministic differential fixture, not security randomness.
	paths := []string{"a.md", "./a.md", "b.md", "other.md", "", ".", "unknown.png", "/outside.md", "../outside.md"}
	reasons := []string{"", "custom", ReasonContentDraft, ReasonFeedLimit, ReasonFeedOffset}
	for i := 0; i < 300; i++ {
		feed := []string{"", "a", "b", "c"}[random.Intn(4)]
		rows := make([]ContentFeedObservation, random.Intn(9))
		for j := range rows {
			rows[j] = ContentFeedObservation{Path: paths[random.Intn(len(paths))], Included: random.Intn(2) == 0}
			for k := 0; k < random.Intn(6); k++ {
				rows[j].Reasons = append(rows[j].Reasons, reasons[random.Intn(len(reasons))])
			}
		}
		if i%31 == 0 {
			got.Reset()
			want.Reset()
		} else if i%17 == 0 {
			got.Discover(paths)
			want.Discover(paths)
		}
		if i%2 == 0 {
			for _, row := range rows {
				got.RecordFeed(row.Path, feed, row.Included, row.Reasons...)
			}
		} else {
			got.RecordFeedBatch(feed, rows)
		}
		for _, row := range rows {
			legacyRecordFeed(want, row.Path, feed, row.Included, row.Reasons...)
		}
		if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
			t.Fatalf("mixed operation %d differs", i)
		}
	}
}

func TestRecordFeedBatchConcurrentResetDiscover(t *testing.T) {
	ledger := NewContentLedger()
	rows := []ContentFeedObservation{{Path: "a.md", Included: true}, {Path: "b.md", Included: true}}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			ledger.Discover([]string{"a.md", "b.md"})
			ledger.RecordFeedBatch("feed", rows)
			ledger.Reset()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			snapshot := ledger.Snapshot()
			if len(snapshot.Entries) == 0 {
				continue
			}
			if len(snapshot.Entries) != 2 || len(snapshot.Entries[0].Feeds) != len(snapshot.Entries[1].Feeds) {
				t.Error("snapshot observed partial reset/discover/batch")
				return
			}
		}
	}()
	wg.Wait()
}

func TestRecordFeedBatchPreexistingSingleWrites(t *testing.T) {
	got, want := &ContentLedger{}, &ContentLedger{}
	for _, row := range []struct{ path, feed string }{
		{"a.md", ""}, {"b.md", "preexisting"}, {"a.md", "preexisting"},
	} {
		got.RecordFeed(row.path, row.feed, false, "before")
		legacyRecordFeed(want, row.path, row.feed, false, "before")
	}

	got.RecordFeedBatch("unused", nil)
	rows := []ContentFeedObservation{
		{Path: "a.md", Included: true},
		{Path: "./a.md", Reasons: []string{"after"}},
		{Path: "b.md", Included: true},
	}
	for _, feed := range []string{"new", "", "preexisting"} {
		got.RecordFeedBatch(feed, rows)
		for _, row := range rows {
			legacyRecordFeed(want, row.Path, feed, row.Included, row.Reasons...)
		}
	}
	got.RecordFeed("b.md", "single-after", false, "single")
	legacyRecordFeed(want, "b.md", "single-after", false, "single")
	got.RecordFeed("unknown.png", "noncandidate", true)
	legacyRecordFeed(want, "unknown.png", "noncandidate", true)
	got.RecordFeedBatch("single-after", rows)
	for _, row := range rows {
		legacyRecordFeed(want, row.Path, "single-after", row.Included, row.Reasons...)
	}
	if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
		t.Fatal("preexisting or subsequent single writes changed ordered batch observations")
	}
}

func TestRecordFeedBatchSlabBoundariesAndLaterUpdates(t *testing.T) {
	const count = 2*contentFeedObjectSlabSlots + 7
	rows := make([]ContentFeedObservation, count)
	for i := range rows {
		reason := fmt.Sprintf("initial-%03d", i)
		rows[i] = ContentFeedObservation{
			Path:    fmt.Sprintf("posts/%03d.md", i),
			Reasons: []string{"", reason, "shared", reason, ""},
		}
	}
	got, want := NewContentLedger(), NewContentLedger()
	got.RecordFeedBatch("feed", rows)
	for _, row := range rows {
		legacyRecordFeed(want, row.Path, "feed", row.Included, row.Reasons...)
	}
	pointers := make([]*ContentFeedDisposition, count)
	for i, row := range rows {
		disposition := got.entries[row.Path].feeds["feed"]
		pointers[i] = disposition
		expected := []string{row.Reasons[1], "shared"}
		if !reflect.DeepEqual(disposition.Reasons, expected) || cap(disposition.Reasons) != len(expected) {
			t.Fatalf("row %d reason order/ownership = %v cap=%d", i, disposition.Reasons, cap(disposition.Reasons))
		}
	}
	before := got.Snapshot()
	// Both singles and later batches append beyond initial clamped segments.
	// Repeated normalized paths merge without replacing the slab object.
	for i, row := range rows {
		got.RecordFeed(row.Path, "feed", true, "single", "shared", "")
		legacyRecordFeed(want, row.Path, "feed", true, "single", "shared", "")
		if got.entries[row.Path].feeds["feed"] != pointers[i] {
			t.Fatalf("row %d pointer changed after append", i)
		}
	}
	updates := []ContentFeedObservation{
		{Path: "./posts/000.md", Reasons: []string{"batch", "batch"}},
		{Path: "posts/000.md", Included: true, Reasons: []string{"final"}},
		{Path: "posts/010.md", Reasons: []string{"boundary"}},
		{Path: "new.md", Reasons: []string{"new"}},
	}
	got.RecordFeedBatch("feed", updates)
	for _, row := range updates {
		legacyRecordFeed(want, row.Path, "feed", row.Included, row.Reasons...)
	}
	runtime.GC() // Map-held pointers must retain every previously allocated slab.
	for i, row := range rows {
		if got.entries[row.Path].feeds["feed"] != pointers[i] {
			t.Fatalf("row %d pointer changed after later batch", i)
		}
	}
	if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
		t.Fatal("slab growth or later append changed ordered observations")
	}
	// Caller mutation must not alter initial segments or snapshots.
	for i := range rows {
		rows[i].Reasons[1] = "caller-change"
	}
	if !reflect.DeepEqual(got.Snapshot(), want.Snapshot()) {
		t.Fatal("slab retained caller reasons")
	}
	for _, entry := range before.Entries {
		if len(entry.Feeds[0].Reasons) != 2 {
			t.Fatal("later updates mutated earlier snapshot")
		}
	}
	got.Reset()
	got.RecordFeedBatch("feed", []ContentFeedObservation{{Path: "posts/000.md", Reasons: []string{"replacement"}}})
	got.Discover([]string{"different.md"})
	if pointers[0].Reasons[0] != "initial-000" || before.Entries[0].Feeds[0].Reasons[0] != "initial-000" {
		t.Fatal("reset/discovery reused live slab or snapshot storage")
	}
}

func TestContentFeedBatchArenaBoundsAndStablePointers(t *testing.T) {
	const count = 2*contentFeedObjectSlabSlots + 3
	arena := contentFeedBatchArena{}
	pointers := make([]*ContentFeedDisposition, count)
	for i := range pointers {
		arena.remaining = count - i
		feed := fmt.Sprintf("feed-%d", i)
		pointers[i] = arena.newDisposition(feed, []string{"", feed, feed, "shared"})
		for _, reason := range []string{"", feed, feed, "shared"} {
			addFeedReason(pointers[i], reason)
		}
		if len(arena.objects) >= contentFeedObjectSlabSlots || len(arena.reasons) >= contentFeedReasonSlabSlots {
			t.Fatal("allocator exceeded fixed slab bounds")
		}
	}
	runtime.GC()
	for i, disposition := range pointers {
		expected := fmt.Sprintf("feed-%d", i)
		if disposition.Feed != expected || !reflect.DeepEqual(disposition.Reasons, []string{expected, "shared"}) {
			t.Fatalf("published pointer %d changed after slab growth", i)
		}
	}
	addFeedReason(pointers[0], "later")
	if !reflect.DeepEqual(pointers[1].Reasons, []string{"feed-1", "shared"}) {
		t.Fatal("append overwrote neighboring reason segment")
	}
	tail := contentFeedBatchArena{remaining: 1}
	disposition := tail.newDisposition("tail", []string{"one", "two"})
	addFeedReason(disposition, "one")
	addFeedReason(disposition, "two")
	if len(tail.objects) != 0 || len(tail.reasons) != 0 || cap(disposition.Reasons) != 2 {
		t.Fatal("one-observation tail was not exact-sized")
	}
}

func TestRecordFeedBatchOversizedInitialReasons(t *testing.T) {
	var reasons, expected []string
	for i := 0; i < contentFeedReasonSlabSlots+9; i++ {
		reason := fmt.Sprintf("reason-%03d", i)
		reasons = append(reasons, "", reason, reason)
		expected = append(expected, reason)
	}
	ledger := NewContentLedger()
	ledger.RecordFeedBatch("", []ContentFeedObservation{
		{Path: "large.md", Reasons: reasons},
		{Path: "empty.md", Included: true, Reasons: []string{"", ""}},
		{Path: "neighbor.md", Reasons: []string{"neighbor"}},
	})
	disposition := ledger.entries["large.md"].feeds[""]
	if !reflect.DeepEqual(disposition.Reasons, expected) {
		t.Fatal("oversized reasons were not ordered, deduplicated, and independently owned")
	}
	if ledger.entries["empty.md"].feeds[""].Reasons != nil {
		t.Fatal("empty-only reasons lost nil semantics")
	}
	for i := range reasons {
		reasons[i] = "changed"
	}
	ledger.RecordFeed("large.md", "", true, "later", expected[0])
	if !reflect.DeepEqual(disposition.Reasons, append(expected, "later")) {
		t.Fatal("oversized storage changed after caller mutation or later append")
	}
	if !reflect.DeepEqual(ledger.entries["neighbor.md"].feeds[""].Reasons, []string{"neighbor"}) {
		t.Fatal("oversized reason update corrupted neighbor")
	}
}

func TestRecordFeedBatchExistingUpdatesAllocateNoSlabs(t *testing.T) {
	ledger := NewContentLedger()
	rows := []ContentFeedObservation{{Path: "a.md", Reasons: []string{"reason", "reason"}}, {Path: "b.md", Included: true}}
	ledger.RecordFeedBatch("feed", rows)
	if allocations := testing.AllocsPerRun(100, func() { ledger.RecordFeedBatch("feed", rows) }); allocations != 0 {
		t.Fatalf("existing updates allocated %v times", allocations)
	}
	arena := contentFeedBatchArena{remaining: 2}
	ledger.mu.Lock()
	ledger.recordFeedLocked("a.md", "feed", false, []string{"reason"}, &arena)
	ledger.recordFeedLocked("unknown.png", "feed", false, []string{"ignored"}, &arena)
	ledger.mu.Unlock()
	if arena.objects != nil || arena.reasons != nil {
		t.Fatal("existing or noncandidate observations allocated persistent slabs")
	}
}
