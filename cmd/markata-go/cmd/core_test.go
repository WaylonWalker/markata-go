package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigBaseDir(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte("title = \"test\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	got := resolveConfigBaseDir(configPath)
	want := filepath.Clean(configDir)
	if got != want {
		t.Fatalf("resolveConfigBaseDir() = %q, want %q", got, want)
	}
}

func TestResolveConfigRelativePath(t *testing.T) {
	baseDir := t.TempDir()
	absOut := filepath.Join(t.TempDir(), "out")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative path", path: "output", want: filepath.Join(baseDir, "output")},
		{name: "absolute path", path: absOut, want: absOut},
		{name: "empty path", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveConfigRelativePath(baseDir, tt.path); got != tt.want {
				t.Fatalf("resolveConfigRelativePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
