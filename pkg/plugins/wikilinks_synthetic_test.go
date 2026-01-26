package plugins

import (
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// TestWikilinks_SyntheticBlogrollPosts tests that wikilinks resolve to synthetic
// blogroll and reader posts registered during Configure stage.
func TestWikilinks_SyntheticBlogrollPosts(t *testing.T) {
	// Create manager with blogroll config
	m := lifecycle.NewManager()
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"wikilinks_warn_broken": false,
			"blogroll": models.BlogrollConfig{
				Enabled: true,
			},
		},
	}
	m.SetConfig(config)

	// Create blogroll plugin and run Configure to register synthetic posts
	blogrollPlugin := NewBlogrollPlugin()
	if err := blogrollPlugin.Configure(m); err != nil {
		t.Fatalf("BlogrollPlugin.Configure() error = %v", err)
	}

	// Create a test post with wikilinks to blogroll and reader
	testPost := &models.Post{
		Slug:    "test-post",
		Content: "Check out my [[ blogroll ]] and [[ reader ]] pages!",
	}
	m.AddPost(testPost)

	// Run wikilinks plugin
	wikilinksPlugin := NewWikilinksPlugin()
	if err := wikilinksPlugin.Configure(m); err != nil {
		t.Fatalf("WikilinksPlugin.Configure() error = %v", err)
	}
	if err := wikilinksPlugin.Transform(m); err != nil {
		t.Fatalf("WikilinksPlugin.Transform() error = %v", err)
	}

	// Verify blogroll wikilink was converted
	if !strings.Contains(testPost.Content, `<a href="/blogroll/" class="wikilink"`) {
		t.Errorf("expected blogroll wikilink to be converted, got: %s", testPost.Content)
	}
	if !strings.Contains(testPost.Content, `data-title="Blogroll"`) {
		t.Errorf("expected blogroll title in wikilink, got: %s", testPost.Content)
	}

	// Verify reader wikilink was converted
	if !strings.Contains(testPost.Content, `<a href="/reader/" class="wikilink"`) {
		t.Errorf("expected reader wikilink to be converted, got: %s", testPost.Content)
	}
	if !strings.Contains(testPost.Content, `data-title="Reader"`) {
		t.Errorf("expected reader title in wikilink, got: %s", testPost.Content)
	}

	// Verify original text is not present
	if strings.Contains(testPost.Content, "[[ blogroll ]]") {
		t.Errorf("expected [[ blogroll ]] to be replaced, got: %s", testPost.Content)
	}
	if strings.Contains(testPost.Content, "[[ reader ]]") {
		t.Errorf("expected [[ reader ]] to be replaced, got: %s", testPost.Content)
	}
}

// TestWikilinks_SyntheticBlogrollCustomSlugs tests that wikilinks resolve to
// blogroll synthetic posts with custom slugs.
func TestWikilinks_SyntheticBlogrollCustomSlugs(t *testing.T) {
	// Create manager with custom blogroll slugs
	m := lifecycle.NewManager()
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"wikilinks_warn_broken": false,
			"blogroll": models.BlogrollConfig{
				Enabled:      true,
				BlogrollSlug: "my-feeds",
				ReaderSlug:   "feed-reader",
			},
		},
	}
	m.SetConfig(config)

	// Create blogroll plugin and run Configure
	blogrollPlugin := NewBlogrollPlugin()
	if err := blogrollPlugin.Configure(m); err != nil {
		t.Fatalf("BlogrollPlugin.Configure() error = %v", err)
	}

	// Create test post with wikilinks using custom slugs
	testPost := &models.Post{
		Slug:    "test-custom-slugs",
		Content: "Visit my [[ my-feeds ]] or [[ feed-reader ]]!",
	}
	m.AddPost(testPost)

	// Run wikilinks plugin
	wikilinksPlugin := NewWikilinksPlugin()
	if err := wikilinksPlugin.Configure(m); err != nil {
		t.Fatalf("WikilinksPlugin.Configure() error = %v", err)
	}
	if err := wikilinksPlugin.Transform(m); err != nil {
		t.Fatalf("WikilinksPlugin.Transform() error = %v", err)
	}

	// Verify custom blogroll slug wikilink
	if !strings.Contains(testPost.Content, `<a href="/my-feeds/" class="wikilink"`) {
		t.Errorf("expected my-feeds wikilink to be converted, got: %s", testPost.Content)
	}

	// Verify custom reader slug wikilink
	if !strings.Contains(testPost.Content, `<a href="/feed-reader/" class="wikilink"`) {
		t.Errorf("expected feed-reader wikilink to be converted, got: %s", testPost.Content)
	}

	// Verify no raw wikilinks remain
	if strings.Contains(testPost.Content, "[[") || strings.Contains(testPost.Content, "]]") {
		t.Errorf("expected all wikilinks to be converted, got: %s", testPost.Content)
	}
}

