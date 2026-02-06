package plugins

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/aesthetic"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// AestheticCSSPlugin generates aesthetic (non-color) CSS tokens from TOML presets.
//
// It writes css/aesthetic.css and a hashed variant for cache busting. The output
// contains:
// - :root defaults (from the configured aesthetic)
// - [data-aesthetic="..."] overrides for built-in aesthetics (for runtime switching)
type AestheticCSSPlugin struct {
	css  string
	hash string
}

const shadowNone = "none"

func NewAestheticCSSPlugin() *AestheticCSSPlugin {
	return &AestheticCSSPlugin{}
}

func (p *AestheticCSSPlugin) Name() string {
	return "aesthetic_css"
}

func (p *AestheticCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	selected, overrides := getAestheticConfig(config.Extra)

	css, err := generateAestheticCSS(selected, overrides)
	if err != nil {
		return err
	}

	p.css = css

	h := sha256.Sum256([]byte(css))
	p.hash = fmt.Sprintf("%x", h[:4])

	assetHashes := map[string]string{
		"css/aesthetic.css": p.hash,
	}
	templates.SetAssetHashes(assetHashes)
	m.SetAssetHash("css/aesthetic.css", p.hash)

	return nil
}

func (p *AestheticCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	css := p.css
	if css == "" {
		// Fallback for tests that call Write() without Configure().
		selected, overrides := getAestheticConfig(config.Extra)
		var err error
		css, err = generateAestheticCSS(selected, overrides)
		if err != nil {
			return err
		}
	}

	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}

	cssPath := filepath.Join(cssDir, "aesthetic.css")
	//nolint:gosec // G306: aesthetic.css is a public CSS file, 0644 is appropriate
	if err := os.WriteFile(cssPath, []byte(css), 0o644); err != nil {
		return fmt.Errorf("writing aesthetic CSS: %w", err)
	}

	if p.hash == "" {
		h := sha256.Sum256([]byte(css))
		p.hash = fmt.Sprintf("%x", h[:4])
	}
	hashedFilename := fmt.Sprintf("aesthetic.%s.css", p.hash)
	hashedPath := filepath.Join(cssDir, hashedFilename)
	//nolint:gosec // G306: aesthetic CSS is a public CSS file, 0644 is appropriate
	if err := os.WriteFile(hashedPath, []byte(css), 0o644); err != nil {
		return fmt.Errorf("writing hashed aesthetic CSS: %w", err)
	}

	return nil
}

func (p *AestheticCSSPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityDefault
	}
	if stage == lifecycle.StageConfigure {
		return lifecycle.PriorityDefault
	}
	return lifecycle.PriorityDefault
}

type aestheticOverrides struct {
	borderRadius string
	borderWidth  string
	borderStyle  string
	spacingScale float64
	hasScale     bool
	shadowSize   string
	shadowMul    float64
	hasShadowMul bool
}

func getAestheticConfig(extra map[string]interface{}) (string, aestheticOverrides) {
	selected := "balanced"
	if extra != nil {
		if v, ok := extra["aesthetic"].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				selected = v
			}
		}
	}

	var overrides aestheticOverrides
	if extra == nil {
		return selected, overrides
	}

	raw, ok := extra["aesthetic_overrides"].(map[string]interface{})
	if !ok {
		return selected, overrides
	}

	if s, ok := raw["border_radius"].(string); ok {
		overrides.borderRadius = strings.TrimSpace(s)
	}
	if s, ok := raw["border_width"].(string); ok {
		overrides.borderWidth = strings.TrimSpace(s)
	}
	if s, ok := raw["border_style"].(string); ok {
		overrides.borderStyle = strings.TrimSpace(s)
	}
	if s, ok := raw["shadow_size"].(string); ok {
		overrides.shadowSize = strings.TrimSpace(strings.ToLower(s))
	}
	if f, ok := raw["shadow_intensity"].(float64); ok {
		overrides.shadowMul = f
		overrides.hasShadowMul = true
	}
	if i, ok := raw["shadow_intensity"].(int64); ok {
		overrides.shadowMul = float64(i)
		overrides.hasShadowMul = true
	}
	if i, ok := raw["shadow_intensity"].(int); ok {
		overrides.shadowMul = float64(i)
		overrides.hasShadowMul = true
	}
	if f, ok := raw["spacing_scale"].(float64); ok {
		overrides.spacingScale = f
		overrides.hasScale = true
	}
	// TOML ints may come through as int64
	if i, ok := raw["spacing_scale"].(int64); ok {
		overrides.spacingScale = float64(i)
		overrides.hasScale = true
	}
	if i, ok := raw["spacing_scale"].(int); ok {
		overrides.spacingScale = float64(i)
		overrides.hasScale = true
	}

	return selected, overrides
}

