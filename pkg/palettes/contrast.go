package palettes

import (
	"fmt"
	"sort"
)

// ContrastCheck represents a single contrast validation result.
type ContrastCheck struct {
	Foreground    string   `json:"foreground"`     // Foreground color name
	Background    string   `json:"background"`     // Background color name
	ForegroundHex string   `json:"foreground_hex"` // Resolved foreground hex
	BackgroundHex string   `json:"background_hex"` // Resolved background hex
	Ratio         float64  `json:"ratio"`          // Calculated contrast ratio
	Required      float64  `json:"required"`       // Minimum required ratio
	Level         string   `json:"level"`          // WCAG level: "AA", "AAA", "AA Large", "UI"
	Passed        bool     `json:"passed"`         // Whether the check passed
	PassedLevels  []string `json:"passed_levels"`  // All WCAG levels that pass
}

// ContrastCheckSpec defines a contrast check to perform.
type ContrastCheckSpec struct {
	Foreground string  // Foreground color name (semantic or component)
	Background string  // Background color name (semantic or component)
	MinRatio   float64 // Minimum required ratio
	Level      string  // WCAG level description
}

// RequiredChecks defines all contrast checks that palettes should pass.
// These are the minimum requirements for WCAG AA compliance.
var RequiredChecks = []ContrastCheckSpec{
	// Primary text must be readable
	{"text-primary", "bg-primary", 4.5, "AA"},
	{"text-primary", "bg-surface", 4.5, "AA"},
	{"text-primary", "bg-elevated", 4.5, "AA"},

	// Secondary text
	{"text-secondary", "bg-primary", 4.5, "AA"},
	{"text-muted", "bg-primary", 3.0, "AA Large"},

	// Interactive elements
	{"link", "bg-primary", 4.5, "AA"},
	{"accent", "bg-primary", 3.0, "AA Large"},

	// Status colors (used for UI, so 3:1 minimum)
	{"success", "bg-primary", 3.0, "UI"},
	{"warning", "bg-primary", 3.0, "UI"},
	{"error", "bg-primary", 3.0, "UI"},
	{"info", "bg-primary", 3.0, "UI"},

	// Code blocks
	{"code-text", "code-bg", 4.5, "AA"},
	{"code-comment", "code-bg", 3.0, "AA Large"},
	{"code-keyword", "code-bg", 4.5, "AA"},

	// Buttons
	{"button-primary-text", "button-primary-bg", 4.5, "AA"},
	{"button-secondary-text", "button-secondary-bg", 4.5, "AA"},
}

// StrictChecks defines additional checks for AAA compliance.
var StrictChecks = []ContrastCheckSpec{
	// AAA text requirements (7:1 for normal text)
	{"text-primary", "bg-primary", 7.0, "AAA"},
	{"text-secondary", "bg-primary", 7.0, "AAA"},
	{"link", "bg-primary", 7.0, "AAA"},

	// AAA large text (4.5:1)
	{"text-muted", "bg-primary", 4.5, "AAA Large"},
	{"accent", "bg-primary", 4.5, "AAA Large"},
}

// CheckContrast checks all required contrast ratios for the palette.
// Returns a slice of ContrastCheck results.
func (p *Palette) CheckContrast() []ContrastCheck {
	return p.checkContrastWith(RequiredChecks)
}

// CheckContrastStrict checks both required and strict contrast ratios.
// Returns a slice of ContrastCheck results including AAA level checks.
func (p *Palette) CheckContrastStrict() []ContrastCheck {
	allChecks := make([]ContrastCheckSpec, 0, len(RequiredChecks)+len(StrictChecks))
	allChecks = append(allChecks, RequiredChecks...)
	allChecks = append(allChecks, StrictChecks...)
	return p.checkContrastWith(allChecks)
}

// CheckContrastWith checks contrast using custom check specifications.
func (p *Palette) CheckContrastWith(specs []ContrastCheckSpec) []ContrastCheck {
	return p.checkContrastWith(specs)
}

// checkContrastWith performs contrast checks against given specs.
func (p *Palette) checkContrastWith(specs []ContrastCheckSpec) []ContrastCheck {
	results := make([]ContrastCheck, 0, len(specs))

	for _, spec := range specs {
		result := p.checkSingleContrast(spec)
		results = append(results, result)
	}

	return results
}

// checkSingleContrast performs a single contrast check.
func (p *Palette) checkSingleContrast(spec ContrastCheckSpec) ContrastCheck {
	result := ContrastCheck{
		Foreground: spec.Foreground,
		Background: spec.Background,
		Required:   spec.MinRatio,
		Level:      spec.Level,
	}

	// Resolve colors
	fgHex := p.Resolve(spec.Foreground)
	bgHex := p.Resolve(spec.Background)

	result.ForegroundHex = fgHex
	result.BackgroundHex = bgHex

	// If either color can't be resolved, mark as failed
	if fgHex == "" || bgHex == "" {
		result.Passed = false
		return result
	}

	// Calculate contrast ratio
	ratio, err := ContrastRatioFromHex(fgHex, bgHex)
	if err != nil {
		result.Passed = false
		return result
	}

	result.Ratio = ratio
	result.Passed = ratio >= spec.MinRatio

	// Determine all passed levels
	isLargeText := spec.Level == "AA Large" || spec.Level == "AAA Large"
	for _, level := range []WCAGLevel{WCAGLevelA, WCAGLevelAA, WCAGLevelAAA} {
		if MeetsWCAG(ratio, level, isLargeText) {
			result.PassedLevels = append(result.PassedLevels, string(level))
		}
	}

	return result
}

