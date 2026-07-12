package builderadmin

import (
	"fmt"
	"net/http/httptest"
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
			t.Parallel()
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

func TestIndexHTMLUsesCompactBuildRunDetails(t *testing.T) {
	t.Parallel()
	checks := []string{
		`class="run-list" id="builds-body"`,
		`class="run-details"`,
		`<summary>Details</summary>`,
		`function renderBuilds(items)`,
		`View log`,
		`phaseTiming('Queue wait', item.queue_wait_ms)`,
		`phaseTiming('Prune', item.prune_ms)`,
		`data-build-id`,
		`const openDetails = new Set`,
		`Every {{ .Every }}`,
		`queues a build`,
		`class="card control-panel actions"`,
		`live_label: 'Running'`,
		`live_label: 'Queued'`,
	}
	for _, check := range checks {
		if !strings.Contains(indexHTML, check) {
			t.Fatalf("indexHTML missing %q", check)
		}
	}
}

func TestHandleIndex_BuildDetailsIncludeAllPhaseTimings(t *testing.T) {
	t.Parallel()
	svc, err := New(Config{
		SiteDir: t.TempDir(),
		RefreshTasks: []RefreshTaskConfig{{
			Name:                  "reader-update",
			Every:                 "30m",
			EnqueueBuildOnSuccess: true,
			Args:                  []string{"reader", "update"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if svc.leaderLock != nil {
			_ = svc.leaderLock.Close()
		}
	})
	svc.leader = true
	svc.state.Running = &RunningOperation{Kind: "build", TriggerType: "manual-ui", Detail: "Manual build from admin UI", Phase: "build"}
	svc.state.Queue = []QueuedOperation{{Kind: "refresh", TriggerType: "scheduled-refresh", Detail: "Scheduled reader update"}}
	svc.state.Builds = []BuildRecord{{
		ID:          "build-details",
		Status:      "success",
		TriggerType: "manual-ui",
		TotalMS:     600,
		QueueWaitMS: 100,
		PrepareMS:   200,
		BuildMS:     300,
		PromoteMS:   400,
		PruneMS:     500,
	}}
	recorder := httptest.NewRecorder()
	svc.handleIndex(recorder, httptest.NewRequest("GET", "/", nil))
	body := recorder.Body.String()
	for _, want := range []string{"Running build", "Queued refresh", "Queue wait", "Prepare", "Build", "Promote", "Prune", "0.10s", "0.50s", "reader-update", "Every 30m", "queues a build"} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered index missing %q", want)
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
			t.Parallel()
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
func TestNew_DefaultReleaseRetentionKeepsTwentyFive(t *testing.T) {
	t.Parallel()
	svc, err := New(Config{SiteDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if svc.leaderLock != nil {
			_ = svc.leaderLock.Close()
		}
	})
	if svc.cfg.ReleasesKeep != 25 {
		t.Fatalf("ReleasesKeep=%d, want 25", svc.cfg.ReleasesKeep)
	}
}

func TestPruneReleases_KeepsCurrentAndTwentyFourRollbackTargets(t *testing.T) {
	t.Parallel()
	siteDir := t.TempDir()
	releasesDir := filepath.Join(siteDir, "releases")
	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 30 {
		if err := os.Mkdir(filepath.Join(releasesDir, fmt.Sprintf("release-%02d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("releases", "release-00"), filepath.Join(siteDir, "current")); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SiteDir: siteDir, HistoryDir: filepath.Join(siteDir, ".builder-admin")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if svc.leaderLock != nil {
			_ = svc.leaderLock.Close()
		}
	})
	if err := svc.pruneReleases(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 25 {
		t.Fatalf("release count=%d, want 25", len(entries))
	}
	if _, err := os.Stat(filepath.Join(releasesDir, "release-00")); err != nil {
		t.Fatalf("current release was pruned: %v", err)
	}
}
