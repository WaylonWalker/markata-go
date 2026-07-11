package builderadmin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestReleaseTimestampFromID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		id   string
		want string
		ok   bool
	}{
		{name: "builder admin style", id: "20260711T203402Z-pod-name", want: "2026-07-11T20:34:02Z", ok: true},
		{name: "legacy numeric", id: "20260711160013", want: "2026-07-11T16:00:13Z", ok: true},
		{name: "invalid", id: "not-a-release", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := releaseTimestampFromID(tt.id)
			if ok != tt.ok {
				t.Fatalf("releaseTimestampFromID(%q) ok=%v want %v", tt.id, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got.Format(time.RFC3339) != tt.want {
				t.Fatalf("releaseTimestampFromID(%q)=%s want %s", tt.id, got.Format(time.RFC3339), tt.want)
			}
		})
	}
}

func TestDiscoverReleasesPrefersBuildFinishedAtAndCurrentFirst(t *testing.T) {
	t.Parallel()
	siteDir := t.TempDir()
	historyDir := filepath.Join(siteDir, ".builder-admin")
	releasesDir := filepath.Join(siteDir, "releases")
	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := "20260711T153738Z-old"
	current := "20260711T203402Z-current"
	for _, releaseID := range []string{older, current} {
		if err := os.MkdirAll(filepath.Join(releasesDir, releaseID), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("releases", current), filepath.Join(siteDir, "current")); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SiteDir: siteDir, HistoryDir: historyDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if svc.leaderLock != nil {
			_ = svc.leaderLock.Close()
		}
	})
	svc.state.Builds = []BuildRecord{
		{ID: "build-new", ReleaseID: current, Status: "success", FinishedAt: time.Date(2026, 7, 11, 20, 34, 4, 0, time.UTC)},
		{ID: "build-old", ReleaseID: older, Status: "success", FinishedAt: time.Date(2026, 7, 11, 15, 37, 50, 0, time.UTC)},
	}
	svc.leader = true
	views := svc.discoverReleases()
	if len(views) != 2 {
		t.Fatalf("discoverReleases() len=%d want 2", len(views))
	}
	if !views[0].Current || views[0].ID != current {
		t.Fatalf("views[0]=%+v want current release first", views[0])
	}
	if views[0].BuildID != "build-new" {
		t.Fatalf("views[0].BuildID=%q want build-new", views[0].BuildID)
	}
	if got := views[0].CreatedAt.Format(time.RFC3339); got != "2026-07-11T20:34:04Z" {
		t.Fatalf("views[0].CreatedAt=%s want 2026-07-11T20:34:04Z", got)
	}
}
