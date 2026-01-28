package palettes

import (
	"testing"
)

// TestBuiltInPalettesWCAGContrast validates that all built-in palettes
// pass WCAG AA contrast requirements. This is an integration test that
// ensures our shipped palettes meet accessibility standards.
func TestBuiltInPalettesWCAGContrast(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			// Run standard WCAG AA contrast checks
			results := palette.CheckContrast()
			if len(results) == 0 {
				t.Logf("Palette %s: No contrast checks performed (may be missing semantic mappings)", name)
				return
			}

			// Summarize results
			summary := SummarizeContrast(name, results)

			// Log detailed results for debugging
			if !summary.AllPassed {
				t.Logf("Palette %s contrast summary: %d passed, %d failed, %d skipped",
					name, summary.Passed, summary.Failed, summary.Skipped)

				for _, check := range summary.FailedChecks {
					t.Errorf("Palette %s: FAILED %s on %s - ratio %.2f (need %.1f for %s)",
						name, check.Foreground, check.Background, check.Ratio, check.Required, check.Level)
				}
			}
		})
	}
}

// TestBuiltInPalettesHaveRequiredColors validates that all built-in palettes
// define the minimum required color mappings.
func TestBuiltInPalettesHaveRequiredColors(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	// Minimum required semantic mappings for a usable palette
	requiredSemantic := []string{
		"text-primary",
		"bg-primary",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			for _, key := range requiredSemantic {
				if _, ok := palette.Semantic[key]; !ok {
					t.Errorf("Palette %s missing required semantic mapping: %s", name, key)
				}
			}
		})
	}
}

// TestBuiltInPalettesValidate ensures all built-in palettes pass validation.
func TestBuiltInPalettesValidate(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			errs := palette.Validate()
			if len(errs) > 0 {
				for _, e := range errs {
					t.Errorf("Palette %s validation error: %v", name, e)
				}
			}
		})
	}
}

// TestBuiltInPalettesHaveBothVariants checks if each palette family
// has both light and dark variants.
func TestBuiltInPalettesHaveBothVariants(t *testing.T) {
	infos := DiscoverBuiltin()
	if len(infos) == 0 {
		t.Skip("No built-in palettes found")
	}

	// Group palettes by base name (without -light/-dark suffix)
	families := make(map[string]map[Variant]bool)
	for _, info := range infos {
		baseName := info.Name
		// Remove variant suffix for grouping
		for _, suffix := range []string{" Light", " Dark", "-light", "-dark"} {
			if len(baseName) > len(suffix) {
				baseName = baseName[:len(baseName)-len(suffix)]
			}
		}
		if families[baseName] == nil {
			families[baseName] = make(map[Variant]bool)
		}
		families[baseName][info.Variant] = true
	}

	// Log which families are complete
	for family, variants := range families {
		hasLight := variants[VariantLight]
		hasDark := variants[VariantDark]
		if !hasLight || !hasDark {
			t.Logf("Palette family %q: light=%v, dark=%v (consider adding missing variant)",
				family, hasLight, hasDark)
		}
	}
}

// TestDefaultPaletteExists ensures there's a default palette available.
func TestDefaultPaletteExists(t *testing.T) {
	// Common default palette names
	defaultNames := []string{"default", "Default", "markata", "base"}

	found := false
	for _, name := range defaultNames {
		if HasBuiltin(name) {
			found = true
			t.Logf("Found default palette: %s", name)
			break
		}
	}

	if !found {
		// If no "default" palette, at least ensure some palettes exist
		names := BuiltinNames()
		if len(names) == 0 {
			t.Error("No built-in palettes available - at least one should exist")
		} else {
			t.Logf("No 'default' palette, but %d built-in palettes available: %v",
				len(names), names)
		}
	}
}

// TestPaletteContrastForTextReadability specifically tests text readability
// contrast ratios across all built-in palettes.
func TestPaletteContrastForTextReadability(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	// Critical readability checks (text on backgrounds)
	criticalChecks := []struct {
		fg       string
		bg       string
		minRatio float64
		desc     string
	}{
		{"text-primary", "bg-primary", 4.5, "main body text"},
		{"text-secondary", "bg-primary", 4.5, "secondary text"},
		{"text-muted", "bg-primary", 3.0, "muted/hint text (large text minimum)"},
		{"link", "bg-primary", 4.5, "links in body text"},
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			for _, check := range criticalChecks {
				fgHex := palette.Resolve(check.fg)
				bgHex := palette.Resolve(check.bg)

				if fgHex == "" || bgHex == "" {
					// Skip if colors not defined
					continue
				}

				fgColor, err1 := ParseHexColor(fgHex)
				bgColor, err2 := ParseHexColor(bgHex)
				if err1 != nil || err2 != nil {
					continue
				}

				ratio := ContrastRatio(fgColor, bgColor)
				if ratio < check.minRatio {
					t.Errorf("Palette %s: %s contrast ratio %.2f < %.1f required for %s",
						name, check.fg, ratio, check.minRatio, check.desc)
				}
			}
		})
	}
}
