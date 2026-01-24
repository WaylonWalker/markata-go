package importer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRSSImporter_Import(t *testing.T) {
	// Sample RSS 2.0 feed
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test Blog</title>
    <link>https://example.com</link>
    <description>A test blog</description>
    <item>
      <title>First Post</title>
      <link>https://example.com/first-post</link>
      <description>This is the first post.</description>
      <content:encoded><![CDATA[<p>This is the <strong>full content</strong> of the first post.</p>]]></content:encoded>
      <pubDate>Mon, 20 Jan 2025 10:00:00 +0000</pubDate>
      <guid>https://example.com/first-post</guid>
      <category>tech</category>
      <category>golang</category>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/second-post</link>
      <description>This is the second post.</description>
      <pubDate>Tue, 21 Jan 2025 12:00:00 +0000</pubDate>
      <guid>https://example.com/second-post</guid>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		if _, err := w.Write([]byte(rssFeed)); err != nil {
			return
		}
	}))
	defer server.Close()

	imp, err := NewRSSImporter(server.URL)
	if err != nil {
		t.Fatalf("NewRSSImporter() error = %v", err)
	}

	if imp.Name() != SourceTypeRSS {
		t.Errorf("Name() = %q, want %q", imp.Name(), SourceTypeRSS)
	}

	posts, err := imp.Import(ImportOptions{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("Import() returned %d posts, want 2", len(posts))
	}

	// Check first post
	post := posts[0]
	if post.Title != "First Post" {
		t.Errorf("Title = %q, want %q", post.Title, "First Post")
	}
	if post.SourceURL != "https://example.com/first-post" {
		t.Errorf("SourceURL = %q, want %q", post.SourceURL, "https://example.com/first-post")
	}
	if post.SourceType != SourceTypeRSS {
		t.Errorf("SourceType = %q, want %q", post.SourceType, SourceTypeRSS)
	}
	if post.Slug != "first-post" {
		t.Errorf("Slug = %q, want %q", post.Slug, "first-post")
	}
	if len(post.Tags) != 2 || post.Tags[0] != "tech" || post.Tags[1] != "golang" {
		t.Errorf("Tags = %v, want [tech golang]", post.Tags)
	}
}

func TestRSSImporter_ImportWithSinceFilter(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Blog</title>
    <item>
      <title>Old Post</title>
      <link>https://example.com/old</link>
      <pubDate>Mon, 01 Jan 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>New Post</title>
      <link>https://example.com/new</link>
      <pubDate>Fri, 20 Jan 2025 10:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(rssFeed)); err != nil {
			return
		}
	}))
	defer server.Close()

	imp, err := NewRSSImporter(server.URL)
	if err != nil {
		t.Fatalf("NewRSSImporter() error = %v", err)
	}

	// Filter to posts after Jan 1, 2025
	since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	posts, err := imp.Import(ImportOptions{Since: &since})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("Import() returned %d posts, want 1 (filtered)", len(posts))
	}
	if posts[0].Title != "New Post" {
		t.Errorf("Expected 'New Post', got %q", posts[0].Title)
	}
}

func TestAtomImporter_Import(t *testing.T) {
	atomFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Blog</title>
  <link href="https://example.com"/>
  <entry>
    <title>Atom Post</title>
    <link href="https://example.com/atom-post"/>
    <id>https://example.com/atom-post</id>
    <published>2025-01-21T10:00:00Z</published>
    <updated>2025-01-21T11:00:00Z</updated>
    <summary>A short summary</summary>
    <content type="html"><![CDATA[<p>Full atom content</p>]]></content>
    <author>
      <name>John Doe</name>
    </author>
    <category term="atom"/>
    <category term="test"/>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		if _, err := w.Write([]byte(atomFeed)); err != nil {
			return
		}
	}))
	defer server.Close()

	imp, err := NewRSSImporter(server.URL)
	if err != nil {
		t.Fatalf("NewRSSImporter() error = %v", err)
	}
	posts, err := imp.Import(ImportOptions{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("Import() returned %d posts, want 1", len(posts))
	}

	post := posts[0]
	if post.Title != "Atom Post" {
		t.Errorf("Title = %q, want %q", post.Title, "Atom Post")
	}
	if post.SourceType != SourceTypeAtom {
		t.Errorf("SourceType = %q, want %q", post.SourceType, SourceTypeAtom)
	}
	if post.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", post.Author, "John Doe")
	}
	if post.Updated == nil {
		t.Error("Updated should not be nil")
	}
}

