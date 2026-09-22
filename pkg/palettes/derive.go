package palettes

import (
	"fmt"
	"sort"
	"strings"
)

// Derived-palette tuning. Backgrounds are compressed into a narrow band near
// the target extreme so surfaces stay subtle; accents are kept in a readable
// mid-band so links and buttons remain legible on the flipped background.
const (
	derivedLightBackgroundTop = 0.985
	derivedDarkBackgroundTop  = 0.11
	derivedBackgroundSpan     = 0.12
	derivedBackgroundSpread   = 0.6
	derivedBackgroundSatScale = 0.7
	derivedLightAccentMin     = 0.24
	derivedLightAccentMax     = 0.46
	derivedDarkAccentMin      = 0.56
	derivedDarkAccentMax      = 0.80
	derivedTextMinContrast    = 4.5
	derivedPrimaryMinContrast = 7.0
)

// derivedBackgroundRoles are the semantic and component roles whose raw colors
// are treated as surfaces when deriving a counterpart palette.
var derivedBackgroundRoles = []string{
	"bg-primary", "bg-secondary", "bg-surface", "bg-elevated",
	"code-bg", "nav-bg", "card-bg", "card-border", "card-shadow", "border",
	"button-secondary-bg",
	"admonition-note-bg", "admonition-tip-bg", "admonition-warn-bg", "admonition-error-bg",
}

// derivedTextRoles are the roles whose raw colors are treated as body text.
var derivedTextRoles = []string{
	"text-primary", "text-secondary", "text-muted",
	"code-text", "code-comment", "nav-text", "button-secondary-text",
}

// derivedPrimaryTextRoles receive a stronger contrast target than other text.
var derivedPrimaryTextRoles = map[string]bool{"text-primary": true, "text-secondary": true}

// derivedContrastBackgrounds are the surfaces every text and accent color must
// remain readable against in the derived palette.
var derivedContrastBackgrounds = []string{"bg-primary", "bg-secondary", "bg-surface", "bg-elevated"}

// CounterpartName returns the conventional name of the opposite-variant
// counterpart for a palette name, e.g. "dracula" -> "dracula-light".
func CounterpartName(name string, variant Variant) string {
	if variant == VariantLight {
		return name + "-light"
	}
	return name + "-dark"
}

// IsDerivedCounterpart reports whether name has a variant suffix and, if so,
// returns the base name and the variant the suffix requests.
func IsDerivedCounterpart(name string) (base string, variant Variant, ok bool) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "-light"):
		return name[:len(name)-len("-light")], VariantLight, true
	case strings.HasSuffix(lower, "-dark"):
		return name[:len(name)-len("-dark")], VariantDark, true
	}
	return "", "", false
}

// DeriveCounterpart builds the opposite-variant palette for p.
//
// Every palette is guaranteed a light and a dark mode: when a family ships only
// one variant, the other is derived by flipping lightness while preserving hue.
// Raw colors are classified by how the palette's semantic and component layers
// use them. Surfaces are compressed into a narrow band near white (or near
// black), text is inverted and pushed until it meets WCAG AA against every
// derived surface, and accents are inverted into a mid-lightness band and then
// adjusted so links stay readable. Semantic and component references are copied
// unchanged so the derived palette keeps the original's structure.
func DeriveCounterpart(p *Palette) *Palette {
	if p == nil {
		return nil
	}
	target := counterpartVariant(p.Variant)
	derived := newDerivedPalette(p, target)
	backgrounds, texts, primaryText := classifyDerivedColors(p)

	deriveBackgroundColors(p, derived, target, backgrounds)
	contrastBgs := derivedBackgroundColorsForContrast(p, derived)
	deriveForegroundColors(p, derived, target, backgrounds, texts, primaryText, contrastBgs)

	return derived
}

func counterpartVariant(variant Variant) Variant {
	if variant == VariantLight {
		return VariantDark
	}
	return VariantLight
}

func newDerivedPalette(p *Palette, target Variant) *Palette {
	derived := NewPalette(counterpartDisplayName(p.Name, target), target)
	derived.Author = p.Author
	derived.License = p.License
	derived.Homepage = p.Homepage
	derived.Description = fmt.Sprintf("%s counterpart derived from %s", target, p.Name)
	derived.Source = p.Source
	derived.SourcePath = p.SourcePath
	for k, v := range p.Semantic {
		derived.Semantic[k] = v
	}
	for k, v := range p.Components {
		derived.Components[k] = v
	}
	return derived
}

