package palettes

import (
	"strings"
	"testing"
)

func TestPalette_CheckContrast(t *testing.T) {
	// Create a palette with known contrast values
	p := &Palette{
		Name:    "Test Palette",
		Variant: VariantDark,
		Colors: map[string]string{
			"white":     "#ffffff",
			"black":     "#000000",
			"darkgray":  "#333333",
			"lightgray": "#cccccc",
		},
		Semantic: map[string]string{
			"text-primary":   "white",
			"text-secondary": "lightgray",
			"text-muted":     "lightgray",
			"bg-primary":     "black",
			"bg-surface":     "darkgray",
			"bg-elevated":    "darkgray",
			"link":           "white",
			"accent":         "white",
			"success":        "white",
			"warning":        "white",
			"error":          "white",
			"info":           "white",
		},
		Components: map[string]string{
			"code-text":             "white",
			"code-bg":               "darkgray",
			"code-comment":          "lightgray",
			"code-keyword":          "white",
			"button-primary-text":   "black",
			"button-primary-bg":     "white",
			"button-secondary-text": "white",
			"button-secondary-bg":   "darkgray",
		},
	}

	results := p.CheckContrast()

	// Should have results for all required checks
	if len(results) == 0 {
		t.Fatal("CheckContrast() returned no results")
	}

	// White on black should always pass
	for _, r := range results {
		if r.Foreground == "text-primary" && r.Background == "bg-primary" {
			if !r.Passed {
				t.Errorf("text-primary on bg-primary should pass, got ratio %.2f", r.Ratio)
			}
			if r.Ratio < 20 {
				t.Errorf("white on black ratio should be ~21, got %.2f", r.Ratio)
			}
		}
	}
}

func TestPalette_CheckContrastStrict(t *testing.T) {
	p := &Palette{
		Name:    "Test",
		Variant: VariantDark,
		Colors: map[string]string{
			"white": "#ffffff",
			"black": "#000000",
		},
		Semantic: map[string]string{
			"text-primary":   "white",
			"text-secondary": "white",
			"bg-primary":     "black",
			"link":           "white",
		},
	}

	results := p.CheckContrastStrict()

	// Should have more results than regular check (includes AAA)
	regularResults := p.CheckContrast()

	if len(results) <= len(regularResults) {
		t.Errorf("CheckContrastStrict() should return more results than CheckContrast()")
	}
}

func TestSummarizeContrast(t *testing.T) {
	results := []ContrastCheck{
		{Foreground: "text", Background: "bg", ForegroundHex: "#fff", BackgroundHex: "#000", Ratio: 21.0, Required: 4.5, Passed: true},
		{Foreground: "link", Background: "bg", ForegroundHex: "#aaa", BackgroundHex: "#000", Ratio: 3.0, Required: 4.5, Passed: false},
		{Foreground: "unknown", Background: "bg", ForegroundHex: "", BackgroundHex: "#000", Ratio: 0, Required: 4.5, Passed: false},
	}

	summary := SummarizeContrast("test", results)

	if summary.Palette != "test" {
		t.Errorf("Palette = %q, want %q", summary.Palette, "test")
	}
	if summary.Total != 3 {
		t.Errorf("Total = %d, want 3", summary.Total)
	}
	if summary.Passed != 1 {
		t.Errorf("Passed = %d, want 1", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", summary.Skipped)
	}
	if summary.AllPassed {
		t.Error("AllPassed should be false")
	}
	if len(summary.FailedChecks) != 1 {
		t.Errorf("FailedChecks count = %d, want 1", len(summary.FailedChecks))
	}
}

func TestFormatContrastResult(t *testing.T) {
	tests := []struct {
		name   string
		result ContrastCheck
		want   string
	}{
		{
			name: "passed",
			result: ContrastCheck{
				Foreground:    "text",
				Background:    "bg",
				ForegroundHex: "#fff",
				BackgroundHex: "#000",
				Ratio:         21.0,
				Required:      4.5,
				Passed:        true,
				PassedLevels:  []string{"A", "AA", "AAA"},
			},
			want: "\u2713", // checkmark
		},
		{
			name: "failed",
			result: ContrastCheck{
				Foreground:    "text",
				Background:    "bg",
				ForegroundHex: "#888",
				BackgroundHex: "#777",
				Ratio:         1.2,
				Required:      4.5,
				Passed:        false,
			},
			want: "\u2717", // X mark
		},
		{
			name: "skipped",
			result: ContrastCheck{
				Foreground:    "unknown",
				Background:    "bg",
				ForegroundHex: "",
				BackgroundHex: "#000",
			},
			want: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatContrastResult(tt.result)
			if !strings.Contains(got, tt.want) {
				t.Errorf("FormatContrastResult() should contain %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFormatContrastSummary(t *testing.T) {
	summary := ContrastSummary{
		Palette:   "Test Palette",
		Total:     10,
		Passed:    8,
		Failed:    1,
		Skipped:   1,
		AllPassed: false,
		FailedChecks: []ContrastCheck{
			{Foreground: "text", Background: "bg", Ratio: 2.0, Required: 4.5, Level: "AA"},
		},
	}

	got := FormatContrastSummary(summary)

	if !strings.Contains(got, "Test Palette") {
		t.Error("Summary should contain palette name")
	}
	if !strings.Contains(got, "8 passed") {
		t.Error("Summary should contain passed count")
	}
	if !strings.Contains(got, "1 failed") {
		t.Error("Summary should contain failed count")
	}
	if !strings.Contains(got, "Failed checks") {
		t.Error("Summary should contain failed checks section")
	}
}

func TestRequiredChecks(t *testing.T) {
	// Verify RequiredChecks has expected entries
	if len(RequiredChecks) == 0 {
		t.Fatal("RequiredChecks should not be empty")
	}

	// Check for essential entries
	essentialChecks := []string{
		"text-primary",
		"link",
		"accent",
	}

	for _, name := range essentialChecks {
		found := false
		for _, check := range RequiredChecks {
			if check.Foreground == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RequiredChecks missing check for %q", name)
		}
	}
}

func TestContrastCheckSpec(t *testing.T) {
	spec := ContrastCheckSpec{
		Foreground: "text",
		Background: "bg",
		MinRatio:   4.5,
		Level:      "AA",
	}

	if spec.Foreground != "text" {
		t.Errorf("Foreground = %q, want %q", spec.Foreground, "text")
	}
	if spec.MinRatio != 4.5 {
		t.Errorf("MinRatio = %v, want 4.5", spec.MinRatio)
	}
}
