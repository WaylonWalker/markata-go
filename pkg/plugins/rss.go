// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// RSS represents an RSS 2.0 feed.
type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr,omitempty"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents the channel element in an RSS feed.
type RSSChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language,omitempty"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	AtomLink      *AtomLink `xml:"atom:link,omitempty"`
	Items         []RSSItem `xml:"item"`
}

// AtomLink represents an atom:link element for RSS feed self-reference.
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// RSSItem represents an item element in an RSS feed.
type RSSItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	PubDate     string  `xml:"pubDate,omitempty"`
	GUID        RSSGUID `xml:"guid"`
	Author      string  `xml:"author,omitempty"`
}

// RSSGUID represents a globally unique identifier for an RSS item.
type RSSGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

// GenerateRSS generates an RSS 2.0 feed from a lifecycle.Feed.
func GenerateRSS(feed *lifecycle.Feed, config *lifecycle.Config) (string, error) {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)
	siteDesc := getSiteDescription(config)
	feedURL := siteURL + "/" + feed.Path + "/rss.xml"

	// Use feed title if available, otherwise use site title
	title := feed.Title
	if title == "" {
		title = siteTitle
	}

	rss := RSS{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: RSSChannel{
			Title:       title,
			Link:        siteURL,
			Description: siteDesc,
			Language:    "en-us",
			AtomLink: &AtomLink{
				Href: feedURL,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: make([]RSSItem, 0, len(feed.Posts)),
		},
	}

	// Set last build date
	if len(feed.Posts) > 0 {
		rss.Channel.LastBuildDate = time.Now().Format(time.RFC1123Z)
	}

	// Add items
	for _, post := range feed.Posts {
		item := postToRSSItem(post, siteURL)
		rss.Channel.Items = append(rss.Channel.Items, item)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal RSS: %w", err)
	}

	return xml.Header + string(output), nil
}

// GenerateRSSFromFeedConfig generates an RSS 2.0 feed from a FeedConfig.
func GenerateRSSFromFeedConfig(fc *models.FeedConfig, config *lifecycle.Config) (string, error) {
	feed := &lifecycle.Feed{
		Name:  fc.Slug,
		Title: fc.Title,
		Posts: fc.Posts,
		Path:  fc.Slug,
	}
	return GenerateRSS(feed, config)
}

// postToRSSItem converts a Post to an RSSItem.
func postToRSSItem(post *models.Post, siteURL string) RSSItem {
	// Build permalink
	permalink := siteURL + post.Href

	// Get title
	title := ""
	if post.Title != nil {
		title = *post.Title
	} else {
		title = post.Slug
	}

	// Get description (use post description or truncated content)
	var description string
	switch {
	case post.Description != nil:
		description = escapeXML(*post.Description)
	case post.ArticleHTML != "":
		// Use rendered HTML as description (truncated)
		description = escapeXML(truncateHTML(post.ArticleHTML, 500))
	default:
		description = escapeXML(truncateText(post.Content, 500))
	}

	// Get publication date
	pubDate := ""
	if post.Date != nil {
		pubDate = post.Date.Format(time.RFC1123Z)
	}

	return RSSItem{
		Title:       escapeXML(title),
		Link:        permalink,
		Description: description,
		PubDate:     pubDate,
		GUID: RSSGUID{
			Value:       permalink,
			IsPermaLink: true,
		},
	}
}

// getSiteURL retrieves the site URL from config.
func getSiteURL(config *lifecycle.Config) string {
	if config.Extra != nil {
		if url, ok := config.Extra["url"].(string); ok {
			return strings.TrimSuffix(url, "/")
		}
	}
	return "https://example.com"
}

// getSiteTitle retrieves the site title from config.
func getSiteTitle(config *lifecycle.Config) string {
	if config.Extra != nil {
		if title, ok := config.Extra["title"].(string); ok {
			return title
		}
	}
	return "Blog"
}

// getSiteDescription retrieves the site description from config.
func getSiteDescription(config *lifecycle.Config) string {
	if config.Extra != nil {
		if desc, ok := config.Extra["description"].(string); ok {
			return desc
		}
	}
	return ""
}

// escapeXML escapes special XML characters in content.
func escapeXML(s string) string {
	// xml.EscapeString handles &, <, >, ", '
	var buf strings.Builder
	_ = xml.EscapeText(&buf, []byte(s)) //nolint:errcheck // writing to strings.Builder never fails
	return buf.String()
}

// truncateText truncates text to a maximum length, adding ellipsis if truncated.
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find last space before maxLen
	lastSpace := strings.LastIndex(s[:maxLen], " ")
	if lastSpace > 0 {
		return s[:lastSpace] + "..."
	}
	return s[:maxLen] + "..."
}

// truncateHTML attempts to truncate HTML content, stripping tags for the preview.
func truncateHTML(html string, maxLen int) string {
	// Simple tag stripper - remove all HTML tags
	var result strings.Builder
	inTag := false

	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	return truncateText(result.String(), maxLen)
}
