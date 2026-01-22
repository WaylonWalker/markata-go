package palettes

import (
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Color represents an RGB color with methods for contrast calculation.
type Color struct {
	R, G, B uint8
}

// hexColorRegex matches 3, 4, 6, or 8 character hex colors.
var hexColorRegex = regexp.MustCompile(`^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// ParseHexColor parses a hex color string into a Color.
// Supports formats: #RGB, #RGBA, #RRGGBB, #RRGGBBAA (with or without #).
func ParseHexColor(hex string) (Color, error) {
	hex = strings.TrimPrefix(hex, "#")

	if !hexColorRegex.MatchString("#" + hex) {
		return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
	}

	var r, g, b uint8

	switch len(hex) {
	case 3, 4: // RGB or RGBA (short form)
		r64, err := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		g64, err := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		b64, err := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		r, g, b = uint8(r64), uint8(g64), uint8(b64)
	case 6, 8: // RRGGBB or RRGGBBAA
		r64, err := strconv.ParseUint(hex[0:2], 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		g64, err := strconv.ParseUint(hex[2:4], 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		b64, err := strconv.ParseUint(hex[4:6], 16, 8)
		if err != nil {
			return Color{}, fmt.Errorf("%w: %s", ErrInvalidHexColor, hex)
		}
		r, g, b = uint8(r64), uint8(g64), uint8(b64)
	}

	return Color{R: r, G: g, B: b}, nil
}

// Hex returns the color as a hex string with # prefix.
func (c Color) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// RGBA implements the color.Color interface.
func (c Color) RGBA() (r, g, b, a uint32) {
	return uint32(c.R) * 257, uint32(c.G) * 257, uint32(c.B) * 257, 65535
}

// RelativeLuminance calculates the relative luminance of the color.
// Based on WCAG 2.1 definition using sRGB color space.
// Returns a value between 0 (black) and 1 (white).
func (c Color) RelativeLuminance() float64 {
	// Convert to 0-1 range
	rLinear := linearize(float64(c.R) / 255.0)
	gLinear := linearize(float64(c.G) / 255.0)
	bLinear := linearize(float64(c.B) / 255.0)

	// ITU-R BT.709 coefficients
	return 0.2126*rLinear + 0.7152*gLinear + 0.0722*bLinear
}

// linearize converts sRGB gamma-corrected value to linear RGB.
func linearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// ContrastRatio calculates the WCAG 2.1 contrast ratio between two colors.
// Returns a value between 1:1 (same color) and 21:1 (black/white).
func ContrastRatio(fg, bg Color) float64 {
	l1 := fg.RelativeLuminance()
	l2 := bg.RelativeLuminance()

	// Ensure l1 is the lighter color
	if l1 < l2 {
		l1, l2 = l2, l1
	}

	return (l1 + 0.05) / (l2 + 0.05)
}

// ContrastRatioFromHex calculates contrast ratio from hex color strings.
func ContrastRatioFromHex(fgHex, bgHex string) (float64, error) {
	fg, err := ParseHexColor(fgHex)
	if err != nil {
		return 0, fmt.Errorf("invalid foreground color: %w", err)
	}

	bg, err := ParseHexColor(bgHex)
	if err != nil {
		return 0, fmt.Errorf("invalid background color: %w", err)
	}

	return ContrastRatio(fg, bg), nil
}

// WCAGLevel represents WCAG compliance levels.
type WCAGLevel string

const (
	WCAGLevelA   WCAGLevel = "A"
	WCAGLevelAA  WCAGLevel = "AA"
	WCAGLevelAAA WCAGLevel = "AAA"
)

// ContrastRequirement defines minimum contrast ratios for different contexts.
type ContrastRequirement struct {
	NormalText float64 // Minimum for normal text
	LargeText  float64 // Minimum for large text (18pt+ or 14pt bold)
	UI         float64 // Minimum for UI components
}

// WCAGRequirements defines contrast requirements for each WCAG level.
var WCAGRequirements = map[WCAGLevel]ContrastRequirement{
	WCAGLevelA:   {NormalText: 3.0, LargeText: 3.0, UI: 3.0},
	WCAGLevelAA:  {NormalText: 4.5, LargeText: 3.0, UI: 3.0},
	WCAGLevelAAA: {NormalText: 7.0, LargeText: 4.5, UI: 4.5},
}

// MeetsWCAG checks if a contrast ratio meets the specified WCAG level.
func MeetsWCAG(ratio float64, level WCAGLevel, isLargeText bool) bool {
	req, ok := WCAGRequirements[level]
	if !ok {
		return false
	}

	if isLargeText {
		return ratio >= req.LargeText
	}
	return ratio >= req.NormalText
}

// MeetsWCAGUI checks if a contrast ratio meets WCAG for UI components.
func MeetsWCAGUI(ratio float64, level WCAGLevel) bool {
	req, ok := WCAGRequirements[level]
	if !ok {
		return false
	}
	return ratio >= req.UI
}

// PassedLevels returns all WCAG levels that a contrast ratio passes.
func PassedLevels(ratio float64, isLargeText bool) []WCAGLevel {
	var levels []WCAGLevel
	for _, level := range []WCAGLevel{WCAGLevelA, WCAGLevelAA, WCAGLevelAAA} {
		if MeetsWCAG(ratio, level, isLargeText) {
			levels = append(levels, level)
		}
	}
	return levels
}

// ColorFromStdlib converts a standard library color.Color to our Color type.
func ColorFromStdlib(c color.Color) Color {
	r, g, b, _ := c.RGBA()
	// Right-shifting by 8 converts from 16-bit (0-65535) to 8-bit (0-255).
	// The result is always in uint8 range, so the conversion is safe.
	return Color{
		R: uint8(r >> 8), //nolint:gosec // r>>8 is always <= 255
		G: uint8(g >> 8), //nolint:gosec // g>>8 is always <= 255
		B: uint8(b >> 8), //nolint:gosec // b>>8 is always <= 255
	}
}

// Lighten returns a lighter version of the color.
// amount should be between 0 (no change) and 1 (white).
func (c Color) Lighten(amount float64) Color {
	return Color{
		R: uint8(float64(c.R) + (255-float64(c.R))*amount),
		G: uint8(float64(c.G) + (255-float64(c.G))*amount),
		B: uint8(float64(c.B) + (255-float64(c.B))*amount),
	}
}

// Darken returns a darker version of the color.
// amount should be between 0 (no change) and 1 (black).
func (c Color) Darken(amount float64) Color {
	return Color{
		R: uint8(float64(c.R) * (1 - amount)),
		G: uint8(float64(c.G) * (1 - amount)),
		B: uint8(float64(c.B) * (1 - amount)),
	}
}

// AdjustForContrast adjusts the color to meet a minimum contrast ratio against a background.
// Returns the adjusted color and whether adjustment was successful.
func (c Color) AdjustForContrast(bg Color, minRatio float64) (Color, bool) {
	// Check if already meets requirement
	if ContrastRatio(c, bg) >= minRatio {
		return c, true
	}

	bgLum := bg.RelativeLuminance()

	// Try lightening or darkening based on background luminance
	// If background is dark, try lightening; if light, try darkening
	if bgLum < 0.5 {
		// Dark background - try lightening
		for i := 0.0; i <= 1.0; i += 0.01 {
			adjusted := c.Lighten(i)
			if ContrastRatio(adjusted, bg) >= minRatio {
				return adjusted, true
			}
		}
	} else {
		// Light background - try darkening
		for i := 0.0; i <= 1.0; i += 0.01 {
			adjusted := c.Darken(i)
			if ContrastRatio(adjusted, bg) >= minRatio {
				return adjusted, true
			}
		}
	}

	return c, false
}
