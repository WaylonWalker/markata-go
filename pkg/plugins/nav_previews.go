package plugins

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// buildNavPreviews resolves configured local navigation links once per build.
// The result is read-only while templates render concurrently.
func buildNavPreviews(config *models.Config, posts []*models.Post, feeds []models.FeedConfig, blogroll models.BlogrollConfig, random RandomPostConfig) map[string]map[string]interface{} {
	if config == nil {
		return nil
	}
	items := config.Components.Nav.Items
	if len(items) == 0 {
		items = config.Nav
	}
	items = append(append([]models.NavItem{}, items...), config.Components.Footer.Links...)
	if len(items) == 0 {
		return nil
	}

	feedsByPath := make(map[string]*models.FeedConfig, len(feeds))
	for i := range feeds {
		feed := &feeds[i]
		if !feed.IncludesPrivate() {
			feedsByPath[cleanNavPath("/"+feed.Slug)] = feed
		}
	}
	postsByPath := make(map[string]*models.Post, len(posts))
	for _, post := range posts {
		if !eligibleNavPost(post) {
			continue
		}
		postsByPath[cleanNavPath(post.Href)] = post
	}

	previews := make(map[string]map[string]interface{}, len(items))
	for _, item := range items {
		key, ok := localNavPath(item)
		if !ok {
			continue
		}
		if feed := feedsByPath[key]; feed != nil {
			previews[item.URL] = feedNavPreview(feed)
		} else if post := postsByPath[key]; post != nil {
			previews[item.URL] = postNavPreview(post)
		} else if special := specialNavPreview(key, posts, blogroll, random); special != nil {
			previews[item.URL] = special
		}
	}
	return previews
}

func specialNavPreview(key string, posts []*models.Post, blogroll models.BlogrollConfig, random RandomPostConfig) map[string]interface{} {
	if blogroll.Enabled {
		blogrollSlug := blogroll.BlogrollSlug
		if blogrollSlug == "" {
			blogrollSlug = defaultBlogrollSlug
		}
		readerSlug := blogroll.ReaderSlug
		if readerSlug == "" {
			readerSlug = defaultReaderSlug
		}
		blogrollPath := cleanNavPath("/" + blogrollSlug)
		readerPath := cleanNavPath("/" + readerSlug)
		if key == blogrollPath || key == readerPath {
			count := 0
			for i := range blogroll.Feeds {
				if blogroll.Feeds[i].IsActive() {
					count++
				}
			}
			if key == readerPath {
				return map[string]interface{}{
					"kind": "reader", "description": "Recent articles, videos, and podcasts from followed RSS feeds.",
					"count": count, "count_label": "sources",
				}
			}
			return map[string]interface{}{
				"kind": "followed", "description": "The blogs and feeds I follow.",
				"count": count, "count_label": "sources",
			}
		}
	}
	if random.Enabled && key == cleanNavPath("/"+normalizeRandomPostPath(random.Path)) {
		return map[string]interface{}{
			"kind": "surprise", "description": "Find a post you might have missed.",
			"count": len(eligibleRandomPostHrefs(posts, random.ExcludeTags)), "count_label": "posts in the draw",
		}
	}
	return nil
}

func eligibleNavPost(post *models.Post) bool {
	return post != nil && !post.Skip && !post.Private && post.Published && !post.Draft && post.Href != ""
}

func localNavPath(item models.NavItem) (string, bool) {
	if item.External || item.URL == "" {
		return "", false
	}
	u, err := url.Parse(item.URL)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") {
		return "", false
	}
	return cleanNavPath(u.Path), true
}

func cleanNavPath(raw string) string {
	cleaned := path.Clean("/" + strings.Trim(raw, "/"))
	if cleaned == "/" {
		return cleaned
	}
	return cleaned + "/"
}

func feedNavPreview(feed *models.FeedConfig) map[string]interface{} {
	preview := map[string]interface{}{
		"kind":        "feed",
		"description": feed.Description,
	}
	count, words, minutes := 0, 0, 0
	for _, post := range feed.Posts {
		if post == nil || post.Private || post.Skip || !post.Published || post.Draft {
			continue
		}
		count++
		words += postStat(post, "word_count")
		minutes += postStat(post, "reading_time")
	}
	preview["count"] = count
	preview["words"] = words
	preview["minutes"] = minutes
	preview["words_display"] = formatNavNumber(words)
	preview["reading_display"] = formatNavReadingTime(minutes)
	window := computeSparklineWindow(feed.Posts, false)
	preview["sparkline"] = buildFeedSparkline(feed.Posts, window, false)
	preview["sparkline_title"] = buildFeedSparklineTitle(feed.Posts, window, false)
	if strings.HasPrefix(feed.Slug, "tags/") {
		preview["tags"] = []string{strings.TrimPrefix(feed.Slug, "tags/")}
	}
	return preview
}

func postNavPreview(post *models.Post) map[string]interface{} {
	preview := map[string]interface{}{
		"kind": "post",
	}
	words := postStat(post, "word_count")
	minutes := postStat(post, "reading_time")
	preview["words"] = words
	preview["minutes"] = minutes
	preview["words_display"] = formatNavNumber(words)
	preview["reading_display"] = formatNavReadingTime(minutes)
	if post.Description != nil {
		preview["description"] = *post.Description
	}
	if len(post.Tags) > 0 {
		preview["tags"] = post.Tags[:min(len(post.Tags), 3)]
	}
	return preview
}

func postStat(post *models.Post, key string) int {
	if post.Extra == nil {
		return 0
	}
	value, ok := post.Extra[key].(int)
	if !ok || value < 0 {
		return 0
	}
	return value
}

func formatNavNumber(value int) string {
	digits := strconv.Itoa(value)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

func formatNavReadingTime(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d hr", minutes/60)
	}
	return fmt.Sprintf("%d hr %d min", minutes/60, minutes%60)
}
