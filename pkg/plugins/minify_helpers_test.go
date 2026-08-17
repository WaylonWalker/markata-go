package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMinificationCachesUnchangedFiles(t *testing.T) {
	output := t.TempDir()
	cssDir := filepath.Join(output, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cssDir, "site.css")
	if err := os.WriteFile(file, []byte("body { color: red; }"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	minify := func(path string) (int64, int64, error) {
		calls++
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, 0, err
		}
		return int64(len(data)), int64(len(data)), nil
	}
	runMinification("css_minify", []string{file}, func(string) bool { return false }, minify, 1)
	runMinification("css_minify", []string{file}, func(string) bool { return false }, minify, 1)
	if calls != 1 {
		t.Fatalf("minifier called %d times, want 1", calls)
	}
}
