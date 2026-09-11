package diagnostics

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarshalArtifact_PreservesSnapshotAndMetadata(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"z.md", "a.md", "static/site.css"})
	ledger.MarkLoaded("a.md")
	ledger.MarkFrontmatter("a.md", true, true)
	ledger.MarkPost("a.md", true)
	ledger.MarkRendered("a.md")
	ledger.MarkEmitted("a.md")
	ledger.RecordFeed("a.md", "archive", true)
	ledger.AddIssue(Issue{
		File:     "z.md",
		Code:     ReasonFrontmatterSuspiciousDelimiter,
		Severity: SeverityWarning,
		Message:  "expected the standard frontmatter delimiter",
	})

	snapshot := ledger.Snapshot()
	builtAt := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	data, err := MarshalArtifact(snapshot, ArtifactBuildInfo{
		MarkataVersion: "0.5.0",
		MarkataCommit:  "markata-commit",
		SourceCommit:   "source-commit",
		BuiltAt:        builtAt,
	})
	if err != nil {
		t.Fatalf("MarshalArtifact() error = %v", err)
	}
	second, err := MarshalArtifact(snapshot, ArtifactBuildInfo{
		MarkataVersion: "0.5.0",
		MarkataCommit:  "markata-commit",
		SourceCommit:   "source-commit",
		BuiltAt:        builtAt,
	})
	if err != nil {
		t.Fatalf("second MarshalArtifact() error = %v", err)
	}
	if !bytes.Equal(data, second) {
		t.Fatalf("artifact JSON is not deterministic:\n%s\n%s", data, second)
	}

	artifact, err := ParseArtifact(data)
	if err != nil {
		t.Fatalf("ParseArtifact() error = %v", err)
	}
	if artifact.SchemaVersion != ArtifactSchemaVersion || artifact.Schema != ArtifactSchema {
		t.Fatalf("artifact identity = %#v", artifact)
	}
	if artifact.Generator.Name != "markata-go" || artifact.Generator.Version != "0.5.0" || artifact.Generator.Commit != "markata-commit" {
		t.Fatalf("generator = %#v", artifact.Generator)
	}
	if artifact.Source == nil || artifact.Source.Commit != "source-commit" {
		t.Fatalf("source = %#v", artifact.Source)
	}
	if !artifact.BuiltAt.Equal(builtAt) {
		t.Fatalf("built_at = %s, want %s", artifact.BuiltAt, builtAt)
	}
	if artifact.Summary != snapshot.Summary {
		t.Fatalf("summary = %#v, want %#v", artifact.Summary, snapshot.Summary)
	}
	if len(artifact.Entries) != 3 || artifact.Entries[0].Path != "a.md" || artifact.Entries[1].Path != "static/site.css" || artifact.Entries[2].Path != "z.md" {
		t.Fatalf("entries are not sorted: %#v", artifact.Entries)
	}
	if len(artifact.Entries[2].Diagnostics) != 1 || artifact.Entries[2].Diagnostics[0].Code != ReasonFrontmatterSuspiciousDelimiter {
		t.Fatalf("diagnostics = %#v", artifact.Entries[2].Diagnostics)
	}
}

