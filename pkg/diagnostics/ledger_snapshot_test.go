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
