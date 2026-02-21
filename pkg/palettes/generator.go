package palettes

import "fmt"

// HSL represents a color in Hue, Saturation, Lightness space
// H is 0-360, S is 0-1, L is 0-1
type HSL struct {
	H, S, L float64
}

// ToColor converts HSL to Color
func (hsl HSL) ToColor() Color {
	return HSLToRGB(hsl.H, hsl.S, hsl.L)
}

// RGBToHSL converts RGB values to HSL
func (c Color) ToHSL() HSL {
	return RGBToHSL(c)
}

// RGBToHSL converts Color to H, S, L
func RGBToHSL(c Color) HSL {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	maxV := r
	if g > maxV {
		maxV = g
	}
	if b > maxV {
		maxV = b
	}

	minV := r
	if g < minV {
		minV = g
	}
	if b < minV {
		minV = b
	}

	h, s, l := 0.0, 0.0, (maxV+minV)/2.0

	if maxV == minV {
		h = 0
		s = 0 // achromatic
	} else {
		d := maxV - minV
		if l > 0.5 {
			s = d / (2.0 - maxV - minV)
		} else {
			s = d / (maxV + minV)
		}

		switch maxV {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}
		h /= 6
	}

	return HSL{H: h * 360, S: s, L: l}
}

// hue2rgb is a helper for HSLToRGB
func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// HSLToRGB converts H, S, L to Color
func HSLToRGB(h, s, l float64) Color {
	r, g, b := 0.0, 0.0, 0.0

	h /= 360.0

	if s == 0 {
		r, g, b = l, l, l // achromatic
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hue2rgb(p, q, h+1.0/3.0)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3.0)
	}

	return Color{
		R: uint8(r*255.0 + 0.5),
		G: uint8(g*255.0 + 0.5),
		B: uint8(b*255.0 + 0.5),
	}
}

// GenerateTriadicPalette generates a complete color palette from a single seed hex color.
func GenerateTriadicPalette(seedHex string, variant Variant) (*Palette, error) {
	seedColor, err := ParseHexColor(seedHex)
	if err != nil {
		return nil, fmt.Errorf("invalid seed color: %w", err)
	}

	seedHSL := seedColor.ToHSL()

	palette := NewPalette("generated-"+string(variant), variant)
	palette.Source = "generated"

	// Primary is the seed color
	palette.Colors["primary"] = seedColor.Hex()
	palette.Colors["primary-light"] = HSLToRGB(seedHSL.H, seedHSL.S, clamp(seedHSL.L+0.15)).Hex()
	palette.Colors["primary-dark"] = HSLToRGB(seedHSL.H, seedHSL.S, clamp(seedHSL.L-0.15)).Hex()

	// Triadic secondary (+120 degrees)
	secondaryHSL := HSL{H: mathMod(seedHSL.H+120, 360), S: seedHSL.S, L: seedHSL.L}
	palette.Colors["secondary"] = secondaryHSL.ToColor().Hex()

	// Triadic tertiary (+240 degrees)
	tertiaryHSL := HSL{H: mathMod(seedHSL.H+240, 360), S: seedHSL.S, L: seedHSL.L}
	palette.Colors["tertiary"] = tertiaryHSL.ToColor().Hex()

	// Generate backgrounds and text based on variant
	if variant == VariantLight {
		// Very light backgrounds, tinted with seed hue
		palette.Colors["background"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.1), 0.98).Hex()
		palette.Colors["surface"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.15), 0.95).Hex()
		palette.Colors["border"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.2), 0.85).Hex()

		// Dark text
		palette.Colors["text"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.1), 0.1).Hex()
		palette.Colors["text-muted"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.1), 0.35).Hex()
	} else {
		// Very dark backgrounds, tinted with seed hue
		palette.Colors["background"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.15), 0.05).Hex()
		palette.Colors["surface"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.2), 0.1).Hex()
		palette.Colors["border"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.2), 0.2).Hex()

		// Light text
		palette.Colors["text"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.1), 0.95).Hex()
		palette.Colors["text-muted"] = HSLToRGB(seedHSL.H, clamp(seedHSL.S*0.1), 0.65).Hex()
	}

	// Standard status colors
	palette.Colors["success"] = "#10b981"
	palette.Colors["warning"] = "#f59e0b"
	palette.Colors["error"] = "#ef4444"
	palette.Colors["info"] = "#3b82f6"

	// Map semantic colors
	palette.Semantic["bg-primary"] = "background"
	palette.Semantic["bg-surface"] = "surface"
	palette.Semantic["text-primary"] = "text"
	palette.Semantic["text-secondary"] = "text-muted"
	palette.Semantic["text-muted"] = "text-muted"
	palette.Semantic["link"] = "primary"
	palette.Semantic["link-hover"] = "primary-dark"
	palette.Semantic["link-visited"] = "secondary"

	return palette, nil
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func mathMod(a, b float64) float64 {
	m := float64(int(a) % int(b))
	if m < 0 {
		m += b
	}
	return m
}
