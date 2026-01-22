package plugins

import (
	"strings"
	"testing"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// =============================================================================
// WikilinksPlugin Tests based on tests.yaml
// =============================================================================

func TestWikilinksPlugin_Name(t *testing.T) {
	p := NewWikilinksPlugin()
	if p.Name() != "wikilinks" {
		t.Errorf("expected name 'wikilinks', got %q", p.Name())
	}
}

func TestWikilinksPlugin_Configure(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()
	if err := p.Configure(m); err != nil {
		t.Errorf("Configure returned error: %v", err)
	}
}

func TestWikilinksPlugin_ConfigureWarnBroken(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()
	config := m.Config()
	config.Extra = map[string]interface{}{
		"wikilinks_warn_broken": false,
	}

	if err := p.Configure(m); err != nil {
		t.Errorf("Configure returned error: %v", err)
	}

	// Verify setting was applied (indirectly through behavior)
	if p.warnOnBroken != false {
		t.Error("expected warnOnBroken to be false after configuration")
	}
}

func TestWikilinksPlugin_BasicWikilink(t *testing.T) {
	// Test case from tests.yaml: "basic wikilink"
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Other Post"
	targetPost := &models.Post{
		Slug:  "other-post",
		Title: &targetTitle,
		Href:  "/other-post/",
	}
	sourcePost := &models.Post{
		Content: "Check out [[other-post]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	expected := `<a href="/other-post/" class="wikilink">Other Post</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_WikilinkWithCustomText(t *testing.T) {
	// Test case from tests.yaml: "wikilink with custom text"
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Other Post"
	targetPost := &models.Post{
		Slug:  "other-post",
		Title: &targetTitle,
		Href:  "/other-post/",
	}
	sourcePost := &models.Post{
		Content: "See [[other-post|this article]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	expected := `<a href="/other-post/" class="wikilink">this article</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_WikilinkToMissingPost(t *testing.T) {
	// Test case from tests.yaml: "wikilink to missing post"
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Content: "Link to [[nonexistent]]",
		Slug:    "source-post",
		Extra:   make(map[string]interface{}),
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Content should be unchanged - wikilink left as is
	if !strings.Contains(source.Content, "[[nonexistent]]") {
		t.Errorf("expected [[nonexistent]] to remain in content, got %q", source.Content)
	}

	// Check for warning
	warnings, ok := source.Extra["wikilink_warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Error("expected wikilink warning for missing post")
	} else if !strings.Contains(warnings[0], "broken wikilink") {
		t.Errorf("expected broken wikilink warning, got %q", warnings[0])
	}
}

func TestWikilinksPlugin_CaseInsensitiveLookup(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "My Post"
	targetPost := &models.Post{
		Slug:  "my-post",
		Title: &targetTitle,
		Href:  "/my-post/",
	}
	sourcePost := &models.Post{
		Content: "Check out [[MY-POST]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	expected := `<a href="/my-post/" class="wikilink">My Post</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected case-insensitive match %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_SlugWithSpaces(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Hello World"
	targetPost := &models.Post{
		Slug:  "hello-world",
		Title: &targetTitle,
		Href:  "/hello-world/",
	}
	sourcePost := &models.Post{
		Content: "See [[hello world]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	expected := `<a href="/hello-world/" class="wikilink">Hello World</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected space to hyphen conversion %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_MultipleWikilinks(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	post1Title := "Post One"
	post2Title := "Post Two"
	post1 := &models.Post{Slug: "post-one", Title: &post1Title, Href: "/post-one/"}
	post2 := &models.Post{Slug: "post-two", Title: &post2Title, Href: "/post-two/"}
	sourcePost := &models.Post{
		Content: "See [[post-one]] and [[post-two]] for more",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{post1, post2, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	if !strings.Contains(source.Content, `<a href="/post-one/" class="wikilink">Post One</a>`) {
		t.Errorf("expected first wikilink to be converted, got %q", source.Content)
	}
	if !strings.Contains(source.Content, `<a href="/post-two/" class="wikilink">Post Two</a>`) {
		t.Errorf("expected second wikilink to be converted, got %q", source.Content)
	}
}

func TestWikilinksPlugin_SkippedPost(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Target"
	targetPost := &models.Post{Slug: "target", Title: &targetTitle, Href: "/target/"}
	sourcePost := &models.Post{
		Content: "See [[target]]",
		Slug:    "source-post",
		Skip:    true,
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Content should be unchanged for skipped posts
	if !strings.Contains(source.Content, "[[target]]") {
		t.Errorf("expected skipped post content to be unchanged, got %q", source.Content)
	}
}

func TestWikilinksPlugin_EmptyContent(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Content: "",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	if posts[0].Content != "" {
		t.Errorf("expected empty content to remain empty, got %q", posts[0].Content)
	}
}

func TestWikilinksPlugin_PostWithoutTitle(t *testing.T) {
	// When target post has no title, should use slug
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetPost := &models.Post{
		Slug: "my-post",
		Href: "/my-post/",
		// No Title set
	}
	sourcePost := &models.Post{
		Content: "See [[my-post]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Should use slug as display text
	expected := `<a href="/my-post/" class="wikilink">my-post</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected slug as display text %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_HTMLEscaping(t *testing.T) {
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Test & Demo"
	targetPost := &models.Post{
		Slug:  "test-demo",
		Title: &targetTitle,
		Href:  "/test-demo/",
	}
	sourcePost := &models.Post{
		Content: "See [[test-demo]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Ampersand should be escaped
	if !strings.Contains(source.Content, "&amp;") {
		t.Errorf("expected HTML-escaped content, got %q", source.Content)
	}
}

func TestWikilinksPlugin_NoHref(t *testing.T) {
	// When target post has no href, should generate from slug
	p := NewWikilinksPlugin()
	m := lifecycle.NewManager()

	targetTitle := "Target"
	targetPost := &models.Post{
		Slug:  "target",
		Title: &targetTitle,
		// No Href set
	}
	sourcePost := &models.Post{
		Content: "See [[target]]",
		Slug:    "source-post",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Should generate href from slug
	expected := `<a href="/target/" class="wikilink">Target</a>`
	if !strings.Contains(source.Content, expected) {
		t.Errorf("expected generated href %q in content, got %q", expected, source.Content)
	}
}

func TestWikilinksPlugin_SetWarnOnBroken(t *testing.T) {
	p := NewWikilinksPlugin()

	// Default should be true
	if p.warnOnBroken != true {
		t.Error("expected default warnOnBroken to be true")
	}

	p.SetWarnOnBroken(false)
	if p.warnOnBroken != false {
		t.Error("expected warnOnBroken to be false after SetWarnOnBroken(false)")
	}

	p.SetWarnOnBroken(true)
	if p.warnOnBroken != true {
		t.Error("expected warnOnBroken to be true after SetWarnOnBroken(true)")
	}
}

func TestWikilinksPlugin_MissingPostNoWarningWhenDisabled(t *testing.T) {
	p := NewWikilinksPlugin()
	p.SetWarnOnBroken(false)

	m := lifecycle.NewManager()
	sourcePost := &models.Post{
		Content: "Link to [[nonexistent]]",
		Slug:    "source-post",
		Extra:   make(map[string]interface{}),
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform error: %v", err)
	}

	posts := m.Posts()
	var source *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			source = post
			break
		}
	}

	// Should not have warning when disabled
	warnings, ok := source.Extra["wikilink_warnings"].([]string)
	if ok && len(warnings) > 0 {
		t.Error("expected no warnings when warnOnBroken is disabled")
	}
}

// Interface compliance tests
func TestWikilinksPlugin_Interfaces(t *testing.T) {
	p := NewWikilinksPlugin()

	var _ lifecycle.Plugin = p
	var _ lifecycle.ConfigurePlugin = p
	var _ lifecycle.TransformPlugin = p
}

// TestWikilinksPlugin_PreservesCodeBlocks tests that wikilinks inside code blocks are not transformed.
func TestWikilinksPlugin_PreservesCodeBlocks(t *testing.T) {
	p := NewWikilinksPlugin()

	m := lifecycle.NewManager()
	config := m.Config()
	config.Extra = map[string]interface{}{
		"wikilinks_warn_broken": false,
	}

	// Create target posts
	title := "Getting Started"
	target := &models.Post{
		Slug:  "getting-started",
		Title: &title,
		Href:  "/getting-started/",
	}

	// Create source post with wikilinks inside and outside code blocks
	source := &models.Post{
		Slug: "source-post",
		Content: `Here is a link: [[getting-started]]

` + "```markdown" + `
This [[getting-started]] should NOT be converted.
` + "```" + `

And another link: [[getting-started|custom text]]

` + "~~~" + `
This [[getting-started]] in tilde fence should also be preserved.
` + "~~~" + `

Final link: [[getting-started]]`,
		Extra: make(map[string]interface{}),
	}

	m.SetPosts([]*models.Post{source, target})

	// Run the plugin
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	var result *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			result = post
			break
		}
	}

	// Check that wikilinks outside code blocks are converted
	if !strings.Contains(result.Content, `<a href="/getting-started/" class="wikilink">Getting Started</a>`) {
		t.Errorf("expected wikilinks outside code blocks to be converted, got %q", result.Content)
	}

	// Check that wikilinks inside backtick code blocks are preserved
	if !strings.Contains(result.Content, "This [[getting-started]] should NOT be converted.") {
		t.Errorf("expected wikilinks inside backtick code blocks to be preserved, got %q", result.Content)
	}

	// Check that wikilinks inside tilde code blocks are preserved
	if !strings.Contains(result.Content, "This [[getting-started]] in tilde fence should also be preserved.") {
		t.Errorf("expected wikilinks inside tilde code blocks to be preserved, got %q", result.Content)
	}

	// Check custom text link is converted
	if !strings.Contains(result.Content, `<a href="/getting-started/" class="wikilink">custom text</a>`) {
		t.Errorf("expected custom text wikilink to be converted, got %q", result.Content)
	}
}
