package cmd

import (
	"strings"
	"testing"
)

func TestFontReportSummaryLabelsGeneratedAssetsNotNetworkTransfers(t *testing.T) {
	got := fontReportSummary("handwritten", "bundled", 4, 35, 4144008)
	want := "Pack: handwritten\nPerformance class: bundled\nFamilies: 4\nFiles emitted: 35\nEmitted font asset bytes: 4144008"
	if got != want {
		t.Fatalf("font report summary mismatch:\ngot:  %q\nwant: %q", got, want)
	}
	if strings.Contains(got, "Transferred") {
		t.Fatalf("font report must not claim emitted assets were transferred: %q", got)
	}
}
