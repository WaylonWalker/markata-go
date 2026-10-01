package diagnostics

import (
	"reflect"
	"sync"
	"testing"
)

func TestContentLedger_SnapshotDeepOwnership(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"post.md", "image.png"})
	ledger.AddReason("post.md", ReasonContentDraft)
	ledger.AddIssue(Issue{File: "post.md", Code: "custom", Message: "original"})
	ledger.RecordFeed("post.md", "z", false, ReasonContentFiltered, ReasonFeedLimit)
	ledger.RecordFeed("post.md", "a", true, ReasonFeedOffset)
	before := ledger.Snapshot()
	untouched := ledger.Snapshot()
	original := ledgerSnapshotCopyForTest(untouched)

	ledger.MarkLoaded("post.md")
	ledger.AddReason("post.md", ReasonContentPrivate)
	ledger.AddIssue(Issue{File: "post.md", Code: "second", Message: "later"})
	ledger.RecordFeed("post.md", "z", true, ReasonFeedOffset)
	ledger.RecordFeed("post.md", "new", false, ReasonContentFiltered)
	if !reflect.DeepEqual(before, untouched) {
		t.Fatal("ledger updates changed a prior snapshot")
	}
	current := ledger.Snapshot()
	expected := ledger.Snapshot()

	before.Entries[1].Path = "mutated.md"
	before.Entries[1].Reasons[0] = "mutated"
	before.Entries[1].Diagnostics[0].Message = "mutated"
	before.Entries[1].Feeds[0].Feed = "mutated"
	before.Entries[1].Feeds[0].Reasons[0] = "mutated"
	before.Entries[1].Feeds[1].Reasons[0] = "mutated"
	current.Entries[1].Reasons[0] = "mutated again"
	current.Entries[1].Diagnostics[0].Code = "mutated again"
	current.Entries[1].Feeds[0].Reasons[0] = "mutated again"
	if !reflect.DeepEqual(ledger.Snapshot(), expected) {
		t.Fatal("caller mutation changed ledger values")
	}
	if !reflect.DeepEqual(untouched, original) {
		t.Fatal("caller mutation changed another prior snapshot")
	}
	expectedCopy := ledgerSnapshotCopyForTest(expected)
	ledger.Reset()
	if !reflect.DeepEqual(expected, expectedCopy) {
		t.Fatal("reset changed independently owned snapshot")
	}
}

// Copy raw values, preserving nil slices and without artifact sanitation.
func ledgerSnapshotCopyForTest(snapshot ContentLedgerSnapshot) ContentLedgerSnapshot {
	result := snapshot
	result.Entries = append([]ContentDisposition(nil), snapshot.Entries...)
	for index := range result.Entries {
		entry := &result.Entries[index]
		entry.Reasons = append([]string(nil), entry.Reasons...)
		entry.Diagnostics = append([]Issue(nil), entry.Diagnostics...)
		entry.Feeds = append([]ContentFeedDisposition(nil), entry.Feeds...)
		for feedIndex := range entry.Feeds {
			entry.Feeds[feedIndex].Reasons = append([]string(nil), entry.Feeds[feedIndex].Reasons...)
		}
	}
	return result
}

