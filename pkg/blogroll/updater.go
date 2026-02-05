// Package blogroll provides functionality for managing blogroll metadata.
package blogroll

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Updater handles fetching and extracting metadata from external sites.
type Updater struct {
	client  *http.Client
	timeout time.Duration
}

// NewUpdater creates a new Updater with the given timeout.
func NewUpdater(timeout time.Duration) *Updater {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Updater{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Metadata represents extracted metadata from a website.
type Metadata struct {
	// From OpenGraph
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	SiteURL     string `json:"site_url,omitempty"`

	// Author information
	Author string `json:"author,omitempty"`

	// Avatar (person/site representative image)
	// Distinct from ImageURL which may be an article/page image
	AvatarURL    string       `json:"avatar_url,omitempty"`
	AvatarSource AvatarSource `json:"avatar_source,omitempty"`

	// From feed
	FeedTitle       string     `json:"feed_title,omitempty"`
	FeedDescription string     `json:"feed_description,omitempty"`
	FeedAuthor      string     `json:"feed_author,omitempty"`
	FeedImageURL    string     `json:"feed_image_url,omitempty"`
	LastUpdated     *time.Time `json:"last_updated,omitempty"`

	// Source tracking
	Source string `json:"source,omitempty"` // "opengraph", "meta", "feed"
}

// UpdateResult contains the result of updating a single feed's metadata.
type UpdateResult struct {
	FeedURL     string    `json:"feed_url"`
	Handle      string    `json:"handle,omitempty"`
	OldMetadata *Metadata `json:"old_metadata,omitempty"`
	NewMetadata *Metadata `json:"new_metadata,omitempty"`
	Updated     bool      `json:"updated"`
	Error       string    `json:"error,omitempty"`
}

// FetchMetadata fetches metadata from a site URL or feed URL.
// It tries multiple sources in order: OpenGraph, HTML meta tags, feed metadata.
// Additionally, it attempts avatar discovery via h-card, WebFinger, and .well-known/avatar.
func (u *Updater) FetchMetadata(ctx context.Context, feedURL string) (*Metadata, error) {
	return u.FetchMetadataWithResource(ctx, feedURL, "")
}

// FetchMetadataWithResource fetches metadata and attempts avatar discovery.
// The resource parameter is used for WebFinger lookups (e.g., "acct:user@example.com").
func (u *Updater) FetchMetadataWithResource(ctx context.Context, feedURL, resource string) (*Metadata, error) {
	metadata := &Metadata{}

	// First, try to extract site URL from feed URL
	siteURL, err := extractSiteURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid feed URL: %w", err)
	}

	// Fetch site metadata (OpenGraph + HTML meta)
	siteMetadata, siteErr := u.fetchSiteMetadata(ctx, siteURL)
	if siteErr == nil && siteMetadata != nil {
		mergeMetadata(metadata, siteMetadata)
	}

	// Fetch feed metadata
	feedMetadata, feedErr := u.fetchFeedMetadata(ctx, feedURL)
	if feedErr == nil && feedMetadata != nil {
		// Feed metadata fills in gaps but doesn't overwrite
		mergeMetadataWithoutOverwrite(metadata, feedMetadata)
	}

	// If we couldn't get any metadata, return an error
	if siteErr != nil && feedErr != nil {
		return nil, fmt.Errorf("failed to fetch metadata from %s: %w", feedURL, siteErr)
	}

	// Set the site URL if not already set
	if metadata.SiteURL == "" {
		metadata.SiteURL = siteURL
	}

	// Attempt avatar discovery (best-effort, doesn't fail the overall operation)
	if avatarResult, _ := u.DiscoverAvatar(ctx, siteURL, resource); avatarResult != nil {
		metadata.AvatarURL = avatarResult.URL
		metadata.AvatarSource = avatarResult.Source
	}

	return metadata, nil
}

// fetchURL is a helper that fetches a URL and returns the body as bytes.
func (u *Updater) fetchURL(ctx context.Context, targetURL, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "markata-go/1.0 (Blogroll Updater)")
	req.Header.Set("Accept", accept)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read body with a limit (1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// fetchSiteMetadata fetches OpenGraph and HTML meta tags from a site.
func (u *Updater) fetchSiteMetadata(ctx context.Context, siteURL string) (*Metadata, error) {
	body, err := u.fetchURL(ctx, siteURL, "text/html,application/xhtml+xml")
	if err != nil {
		return nil, err
	}
	return parseHTMLMetadata(string(body), siteURL), nil
}

// fetchFeedMetadata fetches metadata from an RSS/Atom feed.
func (u *Updater) fetchFeedMetadata(ctx context.Context, feedURL string) (*Metadata, error) {
	body, err := u.fetchURL(ctx, feedURL, "application/rss+xml, application/atom+xml, application/xml, text/xml")
	if err != nil {
		return nil, err
	}
	return parseFeedMetadata(string(body)), nil
}

// parseHTMLMetadata extracts OpenGraph and meta tags from HTML using regex.
func parseHTMLMetadata(htmlContent, baseURL string) *Metadata {
	metadata := &Metadata{}

	// Extract <title> tag
	titleRe := regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)
	if matches := titleRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		metadata.Title = strings.TrimSpace(matches[1])
	}

	// Extract meta tags with pattern: <meta property="..." content="..."> or <meta name="..." content="...">
	metaRe := regexp.MustCompile(`(?i)<meta\s+[^>]*(?:property|name)=["']([^"']+)["'][^>]*content=["']([^"']*)["'][^>]*>`)
	metaRe2 := regexp.MustCompile(`(?i)<meta\s+[^>]*content=["']([^"']*)["'][^>]*(?:property|name)=["']([^"']+)["'][^>]*>`)

	processMetaMatch := func(prop, content string) {
		processOpenGraphMeta(prop, content, metadata, baseURL)
		processHTMLMeta(prop, content, metadata, baseURL)
	}

	for _, matches := range metaRe.FindAllStringSubmatch(htmlContent, -1) {
		if len(matches) > 2 {
			processMetaMatch(matches[1], matches[2])
		}
	}

	for _, matches := range metaRe2.FindAllStringSubmatch(htmlContent, -1) {
		if len(matches) > 2 {
			processMetaMatch(matches[2], matches[1])
		}
	}

	// Extract link rel="icon" for favicon
	if metadata.ImageURL == "" {
		iconRe := regexp.MustCompile(`(?i)<link\s+[^>]*rel=["'](?:icon|shortcut icon|apple-touch-icon)[^"']*["'][^>]*href=["']([^"']+)["'][^>]*>`)
		iconRe2 := regexp.MustCompile(`(?i)<link\s+[^>]*href=["']([^"']+)["'][^>]*rel=["'](?:icon|shortcut icon|apple-touch-icon)[^"']*["'][^>]*>`)

		if matches := iconRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
			metadata.ImageURL = resolveURL(matches[1], baseURL)
		} else if matches := iconRe2.FindStringSubmatch(htmlContent); len(matches) > 1 {
			metadata.ImageURL = resolveURL(matches[1], baseURL)
		}
	}

	if metadata.Title != "" || metadata.Description != "" || metadata.ImageURL != "" {
		metadata.Source = "opengraph"
	}

	return metadata
}

