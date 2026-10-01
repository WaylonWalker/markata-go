package diagnostics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWriteArtifact_ByteParityAndImmutability(t *testing.T) {
	// The entry reasons and both feed reason slices deliberately overlap.
	reasons := []string{"z", "a", "a", "é<&\"\n", "last"}
	complexSnapshot := ContentLedgerSnapshot{
		Summary: ContentSummary{Discovered: 4, Candidates: 3, Warnings: 2},
		Entries: []ContentDisposition{
			{
				Path: "z/../é<&\"\n.md", Candidate: true, Reasons: reasons[:4],
				Diagnostics: []Issue{
					{File: `C:\private\secret.md`, Code: "unknown<&", Message: "RAW_SECRET", Severity: SeverityWarning},
					{File: "../private/secret.md", Code: ReasonContentLoadError, Message: "RAW_SECRET", Severity: SeverityError},
					{File: "./a.md", Code: ReasonFrontmatterParseError, Message: "RAW_SECRET", Range: Range{EndLine: 2}},
					{File: "a.md", Code: ReasonFrontmatterParseError, Message: "OTHER_SECRET", Range: Range{EndLine: 1}},
				},
				Feeds: []ContentFeedDisposition{
					{Feed: "z<&", Included: false, Reasons: reasons[1:4]},
					{Feed: "a", Included: false, Reasons: reasons[:3]},
					{Feed: "a", Included: true, Reasons: []string{"second duplicate"}},
				},
			},
			{Path: "/private/./secret.md", Loaded: true},
			{Path: "./a.md"},
			{Path: "/private/secret.md", Loaded: false},
			{Path: "a.md", Emitted: true},
			{Path: ""},
		},
	}
	for _, snapshot := range []ContentLedgerSnapshot{
		{},
		{Entries: []ContentDisposition{}},
		{Entries: []ContentDisposition{{Path: "single.md"}}},
		complexSnapshot,
	} {
		name := "empty"
		if len(snapshot.Entries) > 0 {
			name = snapshot.Entries[0].Path
		}
		t.Run(name, func(t *testing.T) {
			for _, executor := range []string{"", ArtifactExecutorLegacy, ArtifactExecutorDAG} {
				t.Run(executor, func(t *testing.T) {
					info := ArtifactBuildInfo{
						BuiltAt:        time.Date(2026, 9, 10, 12, 13, 14, 123, time.FixedZone("local", -5*60*60)),
						MarkataVersion: " test<&\"é\n ", MarkataCommit: " unknown ",
						SourceCommit: " source<&\"é ", Executor: executor,
					}
					if executor == "" {
						info.MarkataVersion = ""
						info.MarkataCommit = " dev "
						info.SourceCommit = " none "
					} else if executor == ArtifactExecutorLegacy {
						info.MarkataCommit = " reliable-commit "
					}
					before, err := json.Marshal(snapshot)
					if err != nil {
						t.Fatal(err)
					}
					reasonsBefore := append([]string(nil), reasons...)
					want, err := MarshalArtifact(snapshot, info)
					if err != nil {
						t.Fatal(err)
					}
					var got bytes.Buffer
					if err := WriteArtifact(&got, snapshot, info); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got.Bytes(), want) {
						t.Fatalf("stream differs from MarshalArtifact:\ngot:\n%s\nwant:\n%s", got.Bytes(), want)
					}
					if bytes.HasSuffix(got.Bytes(), []byte("\n")) {
						t.Fatal("stream added a trailing newline")
					}
					if strings.Contains(got.String(), "RAW_SECRET") || strings.Contains(got.String(), "OTHER_SECRET") {
						t.Fatal("stream leaked producer diagnostic text")
					}
					after, err := json.Marshal(snapshot)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(before, after) || !reflect.DeepEqual(reasons, reasonsBefore) {
						t.Fatal("artifact encoding mutated snapshot or aliased backing storage")
					}
					if len(snapshot.Entries) == 0 && !strings.Contains(got.String(), `"entries": []`) {
						t.Fatal("empty entries must be []")
					}
				})
			}
		})
	}
}

