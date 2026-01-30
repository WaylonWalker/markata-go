// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"sort"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// AutoFeedsConfig configures automatic feed generation.
type AutoFeedsConfig struct {
	// Tags configures automatic tag feeds
	Tags AutoFeedTypeConfig `json:"tags" yaml:"tags" toml:"tags"`

	// Categories configures automatic category feeds
	Categories AutoFeedTypeConfig `json:"categories" yaml:"categories" toml:"categories"`

	// Archives configures automatic date archive feeds
	Archives AutoArchiveConfig `json:"archives" yaml:"archives" toml:"archives"`
}

// AutoFeedTypeConfig configures a type of auto-generated feed (tags, categories).
type AutoFeedTypeConfig struct {
	// Enabled enables generation of this feed type
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// SlugPrefix is the URL prefix for feeds (e.g., "tags" -> /tags/python/)
	SlugPrefix string `json:"slug_prefix" yaml:"slug_prefix" toml:"slug_prefix"`

	// Formats specifies which output formats to generate
	Formats models.FeedFormats `json:"formats" yaml:"formats" toml:"formats"`
}

// AutoArchiveConfig configures automatic date archive feeds.
type AutoArchiveConfig struct {
	// Enabled enables generation of archive feeds
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// SlugPrefix is the URL prefix for archives (e.g., "archive" -> /archive/2024/)
	SlugPrefix string `json:"slug_prefix" yaml:"slug_prefix" toml:"slug_prefix"`

	// YearlyFeeds enables year-based archive feeds
	YearlyFeeds bool `json:"yearly_feeds" yaml:"yearly_feeds" toml:"yearly_feeds"`

	// MonthlyFeeds enables month-based archive feeds
	MonthlyFeeds bool `json:"monthly_feeds" yaml:"monthly_feeds" toml:"monthly_feeds"`

	// Formats specifies which output formats to generate
	Formats models.FeedFormats `json:"formats" yaml:"formats" toml:"formats"`
}

// Default slug prefix constants for auto-generated feeds.
const (
	defaultTagsPrefix       = "tags"
	defaultCategoriesPrefix = "categories"
	defaultArchivePrefix    = "archive"
)

// AutoFeedsPlugin automatically generates feeds for tags, categories, and date archives.
// It implements ConfigurePlugin to pre-register synthetic posts for wikilink resolution,
// and CollectPlugin to generate the actual feed content.
type AutoFeedsPlugin struct{}

// NewAutoFeedsPlugin creates a new AutoFeedsPlugin.
func NewAutoFeedsPlugin() *AutoFeedsPlugin {
	return &AutoFeedsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *AutoFeedsPlugin) Name() string {
	return "auto_feeds"
}

// Priority returns the plugin's priority for a given stage.
func (p *AutoFeedsPlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageLoad:
		// Run late in Load to ensure all posts are loaded
		// This allows us to scan for tags/categories before Transform stage
		return lifecycle.PriorityLate
	case lifecycle.StageCollect:
		// Run early in Collect to generate feeds before other plugins need them
		return lifecycle.PriorityDefault
	default:
		return lifecycle.PriorityDefault
	}
}

// Load pre-registers synthetic posts for auto-generated feeds
// so they can be resolved by wikilinks during the Transform stage.
// This runs after posts are loaded but before Transform stage.
func (p *AutoFeedsPlugin) Load(m *lifecycle.Manager) error {
	posts := m.Posts()
	config := m.Config()

	autoConfig := getAutoFeedsConfig(config)

	// Pre-register tag feed synthetic posts
	if autoConfig.Tags.Enabled {
		p.registerTagSyntheticPosts(m, posts, autoConfig.Tags)
	}

	// Pre-register category feed synthetic posts
	if autoConfig.Categories.Enabled {
		p.registerCategorySyntheticPosts(m, posts, autoConfig.Categories)
	}

	// Pre-register archive feed synthetic posts
	if autoConfig.Archives.Enabled {
		p.registerArchiveSyntheticPosts(m, posts, autoConfig.Archives)
	}

	return nil
}

// autoFeedsStrPtr returns a pointer to the given string.
func autoFeedsStrPtr(s string) *string { return &s }