// ContrastSummary provides a summary of contrast check results.
type ContrastSummary struct {
	Palette      string          `json:"palette"`
	Total        int             `json:"total"`
	Passed       int             `json:"passed"`
	Failed       int             `json:"failed"`
	Skipped      int             `json:"skipped"` // Colors not found
	AllPassed    bool            `json:"all_passed"`
	FailedChecks []ContrastCheck `json:"failed_checks,omitempty"`
}

// SummarizeContrast generates a summary from contrast check results.
func SummarizeContrast(paletteName string, results []ContrastCheck) ContrastSummary {
	summary := ContrastSummary{
		Palette: paletteName,
		Total:   len(results),
	}

	for i := range results {
		r := &results[i]
		switch {
		case r.ForegroundHex == "" || r.BackgroundHex == "":
			summary.Skipped++
		case r.Passed:
			summary.Passed++
		default:
			summary.Failed++
			summary.FailedChecks = append(summary.FailedChecks, *r)
		}
	}

	summary.AllPassed = summary.Failed == 0

	return summary
}

// FormatContrastResult formats a single contrast check result as a string.
func FormatContrastResult(r ContrastCheck) string {
	status := "\u2713" // checkmark
	if !r.Passed {
		status = "\u2717" // X
	}
	if r.ForegroundHex == "" || r.BackgroundHex == "" {
		status = "?" // Unknown
	}

	levelsStr := ""
	if len(r.PassedLevels) > 0 {
		levelsStr = fmt.Sprintf(" (%s)", joinLevels(r.PassedLevels))
	}

	return fmt.Sprintf("  %s %s on %s: %.1f:1%s",
		status,
		r.Foreground,
		r.Background,
		r.Ratio,
		levelsStr,
	)
}

// FormatContrastSummary formats a contrast summary as a human-readable string.
func FormatContrastSummary(summary ContrastSummary) string {
	result := fmt.Sprintf("Contrast Check: %s\n\n", summary.Palette)

	if summary.AllPassed {
		result += fmt.Sprintf("All %d checks passed!\n", summary.Passed)
	} else {
		result += fmt.Sprintf("Results: %d passed, %d failed", summary.Passed, summary.Failed)
		if summary.Skipped > 0 {
			result += fmt.Sprintf(", %d skipped (colors not found)", summary.Skipped)
		}
		result += "\n\nFailed checks:\n"
		for i := range summary.FailedChecks {
			fc := &summary.FailedChecks[i]
			result += fmt.Sprintf("  %s on %s: %.1f:1 (required %.1f:1 for %s)\n",
				fc.Foreground,
				fc.Background,
				fc.Ratio,
				fc.Required,
				fc.Level,
			)
		}
	}

	return result
}

// joinLevels joins WCAG level strings.
func joinLevels(levels []string) string {
	if len(levels) == 0 {
		return ""
	}
	result := levels[0]
	for i := 1; i < len(levels); i++ {
		result += ", " + levels[i]
	}
	return result
}

// AllContrastPairs returns all unique foreground/background pairs in the palette.
// Useful for comprehensive contrast checking.
func (p *Palette) AllContrastPairs() []ContrastCheckSpec {
	// Collect all foreground-like colors
	fgColors := []string{}
	bgColors := []string{}

	// Identify semantic colors by naming convention
	for name := range p.Semantic {
		if isLikelyForeground(name) {
			fgColors = append(fgColors, name)
		}
		if isLikelyBackground(name) {
			bgColors = append(bgColors, name)
		}
	}

	// Also check component colors
	for name := range p.Components {
		if isLikelyForeground(name) {
			fgColors = append(fgColors, name)
		}
		if isLikelyBackground(name) {
			bgColors = append(bgColors, name)
		}
	}

	// Sort for consistent output
	sort.Strings(fgColors)
	sort.Strings(bgColors)

	// Generate all pairs
	var pairs []ContrastCheckSpec
	for _, fg := range fgColors {
		for _, bg := range bgColors {
			pairs = append(pairs, ContrastCheckSpec{
				Foreground: fg,
				Background: bg,
				MinRatio:   4.5, // Default to AA text requirement
				Level:      "AA",
			})
		}
	}

	return pairs
}

// isLikelyForeground checks if a color name suggests it's used as foreground.
func isLikelyForeground(name string) bool {
	fgIndicators := []string{
		"text", "link", "accent", "success", "warning", "error", "info",
		"-text", "-fg", "-color",
	}
	for _, indicator := range fgIndicators {
		if contains(name, indicator) {
			return true
		}
	}
	return false
}

// isLikelyBackground checks if a color name suggests it's used as background.
func isLikelyBackground(name string) bool {
	bgIndicators := []string{
		"bg-", "bg", "-bg", "surface", "elevated", "background",
	}
	for _, indicator := range bgIndicators {
		if contains(name, indicator) {
			return true
		}
	}
	return false
}

// contains checks if s contains substr (case-insensitive prefix/suffix match).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr))
}

// findInString checks if substr is anywhere in s.
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
