// Package criticalcss provides CSS extraction utilities for critical CSS optimization.
//
// # Critical CSS Optimization
//
// Critical CSS refers to the CSS required to render the above-the-fold content of a page.
// By inlining this CSS directly in the HTML and async loading the rest, we can significantly
// improve First Contentful Paint (FCP) metrics.
//
// # Selector-Based Extraction
//
// This package uses a selector-based approach to identify critical CSS rules.
// It parses CSS files and extracts rules that match a predefined set of selectors
// known to appear above the fold in most layouts.
//
// Critical selectors include:
//   - Base elements: html, body, main, article, header, nav, footer
//   - Typography: h1, h2, h3, p, a, span
//   - Common classes: .site-header, .container, .content-width, .post, .feed
//   - Media elements: img, video, figure
//   - Layout elements: .page-wrapper, .main-content
//
// # Usage
//
// The Extractor type provides the main API:
//
//	ext := criticalcss.NewExtractor()
//	critical, remaining, err := ext.Extract(cssContent)
//
// The critical CSS can then be inlined in the HTML <head>, while the remaining
// CSS is loaded asynchronously via <link rel="preload">.
//
// # Performance Benefits
//
// Expected improvements:
//   - 200-800ms reduction in First Contentful Paint (FCP)
//   - Elimination of render-blocking CSS resources
//   - Better Core Web Vitals scores
package criticalcss
