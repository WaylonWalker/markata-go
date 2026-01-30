// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// AtomFeed represents an Atom feed.
type AtomFeed struct {
	XMLName xml.Name       `xml:"feed"`
	Xmlns   string         `xml:"xmlns,attr"`
	Title   string         `xml:"title"`
	ID      string         `xml:"id"`
	Updated string         `xml:"updated,omitempty"`
	Links   []AtomFeedLink `xml:"link"`
	Author  *AtomAuthor    `xml:"author,omitempty"`
	Entries []AtomEntry    `xml:"entry"`
}

// AtomFeedLink represents a link element in an Atom feed.
type AtomFeedLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

// AtomAuthor represents an author element in an Atom feed.
type AtomAuthor struct {
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
	URI   string `xml:"uri,omitempty"`
}

// AtomEntry represents an entry element in an Atom feed.
type AtomEntry struct {
	Title     string         `xml:"title"`
	ID        string         `xml:"id"`
	Updated   string         `xml:"updated,omitempty"`
	Published string         `xml:"published,omitempty"`
	Links     []AtomFeedLink `xml:"link"`
	Summary   *AtomContent   `xml:"summary,omitempty"`
	Content   *AtomContent   `xml:"content,omitempty"`
	Author    *AtomAuthor    `xml:"author,omitempty"`
}

// AtomContent represents content with a type attribute.
type AtomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// GenerateAtom generates an Atom feed from a lifecycle.Feed.
func GenerateAtom(feed *lifecycle.Feed, config *lifecycle.Config) (string, error) {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)
	author := getSiteAuthor(config)
	feedPath := feed.Path
	if feedPath == "" {
		feedPath = DefaultFeedPath
	}
	feedURL := siteURL + "/" + feedPath + "/atom.xml"

	// Use feed title if available, otherwise use site title
	title := feed.Title
	if title == "" {
		title = siteTitle
	}

	// Determine updated time based on most recent post date (deterministic)
	var updatedTime *time.Time
	for _, post := range feed.Posts {
		if post.Date == nil {
			continue
		}
		if updatedTime == nil || post.Date.After(*updatedTime) {
			t := *post.Date
			updatedTime = &t
		}
	}

	atomFeed := AtomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   title,
		ID:      feedURL,
		Updated: "",
		Links: []AtomFeedLink{
			{Href: siteURL, Rel: "alternate", Type: "text/html"},
			{Href: feedURL, Rel: "self", Type: "application/atom+xml"},
		},
		Entries: make([]AtomEntry, 0, len(feed.Posts)),
	}

	if updatedTime != nil {
		atomFeed.Updated = updatedTime.Format(time.RFC3339)
	}

	// Add author if available
	if author != "" {
		atomFeed.Author = &AtomAuthor{Name: author}
	}

	// Add entries
	for _, post := range feed.Posts {
		// Skip private posts from Atom feed
		if post.Private {
			continue
		}
		entry := postToAtomEntry(post, siteURL)
		atomFeed.Entries = append(atomFeed.Entries, entry)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(atomFeed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Atom feed: %w", err)
	}

	// Add XSL stylesheet processing instruction for human-readable display in browsers
	xslPI := `<?xml-stylesheet href="/atom.xsl" type="text/xsl"?>` + "\n"
	return xml.Header + xslPI + string(output), nil
}

// GenerateAtomFromFeedConfig generates an Atom feed from a FeedConfig.
func GenerateAtomFromFeedConfig(fc *models.FeedConfig, config *lifecycle.Config) (string, error) {
	feed := &lifecycle.Feed{
		Name:  fc.Slug,
		Title: fc.Title,
		Posts: fc.Posts,
		Path:  fc.Slug,
	}
	return GenerateAtom(feed, config)
}

// postToAtomEntry converts a Post to an AtomEntry.
func postToAtomEntry(post *models.Post, siteURL string) AtomEntry {
	// Build permalink
	permalink := siteURL + post.Href

	// Get title
	title := ""
	if post.Title != nil {
		title = *post.Title
	} else {
		title = post.Slug
	}

	// Determine dates
	var updated, published string
	if post.Date != nil {
		dateStr := post.Date.Format(time.RFC3339)
		updated = dateStr
		published = dateStr
	}

	entry := AtomEntry{
		Title:     title,
		ID:        permalink,
		Updated:   updated,
		Published: published,
		Links: []AtomFeedLink{
			{Href: permalink, Rel: "alternate", Type: "text/html"},
		},
	}

	// Add summary
	if post.Description != nil && *post.Description != "" {
		entry.Summary = &AtomContent{
			Type:  "text",
			Value: *post.Description,
		}
	}

	// Add content
	if post.ArticleHTML != "" {
		entry.Content = &AtomContent{
			Type:  "html",
			Value: post.ArticleHTML,
		}
	} else if post.Content != "" {
		entry.Content = &AtomContent{
			Type:  "text",
			Value: post.Content,
		}
	}

	return entry
}

// getSiteAuthor retrieves the site author from config.
func getSiteAuthor(config *lifecycle.Config) string {
	if config.Extra != nil {
		if author, ok := config.Extra["author"].(string); ok {
			return author
		}
	}
	return ""
}
