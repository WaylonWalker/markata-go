package assets

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Common errors for asset downloading.
var (
	ErrAssetNotFound     = errors.New("asset not found in registry")
	ErrIntegrityMismatch = errors.New("asset integrity check failed")
	ErrDownloadFailed    = errors.New("asset download failed")
)

// DownloadResult represents the result of downloading a single asset.
type DownloadResult struct {
	Asset    Asset
	Cached   bool
	Error    error
	Size     int64
	Duration time.Duration
}

// Downloader handles downloading and caching of CDN assets.
type Downloader struct {
	cacheDir        string
	verifyIntegrity bool
	httpClient      *http.Client
	userAgent       string
}

// NewDownloader creates a new asset downloader.
func NewDownloader(cacheDir string, verifyIntegrity bool) *Downloader {
	return &Downloader{
		cacheDir:        cacheDir,
		verifyIntegrity: verifyIntegrity,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		userAgent: "markata-go/1.0 (CDN Asset Downloader)",
	}
}

// Download downloads a single asset to the cache directory.
// Returns the cached file path on success.
func (d *Downloader) Download(ctx context.Context, asset Asset) (*DownloadResult, error) {
	start := time.Now()
	result := &DownloadResult{Asset: asset}

	// Check if already cached
	cachedPath := d.getCachePath(asset)
	if info, err := os.Stat(cachedPath); err == nil {
		result.Cached = true
		result.Size = info.Size()
		result.Duration = time.Since(start)
		return result, nil
	}

	// Create cache directory
	cacheDir := filepath.Dir(cachedPath)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		result.Error = fmt.Errorf("create cache dir: %w", err)
		return result, result.Error
	}

	// Download the asset
	req, err := http.NewRequestWithContext(ctx, "GET", asset.URL, http.NoBody)
	if err != nil {
		result.Error = fmt.Errorf("create request: %w", err)
		return result, result.Error
	}
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("%w: %w", ErrDownloadFailed, err)
		return result, result.Error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
		return result, result.Error
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("read response: %w", err)
		return result, result.Error
	}

	// Verify integrity if hash provided and verification enabled
	if d.verifyIntegrity && asset.Integrity != "" {
		if err := verifyIntegrity(data, asset.Integrity); err != nil {
			result.Error = fmt.Errorf("%w: %w", ErrIntegrityMismatch, err)
			return result, result.Error
		}
	}

	// Write to cache
	if err := os.WriteFile(cachedPath, data, 0o644); err != nil { //nolint:gosec // cache files need to be readable by user
		result.Error = fmt.Errorf("write cache file: %w", err)
		return result, result.Error
	}

	result.Size = int64(len(data))
	result.Duration = time.Since(start)
	return result, nil
}

// DownloadAll downloads all registered assets concurrently.
func (d *Downloader) DownloadAll(ctx context.Context, concurrency int) []DownloadResult {
	assets := Registry()
	if concurrency <= 0 {
		concurrency = 4
	}

	results := make([]DownloadResult, len(assets))
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, asset := range assets {
		wg.Add(1)
		go func(idx int, a Asset) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := d.Download(ctx, a)
			if err != nil {
				result.Error = err
			}
			results[idx] = *result
		}(i, asset)
	}

	wg.Wait()
	return results
}

// IsCached checks if an asset is already cached.
func (d *Downloader) IsCached(asset Asset) bool {
	cachedPath := d.getCachePath(asset)
	_, err := os.Stat(cachedPath)
	return err == nil
}

// GetCachedPath returns the path to the cached asset file.
// Returns empty string if not cached.
func (d *Downloader) GetCachedPath(asset Asset) string {
	cachedPath := d.getCachePath(asset)
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath
	}
	return ""
}

// CopyToOutput copies a cached asset to the output directory.
func (d *Downloader) CopyToOutput(asset Asset, outputDir string) error {
	cachedPath := d.getCachePath(asset)
	if _, err := os.Stat(cachedPath); err != nil {
		return fmt.Errorf("asset not cached: %s", asset.Name)
	}

	outputPath := filepath.Join(outputDir, asset.LocalPath)
	outputParent := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputParent, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Read from cache
	data, err := os.ReadFile(cachedPath)
	if err != nil {
		return fmt.Errorf("read cached file: %w", err)
	}

	// Write to output
	if err := os.WriteFile(outputPath, data, 0o644); err != nil { //nolint:gosec // output files need to be readable by web server
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

// CopyAllToOutput copies all cached assets to the output directory.
func (d *Downloader) CopyAllToOutput(outputDir string) error {
	for _, asset := range Registry() {
		if d.IsCached(asset) {
			if err := d.CopyToOutput(asset, outputDir); err != nil {
				return err
			}
		}
	}
	return nil
}

// Clean removes all cached assets.
func (d *Downloader) Clean() error {
	return os.RemoveAll(d.cacheDir)
}

// Status returns the status of all assets.
func (d *Downloader) Status() []AssetStatus {
	assets := Registry()
	statuses := make([]AssetStatus, len(assets))
	for i, asset := range assets {
		statuses[i] = AssetStatus{
			Asset:  asset,
			Cached: d.IsCached(asset),
		}
		if statuses[i].Cached {
			if info, err := os.Stat(d.getCachePath(asset)); err == nil {
				statuses[i].Size = info.Size()
				statuses[i].CachedAt = info.ModTime()
			}
		}
	}
	return statuses
}

// AssetStatus represents the status of an asset.
type AssetStatus struct {
	Asset    Asset
	Cached   bool
	Size     int64
	CachedAt time.Time
}

// getCachePath returns the full path to the cached asset file.
func (d *Downloader) getCachePath(asset Asset) string {
	return filepath.Join(d.cacheDir, asset.LocalPath)
}

// verifyIntegrity verifies the integrity of data against an SRI hash.
// Supports sha256, sha384, and sha512 prefixed hashes.
func verifyIntegrity(data []byte, integrity string) error {
	// SRI format: algorithm-base64hash
	parts := strings.SplitN(integrity, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid integrity format: %s", integrity)
	}

	algorithm := parts[0]
	expectedHash := parts[1]

	var hasher hash.Hash
	switch algorithm {
	case "sha256":
		hasher = sha256.New()
	case "sha384":
		hasher = sha512.New384()
	case "sha512":
		hasher = sha512.New()
	default:
		return fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	hasher.Write(data)
	actualHash := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}
