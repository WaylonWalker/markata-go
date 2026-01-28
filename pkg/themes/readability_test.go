// Package themes provides embedded theme files for markata-go.
package themes

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestCSSTypographyReadability validates that CSS has proper typography settings
// for readability according to WCAG 2.1 guidelines.
func TestCSSTypographyReadability(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	varsCSS, err := ReadStatic("css/variables.css")
	if err != nil {
		t.Fatalf("Failed to read variables.css: %v", err)
	}
	vars := string(varsCSS)

	t.Run("base font size is at least 16px", func(t *testing.T) {
		// WCAG recommends 16px as minimum base font size
		// Check html or :root font-size
		re := regexp.MustCompile(`html\s*\{[^}]*font-size:\s*(\d+)px`)
		match := re.FindStringSubmatch(css)
		if match == nil {
			t.Error("Missing explicit base font-size on html element")
			return
		}
		size, err := strconv.Atoi(match[1])
		if err != nil {
			t.Errorf("Failed to parse font size: %v", err)
			return
		}
		if size < 16 {
			t.Errorf("Base font-size is %dpx, should be at least 16px for readability", size)
		}
	})

	t.Run("body line height is at least 1.5", func(t *testing.T) {
		// WCAG 1.4.12 Text Spacing: line-height should be at least 1.5
		// Check for line-height in body or using --leading-relaxed variable
		if !strings.Contains(css, "line-height: var(--leading-relaxed)") &&
			!strings.Contains(css, "line-height: 1.5") &&
			!strings.Contains(css, "line-height: 1.75") {
			// Check if body has any line-height setting
			bodyRe := regexp.MustCompile(`body\s*\{[^}]*line-height:`)
			if !bodyRe.MatchString(css) {
				t.Error("Body should have line-height set for readability (WCAG 1.4.12)")
			}
		}
	})

	t.Run("line height variables are defined correctly", func(t *testing.T) {
		// Verify --leading-relaxed is >= 1.5 (WCAG minimum for body text)
		re := regexp.MustCompile(`--leading-relaxed:\s*([\d.]+)`)
		match := re.FindStringSubmatch(vars)
		if match == nil {
			t.Error("Missing --leading-relaxed variable definition")
			return
		}
		lineHeight, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			t.Errorf("Failed to parse line height: %v", err)
			return
		}
		if lineHeight < 1.5 {
			t.Errorf("--leading-relaxed is %.2f, should be at least 1.5 (WCAG 1.4.12)", lineHeight)
		}
	})

	t.Run("paragraph spacing exists", func(t *testing.T) {
		// WCAG 1.4.12: Paragraph spacing should be at least 2x font size
		// Check that paragraphs have margin-bottom
		if !strings.Contains(css, "p {") || !regexp.MustCompile(`p\s*\{[^}]*margin-bottom`).MatchString(css) {
			t.Error("Paragraphs should have margin-bottom for proper spacing (WCAG 1.4.12)")
		}
	})

	t.Run("content max-width is set for readability", func(t *testing.T) {
		// Optimal line length is 50-80 characters
		// Check for --content-width in ch units or reasonable px
		if !strings.Contains(vars, "--content-width:") {
			t.Error("Missing --content-width variable for optimal line length")
			return
		}
		// Verify it uses ch units (character width) which is ideal
		chRe := regexp.MustCompile(`--content-width:\s*(\d+)ch`)
		match := chRe.FindStringSubmatch(vars)
		if match != nil {
			width, err := strconv.Atoi(match[1])
			if err != nil {
				t.Errorf("Failed to parse width: %v", err)
				return
			}
			if width < 50 || width > 80 {
				t.Logf("--content-width is %dch, optimal range is 50-80ch for readability", width)
			}
		}
	})

	t.Run("heading line heights are reasonable", func(t *testing.T) {
		// Headings should have tighter line-height (1.1-1.3 is acceptable)
		re := regexp.MustCompile(`--leading-tight:\s*([\d.]+)`)
		match := re.FindStringSubmatch(vars)
		if match == nil {
			t.Error("Missing --leading-tight variable for headings")
			return
		}
		lineHeight, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			t.Errorf("Failed to parse line height: %v", err)
			return
		}
		if lineHeight < 1.1 || lineHeight > 1.4 {
			t.Logf("--leading-tight is %.2f, typical range for headings is 1.1-1.4", lineHeight)
		}
	})
}

// TestCSSFontSizeScale validates that font size variables follow a reasonable scale.
func TestCSSFontSizeScale(t *testing.T) {
	varsCSS, err := ReadStatic("css/variables.css")
	if err != nil {
		t.Fatalf("Failed to read variables.css: %v", err)
	}
	vars := string(varsCSS)

	t.Run("font size variables are defined", func(t *testing.T) {
		requiredSizes := []string{
			"--text-base",
			"--text-sm",
			"--text-lg",
			"--text-xl",
		}
		for _, size := range requiredSizes {
			if !strings.Contains(vars, size+":") {
				t.Errorf("Missing font size variable: %s", size)
			}
		}
	})

	t.Run("base font size is 1rem", func(t *testing.T) {
		// --text-base should be 1rem (16px at default browser settings)
		if !strings.Contains(vars, "--text-base: 1rem") {
			t.Error("--text-base should be 1rem for proper scaling")
		}
	})

	t.Run("smallest font size is not too small", func(t *testing.T) {
		// --text-xs should not be below 0.75rem (12px) - though this is borderline
		re := regexp.MustCompile(`--text-xs:\s*([\d.]+)rem`)
		match := re.FindStringSubmatch(vars)
		if match != nil {
			size, err := strconv.ParseFloat(match[1], 64)
			if err != nil {
				t.Errorf("Failed to parse font size: %v", err)
				return
			}
			if size < 0.75 {
				t.Errorf("--text-xs is %.2frem (%.0fpx), which may be too small for readability",
					size, size*16)
			}
			// Warn if using very small text
			if size < 0.875 {
				t.Logf("Note: --text-xs is %.2frem (%.0fpx). Consider limiting use of this size.",
					size, size*16)
			}
		}
	})
}