// registerTagSyntheticPosts creates synthetic posts for tag feeds.
func (p *AutoFeedsPlugin) registerTagSyntheticPosts(m *lifecycle.Manager, posts []*models.Post, config AutoFeedTypeConfig) {
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultTagsPrefix
	}

	// Collect all unique tags
	tagsMap := make(map[string]bool)
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagsMap[tag] = true
		}
	}

	// Sort tags for deterministic ordering
	tags := make([]string, 0, len(tagsMap))
	for tag := range tagsMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// Register synthetic post for each tag
	for _, tag := range tags {
		slug := prefix + "/" + slugify(tag)
		syntheticPost := &models.Post{
			Slug:        slug,
			Title:       autoFeedsStrPtr(fmt.Sprintf("Posts tagged: %s", tag)),
			Description: autoFeedsStrPtr(fmt.Sprintf("All posts with the tag %q", tag)),
			Href:        "/" + slug + "/",
			Published:   true,
			Skip:        true,
			// Add aliases so [[ python ]] resolves to tags/python
			Extra: map[string]interface{}{
				"aliases": []interface{}{tag, slugify(tag)},
			},
		}
		m.AddPost(syntheticPost)
	}
}

// registerCategorySyntheticPosts creates synthetic posts for category feeds.
func (p *AutoFeedsPlugin) registerCategorySyntheticPosts(m *lifecycle.Manager, posts []*models.Post, config AutoFeedTypeConfig) {
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultCategoriesPrefix
	}

	// Collect all unique categories
	categoriesMap := make(map[string]bool)
	for _, post := range posts {
		if cat, ok := post.Extra["category"].(string); ok && cat != "" {
			categoriesMap[cat] = true
		}
	}

	// Sort categories for deterministic ordering
	categories := make([]string, 0, len(categoriesMap))
	for cat := range categoriesMap {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Register synthetic post for each category
	for _, cat := range categories {
		slug := prefix + "/" + slugify(cat)
		syntheticPost := &models.Post{
			Slug:        slug,
			Title:       autoFeedsStrPtr(fmt.Sprintf("Category: %s", cat)),
			Description: autoFeedsStrPtr(fmt.Sprintf("All posts in the %q category", cat)),
			Href:        "/" + slug + "/",
			Published:   true,
			Skip:        true,
			// Add aliases so [[ Technology ]] resolves to categories/technology
			Extra: map[string]interface{}{
				"aliases": []interface{}{cat, slugify(cat)},
			},
		}
		m.AddPost(syntheticPost)
	}
}

// registerArchiveSyntheticPosts creates synthetic posts for archive feeds.
func (p *AutoFeedsPlugin) registerArchiveSyntheticPosts(m *lifecycle.Manager, posts []*models.Post, config AutoArchiveConfig) {
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultArchivePrefix
	}

	// Collect all unique year/month combinations
	yearsMap := make(map[int]bool)
	yearMonthsMap := make(map[string]bool)

	for _, post := range posts {
		if post.Date != nil {
			year := post.Date.Year()
			month := post.Date.Month()
			yearsMap[year] = true
			yearMonthsMap[fmt.Sprintf("%04d/%02d", year, month)] = true
		}
	}

	// Register synthetic posts for yearly archives
	if config.YearlyFeeds {
		p.registerYearlyArchivePosts(m, yearsMap, prefix)
	}

	// Register synthetic posts for monthly archives
	if config.MonthlyFeeds {
		p.registerMonthlyArchivePosts(m, yearMonthsMap, prefix)
	}
}

// registerYearlyArchivePosts creates synthetic posts for yearly archive feeds.
func (p *AutoFeedsPlugin) registerYearlyArchivePosts(m *lifecycle.Manager, yearsMap map[int]bool, prefix string) {
	years := make([]int, 0, len(yearsMap))
	for year := range yearsMap {
		years = append(years, year)
	}
	sort.Ints(years)

	for _, year := range years {
		slug := fmt.Sprintf("%s/%04d", prefix, year)
		syntheticPost := &models.Post{
			Slug:        slug,
			Title:       autoFeedsStrPtr(fmt.Sprintf("Archive: %d", year)),
			Description: autoFeedsStrPtr(fmt.Sprintf("All posts from %d", year)),
			Href:        "/" + slug + "/",
			Published:   true,
			Skip:        true,
		}
		m.AddPost(syntheticPost)
	}
}

// registerMonthlyArchivePosts creates synthetic posts for monthly archive feeds.
func (p *AutoFeedsPlugin) registerMonthlyArchivePosts(m *lifecycle.Manager, yearMonthsMap map[string]bool, prefix string) {
	yearMonths := make([]string, 0, len(yearMonthsMap))
	for ym := range yearMonthsMap {
		yearMonths = append(yearMonths, ym)
	}
	sort.Strings(yearMonths)

	for _, ym := range yearMonths {
		var year, month int
		//nolint:errcheck // best-effort parsing
		fmt.Sscanf(ym, "%d/%d", &year, &month)

		slug := fmt.Sprintf("%s/%s", prefix, ym)
		monthName := time.Month(month).String()
		syntheticPost := &models.Post{
			Slug:        slug,
			Title:       autoFeedsStrPtr(fmt.Sprintf("Archive: %s %d", monthName, year)),
			Description: autoFeedsStrPtr(fmt.Sprintf("All posts from %s %d", monthName, year)),
			Href:        "/" + slug + "/",
			Published:   true,
			Skip:        true,
		}
		m.AddPost(syntheticPost)
	}
}

