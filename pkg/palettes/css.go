package palettes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CSSFormat specifies how CSS should be generated.
type CSSFormat struct {
	// UseReferences determines whether semantic/component colors should
	// use var(--palette-*) references or resolved hex values.
	UseReferences bool

	// Prefix for raw color CSS variables (default: "palette")
	RawPrefix string

	// Prefix for semantic color CSS variables (default: "color")
	SemanticPrefix string

	// Prefix for component color CSS variables (default: none, use name directly)
	ComponentPrefix string

	// IncludeRaw includes raw color definitions
	IncludeRaw bool

	// IncludeSemantic includes semantic color definitions
	IncludeSemantic bool

	// IncludeComponents includes component color definitions
	IncludeComponents bool

	// Minify removes whitespace for smaller output
	Minify bool
}

// DefaultCSSFormat returns the default CSS format options.
func DefaultCSSFormat() CSSFormat {
	return CSSFormat{
		UseReferences:     true,
		RawPrefix:         "palette",
		SemanticPrefix:    "color",
		ComponentPrefix:   "",
		IncludeRaw:        true,
		IncludeSemantic:   true,
		IncludeComponents: true,
		Minify:            false,
	}
}

// GenerateCSS generates CSS custom properties from the palette.
func (p *Palette) GenerateCSS() string {
	return p.GenerateCSSWithFormat(DefaultCSSFormat())
}

// GenerateCSSWithFormat generates CSS with custom format options.
func (p *Palette) GenerateCSSWithFormat(format CSSFormat) string {
	var buf bytes.Buffer

	nl := "\n"
	indent := "  "
	if format.Minify {
		nl = ""
		indent = ""
	}

	// Write header comment
	if !format.Minify {
		fmt.Fprintf(&buf, "/* Generated from %s palette */\n", p.Name)
	}

	buf.WriteString(":root {")
	buf.WriteString(nl)

	// Write raw colors
	if format.IncludeRaw && len(p.Colors) > 0 {
		if !format.Minify {
			buf.WriteString(indent + "/* Raw colors */\n")
		}
		writeColorVars(&buf, p.Colors, format.RawPrefix, "", indent, nl, format.Minify)
	}

	// Write semantic colors
	if format.IncludeSemantic && len(p.Semantic) > 0 {
		if !format.Minify {
			buf.WriteString(nl + indent + "/* Semantic colors */\n")
		}
		if format.UseReferences {
			writeReferenceVars(&buf, p.Semantic, format.SemanticPrefix, format.RawPrefix, p, indent, nl, format.Minify)
		} else {
			writeResolvedVars(&buf, p.Semantic, format.SemanticPrefix, p, indent, nl, format.Minify)
		}
	}

	// Write component colors
	if format.IncludeComponents && len(p.Components) > 0 {
		if !format.Minify {
			buf.WriteString(nl + indent + "/* Component colors */\n")
		}
		if format.UseReferences {
			writeReferenceVars(&buf, p.Components, format.ComponentPrefix, format.RawPrefix, p, indent, nl, format.Minify)
		} else {
			writeResolvedVars(&buf, p.Components, format.ComponentPrefix, p, indent, nl, format.Minify)
		}
	}

	buf.WriteString("}")
	buf.WriteString(nl)

	return buf.String()
}

// writeColorVars writes CSS variables with hex values.
func writeColorVars(buf *bytes.Buffer, colors map[string]string, prefix, suffix, indent, _ string, minify bool) {
	names := sortedKeys(colors)
	for _, name := range names {
		hex := colors[name]
		varName := cssVarName(prefix, name, suffix)
		if minify {
			fmt.Fprintf(buf, "%s:%s;", varName, hex)
		} else {
			fmt.Fprintf(buf, "%s%s: %s;\n", indent, varName, hex)
		}
	}
}

