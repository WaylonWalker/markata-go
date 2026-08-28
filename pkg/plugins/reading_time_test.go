package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestReadingTimePlugin_StatsOwnsOverlappingFields(t *testing.T) {
	m := lifecycle.NewManager()
	post := models.NewPost("post.md")
	post.Content = "one two three"
	post.Set("reading_time", 99)
	m.SetPosts([]*models.Post{post})
	m.RegisterPlugin(NewStatsPlugin())

	if err := NewReadingTimePlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if got := post.Extra["reading_time"]; got != 99 {
		t.Fatalf("reading_time = %v, want existing Stats-owned value", got)
	}
}

func TestReadingTimePlugin_WritesStandaloneFields(t *testing.T) {
	m := lifecycle.NewManager()
	post := models.NewPost("post.md")
	post.Content = "one two three"
	m.SetPosts([]*models.Post{post})

	if err := NewReadingTimePlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if got := post.Extra["word_count"]; got != 3 {
		t.Fatalf("word_count = %v, want 3", got)
	}
	if got := post.Extra["reading_time"]; got != 1 {
		t.Fatalf("reading_time = %v, want 1", got)
	}
}