func generateAestheticCSS(selected string, overrides aestheticOverrides) (string, error) {
	// Load the selected aesthetic (built-in, user, or project).
	rootAesthetic, err := aesthetic.Load(selected)
	if err != nil {
		log.Printf("[aesthetic_css] Failed to load aesthetic %q: %v (falling back to balanced)", selected, err)
		rootAesthetic, err = aesthetic.Load("balanced")
		if err != nil {
			return "", fmt.Errorf("loading fallback aesthetic: %w", err)
		}
		selected = "balanced"
	}

	applyAestheticOverrides(rootAesthetic, overrides)

	names := aesthetic.BuiltinNames()
	if len(names) == 0 {
		names = []string{"balanced"}
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("/* Aesthetic tokens - generated by markata-go */\n")
	sb.WriteString("/* :root is the configured default; data-aesthetic enables runtime switching */\n\n")

	writeAestheticBlock(&sb, ":root", selected, rootAesthetic)

	for _, name := range names {
		a, err := aesthetic.LoadBuiltin(name)
		if err != nil {
			continue
		}
		writeAestheticBlock(&sb, fmt.Sprintf("[data-aesthetic=%q]", name), name, a)
	}

	return sb.String(), nil
}

func applyAestheticOverrides(a *aesthetic.Aesthetic, o aestheticOverrides) {
	if a == nil {
		return
	}
	// Ensure maps are initialized
	if a.Tokens.Radius == nil {
		a.Tokens.Radius = make(map[string]string)
	}
	if a.Tokens.Border == nil {
		a.Tokens.Border = make(map[string]string)
	}
	if a.Tokens.Shadow == nil {
		a.Tokens.Shadow = make(map[string]string)
	}
	if a.Tokens.Typography == nil {
		a.Tokens.Typography = make(map[string]string)
	}
	if a.Tokens.Spacing == nil {
		a.Tokens.Spacing = &aesthetic.SpacingTokens{Scale: 1.0}
	}

	if o.borderRadius != "" {
		// Treat as a global rounding override.
		a.Tokens.Radius["sm"] = o.borderRadius
		a.Tokens.Radius["md"] = o.borderRadius
		a.Tokens.Radius["lg"] = o.borderRadius
		a.Tokens.Radius["xl"] = o.borderRadius
	}
	if o.borderWidth != "" {
		a.Tokens.Border["width_thin"] = o.borderWidth
		a.Tokens.Border["width_normal"] = o.borderWidth
		a.Tokens.Border["width_thick"] = o.borderWidth
	}
	if o.borderStyle != "" {
		a.Tokens.Border["style"] = o.borderStyle
	}
	if o.hasScale {
		// Clamp to a reasonable range.
		s := o.spacingScale
		if s < 0.5 {
			s = 0.5
		}
		if s > 2.0 {
			s = 2.0
		}
		a.Tokens.Spacing.Scale = s
	}

	if o.shadowSize != "" {
		setShadowSize(a, o.shadowSize)
	}
	if o.hasShadowMul {
		mul := o.shadowMul
		if mul < 0 {
			mul = 0
		}
		if mul > 3 {
			mul = 3
		}
		for k, v := range a.Tokens.Shadow {
			a.Tokens.Shadow[k] = adjustRGBAAlpha(v, mul)
		}
	}
}

func setShadowSize(a *aesthetic.Aesthetic, size string) {
	if a == nil || a.Tokens.Shadow == nil {
		return
	}

	size = strings.TrimSpace(strings.ToLower(size))
	if size == shadowNone {
		a.Tokens.Shadow["sm"] = shadowNone
		a.Tokens.Shadow["md"] = shadowNone
		a.Tokens.Shadow["lg"] = shadowNone
		a.Tokens.Shadow["xl"] = shadowNone
		return
	}

	val := strings.TrimSpace(a.Tokens.Shadow[size])
	if val == "" {
		return
	}

	// Apply globally by normalizing all sizes to the selected preset.
	a.Tokens.Shadow["sm"] = val
	a.Tokens.Shadow["md"] = val
	a.Tokens.Shadow["lg"] = val
	a.Tokens.Shadow["xl"] = val
}

func adjustRGBAAlpha(s string, mul float64) string {
	if s == "" || mul == 1 {
		return s
	}

	// Only handle rgba(...) alpha multipliers; keep the rest as-is.
	// Example: rgba(0,0,0,0.08) -> rgba(0,0,0,0.12) when mul=1.5
	re := regexp.MustCompile(`rgba\((\s*\d+\s*),(\s*\d+\s*),(\s*\d+\s*),(\s*[0-9]*\.?[0-9]+\s*)\)`) //nolint:lll // keep regex in one line for readability
	return re.ReplaceAllStringFunc(s, func(m string) string {
		parts := re.FindStringSubmatch(m)
		if len(parts) != 5 {
			return m
		}
		a, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
		if err != nil {
			return m
		}
		a *= mul
		if a < 0 {
			a = 0
		}
		if a > 1 {
			a = 1
		}
		alpha := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", a), "0"), ".")
		if alpha == "" {
			alpha = "0"
		}
		return fmt.Sprintf("rgba(%s,%s,%s,%s)", strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), strings.TrimSpace(parts[3]), alpha)
	})
}

