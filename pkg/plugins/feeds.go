// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/filter"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// FeedsPlugin processes configured feeds during the collect stage.
// It filters posts, sorts them, and paginates the results.
type FeedsPlugin struct{}

// NewFeedsPlugin creates a new FeedsPlugin.
func NewFeedsPlugin() *FeedsPlugin {
	return &FeedsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *FeedsPlugin) Name() string {
	return "feeds"
}

// Collect processes each FeedConfig and creates feeds with filtered, sorted, and paginated posts.
func (p *FeedsPlugin) Collect(m *lifecycle.Manager) error {
	posts := m.Posts()
	config := m.Config()
	useIncremental := false
	if config.Extra != nil {
		if enabled, ok := config.Extra["feeds_incremental"].(bool); ok && enabled {
			useIncremental = true
		}
	}
	filterCache := newFeedFilterCache(posts)

	// Get feed configs from manager's extra config
	feedConfigs := getFeedConfigs(config)
	feedDefaults := getFeedDefaults(config)

	feeds := make([]*lifecycle.Feed, 0, len(feedConfigs))

	for i := range feedConfigs {
		fc := &feedConfigs[i]
		if useIncremental && p.shouldSkipFeedCollect(fc, m) {
			continue
		}

		// Apply defaults
		fc.ApplyDefaults(feedDefaults)

		usePresetPosts := fc.Type == models.FeedTypeSeries && len(fc.Posts) > 0

		var filteredPosts []*models.Post
		if usePresetPosts {
			filteredPosts = cloneFeedPosts(fc.Posts)
		} else {
			// Filter posts
			var err error
			filteredPosts, err = filterCache.FilterPosts(fc.Filter, fc.IncludePrivate)
			if err != nil {
				return fmt.Errorf("feed %q: %w", fc.Slug, err)
			}
			filteredPosts = cloneFeedPosts(filteredPosts)
		}

		if !usePresetPosts {
			// Determine sort field and direction based on feed type
			sortField := fc.Sort
			reverse := fc.Reverse

			// For guide-type feeds, default to sorting by guide_order (ascending)
			if fc.Type == models.FeedTypeGuide {
				if sortField == "" {
					sortField = "guide_order"
					reverse = false // Guides should be in ascending order by default
				}
			} else {
				// Default to date sorting for non-guide feeds
				if sortField == "" {
					sortField = "date"
					reverse = true // Newest first by default
				}
			}

			sortPosts(filteredPosts, sortField, reverse)

			// For guide-type feeds, set up prev/next navigation on each post
			if fc.Type == models.FeedTypeGuide || fc.Type == models.FeedTypeSeries {
				setGuideNavigation(filteredPosts, fc.Slug)
			}
		}

		// Store posts in feed config
		fc.Posts = filteredPosts

		// Get base URL for pagination
		baseURL := "/" + fc.Slug
		if fc.Slug == "" {
			baseURL = ""
		}

		// Paginate results
		fc.Paginate(baseURL)

		// Create lifecycle.Feed for each page
		feed := &lifecycle.Feed{
			Name:  fc.Slug,
			Title: fc.Title,
			Posts: filteredPosts,
			Path:  fc.Slug,
		}

		feeds = append(feeds, feed)
	}

	m.SetFeeds(feeds)

	// Store feed configs back in cache for publish_feeds to access
	m.Cache().Set("feed_configs", feedConfigs)

	return nil
}

func (p *FeedsPlugin) shouldSkipFeedCollect(fc *models.FeedConfig, m *lifecycle.Manager) bool {
	cache := GetBuildCache(m)
	if cache == nil {
		return false
	}
	if len(fc.Posts) == 0 {
		return true
	}

	for _, post := range fc.Posts {
		currentHash := computePostFeedItemHash(post)
		cachedHash, _, _ := cache.GetPostSemanticHashes(post.Path)
		if cachedHash != currentHash {
			return false
		}
	}

	return true
}

type feedFilterKey struct {
	expr           string
	includePrivate bool
}

type feedFilterCache struct {
	parsed      map[string]*filter.Filter
	filtered    map[feedFilterKey][]*models.Post
	publicPosts []*models.Post
	allPosts    []*models.Post
}

func newFeedFilterCache(posts []*models.Post) *feedFilterCache {
	publicPosts := make([]*models.Post, 0, len(posts))
	for _, post := range posts {
		if !post.Private {
			publicPosts = append(publicPosts, post)
		}
	}

	return &feedFilterCache{
		parsed:      make(map[string]*filter.Filter),
		filtered:    make(map[feedFilterKey][]*models.Post),
		publicPosts: publicPosts,
		allPosts:    posts,
	}
}

