package importer

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Source type constants
const (
	SourceTypeRSS      = "rss"
	SourceTypeAtom     = "atom"
	SourceTypeJSONFeed = "jsonfeed"
)

// RSSImporter imports content from RSS/Atom feeds.
type RSSImporter struct {
	url string
}

// NewRSSImporter creates a new RSS importer for the given URL.
func NewRSSImporter(url string) (*RSSImporter, error) {
	if url == "" {
		return nil, fmt.Errorf("RSS feed URL is required")
	}
	return &RSSImporter{url: url}, nil
}

// Name returns the importer name.
func (r *RSSImporter) Name() string {
	return SourceTypeRSS
}

// SourceURL returns the feed URL.
func (r *RSSImporter) SourceURL() string {
	return r.url
}

// Import fetches and parses the RSS feed.
func (r *RSSImporter) Import(opts ImportOptions) ([]*ImportedPost, error) {
	// Fetch the feed with context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch RSS feed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS feed: %w", err)
	}

	// Try to parse as RSS 2.0 first, then Atom
	posts, err := r.parseRSS2(body, opts)
	if err != nil {
		posts, err = r.parseAtom(body, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed (tried RSS 2.0 and Atom): %w", err)
		}
	}

	return posts, nil
}

// RSS 2.0 structures
type rss2Feed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel rss2Channel `xml:"channel"`
}

type rss2Channel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Items       []rss2Item `xml:"item"`
}

type rss2Item struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Content     string   `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate     string   `xml:"pubDate"`
	GUID        string   `xml:"guid"`
	Author      string   `xml:"author"`
	Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Categories  []string `xml:"category"`
}

// Atom structures
type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Link      []atomLink `xml:"link"`
	ID        string     `xml:"id"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
	Summary   string     `xml:"summary"`
	Content   atomText   `xml:"content"`
	Author    atomAuthor `xml:"author"`
	Category  []atomCat  `xml:"category"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomCat struct {
	Term string `xml:"term,attr"`
}

func (r *RSSImporter) parseRSS2(data []byte, opts ImportOptions) ([]*ImportedPost, error) {
	var feed rss2Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	if feed.XMLName.Local != "rss" {
		return nil, fmt.Errorf("not an RSS 2.0 feed")
	}

	now := time.Now()
	posts := make([]*ImportedPost, 0, len(feed.Channel.Items))

	for i := range feed.Channel.Items {
		item := &feed.Channel.Items[i]
		// Parse publish date
		pubDate := parseRSSDate(item.PubDate)

		// Filter by date if specified
		if opts.Since != nil && pubDate.Before(*opts.Since) {
			continue
		}

		// Determine content - prefer content:encoded over description
		content := item.Content
		if content == "" {
			content = item.Description
		}

		// Determine author
		author := item.Author
		if author == "" {
			author = item.Creator
		}

		// Determine ID
		id := item.GUID
		if id == "" {
			id = item.Link
		}

		tags := item.Categories
		tags = append(tags, opts.AddTags...)

		post := &ImportedPost{
			ID:          id,
			SourceURL:   item.Link,
			SourceType:  SourceTypeRSS,
			Title:       item.Title,
			Content:     stripHTML(content),
			ContentHTML: content,
			Published:   pubDate,
			Imported:    now,
			Author:      author,
			Tags:        tags,
			Summary:     truncate(stripHTML(item.Description), 200),
			Slug:        generateSlug(item.Title),
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *RSSImporter) parseAtom(data []byte, opts ImportOptions) ([]*ImportedPost, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	now := time.Now()
	posts := make([]*ImportedPost, 0, len(feed.Entries))

	for i := range feed.Entries {
		entry := &feed.Entries[i]
		// Parse dates
		pubDate := parseAtomDate(entry.Published)
		if pubDate.IsZero() {
			pubDate = parseAtomDate(entry.Updated)
		}

		// Filter by date if specified
		if opts.Since != nil && pubDate.Before(*opts.Since) {
			continue
		}

		// Get link
		link := ""
		for _, l := range entry.Link {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}

		// Determine content
		content := entry.Content.Body
		if content == "" {
			content = entry.Summary
		}

		// Get tags
		var tags []string
		for _, cat := range entry.Category {
			if cat.Term != "" {
				tags = append(tags, cat.Term)
			}
		}
		tags = append(tags, opts.AddTags...)

		post := &ImportedPost{
			ID:          entry.ID,
			SourceURL:   link,
			SourceType:  SourceTypeAtom,
			Title:       entry.Title,
			Content:     stripHTML(content),
			ContentHTML: content,
			Published:   pubDate,
			Imported:    now,
			Author:      entry.Author.Name,
			Tags:        tags,
			Summary:     truncate(stripHTML(entry.Summary), 200),
			Slug:        generateSlug(entry.Title),
		}

		if entry.Updated != "" {
			updated := parseAtomDate(entry.Updated)
			if !updated.IsZero() {
				post.Updated = &updated
			}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// parseRSSDate parses common RSS date formats.
func parseRSSDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseAtomDate parses Atom date format (RFC3339).
func parseAtomDate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try without timezone
		if parsed, parseErr := time.Parse("2006-01-02T15:04:05", s); parseErr == nil {
			return parsed
		}
	}
	return t
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	// Simple regex-based HTML stripping
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")

	// Collapse whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen]) + "..."
}

// generateSlug creates a URL-safe slug from a title.
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters (except hyphens)
	reg := regexp.MustCompile(`[^a-z0-9\-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Collapse multiple hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 80 {
		slug = slug[:80]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}
