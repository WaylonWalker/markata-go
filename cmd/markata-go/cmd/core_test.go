package cmd

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigBaseDir(t *testing.T) {
	got := resolveConfigBaseDir("/tmp/site/markata-go.toml")
	want := filepath.Clean("/tmp/site")
	if got != want {
		t.Fatalf("resolveConfigBaseDir() = %q, want %q", got, want)
	}
}

func TestResolveConfigRelativePath(t *testing.T) {
	baseDir := "/tmp/site"

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative path", path: "output", want: filepath.Join(baseDir, "output")},
		{name: "absolute path", path: "/var/out", want: "/var/out"},
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
