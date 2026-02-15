package plugins

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestLinkAvatars_Render_LocalModeInjectsAndCaches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write([]byte("fake-png")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	config := &lifecycle.Config{
		OutputDir: outputDir,
		Extra: map[string]interface{}{
			"models_config": &models.Config{URL: "https://mysite.test"},
			"link_avatars": map[string]any{
				"enabled":  true,
				"mode":     "local",
				"service":  "custom",
				"template": server.URL + "/favicon/{host}.png",
				"size":     16,
			},
		},
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	post := &models.Post{Path: "post.md", ArticleHTML: `<p><a href="https://example.com/path">Example</a></p>`}
	m.SetPosts([]*models.Post{post})

	plugin := NewLinkAvatarsPlugin()
	if err := plugin.Configure(m); err != nil {
		t.Fatalf("configure error: %v", err)
	}

	if err := plugin.Render(m); err != nil {
		t.Fatalf("render error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "has-avatar") {
		t.Errorf("expected injected avatar class, got %s", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "data-favicon=") {
		t.Errorf("expected data-favicon attribute, got %s", post.ArticleHTML)
	}

	iconPath := filepath.Join(outputDir, "assets", "markata", "link-avatars", "example.com.png")
	if _, err := os.Stat(iconPath); err != nil {
		t.Fatalf("expected cached icon at %s: %v", iconPath, err)
	}
}

func TestLinkAvatars_ConfigureHostedRequiresBaseURL(t *testing.T) {
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"models_config": &models.Config{URL: "https://mysite.test"},
			"link_avatars": map[string]any{
				"enabled": true,
				"mode":    "hosted",
			},
		},
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	plugin := NewLinkAvatarsPlugin()
	if err := plugin.Configure(m); err == nil {
		t.Fatal("expected error when hosted_base_url is missing")
	}
}

func TestLinkAvatars_LocalMode_WithURLPathPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write([]byte("fake-png")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	outputDir := t.TempDir()
	config := &lifecycle.Config{
		OutputDir: outputDir,
		Extra: map[string]interface{}{
			"models_config": &models.Config{URL: "https://mysite.test/blog"},
			"link_avatars": map[string]any{
				"enabled":  true,
				"mode":     "local",
				"service":  "custom",
				"template": server.URL + "/favicon/{host}.png",
				"size":     16,
			},
		},
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	post := &models.Post{Path: "post.md", ArticleHTML: `<p><a href="https://example.com/path">Example</a></p>`}
	m.SetPosts([]*models.Post{post})

	plugin := NewLinkAvatarsPlugin()
	if err := plugin.Configure(m); err != nil {
		t.Fatalf("configure error: %v", err)
	}

	if err := plugin.Render(m); err != nil {
		t.Fatalf("render error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "has-avatar") {
		t.Errorf("expected injected avatar class, got %s", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `data-favicon="/assets/markata/link-avatars/`) {
		t.Errorf("expected data-favicon with absolute path, got %s", post.ArticleHTML)
	}

	iconPath := filepath.Join(outputDir, "assets", "markata", "link-avatars", "example.com.png")
	if _, err := os.Stat(iconPath); err != nil {
		t.Fatalf("expected cached icon at %s: %v", iconPath, err)
	}
}