func (c *feedFilterCache) FilterPosts(filterExpr string, includePrivate bool) ([]*models.Post, error) {
	key := feedFilterKey{expr: filterExpr, includePrivate: includePrivate}
	if cached, ok := c.filtered[key]; ok {
		return cached, nil
	}

	candidatePosts := c.publicPosts
	if includePrivate {
		candidatePosts = c.allPosts
	}

	if filterExpr == "" {
		c.filtered[key] = candidatePosts
		return candidatePosts, nil
	}

	f, ok := c.parsed[filterExpr]
	if !ok {
		var err error
		f, err = filter.Parse(filterExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter expression: %w", err)
		}
		c.parsed[filterExpr] = f
	}

	result := f.MatchAll(candidatePosts)
	c.filtered[key] = result
	return result, nil
}

// getFeedConfigs retrieves feed configurations from the manager config.
func getFeedConfigs(config *lifecycle.Config) []models.FeedConfig {
	if config.Extra == nil {
		return nil
	}

	if feeds, ok := config.Extra["feeds"]; ok {
		if feedConfigs, ok := feeds.([]models.FeedConfig); ok {
			return feedConfigs
		}
	}

	return nil
}

// getFeedDefaults retrieves feed defaults from the manager config.
func getFeedDefaults(config *lifecycle.Config) models.FeedDefaults {
	if config.Extra == nil {
		return models.NewFeedDefaults()
	}

	if defaults, ok := config.Extra["feed_defaults"]; ok {
		if feedDefaults, ok := defaults.(models.FeedDefaults); ok {
			return feedDefaults
		}
	}

	return models.NewFeedDefaults()
}

// filterPosts applies a filter expression to posts.
// If includePrivate is false, private posts are excluded before filtering.
func filterPosts(posts []*models.Post, filterExpr string, includePrivate bool) ([]*models.Post, error) {
	// First, filter out private posts if includePrivate is false
	var candidatePosts []*models.Post
	if includePrivate {
		candidatePosts = posts
	} else {
		for _, post := range posts {
			if !post.Private {
				candidatePosts = append(candidatePosts, post)
			}
		}
	}

	if filterExpr == "" {
		// Return a copy of all candidate posts
		result := make([]*models.Post, len(candidatePosts))
		copy(result, candidatePosts)
		return result, nil
	}

	f, err := filter.Parse(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter expression: %w", err)
	}

	return f.MatchAll(candidatePosts), nil
}

func cloneFeedPosts(posts []*models.Post) []*models.Post {
	if len(posts) == 0 {
		return nil
	}
	result := make([]*models.Post, len(posts))
	copy(result, posts)
	return result
}

// sortPosts sorts posts by the specified field.
// When primary sort values are equal, uses Path as a secondary sort key
// to ensure deterministic ordering across builds.
func sortPosts(posts []*models.Post, field string, reverse bool) {
	sort.SliceStable(posts, func(i, j int) bool {
		vi := getFieldValue(posts[i], field)
		vj := getFieldValue(posts[j], field)

		cmp := compareFieldValues(vi, vj)

		// Use path as tie-breaker for deterministic ordering
		if cmp == 0 {
			cmp = strings.Compare(posts[i].Path, posts[j].Path)
		}

		if reverse {
			return cmp > 0
		}
		return cmp < 0
	})
}

// getFieldValue retrieves a field value from a post.
func getFieldValue(post *models.Post, field string) interface{} {
	// Check Extra fields first
	if post.Extra != nil {
		if v, ok := post.Extra[field]; ok {
			return v
		}
	}

	// Use reflection for struct fields
	v := reflect.ValueOf(post).Elem()
	t := v.Type()

	// Try to find field by name (case-insensitive)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if strings.EqualFold(f.Name, field) {
			fv := v.Field(i)
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					return nil
				}
				return fv.Elem().Interface()
			}
			return fv.Interface()
		}
	}

	return nil
}

// compareFieldValues compares two field values for sorting.
func compareFieldValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Compare time.Time
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			if ta.Before(tb) {
				return -1
			}
			if ta.After(tb) {
				return 1
			}
			return 0
		}
	}

	// Compare strings
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			return strings.Compare(sa, sb)
		}
	}

	// Compare integers (for guide_order)
	if ia, ok := toInt(a); ok {
		if ib, ok := toInt(b); ok {
			if ia < ib {
				return -1
			}
			if ia > ib {
				return 1
			}
			return 0
		}
	}

	// Compare as formatted strings
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// toInt attempts to convert an interface{} to an int.
// It handles int, int64, float64, and other common numeric types.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}

// setGuideNavigation sets up Prev/Next navigation for posts in a guide series.
// Posts should already be sorted in the desired order before calling this function.
func setGuideNavigation(posts []*models.Post, feedSlug string) {
	for i, post := range posts {
		// Set feed reference for navigation context
		post.PrevNextFeed = feedSlug

		// Set Prev pointer
		if i > 0 {
			post.Prev = posts[i-1]
		} else {
			post.Prev = nil
		}

		// Set Next pointer
		if i < len(posts)-1 {
			post.Next = posts[i+1]
		} else {
			post.Next = nil
		}
	}
}

// Ensure FeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin        = (*FeedsPlugin)(nil)
	_ lifecycle.CollectPlugin = (*FeedsPlugin)(nil)
)
