package renderingcontract

import (
	"encoding/json"
	"fmt"
	"os"
)

// RenderPlanFixtures is the portable fixture document. Inputs are deliberately
// JSON values so the same document can be consumed by non-Go renderers.
type RenderPlanFixtures struct {
	SchemaVersion   int                 `json:"schema_version"`
	ContractVersion int                 `json:"contract_version"`
	Fixtures        []RenderPlanFixture `json:"fixtures"`
}

type RenderPlanFixture struct {
	ID                 string         `json:"id"`
	Tags               []string       `json:"tags"`
	Inputs             map[string]any `json:"inputs"`
	ExpectedNormalized SemanticState  `json:"expected_normalized"`
	ExpectedPlan       RenderPlan     `json:"expected_plan"`
}

// SemanticState contains normalized, renderer-independent theme semantics.
type SemanticState struct {
	ContractVersion int            `json:"contract_version"`
	Palette         string         `json:"palette"`
	Aesthetic       string         `json:"aesthetic"`
	Fontpack        string         `json:"fontpack"`
	Texture         Dial           `json:"texture"`
	HeadingTexture  Dial           `json:"heading_texture"`
	Motif           MotifState     `json:"motif"`
	Variables       map[string]any `json:"variables,omitempty"`
}

// RenderPlan describes semantic passes. Recipe is a contract recipe ID, never
// a browser URL, CSS declaration, or other consumer-specific representation.
type RenderPlan struct {
	Palette        string        `json:"palette"`
	Aesthetic      string        `json:"aesthetic"`
	Fontpack       string        `json:"fontpack"`
	Texture        RenderPass    `json:"texture"`
	HeadingTexture RenderPass    `json:"heading_texture"`
	Motif          MotifPlan     `json:"motif"`
	Layers         []RenderLayer `json:"layers"`
}

type RenderPass struct {
	Recipe string  `json:"recipe"`
	Mix    float64 `json:"mix"`
	Scale  float64 `json:"scale"`
	Scope  string  `json:"scope,omitempty"`
}

type MotifPlan struct {
	Recipe  string  `json:"recipe"`
	Kind    string  `json:"kind"`
	Glyph   string  `json:"glyph"`
	Size    string  `json:"size"`
	Gap     string  `json:"gap"`
	Offset  float64 `json:"row_offset"`
	Wobble  float64 `json:"wobble"`
	Scatter float64 `json:"scatter"`
	Color   string  `json:"color"`
	Mix     float64 `json:"mix"`
	URL     string  `json:"url,omitempty"`
	Layer   string  `json:"layer"`
}

type RenderLayer struct {
	Role   string  `json:"role"`
	Recipe string  `json:"recipe"`
	Mix    float64 `json:"mix"`
}

// LoadRenderPlanFixtures loads and validates one fixture document.
func LoadRenderPlanFixtures(path string) (RenderPlanFixtures, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RenderPlanFixtures{}, fmt.Errorf("read render-plan fixtures: %w", err)
	}
	var fixtures RenderPlanFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return fixtures, fmt.Errorf("decode render-plan fixtures: %w", err)
	}
	if err := ValidateRenderPlanFixtures(fixtures); err != nil {
		return fixtures, err
	}
	return fixtures, nil
}

// ValidateRenderPlanFixtures checks document shape and required scenario coverage.
func ValidateRenderPlanFixtures(f RenderPlanFixtures) error {
	if f.SchemaVersion != 1 || f.ContractVersion != 1 {
		return fmt.Errorf("unsupported fixture or contract version")
	}
	required := map[string]bool{"default": false, "variants": false, "mix-endpoints": false, "heading-modes": false, "scales": false, "motif-layers": false, "motif-geometry": false, "motif-url": false, "legacy": false, "zero": false, "stress": false}
	seen := map[string]bool{}
	for _, fixture := range f.Fixtures {
		if fixture.ID == "" || seen[fixture.ID] {
			return fmt.Errorf("duplicate or empty fixture id %q", fixture.ID)
		}
		seen[fixture.ID] = true
		for _, tag := range fixture.Tags {
			if _, ok := required[tag]; ok {
				required[tag] = true
			}
		}
		if fixture.ExpectedNormalized.Palette == "" || fixture.ExpectedPlan.Palette == "" || len(fixture.ExpectedPlan.Layers) == 0 {
			return fmt.Errorf("fixture %q is incomplete", fixture.ID)
		}
	}
	for tag, ok := range required {
		if !ok {
			return fmt.Errorf("missing required fixture matrix %q", tag)
		}
	}
	return nil
}

