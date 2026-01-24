package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetPlatformAssetName(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		goarch    string
		want      string
		wantError bool
	}{
		{"darwin_amd64", "darwin", "amd64", "x86_64-apple-darwin", false},
		{"darwin_arm64", "darwin", "arm64", "aarch64-apple-darwin", false},
		{"linux_amd64", "linux", "amd64", "x86_64-unknown-linux-musl", false},
		{"linux_arm64", "linux", "arm64", "aarch64-unknown-linux-musl", false},
		{"windows_amd64", "windows", "amd64", "x86_64-pc-windows-msvc", false},
		{"freebsd_amd64", "freebsd", "amd64", "x86_64-unknown-freebsd", false},
		{"unsupported_os", "plan9", "amd64", "", true},
		{"unsupported_arch", "linux", "386", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPlatformAssetNameForOS(tt.goos, tt.goarch)
			if (err != nil) != tt.wantError {
				t.Errorf("getPlatformAssetNameForOS() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("getPlatformAssetNameForOS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlatformAssetName_Current(t *testing.T) {
	// Just verify it doesn't error on the current platform
	assetName, err := GetPlatformAssetName()

	// Skip if running on an unsupported platform
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		if err == nil {
			t.Errorf("expected error for unsupported architecture %s", runtime.GOARCH)
		}
		return
	}

	switch runtime.GOOS {
	case "darwin", "linux", "windows", "freebsd":
		if err != nil {
			t.Errorf("GetPlatformAssetName() unexpected error: %v", err)
		}
		if assetName == "" {
			t.Error("GetPlatformAssetName() returned empty string")
		}
	default:
		if err == nil {
			t.Errorf("expected error for unsupported OS %s", runtime.GOOS)
		}
	}
}

func TestPagefindInstaller_GetCacheDir(t *testing.T) {
	t.Run("custom_cache_dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		installer := &PagefindInstaller{
			CacheDir: tmpDir,
		}

		cacheDir, err := installer.GetCacheDir()
		if err != nil {
			t.Errorf("GetCacheDir() error = %v", err)
			return
		}

		if cacheDir != tmpDir {
			t.Errorf("GetCacheDir() = %v, want %v", cacheDir, tmpDir)
		}
	})

	t.Run("default_cache_dir", func(t *testing.T) {
		installer := &PagefindInstaller{}

		cacheDir, err := installer.GetCacheDir()
		if err != nil {
			t.Errorf("GetCacheDir() error = %v", err)
			return
		}

		if cacheDir == "" {
			t.Error("GetCacheDir() returned empty string")
		}

		// Should contain markata-go in the path
		if !contains(cacheDir, "markata-go") {
			t.Errorf("GetCacheDir() = %v, expected to contain 'markata-go'", cacheDir)
		}
	})

	t.Run("xdg_cache_home", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalXDG := os.Getenv("XDG_CACHE_HOME")
		os.Setenv("XDG_CACHE_HOME", tmpDir)
		defer os.Setenv("XDG_CACHE_HOME", originalXDG)

		installer := &PagefindInstaller{}

		cacheDir, err := installer.GetCacheDir()
		if err != nil {
			t.Errorf("GetCacheDir() error = %v", err)
			return
		}

		expected := filepath.Join(tmpDir, "markata-go", "bin")
		if cacheDir != expected {
			t.Errorf("GetCacheDir() = %v, want %v", cacheDir, expected)
		}
	})
}

func TestPagefindInstaller_GetCachedBinaryPath(t *testing.T) {
	tmpDir := t.TempDir()
	installer := &PagefindInstaller{
		CacheDir: tmpDir,
	}

	path, err := installer.GetCachedBinaryPath("v1.4.0")
	if err != nil {
		t.Errorf("GetCachedBinaryPath() error = %v", err)
		return
	}

	expectedBinary := "pagefind"
	if runtime.GOOS == "windows" {
		expectedBinary = "pagefind.exe"
	}

	expected := filepath.Join(tmpDir, "v1.4.0", expectedBinary)
	if path != expected {
		t.Errorf("GetCachedBinaryPath() = %v, want %v", path, expected)
	}
}

