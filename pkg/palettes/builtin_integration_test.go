package palettes

import (
	"testing"
)

func resolveFirst(palette *Palette, names ...string) (name, hex string) {
	for _, name := range names {
		if resolved := palette.Resolve(name); resolved != "" {
			return name, resolved
		}
	}
	return "", ""
}

func requireContrast(t *testing.T, paletteName, fgName, fgHex, bgName, bgHex string, minRatio float64, desc string) {
	t.Helper()

	ratio, err := ContrastRatioFromHex(fgHex, bgHex)
	if err != nil {
		t.Fatalf("Palette %s: invalid resolved colors for %s: %q on %q (%v)", paletteName, desc, fgHex, bgHex, err)
	}

	if ratio < minRatio {
		t.Errorf("Palette %s: %s (%s on %s) contrast ratio %.2f < %.1f required", paletteName, desc, fgName, bgName, ratio, minRatio)
	}
}

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

// TestPaletteContrastForDefaultThemeReadability specifically tests text readability
// contrast ratios used by the bundled default theme across all built-in palettes.
func TestPaletteContrastForDefaultThemeReadability(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	// Critical readability checks from default theme CSS token usage.
	// Lighthouse treats small muted UI labels as normal text, so require AA 4.5:1.
	defaultThemeChecks := []struct {
		fg       string
		bg       string
		minRatio float64
		desc     string
	}{
		{"text-primary", "bg-primary", 4.5, "main body text"},
		{"text-primary", "bg-surface", 4.5, "text on cards and surfaces"},
		{"text-primary", "bg-elevated", 4.5, "text on elevated surfaces"},
		{"text-secondary", "bg-primary", 4.5, "secondary text"},
		{"text-secondary", "bg-surface", 4.5, "secondary text on cards and surfaces"},
		{"text-muted", "bg-primary", 4.5, "muted normal-size text"},
		{"text-muted", "bg-secondary", 4.5, "muted text on secondary backgrounds"},
		{"text-muted", "bg-surface", 4.5, "muted text on cards and surfaces"},
		{"text-muted", "bg-elevated", 4.5, "muted text on elevated surfaces"},
		{"link", "bg-primary", 4.5, "links in body text"},
		{"link", "bg-surface", 4.5, "links on cards and surfaces"},
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			for _, check := range defaultThemeChecks {
				fgHex := palette.Resolve(check.fg)
				bgHex := palette.Resolve(check.bg)

				if fgHex == "" || bgHex == "" {
					t.Errorf("Palette %s: missing color mapping for %s on %s used by default theme",
						name, check.fg, check.bg)
					continue
				}

				fgColor, err1 := ParseHexColor(fgHex)
				bgColor, err2 := ParseHexColor(bgHex)
				if err1 != nil || err2 != nil {
					t.Errorf("Palette %s: invalid resolved colors for %s on %s: %q on %q",
						name, check.fg, check.bg, fgHex, bgHex)
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

// TestPaletteContrastForDefaultThemeInteractiveComponents validates the explicit
// color combinations used by compact default-theme controls that have regressed
// in Lighthouse before. These checks are intentionally named after the UI they
// protect so future failures map back to concrete theme elements.
func TestPaletteContrastForDefaultThemeInteractiveComponents(t *testing.T) {
	t.Parallel()

	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in palettes found")
	}

	componentChecks := []struct {
		name      string
		fgOptions []string
		bgOptions []string
		minRatio  float64
		level     string
		desc      string
	}{
		{
			name:      "post copy label",
			fgOptions: []string{"text-primary"},
			bgOptions: []string{"bg-primary"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "post copy summary text",
		},
		{
			name:      "admonition title",
			fgOptions: []string{"text-primary"},
			bgOptions: []string{"admonition-note-bg", "bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "admonition title text on note backgrounds",
		},
		{
			name:      "warning admonition title",
			fgOptions: []string{"text-primary"},
			bgOptions: []string{"admonition-warning-bg", "admonition-warn-bg", "bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "admonition title text on warning backgrounds",
		},
		{
			name:      "feed nav button label",
			fgOptions: []string{"text-primary"},
			bgOptions: []string{"bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "feed navigation button glyphs",
		},
		{
			name:      "card domain link",
			fgOptions: []string{"text-primary"},
			bgOptions: []string{"bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "compact card domain links",
		},
		{
			name:      "homepage updated label",
			fgOptions: []string{"text-secondary"},
			bgOptions: []string{"bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "small home card metadata labels",
		},
		{
			name:      "webmention count label",
			fgOptions: []string{"text-secondary"},
			bgOptions: []string{"bg-surface"},
			minRatio:  4.5,
			level:     "AA",
			desc:      "compact webmention count labels on cards",
		},
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := LoadBuiltin(name)
			if err != nil {
				t.Fatalf("Failed to load palette %s: %v", name, err)
			}

			for _, check := range componentChecks {
				fgName, fgHex := resolveFirst(palette, check.fgOptions...)
				bgName, bgHex := resolveFirst(palette, check.bgOptions...)

				if fgName == "" || bgName == "" {
					t.Errorf("Palette %s: missing color mapping for %s (%v on %v)",
						name, check.name, check.fgOptions, check.bgOptions)
					continue
				}

				requireContrast(t, name, fgName, fgHex, bgName, bgHex, check.minRatio, check.desc)
			}
		})
	}
}
