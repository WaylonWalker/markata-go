package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/contentindex"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestContentIndexPlugin_WritesPublicMetadataAndFeeds(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "content-index.json")
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{ContentDir: dir, OutputDir: dir, Extra: map[string]interface{}{"content_index": map[string]interface{}{"enabled": true, "output": output}, "markata_version": "test"}})
	public := models.NewPost("posts/public.md")
	public.Slug, public.Href, public.Published, public.Tags = "public", "/public/", true, []string{"z", "a"}
	unpublished := models.NewPost("pages/unpublished.md")
	unpublished.Slug, unpublished.Href = "unpublished", "/unpublished/"
	private := models.NewPost("posts/private.md")
	private.Private, private.Published = true, true
	draft := models.NewPost("posts/draft.md")
	draft.Draft = true
	m.SetPosts([]*models.Post{private, draft, public, unpublished})
	m.SetFeeds([]*lifecycle.Feed{{Name: "blog", Posts: []*models.Post{public, private}}, {Name: "archive", Posts: []*models.Post{public}}, {Name: "draft", Posts: []*models.Post{unpublished}}})
	if err := NewContentIndexPlugin().Write(m); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	index, err := contentindex.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if index.DocumentCount != 2 || index.Documents[0].Path != unpublished.Path || index.Documents[1].Path != public.Path {
		t.Fatalf("privacy filtering failed: %#v", index)
	}
	if got := index.Documents[1].Feeds; len(got) != 2 || got[0] != "archive" || got[1] != "blog" {
		t.Fatalf("feeds = %v", got)
	}
	if index.Documents[0].Published || len(index.Documents[0].Feeds) != 1 || index.Documents[0].Feeds[0] != "draft" {
		t.Fatalf("unpublished direct page or draft feed was conflated: %#v", index.Documents[0])
	}
	headers, err := os.ReadFile(filepath.Join(dir, "_headers"))
	if err != nil {
		t.Fatalf("read generated headers: %v", err)
	}
	if got := string(headers); got != "/content-index.json\n  Access-Control-Allow-Origin: *\n" {
		t.Fatalf("headers = %q", got)
	}
}

func TestContentIndexPlugin_DisabledByDefault(t *testing.T) {
	dir := t.TempDir()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{OutputDir: dir, Extra: map[string]interface{}{}})
	if err := NewContentIndexPlugin().Write(m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "content-index.json")); !os.IsNotExist(err) {
		t.Fatalf("index was written while disabled: %v", err)
	}
}

func TestContentIndexPlugin_RejectsOutputTraversal(t *testing.T) {
	dir := t.TempDir()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{OutputDir: dir, Extra: map[string]interface{}{"content_index": map[string]interface{}{"enabled": true, "output": "../escape.json"}}})
	if err := NewContentIndexPlugin().Write(m); err == nil {
		t.Fatal("expected output traversal error")
	}
}

func TestContentIndexPlugin_RejectsHeadersSidecarOutput(t *testing.T) {
	dir := t.TempDir()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{OutputDir: dir, Extra: map[string]interface{}{"content_index": map[string]interface{}{"enabled": true, "output": "_headers"}}})
	if err := NewContentIndexPlugin().Write(m); err == nil {
		t.Fatal("expected _headers output to be rejected")
	}
}

func TestContentIndexPlugin_RejectsInvalidConfiguration(t *testing.T) {
	for name, config := range map[string]interface{}{
		"invalid table":      true,
		"invalid enabled":    map[string]interface{}{"enabled": "true"},
		"invalid output":     map[string]interface{}{"enabled": true, "output": 42},
		"fractional version": map[string]interface{}{"enabled": true, "schema_version": 1.5},
	} {
		t.Run(name, func(t *testing.T) {
			m := lifecycle.NewManager()
			m.SetConfig(&lifecycle.Config{Extra: map[string]interface{}{"content_index": config}})
			if err := NewContentIndexPlugin().Write(m); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestWriteContentIndexHeaders_NestedAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "nested", "content-index.json")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeContentIndexHeaders(dir, destination); err != nil {
		t.Fatal(err)
	}
	if err := writeContentIndexHeaders(dir, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "_headers"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "/nested/content-index.json\n  Access-Control-Allow-Origin: *\n" {
		t.Fatalf("headers = %q", got)
	}
}

func TestWriteContentIndexHeadersPreservesExistingRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_headers")
	initial := "/api/content-index.json\n  Cache-Control: max-age=60\n\n/content-index.json\n  Cache-Control: max-age=60\n\n/security.txt\n  X-Content-Type-Options: nosniff\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeContentIndexHeaders(dir, filepath.Join(dir, "content-index.json")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "/api/content-index.json\n  Cache-Control: max-age=60") || !strings.Contains(got, "/security.txt\n  X-Content-Type-Options: nosniff") {
		t.Fatalf("existing rules were not preserved: %q", got)
	}
	if !strings.Contains(got, "/content-index.json\n  Access-Control-Allow-Origin: *\n  Cache-Control: max-age=60") {
		t.Fatalf("artifact CORS rule was not added to its exact block: %q", got)
	}
	if err := os.WriteFile(path, []byte("/content-index.json\n  Access-Control-Allow-Origin: https://md.waylonwalker.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeContentIndexHeaders(dir, filepath.Join(dir, "content-index.json")); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "/content-index.json\n  Access-Control-Allow-Origin: https://md.waylonwalker.com\n" {
		t.Fatalf("restrictive CORS policy was overwritten: %q", got)
	}
}

func TestWriteContentIndexHeadersSkipsExternalArtifact(t *testing.T) {
	dir := t.TempDir()
	if err := writeContentIndexHeaders(dir, filepath.Join(t.TempDir(), "content-index.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_headers")); !os.IsNotExist(err) {
		t.Fatalf("external artifact created an output header file: %v", err)
	}
}

func TestWriteContentIndexHeadersRejectsPatternRoutes(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"*.json", ":asset.json", "unsafe\n/secret.json"} {
		if err := writeContentIndexHeaders(dir, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "_headers")); !os.IsNotExist(err) {
		t.Fatalf("pattern artifact created a broad CORS rule: %v", err)
	}
}

func TestContentIndexPlugin_RejectsCaseInsensitiveHeadersSidecarOutput(t *testing.T) {
	dir := t.TempDir()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{OutputDir: dir, Extra: map[string]interface{}{"content_index": map[string]interface{}{"enabled": true, "output": "_HEADERS"}}})
	if err := NewContentIndexPlugin().Write(m); err == nil {
		t.Fatal("expected case-insensitive _headers output to be rejected")
	}
}

func TestWriteContentIndexHeadersPreservesRestrictiveCRLFPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_headers")
	content := "/content-index.json\r\n  Access-Control-Allow-Origin: https://md.waylonwalker.com\r\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeContentIndexHeaders(dir, filepath.Join(dir, "content-index.json")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != content {
		t.Fatalf("restrictive CRLF policy was overwritten or duplicated: %q", got)
	}
}