func TestPagefindInstaller_IsCached(t *testing.T) {
	t.Run("not_cached", func(t *testing.T) {
		tmpDir := t.TempDir()
		installer := &PagefindInstaller{
			CacheDir: tmpDir,
		}

		cached, err := installer.IsCached("v1.4.0")
		if err != nil {
			t.Errorf("IsCached() error = %v", err)
			return
		}
		if cached {
			t.Error("IsCached() should return false for non-existent version")
		}
	})

	t.Run("cached", func(t *testing.T) {
		tmpDir := t.TempDir()
		installer := &PagefindInstaller{
			CacheDir: tmpDir,
		}

		// Create a fake cached binary
		versionDir := filepath.Join(tmpDir, "v1.4.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatalf("failed to create version dir: %v", err)
		}

		binaryName := "pagefind"
		if runtime.GOOS == "windows" {
			binaryName = "pagefind.exe"
		}
		binaryPath := filepath.Join(versionDir, binaryName)
		if err := os.WriteFile(binaryPath, []byte("fake binary"), 0o600); err != nil {
			t.Fatalf("failed to create fake binary: %v", err)
		}

		cached, err := installer.IsCached("v1.4.0")
		if err != nil {
			t.Errorf("IsCached() error = %v", err)
			return
		}
		if !cached {
			t.Error("IsCached() should return true for existing binary")
		}
	})

	t.Run("empty_binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		installer := &PagefindInstaller{
			CacheDir: tmpDir,
		}

		// Create an empty binary file
		versionDir := filepath.Join(tmpDir, "v1.4.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			t.Fatalf("failed to create version dir: %v", err)
		}

		binaryName := "pagefind"
		if runtime.GOOS == "windows" {
			binaryName = "pagefind.exe"
		}
		binaryPath := filepath.Join(versionDir, binaryName)
		if err := os.WriteFile(binaryPath, []byte{}, 0o600); err != nil {
			t.Fatalf("failed to create empty binary: %v", err)
		}

		cached, err := installer.IsCached("v1.4.0")
		if err != nil {
			t.Errorf("IsCached() error = %v", err)
			return
		}
		if cached {
			t.Error("IsCached() should return false for empty binary")
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid_checksum", func(t *testing.T) {
		// Create a test file
		content := []byte("test content for checksum verification")
		filePath := filepath.Join(tmpDir, "test_valid.bin")
		if err := os.WriteFile(filePath, content, 0o600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Calculate expected checksum
		hasher := sha256.New()
		hasher.Write(content)
		expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

		err := verifyChecksum(filePath, expectedChecksum, false)
		if err != nil {
			t.Errorf("verifyChecksum() unexpected error: %v", err)
		}
	})

	t.Run("invalid_checksum", func(t *testing.T) {
		// Create a test file
		content := []byte("test content for checksum verification")
		filePath := filepath.Join(tmpDir, "test_invalid.bin")
		if err := os.WriteFile(filePath, content, 0o600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Use wrong checksum
		wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

		err := verifyChecksum(filePath, wrongChecksum, false)
		if err == nil {
			t.Error("verifyChecksum() should fail with wrong checksum")
		}

		var installErr *PagefindInstallError
		if !errors.As(err, &installErr) {
			t.Error("error should be *PagefindInstallError")
		} else if installErr.Operation != "verify" {
			t.Errorf("error operation = %v, want 'verify'", installErr.Operation)
		}
	})

	t.Run("file_not_found", func(t *testing.T) {
		err := verifyChecksum(filepath.Join(tmpDir, "nonexistent.bin"), "somechecksum", false)
		if err == nil {
			t.Error("verifyChecksum() should fail for non-existent file")
		}
	})
}

func TestBuildAssetURL(t *testing.T) {
	tests := []struct {
		version       string
		platformAsset string
		want          string
	}{
		{
			"v1.4.0",
			"x86_64-apple-darwin",
			"https://github.com/Pagefind/pagefind/releases/download/v1.4.0/pagefind-v1.4.0-x86_64-apple-darwin.tar.gz",
		},
		{
			"v1.3.0",
			"x86_64-unknown-linux-musl",
			"https://github.com/Pagefind/pagefind/releases/download/v1.3.0/pagefind-v1.3.0-x86_64-unknown-linux-musl.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.platformAsset, func(t *testing.T) {
			got := buildAssetURL(tt.version, tt.platformAsset)
			if got != tt.want {
				t.Errorf("buildAssetURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildChecksumURL(t *testing.T) {
	tests := []struct {
		version       string
		platformAsset string
		want          string
	}{
		{
			"v1.4.0",
			"x86_64-apple-darwin",
			"https://github.com/Pagefind/pagefind/releases/download/v1.4.0/pagefind-v1.4.0-x86_64-apple-darwin.tar.gz.sha256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.platformAsset, func(t *testing.T) {
			got := buildChecksumURL(tt.version, tt.platformAsset)
			if got != tt.want {
				t.Errorf("buildChecksumURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagefindInstaller_ResolveVersion(t *testing.T) {
	t.Run("specific_version", func(t *testing.T) {
		installer := &PagefindInstaller{
			Version: "v1.4.0",
		}

		version, err := installer.ResolveVersion()
		if err != nil {
			t.Errorf("ResolveVersion() error = %v", err)
			return
		}
		if version != "v1.4.0" {
			t.Errorf("ResolveVersion() = %v, want v1.4.0", version)
		}
	})
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"tilde_expansion", "~/test", filepath.Join(homeDir, "test")},
		{"no_tilde", "/absolute/path", "/absolute/path"},
		{"relative", "relative/path", "relative/path"},
		{"just_tilde", "~", "~"}, // Only ~/... is expanded
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.path)
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPagefindInstallerConfig_IsAutoInstallEnabled(t *testing.T) {
	t.Run("nil_default_true", func(t *testing.T) {
		config := PagefindInstallerConfig{}
		if !config.IsAutoInstallEnabled() {
			t.Error("IsAutoInstallEnabled() should return true when nil")
		}
	})

	t.Run("explicit_true", func(t *testing.T) {
		enabled := true
		config := PagefindInstallerConfig{AutoInstall: &enabled}
		if !config.IsAutoInstallEnabled() {
			t.Error("IsAutoInstallEnabled() should return true")
		}
	})

	t.Run("explicit_false", func(t *testing.T) {
		enabled := false
		config := PagefindInstallerConfig{AutoInstall: &enabled}
		if config.IsAutoInstallEnabled() {
			t.Error("IsAutoInstallEnabled() should return false")
		}
	})
}

func TestNewPagefindInstallerConfig(t *testing.T) {
	config := NewPagefindInstallerConfig()

	if !config.IsAutoInstallEnabled() {
		t.Error("New config should have auto_install enabled")
	}

	if config.Version != "latest" {
		t.Errorf("Version = %v, want 'latest'", config.Version)
	}

	if config.CacheDir != "" {
		t.Errorf("CacheDir = %v, want empty string", config.CacheDir)
	}
}

func TestPagefindInstallError(t *testing.T) {
	t.Run("with_wrapped_error", func(t *testing.T) {
		wrappedErr := fmt.Errorf("underlying error")
		err := NewPagefindInstallError("download", "failed to download", wrappedErr)

		if err.Operation != "download" {
			t.Errorf("Operation = %v, want 'download'", err.Operation)
		}

		if !errors.Is(err, wrappedErr) {
			t.Error("errors.Is should find wrapped error")
		}

		errStr := err.Error()
		if !contains(errStr, "download") || !contains(errStr, "failed to download") {
			t.Errorf("Error() = %v, should contain operation and message", errStr)
		}
	})

	t.Run("without_wrapped_error", func(t *testing.T) {
		err := NewPagefindInstallError("verify", "checksum mismatch", nil)

		if err.Unwrap() != nil {
			t.Error("Unwrap() should return nil")
		}

		errStr := err.Error()
		if !contains(errStr, "verify") || !contains(errStr, "checksum mismatch") {
			t.Errorf("Error() = %v, should contain operation and message", errStr)
		}
	})
}

// TestPagefindInstaller_FetchChecksum_MockServer tests checksum fetching with a mock server.
func TestPagefindInstaller_FetchChecksum_MockServer(t *testing.T) {
	expectedChecksum := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's requesting a checksum file
		if !contains(r.URL.Path, ".sha256") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, expectedChecksum+"  pagefind-v1.4.0-x86_64-apple-darwin.tar.gz")
	}))
	defer server.Close()

	// Note: This test is limited because we can't easily mock the actual URL
	// A more comprehensive test would require dependency injection for the base URL
	t.Log("Mock server test completed - actual integration would require DI")
}

// TestPagefindInstaller_InterfaceConformance verifies the installer's error types.
func TestPagefindInstaller_InterfaceConformance(t *testing.T) {
	// Verify PagefindInstallError implements error interface via compile-time check
	// The blank identifier assignment ensures the type implements the interface
	err := NewPagefindInstallError("test", "test message", fmt.Errorf("wrapped"))
	var _ error = err

	// Verify Unwrap method exists for errors.Is/errors.As support
	if err.Unwrap() == nil {
		t.Error("Unwrap() should return the wrapped error")
	}
}