// processOpenGraphMeta handles OpenGraph protocol meta tags.
func processOpenGraphMeta(property, content string, metadata *Metadata, baseURL string) {
	switch property {
	case "og:title":
		if metadata.Title == "" {
			metadata.Title = content
		}
	case "og:description":
		if metadata.Description == "" {
			metadata.Description = content
		}
	case "og:image":
		if metadata.ImageURL == "" {
			metadata.ImageURL = resolveURL(content, baseURL)
		}
	case "og:url":
		if metadata.SiteURL == "" {
			metadata.SiteURL = content
		}
	case "og:site_name":
		// Can be used as fallback title
		if metadata.Title == "" {
			metadata.Title = content
		}
	}
}

// processHTMLMeta handles standard HTML meta tags.
func processHTMLMeta(name, content string, metadata *Metadata, baseURL string) {
	switch name {
	case "description":
		if metadata.Description == "" {
			metadata.Description = content
		}
	case "author":
		if metadata.Author == "" {
			metadata.Author = content
		}
	case "twitter:title":
		if metadata.Title == "" {
			metadata.Title = content
		}
	case "twitter:description":
		if metadata.Description == "" {
			metadata.Description = content
		}
	case "twitter:image":
		if metadata.ImageURL == "" {
			metadata.ImageURL = resolveURL(content, baseURL)
		}
	}
}