// ResolveRenderPlan normalizes canonical or legacy map input and expands it to
// semantic passes. It rejects unknown palette IDs rather than silently falling back.
func ResolveRenderPlan(raw map[string]any) (SemanticState, RenderPlan, []string, error) {
	c, err := Load()
	if err != nil {
		return SemanticState{}, RenderPlan{}, nil, err
	}
	normalized, warnings := NormalizeTheme(raw)
	version := intValue(normalized["contract_version"])
	if version != c.ContractVersion {
		return SemanticState{}, RenderPlan{}, warnings, fmt.Errorf("unsupported contract version %d", version)
	}
	state := SemanticState{ContractVersion: version, Palette: stringValue(normalized["palette"]), Aesthetic: stringValue(normalized["aesthetic"]), Fontpack: stringValue(normalized["fontpack"]), Variables: mapValue(normalized["variables"])}
	if state.Palette == "" {
		return state, RenderPlan{}, warnings, fmt.Errorf("missing palette")
	}
	paletteOK := false
	for _, p := range c.Palettes {
		if p.ID == state.Palette {
			paletteOK = true
			break
		}
	}
	if !paletteOK {
		return state, RenderPlan{}, warnings, fmt.Errorf("unknown palette %q", state.Palette)
	}
	if _, ok := c.Aesthetics[state.Aesthetic]; !ok {
		return state, RenderPlan{}, warnings, fmt.Errorf("unknown aesthetic %q", state.Aesthetic)
	}
	if _, ok := c.Fontpacks[state.Fontpack]; !ok {
		return state, RenderPlan{}, warnings, fmt.Errorf("unknown fontpack %q", state.Fontpack)
	}
	applyDialDefaults(normalized, "texture", c.Defaults.Texture)
	applyDialDefaults(normalized, "heading_texture", c.Defaults.HeadingTexture)
	applyDialDefaults(normalized, "motif", c.Defaults.Motif)
	state.Texture = dialValue(normalized["texture"])
	state.HeadingTexture = dialValue(normalized["heading_texture"])
	dial := dialValue(normalized["motif"])
	state.Motif = MotifState{Kind: dial.Kind, ColorMix: dial.ColorMix, Glyph: dial.Glyph, Size: dial.Size, Gap: dial.Gap, RowOffset: dial.RowOffset, Wobble: dial.Wobble, Scatter: dial.Scatter, Layer: dial.Layer, Color: dial.Color, URL: dial.URL}
	if state.HeadingTexture.Kind == "inherit" {
		state.HeadingTexture.Kind = state.Texture.Kind
	}
	if !contains(c.Enums["textures"], state.Texture.Kind) || !contains(c.Enums["heading_textures"], state.HeadingTexture.Kind) || !contains(c.Enums["motifs"], state.Motif.Kind) || !contains(c.Enums["motif_layers"], state.Motif.Layer) {
		return state, RenderPlan{}, warnings, fmt.Errorf("unsupported render recipe or layer")
	}
	plan := RenderPlan{Palette: state.Palette, Aesthetic: state.Aesthetic, Fontpack: state.Fontpack,
		Texture:        RenderPass{Recipe: state.Texture.Kind, Mix: state.Texture.ColorMix, Scale: state.Texture.Scale, Scope: state.Texture.Scope},
		HeadingTexture: RenderPass{Recipe: state.HeadingTexture.Kind, Mix: state.HeadingTexture.ColorMix, Scale: state.HeadingTexture.Scale},
		Motif:          MotifPlan{Recipe: state.Motif.Kind, Kind: state.Motif.Kind, Glyph: state.Motif.Glyph, Size: state.Motif.Size, Gap: state.Motif.Gap, Offset: state.Motif.RowOffset, Wobble: state.Motif.Wobble, Scatter: state.Motif.Scatter, Color: state.Motif.Color, Mix: state.Motif.ColorMix, URL: state.Motif.URL, Layer: state.Motif.Layer}}
	plan.Layers = []RenderLayer{{Role: "surface", Recipe: state.Texture.Kind, Mix: state.Texture.ColorMix}}
	if state.Motif.Kind != "off" {
		if state.Motif.Layer == "under" || state.Motif.Layer == "sandwich" {
			plan.Layers = append([]RenderLayer{{Role: "motif-under", Recipe: state.Motif.Kind, Mix: state.Motif.ColorMix}}, plan.Layers...)
		}
		if state.Motif.Layer == "over" || state.Motif.Layer == "sandwich" {
			plan.Layers = append(plan.Layers, RenderLayer{Role: "motif-over", Recipe: state.Motif.Kind, Mix: state.Motif.ColorMix})
		}
	}
	return state, plan, warnings, nil
}

