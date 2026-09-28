package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestIconVendorPackDownloadsThenUsesCache(t *testing.T) {
	archive := makeIconVendorArchive(t, map[string]string{
		"package/icons/smile.svg": testIconSVG,
		"package/LICENSE":         "Example icon license\n",
	})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/octet-stream")
		writeIconVendorArchive(t, w, archive)
	}))
	t.Cleanup(server.Close)

	root := t.TempDir()
	plugin := NewIconVendorPlugin()
	plugin.config.cacheDir = filepath.Join(root, "cache")
	plugin.config.target = filepath.Join(root, "static", ".icons")
	pack := iconVendorPack{
		name:        "lucide",
		version:     "1.2.3",
		source:      "url",
		url:         server.URL + "/lucide.tgz",
		archivePath: "package",
		iconsPath:   "icons",
		licensePath: "LICENSE",
	}

	if err := pack.validate(); err != nil {
		t.Fatal(err)
	}
	if err := plugin.vendorPack(t.Context(), pack); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests after first vendor = %d, want 1", got)
	}
	assertIconVendorFiles(t, plugin.config.target)

	// Remove only the materialized target. The archive cache remains, so the
	// second materialization must not hit the network again.
	if err := os.RemoveAll(plugin.config.target); err != nil {
		t.Fatal(err)
	}
	if err := plugin.vendorPack(t.Context(), pack); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests after cached vendor = %d, want 1", got)
	}
	assertIconVendorFiles(t, plugin.config.target)
}

func TestIconVendorPackUsesArchiveCacheOffline(t *testing.T) {
	archive := makeIconVendorArchive(t, map[string]string{
		"package/icons/smile.svg": testIconSVG,
		"package/LICENSE":         "Example icon license\n",
	})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeIconVendorArchive(t, w, archive)
	}))
	t.Cleanup(server.Close)

	root := t.TempDir()
	plugin := NewIconVendorPlugin()
	plugin.config.cacheDir = filepath.Join(root, "cache")
	plugin.config.target = filepath.Join(root, "static", ".icons")
	pack := iconVendorPack{
		name:        "lucide",
		version:     "1.2.3",
		source:      "url",
		url:         server.URL + "/lucide.tgz",
		archivePath: "package",
		iconsPath:   "icons",
		licensePath: "LICENSE",
	}
	if err := plugin.vendorPack(t.Context(), pack); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(plugin.config.target); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MARKATA_GO_OFFLINE", "1")
	if err := plugin.vendorPack(t.Context(), pack); err != nil {
		t.Fatalf("offline cached vendor failed: %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("offline cached vendor performed a network request: %d", got)
	}
	assertIconVendorFiles(t, plugin.config.target)
}

func TestIconVendorPackKeepsExistingTargetOffline(t *testing.T) {
	t.Setenv("MARKATA_GO_OFFLINE", "1")
	root := t.TempDir()
	plugin := NewIconVendorPlugin()
	plugin.config.cacheDir = filepath.Join(root, "cache")
	plugin.config.target = filepath.Join(root, "static", ".icons")
	iconPath := filepath.Join(plugin.config.target, "lucide", "smile.svg")
	if err := os.MkdirAll(filepath.Dir(iconPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iconPath, []byte(testIconSVG), 0o600); err != nil {
		t.Fatal(err)
	}

	pack := iconVendorPack{
		name:        "lucide",
		version:     "1.2.3",
		source:      "url",
		url:         "https://invalid.example.test/lucide.tgz",
		archivePath: "package",
		iconsPath:   "icons",
		licensePath: "LICENSE",
	}
	if err := plugin.vendorPack(t.Context(), pack); err != nil {
		t.Fatalf("existing offline target should be accepted: %v", err)
	}
}

func TestIconVendorRejectsUnsafeSVG(t *testing.T) {
	archive := makeIconVendorArchive(t, map[string]string{
		"package/icons/bad.svg": `<svg viewBox="0 0 1 1"><script>alert(1)</script></svg>`,
		"package/LICENSE":       "Example icon license\n",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeIconVendorArchive(t, w, archive)
	}))
	t.Cleanup(server.Close)

	root := t.TempDir()
	plugin := NewIconVendorPlugin()
	plugin.config.cacheDir = filepath.Join(root, "cache")
	plugin.config.target = filepath.Join(root, "static", ".icons")
	pack := iconVendorPack{
		name:        "unsafe",
		version:     "1.0.0",
		source:      "url",
		url:         server.URL + "/unsafe.tgz",
		archivePath: "package",
		iconsPath:   "icons",
		licensePath: "LICENSE",
	}

	err := plugin.vendorPack(t.Context(), pack)
	if err == nil || !strings.Contains(err.Error(), "refusing unsafe SVG") {
		t.Fatalf("expected unsafe SVG error, got %v", err)
	}
}

func TestIconVendorConfigApply(t *testing.T) {
	config := defaultIconVendorConfig()
	err := config.apply(map[string]interface{}{
		"enabled":   true,
		"cache_dir": "cache/icons",
		"target":    "static/custom-icons",
		"packs": []interface{}{
			map[string]interface{}{
				"name":         "lucide",
				"version":      "1.2.3",
				"source":       "npm",
				"package":      "lucide-static",
				"icons_path":   "icons",
				"license_path": "LICENSE",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.enabled || config.cacheDir != "cache/icons" || config.target != "static/custom-icons" {
		t.Fatalf("unexpected config: %+v", config)
	}
	if len(config.packs) != 1 {
		t.Fatalf("packs = %d, want 1", len(config.packs))
	}
	pack := config.packs[0]
	if err := pack.validate(); err != nil {
		t.Fatal(err)
	}
	if got := pack.archiveURL(); got != "https://registry.npmjs.org/lucide-static/-/lucide-static-1.2.3.tgz" {
		t.Fatalf("archiveURL = %q", got)
	}
}

func writeIconVendorArchive(t *testing.T, w http.ResponseWriter, archive []byte) {
	t.Helper()
	if _, err := w.Write(archive); err != nil {
		t.Errorf("write icon vendor archive response: %v", err)
	}
}

func assertIconVendorFiles(t *testing.T, target string) {
	t.Helper()
	icon, err := os.ReadFile(filepath.Join(target, "lucide", "smile.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if string(icon) != testIconSVG {
		t.Fatalf("vendored icon changed: %q", icon)
	}
	license, err := os.ReadFile(filepath.Join(target, "licenses", "lucide.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(license) != "Example icon license\n" {
		t.Fatalf("license = %q", license)
	}
}

func makeIconVendorArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, contents := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(contents)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tarWriter, contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
