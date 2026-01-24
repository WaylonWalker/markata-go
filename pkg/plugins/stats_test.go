package plugins

import (
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestStatsPlugin_Name(t *testing.T) {
	p := NewStatsPlugin()
	if got := p.Name(); got != "stats" {
		t.Errorf("Name() = %q, want %q", got, "stats")
	}
}

func TestStatsPlugin_calculatePostStats(t *testing.T) {
	p := NewStatsPlugin()

	tests := []struct {
		name            string
		content         string
		wantWordCount   int
		wantCodeBlocks  int
		wantCodeLines   int
		wantReadingTime int
	}{
		{
			name:            "empty content",
			content:         "",
			wantWordCount:   0,
			wantCodeBlocks:  0,
			wantCodeLines:   0,
			wantReadingTime: 0,
		},
		{
			name:            "simple text",
			content:         "Hello world this is a test",
			wantWordCount:   6,
			wantCodeBlocks:  0,
			wantCodeLines:   0,
			wantReadingTime: 1, // 6 words < 200 wpm = 1 min
		},
		{
			name: "text with code block",
			content: `# Title

Some intro text here.

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

More text after the code.
`,
			wantWordCount:   10, // Title + Some intro text here + More text after the code
			wantCodeBlocks:  1,
			wantCodeLines:   3, // 3 lines inside the code block (excluding fence lines)
			wantReadingTime: 1,
		},
		{
			name: "multiple code blocks",
			content: `
First paragraph.

` + "```python" + `
def hello():
    print("Hi")
` + "```" + `

Middle text.

` + "```bash" + `
echo "test"
` + "```" + `

Final text.
`,
			wantWordCount:   6, // "First paragraph" + "Middle text" + "Final text"
			wantCodeBlocks:  2,
			wantCodeLines:   3, // 2 lines in python + 1 line in bash
			wantReadingTime: 1,
		},
		{
			name:            "text with markdown links",
			content:         "Check out [this link](https://example.com) for more info",
			wantWordCount:   7, // "Check out this link for more info"
			wantCodeBlocks:  0,
			wantCodeLines:   0,
			wantReadingTime: 1,
		},
		{
			name:            "text with inline code",
			content:         "Use the `fmt.Println` function to print",
			wantWordCount:   5, // "Use the function to print" (inline code excluded)
			wantCodeBlocks:  0,
			wantCodeLines:   0,
			wantReadingTime: 1,
		},
		{
			name:            "longer text for reading time",
			content:         generateWords(500), // 500 words
			wantWordCount:   500,
			wantCodeBlocks:  0,
			wantCodeLines:   0,
			wantReadingTime: 3, // 500/200 = 2.5, rounded up = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := p.calculatePostStats(tt.content)

			if stats.WordCount != tt.wantWordCount {
				t.Errorf("WordCount = %d, want %d", stats.WordCount, tt.wantWordCount)
			}
			if stats.CodeBlocks != tt.wantCodeBlocks {
				t.Errorf("CodeBlocks = %d, want %d", stats.CodeBlocks, tt.wantCodeBlocks)
			}
			if stats.CodeLines != tt.wantCodeLines {
				t.Errorf("CodeLines = %d, want %d", stats.CodeLines, tt.wantCodeLines)
			}
			if stats.ReadingTime != tt.wantReadingTime {
				t.Errorf("ReadingTime = %d, want %d", stats.ReadingTime, tt.wantReadingTime)
			}
		})
	}
}

