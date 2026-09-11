package diagnostics

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestContentLedger_Snapshot(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"z.md", "assets/site.css", "./a.md", "a.md"})

	ledger.MarkLoaded("a.md")
	ledger.MarkFrontmatter("a.md", true, true)
	ledger.MarkPost("a.md", true)
	ledger.MarkRendered("a.md")
	ledger.MarkEmitted("a.md")
	ledger.RecordFeed("a.md", "home", true)
	ledger.RecordFeed("a.md", "archive", false, ReasonContentFiltered)

	ledger.MarkLoaded("z.md")
	ledger.MarkFrontmatter("z.md", true, false)
	ledger.MarkPost("z.md", false)
	ledger.AddIssue(Issue{
		File:     "z.md",
		Code:     ReasonFrontmatterParseError,
		Severity: SeverityError,
		Message:  "invalid YAML",
	})
	ledger.AddIssue(Issue{
		File:     "z.md",
		Code:     ReasonFrontmatterParseError,
		Severity: SeverityError,
		Message:  "invalid YAML",
	})

	snapshot := ledger.Snapshot()
	if len(snapshot.Entries) != 3 {
		t.Fatalf("entry count = %d, want 3", len(snapshot.Entries))
	}
	if got := snapshot.Entries[0].Path; got != "a.md" {
		t.Errorf("first entry = %q, want a.md", got)
	}
	if got := snapshot.Entries[1].Disposition; got != DispositionNotCandidate {
		t.Errorf("asset disposition = %q, want %q", got, DispositionNotCandidate)
	}
	if got := snapshot.Entries[2].Disposition; got != DispositionExcluded {
		t.Errorf("z.md disposition = %q, want %q", got, DispositionExcluded)
	}

	if snapshot.Summary != (ContentSummary{
		Discovered:       3,
		Candidates:       2,
		Loaded:           2,
		FrontmatterValid: 1,
		Posts:            2,
		Eligible:         1,
		Rendered:         1,
		Emitted:          1,
		Excluded:         1,
		Errors:           1,
	}) {
		t.Errorf("summary = %+v", snapshot.Summary)
	}

	a := snapshot.Entries[0]
	if a.Disposition != DispositionEmitted || a.Excluded {
		t.Errorf("a.md state = %+v", a)
	}
	if len(a.Diagnostics) != 0 {
		t.Errorf("a.md diagnostics = %+v, want none", a.Diagnostics)
	}
	if len(a.Feeds) != 2 || a.Feeds[0].Feed != "archive" || a.Feeds[0].Included {
		t.Errorf("a.md feeds = %+v", a.Feeds)
	}
	if len(a.Reasons) != 1 || a.Reasons[0] != ReasonContentFiltered {
		t.Errorf("a.md reasons = %v, want filtered", a.Reasons)
	}

	z := snapshot.Entries[2]
	if len(z.Diagnostics) != 1 {
		t.Errorf("z.md diagnostics = %d, want duplicate diagnostics removed", len(z.Diagnostics))
	}
	if len(z.Reasons) != 1 || z.Reasons[0] != ReasonFrontmatterParseError {
		t.Errorf("z.md reasons = %v", z.Reasons)
	}
}

func TestContentLedger_SnapshotIsCopy(t *testing.T) {
	ledger := NewContentLedger()
	ledger.Discover([]string{"post.md"})
	ledger.AddIssue(Issue{File: "post.md", Code: "custom", Severity: SeverityWarning, Message: "warning"})

	snapshot := ledger.Snapshot()
	snapshot.Entries[0].Reasons = append(snapshot.Entries[0].Reasons, "mutated")
	snapshot.Entries[0].Diagnostics[0].Message = "mutated"

	again := ledger.Snapshot()
	if len(again.Entries[0].Reasons) != 2 || again.Entries[0].Reasons[0] != ReasonContentNoOutput || again.Entries[0].Reasons[1] != "diagnostic.custom" {
		t.Errorf("ledger reasons changed through snapshot: %v", again.Entries[0].Reasons)
	}
	if again.Entries[0].Diagnostics[0].Message != "warning" {
		t.Errorf("ledger diagnostics changed through snapshot: %+v", again.Entries[0].Diagnostics)
	}
}

func TestIsContentCandidate(t *testing.T) {
	for _, path := range []string{"post.md", "post.MARKDOWN", "post.adoc", "post.html"} {
		if !IsContentCandidate(path) {
			t.Errorf("IsContentCandidate(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"site.css", "image.png", "data.json", "README"} {
		if IsContentCandidate(path) {
			t.Errorf("IsContentCandidate(%q) = true, want false", path)
		}
	}
}

func TestReasonForDiagnosticCode_PreservesFeedWindowCodes(t *testing.T) {
	for _, code := range []string{ReasonFeedOffset, ReasonFeedLimit} {
		if got := ReasonForDiagnosticCode(code); got != code {
			t.Errorf("ReasonForDiagnosticCode(%q) = %q, want %q", code, got, code)
		}
	}
}

func TestContentLedger_RedactsExternalPaths(t *testing.T) {
	path := filepath.Join("..", "private", "post.md")
	ledger := NewContentLedger()
	ledger.Discover([]string{path})
	ledger.MarkLoaded(path)

	snapshot := ledger.Snapshot()
	if len(snapshot.Entries) != 1 {
		t.Fatalf("entries = %+v, want one entry", snapshot.Entries)
	}
	got := snapshot.Entries[0].Path
	if filepath.IsAbs(got) || got == ".." || strings.HasPrefix(got, "../") {
		t.Fatalf("external path was not redacted: %q", got)
	}
	if !strings.HasPrefix(got, "__outside_content_root__/") {
		t.Fatalf("external path = %q, want redaction marker", got)
	}
}