// parseFeedMetadata extracts metadata from RSS/Atom feed content.
func parseFeedMetadata(feedContent string) *Metadata {
	metadata := &Metadata{
		Source: "feed",
	}

	// Detect feed type and parse accordingly
	if strings.Contains(feedContent, "<feed") && strings.Contains(feedContent, "xmlns=\"http://www.w3.org/2005/Atom\"") {
		parseAtomFeedMetadata(feedContent, metadata)
	} else {
		parseRSSFeedMetadata(feedContent, metadata)
	}

	return metadata
}

// parseAtomFeedMetadata parses Atom feed metadata.
func parseAtomFeedMetadata(content string, metadata *Metadata) {
	// Extract title
	if title := extractXMLTag(content, "title"); title != "" {
		metadata.FeedTitle = stripCDATA(title)
	}

	// Extract subtitle (description)
	if subtitle := extractXMLTag(content, "subtitle"); subtitle != "" {
		metadata.FeedDescription = stripCDATA(subtitle)
	}

	// Extract author
	if authorName := extractNestedXMLTag(content, "author", "name"); authorName != "" {
		metadata.FeedAuthor = stripCDATA(authorName)
	}

	// Extract icon/logo
	if icon := extractXMLTag(content, "icon"); icon != "" {
		metadata.FeedImageURL = stripCDATA(icon)
	} else if logo := extractXMLTag(content, "logo"); logo != "" {
		metadata.FeedImageURL = stripCDATA(logo)
	}

	// Extract updated timestamp
	if updated := extractXMLTag(content, "updated"); updated != "" {
		if t, err := time.Parse(time.RFC3339, stripCDATA(updated)); err == nil {
			metadata.LastUpdated = &t
		}
	}
}

// parseRSSFeedMetadata parses RSS feed metadata.
func parseRSSFeedMetadata(content string, metadata *Metadata) {
	// Find the channel section
	channelStart := strings.Index(content, "<channel")
	channelEnd := strings.Index(content, "</channel>")
	if channelStart == -1 || channelEnd == -1 {
		return
	}
	channel := content[channelStart:channelEnd]

	// Extract title (avoid item titles)
	if title := extractChannelTag(channel, "title"); title != "" {
		metadata.FeedTitle = stripCDATA(title)
	}

	// Extract description
	if desc := extractChannelTag(channel, "description"); desc != "" {
		metadata.FeedDescription = stripCDATA(desc)
	}

	// Extract managing editor or webmaster as author
	if author := extractChannelTag(channel, "managingEditor"); author != "" {
		metadata.FeedAuthor = stripCDATA(extractEmailName(author))
	} else if author := extractChannelTag(channel, "webMaster"); author != "" {
		metadata.FeedAuthor = stripCDATA(extractEmailName(author))
	} else if author := extractChannelTag(channel, "author"); author != "" {
		metadata.FeedAuthor = stripCDATA(author)
	}

	// Extract image (try multiple sources)
	if imgURL := extractNestedXMLTag(channel, "image", "url"); imgURL != "" {
		metadata.FeedImageURL = stripCDATA(imgURL)
	} else if icon := extractXMLTag(channel, "icon"); icon != "" {
		// Fallback to Atom-style icon tag (some feeds mix formats)
		metadata.FeedImageURL = stripCDATA(icon)
	} else if logo := extractXMLTag(channel, "logo"); logo != "" {
		// Fallback to Atom-style logo tag
		metadata.FeedImageURL = stripCDATA(logo)
	}

	// Extract lastBuildDate or pubDate
	if lastBuild := extractChannelTag(channel, "lastBuildDate"); lastBuild != "" {
		if t, err := parseRSSDate(stripCDATA(lastBuild)); err == nil {
			metadata.LastUpdated = &t
		}
	} else if pubDate := extractChannelTag(channel, "pubDate"); pubDate != "" {
		if t, err := parseRSSDate(stripCDATA(pubDate)); err == nil {
			metadata.LastUpdated = &t
		}
	}
}

// extractChannelTag extracts a tag value from the channel section, avoiding item content.
func extractChannelTag(channel, tag string) string {
	// Find first occurrence before any <item> tag
	itemStart := strings.Index(channel, "<item")
	searchContent := channel
	if itemStart > 0 {
		searchContent = channel[:itemStart]
	}
	return extractXMLTag(searchContent, tag)
}

