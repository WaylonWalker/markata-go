package cmd

import "testing"

func TestFontReportLabelsOutputBytesAsEmittedAssets(t *testing.T) {
	if got, want := fontReportAssetBytesLine(12345), "Emitted font asset bytes: 12345"; got != want {
		t.Fatalf("font report byte line = %q, want %q", got, want)
	}
}
