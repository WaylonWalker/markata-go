// Package themes provides embedded theme files for markata-go.
package themes

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// minInt returns the smaller of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestCSSFocusIndicators validates that interactive elements have visible focus states.
// This is required for WCAG 2.4.7 Focus Visible (Level AA).
func TestCSSFocusIndicators(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	allCSS := css + "\n" + components

	t.Run("links have focus state", func(t *testing.T) {
		// Check for a:focus or a:focus-visible styles
		if !strings.Contains(allCSS, "a:focus") &&
			!strings.Contains(allCSS, ":focus") {
			t.Error("Missing focus styles for links (WCAG 2.4.7)")
		}
	})

	t.Run("focus-visible is used for keyboard navigation", func(t *testing.T) {
		// Modern approach: use :focus-visible for keyboard-only focus
		if !strings.Contains(allCSS, "focus-visible") {
			t.Log("Consider using :focus-visible for better keyboard/mouse focus distinction")
		}
	})

	t.Run("focus has visible outline or ring", func(t *testing.T) {
		// Focus styles should include outline, box-shadow, or border change
		focusRe := regexp.MustCompile(`:focus[^{]*\{[^}]*(outline|box-shadow|border)`)
		if !focusRe.MatchString(allCSS) {
			t.Error("Focus styles should include visible indicator (outline, box-shadow, or border)")
		}
	})

	t.Run("navigation links have focus styles", func(t *testing.T) {
		// Nav links specifically should have focus states
		// Check for specific nav focus patterns or general link focus
		navFocusPatterns := []string{
			".nav-link:focus",
			".site-nav a:focus",
			".nav-link:focus-visible",
			".breadcrumb-link:focus",
		}
		hasNavFocus := false
		for _, pattern := range navFocusPatterns {
			if strings.Contains(allCSS, pattern) {
				hasNavFocus = true
				break
			}
		}
		// General link focus also covers nav links
		hasGeneralFocus := strings.Contains(allCSS, "a:focus") ||
			strings.Contains(allCSS, ":focus {") ||
			strings.Contains(allCSS, ":focus-visible")

		if !hasNavFocus && !hasGeneralFocus {
			t.Error("Navigation links should have visible focus states (WCAG 2.4.7)")
		}
	})

	t.Run("buttons/pagination have focus styles", func(t *testing.T) {
		// Check for focus on button-like elements
		buttonFocusPatterns := []string{
			"button:focus",
			".pagination-page:focus",
			".pagination-prev:focus",
			".pagination-next:focus",
		}
		found := false
		for _, pattern := range buttonFocusPatterns {
			if strings.Contains(allCSS, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Log("Consider adding explicit focus styles for button-like elements")
		}
	})

	t.Run("focus outline is not removed without replacement", func(t *testing.T) {
		// Check for dangerous "outline: none" without replacement
		// This is a WCAG violation if there's no visible alternative
		outlineNoneRe := regexp.MustCompile(`:focus\s*\{[^}]*outline:\s*none`)
		matches := outlineNoneRe.FindAllStringIndex(allCSS, -1)

		for _, match := range matches {
			// Get the context around the match (the full rule)
			start := match[0]
			end := match[1]

			// Find the end of the rule
			ruleEnd := strings.Index(allCSS[start:], "}")
			if ruleEnd == -1 {
				continue
			}
			ruleContent := allCSS[start : start+ruleEnd]

			// Check if there's a replacement in the same rule (box-shadow, border)
			hasReplacement := strings.Contains(ruleContent, "box-shadow") ||
				strings.Contains(ruleContent, "border-color")

			if hasReplacement {
				continue // Has visible replacement in same rule
			}

			// Check if this is part of a focus/focus-visible pattern
			// Look for focus-visible with proper styling after this rule
			afterRule := allCSS[end:]
			lookAhead := afterRule
			if len(lookAhead) > 400 {
				lookAhead = lookAhead[:400]
			}
			// Check for focus-visible with outline, box-shadow, or border
			focusVisibleRe := regexp.MustCompile(`:focus-visible\s*\{[^}]*(outline|box-shadow|border)`)
			if focusVisibleRe.MatchString(lookAhead) {
				continue // Has focus-visible pattern following
			}

			// If we get here, it's a potential violation
			t.Errorf("Found 'outline: none' on :focus without visible replacement around: %s...",
				ruleContent[:minInt(len(ruleContent), 80)])
		}
	})
}

// TestCSSTouchTargets validates that interactive elements meet minimum touch target sizes.
// This is recommended by WCAG 2.5.5 Target Size (Level AAA) and mobile best practices.
func TestCSSTouchTargets(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	allCSS := css + "\n" + components

	t.Run("pagination items have minimum size", func(t *testing.T) {
		// WCAG recommends 44x44px minimum, 36px is acceptable with proper spacing
		// Check for min-width/height on pagination items
		paginationRe := regexp.MustCompile(`\.pagination-page[^{]*\{[^}]*(min-width|height):\s*(\d+)px`)
		match := paginationRe.FindStringSubmatch(allCSS)
		if match != nil {
			size, err := strconv.Atoi(match[2])
			if err != nil {
				t.Errorf("Failed to parse size: %v", err)
				return
			}
			if size < 36 {
				t.Errorf("Pagination touch targets are %dpx, should be at least 36px (ideally 44px)", size)
			}
		} else {
			t.Log("Consider setting explicit min-width/height for pagination items")
		}
	})

	t.Run("navigation links have adequate padding", func(t *testing.T) {
		// Nav links should have padding for touch targets
		navPaddingRe := regexp.MustCompile(`\.nav-link[^{]*\{[^}]*padding`)
		if !navPaddingRe.MatchString(allCSS) {
			t.Error("Navigation links should have padding for adequate touch targets")
		}
	})

	t.Run("buttons have minimum touch area", func(t *testing.T) {
		// Check for button-like elements having minimum dimensions
		// .pagination-prev, .pagination-next should have padding
		if strings.Contains(allCSS, ".pagination-prev") {
			paginationBtnRe := regexp.MustCompile(`\.pagination-(?:prev|next)[^{]*\{[^}]*padding`)
			if !paginationBtnRe.MatchString(allCSS) {
				t.Error("Pagination buttons should have padding for touch targets")
			}
		}
	})

	t.Run("tags have adequate touch size", func(t *testing.T) {
		// .tag elements should have padding
		tagRe := regexp.MustCompile(`\.tag[^{]*\{[^}]*padding`)
		if !tagRe.MatchString(allCSS) {
			t.Log("Consider adding padding to .tag elements for better touch targets")
		}
	})
}

// TestCSSScreenReaderSupport validates that CSS supports screen reader utilities.
func TestCSSScreenReaderSupport(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	allCSS := css + "\n" + components

	t.Run("sr-only class exists", func(t *testing.T) {
		// .sr-only is the standard class for screen-reader-only content
		if !strings.Contains(allCSS, ".sr-only") {
			t.Error("Missing .sr-only class for screen reader content")
		}
	})

	t.Run("sr-only uses correct technique", func(t *testing.T) {
		// sr-only should use position/clip technique, not display:none or visibility:hidden
		srOnlyRe := regexp.MustCompile(`\.sr-only[^{]*\{[^}]*position:\s*absolute`)
		if !srOnlyRe.MatchString(allCSS) {
			t.Error(".sr-only should use position: absolute (not display: none)")
		}
		// Check for clip or clip-path
		srOnlyClipRe := regexp.MustCompile(`\.sr-only[^{]*\{[^}]*clip`)
		if !srOnlyClipRe.MatchString(allCSS) {
			t.Error(".sr-only should use clip technique for hiding")
		}
	})

	t.Run("visually-hidden class exists", func(t *testing.T) {
		// Alternative name for sr-only
		if !strings.Contains(allCSS, ".visually-hidden") && !strings.Contains(allCSS, ".sr-only") {
			t.Error("Missing visually hidden utility class")
		}
	})
}

// TestCSSReducedMotion validates that CSS respects prefers-reduced-motion.
// This is WCAG 2.3.3 Animation from Interactions (Level AAA).
func TestCSSReducedMotion(t *testing.T) {
	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	t.Run("reduced motion media query exists", func(t *testing.T) {
		if !strings.Contains(components, "prefers-reduced-motion") {
			t.Log("Consider adding @media (prefers-reduced-motion: reduce) for users who prefer less motion")
		}
	})

	t.Run("animations respect reduced motion", func(t *testing.T) {
		// If there are animations, check for reduced motion handling
		if strings.Contains(components, "@keyframes") ||
			strings.Contains(components, "animation:") ||
			strings.Contains(components, "transition:") {
			if !strings.Contains(components, "prefers-reduced-motion") {
				t.Log("CSS has animations/transitions - consider adding prefers-reduced-motion support")
			}
		}
	})
}

// TestCSSHighContrastMode validates support for high contrast preferences.
func TestCSSHighContrastMode(t *testing.T) {
	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	t.Run("high contrast mode support", func(t *testing.T) {
		// Check for prefers-contrast media query
		if !strings.Contains(components, "prefers-contrast") {
			t.Log("Consider adding @media (prefers-contrast: more) for high contrast mode support")
		}
	})
}

// TestCSSSkipLink validates that skip link pattern is supported.
func TestCSSSkipLink(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	allCSS := css + "\n" + components

	t.Run("skip link class available", func(t *testing.T) {
		// Check if there's a skip-link or sr-only:focus-within pattern
		hasSkipLink := strings.Contains(allCSS, ".skip-link") ||
			strings.Contains(allCSS, ".skip-to-content")
		hasSrOnly := strings.Contains(allCSS, ".sr-only")

		if !hasSkipLink && !hasSrOnly {
			t.Log("Consider adding skip link styles for keyboard navigation (WCAG 2.4.1)")
		}
	})
}

// TestCSSLinkDistinguishability validates that links are distinguishable.
// WCAG 1.4.1 Use of Color requires links be distinguishable not just by color.
func TestCSSLinkDistinguishability(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	t.Run("links have underline or other indicator on hover", func(t *testing.T) {
		// Links should be distinguishable on hover (underline is common)
		hoverUnderlineRe := regexp.MustCompile(`a:hover[^{]*\{[^}]*text-decoration`)
		if !hoverUnderlineRe.MatchString(css) {
			t.Log("Links should have text-decoration or other visual indicator on hover")
		}
	})

	t.Run("links use color variable", func(t *testing.T) {
		// Links should use color variables for theming support
		linkColorRe := regexp.MustCompile(`a\s*\{[^}]*color:\s*var\(--color-primary`)
		if !linkColorRe.MatchString(css) {
			t.Log("Links should use --color-primary variable for consistent theming")
		}
	})
}

// TestCSSFormAccessibility validates form-related accessibility (if forms exist).
func TestCSSFormAccessibility(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	allCSS := css + "\n" + components

	// Only run if forms/inputs exist
	if !strings.Contains(allCSS, "input") && !strings.Contains(allCSS, "form") {
		t.Skip("No form styles found - skipping form accessibility tests")
	}

	t.Run("input focus is visible", func(t *testing.T) {
		inputFocusRe := regexp.MustCompile(`input[^{]*:focus`)
		if strings.Contains(allCSS, "input") && !inputFocusRe.MatchString(allCSS) {
			t.Log("Consider adding :focus styles for input elements")
		}
	})
}

// TestCSSPrintStyles validates print stylesheet considerations.
func TestCSSPrintStyles(t *testing.T) {
	componentsCSS, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatalf("Failed to read components.css: %v", err)
	}
	components := string(componentsCSS)

	t.Run("print media query exists", func(t *testing.T) {
		if !strings.Contains(components, "@media print") {
			t.Log("Consider adding @media print styles for better print output")
		}
	})
}
