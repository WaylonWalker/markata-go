// Package aesthetic provides design token management for markata-go themes.
//
// # Overview
//
// Aesthetics define non-color design tokens including:
//   - Radius tokens: Border radius values for different UI elements
//   - Spacing tokens: Spacing scale for consistent layout
//   - Border tokens: Border width and style definitions
//   - Shadow tokens: Box shadow definitions for elevation
//   - Typography tokens: Font families, sizes, and line heights
//
// # Aesthetic Discovery
//
// Aesthetics are discovered in order:
//  1. Project local: ./aesthetics/{name}.toml
//  2. User config: ~/.config/markata-go/aesthetics/{name}.toml
//  3. Built-in: embedded aesthetics
//
// # Usage
//
//	// Load an aesthetic by name
//	a, err := aesthetic.LoadAesthetic("modern")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Generate CSS custom properties
//	css := a.GenerateCSS()
//
//	// List available aesthetics
//	names := aesthetic.ListAesthetics()
//
// # Relationship to Palettes
//
// Aesthetics complement palettes: palettes define colors, aesthetics define
// everything else. Together they form a complete theme system.
package aesthetic
