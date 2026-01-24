// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// StatsPlugin calculates comprehensive content statistics for posts and feeds.
// It provides word count, reading time, character count, and code block metrics
// for individual posts, and aggregates these statistics at the feed level.
type StatsPlugin struct {
	// wordsPerMinute is the average reading speed (default: 200)
	wordsPerMinute int
	// includeCodeInCount includes code block content in word count
	includeCodeInCount bool
	// trackCodeBlocks enables counting of code block lines
	trackCodeBlocks bool
}

// PostStats contains calculated statistics for a single post.
type PostStats struct {
	// WordCount is the number of words in the post
	WordCount int `json:"word_count"`
	// CharCount is the number of characters (excluding whitespace)
	CharCount int `json:"char_count"`
	// ReadingTime is the estimated reading time in minutes
	ReadingTime int `json:"reading_time"`
	// ReadingTimeText is a formatted reading time string
	ReadingTimeText string `json:"reading_time_text"`
	// CodeLines is the number of lines of code in code blocks
	CodeLines int `json:"code_lines"`
	// CodeBlocks is the number of code blocks in the post
	CodeBlocks int `json:"code_blocks"`
}

// FeedStats contains aggregated statistics for a feed.
type FeedStats struct {
	// PostCount is the total number of posts in the feed
	PostCount int `json:"post_count"`
	// TotalWords is the sum of word counts across all posts
	TotalWords int `json:"total_words"`
	// TotalChars is the sum of character counts across all posts
	TotalChars int `json:"total_chars"`
	// TotalReadingTime is the sum of reading times in minutes
	TotalReadingTime int `json:"total_reading_time"`
	// TotalReadingTimeText is a formatted total reading time
	TotalReadingTimeText string `json:"total_reading_time_text"`
	// AverageWords is the average word count per post
	AverageWords int `json:"average_words"`
	// AverageReadingTime is the average reading time per post
	AverageReadingTime int `json:"average_reading_time"`
	// AverageReadingTimeText is formatted average reading time
	AverageReadingTimeText string `json:"average_reading_time_text"`
	// TotalCodeLines is the sum of code lines across all posts
	TotalCodeLines int `json:"total_code_lines"`
	// TotalCodeBlocks is the sum of code blocks across all posts
	TotalCodeBlocks int `json:"total_code_blocks"`
}

// SiteStats contains global statistics across all posts.
type SiteStats struct {
	// TotalPosts is the total number of posts on the site
	TotalPosts int `json:"total_posts"`
	// TotalWords is the sum of word counts across all posts
	TotalWords int `json:"total_words"`
	// TotalChars is the sum of character counts
	TotalChars int `json:"total_chars"`
	// TotalReadingTime is the total reading time in minutes
	TotalReadingTime int `json:"total_reading_time"`
	// TotalReadingTimeText is formatted total reading time
	TotalReadingTimeText string `json:"total_reading_time_text"`
	// AverageWords is the average word count per post
	AverageWords int `json:"average_words"`
	// AverageReadingTime is the average reading time per post
	AverageReadingTime int `json:"average_reading_time"`
	// AverageReadingTimeText is formatted average reading time
	AverageReadingTimeText string `json:"average_reading_time_text"`
	// TotalCodeLines is the total lines of code across the site
	TotalCodeLines int `json:"total_code_lines"`
	// TotalCodeBlocks is the total number of code blocks
	TotalCodeBlocks int `json:"total_code_blocks"`
}

// NewStatsPlugin creates a new StatsPlugin with default settings.
func NewStatsPlugin() *StatsPlugin {
	return &StatsPlugin{
		wordsPerMinute:     200,
		includeCodeInCount: false,
		trackCodeBlocks:    true,
	}
}

// Name returns the unique name of the plugin.
func (p *StatsPlugin) Name() string {
	return "stats"
}

// Configure reads configuration options for the plugin.
func (p *StatsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for stats plugin config
	if statsConfig, ok := config.Extra["stats"].(map[string]interface{}); ok {
		if wpm, ok := statsConfig["words_per_minute"].(int); ok && wpm > 0 {
			p.wordsPerMinute = wpm
		}
		if includeCode, ok := statsConfig["include_code_in_count"].(bool); ok {
			p.includeCodeInCount = includeCode
		}
		if trackCode, ok := statsConfig["track_code_blocks"].(bool); ok {
			p.trackCodeBlocks = trackCode
		}
	}

	// Also check top-level words_per_minute for backwards compatibility
	if wpm, ok := config.Extra["words_per_minute"].(int); ok && wpm > 0 {
		p.wordsPerMinute = wpm
	}

	return nil
}

// Transform calculates statistics for each post.
func (p *StatsPlugin) Transform(m *lifecycle.Manager) error {
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		stats := p.calculatePostStats(post.Content)

		// Store individual stats in Extra for template access
		post.Set("word_count", stats.WordCount)
		post.Set("char_count", stats.CharCount)
		post.Set("reading_time", stats.ReadingTime)
		post.Set("reading_time_text", stats.ReadingTimeText)
		post.Set("code_lines", stats.CodeLines)
		post.Set("code_blocks", stats.CodeBlocks)

		// Also store as a stats object
		post.Set("stats", stats)

		return nil
	})
}

