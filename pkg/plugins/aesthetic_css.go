package plugins

import (
	"crypto/sha256"
	"encoding/json"
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
	"github.com/WaylonWalker/markata-go/pkg/models"
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
	effects      map[string]string
}

func getAestheticConfig(extra map[string]interface{}) (string, aestheticOverrides) {
	selected := getSelectedAesthetic(extra)
	overrides := getAestheticOverrides(extra)
	return selected, overrides
}

func getSelectedAesthetic(extra map[string]interface{}) string {
	selected := "balanced"
	if extra == nil {
		return selected
	}

	if cfg, ok := extra["models_config"].(*models.Config); ok {
		if v := strings.TrimSpace(cfg.Aesthetic); v != "" {
			selected = v
		}
	}

	if v, ok := extra["aesthetic"].(string); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			selected = v
		}
	}

	return selected
}

func getAestheticOverrides(extra map[string]interface{}) aestheticOverrides {
	var overrides aestheticOverrides
	if extra == nil {
		return overrides
	}

	raw := getRawAestheticOverrides(extra)
	if raw == nil {
		return overrides
	}

	overrides.effects = parseStringMap(raw["effects"])
	overrides.borderRadius = parseString(raw["border_radius"])
	overrides.borderWidth = parseString(raw["border_width"])
	overrides.borderStyle = parseString(raw["border_style"])
	overrides.shadowSize = strings.ToLower(parseString(raw["shadow_size"]))
	overrides.shadowMul, overrides.hasShadowMul = parseFloat(raw["shadow_intensity"])
	overrides.spacingScale, overrides.hasScale = parseFloat(raw["spacing_scale"])

	return overrides
}

func getRawAestheticOverrides(extra map[string]interface{}) map[string]interface{} {
	// Prefer typed config if available.
	if cfg, ok := extra["models_config"].(*models.Config); ok {
		if len(cfg.AestheticOverrides) > 0 {
			result := make(map[string]interface{}, len(cfg.AestheticOverrides))
			for k, v := range cfg.AestheticOverrides {
				result[k] = v
			}
			return result
		}
	}

	raw, ok := extra["aesthetic_overrides"].(map[string]interface{})
	if !ok {
		return nil
	}
	return raw
}

func parseString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func parseFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

func parseStringMap(v interface{}) map[string]string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, vv := range m {
		result[k] = fmt.Sprintf("%v", vv)
	}
	return result
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

	manifestJSON, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("marshaling aesthetic manifest: %w", err)
	}
	// Escape single quotes in the JSON for CSS single-quoted string.
	escapedManifest := strings.ReplaceAll(string(manifestJSON), "'", "\\'")

	var sb strings.Builder
	sb.WriteString("/* Aesthetic tokens - generated by markata-go */\n")
	sb.WriteString("/* :root is the configured default; data-aesthetic enables runtime switching */\n\n")

	writeAestheticBlock(&sb, ":root", selected, rootAesthetic, selected, escapedManifest)

	for _, name := range names {
		a, err := aesthetic.LoadBuiltin(name)
		if err != nil {
			continue
		}
		writeAestheticBlock(&sb, fmt.Sprintf("[data-aesthetic=%q]", name), name, a, "", "")
	}

	return sb.String(), nil
}

func applyAestheticOverrides(a *aesthetic.Aesthetic, o aestheticOverrides) {
	if a == nil {
		return
	}

	ensureAestheticTokenMaps(a)
	applyGlobalRadiusOverride(a, o.borderRadius)
	applyBorderOverrides(a, o.borderWidth, o.borderStyle)
	applySpacingScaleOverride(a, o.spacingScale, o.hasScale)
	applyShadowOverrides(a, o.shadowSize, o.shadowMul, o.hasShadowMul)
	applyEffectsOverrides(a, o.effects)
}

func ensureAestheticTokenMaps(a *aesthetic.Aesthetic) {
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
	if a.Tokens.Effects == nil {
		a.Tokens.Effects = make(map[string]string)
	}
	if a.Tokens.Spacing == nil {
		a.Tokens.Spacing = &aesthetic.SpacingTokens{Scale: 1.0}
	}
}

func applyGlobalRadiusOverride(a *aesthetic.Aesthetic, borderRadius string) {
	if borderRadius == "" {
		return
	}

	// Treat as a global rounding override.
	a.Tokens.Radius["sm"] = borderRadius
	a.Tokens.Radius["md"] = borderRadius
	a.Tokens.Radius["lg"] = borderRadius
	a.Tokens.Radius["xl"] = borderRadius
}

func applyBorderOverrides(a *aesthetic.Aesthetic, borderWidth, borderStyle string) {
	if borderWidth != "" {
		a.Tokens.Border["width_thin"] = borderWidth
		a.Tokens.Border["width_normal"] = borderWidth
		a.Tokens.Border["width_thick"] = borderWidth
	}
	if borderStyle != "" {
		a.Tokens.Border["style"] = borderStyle
	}
}

func applySpacingScaleOverride(a *aesthetic.Aesthetic, spacingScale float64, hasScale bool) {
	if !hasScale {
		return
	}

	// Clamp to a reasonable range.
	s := spacingScale
	if s < 0.5 {
		s = 0.5
	}
	if s > 2.0 {
		s = 2.0
	}
	a.Tokens.Spacing.Scale = s
}

func applyShadowOverrides(a *aesthetic.Aesthetic, shadowSize string, shadowMul float64, hasShadowMul bool) {
	if shadowSize != "" {
		setShadowSize(a, shadowSize)
	}
	if !hasShadowMul {
		return
	}

	mul := shadowMul
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

func applyEffectsOverrides(a *aesthetic.Aesthetic, effects map[string]string) {
	for k, v := range effects {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		a.Tokens.Effects[k] = v
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

func writeAestheticBlock(sb *strings.Builder, selector, name string, a *aesthetic.Aesthetic, selected, escapedManifest string) {
	if a == nil {
		return
	}

	fmt.Fprintf(sb, "/* Aesthetic: %s */\n", name)
	sb.WriteString(selector)
	sb.WriteString(" {\n")

	if selector == ":root" {
		fmt.Fprintf(sb, "  --aesthetic-selected: %q;\n", selected)
		fmt.Fprintf(sb, "  --aesthetic-manifest: '%s';\n", escapedManifest)
		sb.WriteString("  --aesthetic-switcher-enabled: 1;\n")
	}

	// Radius tokens
	writeVarMap(sb, "radius", a.Tokens.Radius)

	// Border tokens
	writeVarMap(sb, "border", a.Tokens.Border)

	// Shadow tokens
	writeVarMap(sb, "shadow", a.Tokens.Shadow)

	// Effects tokens
	writeVarMap(sb, "fx", a.Tokens.Effects)

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