func TestJSONFeedImporter_Import(t *testing.T) {
	jsonFeed := `{
  "version": "https://jsonfeed.org/version/1.1",
  "title": "Test JSON Feed",
  "home_page_url": "https://example.com",
  "items": [
    {
      "id": "1",
      "url": "https://example.com/json-post",
      "title": "JSON Post",
      "content_text": "This is plain text content.",
      "date_published": "2025-01-22T10:00:00Z",
      "tags": ["json", "feed"]
    },
    {
      "id": "2",
      "url": "https://example.com/html-post",
      "title": "HTML Post",
      "content_html": "<p>This is <strong>HTML</strong> content.</p>",
      "date_published": "2025-01-23T10:00:00Z",
      "authors": [{"name": "Jane Doe"}]
    }
  ]
}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/feed+json")
		if _, err := w.Write([]byte(jsonFeed)); err != nil {
			return
		}
	}))
	defer server.Close()

	imp, err := NewJSONFeedImporter(server.URL)
	if err != nil {
		t.Fatalf("NewJSONFeedImporter() error = %v", err)
	}

	if imp.Name() != SourceTypeJSONFeed {
		t.Errorf("Name() = %q, want %q", imp.Name(), SourceTypeJSONFeed)
	}

	posts, err := imp.Import(ImportOptions{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("Import() returned %d posts, want 2", len(posts))
	}

	// Check first post (plain text)
	post := posts[0]
	if post.Title != "JSON Post" {
		t.Errorf("Title = %q, want %q", post.Title, "JSON Post")
	}
	if post.Content != "This is plain text content." {
		t.Errorf("Content = %q, want %q", post.Content, "This is plain text content.")
	}
	if post.SourceType != SourceTypeJSONFeed {
		t.Errorf("SourceType = %q, want %q", post.SourceType, SourceTypeJSONFeed)
	}

	// Check second post (HTML content)
	post2 := posts[1]
	if post2.Author != "Jane Doe" {
		t.Errorf("Author = %q, want %q", post2.Author, "Jane Doe")
	}
	// Content should be stripped HTML
	if post2.Content != "This is HTML content." {
		t.Errorf("Content = %q, want %q", post2.Content, "This is HTML content.")
	}
}

func TestWriter_Write(t *testing.T) {
	tmpDir := t.TempDir()

	posts := []*ImportedPost{
		{
			Title:      "Test Post",
			Slug:       "test-post",
			SourceURL:  "https://example.com/test",
			SourceType: "rss",
			Content:    "This is the content.",
			Published:  time.Date(2025, 1, 20, 10, 0, 0, 0, time.UTC),
			Imported:   time.Now(),
			Tags:       []string{"test", "example"},
		},
	}

	writer := NewWriter(tmpDir)
	result, err := writer.Write(posts, false)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if result.Written != 1 {
		t.Errorf("Written = %d, want 1", result.Written)
	}

	// Check file was created
	expectedPath := filepath.Join(tmpDir, "test-post.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", expectedPath)
	}

	// Check file content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, `title: "Test Post"`) {
		t.Error("File should contain title")
	}
	if !contains(contentStr, `source_url: "https://example.com/test"`) {
		t.Error("File should contain source_url")
	}
	if !contains(contentStr, "source_type: rss") {
		t.Error("File should contain source_type")
	}
	if !contains(contentStr, "- imported") {
		t.Error("File should contain imported tag")
	}
	if !contains(contentStr, "This is the content.") {
		t.Error("File should contain content")
	}
}

func TestWriter_WriteDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	posts := []*ImportedPost{
		{
			Title:      "Dry Run Post",
			Slug:       "dry-run-post",
			SourceURL:  "https://example.com/dry",
			SourceType: "rss",
			Content:    "Content",
			Published:  time.Now(),
			Imported:   time.Now(),
		},
	}

	writer := NewWriter(tmpDir)
	result, err := writer.Write(posts, true) // dry-run
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if result.Written != 1 {
		t.Errorf("Written = %d, want 1", result.Written)
	}

	// File should NOT be created in dry-run
	expectedPath := filepath.Join(tmpDir, "dry-run-post.md")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("File %s should NOT exist in dry-run mode", expectedPath)
	}
}

func TestWriter_WriteSkipsExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file
	existingPath := filepath.Join(tmpDir, "existing-post.md")
	if err := os.WriteFile(existingPath, []byte("existing content"), 0o600); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	posts := []*ImportedPost{
		{
			Title:      "Existing Post",
			Slug:       "existing-post",
			SourceURL:  "https://example.com/existing",
			SourceType: "rss",
			Content:    "New content",
			Published:  time.Now(),
			Imported:   time.Now(),
		},
	}

	writer := NewWriter(tmpDir)
	result, err := writer.Write(posts, false)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if result.Written != 0 {
		t.Errorf("Written = %d, want 0", result.Written)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}

	// File content should remain unchanged
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "existing content" {
		t.Error("Existing file should not be overwritten")
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"This is a Test!", "this-is-a-test"},
		{"  Multiple   Spaces  ", "multiple-spaces"},
		{"Special@#$Characters", "specialcharacters"},
		{"Numbers 123 and Words", "numbers-123-and-words"},
		{"", ""},
		{"Already-Slugified", "already-slugified"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := generateSlug(tt.title)
			if got != tt.want {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<strong>Bold</strong> text", "Bold text"},
		{"<a href='#'>Link</a>", "Link"},
		{"No tags here", "No tags here"},
		{"&amp; &lt; &gt;", "& < >"},
		{"<p>Multiple</p><p>Paragraphs</p>", "MultipleParagraphs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
