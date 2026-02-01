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

	// Step 1: Find and scan HTML files
	used, err := scanHTMLFilesForSelectors(outputDir, m.Concurrency(), verbose)
	if err != nil {
		return err
	}
	if used == nil {
		return nil // No HTML files found
	}

	// Step 2: Find CSS files
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

	// Step 3: Build purge options
	opts := buildPurgeOptions(purgeConfig, verbose)

	// Step 4: Process CSS files
	stats := processCSSFiles(cssFiles, outputDir, used, opts, purgeConfig, verbose)

	// Step 5: Report summary
	reportPurgeSummary(stats, purgeConfig, verbose)

	return nil
}

// purgeProcessingStats holds statistics from CSS processing.
type purgeProcessingStats struct {
	totalOriginal  int
	totalPurged    int
	filesProcessed int
	filesSkipped   int
}

// scanHTMLFilesForSelectors finds and scans HTML files for used selectors.
func scanHTMLFilesForSelectors(outputDir string, concurrency int, verbose bool) (*csspurge.UsedSelectors, error) {
	htmlFiles, err := findHTMLFiles(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find HTML files: %w", err)
	}

	if len(htmlFiles) == 0 {
		if verbose {
			fmt.Printf("[css_purge] No HTML files found, skipping\n")
		}
		return nil, nil
	}

	if verbose {
		fmt.Printf("[css_purge] Found %d HTML files to analyze\n", len(htmlFiles))
	}

	used := scanHTMLFilesConcurrently(htmlFiles, concurrency)

	if verbose {
		fmt.Printf("[css_purge] Found %d classes, %d IDs, %d elements, %d attributes\n",
			len(used.Classes), len(used.IDs), len(used.Elements), len(used.Attributes))
	}

	return used, nil
}

// buildPurgeOptions creates PurgeOptions from configuration.
func buildPurgeOptions(purgeConfig models.CSSPurgeConfig, verbose bool) csspurge.PurgeOptions {
	preserve := purgeConfig.Preserve
	if len(preserve) == 0 {
		preserve = csspurge.DefaultPreservePatterns()
	}

	return csspurge.PurgeOptions{
		Preserve: preserve,
		Verbose:  verbose,
	}
}

// processCSSFiles processes each CSS file and returns statistics.
func processCSSFiles(cssFiles []string, outputDir string, used *csspurge.UsedSelectors, opts csspurge.PurgeOptions, purgeConfig models.CSSPurgeConfig, verbose bool) purgeProcessingStats {
	var stats purgeProcessingStats

	for _, cssFile := range cssFiles {
		relPath, err := filepath.Rel(outputDir, cssFile)
		if err != nil {
			relPath = cssFile
		}

		if shouldSkipCSSFile(relPath, purgeConfig.SkipFiles) {
			if verbose {
				fmt.Printf("[css_purge] Skipping %s (matches skip pattern)\n", relPath)
			}
			stats.filesSkipped++
			continue
		}

		processSingleCSSFile(cssFile, relPath, used, opts, &stats, verbose)
	}

	return stats
}

// processSingleCSSFile processes a single CSS file.
func processSingleCSSFile(cssFile, relPath string, used *csspurge.UsedSelectors, opts csspurge.PurgeOptions, stats *purgeProcessingStats, verbose bool) {
	content, err := os.ReadFile(cssFile)
	if err != nil {
		fmt.Printf("[css_purge] WARNING: failed to read %s: %v\n", relPath, err)
		return
	}

	purged, purgeStats := csspurge.PurgeCSS(string(content), used, opts)

	stats.totalOriginal += purgeStats.OriginalSize
	stats.totalPurged += purgeStats.PurgedSize

	if purgeStats.RemovedRules > 0 {
		if err := os.WriteFile(cssFile, []byte(purged), 0o644); err != nil {
			fmt.Printf("[css_purge] WARNING: failed to write %s: %v\n", relPath, err)
			return
		}

		if verbose {
			fmt.Printf("[css_purge] %s: removed %d/%d rules (%.1f%% reduction, %d -> %d bytes)\n",
				relPath, purgeStats.RemovedRules, purgeStats.TotalRules,
				purgeStats.SavingsPercent(), purgeStats.OriginalSize, purgeStats.PurgedSize)
		}
	} else if verbose {
		fmt.Printf("[css_purge] %s: all %d rules are used\n", relPath, purgeStats.TotalRules)
	}

	stats.filesProcessed++
}

// reportPurgeSummary reports the purging summary.
func reportPurgeSummary(stats purgeProcessingStats, purgeConfig models.CSSPurgeConfig, verbose bool) {
	if stats.filesProcessed > 0 {
		savings := float64(stats.totalOriginal-stats.totalPurged) / float64(stats.totalOriginal) * 100
		fmt.Printf("[css_purge] Processed %d CSS files: %d -> %d bytes (%.1f%% reduction)\n",
			stats.filesProcessed, stats.totalOriginal, stats.totalPurged, savings)

		if purgeConfig.WarningThreshold > 0 && int(savings) > purgeConfig.WarningThreshold {
			fmt.Printf("[css_purge] WARNING: Removed %.1f%% of CSS (threshold: %d%%). "+
				"This might indicate overly aggressive purging. "+
				"Consider adding patterns to 'preserve' config.\n",
				savings, purgeConfig.WarningThreshold)
		}
	}

	if stats.filesSkipped > 0 && verbose {
		fmt.Printf("[css_purge] Skipped %d CSS files\n", stats.filesSkipped)
	}
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
func scanHTMLFilesConcurrently(files []string, concurrency int) *csspurge.UsedSelectors {
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

	return combined
}

// shouldSkipCSSFile checks if a CSS file matches any skip pattern.
func shouldSkipCSSFile(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}
		// Also check just the filename
		matched, err = filepath.Match(pattern, filepath.Base(relPath))
		if err == nil && matched {
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
