// Package palettes provides color palette management for markata-go themes.
//
// # Overview
//
// Palettes define color schemes using a three-layer architecture:
//   - Raw colors: Pure hex values with no semantic meaning
//   - Semantic colors: Role-based mappings (text-primary, bg-surface, etc.)
//   - Component colors: Fine-grained component styling (code-keyword, button-bg)
//
// # Palette Discovery
//
// Palettes are discovered in order:
//  1. Project local: ./palettes/{name}.toml
//  2. User config: ~/.config/markata-go/palettes/{name}.toml
//  3. Built-in: embedded palettes
//
// # Usage
//
//	// Load a palette by name
//	p, err := palettes.Load("catppuccin-mocha")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get resolved colors
//	textColor := p.Resolve("text-primary")  // Returns "#cdd6f4"
//
//	// Generate CSS
//	css := p.GenerateCSS()
//
//	// Check contrast ratios
//	results := p.CheckContrast()
//	for _, r := range results {
//	    if !r.Passed {
//	        fmt.Printf("FAIL: %s on %s (%.2f:1)\n", r.Foreground, r.Background, r.Ratio)
//	    }
//	}
//
// # WCAG Compliance
//
// The package includes WCAG 2.1 contrast ratio validation:
//   - AA: 4.5:1 for normal text, 3:1 for large text/UI
//   - AAA: 7:1 for normal text, 4.5:1 for large text
package palettes
