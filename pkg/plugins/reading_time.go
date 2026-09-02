// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

const readingTimePluginName = "reading_time"

// ReadingTimePlugin calculates the word count and estimated reading time
// for each post during the transform stage.
type ReadingTimePlugin struct {
	// wordsPerMinute is the average reading speed (default: 200)
	wordsPerMinute int
}

// NewReadingTimePlugin creates a new ReadingTimePlugin with default settings.
func NewReadingTimePlugin() *ReadingTimePlugin {
	return &ReadingTimePlugin{
		wordsPerMinute: defaultWordsPerMinute,
	}
}

// Name returns the unique name of the plugin.
func (p *ReadingTimePlugin) Name() string {
	return readingTimePluginName
}

// Configure reads configuration options for the plugin.
func (p *ReadingTimePlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra != nil {
		if wpm, ok := config.Extra["words_per_minute"].(int); ok && wpm > 0 {
			p.wordsPerMinute = wpm
		}
	}
	return nil
}

// Transform calculates word count and reading time for each post.
func (p *ReadingTimePlugin) Transform(m *lifecycle.Manager) error {
	posts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && post.Content != ""
	})

	if lifecycle.IsServeIncremental(m) {
		if affected := lifecycle.GetServeAffectedPaths(m); len(affected) > 0 {
			filtered := posts[:0]
			for _, post := range posts {
				if affected[post.Path] {
					filtered = append(filtered, post)
				}
			}
			posts = filtered
		}
	}

	return m.ProcessPostsSliceConcurrently(posts, func(post *models.Post) error {
		metrics := calculateReadingTimeMetrics(post.Content, p.wordsPerMinute, false)
		post.Set("word_count", metrics.WordCount)
		post.Set("reading_time", metrics.ReadingTime)
		post.Set("reading_time_text", metrics.ReadingTimeText)

		return nil
	})
}

// countWords counts the number of words in markdown content.
// It excludes code blocks, URLs, and other non-prose elements.
func (p *ReadingTimePlugin) countWords(content string) int {
	return countReadingWords(content, false)
}

// calculateReadingTime estimates reading time in minutes based on word count.
// Returns at least 1 minute for any non-empty content.
func (p *ReadingTimePlugin) calculateReadingTime(wordCount int) int {
	return calculateReadingMinutes(wordCount, p.wordsPerMinute)
}

// formatReadingTime creates a human-readable reading time string.
func (p *ReadingTimePlugin) formatReadingTime(minutes int) string {
	return formatReadingTimeText(minutes)
}

// SetWordsPerMinute sets the average reading speed.
func (p *ReadingTimePlugin) SetWordsPerMinute(wpm int) {
	if wpm > 0 {
		p.wordsPerMinute = wpm
	}
}

// ReadingTimeResult holds the calculated reading metrics for a post.
type ReadingTimeResult struct {
	// WordCount is the number of words in the post
	WordCount int `json:"word_count"`

	// ReadingTime is the estimated reading time in minutes
	ReadingTime int `json:"reading_time"`

	// ReadingTimeText is a formatted reading time string
	ReadingTimeText string `json:"reading_time_text"`
}

// GetReadingTime extracts reading time data from a post's Extra map.
// Returns nil if reading time hasn't been calculated.
func GetReadingTime(post *models.Post) *ReadingTimeResult {
	if post.Extra == nil {
		return nil
	}

	wordCount, hasWC := post.Extra["word_count"].(int)
	readingTime, hasRT := post.Extra["reading_time"].(int)
	readingTimeText, hasRTT := post.Extra["reading_time_text"].(string)

	if !hasWC || !hasRT {
		return nil
	}

	result := &ReadingTimeResult{
		WordCount:   wordCount,
		ReadingTime: readingTime,
	}

	if hasRTT {
		result.ReadingTimeText = readingTimeText
	}

	return result
}

// Ensure ReadingTimePlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*ReadingTimePlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*ReadingTimePlugin)(nil)
	_ lifecycle.TransformPlugin = (*ReadingTimePlugin)(nil)
)
