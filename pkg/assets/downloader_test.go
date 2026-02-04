package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDownloader(t *testing.T) {
	d := NewDownloader("/tmp/test-cache", true)
	if d.cacheDir != "/tmp/test-cache" {
		t.Errorf("expected cacheDir /tmp/test-cache, got %s", d.cacheDir)
	}
	if !d.verifyIntegrity {
		t.Error("expected verifyIntegrity to be true")
	}
	if d.httpClient == nil {
		t.Error("expected httpClient to be non-nil")
	}
}

func TestDownloader_Download(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Create temp cache dir
	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "test-asset",
		URL:       server.URL + "/test.js",
		LocalPath: "test/test.js",
		Type:      "js",
	}

	ctx := context.Background()
	result, err := d.Download(ctx, asset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Cached {
		t.Error("expected Cached to be false on first download")
	}
	if result.Size != 12 { // "test content" = 12 bytes
		t.Errorf("expected size 12, got %d", result.Size)
	}
	if result.Error != nil {
		t.Errorf("unexpected result error: %v", result.Error)
	}

	// Verify file exists
	cachedPath := filepath.Join(cacheDir, asset.LocalPath)
	if _, err := os.Stat(cachedPath); os.IsNotExist(err) {
		t.Error("expected cached file to exist")
	}

	// Second download should be cached
	result2, err := d.Download(ctx, asset)
	if err != nil {
		t.Fatalf("unexpected error on second download: %v", err)
	}
	if !result2.Cached {
		t.Error("expected Cached to be true on second download")
	}
}

func TestDownloader_Download_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "missing-asset",
		URL:       server.URL + "/missing.js",
		LocalPath: "missing/missing.js",
		Type:      "js",
	}

	ctx := context.Background()
	result, err := d.Download(ctx, asset)
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if result.Error == nil {
		t.Error("expected result.Error to be set")
	}
}

func TestDownloader_IsCached(t *testing.T) {
	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "test-asset",
		LocalPath: "test/file.js",
	}

	// Not cached initially
	if d.IsCached(asset) {
		t.Error("expected IsCached to return false initially")
	}

	// Create the cached file
	cachedPath := filepath.Join(cacheDir, asset.LocalPath)
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachedPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Now should be cached
	if !d.IsCached(asset) {
		t.Error("expected IsCached to return true after file created")
	}
}

func TestDownloader_GetCachedPath(t *testing.T) {
	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "test-asset",
		LocalPath: "test/file.js",
	}

	// Should return empty string when not cached
	path := d.GetCachedPath(asset)
	if path != "" {
		t.Errorf("expected empty string, got %s", path)
	}

	// Create cached file
	cachedPath := filepath.Join(cacheDir, asset.LocalPath)
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachedPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should return path when cached
	path = d.GetCachedPath(asset)
	if path != cachedPath {
		t.Errorf("expected %s, got %s", cachedPath, path)
	}
}

func TestDownloader_CopyToOutput(t *testing.T) {
	cacheDir := t.TempDir()
	outputDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "test-asset",
		LocalPath: "test/file.js",
	}

	// Create cached file
	cachedPath := filepath.Join(cacheDir, asset.LocalPath)
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("test content for copy")
	if err := os.WriteFile(cachedPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Copy to output
	if err := d.CopyToOutput(asset, outputDir); err != nil {
		t.Fatalf("CopyToOutput failed: %v", err)
	}

	// Verify output file
	outputPath := filepath.Join(outputDir, asset.LocalPath)
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: expected %s, got %s", content, data)
	}
}

func TestDownloader_CopyToOutput_NotCached(t *testing.T) {
	cacheDir := t.TempDir()
	outputDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	asset := Asset{
		Name:      "uncached-asset",
		LocalPath: "uncached/file.js",
	}

	err := d.CopyToOutput(asset, outputDir)
	if err == nil {
		t.Error("expected error when copying uncached asset")
	}
}

func TestDownloader_Clean(t *testing.T) {
	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	// Create some files
	testFile := filepath.Join(cacheDir, "test", "file.js")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Clean
	if err := d.Clean(); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify cache dir is removed
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("expected cache dir to be removed")
	}
}

func TestDownloader_Status(t *testing.T) {
	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	statuses := d.Status()
	if len(statuses) == 0 {
		t.Error("expected non-empty status list")
	}

	// All should be uncached initially
	for _, s := range statuses {
		if s.Cached {
			t.Errorf("expected asset %s to be uncached", s.Asset.Name)
		}
	}
}

func TestDownloader_DownloadAll(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	d := NewDownloader(cacheDir, false)

	// Override the registry temporarily (we can't do this in current impl)
	// Instead, just test that DownloadAll runs without panic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will try to download real assets which may fail in test environment
	// Just verify it doesn't panic
	_ = d.DownloadAll(ctx, 2)
}

func TestVerifyIntegrity(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		integrity string
		wantErr   bool
	}{
		{
			name:      "valid sha256",
			data:      []byte("hello"),
			integrity: "sha256-LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ=",
			wantErr:   false,
		},
		{
			name:      "invalid sha256",
			data:      []byte("hello"),
			integrity: "sha256-invalidhash",
			wantErr:   true,
		},
		{
			name:      "invalid format",
			data:      []byte("hello"),
			integrity: "nohyphen",
			wantErr:   true,
		},
		{
			name:      "unsupported algorithm",
			data:      []byte("hello"),
			integrity: "md5-rL0Y20zC+Fzt72VPzMSk2A==",
			wantErr:   true,
		},
		{
			name:      "valid sha384",
			data:      []byte("hello"),
			integrity: "sha384-WeF0h3dEjGnea4ANejO7+5/xtGPkQ1TDVTvNucZm+pASWjx5+QOXvfX2oT3oKGhP",
			wantErr:   false,
		},
		{
			name:      "valid sha512",
			data:      []byte("hello"),
			integrity: "sha512-m3HSJL1i83hdltRq0+o9czGb+8KJDKra4t/3JRlnPKcjI8PZm6XBHXx6zG4UuMXaDEZjR1wuXDre9G9zvN7AQw==",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyIntegrity(tt.data, tt.integrity)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyIntegrity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
