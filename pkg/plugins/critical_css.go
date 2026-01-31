// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/criticalcss"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// CriticalCSSPlugin extracts critical CSS and inlines it in HTML files.
// It runs during the Write stage after publish_html to process generated HTML.
//
// Critical CSS optimization improves First Contentful Paint (FCP) by:
// 1. Extracting CSS needed for above-the-fold content
// 2. Inlining critical CSS directly in the HTML <head>
// 3. Async loading non-critical CSS via link rel="preload"
//
// This typically improves FCP by 200-800ms by eliminating render-blocking CSS.
type CriticalCSSPlugin struct {
	config    models.CriticalCSSConfig
	extractor *criticalcss.Extractor
}

// NewCriticalCSSPlugin creates a new CriticalCSSPlugin.
func NewCriticalCSSPlugin() *CriticalCSSPlugin {
	return &CriticalCSSPlugin{}
}

// Name returns the unique name of the plugin.
func (p *CriticalCSSPlugin) Name() string {
	return "critical_css"
}

// Configure loads the critical CSS configuration from the lifecycle manager.
func (p *CriticalCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Extract CriticalCSSConfig from Extra
	if config.Extra != nil {
		if cfg, ok := config.Extra["critical_css"].(models.CriticalCSSConfig); ok {
			p.config = cfg
		}
	}

	// Use defaults if not configured
	if p.config.ViewportWidth == 0 {
		p.config.ViewportWidth = 1300
	}
	if p.config.ViewportHeight == 0 {
		p.config.ViewportHeight = 900
	}
	if p.config.InlineThreshold == 0 {
		p.config.InlineThreshold = 50000
	}

	// Initialize extractor
	p.extractor = criticalcss.NewExtractor().
		WithMinify(p.config.IsMinify()).
		WithSelectors(p.config.ExtraSelectors).
		WithExcludeSelectors(p.config.ExcludeSelectors)

	return nil
}

// Write processes HTML files to inline critical CSS.
func (p *CriticalCSSPlugin) Write(m *lifecycle.Manager) error {
	// Skip if not enabled
	if !p.config.IsEnabled() {
		return nil
	}

	config := m.Config()
	outputDir := config.OutputDir

	log.Printf("[critical_css] Processing HTML files in %s", outputDir)

	// Load all CSS files from the output directory
	cssContent, err := p.loadCSSFiles(outputDir)
	if err != nil {
		return fmt.Errorf("loading CSS files: %w", err)
	}

	if len(cssContent) == 0 {
		log.Printf("[critical_css] No CSS files found, skipping")
		return nil
	}

	// Extract critical CSS once (same for all pages with this approach)
	result, err := p.extractor.ExtractMultiple(cssContent)
	if err != nil {
		return fmt.Errorf("extracting critical CSS: %w", err)
	}

	log.Printf("[critical_css] Extracted %d bytes critical CSS (%.1f%% of %d total)",
		result.CriticalSize, float64(result.CriticalSize)/float64(result.TotalSize)*100, result.TotalSize)

	// Check if critical CSS exceeds threshold
	if result.CriticalSize > p.config.InlineThreshold {
		log.Printf("[critical_css] Critical CSS (%d bytes) exceeds threshold (%d bytes), skipping inline",
			result.CriticalSize, p.config.InlineThreshold)
		return nil
	}

	// Process all HTML files
	return p.processHTMLFiles(outputDir, result.Critical)
}