func TestWriteArtifact_StableNormalizedCollisions(t *testing.T) {
	external := "/private/secret.md"
	snapshot := ContentLedgerSnapshot{Entries: []ContentDisposition{
		{Path: "./same.md", Loaded: true, Feeds: []ContentFeedDisposition{
			{Feed: "duplicate", Included: true},
			{Feed: "duplicate", Included: false},
		}},
		{Path: external, Loaded: true},
		{Path: "same.md"},
		// An already-redacted path collides with the normalized absolute path.
		{Path: normalizeContentPath(external)},
	}}
	var buffer bytes.Buffer
	if err := WriteArtifact(&buffer, snapshot, ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)}); err != nil {
		t.Fatal(err)
	}
	artifact, err := ParseArtifact(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < len(artifact.Entries); index += 2 {
		first, second := artifact.Entries[index], artifact.Entries[index+1]
		if first.Path != second.Path || !first.Loaded || second.Loaded {
			t.Fatalf("normalized collision lost stable order: %#v", artifact.Entries)
		}
	}
	feeds := artifact.Entries[2].Feeds
	if len(feeds) != 2 || !feeds[0].Included || feeds[1].Included {
		t.Fatalf("duplicate feeds lost stable order or completeness: %#v", feeds)
	}
}

type artifactTestWriter struct {
	calls  int
	failAt int
	err    error
}

func (w *artifactTestWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		if w.err != nil {
			return 0, w.err
		}
		return len(data) - 1, nil
	}
	return len(data), nil
}

func TestWriteArtifact_WriterFailures(t *testing.T) {
	for _, snapshot := range []ContentLedgerSnapshot{
		{},
		{Entries: []ContentDisposition{{Path: "b.md", Feeds: []ContentFeedDisposition{
			{Feed: "same", Reasons: []string{"z", "a"}},
		}}, {Path: "a.md", Feeds: []ContentFeedDisposition{
			{Feed: "same", Reasons: []string{"a", "z"}},
		}}}},
	} {
		info := ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)}
		counter := &artifactTestWriter{}
		if err := WriteArtifact(counter, snapshot, info); err != nil {
			t.Fatal(err)
		}
		sentinel := errors.New("writer failure")
		for failAt := 1; failAt <= counter.calls; failAt++ {
			for _, failure := range []error{nil, sentinel} {
				writer := &artifactTestWriter{failAt: failAt, err: failure}
				want := failure
				if want == nil {
					want = io.ErrShortWrite
				}
				if err := WriteArtifact(writer, snapshot, info); !errors.Is(err, want) {
					t.Fatalf("write %d: error = %v, want %v", failAt, err, want)
				}
				if writer.calls != failAt {
					t.Fatalf("writer called after failure: %d calls, want %d", writer.calls, failAt)
				}
			}
		}
	}
}