// Collect generates automatic feeds for tags, categories, and date archives.
func (p *AutoFeedsPlugin) Collect(m *lifecycle.Manager) error {
	posts := m.Posts()
	config := m.Config()
	filterCache := newFeedFilterCache(posts)

	autoConfig := getAutoFeedsConfig(config)

	// Collect only auto-generated feed configs
	var autoFeedConfigs []models.FeedConfig

	// Generate tag feeds
	if autoConfig.Tags.Enabled {
		tagFeeds := p.generateTagFeeds(posts, autoConfig.Tags)
		autoFeedConfigs = append(autoFeedConfigs, tagFeeds...)
	}

	// Generate category feeds
	if autoConfig.Categories.Enabled {
		categoryFeeds := p.generateCategoryFeeds(posts, autoConfig.Categories)
		autoFeedConfigs = append(autoFeedConfigs, categoryFeeds...)
	}

	// Generate archive feeds
	if autoConfig.Archives.Enabled {
		archiveFeeds := p.generateArchiveFeeds(posts, autoConfig.Archives)
		autoFeedConfigs = append(autoFeedConfigs, archiveFeeds...)
	}

	// If no auto-feeds were generated, nothing to do
	if len(autoFeedConfigs) == 0 {
		return nil
	}

	// Process auto-generated feeds
	feedDefaults := getFeedDefaults(config)

	// Get existing feeds from FeedsPlugin
	feeds := m.Feeds()

	// Get existing feed configs from cache to append auto-generated ones
	allFeedConfigs := make([]models.FeedConfig, 0, len(autoFeedConfigs))
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			allFeedConfigs = append(allFeedConfigs, fcs...)
		}
	}

	for i := range autoFeedConfigs {
		fc := &autoFeedConfigs[i]

		// Apply defaults
		fc.ApplyDefaults(feedDefaults)

		// Filter posts for this feed
		filteredPosts, err := filterCache.FilterPosts(fc.Filter, fc.IncludePrivate)
		if err != nil {
			return fmt.Errorf("auto feed %q: %w", fc.Slug, err)
		}
		filteredPosts = cloneFeedPosts(filteredPosts)

		// Sort posts by date, newest first
		sortPosts(filteredPosts, "date", true)

		// Store posts in feed config
		fc.Posts = filteredPosts

		// Get base URL for pagination
		baseURL := "/" + fc.Slug

		// Paginate results
		fc.Paginate(baseURL)

		// Create lifecycle.Feed
		feed := &lifecycle.Feed{
			Name:  fc.Slug,
			Title: fc.Title,
			Posts: filteredPosts,
			Path:  fc.Slug,
		}

		feeds = append(feeds, feed)
		allFeedConfigs = append(allFeedConfigs, *fc)
	}

	m.SetFeeds(feeds)

	// Update cache with all feed configs (original + auto-generated)
	m.Cache().Set("feed_configs", allFeedConfigs)

	return nil
}

// generateTagFeeds creates feed configurations for each unique tag.
func (p *AutoFeedsPlugin) generateTagFeeds(posts []*models.Post, config AutoFeedTypeConfig) []models.FeedConfig {
	// Collect all unique tags
	tagCounts := make(map[string]int)
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagCounts[tag]++
		}
	}

	// Sort tags alphabetically
	tags := make([]string, 0, len(tagCounts))
	for tag := range tagCounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// Create feed config for each tag
	feeds := make([]models.FeedConfig, 0, len(tags))
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultTagsPrefix
	}

	for _, tag := range tags {
		slug := prefix + "/" + slugify(tag)
		feeds = append(feeds, models.FeedConfig{
			Slug:        slug,
			Title:       fmt.Sprintf("Posts tagged: %s", tag),
			Description: fmt.Sprintf("All posts with the tag %q", tag),
			Filter:      fmt.Sprintf("%q in tags", tag),
			Sort:        "date",
			Reverse:     true,
			Formats:     config.Formats,
		})
	}

	return feeds
}

