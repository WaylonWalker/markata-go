package themes

import (
	"strings"
	"testing"
)

func TestCalendarFeedMonthGridUsesResponsiveDensitySteps(t *testing.T) {
	content, err := ReadStatic("css/calendar-feed.css")
	if err != nil {
		t.Fatalf("ReadStatic(calendar-feed.css) error = %v", err)
	}

	css := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"@media (min-width: 30rem)",
		"grid-template-columns: repeat(2, minmax(0, 1fr));",
		"@media (min-width: 48rem)",
		"grid-template-columns: repeat(3, minmax(0, 1fr));",
		"@media (min-width: 72rem)",
		"grid-template-columns: repeat(4, minmax(0, 1fr));",
		"@media (min-width: 96rem)",
		"grid-template-columns: repeat(6, minmax(0, 1fr));",
	} {
		if !strings.Contains(css, needle) {
			t.Fatalf("calendar-feed.css missing responsive month density contract %q", needle)
		}
	}

	if strings.Contains(css, "minmax(min(100%, 17rem), 1fr)") {
		t.Fatalf("calendar-feed.css still uses the old wide auto-fit month grid")
	}
}
