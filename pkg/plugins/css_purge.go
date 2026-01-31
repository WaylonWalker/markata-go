package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/csspurge"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// CSSPurgePlugin removes unused CSS rules from stylesheets based on
// which selectors are actually used in the generated HTML files.
//
// This plugin runs in the Cleanup stage after all HTML has been written,
// allowing it to scan the final output and optimize CSS accordingly.
//
// The plugin supports preserving dynamically-added classes (like those
// from JavaScript frameworks) via glob patterns in the configuration.
type CSSPurgePlugin struct{}

// NewCSSPurgePlugin creates a new CSSPurgePlugin.
func NewCSSPurgePlugin() *CSSPurgePlugin {
	return &CSSPurgePlugin{}
}

// Name returns the unique name of the plugin.
func (p *CSSPurgePlugin) Name() string {
	return "css_purge"
}

// Cleanup scans HTML files and removes unused CSS rules.
// This runs after all content has been written to the output directory.
func (p *CSSPurgePlugin) Cleanup(m *lifecycle.Manager) error {
	config := m.Config()
	purgeConfig := getCSSPurgeConfig(config)

	// Skip if disabled
	if !purgeConfig.Enabled {
		return nil
	}

	outputDir := config.OutputDir
	verbose := purgeConfig.Verbose

	if verbose {
		fmt.Printf("[css_purge] Analyzing CSS usage in %s\n", outputDir)
	}

	// Step 1: Find all HTML files
	htmlFiles, err := findHTMLFiles(outputDir)
	if err != nil {
		return fmt.Errorf("failed to find HTML files: %w", err)
	}

	if len(htmlFiles) == 0 {
		if verbose {
			fmt.Printf("[css_purge] No HTML files found, skipping\n")
		}
		return nil
	}

	if verbose {
		fmt.Printf("[css_purge] Found %d HTML files to analyze\n", len(htmlFiles))
	}

	// Step 2: Scan HTML files concurrently to find used selectors
	used, err := scanHTMLFilesConcurrently(htmlFiles, m.Concurrency())
	if err != nil {
		return fmt.Errorf("failed to scan HTML files: %w", err)
	}

	if verbose {
		fmt.Printf("[css_purge] Found %d classes, %d IDs, %d elements, %d attributes\n",
			len(used.Classes), len(used.IDs), len(used.Elements), len(used.Attributes))
	}

	// Step 3: Find all CSS files
	cssFiles, err := findCSSFiles(outputDir)
	if err != nil {
		return fmt.Errorf("failed to find CSS files: %w", err)
	}

	if len(cssFiles) == 0 {
		if verbose {
			fmt.Printf("[css_purge] No CSS files found, skipping\n")
		}
		return nil
	}

	// Step 4: Build preserve patterns
	preserve := purgeConfig.Preserve
	if len(preserve) == 0 {
		preserve = csspurge.DefaultPreservePatterns()
	}

	opts := csspurge.PurgeOptions{
		Preserve: preserve,
		Verbose:  verbose,
	}

	// Step 5: Process each CSS file
	var totalOriginal, totalPurged int
	var filesProcessed, filesSkipped int

	for _, cssFile := range cssFiles {
		// Check if file should be skipped
		relPath, _ := filepath.Rel(outputDir, cssFile)
		if shouldSkipCSSFile(relPath, purgeConfig.SkipFiles) {
			if verbose {
				fmt.Printf("[css_purge] Skipping %s (matches skip pattern)\n", relPath)
			}
			filesSkipped++
			continue
		}

		// Read CSS file
		content, err := os.ReadFile(cssFile)
		if err != nil {
			fmt.Printf("[css_purge] WARNING: failed to read %s: %v\n", relPath, err)
			continue
		}

		// Purge unused CSS
		purged, stats := csspurge.PurgeCSS(string(content), used, opts)

		totalOriginal += stats.OriginalSize
		totalPurged += stats.PurgedSize

		// Write back if anything was removed
		if stats.RemovedRules > 0 {
			if err := os.WriteFile(cssFile, []byte(purged), 0644); err != nil {
				fmt.Printf("[css_purge] WARNING: failed to write %s: %v\n", relPath, err)
				continue
			}

			if verbose {
				fmt.Printf("[css_purge] %s: removed %d/%d rules (%.1f%% reduction, %d -> %d bytes)\n",
					relPath, stats.RemovedRules, stats.TotalRules,
					stats.SavingsPercent(), stats.OriginalSize, stats.PurgedSize)
			}
		} else if verbose {
			fmt.Printf("[css_purge] %s: all %d rules are used\n", relPath, stats.TotalRules)
		}

		filesProcessed++
	}

	// Report summary
	if filesProcessed > 0 {
		savings := float64(totalOriginal-totalPurged) / float64(totalOriginal) * 100
		fmt.Printf("[css_purge] Processed %d CSS files: %d -> %d bytes (%.1f%% reduction)\n",
			filesProcessed, totalOriginal, totalPurged, savings)

		// Check warning threshold
		if purgeConfig.WarningThreshold > 0 && int(savings) > purgeConfig.WarningThreshold {
			fmt.Printf("[css_purge] WARNING: Removed %.1f%% of CSS (threshold: %d%%). "+
				"This might indicate overly aggressive purging. "+
				"Consider adding patterns to 'preserve' config.\n",
				savings, purgeConfig.WarningThreshold)
		}
	}

	if filesSkipped > 0 && verbose {
		fmt.Printf("[css_purge] Skipped %d CSS files\n", filesSkipped)
	}

	return nil
}

