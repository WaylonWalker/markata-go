package builderadmin

import (
	"strings"
	"testing"
)

func TestIgnoreWatchPath(t *testing.T) {
	t.Parallel()
	root := "/tmp/site"
	tests := []struct {
		path string
		want bool
	}{
		{path: "/tmp/site/pages/post.md", want: false},
		{path: "/tmp/site/.git/index", want: true},
		{path: "/tmp/site/.markata/cache.json", want: true},
		{path: "/tmp/site/.builder-admin/state.json", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := ignoreWatchPath(root, tt.path); got != tt.want {
				t.Fatalf("ignoreWatchPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractPerfSummaryFromFileMissing(t *testing.T) {
	t.Parallel()
	if got := extractPerfSummaryFromFile("/does/not/exist"); got != nil {
		t.Fatalf("extractPerfSummaryFromFile() = %#v, want nil", got)
	}
}

func TestIndexHTMLIncludesDynamicFavicon(t *testing.T) {
	t.Parallel()
	checks := []string{
		`id="app-favicon"`,
		`function updateFavicon(stateName)`,
		`function faviconState(state)`,
		`updateFavicon('error');`,
	}
	for _, check := range checks {
		if !strings.Contains(indexHTML, check) {
			t.Fatalf("indexHTML missing %q", check)
		}
	}
}
