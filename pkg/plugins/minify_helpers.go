// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const minifyStatusRestored = "restored"

type minifyFunc func([]byte) ([]byte, error)
type excludeFunc func(path string) bool

type minifyResult struct {
	status   string
	original int64
	minified int64
}

func (c *minifyCache) reuse(path, digest, recipe string, current []byte, record *minifyRecord) (string, error) {
	if record == nil {
		return "", nil
	}
	if recipe != record.Recipe {
		log.Printf("[asset_minify] Recipe changed for %s; transforming current stage-input bytes", path)
		return "", nil
	}
	// Only the exact input/recipe relation authorizes reuse. Even an identical
	// prior output may need another transform: CSS comment handling is not
	// necessarily idempotent. Historical input/source blobs are never selected.
	if digest == record.SourceHash {
		result, err := c.readBlob(record.OutputHash)
		if err == nil {
			if !isEmptyJSAsset(path, current, result) {
				if err := writeGeneratedFile(path, result); err != nil {
					return "", fmt.Errorf("restoring asset: %w", err)
				}
			}
			return minifyStatusRestored, nil
		}
		log.Printf("[asset_minify] Warning: cached result for %s unavailable: %v; transforming current stage-input bytes", path, err)
	}
	return "", nil
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

func isEmptyJSAsset(path string, current, result []byte) bool {
	return len(current) == 0 && len(result) == 0 && strings.EqualFold(filepath.Ext(path), ".js")
}

func minifyAsset(path, recipe string, cache *minifyCache, transform minifyFunc) (minifyResult, error) {
	var asset string
	if cache != nil {
		var err error
		asset, err = cache.identity(path)
		if err != nil {
			return minifyResult{}, err
		}
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return minifyResult{}, fmt.Errorf("reading asset: %w", err)
	}
	digest := minifyHash(current)
	mayHaveOldRecord := false
	if cache != nil {
		record, readErr := cache.readRecord(asset)
		mayHaveOldRecord = record != nil || readErr != nil
		if readErr != nil {
			log.Printf("[asset_minify] Warning: reading cache record for %s: %v", path, readErr)
		}
		status, err := cache.reuse(path, digest, recipe, current, record)
		if err != nil {
			return minifyResult{}, err
		}
		if status != "" {
			return minifyResult{status: status}, nil
		}
	}
	result, err := transform(current)
	if err != nil {
		return minifyResult{}, err
	}
	outputHash := minifyHash(result)
	if cache != nil {
		record := &minifyRecord{
			Version: 1, Scope: cache.scope, Asset: asset, Recipe: recipe,
			OutputHash: outputHash, SourceHash: digest,
		}
		if err := cache.persist(record, result); err != nil {
			log.Printf("[asset_minify] Warning: persisting cache for %s: %v", path, err)
			// Publishing new bytes while an old many-to-one relation survives
			// would allow a later recipe change to resurrect the wrong source.
			if mayHaveOldRecord {
				if invalidateErr := cache.invalidate(asset); invalidateErr != nil {
					return minifyResult{}, fmt.Errorf("cache persistence and invalidation failed; leaving target unchanged: %w", invalidateErr)
				}
				log.Printf("[asset_minify] Warning: cache record invalidated for %s; writing uncached result", path)
			} else {
				log.Printf("[asset_minify] Warning: no previous cache record for %s; writing uncached result", path)
			}
		}
	}
	if !isEmptyJSAsset(path, current, result) {
		if err := writeGeneratedFile(path, result); err != nil {
			return minifyResult{}, fmt.Errorf("writing minified asset: %w", err)
		}
	}
	return minifyResult{status: "transformed", original: int64(len(current)), minified: int64(len(result))}, nil
}

// runMinification retains warning-only per-file failures and bounded workers.
func runMinification(pluginName string, files []string, isExcluded excludeFunc, transform minifyFunc, concurrency int, cache *minifyCache, recipe string) {
	if len(files) == 0 {
		log.Printf("[%s] No files found", pluginName)
		return
	}
	toProcess := make([]string, 0, len(files))
	counts := map[string]int{"transformed": 0, minifyStatusRestored: 0, "excluded": 0, "failed": 0}
	for _, file := range files {
		if isExcluded(file) {
			counts["excluded"]++
		} else {
			toProcess = append(toProcess, file)
		}
	}
	workers := concurrency
	if workers < 1 {
		workers = 1
	}
	results := make(chan minifyResult, len(toProcess))
	semaphore := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, file := range toProcess {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			result, err := minifyAsset(path, recipe, cache, transform)
			if err != nil {
				log.Printf("[%s] Warning: failed to minify %s: %v", pluginName, path, err)
				result.status = "failed"
			}
			results <- result
		}(file)
	}
	wg.Wait()
	close(results)
	var original, minified int64
	for result := range results {
		counts[result.status]++
		original += result.original
		minified += result.minified
	}
	log.Printf("[%s] Completed: %d transformed, %d restored, %d excluded, %d failed",
		pluginName, counts["transformed"], counts[minifyStatusRestored], counts["excluded"], counts["failed"])
	if original > 0 {
		log.Printf("[%s] Size reduction: %d -> %d bytes (%.1f%% smaller)", pluginName,
			original, minified, float64(original-minified)/float64(original)*100)
	}
}

// isExcludedByPatterns checks exact filenames and glob patterns.
func isExcludedByPatterns(filename string, excludeMap map[string]bool) bool {
	if excludeMap[filename] {
		return true
	}
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
