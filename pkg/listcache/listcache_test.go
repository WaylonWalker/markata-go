package listcache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadChangedPosts_ParsesAllChangedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	changed := make(map[string]bool, 12)
	for i := 0; i < 12; i++ {
		path := filepath.Join("post-", string(rune('a'+i)), "index.md")
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte("---\ntitle: Test\n---\n\nContent"), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", fullPath, err)
		}
		changed[path] = true
	}

	posts, err := loadChangedPosts(dir, changed, nil, 4)
	if err != nil {
		t.Fatalf("loadChangedPosts() error = %v", err)
	}
	if len(posts) != len(changed) {
		t.Fatalf("loadChangedPosts() returned %d posts, want %d", len(posts), len(changed))
	}
	for _, post := range posts {
		if !changed[post.Path] {
			t.Errorf("unexpected post path %q", post.Path)
		}
	}
}