// generateCategoryFeeds creates feed configurations for each unique category.
func (p *AutoFeedsPlugin) generateCategoryFeeds(posts []*models.Post, config AutoFeedTypeConfig) []models.FeedConfig {
	// Collect all unique categories from Extra["category"]
	categoryCounts := make(map[string]int)
	for _, post := range posts {
		if cat, ok := post.Extra["category"].(string); ok && cat != "" {
			categoryCounts[cat]++
		}
	}

	// Sort categories alphabetically
	categories := make([]string, 0, len(categoryCounts))
	for cat := range categoryCounts {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Create feed config for each category
	feeds := make([]models.FeedConfig, 0, len(categories))
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultCategoriesPrefix
	}

	for _, cat := range categories {
		slug := prefix + "/" + slugify(cat)
		feeds = append(feeds, models.FeedConfig{
			Slug:        slug,
			Title:       fmt.Sprintf("Category: %s", cat),
			Description: fmt.Sprintf("All posts in the %q category", cat),
			Filter:      fmt.Sprintf("category == %q", cat),
			Sort:        "date",
			Reverse:     true,
			Formats:     config.Formats,
		})
	}

	return feeds
}

// generateArchiveFeeds creates feed configurations for year and month archives.
func (p *AutoFeedsPlugin) generateArchiveFeeds(posts []*models.Post, config AutoArchiveConfig) []models.FeedConfig {
	// Collect all unique year/month combinations
	yearMonths := make(map[string]bool)
	years := make(map[int]bool)

	for _, post := range posts {
		if post.Date != nil {
			year := post.Date.Year()
			month := post.Date.Month()

			years[year] = true
			yearMonths[fmt.Sprintf("%04d/%02d", year, month)] = true
		}
	}

	var feeds []models.FeedConfig
	prefix := config.SlugPrefix
	if prefix == "" {
		prefix = defaultArchivePrefix
	}

	// Create yearly feeds
	if config.YearlyFeeds {
		var yearList []int
		for year := range years {
			yearList = append(yearList, year)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(yearList)))

		for _, year := range yearList {
			slug := fmt.Sprintf("%s/%04d", prefix, year)
			startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

			feeds = append(feeds, models.FeedConfig{
				Slug:        slug,
				Title:       fmt.Sprintf("Archive: %d", year),
				Description: fmt.Sprintf("All posts from %d", year),
				Filter:      fmt.Sprintf("date >= %q and date < %q", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)),
				Sort:        "date",
				Reverse:     true,
				Formats:     config.Formats,
			})
		}
	}

	// Create monthly feeds
	if config.MonthlyFeeds {
		var ymList []string
		for ym := range yearMonths {
			ymList = append(ymList, ym)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(ymList)))

		for _, ym := range ymList {
			var year, month int
			//nolint:errcheck // best-effort parsing, invalid format will result in zero values
			fmt.Sscanf(ym, "%d/%d", &year, &month)

			slug := fmt.Sprintf("%s/%s", prefix, ym)
			monthName := time.Month(month).String()
			startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endDate := startDate.AddDate(0, 1, 0)

			feeds = append(feeds, models.FeedConfig{
				Slug:        slug,
				Title:       fmt.Sprintf("Archive: %s %d", monthName, year),
				Description: fmt.Sprintf("All posts from %s %d", monthName, year),
				Filter:      fmt.Sprintf("date >= %q and date < %q", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)),
				Sort:        "date",
				Reverse:     true,
				Formats:     config.Formats,
			})
		}
	}

	return feeds
}

// getAutoFeedsConfig retrieves auto feeds configuration from the manager config.
func getAutoFeedsConfig(config *lifecycle.Config) AutoFeedsConfig {
	defaultConfig := AutoFeedsConfig{
		Tags: AutoFeedTypeConfig{
			Enabled:    true, // Tag feeds enabled by default
			SlugPrefix: defaultTagsPrefix,
			Formats: models.FeedFormats{
				HTML: true,
				RSS:  true,
			},
		},
		Categories: AutoFeedTypeConfig{
			Enabled:    false,
			SlugPrefix: defaultCategoriesPrefix,
			Formats: models.FeedFormats{
				HTML: true,
				RSS:  true,
			},
		},
		Archives: AutoArchiveConfig{
			Enabled:      false,
			SlugPrefix:   defaultArchivePrefix,
			YearlyFeeds:  true,
			MonthlyFeeds: false,
			Formats: models.FeedFormats{
				HTML: true,
				RSS:  false,
			},
		},
	}

	if config.Extra == nil {
		return defaultConfig
	}

	if autoFeeds, ok := config.Extra["auto_feeds"]; ok {
		if ac, ok := autoFeeds.(AutoFeedsConfig); ok {
			return ac
		}
	}

	return defaultConfig
}

// slugify converts a string to a URL-safe slug.
// This is a convenience wrapper around models.Slugify for internal use.
func slugify(s string) string {
	return models.Slugify(s)
}

// Ensure AutoFeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*AutoFeedsPlugin)(nil)
	_ lifecycle.LoadPlugin     = (*AutoFeedsPlugin)(nil)
	_ lifecycle.CollectPlugin  = (*AutoFeedsPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*AutoFeedsPlugin)(nil)
)
