package blogroll

import "testing"

func TestParseFeedMetadata_RSSFeedTags(t *testing.T) {
	metadata := parseFeedMetadata(`
		<rss version="2.0">
		  <channel>
		    <title>Example Feed</title>
		    <description>Example Description</description>
		    <category>Go</category>
		    <category>Feeds</category>
		    <item>
		      <title>Post One</title>
		      <category>Ignored Entry Tag</category>
		    </item>
		  </channel>
		</rss>`)

	if metadata.FeedTitle != "Example Feed" {
		t.Fatalf("FeedTitle = %q, want %q", metadata.FeedTitle, "Example Feed")
	}
	if len(metadata.FeedTags) != 2 || metadata.FeedTags[0] != "Go" || metadata.FeedTags[1] != "Feeds" {
		t.Fatalf("FeedTags = %v, want [Go Feeds]", metadata.FeedTags)
	}
}

func TestParseFeedMetadata_AtomFeedTags(t *testing.T) {
	metadata := parseFeedMetadata(`
		<feed xmlns="http://www.w3.org/2005/Atom">
		  <title>Channel Feed</title>
		  <subtitle>Channel Description</subtitle>
		  <category term="Video" />
		  <category term="Tutorials" />
		  <entry>
		    <title>Ignored Entry</title>
		    <category term="Ignored Entry Tag" />
		  </entry>
		</feed>`)

	if metadata.FeedTitle != "Channel Feed" {
		t.Fatalf("FeedTitle = %q, want %q", metadata.FeedTitle, "Channel Feed")
	}
	if len(metadata.FeedTags) != 2 || metadata.FeedTags[0] != "Video" || metadata.FeedTags[1] != "Tutorials" {
		t.Fatalf("FeedTags = %v, want [Video Tutorials]", metadata.FeedTags)
	}
}

func TestParseHTMLMetadata_Tags(t *testing.T) {
	metadata := parseHTMLMetadata(`
		<html>
		  <head>
		    <title>Example Site</title>
		    <meta name="keywords" content="go, rss, feeds">
		    <meta property="article:tag" content="atom">
		    <meta property="article:tag" content="Go">
		  </head>
		</html>`, "https://example.com")

	if len(metadata.Tags) != 4 {
		t.Fatalf("Tags length = %d, want 4 (%v)", len(metadata.Tags), metadata.Tags)
	}
	if metadata.Tags[0] != "go" || metadata.Tags[1] != "rss" || metadata.Tags[2] != "feeds" || metadata.Tags[3] != "atom" {
		t.Fatalf("Tags = %v, want [go rss feeds atom]", metadata.Tags)
	}
}
