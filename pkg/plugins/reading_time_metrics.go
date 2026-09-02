package plugins

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

const defaultWordsPerMinute = 200

// readingTimeMetrics contains the shared normalized text metrics used by the
// ReadingTimePlugin and StatsPlugin. CharCount is shared because it uses the
// same normalized letters-and-digits stream as WordCount.
type readingTimeMetrics struct {
	WordCount       int
	CharCount       int
	ReadingTime     int
	ReadingTimeText string
}

var (
	// Match fenced code blocks and inline code, which are excluded by default.
	codeBlockPattern = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~|`[^`]+`")

	// Match HTML tags.
	htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

	// Match markdown link URLs while keeping link text.
	linkURLPattern = regexp.MustCompile(`\]\([^)]+\)`)

	// Match markdown image definitions.
	imagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)

	// Match standalone URLs.
	urlPattern = regexp.MustCompile(`https?://\S+`)

	// Use one markdown normalizer for all shared reading-time calculations.
	readingTimeMarkdownReplacer = strings.NewReplacer(
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
	)
)

// calculateReadingTimeMetrics computes all post-level reading-time fields.
// The includeCode option is retained for standalone StatsPlugin users.
func calculateReadingTimeMetrics(content string, wordsPerMinute int, includeCode bool) readingTimeMetrics {
	text := normalizeReadingText(content, includeCode)
	wordCount, charCount := countReadingMetrics(text)
	readingTime := calculateReadingMinutes(wordCount, wordsPerMinute)
	return readingTimeMetrics{
		WordCount:       wordCount,
		CharCount:       charCount,
		ReadingTime:     readingTime,
		ReadingTimeText: formatReadingTimeText(readingTime),
	}
}

// countReadingWords counts prose words using the canonical reading-time rules.
func countReadingWords(content string, includeCode bool) int {
	text := normalizeReadingText(content, includeCode)
	wordCount, _ := countReadingMetrics(text)
	return wordCount
}

func normalizeReadingText(content string, includeCode bool) string {
	text := content
	if includeCode {
		text = includeCodeContent(content)
	} else {
		text = codeBlockPattern.ReplaceAllString(text, " ")
	}

	text = imagePattern.ReplaceAllString(text, " ")
	text = linkURLPattern.ReplaceAllString(text, "]")
	text = urlPattern.ReplaceAllString(text, " ")
	text = htmlTagPattern.ReplaceAllString(text, " ")
	return readingTimeMarkdownReplacer.Replace(text)
}

func countReadingMetrics(text string) (wordCount, charCount int) {
	inWord := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			charCount++
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return wordCount, charCount
}

// includeCodeContent removes fence markers while preserving fenced code text.
// This keeps StatsPlugin's include_code_in_count behavior in the shared
// calculator without creating a second word-count implementation.
func includeCodeContent(content string) string {
	blocks := extractCodeBlocks(content)
	text := content
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		replacement := " "
		if len(lines) > 2 {
			replacement = strings.Join(lines[1:len(lines)-1], " ")
		}
		text = strings.Replace(text, block, replacement, 1)
	}
	return text
}

// calculateReadingMinutes converts a word count to the established reading
// time format: zero for no words, otherwise a ceiling with a one-minute
// minimum.
func calculateReadingMinutes(wordCount, wordsPerMinute int) int {
	if wordCount == 0 {
		return 0
	}
	if wordsPerMinute <= 0 {
		wordsPerMinute = defaultWordsPerMinute
	}

	minutes := float64(wordCount) / float64(wordsPerMinute)
	roundedMinutes := int(math.Ceil(minutes))
	if roundedMinutes < 1 {
		return 1
	}
	return roundedMinutes
}

// formatReadingTimeText creates the established human-readable value.
func formatReadingTimeText(minutes int) string {
	if minutes == 0 {
		return "< 1 min read"
	}
	if minutes == 1 {
		return "1 min read"
	}
	return fmt.Sprintf("%d min read", minutes)
}
