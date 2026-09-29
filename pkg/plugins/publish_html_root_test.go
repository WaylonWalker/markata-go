package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestPublishHTMLPlugin_ExplicitHomepageWritesAllEnabledFormats(t *testing.T) {
	outputDir := t.TempDir()
	config := &lifecycle.Config{
		OutputDir: outputDir,
		Extra: map[string]interface{}{
			"url":   "https://example.com",
			"title": "Example",
			"post_formats": models.PostFormatsConfig{
				Markdown: true,
				Text:     true,
				ANSI:     true,
				OG:       true,
			},
		},
	}
	title := "Homepage"
	post := &models.Post{
		Path:        "pages/post/index.md",
		Slug:        "",
		Title:       &title,
		Content:     "# Homepage\n",
		HTML:        "<html><body>Homepage</body></html>",
		ArticleHTML: "<h1>Homepage</h1>",
		Published:   true,
	}
	post.Set("_slug_explicit", true)

	plugin := NewPublishHTMLPlugin()
	if err := plugin.writePost(post, config, nil, createTestManager(t, config)); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	for _, relative := range []string{
		"index.html",
		"index.md",
		"index.txt",
		"index.ansi",
		filepath.Join("og", "index.html"),
	} {
		if info, err := os.Stat(filepath.Join(outputDir, relative)); err != nil || !info.Mode().IsRegular() {
			t.Errorf("expected root output %q: info=%v err=%v", relative, info, err)
		}
	}
}
