package plugins

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestParseRSS2Feed_Basic(t *testing.T) {
	rssData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Example Blog</title>
    <link>https://example.com</link>
    <description>A test blog</description>
    <language>en-us</language>
    <item>
      <title>Test Post</title>
      <link>https://example.com/test-post</link>
      <description>This is a test post</description>
      <pubDate>Mon, 01 Jan 2024 12:00:00 GMT</pubDate>
      <guid>https://example.com/test-post</guid>
    </item>
  </channel>
</rss>`

	feed, entries, err := parseRSS2Feed([]byte(rssData))
	if err != nil {
		t.Fatalf("parseRSS2Feed() error = %v", err)
	}

	if feed.Title != "Example Blog" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Example Blog")
	}
	if feed.SiteURL != "https://example.com" {
		t.Errorf("feed.SiteURL = %q, want %q", feed.SiteURL, "https://example.com")
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Title != "Test Post" {
		t.Errorf("entry.Title = %q, want %q", entry.Title, "Test Post")
	}
	if entry.URL != "https://example.com/test-post" {
		t.Errorf("entry.URL = %q, want %q", entry.URL, "https://example.com/test-post")
	}
	if entry.Description != "This is a test post" {
		t.Errorf("entry.Description = %q, want %q", entry.Description, "This is a test post")
	}
}

func TestParseAtomFeed_Basic(t *testing.T) {
	atomData := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Atom Feed</title>
  <subtitle>Atom test</subtitle>
  <link href="https://example.com" rel="alternate" />
  <entry>
    <title>Atom Test Post</title>
    <link href="https://example.com/atom-post" rel="alternate" />
    <id>tag:example.com,2024:1</id>
    <published>2024-01-01T12:00:00Z</published>
    <updated>2024-01-01T12:00:00Z</updated>
    <summary>An atom test post</summary>
    <author>
      <name>John Doe</name>
    </author>
  </entry>
</feed>`

	feed, entries, err := parseAtomFeed([]byte(atomData))
	if err != nil {
		t.Fatalf("parseAtomFeed() error = %v", err)
	}

	if feed.Title != "Example Atom Feed" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Example Atom Feed")
	}
	if feed.SiteURL != "https://example.com" {
		t.Errorf("feed.SiteURL = %q, want %q", feed.SiteURL, "https://example.com")
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Title != "Atom Test Post" {
		t.Errorf("entry.Title = %q, want %q", entry.Title, "Atom Test Post")
	}
	if entry.URL != "https://example.com/atom-post" {
		t.Errorf("entry.URL = %q, want %q", entry.URL, "https://example.com/atom-post")
	}
	if entry.Author != "John Doe" {
		t.Errorf("entry.Author = %q, want %q", entry.Author, "John Doe")
	}
}

func TestParseBlogrollFeedResponse_RSS(t *testing.T) {
	rssData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS</title>
    <link>https://example.com</link>
    <description>Test RSS feed</description>
    <item>
      <title>RSS Item</title>
      <link>https://example.com/item</link>
      <description>RSS description</description>
    </item>
  </channel>
</rss>`

	resp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(rssData))),
	}

	feed, entries, err := parseBlogrollFeedResponse(resp)
	if err != nil {
		t.Fatalf("parseBlogrollFeedResponse() error = %v", err)
	}

	if feed.Title != "Test RSS" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Test RSS")
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}

func TestParseBlogrollFeedResponse_Atom(t *testing.T) {
	atomData := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom</title>
  <link href="https://example.com" rel="alternate" />
  <entry>
    <title>Atom Entry</title>
    <link href="https://example.com/entry" rel="alternate" />
    <id>1</id>
    <updated>2024-01-01T12:00:00Z</updated>
  </entry>
</feed>`

	resp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(atomData))),
	}

	feed, entries, err := parseBlogrollFeedResponse(resp)
	if err != nil {
		t.Fatalf("parseBlogrollFeedResponse() error = %v", err)
	}

	if feed.Title != "Test Atom" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Test Atom")
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}

func TestParseRSSDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{"RFC1123Z", "Mon, 01 Jan 2024 12:00:00 +0000", false},
		{"RFC1123", "Mon, 01 Jan 2024 12:00:00 GMT", false},
		{"RFC822Z", "01 Jan 24 12:00 +0000", false},
		{"empty", "", true},
		{"invalid", "not a date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRSSDate(tt.dateStr)
			if tt.wantErr && !got.IsZero() {
				t.Errorf("parseRSSDate(%q) should return zero time", tt.dateStr)
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseRSSDate(%q) returned zero time", tt.dateStr)
			}
		})
	}
}

func TestParseAtomDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{"RFC3339", "2024-01-01T12:00:00Z", false},
		{"RFC3339 with offset", "2024-01-01T12:00:00+05:30", false},
		{"empty", "", true},
		{"invalid", "not a date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAtomDate(tt.dateStr)
			if tt.wantErr && !got.IsZero() {
				t.Errorf("parseAtomDate(%q) should return zero time", tt.dateStr)
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseAtomDate(%q) returned zero time", tt.dateStr)
			}
		})
	}
}

func TestStripBlogrollHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"with tags", "<p>hello <strong>world</strong></p>", "hello world"},
		{"with entities", "&lt;tag&gt; &amp; text", "& text"}, // HTML entities are decoded, then tags stripped
		{"complex", "<div><p>Test &amp; <a href=\"#\">link</a></p></div>", "Test & link"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBlogrollHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripBlogrollHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
