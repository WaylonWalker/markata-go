package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestSplitFeedRenderablePosts_PartitionsVisibleAndRenderablePosts(t *testing.T) {
	t.Parallel()

	title := "Visible"
	posts := []*models.Post{
		{Slug: "visible", Title: &title, Published: true, Content: "body", ArticleHTML: "<p>body</p>"},
		{Slug: "title-only", Title: &title, Published: true},
		{Slug: "draft", Title: &title, Draft: true, Published: false, Content: "draft"},
		{Slug: "private", Title: &title, Private: true, Content: "private"},
	}

	pagePosts, outputPosts := splitFeedRenderablePosts(posts, false)
	if got := len(pagePosts); got != 2 {
		t.Fatalf("pagePosts len = %d, want 2", got)
	}
	if got := len(outputPosts); got != 1 {
		t.Fatalf("outputPosts len = %d, want 1", got)
	}
	if pagePosts[0].Slug != "visible" || pagePosts[1].Slug != "title-only" {
		t.Fatalf("unexpected pagePosts order: %#v", []string{pagePosts[0].Slug, pagePosts[1].Slug})
	}
	if outputPosts[0].Slug != "visible" {
		t.Fatalf("unexpected outputPosts: %q", outputPosts[0].Slug)
	}

	pagePosts, outputPosts = splitFeedRenderablePosts(posts, true)
	if got := len(pagePosts); got != 3 {
		t.Fatalf("pagePosts len with private = %d, want 3", got)
	}
	if got := len(outputPosts); got != 2 {
		t.Fatalf("outputPosts len with private = %d, want 2", got)
	}
}