// TestWikilinks_SyntheticFeedPages tests that wikilinks resolve to synthetic
// feed posts registered during Configure stage.
func TestWikilinks_SyntheticFeedPages(t *testing.T) {
	// Create feed configs
	feedConfigs := []models.FeedConfig{
		{
			Title: "Python Posts",
			Slug:  "python",
		},
		{
			Title: "Go Posts",
			Slug:  "golang",
		},
	}

	// Create manager
	m := lifecycle.NewManager()
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"wikilinks_warn_broken": false,
		},
	}
	m.SetConfig(config)
	m.Cache().Set("feed_configs", feedConfigs)

	// Create publish_feeds plugin and run Configure
	feedsPlugin := NewPublishFeedsPlugin()
	if err := feedsPlugin.Configure(m); err != nil {
		t.Fatalf("PublishFeedsPlugin.Configure() error = %v", err)
	}

	// Create test post with wikilinks to feed pages
	testPost := &models.Post{
		Slug:    "test-feeds",
		Content: "See [[ python ]] and [[ golang ]] feeds!",
	}
	m.AddPost(testPost)

	// Run wikilinks plugin
	wikilinksPlugin := NewWikilinksPlugin()
	if err := wikilinksPlugin.Configure(m); err != nil {
		t.Fatalf("WikilinksPlugin.Configure() error = %v", err)
	}
	if err := wikilinksPlugin.Transform(m); err != nil {
		t.Fatalf("WikilinksPlugin.Transform() error = %v", err)
	}

	// Verify python feed wikilink
	if !strings.Contains(testPost.Content, `<a href="/python/" class="wikilink"`) {
		t.Errorf("expected python feed wikilink to be converted, got: %s", testPost.Content)
	}
	if !strings.Contains(testPost.Content, `data-title="Python Posts"`) {
		t.Errorf("expected Python Posts title in wikilink, got: %s", testPost.Content)
	}

	// Verify golang feed wikilink
	if !strings.Contains(testPost.Content, `<a href="/golang/" class="wikilink"`) {
		t.Errorf("expected golang feed wikilink to be converted, got: %s", testPost.Content)
	}
	if !strings.Contains(testPost.Content, `data-title="Go Posts"`) {
		t.Errorf("expected Go Posts title in wikilink, got: %s", testPost.Content)
	}

	// Verify no raw wikilinks remain
	if strings.Contains(testPost.Content, "[[ python ]]") || strings.Contains(testPost.Content, "[[ golang ]]") {
		t.Errorf("expected wikilinks to be converted, got: %s", testPost.Content)
	}
}

