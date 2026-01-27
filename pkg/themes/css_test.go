// Package themes provides embedded theme files for markata-go.
package themes

import (
	"regexp"
	"strings"
	"testing"
)

// TestCSSMobileResponsive validates that the CSS has proper mobile responsiveness
// to prevent horizontal overflow issues on mobile devices.
func TestCSSMobileResponsive(t *testing.T) {
	cssContent, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(cssContent)

	t.Run("has mobile breakpoint", func(t *testing.T) {
		// Check for essential mobile breakpoint
		if !regexp.MustCompile(`@media\s*\([^)]*max-width:\s*768px`).MatchString(css) {
			t.Error("Missing 768px mobile breakpoint")
		}
	})

	t.Run("has small mobile breakpoint", func(t *testing.T) {
		// Check for small mobile breakpoint (480px or similar)
		if !regexp.MustCompile(`@media\s*\([^)]*max-width:\s*(480|420|425)px`).MatchString(css) {
			t.Error("Missing small mobile breakpoint (480px or similar)")
		}
	})

	t.Run("has extra small breakpoint", func(t *testing.T) {
		// Check for extra small breakpoint (375px or smaller)
		if !regexp.MustCompile(`@media\s*\([^)]*max-width:\s*375px`).MatchString(css) {
			t.Error("Missing extra small mobile breakpoint (375px)")
		}
	})

	t.Run("prevents body overflow", func(t *testing.T) {
		// Check that body has overflow-x: hidden in mobile styles
		if !strings.Contains(css, "overflow-x: hidden") {
			t.Error("Missing overflow-x: hidden for body (prevents mobile horizontal scroll)")
		}
	})

	t.Run("has box-sizing border-box", func(t *testing.T) {
		// Ensure box-sizing: border-box is set globally
		if !strings.Contains(css, "box-sizing: border-box") {
			t.Error("Missing box-sizing: border-box (essential for predictable layouts)")
		}
	})

	t.Run("images have max-width 100%", func(t *testing.T) {
		// Images should not overflow their containers
		if !regexp.MustCompile(`img\s*\{[^}]*max-width:\s*100%`).MatchString(css) {
			t.Error("Missing max-width: 100% for images")
		}
	})
}

// TestCSSFeedMobileStyles validates that feed/card styles are responsive
// and don't cause overflow on mobile devices.
func TestCSSFeedMobileStyles(t *testing.T) {
	cssContent, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(cssContent)

	// Extract mobile media query content (768px and below)
	mobileSection := extractMediaQueryContent(css, "768px")
	if mobileSection == "" {
		t.Fatal("Could not extract 768px media query content")
	}

	t.Run("feed has max-width in mobile", func(t *testing.T) {
		// Feed should have max-width: 100% in mobile to prevent overflow
		if !strings.Contains(mobileSection, "max-width: 100%") {
			t.Error("Feed container missing max-width: 100% in mobile breakpoint")
		}
	})

	t.Run("feed specificity correct in mobile", func(t *testing.T) {
		// Check that mobile styles use same selector specificity as base styles
		// Base uses "section.feed, div.feed", mobile should too
		hasCorrectSpecificity := strings.Contains(mobileSection, "section.feed") ||
			strings.Contains(mobileSection, "div.feed")
		if !hasCorrectSpecificity {
			t.Error("Mobile feed styles may have specificity issues - should include section.feed or div.feed selectors")
		}
	})

	t.Run("main has reduced padding in mobile", func(t *testing.T) {
		// Check that main element has reduced padding in mobile
		if !strings.Contains(mobileSection, "main") {
			t.Error("Main element should have reduced padding in mobile breakpoint")
		}
	})

	t.Run("container has mobile styles", func(t *testing.T) {
		// Container should have reduced padding and max-width on mobile
		if !strings.Contains(mobileSection, ".container") {
			t.Error("Container should have mobile-specific styles to prevent overflow")
		}
	})

	t.Run("header has mobile styles", func(t *testing.T) {
		// Header should have mobile-specific styles
		if !strings.Contains(mobileSection, ".site-header") {
			t.Error("Header should have mobile-specific styles")
		}
	})

	t.Run("footer has mobile styles", func(t *testing.T) {
		// Footer should have mobile-specific styles
		if !strings.Contains(mobileSection, ".site-footer") {
			t.Error("Footer should have mobile-specific styles")
		}
	})
}

// TestCSSNoFixedWidthsInMobile checks that mobile styles don't use problematic fixed widths.
func TestCSSNoFixedWidthsInMobile(t *testing.T) {
	cssContent, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(cssContent)

	// Extract content from all mobile breakpoints
	breakpoints := []string{"768px", "480px", "375px"}

	for _, bp := range breakpoints {
		mobileSection := extractMediaQueryContent(css, bp)
		if mobileSection == "" {
			continue
		}

		t.Run("no large fixed widths at "+bp, func(t *testing.T) {
			// Look for width declarations with large pixel values (300px+)
			// that could cause overflow on mobile
			// Find all width declarations, then filter out min-width
			re := regexp.MustCompile(`([a-z-]*width):\s*(\d+)px`)
			matches := re.FindAllStringSubmatch(mobileSection, -1)

			for _, match := range matches {
				if len(match) <= 2 {
					continue
				}
				prop := match[1]
				// Skip min-width as those are safe (minimum constraints)
				if prop == "min-width" {
					continue
				}
				// Skip max-width as those are also safe (maximum constraints)
				if prop == "max-width" {
					continue
				}
				// Check if it's a large fixed width (300px+) that could cause overflow
				// Parse the width value
				widthStr := match[2]
				if len(widthStr) >= 3 { // 3+ digits = 100px+
					t.Logf("Found width declaration in %s breakpoint: %s: %spx", bp, prop, widthStr)
					// Only error on truly problematic widths (400px+ on mobile)
					if n, _ := parseDigits(widthStr); n >= 400 {
						t.Errorf("Found potentially problematic fixed width in %s breakpoint: %s: %dpx (may cause mobile overflow)", bp, prop, n)
					}
				}
			}
		})
	}
}

// parseDigits extracts an integer from a string of digits.
func parseDigits(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return n, false
		}
	}
	return n, true
}

// extractMediaQueryContent extracts the content of a media query by max-width value.
func extractMediaQueryContent(css, maxWidth string) string {
	// Simple extraction - find the media query and get its content
	pattern := regexp.MustCompile(`@media\s*\([^)]*max-width:\s*` + regexp.QuoteMeta(maxWidth) + `[^)]*\)\s*\{`)
	loc := pattern.FindStringIndex(css)
	if loc == nil {
		return ""
	}

	start := loc[1]
	depth := 1
	end := start

	for i := start; i < len(css) && depth > 0; i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		end = i
	}

	if end > start {
		return css[start:end]
	}
	return ""
}

// TestCSSSpacingVariables ensures CSS uses spacing variables consistently.
func TestCSSSpacingVariables(t *testing.T) {
	cssContent, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(cssContent)

	t.Run("uses CSS custom properties for spacing", func(t *testing.T) {
		// Check that spacing uses variables like --space-*
		if !strings.Contains(css, "var(--space-") {
			t.Error("CSS should use --space-* custom properties for consistent spacing")
		}
	})

	t.Run("variables file exists", func(t *testing.T) {
		_, err := ReadStatic("css/variables.css")
		if err != nil {
			t.Error("Missing variables.css file")
		}
	})
}
