package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestCollectPinsUsesPublishedPublicLinkPosts(t *testing.T) {
	newer := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	older := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	newerTitle := "A useful project"
	olderTitle := "A good article"
	privateTitle := "Private link"
	description := "Worth keeping around."

	pins := collectPins([]*models.Post{
		{
			Slug:        "article",
			Href:        "/article/",
			Title:       &olderTitle,
			Description: &description,
			Date:        &older,
			Published:   true,
			Extra: map[string]interface{}{
				"link":  "https://example.com/article",
				"image": "https://example.com/card.jpg",
			},
		},
		{
			Slug:      "project",
			Href:      "/project/",
			Title:     &newerTitle,
			Date:      &newer,
			Published: true,
			Extra: map[string]interface{}{
				"link": "https://www.example.org/project",
			},
		},
		{
			Slug:      "private",
			Title:     &privateTitle,
			Published: true,
			Private:   true,
			Extra:     map[string]interface{}{"link": "https://example.net/private"},
		},
		{
			Slug:      "not-a-link-post",
			Published: true,
			Extra:     map[string]interface{}{},
		},
	})

	if len(pins) != 2 {
		t.Fatalf("pin count = %d, want 2", len(pins))
	}
	if pins[0].Title != newerTitle || pins[0].Domain != "example.org" {
		t.Fatalf("first pin = %#v, want newest example.org link", pins[0])
	}
	if pins[1].Title != olderTitle || pins[1].Description != description || pins[1].Image == "" {
		t.Fatalf("second pin = %#v, want article metadata", pins[1])
	}
}

func TestAutoFeedsPluginWriteGeneratesPinsPage(t *testing.T) {
	outDir := t.TempDir()
	title := "Saved article"
	description := "A short note about why this link matters."
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{OutputDir: outDir, Extra: map[string]interface{}{}})
	manager.SetPosts([]*models.Post{
		{
			Slug:        "saved-article",
			Href:        "/saved-article/",
			Title:       &title,
			Description: &description,
			Published:   true,
			Extra: map[string]interface{}{
				"link":        "https://example.com/story",
				"cover_image": "https://example.com/story.jpg",
			},
		},
	})

	if err := NewAutoFeedsPlugin().Write(manager); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	outputPath := filepath.Join(outDir, "pins", "index.html")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}
	page := string(content)
	for _, want := range []string{
		"Saved article",
		"A short note about why this link matters.",
		"https://example.com/story",
		"https://example.com/story.jpg",
		`href="/saved-article/"`,
		"example.com",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("pins page missing %q", want)
		}
	}
}

func TestAutoFeedsPluginWriteSkipsPinsPageWithoutLinkPosts(t *testing.T) {
	outDir := t.TempDir()
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{OutputDir: outDir, Extra: map[string]interface{}{}})
	manager.SetPosts([]*models.Post{{Slug: "regular", Published: true, Extra: map[string]interface{}{}}})

	if err := NewAutoFeedsPlugin().Write(manager); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "pins", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("pins page should not exist without link posts, stat error = %v", err)
	}
}
