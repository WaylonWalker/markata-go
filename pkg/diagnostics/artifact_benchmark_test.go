package diagnostics

import (
	"fmt"
	"io"
	"testing"
	"time"
)

// Hundreds of feeds per source intentionally retain all exclusions, like a
// large site's complete diagnostics artifact. Setup is outside measured time.
func benchmarkCompleteLedger(posts, feeds int) *ContentLedger {
	ledger := NewContentLedger()
	for post := 0; post < posts; post++ {
		path := fmt.Sprintf("posts/%05d.md", post)
		ledger.AddDiscovered(path)
		ledger.MarkLoaded(path)
		ledger.MarkFrontmatter(path, true, true)
		ledger.MarkPost(path, true)
		ledger.MarkRendered(path)
		ledger.MarkEmitted(path)
		ledger.AddIssue(Issue{File: path, Code: ReasonFrontmatterSuspiciousDelimiter, Severity: SeverityWarning})
		for feed := 0; feed < feeds; feed++ {
			ledger.RecordFeed(path, fmt.Sprintf("feed-%03d", feed), feed%10 == 0, ReasonContentFiltered)
		}
	}
	return ledger
}

func BenchmarkContentLedgerSnapshot_CompleteFeeds(b *testing.B) {
	for _, posts := range []int{512, 3911} {
		b.Run(fmt.Sprintf("%dx418", posts), func(b *testing.B) {
			ledger := benchmarkCompleteLedger(posts, 418)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = ledger.Snapshot()
			}
		})
	}
}

func BenchmarkArtifactSerialization_CompleteFeeds(b *testing.B) {
	snapshot := benchmarkCompleteLedger(512, 418).Snapshot()
	info := ArtifactBuildInfo{BuiltAt: time.Unix(1, 0), Executor: ArtifactExecutorDAG}
	b.Run("MarshalArtifact", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := MarshalArtifact(snapshot, info); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("WriteArtifact", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := WriteArtifact(io.Discard, snapshot, info); err != nil {
				b.Fatal(err)
			}
		}
	})
}
