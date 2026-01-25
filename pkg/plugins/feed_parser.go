package plugins

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// parsedFeed holds parsed feed metadata.
type parsedFeed struct {
	Title       string
	Description string
	SiteURL     string
	ImageURL    string
	LastUpdated *time.Time
}

// parseFeedResponse parses an RSS or Atom feed from an HTTP response.
func parseFeedResponse(resp *http.Response) (*parsedFeed, []*models.ExternalEntry, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	// Try to detect feed type and parse
	content := string(body)
	if strings.Contains(content, "<feed") && strings.Contains(content, "xmlns=\"http://www.w3.org/2005/Atom\"") {
		return parseAtomFeed(body)
	}
	// Default to RSS
	return parseRSSFeed(body)
}

// rssChannel represents an RSS channel.
type rssChannel struct {
	XMLName     xml.Name  `xml:"channel"`
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	LastBuild   string    `xml:"lastBuildDate"`
	Image       *rssImage `xml:"image"`
	Items       []rssItem `xml:"item"`
}

// rssImage represents an RSS image.
type rssImage struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

// rssItem represents an RSS item.
type rssItem struct {
	Title          string          `xml:"title"`
	Link           string          `xml:"link"`
	Description    string          `xml:"description"`
	Content        string          `xml:"encoded"` // content:encoded
	PubDate        string          `xml:"pubDate"`
	GUID           string          `xml:"guid"`
	Author         string          `xml:"author"`
	Creator        string          `xml:"creator"` // dc:creator
	Categories     []string        `xml:"category"`
	Enclosure      *rssEnclosure   `xml:"enclosure"`
	MediaContent   *mediaContent   `xml:"http://search.yahoo.com/mrss/ content"`
	MediaThumbnail *mediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

// rssEnclosure represents an RSS enclosure (media).
type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length string `xml:"length,attr"`
}

// mediaContent represents media:content element from Media RSS.
type mediaContent struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
	Type   string `xml:"type,attr"`
}

// mediaThumbnail represents media:thumbnail element from Media RSS.
type mediaThumbnail struct {
	URL    string `xml:"url,attr"`
	Width  string `xml:"width,attr"`
	Height string `xml:"height,attr"`
}

// rssWrapper wraps the RSS document.
type rssWrapper struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

// parseRSSFeed parses an RSS 2.0 feed.
func parseRSSFeed(data []byte) (*parsedFeed, []*models.ExternalEntry, error) {
	var rss rssWrapper
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, nil, fmt.Errorf("unmarshal RSS: %w", err)
	}

	channel := rss.Channel

	feed := &parsedFeed{
		Title:       cleanString(channel.Title),
		Description: cleanString(channel.Description),
		SiteURL:     channel.Link,
	}

	if channel.Image != nil {
		feed.ImageURL = channel.Image.URL
	}

	// Parse last updated
	if channel.LastBuild != "" {
		if t := feedParseDate(channel.LastBuild); t != nil {
			feed.LastUpdated = t
		}
	} else if channel.PubDate != "" {
		if t := feedParseDate(channel.PubDate); t != nil {
			feed.LastUpdated = t
		}
	}

	// Parse items
	entries := make([]*models.ExternalEntry, 0, len(channel.Items))
	for i := range channel.Items {
		item := &channel.Items[i]
		entry := &models.ExternalEntry{
			Title:       cleanString(item.Title),
			URL:         item.Link,
			Description: cleanString(item.Description),
			Content:     item.Content,
			Categories:  item.Categories,
		}

		// Use GUID or link as ID
		if item.GUID != "" {
			entry.ID = item.GUID
		} else {
			entry.ID = item.Link
		}

		// Author (prefer dc:creator over author)
		if item.Creator != "" {
			entry.Author = item.Creator
		} else if item.Author != "" {
			entry.Author = item.Author
		}

		// Parse date
		if item.PubDate != "" {
			entry.Published = feedParseDate(item.PubDate)
		}

		// Estimate reading time
		content := item.Content
		if content == "" {
			content = item.Description
		}
		entry.ReadingTime = estimateReadingTime(content)

		// Extract entry image
		entry.ImageURL = extractRSSEntryImage(item)

		entries = append(entries, entry)
	}

	return feed, entries, nil
}

// atomFeed represents an Atom feed.
type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Links    []atomLink  `xml:"link"`
	Icon     string      `xml:"icon"`
	Logo     string      `xml:"logo"`
	Entries  []atomEntry `xml:"entry"`
}

// atomLink represents an Atom link.
type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
	Href string `xml:"href,attr"`
}

// atomEntry represents an Atom entry.
type atomEntry struct {
	ID        string         `xml:"id"`
	Title     string         `xml:"title"`
	Links     []atomLink     `xml:"link"`
	Published string         `xml:"published"`
	Updated   string         `xml:"updated"`
	Summary   string         `xml:"summary"`
	Content   atomContent    `xml:"content"`
	Author    *atomAuthor    `xml:"author"`
	Category  []atomCategory `xml:"category"`
	// Media RSS support for Atom
	MediaContent   *mediaContent   `xml:"http://search.yahoo.com/mrss/ content"`
	MediaThumbnail *mediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

// atomContent represents Atom content.
type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// atomAuthor represents an Atom author.
type atomAuthor struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
	URI   string `xml:"uri"`
}

// atomCategory represents an Atom category.
type atomCategory struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr"`
}