// extractXMLTag extracts the content of an XML tag.
func extractXMLTag(content, tag string) string {
	// Try both with and without namespace prefix
	patterns := []string{
		fmt.Sprintf("<%s[^>]*>([^<]*)</%s>", tag, tag),
		fmt.Sprintf("<%s[^>]*>(.*?)</%s>", tag, tag),
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// extractNestedXMLTag extracts a nested tag value.
func extractNestedXMLTag(content, parent, child string) string {
	// Find parent tag
	parentPattern := fmt.Sprintf("<%s[^>]*>(.*?)</%s>", parent, parent)
	re := regexp.MustCompile(parentPattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return extractXMLTag(matches[1], child)
	}
	return ""
}

// stripCDATA removes CDATA wrapper and trims whitespace.
func stripCDATA(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<![CDATA[") && strings.HasSuffix(s, "]]>") {
		s = s[9 : len(s)-3]
	}
	return strings.TrimSpace(s)
}

// extractEmailName extracts a name from an email format like "email (Name)" or "Name <email>".
func extractEmailName(s string) string {
	// Format: email (Name)
	if idx := strings.Index(s, "("); idx > 0 {
		end := strings.Index(s, ")")
		if end > idx {
			return strings.TrimSpace(s[idx+1 : end])
		}
	}
	// Format: Name <email>
	if idx := strings.Index(s, "<"); idx > 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// parseRSSDate parses RSS date formats.
func parseRSSDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// extractSiteURL extracts the base site URL from a feed URL.
func extractSiteURL(feedURL string) (string, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return "", err
	}

	// Return just the scheme and host
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(href, baseURL string) string {
	if href == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Protocol-relative
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	// Relative URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(ref).String()
}

// mergeMetadata merges source metadata into target, overwriting non-empty values.
func mergeMetadata(target, source *Metadata) {
	if source.Title != "" {
		target.Title = source.Title
	}
	if source.Description != "" {
		target.Description = source.Description
	}
	if source.ImageURL != "" {
		target.ImageURL = source.ImageURL
	}
	if source.SiteURL != "" {
		target.SiteURL = source.SiteURL
	}
	if source.Author != "" {
		target.Author = source.Author
	}
	if source.AvatarURL != "" {
		target.AvatarURL = source.AvatarURL
		target.AvatarSource = source.AvatarSource
	}
	if source.FeedTitle != "" {
		target.FeedTitle = source.FeedTitle
	}
	if source.FeedDescription != "" {
		target.FeedDescription = source.FeedDescription
	}
	if source.FeedAuthor != "" {
		target.FeedAuthor = source.FeedAuthor
	}
	if source.FeedImageURL != "" {
		target.FeedImageURL = source.FeedImageURL
	}
	if source.LastUpdated != nil {
		target.LastUpdated = source.LastUpdated
	}
	if source.Source != "" {
		target.Source = source.Source
	}
}

// mergeMetadataWithoutOverwrite merges source into target only for empty fields.
func mergeMetadataWithoutOverwrite(target, source *Metadata) {
	if target.Title == "" && source.FeedTitle != "" {
		target.Title = source.FeedTitle
	}
	if target.Description == "" && source.FeedDescription != "" {
		target.Description = source.FeedDescription
	}
	if target.ImageURL == "" && source.FeedImageURL != "" {
		target.ImageURL = source.FeedImageURL
	}
	if target.Author == "" && source.FeedAuthor != "" {
		target.Author = source.FeedAuthor
	}
	if target.AvatarURL == "" && source.AvatarURL != "" {
		target.AvatarURL = source.AvatarURL
		target.AvatarSource = source.AvatarSource
	}
	if target.LastUpdated == nil && source.LastUpdated != nil {
		target.LastUpdated = source.LastUpdated
	}
	// Keep feed-specific fields
	if source.FeedTitle != "" {
		target.FeedTitle = source.FeedTitle
	}
	if source.FeedDescription != "" {
		target.FeedDescription = source.FeedDescription
	}
	if source.FeedAuthor != "" {
		target.FeedAuthor = source.FeedAuthor
	}
	if source.FeedImageURL != "" {
		target.FeedImageURL = source.FeedImageURL
	}
}