// loadCSSFiles loads all CSS files from the output directory's css folder.
func (p *CriticalCSSPlugin) loadCSSFiles(outputDir string) (map[string]string, error) {
	cssDir := filepath.Join(outputDir, "css")
	cssContent := make(map[string]string)

	// Check if css directory exists
	if _, err := os.Stat(cssDir); os.IsNotExist(err) {
		return cssContent, nil
	}

	// Read all CSS files
	entries, err := os.ReadDir(cssDir)
	if err != nil {
		return nil, fmt.Errorf("reading css directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}

		path := filepath.Join(cssDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[critical_css] Warning: could not read %s: %v", path, err)
			continue
		}

		cssContent[entry.Name()] = string(content)
	}

	return cssContent, nil
}

// processHTMLFiles walks the output directory and processes all HTML files.
func (p *CriticalCSSPlugin) processHTMLFiles(outputDir, criticalCSS string) error {
	processedCount := 0

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-HTML files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".html") {
			return nil
		}

		// Read HTML file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Process the HTML
		modified, changed := p.processHTML(string(content), criticalCSS)
		if !changed {
			return nil
		}

		// Write back
		//nolint:gosec // G306: HTML output files need 0644 for web serving
		if err := os.WriteFile(path, []byte(modified), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		processedCount++
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[critical_css] Processed %d HTML files", processedCount)
	return nil
}

// processHTML modifies an HTML document to inline critical CSS and async load the rest.
func (p *CriticalCSSPlugin) processHTML(html, criticalCSS string) (string, bool) {
	// Skip if already processed (has critical-css id)
	if strings.Contains(html, `id="critical-css"`) {
		return html, false
	}

	// Skip if no CSS link tags found
	if !strings.Contains(html, `rel="stylesheet"`) {
		return html, false
	}

	var buf bytes.Buffer
	modified := false

	// Find all CSS link tags
	linkRe := regexp.MustCompile(`<link\s+[^>]*rel=["']stylesheet["'][^>]*>`)

	// Find the position to insert critical CSS (before first stylesheet link)
	matches := linkRe.FindAllStringIndex(html, -1)
	if len(matches) == 0 {
		return html, false
	}

	// Build the critical CSS style tag
	criticalTag := fmt.Sprintf(`<style id="critical-css">%s</style>`, criticalCSS)

	// Track position as we process
	lastEnd := 0

	// Insert critical CSS before the first stylesheet link
	firstMatch := matches[0]
	buf.WriteString(html[:firstMatch[0]])
	buf.WriteString(criticalTag)
	buf.WriteString("\n  ")
	lastEnd = firstMatch[0]
	modified = true

	// Process each stylesheet link
	for _, match := range matches {
		// Write content before this match (if not already written)
		if match[0] > lastEnd {
			buf.WriteString(html[lastEnd:match[0]])
		}

		linkTag := html[match[0]:match[1]]

		// Convert to preload if configured
		if p.config.IsPreloadNonCritical() {
			preloadTag := p.convertToPreload(linkTag)
			buf.WriteString(preloadTag)
		} else {
			// Keep original link tag
			buf.WriteString(linkTag)
		}

		lastEnd = match[1]
	}

	// Write remaining content
	buf.WriteString(html[lastEnd:])

	return buf.String(), modified
}

// convertToPreload converts a stylesheet link to a preload link with async loading.
func (p *CriticalCSSPlugin) convertToPreload(linkTag string) string {
	// Extract href from the link tag
	hrefRe := regexp.MustCompile(`href=["']([^"']+)["']`)
	hrefMatch := hrefRe.FindStringSubmatch(linkTag)
	if len(hrefMatch) < 2 {
		return linkTag // Can't parse, return as-is
	}
	href := hrefMatch[1]

	// Build preload link with onload handler
	// The onload trick loads the stylesheet asynchronously without blocking render
	preloadLink := fmt.Sprintf(
		`<link rel="preload" href="%s" as="style" onload="this.onload=null;this.rel='stylesheet'">`,
		href,
	)

	// Add noscript fallback for browsers without JavaScript
	noscriptFallback := fmt.Sprintf(
		`<noscript><link rel="stylesheet" href="%s"></noscript>`,
		href,
	)

	return preloadLink + "\n  " + noscriptFallback
}

// Priority returns the plugin priority for each stage.
// For Write stage, runs after publish_html to process generated HTML.
func (p *CriticalCSSPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		// Run after publish_html (which has default priority)
		// and after static_assets and palette_css
		return lifecycle.PriorityLate + 50
	}
	return lifecycle.PriorityDefault
}

// Ensure CriticalCSSPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*CriticalCSSPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*CriticalCSSPlugin)(nil)
	_ lifecycle.WritePlugin     = (*CriticalCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*CriticalCSSPlugin)(nil)
)