// TestWikilinks_AllSyntheticPosts is a comprehensive integration test that verifies
// wikilinks work for all types of synthetic posts (blogroll, reader, and feeds)
// in a single test post.
func TestWikilinks_AllSyntheticPosts(t *testing.T) {
	// Create manager with full config
	m := lifecycle.NewManager()
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"wikilinks_warn_broken": false,
			"blogroll": models.BlogrollConfig{
				Enabled: true,
			},
		},
	}
	m.SetConfig(config)

	// Set up feed configs
	feedConfigs := []models.FeedConfig{
		{Title: "Tech", Slug: "tech"},
		{Title: "News", Slug: "news"},
	}
	m.Cache().Set("feed_configs", feedConfigs)

	// Register all synthetic posts
	blogrollPlugin := NewBlogrollPlugin()
	if err := blogrollPlugin.Configure(m); err != nil {
		t.Fatalf("BlogrollPlugin.Configure() error = %v", err)
	}

	feedsPlugin := NewPublishFeedsPlugin()
	if err := feedsPlugin.Configure(m); err != nil {
		t.Fatalf("PublishFeedsPlugin.Configure() error = %v", err)
	}

	// Create test post with wikilinks to all synthetic pages
	testPost := &models.Post{
		Slug: "comprehensive-test",
		Content: `Welcome! Check out:
- My [[ blogroll ]] for blogs I follow
- The [[ reader ]] for latest posts
- [[ tech ]] feed for tech articles
- [[ news ]] feed for news
`,
	}
	m.AddPost(testPost)

	// Run wikilinks
	wikilinksPlugin := NewWikilinksPlugin()
	if err := wikilinksPlugin.Configure(m); err != nil {
		t.Fatalf("WikilinksPlugin.Configure() error = %v", err)
	}
	if err := wikilinksPlugin.Transform(m); err != nil {
		t.Fatalf("WikilinksPlugin.Transform() error = %v", err)
	}

	// Verify all wikilinks were converted
	expectedLinks := []struct {
		href  string
		title string
	}{
		{`href="/blogroll/"`, `data-title="Blogroll"`},
		{`href="/reader/"`, `data-title="Reader"`},
		{`href="/tech/"`, `data-title="Tech"`},
		{`href="/news/"`, `data-title="News"`},
	}

	for _, expected := range expectedLinks {
		if !strings.Contains(testPost.Content, expected.href) {
			t.Errorf("expected %s in content, got: %s", expected.href, testPost.Content)
		}
		if !strings.Contains(testPost.Content, expected.title) {
			t.Errorf("expected %s in content, got: %s", expected.title, testPost.Content)
		}
	}

	// Verify no raw wikilinks remain
	if strings.Contains(testPost.Content, "[[") || strings.Contains(testPost.Content, "]]") {
		t.Errorf("expected all wikilinks to be converted, found raw wikilinks in: %s", testPost.Content)
	}

	// Verify all converted links have wikilink class
	linkCount := strings.Count(testPost.Content, `class="wikilink"`)
	if linkCount != 4 {
		t.Errorf("expected 4 wikilinks, found %d", linkCount)
	}
}

// TestWikilinks_BlogrollDisabled verifies that when blogroll is disabled,
// wikilinks to blogroll/reader are not resolved (treated as broken links).
func TestWikilinks_BlogrollDisabled(t *testing.T) {
	// Create manager with blogroll disabled
	m := lifecycle.NewManager()
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"wikilinks_warn_broken": true,
			"blogroll": models.BlogrollConfig{
				Enabled: false,
			},
		},
	}
	m.SetConfig(config)

	// Blogroll plugin Configure should not register posts when disabled
	blogrollPlugin := NewBlogrollPlugin()
	if err := blogrollPlugin.Configure(m); err != nil {
		t.Fatalf("BlogrollPlugin.Configure() error = %v", err)
	}

	// Create test post with wikilink to blogroll
	testPost := &models.Post{
		Slug:    "test-disabled",
		Content: "My [[ blogroll ]] page.",
	}
	m.AddPost(testPost)

	// Run wikilinks
	wikilinksPlugin := NewWikilinksPlugin()
	if err := wikilinksPlugin.Configure(m); err != nil {
		t.Fatalf("WikilinksPlugin.Configure() error = %v", err)
	}
	if err := wikilinksPlugin.Transform(m); err != nil {
		t.Fatalf("WikilinksPlugin.Transform() error = %v", err)
	}

	// Verify wikilink was NOT converted (blogroll disabled)
	if strings.Contains(testPost.Content, `<a href="/blogroll/"`) {
		t.Errorf("expected wikilink to NOT be converted when blogroll disabled, got: %s", testPost.Content)
	}

	// Verify raw wikilink remains
	if !strings.Contains(testPost.Content, "[[ blogroll ]]") {
		t.Errorf("expected raw wikilink when blogroll disabled, got: %s", testPost.Content)
	}

	// Verify warning was added
	warnings, ok := testPost.Extra["wikilink_warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Error("expected wikilink warning for broken link")
	} else if !strings.Contains(warnings[0], "broken wikilink") {
		t.Errorf("expected broken wikilink warning, got: %v", warnings)
	}
}
