// Package resourcehints provides detection and generation of resource hints
// (preconnect, dns-prefetch, preload, prefetch) for web performance optimization.
//
// # Resource Hints Overview
//
// Resource hints help browsers prepare for external resources before they're needed:
//
//   - preconnect: Establishes early connection to origin (DNS + TCP + TLS)
//   - dns-prefetch: Performs DNS lookup in advance
//   - preload: Fetches critical resources early
//   - prefetch: Fetches resources for future navigation
//
// # Usage
//
// The package provides two main capabilities:
//
//  1. Detection - Scan HTML/CSS for external domains
//  2. Generation - Create link tags for detected or configured hints
//
// Example:
//
//	detector := resourcehints.NewDetector()
//	domains := detector.DetectExternalDomains(htmlContent)
//	hints := detector.SuggestHints(domains)
//
//	generator := resourcehints.NewGenerator()
//	tags := generator.GenerateHintTags(hints)
//
// # Known Domains
//
// The package includes built-in knowledge of common CDNs and services:
//
//   - Google Fonts (fonts.googleapis.com, fonts.gstatic.com)
//   - CDNs (cdn.jsdelivr.net, unpkg.com, cdnjs.cloudflare.com)
//   - Analytics (www.google-analytics.com, www.googletagmanager.com)
//
// # Performance Impact
//
// Typical improvements from resource hints:
//
//   - 100-500ms reduction in resource loading time
//   - 3-8 points improvement in Lighthouse performance score
package resourcehints
