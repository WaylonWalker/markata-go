package plugins

import (
	"path/filepath"
	"testing"
)

func TestSafeOutputPath(t *testing.T) {
	root := t.TempDir()
	path, err := safeOutputPath(root, filepath.Join("nested", "slug"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "nested", "slug"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	for _, traversal := range []string{"..", filepath.Join("..", "shared"), "/tmp/unsafe"} {
		if _, err := safeOutputPath(root, traversal); err == nil {
			t.Fatalf("safeOutputPath accepted traversal path %q", traversal)
		}
	}
}
