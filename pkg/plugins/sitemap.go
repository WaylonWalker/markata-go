// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// SitemapPlugin generates a sitemap.xml file during the write stage.
// The sitemap includes all published posts and feed index pages.
type SitemapPlugin struct{}

// NewSitemapPlugin creates a new SitemapPlugin.
func NewSitemapPlugin() *SitemapPlugin {
	return &SitemapPlugin{}
}

// Name returns the unique name of the plugin.
func (p *SitemapPlugin) Name() string {
	return "sitemap"
}

// Priority returns the plugin priority for the given stage.
// Sitemap should run late in the write stage, after all other content is written.
func (p *SitemapPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLate
	}
	return lifecycle.PriorityDefault
}

// Write generates and writes the sitemap.xml file.
func (p *SitemapPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Get site URL
	siteURL := getSiteURL(config)
	if siteURL == "" {
		siteURL = DefaultSiteURL
	}

	// Build sitemap
	sitemap := p.buildSitemap(m, siteURL)

	// Marshal to XML
	output, err := xml.MarshalIndent(sitemap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshaling sitemap: %w", err)
	}

	// Add XML declaration
	xmlContent := xml.Header + string(output)

	// Write sitemap.xml
	sitemapPath := filepath.Join(outputDir, "sitemap.xml")
	if err := os.WriteFile(sitemapPath, []byte(xmlContent), 0o644); err != nil { //nolint:gosec // sitemap needs world-readable permissions for web serving
		return fmt.Errorf("writing sitemap: %w", err)
	}

	return nil
}

// buildSitemap creates the sitemap structure from posts and feeds.
//
//nolint:gocyclo // complexity is due to straightforward iteration over posts and feeds
func (p *SitemapPlugin) buildSitemap(m *lifecycle.Manager, siteURL string) *URLSet {
	sitemap := &URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]SitemapURL, 0),
	}

	// Find the most recent post date for the home page
	var latestDate time.Time
	posts := m.Posts()
	for _, post := range posts {
		if post.Published && !post.Draft && !post.Skip && post.Date != nil {
			if post.Date.After(latestDate) {
				latestDate = *post.Date
			}
		}
	}

	// Add home page
	homeURL := SitemapURL{
		Loc:        siteURL + "/",
		ChangeFreq: "daily",
		Priority:   "1.0",
	}
	if !latestDate.IsZero() {
		homeURL.LastMod = latestDate.Format("2006-01-02")
	}
	sitemap.URLs = append(sitemap.URLs, homeURL)

	// Add all published posts
	for _, post := range posts {
		if !post.Published || post.Draft || post.Skip || post.Private {
			continue
		}

		url := SitemapURL{
			Loc:        siteURL + post.Href,
			ChangeFreq: "weekly",
			Priority:   "0.8",
		}

		if post.Date != nil {
			url.LastMod = post.Date.Format("2006-01-02")
		}

		sitemap.URLs = append(sitemap.URLs, url)
	}

	// Add feed index pages
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	for i := range feedConfigs {
		fc := &feedConfigs[i]
		if fc.Slug == "" {
			continue // Skip root feed, already covered by home page
		}

		// Find the latest post date in this feed
		var feedLatestDate time.Time
		for _, post := range fc.Posts {
			if post.Date != nil && post.Date.After(feedLatestDate) {
				feedLatestDate = *post.Date
			}
		}

		url := SitemapURL{
			Loc:        siteURL + "/" + fc.Slug + "/",
			ChangeFreq: "weekly",
			Priority:   "0.6",
		}

		if !feedLatestDate.IsZero() {
			url.LastMod = feedLatestDate.Format("2006-01-02")
		}

		sitemap.URLs = append(sitemap.URLs, url)
	}

	return sitemap
}

// URLSet represents the root element of a sitemap.
type URLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapURL represents a single URL entry in the sitemap.
type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// Ensure SitemapPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*SitemapPlugin)(nil)
	_ lifecycle.WritePlugin    = (*SitemapPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*SitemapPlugin)(nil)
)