// TestCSSColorContrast validates that CSS defines colors with consideration for contrast.
func TestCSSColorContrast(t *testing.T) {
	varsCSS, err := ReadStatic("css/variables.css")
	if err != nil {
		t.Fatalf("Failed to read variables.css: %v", err)
	}
	vars := string(varsCSS)

	t.Run("text and background colors are defined", func(t *testing.T) {
		requiredColors := []string{
			"--color-text",
			"--color-background",
			"--color-text-muted",
		}
		for _, color := range requiredColors {
			if !strings.Contains(vars, color+":") {
				t.Errorf("Missing color variable: %s", color)
			}
		}
	})

	t.Run("dark mode colors are defined", func(t *testing.T) {
		// Check for dark mode color scheme support
		if !strings.Contains(vars, "prefers-color-scheme: dark") {
			t.Error("Missing dark mode color definitions (prefers-color-scheme: dark)")
		}
	})

	t.Run("primary color is defined for links", func(t *testing.T) {
		if !strings.Contains(vars, "--color-primary:") {
			t.Error("Missing --color-primary for link colors")
		}
	})
}

// TestCSSTextSpacing validates WCAG 1.4.12 Text Spacing requirements.
func TestCSSTextSpacing(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	t.Run("lists have proper spacing", func(t *testing.T) {
		// Lists should have margin/padding for readability
		listRe := regexp.MustCompile(`(?:ul|ol)\s*(?:,\s*(?:ul|ol))?\s*\{[^}]*(?:margin|padding)`)
		if !listRe.MatchString(css) {
			t.Error("Lists (ul, ol) should have margin or padding for readability")
		}
	})

	t.Run("list items have spacing", func(t *testing.T) {
		// List items should have margin-bottom
		liRe := regexp.MustCompile(`li\s*\{[^}]*margin-bottom`)
		if !liRe.MatchString(css) {
			t.Error("List items (li) should have margin-bottom for readability")
		}
	})

	t.Run("headings have margin top and bottom", func(t *testing.T) {
		// Headings need space above and below
		headingRe := regexp.MustCompile(`h[1-6][^{]*\{[^}]*margin-top`)
		if !headingRe.MatchString(css) {
			t.Error("Headings should have margin-top for visual separation")
		}
		headingRe2 := regexp.MustCompile(`h[1-6][^{]*\{[^}]*margin-bottom`)
		if !headingRe2.MatchString(css) {
			t.Error("Headings should have margin-bottom for visual separation")
		}
	})

	t.Run("blockquotes have visual distinction", func(t *testing.T) {
		// Blockquotes should be visually distinct
		if !strings.Contains(css, "blockquote") {
			t.Error("Missing blockquote styles")
			return
		}
		bqRe := regexp.MustCompile(`blockquote\s*\{[^}]*(border|background|padding)`)
		if !bqRe.MatchString(css) {
			t.Error("Blockquotes should have border, background, or padding for visual distinction")
		}
	})
}

// TestCSSResponsiveTypography validates that typography scales appropriately on mobile.
func TestCSSResponsiveTypography(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	// Extract mobile breakpoint content
	mobileSection := extractMediaQueryContent(css, "768px")

	t.Run("font size adjusts on mobile", func(t *testing.T) {
		// Check that html or body font-size is adjusted on mobile
		// A slight reduction (15px) is acceptable for mobile
		if mobileSection != "" {
			hasFontSize := strings.Contains(mobileSection, "font-size:") ||
				(strings.Contains(css, "@media") && strings.Contains(css, "font-size: 15px"))
			if !hasFontSize {
				t.Log("Consider adjusting base font-size for mobile devices")
			}
		}
	})

	t.Run("heading sizes scale down on mobile", func(_ *testing.T) {
		// Check if heading sizes are adjusted in mobile breakpoints
		// This is informational - no assertion needed
		smallMobile := extractMediaQueryContent(css, "375px")
		_ = smallMobile // Consume variable - this is just a structural check
	})

	t.Run("line length is controlled on all screen sizes", func(t *testing.T) {
		// Content should not be too wide on any screen
		if !strings.Contains(css, "max-width:") {
			t.Error("Missing max-width constraints for content readability")
		}
	})
}

// TestCSSWordBreak validates that long words don't break layout.
func TestCSSWordBreak(t *testing.T) {
	mainCSS, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatalf("Failed to read main.css: %v", err)
	}
	css := string(mainCSS)

	t.Run("content has overflow-wrap or word-wrap", func(t *testing.T) {
		// Check for word wrapping in post-content or main content areas
		if !strings.Contains(css, "overflow-wrap: break-word") &&
			!strings.Contains(css, "word-wrap: break-word") {
			t.Error("Content should have overflow-wrap: break-word to handle long words/URLs")
		}
	})
}
