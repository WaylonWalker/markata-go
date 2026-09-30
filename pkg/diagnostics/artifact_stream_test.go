package diagnostics

import (
	"bytes"
	"encoding/json"
	"errors"
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
		{Entries: []ContentDisposition{{Path: "b.md"}, {Path: "a.md"}}},
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
