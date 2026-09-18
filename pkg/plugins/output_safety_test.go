package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOutputRoot_AllowsSymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	output := filepath.Join(link, "output")

	if err := validateOutputRoot(output); err != nil {
		t.Fatalf("validateOutputRoot() error = %v", err)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateOutputPath(output, filepath.Join(output, "nested", "index.html")); err != nil {
		t.Fatalf("validateOutputPath() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "output")); err != nil {
		t.Fatalf("output was not created through symlinked ancestor: %v", err)
	}
}
