package diagnostics

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestTemplateCacheLedgerOwnershipAndReset(t *testing.T) {
	ledger := NewContentLedger()
	stats := TemplateCacheStats{Classified: 3, MissReasons: TemplateCacheMissReasons{EntryMissing: 2}}
	ledger.SetTemplateCache(&stats)
	want := stats
	stats.Classified = 100
	stats.MissReasons.EntryMissing = 100
	snapshot := ledger.Snapshot()
	if snapshot.TemplateCache == nil || *snapshot.TemplateCache != want {
		t.Fatalf("setter did not copy stats: %+v", snapshot.TemplateCache)
	}
	snapshot.TemplateCache.MissReasons.EntryMissing = 200
	if got := ledger.Snapshot().TemplateCache; got == nil || *got != want {
		t.Fatalf("snapshot did not copy stats: %+v", got)
	}
	for _, clear := range []func(){
		func() { ledger.SetTemplateCache(nil) },
		ledger.Reset,
		func() { ledger.Discover([]string{"post.md"}) },
	} {
		ledger.SetTemplateCache(&want)
		clear()
		if ledger.Snapshot().TemplateCache != nil {
			t.Fatal("stats retained after clear")
		}
	}
	var nilLedger *ContentLedger
	nilLedger.SetTemplateCache(&want)
}

func TestTemplateCacheLedgerConcurrentPublication(t *testing.T) {
	ledger := NewContentLedger()
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				stats := TemplateCacheStats{Classified: i, Restored: i}
				ledger.SetTemplateCache(&stats)
				snapshot := ledger.Snapshot()
				if snapshot.TemplateCache != nil && snapshot.TemplateCache.Classified != snapshot.TemplateCache.Restored {
					t.Error("partially published stats")
				}
				ledger.AddDiscovered("post.md")
			}
		}()
	}
	wg.Wait()
}

func TestTemplateCacheArtifactOptionalCopyAndStreamParity(t *testing.T) {
	info := ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)}
	nonzero := TemplateCacheStats{
		Classified: 12, Skipped: 1, Cacheable: 2, Restored: 1,
		RenderRequired: 11, RenderSucceeded: 8, RenderFailed: 2, ServeDeferred: 1,
		NavPreviewReset: true,
		MissReasons: TemplateCacheMissReasons{
			AffectedPath: 1, CacheUnavailable: 1, InputHashMissing: 1, EntryMissing: 1,
			InputHashMismatch: 1, TemplateMismatch: 1, DependencyChanged: 1,
			SlugChanged: 1, FeedMembershipChanged: 1, LocalPreviewChanged: 1, FullHTMLUnavailable: 1,
		},
	}
	for _, stats := range []*TemplateCacheStats{nil, {}, &nonzero} {
		for _, entries := range [][]ContentDisposition{nil, {}, {{Path: "z.md"}, {Path: "a.md"}}} {
			snapshot := ContentLedgerSnapshot{TemplateCache: stats, Entries: entries}
			before, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			want, err := MarshalArtifact(snapshot, info)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := WriteArtifact(&output, snapshot, info); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(output.Bytes(), want) {
				t.Fatalf("stream differs from marshal: %s != %s", output.Bytes(), want)
			}
			parsed, err := ParseArtifact(want)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(parsed.TemplateCache, stats) {
				t.Fatalf("stats changed on round trip: %+v != %+v", parsed.TemplateCache, stats)
			}
			if bytes.Contains(want, []byte(`"template_cache"`)) != (stats != nil) {
				t.Fatal("optional field presence incorrect")
			}
			after, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("artifact encoding mutated snapshot")
			}
			artifact := NewArtifact(snapshot, info)
			if artifact.TemplateCache != nil {
				artifact.TemplateCache.MissReasons.EntryMissing++
				if *artifact.TemplateCache == *snapshot.TemplateCache {
					t.Fatal("artifact shares stats with snapshot")
				}
			}
		}
	}
}
