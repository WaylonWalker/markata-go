package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JSONFeedImporter imports content from JSON Feed format.
type JSONFeedImporter struct {
	url string
}

// NewJSONFeedImporter creates a new JSON Feed importer for the given URL.
func NewJSONFeedImporter(url string) (*JSONFeedImporter, error) {
	if url == "" {
		return nil, fmt.Errorf("JSON Feed URL is required")
	}
	return &JSONFeedImporter{url: url}, nil
}

// Name returns the importer name.
func (j *JSONFeedImporter) Name() string {
	return SourceTypeJSONFeed
}

// SourceURL returns the feed URL.
func (j *JSONFeedImporter) SourceURL() string {
	return j.url
}

// JSON Feed structures (https://jsonfeed.org/version/1.1)
type jsonFeed struct {
	Version     string         `json:"version"`
	Title       string         `json:"title"`
	HomePageURL string         `json:"home_page_url"`
	FeedURL     string         `json:"feed_url"`
	Description string         `json:"description"`
	Authors     []jsonAuthor   `json:"authors"`
	Items       []jsonFeedItem `json:"items"`
}

type jsonAuthor struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type jsonFeedItem struct {
	ID            string       `json:"id"`
	URL           string       `json:"url"`
	Title         string       `json:"title"`
	ContentHTML   string       `json:"content_html"`
	ContentText   string       `json:"content_text"`
	Summary       string       `json:"summary"`
	DatePublished string       `json:"date_published"`
	DateModified  string       `json:"date_modified"`
	Authors       []jsonAuthor `json:"authors"`
	Tags          []string     `json:"tags"`
}

// Import fetches and parses the JSON Feed.
func (j *JSONFeedImporter) Import(opts ImportOptions) ([]*ImportedPost, error) {
	// Fetch the feed with context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JSON Feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JSON Feed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON Feed: %w", err)
	}

	var feed jsonFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON Feed: %w", err)
	}

	now := time.Now()
	posts := make([]*ImportedPost, 0, len(feed.Items))

	for i := range feed.Items {
		item := &feed.Items[i]
		// Parse publish date
		pubDate := parseJSONFeedDate(item.DatePublished)

		// Filter by date if specified
		if opts.Since != nil && pubDate.Before(*opts.Since) {
			continue
		}

		// Determine content
		content := item.ContentText
		if content == "" && item.ContentHTML != "" {
			content = stripHTML(item.ContentHTML)
		}

		// Determine author
		author := ""
		if len(item.Authors) > 0 {
			author = item.Authors[0].Name
		} else if len(feed.Authors) > 0 {
			author = feed.Authors[0].Name
		}

		// Combine tags
		tags := item.Tags
		tags = append(tags, opts.AddTags...)

		post := &ImportedPost{
			ID:          item.ID,
			SourceURL:   item.URL,
			SourceType:  SourceTypeJSONFeed,
			Title:       item.Title,
			Content:     content,
			ContentHTML: item.ContentHTML,
			Published:   pubDate,
			Imported:    now,
			Author:      author,
			Tags:        tags,
			Summary:     truncate(item.Summary, 200),
			Slug:        generateSlug(item.Title),
		}

		if item.DateModified != "" {
			modified := parseJSONFeedDate(item.DateModified)
			if !modified.IsZero() {
				post.Updated = &modified
			}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// parseJSONFeedDate parses JSON Feed date format (RFC3339).
func parseJSONFeedDate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try without timezone
		if parsed, parseErr := time.Parse("2006-01-02T15:04:05", s); parseErr == nil {
			return parsed
		}
	}
	return t
}