// parseAtomFeed parses an Atom 1.0 feed.
func parseAtomFeed(data []byte) (*parsedFeed, []*models.ExternalEntry, error) {
	var atom atomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		return nil, nil, fmt.Errorf("unmarshal Atom: %w", err)
	}

	feed := &parsedFeed{
		Title:       cleanString(atom.Title),
		Description: cleanString(atom.Subtitle),
	}

	// Find site URL (alternate link)
	for _, link := range atom.Links {
		if link.Rel == "alternate" || link.Rel == "" {
			feed.SiteURL = link.Href
			break
		}
	}

	// Use logo or icon
	if atom.Logo != "" {
		feed.ImageURL = atom.Logo
	} else if atom.Icon != "" {
		feed.ImageURL = atom.Icon
	}

	// Parse updated
	if atom.Updated != "" {
		feed.LastUpdated = feedParseDate(atom.Updated)
	}

	// Parse entries
	entries := make([]*models.ExternalEntry, 0, len(atom.Entries))
	for i := range atom.Entries {
		item := &atom.Entries[i]
		entry := &models.ExternalEntry{
			ID:          item.ID,
			Title:       cleanString(item.Title),
			Description: cleanString(item.Summary),
			Content:     item.Content.Value,
		}

		// Find entry URL (alternate link)
		for _, link := range item.Links {
			if link.Rel == "alternate" || link.Rel == "" {
				entry.URL = link.Href
				break
			}
		}

		// Author
		if item.Author != nil {
			entry.Author = item.Author.Name
		}

		// Published/Updated dates
		if item.Published != "" {
			entry.Published = feedParseDate(item.Published)
		}
		if item.Updated != "" {
			entry.Updated = feedParseDate(item.Updated)
		}
		// If no published date, use updated
		if entry.Published == nil && entry.Updated != nil {
			entry.Published = entry.Updated
		}

		// Categories
		for _, cat := range item.Category {
			if cat.Label != "" {
				entry.Categories = append(entry.Categories, cat.Label)
			} else if cat.Term != "" {
				entry.Categories = append(entry.Categories, cat.Term)
			}
		}

		// Estimate reading time
		entry.ReadingTime = estimateReadingTimeForAtomEntry(item)

		// Extract entry image
		entry.ImageURL = extractAtomEntryImage(item)

		entries = append(entries, entry)
	}

	return feed, entries, nil
}

// estimateReadingTimeForAtomEntry calculates the reading time for an Atom entry.
func estimateReadingTimeForAtomEntry(item *atomEntry) int {
	content := item.Content.Value
	if content == "" {
		content = item.Summary
	}
	return estimateReadingTime(content)
}

// feedParseDate tries to parse a date string in various formats.
func feedParseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	return nil
}

// cleanString removes leading/trailing whitespace and normalizes spaces.
func cleanString(s string) string {
	return strings.TrimSpace(s)
}

// estimateReadingTime estimates reading time in minutes.
func estimateReadingTime(content string) int {
	// Strip HTML and count words
	text := blogrollStripHTML(content)
	words := len(strings.Fields(text))

	// Average reading speed: 200 words per minute
	minutes := words / 200
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}

// extractRSSEntryImage extracts an image URL from an RSS item.
// It checks (in order): media:content, media:thumbnail, enclosure (if image type),
// and falls back to the first <img> in content/description HTML.
func extractRSSEntryImage(item *rssItem) string {
	// 1. Check media:content
	if item.MediaContent != nil && item.MediaContent.URL != "" {
		if isImageMedium(item.MediaContent.Medium, item.MediaContent.Type) {
			return item.MediaContent.URL
		}
	}

	// 2. Check media:thumbnail
	if item.MediaThumbnail != nil && item.MediaThumbnail.URL != "" {
		return item.MediaThumbnail.URL
	}

	// 3. Check enclosure (if image type)
	if item.Enclosure != nil && item.Enclosure.URL != "" {
		if isImageType(item.Enclosure.Type) {
			return item.Enclosure.URL
		}
	}

	// 4. Extract from content HTML
	content := item.Content
	if content == "" {
		content = item.Description
	}
	if imgURL := extractFirstImageFromHTML(content); imgURL != "" {
		return imgURL
	}

	return ""
}

// extractAtomEntryImage extracts an image URL from an Atom entry.
// It checks (in order): media:content, media:thumbnail, enclosure link,
// and falls back to the first <img> in content/summary HTML.
func extractAtomEntryImage(item *atomEntry) string {
	// 1. Check media:content
	if item.MediaContent != nil && item.MediaContent.URL != "" {
		if isImageMedium(item.MediaContent.Medium, item.MediaContent.Type) {
			return item.MediaContent.URL
		}
	}

	// 2. Check media:thumbnail
	if item.MediaThumbnail != nil && item.MediaThumbnail.URL != "" {
		return item.MediaThumbnail.URL
	}

	// 3. Check for enclosure link (rel="enclosure" with image type)
	for _, link := range item.Links {
		if link.Rel == "enclosure" && isImageType(link.Type) {
			return link.Href
		}
	}

	// 4. Extract from content HTML
	content := item.Content.Value
	if content == "" {
		content = item.Summary
	}
	if imgURL := extractFirstImageFromHTML(content); imgURL != "" {
		return imgURL
	}

	return ""
}

// isImageMedium checks if the media content represents an image.
func isImageMedium(medium, mimeType string) bool {
	if medium == "image" {
		return true
	}
	return isImageType(mimeType)
}

// isImageType checks if the MIME type is an image type.
func isImageType(mimeType string) bool {
	mimeType = strings.ToLower(mimeType)
	return strings.HasPrefix(mimeType, "image/")
}

// imgSrcPattern matches src attribute in <img> tags.
var imgSrcPattern = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

// extractFirstImageFromHTML extracts the first image URL from HTML content.
func extractFirstImageFromHTML(htmlContent string) string {
	matches := imgSrcPattern.FindStringSubmatch(htmlContent)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
