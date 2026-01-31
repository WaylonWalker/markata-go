// Package csspurge provides CSS analysis and unused rule removal functionality.
//
// # Overview
//
// The csspurge package scans HTML files to identify used CSS selectors and then
// removes unused CSS rules from stylesheets. This helps reduce CSS file sizes
// by eliminating rules that don't match any elements in the generated HTML.
//
// # Usage
//
// The package is typically used through the css_purge plugin, but can be used
// directly for custom CSS optimization workflows:
//
//	// Scan HTML files to find used selectors
//	used := csspurge.NewUsedSelectors()
//	for _, htmlPath := range htmlFiles {
//	    if err := csspurge.ScanHTML(htmlPath, used); err != nil {
//	        log.Printf("warning: %v", err)
//	    }
//	}
//
//	// Purge unused rules from CSS
//	opts := csspurge.PurgeOptions{
//	    Preserve: []string{"js-*", "htmx-*"},
//	}
//	purged, stats := csspurge.PurgeCSS(cssContent, used, opts)
//
// # Preserved Patterns
//
// Some CSS classes are dynamically added by JavaScript and won't appear in
// the static HTML. The package supports glob patterns to preserve these rules:
//
//   - js-* - JavaScript-added classes
//   - htmx-* - HTMX framework classes
//   - pagefind-* - Pagefind search UI classes
//   - glightbox* - GLightbox image viewer classes
//   - active, hidden, loading - Common state classes
//   - dark, light - Theme mode classes
//
// # CSS Parsing
//
// The package uses a regex-based CSS parser that handles:
//
//   - Standard rule blocks: selector { properties }
//   - Media queries: @media (...) { rules }
//   - Keyframes: @keyframes name { ... }
//   - Font-faces: @font-face { ... }
//   - Imports: @import ...
//
// At-rules like @media queries are preserved if any of their nested rules
// are used. @keyframes and @font-face are always preserved.
package csspurge