// writeReferenceVars writes CSS variables that reference other variables.
func writeReferenceVars(buf *bytes.Buffer, colors map[string]string, prefix, rawPrefix string, p *Palette, indent, _ string, minify bool) {
	names := sortedKeys(colors)
	for _, name := range names {
		ref := colors[name]
		varName := cssVarName(prefix, name, "")

		// Determine if ref is a raw color, semantic, or direct hex
		var value string
		if isHexColor(ref) {
			value = normalizeHexColor(ref)
		} else if _, ok := p.Colors[ref]; ok {
			// Reference to raw color
			value = fmt.Sprintf("var(%s)", cssVarName(rawPrefix, ref, ""))
		} else if _, ok := p.Semantic[ref]; ok {
			// Reference to semantic color (from component)
			value = fmt.Sprintf("var(%s)", cssVarName("color", ref, ""))
		} else {
			// Fallback to resolved value
			value = p.Resolve(name)
			if value == "" {
				value = ref // Use as-is if can't resolve
			}
		}

		if minify {
			fmt.Fprintf(buf, "%s:%s;", varName, value)
		} else {
			fmt.Fprintf(buf, "%s%s: %s;\n", indent, varName, value)
		}
	}
}

// writeResolvedVars writes CSS variables with resolved hex values.
func writeResolvedVars(buf *bytes.Buffer, colors map[string]string, prefix string, p *Palette, indent, _ string, minify bool) {
	names := sortedKeys(colors)
	for _, name := range names {
		varName := cssVarName(prefix, name, "")
		hex := p.Resolve(name)
		if hex == "" {
			hex = colors[name] // Fallback to raw value
		}

		if minify {
			fmt.Fprintf(buf, "%s:%s;", varName, hex)
		} else {
			fmt.Fprintf(buf, "%s%s: %s;\n", indent, varName, hex)
		}
	}
}

// cssVarName generates a CSS variable name.
func cssVarName(prefix, name, suffix string) string {
	parts := []string{}
	if prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, name)
	if suffix != "" {
		parts = append(parts, suffix)
	}
	return "--" + strings.Join(parts, "-")
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GenerateSCSS generates SCSS variables from the palette.
func (p *Palette) GenerateSCSS() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("// Generated from %s palette\n\n", p.Name))

	// Raw colors as SCSS variables
	buf.WriteString("// Raw colors\n")
	names := sortedKeys(p.Colors)
	for _, name := range names {
		buf.WriteString(fmt.Sprintf("$palette-%s: %s;\n", name, p.Colors[name]))
	}

	// Semantic colors
	buf.WriteString("\n// Semantic colors\n")
	names = sortedKeys(p.Semantic)
	for _, name := range names {
		ref := p.Semantic[name]
		if isHexColor(ref) {
			buf.WriteString(fmt.Sprintf("$color-%s: %s;\n", name, normalizeHexColor(ref)))
		} else {
			buf.WriteString(fmt.Sprintf("$color-%s: $palette-%s;\n", name, ref))
		}
	}

	// Component colors
	if len(p.Components) > 0 {
		buf.WriteString("\n// Component colors\n")
		names = sortedKeys(p.Components)
		for _, name := range names {
			ref := p.Components[name]
			if isHexColor(ref) {
				buf.WriteString(fmt.Sprintf("$%s: %s;\n", name, normalizeHexColor(ref)))
			} else if _, ok := p.Colors[ref]; ok {
				buf.WriteString(fmt.Sprintf("$%s: $palette-%s;\n", name, ref))
			} else if _, ok := p.Semantic[ref]; ok {
				buf.WriteString(fmt.Sprintf("$%s: $color-%s;\n", name, ref))
			} else {
				hex := p.Resolve(name)
				buf.WriteString(fmt.Sprintf("$%s: %s;\n", name, hex))
			}
		}
	}

	return buf.String()
}

