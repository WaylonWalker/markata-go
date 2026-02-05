// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// SubscriptionFeedsPlugin creates built-in subscription feeds at root and /archive.
// It generates /rss.xml, /atom.xml, /archive/rss.xml, /archive/atom.xml
// without creating HTML index pages.
//
// This plugin also computes the discovery feed for each post during rendering,
// enabling dynamic <link rel="alternate"> tags based on the post's sidebar feed.
type SubscriptionFeedsPlugin struct{}

// NewSubscriptionFeedsPlugin creates a new SubscriptionFeedsPlugin.
func NewSubscriptionFeedsPlugin() *SubscriptionFeedsPlugin {
	return &SubscriptionFeedsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *SubscriptionFeedsPlugin) Name() string {
	return "subscription_feeds"
}

// Priority returns the plugin's priority for a given stage.
func (p *SubscriptionFeedsPlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageCollect:
		// Run early in Collect to inject subscription feeds before other feed processing
		return lifecycle.PriorityEarly
	default:
		return lifecycle.PriorityDefault
	}
}

// Collect injects built-in subscription feeds into the feed configs.
// These feeds generate RSS/Atom at root (/) and /archive without HTML pages.
func (p *SubscriptionFeedsPlugin) Collect(m *lifecycle.Manager) error {
	config := m.Config()

	// Check if subscription feeds are disabled
	if config.Extra != nil {
		if disabled, ok := config.Extra["subscription_feeds_disabled"].(bool); ok && disabled {
			return nil
		}
	}

	// Get existing feed configs from cache
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	// Check if root subscription feed already exists
	hasRootFeed := false
	hasArchiveFeed := false
	for i := range feedConfigs {
		if feedConfigs[i].Slug == "" {
			hasRootFeed = true
		}
		if feedConfigs[i].Slug == defaultArchivePrefix {
			hasArchiveFeed = true
		}
	}

	// Create root subscription feed (slug="") if not already defined
	if !hasRootFeed {
		rootFeed := models.FeedConfig{
			Slug:        "",
			Title:       getSubscriptionFeedTitle(config, "root"),
			Description: getSubscriptionFeedDescription(config, "root"),
			Filter:      "published == true",
			Sort:        "date",
			Reverse:     true,
			Formats: models.FeedFormats{
				HTML: false, // Don't generate index.html at root
				RSS:  true,
				Atom: true,
				JSON: false,
			},
		}
		feedConfigs = append(feedConfigs, rootFeed)
	}

	// Create archive subscription feed (slug="archive") if not already defined
	if !hasArchiveFeed {
		archiveFeed := models.FeedConfig{
			Slug:        defaultArchivePrefix,
			Title:       getSubscriptionFeedTitle(config, defaultArchivePrefix),
			Description: getSubscriptionFeedDescription(config, defaultArchivePrefix),
			Filter:      "published == true",
			Sort:        "date",
			Reverse:     true,
			Formats: models.FeedFormats{
				HTML: false, // Don't generate index.html at /archive
				RSS:  true,
				Atom: true,
				JSON: false,
			},
		}
		feedConfigs = append(feedConfigs, archiveFeed)
	}

	// Store updated feed configs back to cache
	m.Cache().Set("feed_configs", feedConfigs)

	return nil
}

// getSubscriptionFeedTitle returns the title for a subscription feed.
func getSubscriptionFeedTitle(config *lifecycle.Config, feedType string) string {
	siteTitle := "Site"
	if config.Extra != nil {
		if title, ok := config.Extra["title"].(string); ok && title != "" {
			siteTitle = title
		}
	}

	switch feedType {
	case "root":
		return siteTitle + " Feed"
	case defaultArchivePrefix:
		return siteTitle + " Archive Feed"
	default:
		return siteTitle + " Feed"
	}
}

// getSubscriptionFeedDescription returns the description for a subscription feed.
func getSubscriptionFeedDescription(config *lifecycle.Config, feedType string) string {
	siteDescription := ""
	if config.Extra != nil {
		if desc, ok := config.Extra["description"].(string); ok {
			siteDescription = desc
		}
	}

	if siteDescription != "" {
		return siteDescription
	}

	switch feedType {
	case "root":
		return "All published posts"
	case defaultArchivePrefix:
		return "Archive of all published posts"
	default:
		return "Posts feed"
	}
}

// DiscoveryFeed represents feed discovery information for templates.
// This is injected into template context to enable per-page feed discovery.
type DiscoveryFeed struct {
	// Slug is the feed slug (e.g., "tags/python", "archive", "")
	Slug string

	// Title is the feed title for display
	Title string

	// RSSURL is the RSS feed URL (if RSS format is enabled)
	RSSURL string

	// AtomURL is the Atom feed URL (if Atom format is enabled)
	AtomURL string

	// JSONURL is the JSON feed URL (if JSON format is enabled)
	JSONURL string

	// HasRSS indicates whether RSS format is enabled
	HasRSS bool

	// HasAtom indicates whether Atom format is enabled
	HasAtom bool

	// HasJSON indicates whether JSON format is enabled
	HasJSON bool
}

