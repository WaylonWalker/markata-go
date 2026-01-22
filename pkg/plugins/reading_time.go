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

// ReadingTimePlugin calculates the word count and estimated reading time
// for each post during the transform stage.
type ReadingTimePlugin struct {
	// wordsPerMinute is the average reading speed (default: 200)
	wordsPerMinute int
}

// NewReadingTimePlugin creates a new ReadingTimePlugin with default settings.
func NewReadingTimePlugin() *ReadingTimePlugin {
	return &ReadingTimePlugin{
		wordsPerMinute: 200,
	}
}

// Name returns the unique name of the plugin.
func (p *ReadingTimePlugin) Name() string {
	return "reading_time"
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
	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Content == "" {
			return nil
		}

		// Count words
		wordCount := p.countWords(post.Content)
		post.Set("word_count", wordCount)

		// Calculate reading time in minutes
		readingTime := p.calculateReadingTime(wordCount)
		post.Set("reading_time", readingTime)

		// Also store a formatted string
		post.Set("reading_time_text", p.formatReadingTime(readingTime))

		return nil
	})
}

// Regex patterns for word counting
var (
	// Match code blocks to exclude from word count
	codeBlockPattern = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~|`[^`]+`")

	// Match HTML tags
	htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

	// Match markdown link URLs (keep link text)
	linkURLPattern = regexp.MustCompile(`\]\([^)]+\)`)

	// Match markdown image definitions
	imagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)

	// Match URLs
	urlPattern = regexp.MustCompile(`https?://\S+`)
)

// countWords counts the number of words in markdown content.
// It excludes code blocks, URLs, and other non-prose elements.
func (p *ReadingTimePlugin) countWords(content string) int {
	// Remove code blocks
	text := codeBlockPattern.ReplaceAllString(content, " ")

	// Remove images
	text = imagePattern.ReplaceAllString(text, " ")

	// Remove link URLs but keep link text
	text = linkURLPattern.ReplaceAllString(text, "]")

	// Remove standalone URLs
	text = urlPattern.ReplaceAllString(text, " ")

	// Remove HTML tags
	text = htmlTagPattern.ReplaceAllString(text, " ")

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

	// Count words
	words := 0
	inWord := false

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	return words
}

// calculateReadingTime estimates reading time in minutes based on word count.
// Returns at least 1 minute for any non-empty content.
func (p *ReadingTimePlugin) calculateReadingTime(wordCount int) int {
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
func (p *ReadingTimePlugin) formatReadingTime(minutes int) string {
	if minutes == 0 {
		return "< 1 min read"
	}
	if minutes == 1 {
		return "1 min read"
	}
	return fmt.Sprintf("%d min read", minutes)
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
