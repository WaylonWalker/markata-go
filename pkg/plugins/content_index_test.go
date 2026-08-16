package plugins

import (
	"os"
	"path/filepath"
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
	private := models.NewPost("posts/private.md")
	private.Private, private.Published = true, true
	draft := models.NewPost("posts/draft.md")
	draft.Draft = true
	m.SetPosts([]*models.Post{private, draft, public})
	m.SetFeeds([]*lifecycle.Feed{{Name: "blog", Posts: []*models.Post{public, private}}, {Name: "archive", Posts: []*models.Post{public}}})
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
	if index.DocumentCount != 1 || index.Documents[0].Path != public.Path {
		t.Fatalf("privacy filtering failed: %#v", index)
	}
	if got := index.Documents[0].Feeds; len(got) != 2 || got[0] != "archive" || got[1] != "blog" {
		t.Fatalf("feeds = %v", got)
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
