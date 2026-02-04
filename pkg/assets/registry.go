package assets

// Asset represents an external CDN asset that can be self-hosted.
type Asset struct {
	// Name is a unique identifier for the asset (e.g., "glightbox-js")
	Name string

	// URL is the full CDN URL for the asset
	URL string

	// LocalPath is the path relative to the vendor output directory
	// e.g., "glightbox/glightbox.min.js"
	LocalPath string

	// Integrity is the SRI hash (e.g., "sha384-...")
	// Empty string means no integrity verification
	Integrity string

	// Version is the asset version
	Version string

	// Type is the asset type: "js", "css", or "other"
	Type string
}

// assetRegistry holds all known CDN assets.
var assetRegistry = []Asset{
	// GLightbox - image lightbox
	{
		Name:      "glightbox-css",
		URL:       "https://cdn.jsdelivr.net/npm/glightbox@3.3.0/dist/css/glightbox.min.css",
		LocalPath: "glightbox/glightbox.min.css",
		Integrity: "", // SRI hash can be added later
		Version:   "3.3.0",
		Type:      "css",
	},
	{
		Name:      "glightbox-js",
		URL:       "https://cdn.jsdelivr.net/npm/glightbox@3.3.0/dist/js/glightbox.min.js",
		LocalPath: "glightbox/glightbox.min.js",
		Integrity: "", // SRI hash can be added later
		Version:   "3.3.0",
		Type:      "js",
	},

	// HTMX - hypermedia framework
	{
		Name:      "htmx",
		URL:       "https://unpkg.com/htmx.org@1.9.10",
		LocalPath: "htmx/htmx.min.js",
		Integrity: "", // SRI hash can be added later
		Version:   "1.9.10",
		Type:      "js",
	},

	// Mermaid - diagram/chart library
	{
		Name:      "mermaid",
		URL:       "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs",
		LocalPath: "mermaid/mermaid.esm.min.mjs",
		Integrity: "", // ES modules don't typically use SRI
		Version:   "10",
		Type:      "js",
	},

	// Chart.js - chart library
	{
		Name:      "chartjs",
		URL:       "https://cdn.jsdelivr.net/npm/chart.js",
		LocalPath: "chartjs/chart.min.js",
		Integrity: "", // Version can vary
		Version:   "latest",
		Type:      "js",
	},

	// Cal-Heatmap - calendar heatmap (contribution graph)
	{
		Name:      "cal-heatmap-css",
		URL:       "https://unpkg.com/cal-heatmap/dist/cal-heatmap.css",
		LocalPath: "cal-heatmap/cal-heatmap.css",
		Integrity: "",
		Version:   "latest",
		Type:      "css",
	},
	{
		Name:      "cal-heatmap-js",
		URL:       "https://unpkg.com/cal-heatmap/dist/cal-heatmap.min.js",
		LocalPath: "cal-heatmap/cal-heatmap.min.js",
		Integrity: "",
		Version:   "latest",
		Type:      "js",
	},
	{
		Name:      "cal-heatmap-tooltip",
		URL:       "https://unpkg.com/cal-heatmap/dist/plugins/Tooltip.min.js",
		LocalPath: "cal-heatmap/plugins/Tooltip.min.js",
		Integrity: "",
		Version:   "latest",
		Type:      "js",
	},

	// D3.js - data visualization library (dependency of Cal-Heatmap)
	{
		Name:      "d3",
		URL:       "https://d3js.org/d3.v7.min.js",
		LocalPath: "d3/d3.v7.min.js",
		Integrity: "",
		Version:   "7",
		Type:      "js",
	},

	// Popper.js - tooltip positioning (dependency of Cal-Heatmap Tooltip)
	{
		Name:      "popper",
		URL:       "https://unpkg.com/@popperjs/core@2",
		LocalPath: "popper/popper.min.js",
		Integrity: "",
		Version:   "2",
		Type:      "js",
	},
}

// Registry returns a copy of all registered assets.
func Registry() []Asset {
	result := make([]Asset, len(assetRegistry))
	copy(result, assetRegistry)
	return result
}

// GetAsset returns an asset by name, or nil if not found.
func GetAsset(name string) *Asset {
	for i := range assetRegistry {
		if assetRegistry[i].Name == name {
			asset := assetRegistry[i]
			return &asset
		}
	}
	return nil
}

// GetAssetsByType returns all assets of a given type.
func GetAssetsByType(assetType string) []Asset {
	var result []Asset
	for _, asset := range assetRegistry {
		if asset.Type == assetType {
			result = append(result, asset)
		}
	}
	return result
}

// AssetGroups returns assets grouped by their library name.
// For example: {"glightbox": [...], "htmx": [...], ...}
func AssetGroups() map[string][]Asset {
	groups := make(map[string][]Asset)
	for _, asset := range assetRegistry {
		// Extract library name from LocalPath (first directory)
		libName := asset.LocalPath
		for i, c := range asset.LocalPath {
			if c == '/' {
				libName = asset.LocalPath[:i]
				break
			}
		}
		groups[libName] = append(groups[libName], asset)
	}
	return groups
}

// AssetNames returns the names of all registered assets.
func AssetNames() []string {
	names := make([]string, len(assetRegistry))
	for i, asset := range assetRegistry {
		names[i] = asset.Name
	}
	return names
}