// Priority returns the plugin priority for the cleanup stage.
// CSS purge runs before Pagefind (PriorityDefault) since Pagefind indexes
// content but doesn't need CSS, and we want smaller CSS files deployed.
func (p *CSSPurgePlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageCleanup {
		return lifecycle.PriorityDefault - 10 // Before Pagefind
	}
	return lifecycle.PriorityDefault
}

// findHTMLFiles recursively finds all HTML files in a directory.
func findHTMLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".html") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// findCSSFiles recursively finds all CSS files in a directory.
func findCSSFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".css") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// scanHTMLFilesConcurrently scans HTML files using a worker pool.
func scanHTMLFilesConcurrently(files []string, concurrency int) (*csspurge.UsedSelectors, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(files) {
		concurrency = len(files)
	}

	// Use channels for work distribution
	jobs := make(chan string, len(files))
	results := make(chan *csspurge.UsedSelectors, len(files))
	errors := make(chan error, len(files))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				used := csspurge.NewUsedSelectors()
				if err := csspurge.ScanHTML(path, used); err != nil {
					errors <- fmt.Errorf("scanning %s: %w", path, err)
					continue
				}
				results <- used
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		// Log but don't fail - partial analysis is still useful
		fmt.Printf("[css_purge] WARNING: %v\n", err)
	}

	// Merge results
	combined := csspurge.NewUsedSelectors()
	for used := range results {
		combined.Merge(used)
	}

	return combined, nil
}

// shouldSkipCSSFile checks if a CSS file matches any skip pattern.
func shouldSkipCSSFile(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, relPath)
		if matched {
			return true
		}
		// Also check just the filename
		matched, _ = filepath.Match(pattern, filepath.Base(relPath))
		if matched {
			return true
		}
	}
	return false
}

// getCSSPurgeConfig extracts CSSPurgeConfig from lifecycle.Config.Extra.
// Handles both direct struct assignment and raw map from TOML parsing.
func getCSSPurgeConfig(config *lifecycle.Config) models.CSSPurgeConfig {
	if config.Extra == nil {
		return models.NewCSSPurgeConfig()
	}

	// Check if it's already a typed config
	if pc, ok := config.Extra["css_purge"].(models.CSSPurgeConfig); ok {
		return pc
	}

	// Check for raw map from TOML/YAML/JSON parsing
	rawConfig, ok := config.Extra["css_purge"].(map[string]interface{})
	if !ok {
		return models.NewCSSPurgeConfig()
	}

	// Parse the raw map into config
	result := models.NewCSSPurgeConfig()

	if enabled, ok := rawConfig["enabled"].(bool); ok {
		result.Enabled = enabled
	}
	if verbose, ok := rawConfig["verbose"].(bool); ok {
		result.Verbose = verbose
	}
	if threshold, ok := rawConfig["warning_threshold"].(int64); ok {
		result.WarningThreshold = int(threshold)
	}

	// Parse preserve patterns
	if preserve, ok := rawConfig["preserve"].([]interface{}); ok {
		result.Preserve = make([]string, 0, len(preserve))
		for _, p := range preserve {
			if s, ok := p.(string); ok {
				result.Preserve = append(result.Preserve, s)
			}
		}
	}

	// Parse skip_files patterns
	if skipFiles, ok := rawConfig["skip_files"].([]interface{}); ok {
		result.SkipFiles = make([]string, 0, len(skipFiles))
		for _, sf := range skipFiles {
			if s, ok := sf.(string); ok {
				result.SkipFiles = append(result.SkipFiles, s)
			}
		}
	}

	return result
}

// Ensure CSSPurgePlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*CSSPurgePlugin)(nil)
	_ lifecycle.CleanupPlugin  = (*CSSPurgePlugin)(nil)
	_ lifecycle.PriorityPlugin = (*CSSPurgePlugin)(nil)
)
