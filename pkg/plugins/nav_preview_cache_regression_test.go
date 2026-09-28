package plugins

import (
	"encoding/json"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// TestArchiveNavPreviewStatsInvalidateUnrelatedPostCache characterizes the
// production failure behind #1337. The archive nav preview contains aggregate
// word/read-time stats for every post. Editing one post therefore changes the
// global nav-preview hash, and SetNavPreviewHash clears every cached post even
// when the build-cache dependency pass only invalidated the edited source.
//
// Keep this test while changing the invalidation design: the desired end state
// is for the unrelated post to remain cacheable when only volatile archive
// preview statistics changed.
func TestArchiveNavPreviewStatsInvalidateUnrelatedPostCache(t *testing.T) {
	config := models.NewConfig()
	config.Nav = []models.NavItem{{Label: "archive", URL: "/archive/"}}

	changed := &models.Post{
		Path:      "pages/changed.md",
		Slug:      "changed",
		Href:      "/changed/",
		Published: true,
		InputHash: "changed-hash",
		Template:  "post.html",
		Extra:     map[string]interface{}{"word_count": 100, "reading_time": 1},
	}
	unrelated := &models.Post{
		Path:      "pages/unrelated.md",
		Slug:      "unrelated",
		Href:      "/unrelated/",
		Published: true,
		InputHash: "unrelated-hash",
		Template:  "post.html",
		Extra:     map[string]interface{}{"word_count": 200, "reading_time": 2},
	}
	posts := []*models.Post{changed, unrelated}
	feeds := []models.FeedConfig{{Slug: "archive", Title: "Archive", Posts: posts}}

	cache := buildcache.New(t.TempDir())
	cache.SetNavPreviewHash(navPreviewHashForTest(t, config, posts, feeds))
	cache.MarkRebuiltWithSlug(changed.Path, changed.Slug, changed.InputHash, "output/changed/index.html", changed.Template)
	cache.MarkRebuiltWithSlug(unrelated.Path, unrelated.Slug, unrelated.InputHash, "output/unrelated/index.html", unrelated.Template)

	if reason := cache.ReasonForRebuild(unrelated.Path, unrelated.InputHash, unrelated.Template); reason != buildcache.RebuildReasonNone {
		t.Fatalf("unrelated post unexpectedly missed cache before preview change: %q", reason)
	}

	changed.Extra["word_count"] = 101
	if changedHash := cache.SetNavPreviewHash(navPreviewHashForTest(t, config, posts, feeds)); !changedHash {
		t.Fatal("archive preview hash did not change after aggregate word count changed")
	}

	if reason := cache.ReasonForRebuild(unrelated.Path, unrelated.InputHash, unrelated.Template); reason != buildcache.RebuildReasonMissingEntry {
		t.Fatalf("unrelated post rebuild reason = %q, want %q to reproduce global invalidation", reason, buildcache.RebuildReasonMissingEntry)
	}
}

func navPreviewHashForTest(t *testing.T, config *models.Config, posts []*models.Post, feeds []models.FeedConfig) string {
	t.Helper()
	previews := buildNavPreviews(config, posts, feeds, models.BlogrollConfig{}, RandomPostConfig{})
	encoded, err := json.Marshal(previews)
	if err != nil {
		t.Fatalf("marshal nav previews: %v", err)
	}
	return buildcache.ContentHash(string(encoded))
}