func TestWriteArtifact_ScratchGrowthShrinkAndCacheBypass(t *testing.T) {
	shared := []string{"z", "a", "", "\x00|:\x1f", "é\u2028\u2029<&\"\\"}
	sharedFeeds := []ContentFeedDisposition{
		{Feed: "same\x00|:", Reasons: shared[:4]},
		{Feed: "same\x00|:", Included: true, Reasons: shared[1:]},
		{Feed: "same\x00|:", Reasons: nil},
		{Feed: "same\x00|:", Reasons: []string{}},
	}
	large := make([]ContentFeedDisposition, artifactFeedCacheEntries+100)
	for i := range large {
		large[i] = ContentFeedDisposition{Feed: fmt.Sprintf("%04d", i), Reasons: shared}
	}
	large = append(large, ContentFeedDisposition{
		Feed: strings.Repeat("oversized<&\x00", artifactFeedCacheBytes/8), Reasons: shared,
	})
	snapshot := ContentLedgerSnapshot{Entries: []ContentDisposition{
		{Path: "f.md", Feeds: sharedFeeds[1:]},
		{Path: "a.md", Feeds: sharedFeeds, Reasons: shared},
		{Path: "b.md", Feeds: large, Reasons: shared[1:]},
		{Path: "c.md"},
		{Path: "d.md", Feeds: []ContentFeedDisposition{}},
		{Path: "e.md", Feeds: sharedFeeds[:2]},
	}}
	info := ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)}
	before, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := MarshalArtifact(snapshot, info)
	if err != nil {
		t.Fatal(err)
	}
	for call := 0; call < 3; call++ {
		var got bytes.Buffer
		if err := WriteArtifact(&got, snapshot, info); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Bytes(), want) {
			t.Fatalf("call %d: bytes differ", call)
		}
	}
	after, err := json.Marshal(snapshot)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("scratch modified caller storage")
	}
}

func TestArtifactFeedFragments_ExactKeysAndRetentionBounds(t *testing.T) {
	for _, mode := range []string{"entry cap", "byte cap", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			var cache artifactFeedFragments
			// These would collide with naive delimiter keys. Include nulls,
			// empty reasons, raw JSON escapes and Unicode.
			feeds := []ContentFeedDisposition{
				{Feed: "a|b", Reasons: []string{"c"}},
				{Feed: "a", Reasons: []string{"b|c"}},
				{Feed: "a", Reasons: []string{"", "\x00:\x1f", "\u2028<&\"\\"}},
				{Feed: "a", Included: true},
				{Feed: "a", Reasons: []string{}},
				{Feed: "a", Reasons: nil},
			}
			for i := 0; i < artifactFeedCacheEntries+100; i++ {
				name := fmt.Sprintf("%04d", i)
				if mode == "byte cap" {
					name += strings.Repeat("x", 2048)
				} else if mode == "oversized" {
					name += strings.Repeat("x", artifactFeedCacheBytes)
					if i > 1 {
						break
					}
				}
				feeds = append(feeds, ContentFeedDisposition{Feed: name})
			}
			for _, feed := range feeds {
				want, err := json.MarshalIndent(feed, "        ", "  ")
				if err != nil {
					t.Fatal(err)
				}
				for repeat := 0; repeat < 2; repeat++ {
					got, err := cache.encode(feed)
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("fragment differs: %v", err)
					}
				}
				if len(cache.cache) > artifactFeedCacheEntries || cache.retained > artifactFeedCacheBytes {
					t.Fatal("fragment retention exceeded cap")
				}
			}
			retained := 0
			for key, fragment := range cache.cache {
				retained += len(key) + len(fragment)
			}
			if retained != cache.retained {
				t.Fatalf("retention accounting = %d, actual %d", cache.retained, retained)
			}
			if mode == "entry cap" && len(cache.cache) != artifactFeedCacheEntries {
				t.Fatal("entry cap not exercised")
			}
			if mode == "byte cap" && len(cache.cache) >= artifactFeedCacheEntries {
				t.Fatal("byte cap not exercised")
			}
			if mode == "oversized" && len(cache.cache) != 5 {
				t.Fatalf("oversized values retained: %d entries", len(cache.cache))
			}
		})
	}
}