func classifyDerivedColors(p *Palette) (backgrounds, texts, primaryText map[string]bool) {
	backgrounds = make(map[string]bool)
	for _, role := range derivedBackgroundRoles {
		if raw := p.rawNameFor(role); raw != "" {
			backgrounds[raw] = true
		}
	}
	texts = make(map[string]bool)
	primaryText = make(map[string]bool)
	for _, role := range derivedTextRoles {
		raw := p.rawNameFor(role)
		if raw == "" || backgrounds[raw] {
			continue
		}
		texts[raw] = true
		if derivedPrimaryTextRoles[role] {
			primaryText[raw] = true
		}
	}
	return backgrounds, texts, primaryText
}

func deriveBackgroundColors(p, derived *Palette, target Variant, backgrounds map[string]bool) {
	// Backgrounds first: text and accents are adjusted against them.
	bgNames := make([]string, 0, len(backgrounds))
	for name := range backgrounds {
		bgNames = append(bgNames, name)
	}
	sort.Strings(bgNames)
	lMin, lMax := 1.0, 0.0
	for _, name := range bgNames {
		l := p.rawHSL(name).L
		if l < lMin {
			lMin = l
		}
		if l > lMax {
			lMax = l
		}
	}
	spread := derivedBackgroundSpread
	if lMax-lMin > 0 && (lMax-lMin)*spread > derivedBackgroundSpan {
		spread = derivedBackgroundSpan / (lMax - lMin)
	}
	for _, name := range bgNames {
		hsl := p.rawHSL(name)
		if target == VariantLight {
			hsl.L = derivedLightBackgroundTop - (hsl.L-lMin)*spread
		} else {
			hsl.L = derivedDarkBackgroundTop + (lMax-hsl.L)*spread
		}
		hsl.S *= derivedBackgroundSatScale
		derived.Colors[name] = hsl.ToColor().Hex()
	}
}

func derivedBackgroundColorsForContrast(p, derived *Palette) []Color {
	contrastBgs := make([]Color, 0, len(derivedContrastBackgrounds))
	for _, role := range derivedContrastBackgrounds {
		raw := p.rawNameFor(role)
		if hex, ok := derived.Colors[raw]; ok {
			if c, err := ParseHexColor(hex); err == nil {
				contrastBgs = append(contrastBgs, c)
			}
		}
	}
	return contrastBgs
}

func deriveForegroundColors(p, derived *Palette, target Variant, backgrounds, texts, primaryText map[string]bool, contrastBgs []Color) {
	rawNames := make([]string, 0, len(p.Colors))
	for name := range p.Colors {
		rawNames = append(rawNames, name)
	}
	sort.Strings(rawNames)
	for _, name := range rawNames {
		if backgrounds[name] {
			continue
		}
		hsl := p.rawHSL(name)
		hsl.L = 1 - hsl.L
		minContrast := derivedTextMinContrast
		switch {
		case texts[name]:
			if primaryText[name] {
				minContrast = derivedPrimaryMinContrast
			}
		case target == VariantLight:
			hsl.L = clamp01Range(hsl.L, derivedLightAccentMin, derivedLightAccentMax)
		default:
			hsl.L = clamp01Range(hsl.L, derivedDarkAccentMin, derivedDarkAccentMax)
		}
		derived.Colors[name] = adjustAgainstAll(hsl.ToColor(), contrastBgs, minContrast).Hex()
	}
}

// counterpartDisplayName appends the variant word to a display name, replacing
// an existing " Light"/" Dark" suffix when present.
func counterpartDisplayName(name string, target Variant) string {
	for _, suffix := range []string{" Light", " Dark", "-light", "-dark"} {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}
	if strings.ToLower(name) == name {
		return CounterpartName(name, target)
	}
	if target == VariantLight {
		return name + " Light"
	}
	return name + " Dark"
}

// rawNameFor follows semantic and component references until it reaches a raw
// color name. Returns "" if the role is undefined or resolves to a literal hex.
func (p *Palette) rawNameFor(role string) string {
	seen := map[string]bool{}
	name := role
	for !seen[name] {
		seen[name] = true
		if _, ok := p.Colors[name]; ok {
			return name
		}
		if ref, ok := p.Semantic[name]; ok {
			name = ref
			continue
		}
		if ref, ok := p.Components[name]; ok {
			name = ref
			continue
		}
		return ""
	}
	return ""
}

func (p *Palette) rawHSL(name string) HSL {
	c, err := ParseHexColor(p.Colors[name])
	if err != nil {
		return HSL{}
	}
	return c.ToHSL()
}

func clamp01Range(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// adjustAgainstAll nudges c until it meets minRatio against every background,
// moving away from the mean background luminance.
func adjustAgainstAll(c Color, backgrounds []Color, minRatio float64) Color {
	if len(backgrounds) == 0 {
		return c
	}
	worst := backgrounds[0]
	for _, bg := range backgrounds[1:] {
		if ContrastRatio(c, bg) < ContrastRatio(c, worst) {
			worst = bg
		}
	}
	adjusted, _ := c.AdjustForContrast(worst, minRatio)
	return adjusted
}
