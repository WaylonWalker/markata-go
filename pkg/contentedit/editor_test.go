package contentedit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPost_UsesSlugFromFrontmatter(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	content := "---\ntitle: Hello World\nslug: custom-slug\n---\n\nBody"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	post, err := LoadPost(path)
	if err != nil {
		t.Fatalf("LoadPost() error = %v", err)
	}

	if post.Slug != "custom-slug" {
		t.Fatalf("Slug = %q, want %q", post.Slug, "custom-slug")
	}
	if post.PreviewURL != "/custom-slug/" {
		t.Fatalf("PreviewURL = %q, want %q", post.PreviewURL, "/custom-slug/")
	}
}

func TestSavePost_DetectsConflict(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "post.md")
	original := "---\ntitle: Original\n---\n\nBody"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	post := NewPost(path, "title: Updated", "Body")
	err := SavePost(post, &SaveOptions{BaseHash: ContentHash("stale")})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("SavePost() error = %v, want ErrConflict", err)
	}
}

func TestSavePost_FormatsFrontmatterAndUpdatesHash(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "post.md")
	post := NewPost(path, "title: Hello\ndate: 2026-03-26", "Body")

	if err := SavePost(post, nil); err != nil {
		t.Fatalf("SavePost() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if post.Hash == "" {
		t.Fatal("Hash should be set after save")
	}
	if post.PreviewURL != "/hello/" {
		t.Fatalf("PreviewURL = %q, want %q", post.PreviewURL, "/hello/")
	}
	if got := string(data); got == "" {
		t.Fatal("saved file should not be empty")
	}
}

func TestSavePost_SlugifiesLeadingSlashTitlesForPreviewURL(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "post.md")
	post := NewPost(path, "title: /now\ndate: 2026-03-26", "Body")

	if err := SavePost(post, nil); err != nil {
		t.Fatalf("SavePost() error = %v", err)
	}

	if post.Slug != "now" {
		t.Fatalf("Slug = %q, want %q", post.Slug, "now")
	}
	if post.PreviewURL != "/now/" {
		t.Fatalf("PreviewURL = %q, want %q", post.PreviewURL, "/now/")
	}
}

func TestBuildContent_AddsClosingDelimiterOnNewLine(t *testing.T) {
	t.Helper()
	got := BuildContent("title: /verify\ndate: 2026-02-24T10:36:57Z", "body")
	want := "---\ntitle: /verify\ndate: 2026-02-24T10:36:57Z\n---\n\nbody"
	if got != want {
		t.Fatalf("BuildContent() = %q, want %q", got, want)
	}
}

func TestFormatFrontmatter_PrefersKnownKeyOrder(t *testing.T) {
	t.Helper()
	got, err := FormatFrontmatter("templateKey: blog-post\ntags:\n  - slash\npublished: true\ndate: 2026-02-24T10:36:57Z\ntitle: /verify")
	if err != nil {
		t.Fatalf("FormatFrontmatter() error = %v", err)
	}
	want := "title: /verify\ndate: 2026-02-24T10:36:57Z\npublished: true\ntags:\n  - slash\ntemplateKey: blog-post"
	if got != want {
		t.Fatalf("FormatFrontmatter() = %q, want %q", got, want)
	}
}