func TestNewArtifact_OwnsAllFeedStorage(t *testing.T) {
	reasons := []string{"z", "a", "z"}
	feeds := []ContentFeedDisposition{{Feed: "z", Reasons: reasons}, {Feed: "a", Reasons: reasons[1:]}}
	snapshot := ContentLedgerSnapshot{Entries: []ContentDisposition{
		{Path: "a.md", Reasons: reasons, Feeds: feeds},
		{Path: "b.md", Reasons: reasons, Feeds: feeds},
	}}
	first := NewArtifact(snapshot, ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)})
	second := NewArtifact(snapshot, ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)})
	before, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	otherBefore, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	siblingBefore, err := json.Marshal(first.Entries[1])
	if err != nil {
		t.Fatal(err)
	}
	first.Entries[0].Reasons[0] = "changed"
	first.Entries[0].Feeds[0].Feed = "changed"
	first.Entries[0].Feeds[0].Reasons[0] = "changed"
	first.Entries[0].Feeds[1].Reasons[0] = "changed"
	after, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	otherAfter, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	siblingAfter, err := json.Marshal(first.Entries[1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !bytes.Equal(otherBefore, otherAfter) || !bytes.Equal(siblingBefore, siblingAfter) {
		t.Fatal("NewArtifact shares mutable storage")
	}
}
func TestArtifactFeedLayoutGuard(t *testing.T) {
	wire := reflect.TypeFor[ContentDisposition]()
	if !artifactFeedLayoutSupported(wire) {
		t.Fatal("current v1 layout should support fragment assembly")
	}
	fields := make([]reflect.StructField, wire.NumField())
	for i := range fields {
		fields[i] = wire.Field(i)
	}
	fields[0], fields[len(fields)-1] = fields[len(fields)-1], fields[0]
	if artifactFeedLayoutSupported(reflect.StructOf(fields)) {
		t.Fatal("reordered fields must use whole-entry encoding")
	}
	fields[0], fields[len(fields)-1] = fields[len(fields)-1], fields[0]
	fields[0].Tag = `json:"feeds"`
	if artifactFeedLayoutSupported(reflect.StructOf(fields)) {
		t.Fatal("conflicting feeds tag must use whole-entry encoding")
	}
	if artifactFeedLayoutSupported(reflect.TypeFor[artifactCustomDisposition]()) {
		t.Fatal("custom marshaler must use whole-entry encoding")
	}
}

type artifactCustomDisposition ContentDisposition

func (artifactCustomDisposition) MarshalJSON() ([]byte, error) {
	return []byte(`{}`), nil
}

func TestWriteArtifact_IndependentConcurrentCalls(t *testing.T) {
	snapshot := ContentLedgerSnapshot{Entries: []ContentDisposition{{
		Path: "post.md", Feeds: []ContentFeedDisposition{{Feed: "shared", Reasons: []string{"z", "a"}}},
	}}}
	info := ArtifactBuildInfo{BuiltAt: time.Unix(1, 0)}
	want, err := MarshalArtifact(snapshot, info)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("concurrent", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				var got bytes.Buffer
				if err := WriteArtifact(&got, snapshot, info); err != nil || !bytes.Equal(got.Bytes(), want) {
					t.Fatalf("independent call differs: %v", err)
				}
			})
		}
	})
	snapshot.Entries[0].Feeds[0].Reasons[0] = "new value"
	want, err = MarshalArtifact(snapshot, info)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := WriteArtifact(&got, snapshot, info); err != nil || !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("repeated call retained stale fragment: %v", err)
	}
}

func TestWriteArtifact_SerializationAndValidationFailures(t *testing.T) {
	for _, info := range []ArtifactBuildInfo{
		{Executor: "invalid", BuiltAt: time.Unix(1, 0)},
		{BuiltAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)},
	} {
		snapshot := ContentLedgerSnapshot{Entries: []ContentDisposition{{Path: "a.md"}}}
		_, marshalErr := MarshalArtifact(snapshot, info)
		var buffer bytes.Buffer
		streamErr := WriteArtifact(&buffer, snapshot, info)
		if marshalErr == nil || streamErr == nil || marshalErr.Error() != streamErr.Error() {
			t.Fatalf("errors differ: marshal = %v, stream = %v", marshalErr, streamErr)
		}
		if buffer.Len() != 0 {
			t.Fatal("invalid header wrote partial output")
		}
	}
}