func TestMarshalArtifact_RedactsExternalPathsAndUnreliableCommits(t *testing.T) {
	externalPath := filepath.Join(string(filepath.Separator), "private", "secret.md")
	snapshot := ContentLedgerSnapshot{
		Entries: []ContentDisposition{{
			Path:      externalPath,
			Candidate: true,
			Diagnostics: []Issue{{
				File:     externalPath,
				Code:     ReasonContentLoadError,
				Severity: SeverityError,
				Message:  "PRIVATE_BODY_SECRET",
			}, {
				File:     externalPath,
				Code:     "third-party-diagnostic",
				Severity: SeverityWarning,
				Message:  "RAW_CONFIG_SECRET",
			}},
		}},
	}
	data, err := MarshalArtifact(snapshot, ArtifactBuildInfo{
		MarkataVersion: "dev",
		MarkataCommit:  "none",
		BuiltAt:        time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("MarshalArtifact() error = %v", err)
	}
	text := string(data)
	if strings.Contains(text, externalPath) || strings.Contains(text, "PRIVATE_BODY_SECRET") || strings.Contains(text, "RAW_CONFIG_SECRET") || strings.Contains(text, "\"commit\":\"none\"") {
		t.Fatalf("artifact leaked an external path or unreliable commit: %s", text)
	}
	artifact, err := ParseArtifact(data)
	if err != nil {
		t.Fatalf("ParseArtifact() error = %v", err)
	}
	if artifact.Generator.Commit != "" || artifact.Source != nil {
		t.Fatalf("unreliable commits were published: %#v", artifact)
	}
	if !strings.HasPrefix(artifact.Entries[0].Path, "__outside_content_root__/") || !strings.HasPrefix(artifact.Entries[0].Diagnostics[0].File, "__outside_content_root__/") {
		t.Fatalf("external paths were not redacted: %#v", artifact.Entries[0])
	}
	var foundThirdParty bool
	for _, issue := range artifact.Entries[0].Diagnostics {
		if issue.Code == "third-party-diagnostic" {
			foundThirdParty = true
			if issue.Message != "" {
				t.Fatalf("third-party diagnostic message = %q, want omitted", issue.Message)
			}
		}
	}
	if !foundThirdParty {
		t.Fatal("third-party diagnostic was lost")
	}
	if got := artifact.Entries[0].Diagnostics[0].Message; got != "content could not be loaded" {
		t.Fatalf("diagnostic message = %q, want safe canonical message", got)
	}
}

func TestParseArtifact_RejectsUnsupportedVersion(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-diagnostics/v1","schema":"markata.content-diagnostics","schema_version":2,"generator":{"name":"markata-go","version":"test"},"built_at":"2026-09-10T12:00:00Z","summary":{},"entries":[]}`)
	if _, err := ParseArtifact(data); err == nil {
		t.Fatal("ParseArtifact() accepted an unsupported schema version")
	}
}

func TestNewArtifact_CopiesEntries(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"post.md"})
	ledger.AddIssue(Issue{File: "post.md", Code: "custom", Severity: SeverityWarning, Message: "test warning"})
	snapshot := ledger.Snapshot()
	artifact := NewArtifact(snapshot, ArtifactBuildInfo{BuiltAt: time.Unix(0, 0)})
	artifact.Entries[0].Reasons[0] = "mutated"
	artifact.Entries[0].Diagnostics[0].Message = "mutated"
	artifact.Entries[0].Diagnostics = append(artifact.Entries[0].Diagnostics, Issue{Code: "extra"})

	if snapshot.Entries[0].Reasons[0] != ReasonContentNoOutput {
		t.Fatalf("snapshot reasons changed through artifact: %v", snapshot.Entries[0].Reasons)
	}
	if snapshot.Entries[0].Diagnostics[0].Message != "test warning" || len(snapshot.Entries[0].Diagnostics) != 1 {
		t.Fatalf("snapshot diagnostics changed through artifact: %+v", snapshot.Entries[0].Diagnostics)
	}
}

func TestMarshalArtifact_SortsDiagnosticsByAllSerializedFields(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"post.md"})
	ledger.AddIssue(Issue{
		File:     "post.md",
		Range:    Range{StartLine: 1, StartCol: 2, EndLine: 4, EndCol: 1},
		Code:     "same-code",
		Severity: SeverityWarning,
		Fixable:  true,
		Message:  "same message",
	})
	ledger.AddIssue(Issue{
		File:     "post.md",
		Range:    Range{StartLine: 1, StartCol: 2, EndLine: 3, EndCol: 1},
		Code:     "same-code",
		Severity: SeverityError,
		Fixable:  false,
		Message:  "same message",
	})

	artifact := NewArtifact(ledger.Snapshot(), ArtifactBuildInfo{})
	issues := artifact.Entries[0].Diagnostics
	if len(issues) != 2 {
		t.Fatalf("diagnostics = %#v, want two issues", issues)
	}
	if issues[0].Range.EndLine != 3 || issues[1].Range.EndLine != 4 {
		t.Fatalf("diagnostics were not sorted by end position: %#v", issues)
	}
}