func TestContentLedger_SnapshotFeedReasonArenaOwnership(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		name := "unique"
		if duplicate {
			name = "deduplicated"
		}
		t.Run(name, func(t *testing.T) {
			live := []string{"z", "c", "b", "a", "tail"}
			if duplicate {
				live[1] = "b"
			}
			original := append([]string(nil), live...)
			ledger := NewContentLedger()
			ledger.Discover([]string{"a.md", "b.md", "empty.md"})
			for _, path := range []string{"a.md", "b.md", "empty.md"} {
				ledger.MarkPost(path, true)
				ledger.MarkEmitted(path)
			}
			// Deliberately overlapping live input, including across entries and
			// between entry and feed reasons. Copying must precede all sorting.
			ledger.mu.Lock()
			entry := ledger.entries["a.md"]
			entry.Reasons = live[:3]
			entry.feeds = map[string]*ContentFeedDisposition{
				"a":     {Feed: "a", Reasons: live[:4]},
				"b":     {Feed: "b", Included: true, Reasons: live[1:]},
				"empty": {Feed: "empty", Reasons: []string{}},
				"nil":   {Feed: "nil"},
			}
			ledger.entries["b.md"].feeds["other"] = &ContentFeedDisposition{Feed: "other", Reasons: live[1:]}
			raw := copyContentDisposition(entry)
			ledger.mu.Unlock()
			for _, feed := range raw.Feeds {
				if feed.Reasons == nil || cap(feed.Reasons) != len(feed.Reasons) {
					t.Fatal("raw copy changed empty semantics or failed to clamp capacity")
				}
			}
			first := ledger.Snapshot()
			second := ledger.Snapshot()
			expected := ledgerSnapshotCopyForTest(second)
			if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(live, original) {
				t.Fatal("snapshot sorting changed overlapping live reasons")
			}
			if first.Entries[2].Feeds != nil || second.Summary.Discovered != 3 || second.Summary.Emitted != 3 {
				t.Fatal("empty feeds or counts changed")
			}
			for _, feed := range first.Entries[0].Feeds {
				if cap(feed.Reasons) != len(feed.Reasons) {
					t.Fatal("deduplication left spare arena capacity")
				}
				if (feed.Feed == "empty" || feed.Feed == "nil") && feed.Reasons != nil {
					t.Fatal("snapshot empty reasons must remain nil")
				}
			}
			for index := range first.Entries[0].Feeds {
				before := ledgerSnapshotCopyForTest(first)
				feed := &first.Entries[0].Feeds[index]
				feed.Reasons = append(feed.Reasons, "appended", "again")
				feed.Reasons[0] = "mutated"
				for sibling := range first.Entries[0].Feeds {
					if sibling != index && !reflect.DeepEqual(first.Entries[0].Feeds[sibling], before.Entries[0].Feeds[sibling]) {
						t.Fatal("append or mutation corrupted a sibling feed")
					}
				}
				if !reflect.DeepEqual(first.Entries[1:], before.Entries[1:]) ||
					!reflect.DeepEqual(second, expected) || !reflect.DeepEqual(ledger.Snapshot(), expected) ||
					!reflect.DeepEqual(live, original) {
					t.Fatal("append or mutation changed another entry, snapshot or ledger")
				}
			}
			// The raw copy also owns its segments before normalization.
			for index := range raw.Feeds {
				raw.Feeds[index].Reasons = append(raw.Feeds[index].Reasons, "raw append")
				raw.Feeds[index].Reasons[0] = "raw mutation"
			}
			if !reflect.DeepEqual(live, original) || !reflect.DeepEqual(ledger.Snapshot(), expected) {
				t.Fatal("raw copy changed live input")
			}
			live[0] = "later live mutation"
			if !reflect.DeepEqual(second, expected) {
				t.Fatal("later live mutation changed an independently owned snapshot")
			}
		})
	}
}

func TestContentLedger_SnapshotEmptyFeedReasonArena(t *testing.T) {
	for _, feeds := range []map[string]*ContentFeedDisposition{
		nil,
		{},
		{"nil": {Feed: "nil"}},
		{"empty": {Feed: "empty", Reasons: []string{}}},
		{"nil": {Feed: "nil"}, "empty": {Feed: "empty", Reasons: []string{}}},
	} {
		ledger := NewContentLedger()
		ledger.Discover([]string{"post.md"})
		ledger.mu.Lock()
		entry := ledger.entries["post.md"]
		entry.feeds = feeds
		raw := copyContentDisposition(entry)
		ledger.mu.Unlock()
		for _, feed := range raw.Feeds {
			if feed.Reasons == nil || len(feed.Reasons) != 0 || cap(feed.Reasons) != 0 {
				t.Fatal("raw empty reasons must remain nonnil and capacity-clamped")
			}
		}
		snapshot := ledger.Snapshot()
		got := snapshot.Entries[0].Feeds
		if len(got) != len(feeds) || (len(feeds) == 0 && (got != nil || raw.Feeds != nil)) {
			t.Fatal("empty feed nil semantics changed")
		}
		for _, feed := range got {
			if feed.Reasons != nil {
				t.Fatal("snapshot empty reasons must remain nil")
			}
		}
		if snapshot.Summary.Discovered != 1 || snapshot.Summary.Candidates != 1 ||
			snapshot.Summary.Excluded != 1 || snapshot.Entries[0].Disposition != DispositionExcluded {
			t.Fatal("empty feeds changed derived counts or disposition")
		}
	}
}

func TestContentLedger_SnapshotConcurrentUpdates(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"post.md"})
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for iteration := 0; iteration < 50; iteration++ {
				ledger.RecordFeed("post.md", "feed", iteration%2 == 0, ReasonContentFiltered)
				ledger.AddIssue(Issue{File: "post.md", Code: "custom", Message: "same"})
				ledger.MarkLoaded("post.md")
			}
		}()
	}
	for iteration := 0; iteration < 50; iteration++ {
		snapshot := ledger.Snapshot()
		if len(snapshot.Entries) != 1 || snapshot.Summary.Discovered != 1 {
			t.Fatal("concurrent snapshot lost discovered content")
		}
		snapshotCopy := ledgerSnapshotCopyForTest(snapshot)
		ledger.MarkPost("post.md", true)
		if !reflect.DeepEqual(snapshot, snapshotCopy) {
			t.Fatal("snapshot changed after concurrent ledger update")
		}
	}
	workers.Wait()
	if got := len(ledger.Snapshot().Entries[0].Diagnostics); got != 1 {
		t.Fatalf("diagnostics = %d, want deduplicated issue", got)
	}
}