// Collect aggregates statistics at the feed level.
func (p *StatsPlugin) Collect(m *lifecycle.Manager) error {
	feeds := m.Feeds()
	siteStats := &SiteStats{}

	// Track posts already counted for site stats to avoid double-counting
	countedPaths := make(map[string]bool)

	for _, feed := range feeds {
		feedStats := p.calculateFeedStatsFromLifecycle(feed)

		// Store in feed's Extra or runtime data
		// Note: FeedConfig doesn't have Extra, so we store in cache
		cacheKey := fmt.Sprintf("feed_stats.%s", feed.Name)
		m.Cache().Set(cacheKey, feedStats)

		// Aggregate to site stats (avoiding double counting)
		for _, post := range feed.Posts {
			if countedPaths[post.Path] {
				continue
			}
			countedPaths[post.Path] = true
			stats := p.getPostStats(post)
			siteStats.TotalPosts++
			siteStats.TotalWords += stats.WordCount
			siteStats.TotalChars += stats.CharCount
			siteStats.TotalReadingTime += stats.ReadingTime
			siteStats.TotalCodeLines += stats.CodeLines
			siteStats.TotalCodeBlocks += stats.CodeBlocks
		}
	}

	// Calculate averages for site
	if siteStats.TotalPosts > 0 {
		siteStats.AverageWords = siteStats.TotalWords / siteStats.TotalPosts
		siteStats.AverageReadingTime = siteStats.TotalReadingTime / siteStats.TotalPosts
	}
	siteStats.TotalReadingTimeText = p.formatDuration(siteStats.TotalReadingTime)
	siteStats.AverageReadingTimeText = p.formatReadingTime(siteStats.AverageReadingTime)

	// Store site stats in cache
	m.Cache().Set("site_stats", siteStats)

	// Also store in config extra for template access
	config := m.Config()
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}
	config.Extra["site_stats"] = siteStats

	return nil
}

// Regex patterns for stats calculation
var (
	// Match fenced code blocks (``` or ~~~)
	statsCodeBlockPattern = regexp.MustCompile("(?s)```[^`]*```|~~~[^~]*~~~")
	// Match inline code
	statsInlineCodePattern = regexp.MustCompile("`[^`]+`")
	// Match HTML tags
	statsHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)
	// Match markdown link URLs
	statsLinkURLPattern = regexp.MustCompile(`\]\([^)]+\)`)
	// Match markdown images
	statsImagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	// Match URLs
	statsURLPattern = regexp.MustCompile(`https?://\S+`)
)