// GetDiscoveryFeed returns the discovery feed for a post.
// If the post has a sidebar_feed, that feed is used for discovery.
// Otherwise, the site default feed (root subscription feed) is used.
//
// The post parameter is reserved for future use when discovery logic
// may need to inspect post metadata (e.g., explicit feed assignment).
// This function is called from templates.go renderPost to inject discovery_feed context.
func GetDiscoveryFeed(_ *models.Post, sidebarFeed *models.FeedConfig, allFeeds []models.FeedConfig) *DiscoveryFeed {
	// If post has a sidebar feed, use that for discovery
	if sidebarFeed != nil {
		return feedConfigToDiscoveryFeed(sidebarFeed)
	}

	// Otherwise, use the root subscription feed (slug="")
	for i := range allFeeds {
		if allFeeds[i].Slug == "" {
			return feedConfigToDiscoveryFeed(&allFeeds[i])
		}
	}

	// Fallback: return a default discovery feed pointing to root feeds
	return &DiscoveryFeed{
		Slug:    "",
		Title:   "Site Feed",
		RSSURL:  "/rss.xml",
		AtomURL: "/atom.xml",
		HasRSS:  true,
		HasAtom: true,
		HasJSON: false,
	}
}

// feedConfigToDiscoveryFeed converts a FeedConfig to a DiscoveryFeed.
func feedConfigToDiscoveryFeed(fc *models.FeedConfig) *DiscoveryFeed {
	df := &DiscoveryFeed{
		Slug:    fc.Slug,
		Title:   fc.Title,
		HasRSS:  fc.Formats.RSS,
		HasAtom: fc.Formats.Atom,
		HasJSON: fc.Formats.JSON,
	}

	// Generate feed URLs based on slug
	baseURL := ""
	if fc.Slug != "" {
		baseURL = "/" + fc.Slug
	}

	if df.HasRSS {
		df.RSSURL = baseURL + "/rss.xml"
	}
	if df.HasAtom {
		df.AtomURL = baseURL + "/atom.xml"
	}
	if df.HasJSON {
		df.JSONURL = baseURL + "/feed.json"
	}

	return df
}

// DiscoveryFeedToMap converts a DiscoveryFeed to a map for template context.
func DiscoveryFeedToMap(df *DiscoveryFeed) map[string]interface{} {
	if df == nil {
		return nil
	}
	return map[string]interface{}{
		"slug":     df.Slug,
		"title":    df.Title,
		"rss_url":  df.RSSURL,
		"atom_url": df.AtomURL,
		"json_url": df.JSONURL,
		"has_rss":  df.HasRSS,
		"has_atom": df.HasAtom,
		"has_json": df.HasJSON,
	}
}

// GetFeedBySlug finds a feed config by slug from the feed configs list.
func GetFeedBySlug(slug string, feedConfigs []models.FeedConfig) *models.FeedConfig {
	for i := range feedConfigs {
		if feedConfigs[i].Slug == slug {
			return &feedConfigs[i]
		}
	}
	return nil
}

// FindPostSidebarFeed finds the sidebar feed for a post.
// It checks if the post belongs to any feed configured for sidebar display.
// Returns the feed config and a boolean indicating if a match was found.
func FindPostSidebarFeed(post *models.Post, config *lifecycle.Config, feedConfigs []models.FeedConfig) *models.FeedConfig {
	// Get components config
	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok {
		return nil
	}

	// Check if feed sidebar is enabled
	if components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return nil
	}

	// Get configured feed slugs
	feedSlugs := components.FeedSidebar.Feeds
	if len(feedSlugs) == 0 {
		return nil
	}

	// Check if this post belongs to any of the configured feeds
	for _, feedSlug := range feedSlugs {
		// Handle tag-based feeds (tags/xxx)
		if strings.HasPrefix(feedSlug, "tags/") {
			tagName := strings.TrimPrefix(feedSlug, "tags/")
			for _, postTag := range post.Tags {
				if postTag == tagName {
					// Found a matching tag feed
					return GetFeedBySlug(feedSlug, feedConfigs)
				}
			}
		}

		// Handle explicit feed membership via frontmatter
		if post.PrevNextFeed == feedSlug {
			return GetFeedBySlug(feedSlug, feedConfigs)
		}
		if feed, ok := post.Extra["feed"].(string); ok && feed == feedSlug {
			return GetFeedBySlug(feedSlug, feedConfigs)
		}

		// Handle series-based feeds
		if series, ok := post.Extra["series"].(string); ok {
			seriesSlug := fmt.Sprintf("series/%s", models.Slugify(series))
			if feedSlug == seriesSlug {
				return GetFeedBySlug(feedSlug, feedConfigs)
			}
		}
	}

	return nil
}

// Ensure SubscriptionFeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*SubscriptionFeedsPlugin)(nil)
	_ lifecycle.CollectPlugin  = (*SubscriptionFeedsPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*SubscriptionFeedsPlugin)(nil)
)
