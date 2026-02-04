// Package assets provides functionality for downloading and self-hosting external CDN assets.
//
// # Overview
//
// This package manages external JavaScript and CSS dependencies that are typically
// loaded from CDNs (like jsdelivr, unpkg, etc.). It provides:
//
//   - A registry of known CDN assets with integrity hashes
//   - Download functionality with caching
//   - Integrity verification using SHA-256/384/512 hashes
//   - Copy to output directory for self-hosting
//
// # Asset Registry
//
// The registry defines all supported external assets with their CDN URLs,
// local paths, and optional SRI (Subresource Integrity) hashes:
//
//	asset := assets.GetAsset("glightbox-js")
//	// URL: https://cdn.jsdelivr.net/npm/glightbox@3.3.0/dist/js/glightbox.min.js
//	// LocalPath: glightbox/glightbox.min.js
//
// # Configuration
//
// Asset handling is controlled by the Assets config in markata-go.toml:
//
//	[markata-go.assets]
//	mode = "self-hosted"  # "cdn", "self-hosted", or "auto"
//	cache_dir = ".markata/assets-cache"
//	verify_integrity = true
//
// # CLI Commands
//
// The assets subcommand provides management tools:
//
//	markata-go assets download   # Download all CDN assets to cache
//	markata-go assets list       # Show status of all assets
//	markata-go assets clean      # Remove cached assets
package assets
