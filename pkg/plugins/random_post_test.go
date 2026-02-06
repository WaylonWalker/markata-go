package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestRandomPostPlugin_Name(t *testing.T) {
	p := NewRandomPostPlugin()
	if got := p.Name(); got != "random_post" {
		t.Errorf("Name() = %q, want %q", got, "random_post")
	}
}

func TestRandomPostPlugin_DisabledByDefault(t *testing.T) {
	p := NewRandomPostPlugin()
	m := lifecycle.NewManager()
	outDir := t.TempDir()
	m.SetConfig(&lifecycle.Config{OutputDir: outDir, Extra: map[string]interface{}{}})
	m.SetPosts([]*models.Post{{Slug: "a", Href: "/a/", Published: true}})

	if err := p.Configure(m); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "random", "index.html")); err == nil {
		t.Fatalf("expected no output when disabled")
	}
}

func TestRandomPostPlugin_WritesIndexAndOptionalJSON(t *testing.T) {
	tests := []struct {
		name          string
		emitPostsJSON bool
	}{
		{"index_only", false},
		{"with_posts_json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewRandomPostPlugin()
			m := lifecycle.NewManager()
			outDir := t.TempDir()
			m.SetConfig(&lifecycle.Config{
				OutputDir: outDir,
				Extra: map[string]interface{}{
					"random_post": map[string]any{
						"enabled":         true,
						"path":            "random",
						"emit_posts_json": tt.emitPostsJSON,
						"exclude_tags":    []any{"private", "draft"},
					},
				},
			})
			m.SetPosts([]*models.Post{
				{Slug: "a", Href: "/a/", Published: true},
				{Slug: "b", Href: "/b/", Published: true, Draft: true},
				{Slug: "c", Href: "/c/", Published: true, Private: true},
				{Slug: "d", Href: "/d/", Published: true, Skip: true},
				{Slug: "e", Href: "/e/", Published: false},
				{Slug: "f", Href: "/f/", Published: true, Tags: []string{"draft"}},
			})

			if err := p.Configure(m); err != nil {
				t.Fatalf("Configure() error = %v", err)
			}
			if err := p.Write(m); err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			indexPath := filepath.Join(outDir, "random", "index.html")
			b, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", indexPath, err)
			}
			content := string(b)

			if !strings.Contains(content, "/a/") {
				t.Errorf("index.html should include eligible href /a/")
			}
			if strings.Contains(content, "/b/") || strings.Contains(content, "/c/") || strings.Contains(content, "/d/") || strings.Contains(content, "/e/") {
				t.Errorf("index.html should not include ineligible hrefs")
			}
			if strings.Contains(content, "/f/") {
				t.Errorf("index.html should not include tag-excluded href /f/")
			}

			jsonPath := filepath.Join(outDir, "random", "posts.json")
			_, jsonErr := os.Stat(jsonPath)
			if tt.emitPostsJSON {
				if jsonErr != nil {
					t.Fatalf("expected %s to exist: %v", jsonPath, jsonErr)
				}
			} else {
				if jsonErr == nil {
					t.Fatalf("did not expect %s to exist", jsonPath)
				}
			}
		})
	}
}

func TestRandomPostPlugin_DoesNotClobberExistingOutput(t *testing.T) {
	p := NewRandomPostPlugin()
	m := lifecycle.NewManager()
	outDir := t.TempDir()

	indexPath := filepath.Join(outDir, "random", "index.html")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("existing"), 0o644); err != nil { //nolint:gosec // test fixture
		t.Fatalf("WriteFile() error = %v", err)
	}

	m.SetConfig(&lifecycle.Config{
		OutputDir: outDir,
		Extra: map[string]interface{}{
			"random_post": map[string]any{"enabled": true},
		},
	})
	m.SetPosts([]*models.Post{{Slug: "a", Href: "/a/", Published: true}})

	if err := p.Configure(m); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if err := p.Write(m); err == nil {
		t.Fatalf("expected Write() to error due to existing output")
	}
}
