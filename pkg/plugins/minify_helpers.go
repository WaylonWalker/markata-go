// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// minifyFunc processes a single file and returns original and minified sizes.
type minifyFunc func(path string) (original, minified int64, err error)

// excludeFunc checks if a file should be excluded from processing.
type excludeFunc func(path string) bool

// minifyResult holds the result of minifying a single file.
type minifyResult struct {
	original int64
	minified int64
}

// runMinification processes a list of files through a minifier, logging statistics.
// It is shared between css_minify and js_minify plugins.
// Files are processed concurrently using a worker pool sized to the number of CPUs.
func runMinification(pluginName string, files []string, isExcluded excludeFunc, minify minifyFunc) {
	if len(files) == 0 {
		log.Printf("[%s] No files found", pluginName)
		return
	}

	log.Printf("[%s] Starting minification", pluginName)

	// Filter excluded files first (cheap, serial)
	toProcess := make([]string, 0, len(files))
	var filesSkipped int
	for _, file := range files {
		if isExcluded(file) {
			log.Printf("[%s] Skipping excluded file: %s", pluginName, filepath.Base(file))
			filesSkipped++
			continue
		}
		toProcess = append(toProcess, file)
	}

	if len(toProcess) == 0 {
		log.Printf("[%s] All files excluded (%d skipped)", pluginName, filesSkipped)
		return
	}

	// Process files concurrently with a worker pool
	workers := runtime.NumCPU()
	if workers > len(toProcess) {
		workers = len(toProcess)
	}

	resultsCh := make(chan minifyResult, len(toProcess))
	semaphore := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, file := range toProcess {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			original, minifiedSize, err := minify(f)
			if err != nil {
				log.Printf("[%s] Warning: failed to minify %s: %v", pluginName, filepath.Base(f), err)
				return
			}
			resultsCh <- minifyResult{original: original, minified: minifiedSize}
		}(file)
	}

	// Close results channel once all goroutines complete
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results
	var totalOriginal, totalMinified int64
	var filesProcessed int
	for r := range resultsCh {
		totalOriginal += r.original
		totalMinified += r.minified
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
