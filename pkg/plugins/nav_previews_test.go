package plugins

import (
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

func TestNavPreviews_ResolveAndRender(t *testing.T) {
	visible := true
	description := "A focused article"
	public := &models.Post{
		Href: "/article/", Published: true, Description: &description,
		Tags: []string{"go", "notes"}, Date: datePtr(2026, time.January, 1),
		Extra: map[string]interface{}{"word_count": 800, "reading_time": 4},
	}
	second := &models.Post{
		Href: "/another/", Published: true, Date: datePtr(2026, time.February, 1),
		Extra: map[string]interface{}{"word_count": 200, "reading_time": 1},
	}
	private := &models.Post{Href: "/secret/", Published: true, Private: true, Extra: map[string]interface{}{"word_count": 9000}}
	config := models.NewConfig()
	config.Components.Nav.Enabled = &visible
	config.Components.Nav.Items = []models.NavItem{
		{Label: "Journal", URL: "/journal/"},
		{Label: "Article", URL: "/article/"},
		{Label: "Secret", URL: "/secret/"},
		{Label: "Outside", URL: "https://example.com/journal/", External: true},
	}
	feeds := []models.FeedConfig{{Slug: "journal", Description: "Occasional dispatches", Posts: []*models.Post{public, second, private}}}
	previews := buildNavPreviews(config, []*models.Post{public, second, private}, feeds)
	if len(previews) != 2 {
		t.Fatalf("got %d previews, want 2", len(previews))
	}
	feed := previews["/journal/"]
	if feed["count"] != 2 || feed["words"] != 1000 || feed["minutes"] != 5 || feed["sparkline"] == "" {
		t.Fatalf("unexpected feed preview: %#v", feed)
	}
	if feed["words_display"] != "1,000" || feed["reading_display"] != "5 min" {
		t.Fatalf("unexpected display stats: %#v", feed)
	}
	if previews["/article/"]["description"] != description {
		t.Fatalf("missing article description: %#v", previews["/article/"])
	}

	engine, err := templates.NewEngine("../../templates")
	if err != nil {
		t.Fatal(err)
	}
	ctx := templates.NewContext(nil, "", config)
	ctx.Set("nav_previews", previews)
	html, err := engine.Render("components/nav.html", ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"Occasional dispatches", "Publication rhythm", "A focused article", "target=\"_blank\""} {
		if !strings.Contains(html, fragment) {
			t.Errorf("rendered nav missing %q", fragment)
		}
	}
	if strings.Contains(html, "9000") || strings.Contains(html, "Secret</span>") {
		t.Errorf("private metadata appeared in nav: %s", html)
	}
}

func datePtr(year int, month time.Month, day int) *time.Time {
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &date
}
