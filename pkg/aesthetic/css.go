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

// GenerateCSSWithFormat generates CSS with the specified format options.
func (a *Aesthetic) GenerateCSSWithFormat(format CSSFormat) string {
	var sb strings.Builder

	// Determine formatting strings
	indent := "  "
	newline := "\n"
	if format.Minify {
		indent = ""
		newline = ""
	}

	// Write header comment (not in minified)
	if !format.Minify {
		sb.WriteString(fmt.Sprintf("/* Aesthetic: %s */\n", a.Name))
		if a.Description != "" {
			sb.WriteString(fmt.Sprintf("/* %s */\n", a.Description))
		}
	}

	// Start :root block
	sb.WriteString(":root {")
	sb.WriteString(newline)

	sections := []struct {
		name    string
		include bool
		vars    map[string]string
	}{}

	// Collect sections based on format options
	// Check for legacy flat maps first (used by css_test.go), then tokens
	typography := a.getTypographyMap()
	spacing := a.getSpacingMap()
	borders := a.getBordersMap()
	shadows := a.getShadowsMap()

	if format.IncludeTypography && len(typography) > 0 {
		sections = append(sections, struct {
			name    string
			include bool
			vars    map[string]string
		}{"Typography", true, typography})
	}

	if format.IncludeSpacing && len(spacing) > 0 {
		sections = append(sections, struct {
			name    string
			include bool
			vars    map[string]string
		}{"Spacing", true, spacing})
	}

	if format.IncludeBorders && len(borders) > 0 {
		sections = append(sections, struct {
			name    string
			include bool
			vars    map[string]string
		}{"Borders", true, borders})
	}

	if format.IncludeShadows && len(shadows) > 0 {
		sections = append(sections, struct {
			name    string
			include bool
			vars    map[string]string
		}{"Shadows", true, shadows})
	}

	// Write each section
	for i, section := range sections {
		if !format.Minify && i > 0 {
			sb.WriteString(newline)
		}

		// Section comment (not in minified)
		if !format.Minify {
			sb.WriteString(fmt.Sprintf("%s/* %s */\n", indent, section.name))
		}

		// Sort keys for consistent output
		keys := make([]string, 0, len(section.vars))
		for k := range section.vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Write CSS variables
		for _, key := range keys {
			value := section.vars[key]
			varName := formatVarName(key, format.Prefix)
			if format.Minify {
				sb.WriteString(fmt.Sprintf("%s:%s;", varName, value))
			} else {
				sb.WriteString(fmt.Sprintf("%s%s: %s;\n", indent, varName, value))
			}
		}
	}

	// Close :root block
	sb.WriteString("}")
	if !format.Minify {
		sb.WriteString("\n")
	}

	return sb.String()
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
