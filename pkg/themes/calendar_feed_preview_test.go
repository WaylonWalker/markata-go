package themes

import (
	"strings"
	"testing"
)

func TestCalendarFeedPreview_IsLazyAndUsesDropperThumbs(t *testing.T) {
	content, err := ReadStatic("js/calendar-feed.js")
	if err != nil {
		t.Fatalf("ReadStatic(calendar-feed.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"const dropperHosts = new Set([",
		"function dropperThumbURL(raw)",
		"if (!stem.endsWith('_thumb'))",
		"url.pathname = `${stem}_thumb${url.pathname.slice(dot)}`;",
		"for (const key of ['w', 'h', 'width', 'height']) url.searchParams.delete(key);",
		"function previewImageCandidates(raw)",
		"const previewData = new WeakMap();",
		"previewData.set(details, { year, month, day, posts });",
		"const ensurePreview = (details) => {",
		"image.src = imageCandidates[0];",
		"image.src = imageCandidates[1];",
		"preview.style.pointerEvents = 'none';",
		"preview.style.pointerEvents = 'auto';",
		"event.preventDefault();",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("calendar-feed.js missing responsive preview behavior %q", needle)
		}
	}
}
