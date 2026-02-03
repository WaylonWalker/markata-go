// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// CSSMinifyPlugin minifies CSS files to reduce file sizes and improve
// Lighthouse performance scores. It runs during the Write stage, after
// all other CSS-generating plugins (palette_css, chroma_css, css_bundle)
// have completed.
//
// The plugin:
// - Discovers CSS files in the output directory
// - Minifies each file using tdewolff/minify
// - Preserves specified comments (e.g., copyright notices)
// - Skips excluded files
// - Reports size reduction statistics
type CSSMinifyPlugin struct {
	config   models.CSSMinifyConfig
	minifier *minify.M
	exclude  map[string]bool // excluded file patterns for fast lookup
}

// NewCSSMinifyPlugin creates a new CSSMinifyPlugin.
func NewCSSMinifyPlugin() *CSSMinifyPlugin {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	return &CSSMinifyPlugin{
		minifier: m,
		exclude:  make(map[string]bool),
	}
}

// Name returns the unique name of the plugin.
func (p *CSSMinifyPlugin) Name() string {
	return "css_minify"
}

// Configure reads the CSS minify configuration from the manager's config.
func (p *CSSMinifyPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Set default config first
	p.config = models.NewCSSMinifyConfig()

	if config.Extra == nil {
		return nil
	}

	// Try to get css_minify config from Extra
	// It may be a models.CSSMinifyConfig or a map[string]interface{} from TOML parsing
	switch v := config.Extra["css_minify"].(type) {
	case models.CSSMinifyConfig:
		p.config = v
	case map[string]interface{}:
		p.config = p.parseConfigFromMap(v)
	}

	// Build exclude map for fast lookups
	for _, pattern := range p.config.Exclude {
		p.exclude[pattern] = true
	}

	return nil
}

// parseConfigFromMap parses CSSMinifyConfig from a raw map (TOML parsing result).
func (p *CSSMinifyPlugin) parseConfigFromMap(m map[string]interface{}) models.CSSMinifyConfig {
	cfg := models.NewCSSMinifyConfig()

	if enabled, ok := m["enabled"].(bool); ok {
		cfg.Enabled = enabled
	}

	// Parse exclude list
	if exclude, ok := m["exclude"].([]interface{}); ok {
		cfg.Exclude = make([]string, 0, len(exclude))
		for _, e := range exclude {
			if s, ok := e.(string); ok {
				cfg.Exclude = append(cfg.Exclude, s)
			}
		}
	}

	// Parse preserve_comments list
	if preserveComments, ok := m["preserve_comments"].([]interface{}); ok {
		cfg.PreserveComments = make([]string, 0, len(preserveComments))
		for _, c := range preserveComments {
			if s, ok := c.(string); ok {
				cfg.PreserveComments = append(cfg.PreserveComments, s)
			}
		}
	}

	return cfg
}

// Write performs CSS minification in the output directory.
func (p *CSSMinifyPlugin) Write(m *lifecycle.Manager) error {
	// Skip if not enabled
	if !p.config.Enabled {
		return nil
	}

	config := m.Config()
	outputDir := config.OutputDir

	log.Printf("[css_minify] Starting CSS minification")

	// Find all CSS files in output directory
	cssFiles, err := p.findCSSFiles(outputDir)
	if err != nil {
		return fmt.Errorf("finding CSS files: %w", err)
	}

	if len(cssFiles) == 0 {
		log.Printf("[css_minify] No CSS files found")
		return nil
	}

	// Track statistics
	var totalOriginal, totalMinified int64
	var filesProcessed, filesSkipped int

	// Process each CSS file
	for _, cssFile := range cssFiles {
		// Check if file should be excluded
		if p.isExcluded(cssFile) {
			log.Printf("[css_minify] Skipping excluded file: %s", filepath.Base(cssFile))
			filesSkipped++
			continue
		}

		original, minifiedSize, err := p.minifyFile(cssFile)
		if err != nil {
			log.Printf("[css_minify] Warning: failed to minify %s: %v", filepath.Base(cssFile), err)
			continue
		}

		totalOriginal += original
		totalMinified += minifiedSize
		filesProcessed++
	}

	// Calculate and log statistics
	if totalOriginal > 0 {
		reduction := float64(totalOriginal-totalMinified) / float64(totalOriginal) * 100
		log.Printf("[css_minify] Completed: %d files processed, %d skipped", filesProcessed, filesSkipped)
		log.Printf("[css_minify] Size reduction: %d -> %d bytes (%.1f%% smaller)",
			totalOriginal, totalMinified, reduction)
	}

	return nil
}

// findCSSFiles recursively finds all CSS files in a directory.
func (p *CSSMinifyPlugin) findCSSFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for CSS files
		if strings.HasSuffix(strings.ToLower(path), ".css") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// isExcluded checks if a file should be excluded from minification.
func (p *CSSMinifyPlugin) isExcluded(filePath string) bool {
	filename := filepath.Base(filePath)

	// Check exact match
	if p.exclude[filename] {
		return true
	}

	// Check pattern match
	for pattern := range p.exclude {
		if strings.ContainsAny(pattern, "*?[") {
			matched, err := filepath.Match(pattern, filename)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// minifyFile minifies a single CSS file in place.
// Returns the original size and minified size.
func (p *CSSMinifyPlugin) minifyFile(filePath string) (original, minified int64, err error) {
	// Read the original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("reading file: %w", err)
	}

	original = int64(len(content))

	// Preserve specified comments
	preservedComments := p.extractPreservedComments(string(content))

	// Minify the content
	var buf bytes.Buffer
	if err := p.minifier.Minify("text/css", &buf, bytes.NewReader(content)); err != nil {
		return original, 0, fmt.Errorf("minifying: %w", err)
	}

	// Prepend preserved comments
	var result bytes.Buffer
	for _, comment := range preservedComments {
		result.WriteString(comment)
		result.WriteByte('\n')
	}
	result.Write(buf.Bytes())

	minified = int64(result.Len())

	// Write the minified content back
	//nolint:gosec // G306: CSS files need 0644 for web serving
	if err := os.WriteFile(filePath, result.Bytes(), 0o644); err != nil {
		return original, 0, fmt.Errorf("writing minified file: %w", err)
	}

	return original, minified, nil
}

// extractPreservedComments extracts comments that match the preserve patterns.
func (p *CSSMinifyPlugin) extractPreservedComments(content string) []string {
	if len(p.config.PreserveComments) == 0 {
		return nil
	}

	var preserved []string

	// Find all CSS comments
	i := 0
	for i < len(content) {
		// Look for comment start
		start := strings.Index(content[i:], "/*")
		if start == -1 {
			break
		}
		start += i

		// Look for comment end
		end := strings.Index(content[start+2:], "*/")
		if end == -1 {
			break
		}
		end += start + 2 + 2 // Include the */

		comment := content[start:end]

		// Check if this comment should be preserved
		for _, pattern := range p.config.PreserveComments {
			if strings.Contains(comment, pattern) {
				preserved = append(preserved, comment)
				break
			}
		}

		i = end
	}

	return preserved
}

// Priority returns the plugin priority for the write stage.
// Should run late (after palette_css, chroma_css, css_bundle) to minify all CSS.
func (p *CSSMinifyPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		// Run very late in Write stage, after CSS generators and bundler
		return lifecycle.PriorityLast
	}
	return lifecycle.PriorityDefault
}

// Ensure CSSMinifyPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*CSSMinifyPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*CSSMinifyPlugin)(nil)
	_ lifecycle.WritePlugin     = (*CSSMinifyPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*CSSMinifyPlugin)(nil)
)
