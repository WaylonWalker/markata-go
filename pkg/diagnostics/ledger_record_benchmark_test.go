package diagnostics

import (
	"fmt"
	"testing"
)

func BenchmarkContentLedgerRecord(b *testing.B) {
	for _, size := range []struct {
		posts, feeds int
	}{{64, 8}, {512, 418}, {3911, 418}} {
		b.Run(fmt.Sprintf("Fresh/%dx%d", size.posts, size.feeds), func(b *testing.B) {
			paths := make([]string, size.posts)
			feeds := make([]string, size.feeds)
			for i := range paths {
				paths[i] = fmt.Sprintf("posts/%d.md", i)
			}
			for i := range feeds {
				feeds[i] = fmt.Sprintf("feed-%d", i)
			}
			rows := make([]ContentFeedObservation, len(paths))
			for i, path := range paths {
				rows[i] = ContentFeedObservation{Path: path, Reasons: []string{ReasonContentFiltered}}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				ledger := NewContentLedger()
				for _, feed := range feeds {
					ledger.RecordFeedBatch(feed, rows)
				}
			}
		})
	}
	b.Run("SingleSparse", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			ledger := NewContentLedger()
			ledger.RecordFeed("post.md", "feed", false, ReasonContentFiltered)
		}
	})
	b.Run("BatchSparse", func(b *testing.B) {
		rows := []ContentFeedObservation{{Path: "post.md", Reasons: []string{ReasonContentFiltered}}}
		b.ReportAllocs()
		for b.Loop() {
			ledger := NewContentLedger()
			ledger.RecordFeedBatch("feed", rows)
		}
	})
	b.Run("Existing", func(b *testing.B) {
		ledger := benchmarkCompleteLedger(64, 8)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			ledger.RecordFeed("posts/00000.md", "feed-000", false, ReasonContentFiltered)
		}
	})
	b.Run("SingleUniqueFeeds", func(b *testing.B) {
		feeds := make([]string, 512)
		for i := range feeds {
			feeds[i] = fmt.Sprintf("feed-%d", i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			ledger := NewContentLedger()
			for _, feed := range feeds {
				ledger.RecordFeed("post.md", feed, true)
			}
		}
	})
	b.Run("SingleUniqueSources", func(b *testing.B) {
		paths := make([]string, 512)
		for i := range paths {
			paths[i] = fmt.Sprintf("posts/%d.md", i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			ledger := NewContentLedger()
			for _, path := range paths {
				ledger.RecordFeed(path, "feed", true)
			}
		}
	})
	b.Run("ExistingAfterBatch", func(b *testing.B) {
		ledger := benchmarkCompleteLedger(64, 8)
		ledger.RecordFeedBatch("feed-000", []ContentFeedObservation{{Path: "posts/00000.md"}})
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			ledger.RecordFeed("posts/00000.md", "feed-000", false, ReasonContentFiltered)
		}
	})
	b.Run("BatchExisting", func(b *testing.B) {
		ledger := benchmarkCompleteLedger(64, 8)
		rows := []ContentFeedObservation{{Path: "posts/00000.md", Reasons: []string{ReasonContentFiltered}}}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			ledger.RecordFeedBatch("feed-000", rows)
		}
	})
	b.Run("MixedSingleBatch", func(b *testing.B) {
		rows := []ContentFeedObservation{{Path: "a.md"}, {Path: "b.md"}}
		b.ReportAllocs()
		for b.Loop() {
			ledger := NewContentLedger()
			ledger.RecordFeed("a.md", "feed", true)
			ledger.RecordFeedBatch("feed", rows)
			ledger.RecordFeed("b.md", "other", true)
			ledger.RecordFeedBatch("other", rows)
		}
	})
}
