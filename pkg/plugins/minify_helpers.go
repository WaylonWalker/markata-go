// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"log"
	"path/filepath"
	"strings"
)

// minifyFunc processes a single file and returns original and minified sizes.
type minifyFunc func(path string) (original, minified int64, err error)

// excludeFunc checks if a file should be excluded from processing.
type excludeFunc func(path string) bool

// runMinification processes a list of files through a minifier, logging statistics.
// It is shared between css_minify and js_minify plugins.
func runMinification(pluginName string, files []string, isExcluded excludeFunc, minify minifyFunc) {
	if len(files) == 0 {
		log.Printf("[%s] No files found", pluginName)
		return
	}

	log.Printf("[%s] Starting minification", pluginName)

	var totalOriginal, totalMinified int64
	var filesProcessed, filesSkipped int

	for _, file := range files {
		if isExcluded(file) {
			log.Printf("[%s] Skipping excluded file: %s", pluginName, filepath.Base(file))
			filesSkipped++
			continue
		}

		original, minifiedSize, err := minify(file)
		if err != nil {
			log.Printf("[%s] Warning: failed to minify %s: %v", pluginName, filepath.Base(file), err)
			continue
		}

		totalOriginal += original
		totalMinified += minifiedSize
		filesProcessed++
	}

	if totalOriginal > 0 {
		reduction := float64(totalOriginal-totalMinified) / float64(totalOriginal) * 100
		log.Printf("[%s] Completed: %d files processed, %d skipped", pluginName, filesProcessed, filesSkipped)
		log.Printf("[%s] Size reduction: %d -> %d bytes (%.1f%% smaller)",
			pluginName, totalOriginal, totalMinified, reduction)
	}
}

// isExcludedByPatterns checks if a filename matches any exclusion pattern.
// Supports exact matches and glob patterns (containing *, ?, or [).
func isExcludedByPatterns(filename string, excludeMap map[string]bool) bool {
	// Check exact match
	if excludeMap[filename] {
		return true
	}

	// Check glob pattern match
	for pattern := range excludeMap {
		if strings.ContainsAny(pattern, "*?[") {
			matched, err := filepath.Match(pattern, filename)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}
