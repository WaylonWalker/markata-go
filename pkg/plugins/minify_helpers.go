// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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

type minifyCache struct {
	root  string
	files map[string]string
	mu    sync.Mutex
}

func writeGeneratedFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".markata-minify-")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func loadMinifyCache(path, root string) *minifyCache {
	cache := &minifyCache{root: root, files: make(map[string]string)}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &cache.files); err != nil {
			cache.files = make(map[string]string)
		}
		if cache.files == nil {
			cache.files = make(map[string]string)
		}
	}
	return cache
}

func (c *minifyCache) hashMatches(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)
	return c.files[filepath.ToSlash(rel)] == hex.EncodeToString(hash[:])
}

func (c *minifyCache) record(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return
	}
	hash := sha256.Sum256(data)
	c.mu.Lock()
	c.files[filepath.ToSlash(rel)] = hex.EncodeToString(hash[:])
	c.mu.Unlock()
}

func (c *minifyCache) save(path string) {
	c.mu.Lock()
	data, err := json.Marshal(c.files)
	c.mu.Unlock()
	if err == nil {
		if err := writeGeneratedFile(path, data); err != nil {
			return
		}
	}
}

// runMinification processes a list of files through a minifier, logging statistics.
// It is shared between css_minify and js_minify plugins.
// Files are processed concurrently using a worker pool sized to the given concurrency.
func runMinification(pluginName string, files []string, isExcluded excludeFunc, minify minifyFunc, concurrency int) {
	if len(files) == 0 {
		log.Printf("[%s] No files found", pluginName)
		return
	}

	log.Printf("[%s] Starting minification", pluginName)

	// Filter excluded files first (cheap, serial)
	toProcess := make([]string, 0, len(files))
	var filesSkipped int
	root := filepath.Dir(filepath.Dir(files[0]))
	cachePath := filepath.Join(root, ".markata-"+pluginName+"-cache")
	cache := loadMinifyCache(cachePath, root)
	for _, file := range files {
		if isExcluded(file) {
			log.Printf("[%s] Skipping excluded file: %s", pluginName, filepath.Base(file))
			filesSkipped++
			continue
		}
		if cache.hashMatches(file) {
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
	workers := concurrency
	if workers < 1 {
		workers = 1
	}
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
			cache.record(f)
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
	cache.save(cachePath)

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
