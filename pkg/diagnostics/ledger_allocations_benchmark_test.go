package diagnostics

import (
	"fmt"
	"testing"
)

// The same bounded slab path without initial-list interning, for matched runs.
func recordFeedBatchSlabBaseline(l *ContentLedger, feed string, rows []ContentFeedObservation) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := contentFeedBatchArena{created: -len(rows)}
	for i, row := range rows {
		a.remaining = len(rows) - i
		path := normalizeContentPath(row.Path)
		if path != "" {
			l.recordFeedLocked(path, feed, row.Included, row.Reasons, &a)
		}
	}
}

func BenchmarkContentLedgerInitialReasons(b *testing.B) {
	for _, shape := range []struct {
		name         string
		posts, feeds int
		unique       bool
	}{{"Dense", 3911, 418, false}, {"Mixed", 512, 64, false}, {"Unique", 512, 8, true}, {"Sparse", 1, 1, false}} {
		rows := make([]ContentFeedObservation, shape.posts)
		for i := range rows {
			reasons := []string{ReasonContentFiltered}
			if shape.unique {
				reasons = []string{fmt.Sprintf("custom-%d", i)}
			} else if shape.name == "Mixed" {
				switch i % 4 {
				case 0:
					reasons = nil
				case 1:
					reasons = []string{"", ReasonFeedLimit, "custom", ReasonFeedLimit}
				}
			}
			rows[i] = ContentFeedObservation{Path: fmt.Sprintf("posts/%d.md", i), Included: shape.name != "Dense" && i%4 == 0, Reasons: reasons}
		}
		feeds := make([]string, shape.feeds)
		for i := range feeds {
			feeds[i] = fmt.Sprintf("feed-%d", i)
		}
		for _, baseline := range []bool{true, false} {
			b.Run(fmt.Sprintf("%s/Baseline=%t", shape.name, baseline), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					ledger := NewContentLedger()
					for _, feed := range feeds {
						if baseline {
							recordFeedBatchSlabBaseline(ledger, feed, rows)
						} else {
							ledger.RecordFeedBatch(feed, rows)
						}
					}
				}
			})
		}
	}
}

func BenchmarkContentLedgerSnapshotLayouts(b *testing.B) {
	for _, shape := range []string{"Dense", "Mixed", "Sparse", "DifferentSets"} {
		ledger := NewContentLedger()
		for i := 0; i < 512; i++ {
			path := fmt.Sprintf("%d.md", i)
			count := 418
			if shape == "Sparse" || (shape == "Mixed" && i%2 == 0) {
				count = 2
			}
			for feed := 0; feed < count; feed++ {
				layout := 0
				if shape == "DifferentSets" {
					layout = i % 8
				}
				ledger.RecordFeed(path, fmt.Sprintf("%d-feed-%d", layout, feed), false, "custom")
			}
		}
		for _, baseline := range []bool{true, false} {
			b.Run(fmt.Sprintf("%s/Baseline=%t", shape, baseline), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if baseline {
						_ = rawLedgerSnapshot(ledger)
					} else {
						_ = ledger.Snapshot()
					}
				}
			})
		}
	}
}