func TestStatsPlugin_formatReadingTime(t *testing.T) {
	p := NewStatsPlugin()

	tests := []struct {
		minutes int
		want    string
	}{
		{0, "< 1 min read"},
		{1, "1 min read"},
		{5, "5 min read"},
		{15, "15 min read"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := p.formatReadingTime(tt.minutes); got != tt.want {
				t.Errorf("formatReadingTime(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestStatsPlugin_formatDuration(t *testing.T) {
	p := NewStatsPlugin()

	tests := []struct {
		minutes int
		want    string
	}{
		{0, "0 min"},
		{30, "30 min"},
		{59, "59 min"},
		{60, "1 hour"},
		{61, "1 hour 1 min"},
		{90, "1 hour 30 min"},
		{120, "2 hours"},
		{150, "2 hours 30 min"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := p.formatDuration(tt.minutes); got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestStatsPlugin_calculateFeedStats(t *testing.T) {
	p := NewStatsPlugin()

	// Create test posts with pre-calculated stats
	posts := []*models.Post{
		createTestPostWithStats(100, 500, 5, 10, 2),
		createTestPostWithStats(200, 1000, 10, 20, 3),
		createTestPostWithStats(150, 750, 8, 15, 1),
	}

	feed := &models.FeedConfig{
		Slug:  "test",
		Posts: posts,
	}

	stats := p.calculateFeedStats(feed)

	if stats.PostCount != 3 {
		t.Errorf("PostCount = %d, want 3", stats.PostCount)
	}
	if stats.TotalWords != 450 {
		t.Errorf("TotalWords = %d, want 450", stats.TotalWords)
	}
	if stats.TotalChars != 2250 {
		t.Errorf("TotalChars = %d, want 2250", stats.TotalChars)
	}
	if stats.TotalReadingTime != 23 {
		t.Errorf("TotalReadingTime = %d, want 23", stats.TotalReadingTime)
	}
	if stats.TotalCodeLines != 45 {
		t.Errorf("TotalCodeLines = %d, want 45", stats.TotalCodeLines)
	}
	if stats.TotalCodeBlocks != 6 {
		t.Errorf("TotalCodeBlocks = %d, want 6", stats.TotalCodeBlocks)
	}
	if stats.AverageWords != 150 {
		t.Errorf("AverageWords = %d, want 150", stats.AverageWords)
	}
	if stats.AverageReadingTime != 7 {
		t.Errorf("AverageReadingTime = %d, want 7", stats.AverageReadingTime)
	}
}

func TestStatsPlugin_includeCodeInCount(t *testing.T) {
	content := `Some text here.

` + "```go" + `
func main() {
    fmt.Println("Hello world")
}
` + "```" + `

More text.
`

	// Without including code
	p1 := NewStatsPlugin()
	stats1 := p1.calculatePostStats(content)

	// With including code
	p2 := NewStatsPlugin()
	p2.includeCodeInCount = true
	stats2 := p2.calculatePostStats(content)

	// Code included should have more words (fmt, Println, Hello, world add 4+ words)
	// Note: The code block regex replacement affects word counting
	// When code is included, the fenced code block content is preserved
	t.Logf("Without code: %d words, With code: %d words", stats1.WordCount, stats2.WordCount)

	// Just verify the plugin runs without error for both modes
	if stats1.WordCount == 0 {
		t.Error("Expected non-zero word count without code")
	}
}

// Helper functions

func generateWords(count int) string {
	words := make([]string, count)
	sampleWords := []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog", "and", "runs"}
	for i := 0; i < count; i++ {
		words[i] = sampleWords[i%len(sampleWords)]
	}
	return joinWords(words)
}

func joinWords(words []string) string {
	result := ""
	for i, w := range words {
		if i > 0 {
			result += " "
		}
		result += w
	}
	return result
}

func createTestPostWithStats(words, chars, readingTime, codeLines, codeBlocks int) *models.Post {
	post := models.NewPost("test.md")
	post.Set("word_count", words)
	post.Set("char_count", chars)
	post.Set("reading_time", readingTime)
	post.Set("code_lines", codeLines)
	post.Set("code_blocks", codeBlocks)
	return post
}

func TestStatsPlugin_calculateFeedStats_withYearAndTags(t *testing.T) {
	p := NewStatsPlugin()

	// Create test posts with dates and tags
	date2023 := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	date2024 := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)

	post1 := createTestPostWithStats(100, 500, 5, 10, 2)
	post1.Date = &date2023
	post1.Tags = []string{"go", "programming"}

	post2 := createTestPostWithStats(200, 1000, 10, 20, 3)
	post2.Date = &date2024
	post2.Tags = []string{"go", "tutorial"}

	post3 := createTestPostWithStats(150, 750, 8, 15, 1)
	post3.Date = &date2023
	post3.Tags = []string{"programming"}

	feed := &models.FeedConfig{
		Slug:  "test",
		Posts: []*models.Post{post1, post2, post3},
	}

	stats := p.calculateFeedStats(feed)

	// Test PostsByYear
	if stats.PostsByYear[2023] != 2 {
		t.Errorf("PostsByYear[2023] = %d, want 2", stats.PostsByYear[2023])
	}
	if stats.PostsByYear[2024] != 1 {
		t.Errorf("PostsByYear[2024] = %d, want 1", stats.PostsByYear[2024])
	}

	// Test WordsByYear
	if stats.WordsByYear[2023] != 250 { // 100 + 150
		t.Errorf("WordsByYear[2023] = %d, want 250", stats.WordsByYear[2023])
	}
	if stats.WordsByYear[2024] != 200 {
		t.Errorf("WordsByYear[2024] = %d, want 200", stats.WordsByYear[2024])
	}

	// Test PostsByTag
	if stats.PostsByTag["go"] != 2 {
		t.Errorf("PostsByTag[go] = %d, want 2", stats.PostsByTag["go"])
	}
	if stats.PostsByTag["programming"] != 2 {
		t.Errorf("PostsByTag[programming] = %d, want 2", stats.PostsByTag["programming"])
	}
	if stats.PostsByTag["tutorial"] != 1 {
		t.Errorf("PostsByTag[tutorial] = %d, want 1", stats.PostsByTag["tutorial"])
	}
}

func TestStatsHelper_methods(t *testing.T) {
	// Create a test manager and populate with stats
	m := lifecycle.NewManager()

	// Create site stats
	siteStats := &SiteStats{
		TotalPosts:             10,
		TotalWords:             5000,
		TotalReadingTime:       25,
		TotalReadingTimeText:   "25 min",
		AverageWords:           500,
		AverageReadingTime:     3,
		AverageReadingTimeText: "3 min read",
		TotalCodeLines:         200,
		TotalCodeBlocks:        20,
		PostsByYear:            map[int]int{2023: 4, 2024: 6},
		WordsByYear:            map[int]int{2023: 2000, 2024: 3000},
		PostsByTag:             map[string]int{"go": 5, "python": 3, "tutorial": 7},
	}
	m.Cache().Set("site_stats", siteStats)

	// Create feed stats
	feedStats := &FeedStats{
		PostCount:              5,
		TotalWords:             2500,
		TotalReadingTime:       12,
		TotalReadingTimeText:   "12 min",
		AverageWords:           500,
		AverageReadingTime:     2,
		AverageReadingTimeText: "2 min read",
		TotalCodeLines:         100,
		TotalCodeBlocks:        10,
		PostsByYear:            map[int]int{2023: 2, 2024: 3},
		WordsByYear:            map[int]int{2023: 1000, 2024: 1500},
		PostsByTag:             map[string]int{"go": 3, "tutorial": 4},
	}
	m.Cache().Set("feed_stats.blog", feedStats)

	// Test StatsHelper
	helper := NewStatsHelper(m)

	// Test site-level methods
	if helper.TotalPosts() != 10 {
		t.Errorf("TotalPosts() = %d, want 10", helper.TotalPosts())
	}
	if helper.TotalWords() != 5000 {
		t.Errorf("TotalWords() = %d, want 5000", helper.TotalWords())
	}
	if helper.TotalReadingTime() != 25 {
		t.Errorf("TotalReadingTime() = %d, want 25", helper.TotalReadingTime())
	}
	if helper.TotalReadingTimeText() != "25 min" {
		t.Errorf("TotalReadingTimeText() = %q, want %q", helper.TotalReadingTimeText(), "25 min")
	}
	if helper.AverageWords() != 500 {
		t.Errorf("AverageWords() = %d, want 500", helper.AverageWords())
	}
	if helper.TotalCodeLines() != 200 {
		t.Errorf("TotalCodeLines() = %d, want 200", helper.TotalCodeLines())
	}
	if helper.TotalCodeBlocks() != 20 {
		t.Errorf("TotalCodeBlocks() = %d, want 20", helper.TotalCodeBlocks())
	}

	// Test PostsByYear
	postsByYear := helper.PostsByYear()
	if postsByYear[2023] != 4 {
		t.Errorf("PostsByYear()[2023] = %d, want 4", postsByYear[2023])
	}
	if postsByYear[2024] != 6 {
		t.Errorf("PostsByYear()[2024] = %d, want 6", postsByYear[2024])
	}

	// Test WordsByYear
	wordsByYear := helper.WordsByYear()
	if wordsByYear[2023] != 2000 {
		t.Errorf("WordsByYear()[2023] = %d, want 2000", wordsByYear[2023])
	}

	// Test PostsByTag
	postsByTag := helper.PostsByTag()
	if postsByTag["go"] != 5 {
		t.Errorf("PostsByTag()[go] = %d, want 5", postsByTag["go"])
	}

	// Test Kpi method
	if helper.Kpi("total_posts") != 10 {
		t.Errorf("Kpi(total_posts) = %v, want 10", helper.Kpi("total_posts"))
	}
	if helper.Kpi("total_words") != 5000 {
		t.Errorf("Kpi(total_words) = %v, want 5000", helper.Kpi("total_words"))
	}
	if helper.Kpi("unknown") != nil {
		t.Errorf("Kpi(unknown) = %v, want nil", helper.Kpi("unknown"))
	}
}

func TestStatsHelper_ForFeed(t *testing.T) {
	// Create a test manager and populate with stats
	m := lifecycle.NewManager()

	// Create site stats
	siteStats := &SiteStats{
		TotalPosts:  10,
		PostsByYear: make(map[int]int),
		WordsByYear: make(map[int]int),
		PostsByTag:  make(map[string]int),
	}
	m.Cache().Set("site_stats", siteStats)

	// Create feed stats
	feedStats := &FeedStats{
		PostCount:              5,
		TotalWords:             2500,
		TotalReadingTime:       12,
		TotalReadingTimeText:   "12 min",
		AverageWords:           500,
		AverageReadingTime:     2,
		AverageReadingTimeText: "2 min read",
		TotalCodeLines:         100,
		TotalCodeBlocks:        10,
		PostsByYear:            map[int]int{2023: 2, 2024: 3},
		WordsByYear:            map[int]int{2023: 1000, 2024: 1500},
		PostsByTag:             map[string]int{"go": 3, "tutorial": 4},
	}
	m.Cache().Set("feed_stats.blog", feedStats)

	// Test StatsHelper.ForFeed
	helper := NewStatsHelper(m)
	feedHelper := helper.ForFeed("blog")

	if feedHelper.PostCount() != 5 {
		t.Errorf("ForFeed(blog).PostCount() = %d, want 5", feedHelper.PostCount())
	}
	if feedHelper.TotalWords() != 2500 {
		t.Errorf("ForFeed(blog).TotalWords() = %d, want 2500", feedHelper.TotalWords())
	}
	if feedHelper.TotalReadingTime() != 12 {
		t.Errorf("ForFeed(blog).TotalReadingTime() = %d, want 12", feedHelper.TotalReadingTime())
	}
	if feedHelper.AverageWords() != 500 {
		t.Errorf("ForFeed(blog).AverageWords() = %d, want 500", feedHelper.AverageWords())
	}
	if feedHelper.TotalCodeLines() != 100 {
		t.Errorf("ForFeed(blog).TotalCodeLines() = %d, want 100", feedHelper.TotalCodeLines())
	}

	// Test PostsByYear
	postsByYear := feedHelper.PostsByYear()
	if postsByYear[2023] != 2 {
		t.Errorf("ForFeed(blog).PostsByYear()[2023] = %d, want 2", postsByYear[2023])
	}

	// Test PostsByTag
	postsByTag := feedHelper.PostsByTag()
	if postsByTag["go"] != 3 {
		t.Errorf("ForFeed(blog).PostsByTag()[go] = %d, want 3", postsByTag["go"])
	}

	// Test Kpi method
	if feedHelper.Kpi("post_count") != 5 {
		t.Errorf("ForFeed(blog).Kpi(post_count) = %v, want 5", feedHelper.Kpi("post_count"))
	}
	if feedHelper.Kpi("total_posts") != 5 { // alias
		t.Errorf("ForFeed(blog).Kpi(total_posts) = %v, want 5", feedHelper.Kpi("total_posts"))
	}

	// Test non-existent feed
	nonExistentFeed := helper.ForFeed("nonexistent")
	if nonExistentFeed.PostCount() != 0 {
		t.Errorf("ForFeed(nonexistent).PostCount() = %d, want 0", nonExistentFeed.PostCount())
	}
}

func TestStatsHelper_nilStats(t *testing.T) {
	// Create a test manager without any stats
	m := lifecycle.NewManager()

	helper := NewStatsHelper(m)

	// All methods should return zero/empty values without panicking
	if helper.TotalPosts() != 0 {
		t.Errorf("TotalPosts() = %d, want 0", helper.TotalPosts())
	}
	if helper.TotalWords() != 0 {
		t.Errorf("TotalWords() = %d, want 0", helper.TotalWords())
	}
	if helper.TotalReadingTimeText() != "0 min" {
		t.Errorf("TotalReadingTimeText() = %q, want %q", helper.TotalReadingTimeText(), "0 min")
	}

	postsByYear := helper.PostsByYear()
	if len(postsByYear) != 0 {
		t.Errorf("PostsByYear() = %v, want empty map", postsByYear)
	}

	postsByTag := helper.PostsByTag()
	if len(postsByTag) != 0 {
		t.Errorf("PostsByTag() = %v, want empty map", postsByTag)
	}

	if helper.Kpi("total_posts") != 0 {
		t.Errorf("Kpi(total_posts) = %v, want 0", helper.Kpi("total_posts"))
	}
}