// ExportJSON exports the palette as JSON with resolved colors.
type PaletteExport struct {
	Name        string            `json:"name"`
	Variant     string            `json:"variant"`
	Author      string            `json:"author,omitempty"`
	License     string            `json:"license,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	Description string            `json:"description,omitempty"`
	Colors      map[string]string `json:"colors"`
	Semantic    map[string]string `json:"semantic"`
	Components  map[string]string `json:"components,omitempty"`
	Resolved    map[string]string `json:"resolved,omitempty"`
}

// ExportJSON exports the palette as JSON.
// If includeResolved is true, includes all resolved hex values.
func (p *Palette) ExportJSON(includeResolved bool) ([]byte, error) {
	export := PaletteExport{
		Name:        p.Name,
		Variant:     string(p.Variant),
		Author:      p.Author,
		License:     p.License,
		Homepage:    p.Homepage,
		Description: p.Description,
		Colors:      p.Colors,
		Semantic:    p.Semantic,
		Components:  p.Components,
	}

	if includeResolved {
		resolved, err := p.ResolveAll()
		if err == nil {
			export.Resolved = resolved
		}
	}

	return json.MarshalIndent(export, "", "  ")
}

// GenerateTailwind generates Tailwind CSS configuration for the palette.
func (p *Palette) GenerateTailwind() string {
	var buf bytes.Buffer

	buf.WriteString("// Generated Tailwind CSS config for " + p.Name + "\n")
	buf.WriteString("// Add to tailwind.config.js under theme.extend.colors\n\n")
	buf.WriteString("module.exports = {\n")
	buf.WriteString("  theme: {\n")
	buf.WriteString("    extend: {\n")
	buf.WriteString("      colors: {\n")

	// Raw colors under 'palette' namespace
	buf.WriteString("        palette: {\n")
	names := sortedKeys(p.Colors)
	for i, name := range names {
		comma := ","
		if i == len(names)-1 {
			comma = ""
		}
		buf.WriteString(fmt.Sprintf("          '%s': '%s'%s\n", name, p.Colors[name], comma))
	}
	buf.WriteString("        },\n")

	// Semantic colors with resolved values
	names = sortedKeys(p.Semantic)
	for i, name := range names {
		hex := p.Resolve(name)
		comma := ","
		if i == len(names)-1 && len(p.Components) == 0 {
			comma = ""
		}
		buf.WriteString(fmt.Sprintf("        '%s': '%s'%s\n", name, hex, comma))
	}

	// Component colors
	if len(p.Components) > 0 {
		names = sortedKeys(p.Components)
		for i, name := range names {
			hex := p.Resolve(name)
			comma := ","
			if i == len(names)-1 {
				comma = ""
			}
			buf.WriteString(fmt.Sprintf("        '%s': '%s'%s\n", name, hex, comma))
		}
	}

	buf.WriteString("      }\n")
	buf.WriteString("    }\n")
	buf.WriteString("  }\n")
	buf.WriteString("}\n")

	return buf.String()
}

// GenerateDarkModeCSS generates CSS with dark mode media query.
// Uses lightPalette for default and darkPalette for prefers-color-scheme: dark.
func GenerateDarkModeCSS(lightPalette, darkPalette *Palette) string {
	var buf bytes.Buffer

	format := DefaultCSSFormat()

	// Light mode (default)
	buf.WriteString("/* Light mode (default) */\n")
	buf.WriteString(lightPalette.GenerateCSSWithFormat(format))
	buf.WriteString("\n")

	// Dark mode override
	buf.WriteString("/* Dark mode */\n")
	buf.WriteString("@media (prefers-color-scheme: dark) {\n")

	// Generate dark mode CSS with indent
	darkCSS := darkPalette.GenerateCSSWithFormat(format)
	// Add extra indent to each line
	lines := strings.Split(darkCSS, "\n")
	for _, line := range lines {
		if line != "" {
			buf.WriteString("  " + line + "\n")
		}
	}
	buf.WriteString("}\n")

	return buf.String()
}