// ResolveFixture resolves a fixture's inputs without giving the fixture
// expected values any authority. This keeps expected snapshots test oracles.
func ResolveFixture(fixture RenderPlanFixture) (SemanticState, RenderPlan, []string, error) {
	return ResolveRenderPlan(fixture.Inputs)
}

func stringValue(v any) string { s, _ := v.(string); return s }
func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func mapValue(v any) map[string]any { m, _ := v.(map[string]any); return m }

func applyDialDefaults(theme map[string]any, name string, defaults Dial) {
	m, _ := theme[name].(map[string]any)
	if m == nil {
		m = map[string]any{}
		theme[name] = m
	}
	if _, ok := m["kind"]; !ok {
		m["kind"] = defaults.Kind
	}
	if _, ok := m["color_mix"]; !ok {
		m["color_mix"] = defaults.ColorMix
	}
	if _, ok := m["scale"]; !ok {
		m["scale"] = defaults.Scale
	}
	if _, ok := m["scope"]; !ok && defaults.Scope != "" {
		m["scope"] = defaults.Scope
	}
	if _, ok := m["glyph"]; !ok && defaults.Glyph != "" {
		m["glyph"] = defaults.Glyph
	}
	if _, ok := m["size"]; !ok && defaults.Size != "" {
		m["size"] = defaults.Size
	}
	if _, ok := m["gap"]; !ok && defaults.Gap != "" {
		m["gap"] = defaults.Gap
	}
	if _, ok := m["row_offset"]; !ok {
		m["row_offset"] = defaults.RowOffset
	}
	if _, ok := m["wobble"]; !ok {
		m["wobble"] = defaults.Wobble
	}
	if _, ok := m["scatter"]; !ok {
		m["scatter"] = defaults.Scatter
	}
	if _, ok := m["layer"]; !ok && defaults.Layer != "" {
		m["layer"] = defaults.Layer
	}
	if _, ok := m["color"]; !ok && defaults.Color != "" {
		m["color"] = defaults.Color
	}
	if _, ok := m["url"]; !ok && defaults.URL != "" {
		m["url"] = defaults.URL
	}
}

func dialValue(v any) Dial {
	m, _ := v.(map[string]any)
	return Dial{Kind: stringValue(m["kind"]), ColorMix: NormalizeMix(m["color_mix"], 0), Scale: NormalizeScale(m["scale"], 1), Scope: stringValue(m["scope"]), Glyph: stringValue(m["glyph"]), Size: stringValue(m["size"]), Gap: stringValue(m["gap"]), RowOffset: NormalizeMix(m["row_offset"], 0), Wobble: NormalizeMix(m["wobble"], 0), Scatter: NormalizeMix(m["scatter"], 0), Layer: stringValue(m["layer"]), Color: stringValue(m["color"]), URL: stringValue(m["url"])}
}