func writeAestheticBlock(sb *strings.Builder, selector, name string, a *aesthetic.Aesthetic) {
	if a == nil {
		return
	}

	fmt.Fprintf(sb, "/* Aesthetic: %s */\n", name)
	sb.WriteString(selector)
	sb.WriteString(" {\n")

	// Radius tokens
	writeVarMap(sb, "radius", a.Tokens.Radius)

	// Border tokens
	writeVarMap(sb, "border", a.Tokens.Border)

	// Shadow tokens
	writeVarMap(sb, "shadow", a.Tokens.Shadow)

	// Typography tokens (no prefix)
	writeVarMap(sb, "", a.Tokens.Typography)

	// Derived compatibility + app-level tokens
	writeDerivedTokens(sb, a)

	sb.WriteString("}\n\n")
}

func writeVarMap(sb *strings.Builder, prefix string, m map[string]string) {
	if len(m) == 0 {
		return
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := strings.TrimSpace(m[k])
		if v == "" {
			continue
		}
		name := strings.ReplaceAll(k, "_", "-")
		if prefix != "" {
			name = prefix + "-" + name
		}
		fmt.Fprintf(sb, "  --%s: %s;\n", name, v)
	}
}

func writeDerivedTokens(sb *strings.Builder, a *aesthetic.Aesthetic) {
	// Radius aliases for existing theme CSS.
	md := strings.TrimSpace(a.Tokens.Radius["md"])
	lg := strings.TrimSpace(a.Tokens.Radius["lg"])
	if md != "" {
		fmt.Fprintf(sb, "  --radius: %s;\n", md)
	}
	if lg != "" {
		fmt.Fprintf(sb, "  --radius-lg: %s;\n", lg)
	}

	// Surface elevation hook used by background decorations.
	sb.WriteString("  --article-shadow: var(--shadow-md);\n")

	// Leading scale: derive the actual line-height tokens used by CSS.
	leadingScale := 1.0
	if raw := strings.TrimSpace(a.Tokens.Typography["leading_scale"]); raw != "" {
		if f, err := strconv.ParseFloat(raw, 64); err == nil && f > 0 {
			leadingScale = f
		}
	}
	fmt.Fprintf(sb, "  --leading-scale: %.3g;\n", leadingScale)
	fmt.Fprintf(sb, "  --leading-tight: %.3g;\n", 1.25*leadingScale)
	fmt.Fprintf(sb, "  --leading-normal: %.3g;\n", 1.5*leadingScale)
	fmt.Fprintf(sb, "  --leading-relaxed: %.3g;\n", 1.75*leadingScale)

	// Spacing: write a precomputed scale so we don't rely on calc() multiplication.
	scale := 1.0
	if a.Tokens.Spacing != nil && a.Tokens.Spacing.Scale > 0 {
		scale = a.Tokens.Spacing.Scale
	}
	fmt.Fprintf(sb, "  --spacing-scale: %.3g;\n", scale)

	base := map[string]float64{
		"1":  0.25,
		"2":  0.5,
		"3":  0.75,
		"4":  1.0,
		"5":  1.25,
		"6":  1.5,
		"8":  2.0,
		"12": 3.0,
		"16": 4.0,
	}

	ordered := []string{"1", "2", "3", "4", "5", "6", "8", "12", "16"}
	for _, k := range ordered {
		v := base[k] * scale
		fmt.Fprintf(sb, "  --space-%s: %s;\n", k, formatRem(v))
	}

	// Typography: allow aesthetics to steer font choice via --font-primary.
	if fp := strings.TrimSpace(a.Tokens.Typography["font_primary"]); fp != "" {
		fmt.Fprintf(sb, "  --font-body: %s;\n", fp)
		fmt.Fprintf(sb, "  --font-heading: %s;\n", fp)
	}
}

func formatRem(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		s = "0"
	}
	return s + "rem"
}

var (
	_ lifecycle.Plugin          = (*AestheticCSSPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*AestheticCSSPlugin)(nil)
	_ lifecycle.WritePlugin     = (*AestheticCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*AestheticCSSPlugin)(nil)
)
