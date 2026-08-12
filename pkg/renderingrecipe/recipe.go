// Package renderingrecipe compiles normalized rendering semantics into portable,
// deterministic assets. Consumers receive the bundle; they do not draw it.
package renderingrecipe

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/WaylonWalker/markata-go/pkg/renderingcontract"
)

//go:embed testdata/*.json
var testdata embed.FS

const (
	SchemaVersion   = "rendering-recipe-v1"
	ContractVersion = 1
	semanticDomain  = "rendering-semantic-v1\x00"
	recipeDomain    = "rendering-recipe-v1\x00"
)

type Theme struct {
	ContractVersion int                          `json:"contract_version"`
	Palette         string                       `json:"palette"`
	Aesthetic       string                       `json:"aesthetic"`
	Fontpack        string                       `json:"fontpack"`
	Texture         renderingcontract.Dial       `json:"texture"`
	HeadingTexture  renderingcontract.Dial       `json:"heading_texture"`
	Motif           renderingcontract.MotifState `json:"motif"`
	Variables       map[string]string            `json:"variables,omitempty"`
}

type Asset struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	MediaType string `json:"media_type"`
	ViewBox   string `json:"view_box"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Pass struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	Mode         string `json:"mode"`
	Asset        string `json:"asset"`
	MediaType    string `json:"media_type"`
	ViewBox      string `json:"view_box"`
	Repeat       string `json:"repeat"`
	ScaleMilli   int    `json:"scale_milli"`
	Scope        string `json:"scope,omitempty"`
	OpacityMilli *int   `json:"opacity_milli,omitempty"`
}

type FontAsset struct {
	Role          string   `json:"role"`
	Family        string   `json:"family"`
	AssetPath     string   `json:"asset_path"`
	SHA256        string   `json:"sha256"`
	Weight        int      `json:"weight"`
	Style         string   `json:"style"`
	FallbackStack []string `json:"fallback_stack"`
}

type Manifest struct {
	SchemaVersion   string      `json:"schema_version"`
	ContractVersion int         `json:"contract_version"`
	SemanticHash    string      `json:"semantic_hash"`
	RecipeHash      string      `json:"recipe_hash,omitempty"`
	Assets          []Asset     `json:"assets"`
	Passes          []Pass      `json:"passes"`
	Fonts           []FontAsset `json:"fonts"`
}

type Bundle struct {
	Manifest Manifest
	Assets   map[string][]byte
}

func LoadCanonicalTheme() (Theme, error) {
	data, err := testdata.ReadFile("testdata/canonical-theme.json")
	if err != nil {
		return Theme{}, err
	}
	var raw map[string]any
	if err := decodeStrict(data, &raw); err != nil {
		return Theme{}, err
	}
	return normalize(raw)
}

func normalize(raw map[string]any) (Theme, error) {
	if raw == nil {
		return Theme{}, errors.New("theme must be an object")
	}
	if nested, ok := raw["theme"].(map[string]any); ok {
		if err := rejectKeys(raw, "theme"); err != nil {
			return Theme{}, err
		}
		raw = nested
	}
	if err := rejectKeys(raw, "contract_version", "palette", "aesthetic", "fontpack", "texture", "heading_texture", "motif", "variables"); err != nil {
		return Theme{}, err
	}
	contract, err := renderingcontract.Load()
	if err != nil {
		return Theme{}, err
	}
	get := func(key string, fallback any) any {
		if value, ok := raw[key]; ok {
			return value
		}
		return fallback
	}
	palette := stringValue(get("palette", contract.Defaults.Palette))
	aesthetic := stringValue(get("aesthetic", contract.Defaults.Aesthetic))
	fontpack := stringValue(get("fontpack", contract.Defaults.Fontpack))
	if alias, ok := contract.Aliases[fontpack]; ok {
		fontpack = alias
	}
	texture, err := dialValue(raw, "texture", contract.Defaults.Texture)
	if err != nil {
		return Theme{}, err
	}
	heading, err := dialValue(raw, "heading_texture", contract.Defaults.HeadingTexture)
	if err != nil {
		return Theme{}, err
	}
	motif, err := motifValue(raw, contract.Defaults.Motif)
	if err != nil {
		return Theme{}, err
	}
	variables := map[string]string{}
	if values, ok := raw["variables"].(map[string]any); ok {
		for key, value := range values {
			variables[key] = stringValue(value)
		}
	}
	return Theme{ContractVersion: 1, Palette: palette, Aesthetic: aesthetic, Fontpack: fontpack, Texture: texture, HeadingTexture: heading, Motif: motif, Variables: variables}, nil
}

func dialValue(raw map[string]any, key string, fallback renderingcontract.Dial) (renderingcontract.Dial, error) {
	dial := fallback
	if value, ok := raw[key].(map[string]any); ok {
		if err := rejectKeys(value, "kind", "color_mix", "scale", "scope", "glyph", "size", "gap", "row_offset", "wobble", "scatter", "layer", "color", "url"); err != nil {
			return dial, fmt.Errorf("%s: %w", key, err)
		}
		if err := decodeMap(value, &dial); err != nil {
			return dial, fmt.Errorf("%s: %w", key, err)
		}
	}
	dial.ColorMix = fixed(dial.ColorMix, 0, 1)
	dial.Scale = fixed(dial.Scale, .25, 3)
	return dial, nil
}

func rejectKeys(value map[string]any, allowed ...string) error {
	set := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		set[key] = struct{}{}
	}
	for key := range value {
		if _, ok := set[key]; !ok {
			return fmt.Errorf("obsolete or unknown recipe field %q", key)
		}
	}
	return nil
}

func motifValue(raw map[string]any, fallback renderingcontract.Dial) (renderingcontract.MotifState, error) {
	if value, ok := raw["motif"].(map[string]any); ok {
		if err := rejectKeys(value, "kind", "color_mix", "glyph", "size", "gap", "row_offset", "wobble", "scatter", "layer", "color", "url"); err != nil {
			return renderingcontract.MotifState{}, fmt.Errorf("motif: %w", err)
		}
	}
	dial, err := dialValue(raw, "motif", fallback)
	if err != nil {
		return renderingcontract.MotifState{}, err
	}
	return renderingcontract.MotifState{Kind: dial.Kind, ColorMix: dial.ColorMix, Glyph: dial.Glyph, Size: dial.Size, Gap: dial.Gap, RowOffset: fixed(dial.RowOffset, 0, 1), Wobble: fixed(dial.Wobble, 0, 1), Scatter: fixed(dial.Scatter, 0, 1), Layer: dial.Layer, Color: dial.Color, URL: dial.URL}, nil
}

func fixed(value, min, max float64) float64 {
	value = math.Max(min, math.Min(max, value))
	return math.Round(value*1000) / 1000
}
func stringValue(value any) string { return fmt.Sprint(value) }

func SemanticHash(theme Theme) (string, error) {
	b, err := canonicalJSON(theme)
	if err != nil {
		return "", err
	}
	return digest([]byte(semanticDomain), b), nil
}

// AssetHash is the digest of the exact standalone bytes. Domain separation is
// reserved for semantic and manifest identities; asset digests are portable
// content addresses and must match ordinary SHA-256 tooling.
func AssetHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Compile(theme Theme) (Bundle, error) {
	if err := validateTheme(theme); err != nil {
		return Bundle{}, err
	}
	semantic, err := SemanticHash(theme)
	if err != nil {
		return Bundle{}, err
	}
	assets := map[string][]byte{}
	add := func(path string, data []byte, media, view string, width, height int) Asset {
		assets[path] = append([]byte(nil), data...)
		return Asset{Path: path, SHA256: AssetHash(data), MediaType: media, ViewBox: view, Width: width, Height: height}
	}
	colors := paletteColors(theme.Palette)
	surface := add("assets/surface-screenprint-v1.svg", surfaceSVG(theme.Texture.ColorMix, colors.background, colors.ink), "image/svg+xml", "0 0 180 180", 180, 180)
	heading := add("assets/heading-splatter-v1.svg", headingSVG(theme.HeadingTexture.ColorMix), "image/svg+xml", "0 0 180 180", 180, 180)
	motif := add("assets/motif-block-w-v1.svg", motifSVG(theme.Motif, colors.ink), "image/svg+xml", "0 0 28480 10060", 28480, 10060)
	fonts := canonicalFonts()
	passes := []Pass{}
	if theme.Motif.Layer == "under" || theme.Motif.Layer == "sandwich" {
		passes = append(passes, Pass{ID: "motif-under", Role: "motif-under", Mode: "image", Asset: motif.Path, MediaType: motif.MediaType, ViewBox: motif.ViewBox, Repeat: "repeat", ScaleMilli: 1000})
	}
	passes = append(passes, Pass{ID: "surface-texture", Role: "surface texture", Mode: "image", Asset: surface.Path, MediaType: surface.MediaType, ViewBox: surface.ViewBox, Repeat: "repeat", ScaleMilli: int(fixed(theme.Texture.Scale, .25, 3) * 1000), Scope: theme.Texture.Scope})
	if theme.Motif.Layer == "over" || theme.Motif.Layer == "sandwich" {
		passes = append(passes, Pass{ID: "motif-over", Role: "motif-over", Mode: "image", Asset: motif.Path, MediaType: motif.MediaType, ViewBox: motif.ViewBox, Repeat: "repeat", ScaleMilli: 1000})
	}
	passes = append(passes,
		Pass{ID: "heading-wear", Role: "heading wear", Mode: "alpha-mask", Asset: heading.Path, MediaType: heading.MediaType, ViewBox: heading.ViewBox, Repeat: "repeat", ScaleMilli: int(fixed(theme.HeadingTexture.Scale, .25, 3) * 1000)},
	)
	manifest := Manifest{SchemaVersion: SchemaVersion, ContractVersion: ContractVersion, SemanticHash: semantic, Assets: []Asset{surface, heading, motif}, Passes: passes, Fonts: fonts}
	identity := manifest
	identity.RecipeHash = ""
	encoded, err := canonicalJSON(identity)
	if err != nil {
		return Bundle{}, err
	}
	manifest.RecipeHash = digest([]byte(recipeDomain), encoded)
	return Bundle{Manifest: manifest, Assets: assets}, nil
}

func canonicalFonts() []FontAsset {
	return []FontAsset{
		{Role: "body", Family: "Space Grotesk", AssetPath: "internal/fontcatalog/space-grotesk/space-grotesk-prose-core.woff2", SHA256: "5010c6e79c7e65a058dfd39a71d8d60f7b56070a3c32cc911d9dddbaec7f0851", Weight: 400, Style: "normal", FallbackStack: []string{"system-ui", "sans-serif"}},
		{Role: "heading", Family: "Knewave", AssetPath: "internal/fontcatalog/knewave/knewave-display-core.woff2", SHA256: "56fc586096f8e158fc214e3de31ee47c15ff80fc0abd5e0646ddff47a6af27ba", Weight: 400, Style: "normal", FallbackStack: []string{"cursive", "sans-serif"}},
		{Role: "mono", Family: "DM Mono", AssetPath: "internal/fontcatalog/dm-mono/dm-mono-code-core.woff2", SHA256: "90c42888e73fa88ebf3d0e27e0461535acd661e3892bbcb6eb21047f33b67c32", Weight: 400, Style: "normal", FallbackStack: []string{"ui-monospace", "monospace"}},
	}
}

func digest(domain, data []byte) string {
	sum := sha256.Sum256(append(append([]byte{}, domain...), data...))
	return hex.EncodeToString(sum[:])
}

func canonicalJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	if err := writeCanonical(&out, value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func writeCanonical(out *bytes.Buffer, value any) error {
	switch value := value.(type) {
	case nil:
		out.WriteString("null")
	case string:
		if !utf8.ValidString(value) {
			return errors.New("invalid UTF-8 string")
		}
		b, _ := json.Marshal(value)
		out.Write(b)
	case bool:
		if value {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case int:
		out.WriteString(strconv.Itoa(value))
	case int64:
		out.WriteString(strconv.FormatInt(value, 10))
	case float64:
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			out.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		} else {
			return errors.New("non-finite number")
		}
	case []string:
		out.WriteByte('[')
		for i, item := range value {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := writeCanonical(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case []Asset:
		return writeStruct(out, value)
	case []Pass:
		return writeStruct(out, value)
	case []FontAsset:
		return writeStruct(out, value)
	case []any:
		out.WriteByte('[')
		for i, item := range value {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := writeCanonical(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			b, _ := json.Marshal(key)
			out.Write(b)
			out.WriteByte(':')
			if err := writeCanonical(out, value[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var generic any
		if err := json.Unmarshal(b, &generic); err != nil {
			return err
		}
		return writeCanonical(out, generic)
	}
	return nil
}
func writeStruct(out *bytes.Buffer, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var generic any
	if err = decodeStrict(b, &generic); err != nil {
		return err
	}
	return writeCanonical(out, generic)
}

func decodeStrict(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("input is not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var value any
	if err := decodeValue(dec, &value); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}
func decodeMap(value map[string]any, target any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
func decodeValue(dec *json.Decoder, out *any) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	switch token := token.(type) {
	case json.Delim:
		if token == '{' {
			m := map[string]any{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				ks := key.(string)
				if _, exists := m[ks]; exists {
					return fmt.Errorf("duplicate JSON key %q", ks)
				}
				var v any
				if err := decodeValue(dec, &v); err != nil {
					return err
				}
				m[ks] = v
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
			*out = m
			return nil
		}
		if token == '[' {
			a := []any{}
			for dec.More() {
				var v any
				if err := decodeValue(dec, &v); err != nil {
					return err
				}
				a = append(a, v)
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
			*out = a
			return nil
		}
		return errors.New("invalid JSON delimiter")
	case json.Number:
		if _, err := strconv.ParseFloat(string(token), 64); err != nil {
			return errors.New("invalid or non-finite number")
		}
		*out = token
		return nil
	default:
		*out = token
		return nil
	}
}

type paletteColorSet struct{ background, ink string }

func paletteColors(id string) paletteColorSet {
	contract, err := renderingcontract.Load()
	if err != nil {
		return paletteColorSet{"#fff", "#000"}
	}
	for _, palette := range contract.Palettes {
		if palette.ID == id {
			final := renderingcontract.FinalRenderPalette(palette)
			return paletteColorSet{final["background"], final["text"]}
		}
	}
	return paletteColorSet{"#fff", "#000"}
}
func validateTheme(theme Theme) error {
	if theme.ContractVersion != ContractVersion {
		return fmt.Errorf("unsupported contract_version %d", theme.ContractVersion)
	}
	if theme.Fontpack != "brush" {
		return fmt.Errorf("fontpack %q is not compiled in recipe v1", theme.Fontpack)
	}
	if theme.Texture.Kind != "screenprint" || theme.HeadingTexture.Kind != "splatter" || theme.Motif.Kind != "block-w" {
		return fmt.Errorf("recipe v1 supports screenprint, splatter, and block-w representative primitives only")
	}
	if theme.Motif.Layer != "under" && theme.Motif.Layer != "over" && theme.Motif.Layer != "sandwich" {
		return fmt.Errorf("invalid motif layer %q", theme.Motif.Layer)
	}
	return nil
}
func surfaceSVG(mix float64, background, ink string) []byte {
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 180 180"><rect width="180" height="180" fill="%s"/><circle cx="32" cy="48" r="11" fill="%s" opacity="%.3f"/><circle cx="121" cy="137" r="8" fill="%s" opacity="%.3f"/></svg>`, background, ink, fixed(mix, 0, 1), ink, fixed(mix, 0, 1)))
}
func headingSVG(mix float64) []byte {
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 180 180"><mask id="wear"><rect width="180" height="180" fill="white"/><circle cx="31" cy="47" r="%.3f" fill="black"/><circle cx="129" cy="113" r="%.3f" fill="black"/></mask><rect width="180" height="180" fill="white" mask="url(#wear)"/></svg>`, 1+fixed(mix, 0, 1)*5, 2+fixed(mix, 0, 1)*4))
}
func motifSVG(m renderingcontract.MotifState, color string) []byte {
	const path = "M0 905.76L0 0L414.385 2.88L412.703 512.75L606.319 511.128L608.484 4.32L1074.43 0L1074.67 501.942L1269.33 505.859L1268.94 0L1688.4 0L1688.4 905.76Z"
	var svg strings.Builder
	svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 28480 10060" data-field="16x10">`)
	for index := 0; index < 160; index++ {
		row, column := index/16, index%16
		x := float64(column)*1780 + float64(row%2)*m.RowOffset*890
		y := float64(row) * 1006
		rotation := (float64(index%7) - 3) * m.Wobble * 2
		scale := 1 + (float64((index*13)%11)-5)*m.Scatter*.01
		fmt.Fprintf(&svg, `<path data-index="%d" fill="%s" transform="translate(%.3f %.3f) rotate(%.3f 844.2 452.9) scale(%.5f)" d="%s"/>`, index, color, x, y, rotation, scale, path)
	}
	svg.WriteString(`</svg>`)
	return []byte(svg.String())
}