// calculatePostStats computes all statistics for a post's content.
func (p *StatsPlugin) calculatePostStats(content string) *PostStats {
	stats := &PostStats{}

	// Extract code blocks first
	var codeContent string
	codeBlocks := statsCodeBlockPattern.FindAllString(content, -1)
	stats.CodeBlocks = len(codeBlocks)

	for _, block := range codeBlocks {
		// Remove fence markers and count lines
		lines := strings.Split(block, "\n")
		// Subtract 2 for opening and closing fences
		codeLineCount := len(lines) - 2
		if codeLineCount < 0 {
			codeLineCount = 0
		}
		stats.CodeLines += codeLineCount
		codeContent += block
	}

	// Prepare text for word counting
	text := content
	if !p.includeCodeInCount {
		// Remove code blocks from word counting
		text = statsCodeBlockPattern.ReplaceAllString(text, " ")
	}

	// Remove inline code
	text = statsInlineCodePattern.ReplaceAllString(text, " ")

	// Remove images
	text = statsImagePattern.ReplaceAllString(text, " ")

	// Remove link URLs but keep link text
	text = statsLinkURLPattern.ReplaceAllString(text, "]")

	// Remove standalone URLs
	text = statsURLPattern.ReplaceAllString(text, " ")

	// Remove HTML tags
	text = statsHTMLTagPattern.ReplaceAllString(text, " ")

	// Remove markdown formatting characters
	text = strings.NewReplacer(
		"#", " ",
		"*", " ",
		"_", " ",
		"`", " ",
		">", " ",
		"-", " ",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
	).Replace(text)

	// Count words and characters
	inWord := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			stats.CharCount++
			if !inWord {
				stats.WordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	// Calculate reading time
	stats.ReadingTime = p.calculateReadingTime(stats.WordCount)
	stats.ReadingTimeText = p.formatReadingTime(stats.ReadingTime)

	return stats
}

// calculateFeedStats aggregates stats for a feed's posts (models.FeedConfig).
func (p *StatsPlugin) calculateFeedStats(feed *models.FeedConfig) *FeedStats {
	stats := &FeedStats{
		PostCount: len(feed.Posts),
	}

	for _, post := range feed.Posts {
		postStats := p.getPostStats(post)
		stats.TotalWords += postStats.WordCount
		stats.TotalChars += postStats.CharCount
		stats.TotalReadingTime += postStats.ReadingTime
		stats.TotalCodeLines += postStats.CodeLines
		stats.TotalCodeBlocks += postStats.CodeBlocks
	}

	// Calculate averages
	if stats.PostCount > 0 {
		stats.AverageWords = stats.TotalWords / stats.PostCount
		stats.AverageReadingTime = stats.TotalReadingTime / stats.PostCount
	}

	// Format text values
	stats.TotalReadingTimeText = p.formatDuration(stats.TotalReadingTime)
	stats.AverageReadingTimeText = p.formatReadingTime(stats.AverageReadingTime)

	return stats
}

// calculateFeedStatsFromLifecycle aggregates stats for a lifecycle.Feed.
func (p *StatsPlugin) calculateFeedStatsFromLifecycle(feed *lifecycle.Feed) *FeedStats {
	stats := &FeedStats{
		PostCount: len(feed.Posts),
	}

	for _, post := range feed.Posts {
		postStats := p.getPostStats(post)
		stats.TotalWords += postStats.WordCount
		stats.TotalChars += postStats.CharCount
		stats.TotalReadingTime += postStats.ReadingTime
		stats.TotalCodeLines += postStats.CodeLines
		stats.TotalCodeBlocks += postStats.CodeBlocks
	}

	// Calculate averages
	if stats.PostCount > 0 {
		stats.AverageWords = stats.TotalWords / stats.PostCount
		stats.AverageReadingTime = stats.TotalReadingTime / stats.PostCount
	}

	// Format text values
	stats.TotalReadingTimeText = p.formatDuration(stats.TotalReadingTime)
	stats.AverageReadingTimeText = p.formatReadingTime(stats.AverageReadingTime)

	return stats
}

// getPostStats retrieves stats from a post's Extra map.
func (p *StatsPlugin) getPostStats(post *models.Post) *PostStats {
	stats := &PostStats{}

	if post.Extra != nil {
		if wc, ok := post.Extra["word_count"].(int); ok {
			stats.WordCount = wc
		}
		if cc, ok := post.Extra["char_count"].(int); ok {
			stats.CharCount = cc
		}
		if rt, ok := post.Extra["reading_time"].(int); ok {
			stats.ReadingTime = rt
		}
		if cl, ok := post.Extra["code_lines"].(int); ok {
			stats.CodeLines = cl
		}
		if cb, ok := post.Extra["code_blocks"].(int); ok {
			stats.CodeBlocks = cb
		}
	}

	return stats
}

// calculateReadingTime estimates reading time in minutes.
func (p *StatsPlugin) calculateReadingTime(wordCount int) int {
	if wordCount == 0 {
		return 0
	}

	minutes := float64(wordCount) / float64(p.wordsPerMinute)
	roundedMinutes := int(math.Ceil(minutes))

	if roundedMinutes < 1 {
		return 1
	}

	return roundedMinutes
}

// formatReadingTime creates a human-readable reading time string.
func (p *StatsPlugin) formatReadingTime(minutes int) string {
	if minutes == 0 {
		return "< 1 min read"
	}
	if minutes == 1 {
		return "1 min read"
	}
	return fmt.Sprintf("%d min read", minutes)
}

// formatDuration formats a duration in minutes as hours and minutes.
func (p *StatsPlugin) formatDuration(minutes int) string {
	if minutes == 0 {
		return "0 min"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}

	hours := minutes / 60
	mins := minutes % 60

	if mins == 0 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	if hours == 1 {
		return fmt.Sprintf("1 hour %d min", mins)
	}
	return fmt.Sprintf("%d hours %d min", hours, mins)
}

// Priority returns the plugin priority for the given stage.
// Stats should run early in transform (after content is loaded)
// and late in collect (after feeds are populated).
func (p *StatsPlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageTransform:
		return lifecycle.PriorityEarly // Calculate post stats early
	case lifecycle.StageCollect:
		return lifecycle.PriorityLate // Aggregate after feeds are built
	default:
		return lifecycle.PriorityDefault
	}
}

// GetFeedStats retrieves feed statistics from the cache.
func GetFeedStats(m *lifecycle.Manager, feedSlug string) *FeedStats {
	cacheKey := fmt.Sprintf("feed_stats.%s", feedSlug)
	if stats, ok := m.Cache().Get(cacheKey); ok {
		if feedStats, ok := stats.(*FeedStats); ok {
			return feedStats
		}
	}
	return nil
}

// GetSiteStats retrieves site-wide statistics from the cache.
func GetSiteStats(m *lifecycle.Manager) *SiteStats {
	if stats, ok := m.Cache().Get("site_stats"); ok {
		if siteStats, ok := stats.(*SiteStats); ok {
			return siteStats
		}
	}
	return nil
}

// Ensure StatsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*StatsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*StatsPlugin)(nil)
	_ lifecycle.TransformPlugin = (*StatsPlugin)(nil)
	_ lifecycle.CollectPlugin   = (*StatsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*StatsPlugin)(nil)
)
