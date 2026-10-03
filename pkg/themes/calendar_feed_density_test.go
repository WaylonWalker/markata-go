package themes

import (
	"strings"
	"testing"
)

func TestCalendarFeedMonthGridUsesAvailableContainerWidth(t *testing.T) {
	content, err := ReadStatic("css/calendar-feed.css")
	if err != nil {
		t.Fatalf("ReadStatic(calendar-feed.css) error = %v", err)
	}

	css := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"grid-template-columns: repeat(auto-fit, minmax(min(100%, 17rem), 1fr));",
		"min-height: 2.5rem;",
		".calendar-year-header",
	} {
		if !strings.Contains(css, needle) {
			t.Fatalf("calendar-feed.css missing readable month sizing %q", needle)
		}
	}
	if strings.Contains(css, "grid-template-columns: repeat(6, minmax(0, 1fr));") {
		t.Fatal("calendar still squeezes six months into narrow feed containers")
	}
}
