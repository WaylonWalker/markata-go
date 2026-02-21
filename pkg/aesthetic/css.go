package aesthetic

import (
	"fmt"
	"sort"
	"strings"
)

// CSSFormat controls how CSS is generated.
type CSSFormat struct {
	IncludeTypography bool   // Include typography tokens
	IncludeSpacing    bool   // Include spacing tokens
	IncludeBorders    bool   // Include border tokens
	IncludeShadows    bool   // Include shadow tokens
	IncludeEffects    bool   // Include effects tokens
	Minify            bool   // Produce minified output
	Prefix            string // Custom prefix for CSS variables (e.g., "theme" -> "--theme-radius-sm")
}

// DefaultCSSFormat returns the default CSS format options.
func DefaultCSSFormat() CSSFormat {
	return CSSFormat{
		IncludeTypography: true,
		IncludeSpacing:    true,
		IncludeBorders:    true,
		IncludeShadows:    true,
		IncludeEffects:    true,
		Minify:            false,
		Prefix:            "",
	}
}

// GenerateCSS generates CSS custom properties for the aesthetic.
func (a *Aesthetic) GenerateCSS() string {
	return a.GenerateCSSWithFormat(DefaultCSSFormat())
}

// GenerateCSSMinified generates minified CSS custom properties.
func (a *Aesthetic) GenerateCSSMinified() string {
	format := DefaultCSSFormat()
	format.Minify = true
	return a.GenerateCSSWithFormat(format)
}

// cssSection represents a group of CSS variables.
type cssSection struct {
	name string
	vars map[string]string
}

// GenerateCSSWithFormat generates CSS with the specified format options.
func (a *Aesthetic) GenerateCSSWithFormat(format CSSFormat) string {
	var sb strings.Builder

	// Write header comment (not in minified)
	if !format.Minify {
		fmt.Fprintf(&sb, "/* Aesthetic: %s */\n", a.Name)
		if a.Description != "" {
			fmt.Fprintf(&sb, "/* %s */\n", a.Description)
		}
	}

	// Start :root block
	sb.WriteString(":root {")
	if !format.Minify {
		sb.WriteString("\n")
	}

	// Collect sections based on format options
	sections := a.collectSections(format)

	// Write each section
	for i, section := range sections {
		writeSection(&sb, section, format, i == 0)
	}

	// Close :root block
	sb.WriteString("}")
	if !format.Minify {
		sb.WriteString("\n")
	}

	return sb.String()
}

func (a *Aesthetic) collectSections(format CSSFormat) []cssSection {
	var sections []cssSection

	if format.IncludeTypography {
		if m := a.getTypographyMap(); len(m) > 0 {
			sections = append(sections, cssSection{"Typography", m})
		}
	}
	if format.IncludeSpacing {
		if m := a.getSpacingMap(); len(m) > 0 {
			sections = append(sections, cssSection{"Spacing", m})
		}
	}
	if format.IncludeBorders {
		if m := a.getBordersMap(); len(m) > 0 {
			sections = append(sections, cssSection{"Borders", m})
		}
	}
	if format.IncludeShadows {
		if m := a.getShadowsMap(); len(m) > 0 {
			sections = append(sections, cssSection{"Shadows", m})
		}
	}
	if format.IncludeEffects {
		if m := a.getEffectsMap(); len(m) > 0 {
			sections = append(sections, cssSection{"Effects", m})
		}
	}

	return sections
}

func writeSection(sb *strings.Builder, section cssSection, format CSSFormat, isFirst bool) {
	indent := "  "
	if format.Minify {
		indent = ""
	}

	if !format.Minify && !isFirst {
		sb.WriteString("\n")
	}

	if !format.Minify {
		fmt.Fprintf(sb, "%s/* %s */\n", indent, section.name)
	}

	keys := make([]string, 0, len(section.vars))
	for k := range section.vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := section.vars[key]
		varName := formatVarName(key, format.Prefix)
		if format.Minify {
			fmt.Fprintf(sb, "%s:%s;", varName, value)
		} else {
			fmt.Fprintf(sb, "%s%s: %s;\n", indent, varName, value)
		}
	}
}

// getTypographyMap returns typography tokens as a map for CSS generation.
func (a *Aesthetic) getTypographyMap() map[string]string {
	// Check legacy flat map first
	if len(a.Typography) > 0 {
		return a.Typography
	}
	// Fall back to tokens
	if len(a.Tokens.Typography) > 0 {
		result := make(map[string]string)
		for k, v := range a.Tokens.Typography {
			// Convert token names to CSS var names
			cssKey := tokenToCSSName(k, "")
			result[cssKey] = v
		}
		return result
	}
	return nil
}

// getSpacingMap returns spacing tokens as a map for CSS generation.
func (a *Aesthetic) getSpacingMap() map[string]string {
	// Check legacy flat map first
	if len(a.Spacing) > 0 {
		return a.Spacing
	}
	// For tokens, we generate spacing scale as a single variable
	if a.Tokens.Spacing != nil && a.Tokens.Spacing.Scale != 0 {
		return map[string]string{
			"spacing-scale": fmt.Sprintf("%.2f", a.Tokens.Spacing.Scale),
		}
	}
	return nil
}

// getBordersMap returns border tokens as a map for CSS generation.
func (a *Aesthetic) getBordersMap() map[string]string {
	// Check legacy flat map first
	if len(a.Borders) > 0 {
		return a.Borders
	}
	// Combine radius and border tokens
	result := make(map[string]string)
	for k, v := range a.Tokens.Radius {
		cssKey := tokenToCSSName(k, "radius")
		result[cssKey] = v
	}
	for k, v := range a.Tokens.Border {
		cssKey := tokenToCSSName(k, "border")
		result[cssKey] = v
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// getShadowsMap returns shadow tokens as a map for CSS generation.
func (a *Aesthetic) getShadowsMap() map[string]string {
	// Check legacy flat map first
	if len(a.Shadows) > 0 {
		return a.Shadows
	}
	// Use shadow tokens
	if len(a.Tokens.Shadow) > 0 {
		result := make(map[string]string)
		for k, v := range a.Tokens.Shadow {
			cssKey := tokenToCSSName(k, "shadow")
			result[cssKey] = v
		}
		return result
	}
	return nil
}

// getEffectsMap returns effects tokens as a map for CSS generation.
func (a *Aesthetic) getEffectsMap() map[string]string {
	// Use effects tokens
	if len(a.Tokens.Effects) > 0 {
		result := make(map[string]string)
		for k, v := range a.Tokens.Effects {
			cssKey := tokenToCSSName(k, "effect")
			result[cssKey] = v
		}
		return result
	}
	return nil
}

// tokenToCSSName converts a token key to a CSS variable name.
// e.g., "width_normal" with prefix "border" -> "border-width-normal"
func tokenToCSSName(key, prefix string) string {
	// Replace underscores with hyphens
	name := strings.ReplaceAll(key, "_", "-")
	if prefix != "" {
		return prefix + "-" + name
	}
	return name
}

// formatVarName formats a CSS variable name with optional prefix.
func formatVarName(name, prefix string) string {
	if prefix != "" {
		return "--" + prefix + "-" + name
	}
	return "--" + name
}
