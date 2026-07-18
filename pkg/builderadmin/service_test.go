package builderadmin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuilderAdminAuthentication_RejectsUntrustedOrMissingIdentity(t *testing.T) {
	t.Parallel()
	svc, err := New(Config{SiteDir: t.TempDir(), TrustedProxyCIDRs: []string{"10.42.0.0/24"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })
	mux := http.NewServeMux()
	svc.registerRoutes(mux)

	for _, tt := range []struct {
		name       string
		remoteAddr string
		identity   string
	}{
		{name: "missing identity", remoteAddr: "10.42.0.10:443"},
		{name: "untrusted source", remoteAddr: "192.0.2.10:443", identity: "operator"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.identity != "" {
				req.Header.Set(hlabUserIDHeader, tt.identity)
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestBuilderAdminAuthentication_TrustedIdentityAndCSRF(t *testing.T) {
	t.Parallel()
	svc, err := New(Config{
		SiteDir:           t.TempDir(),
		TrustedProxyCIDRs: []string{"10.42.0.0/24"},
		PublicOrigin:      "https://builder.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })
	svc.leader = true
	mux := http.NewServeMux()
	svc.registerRoutes(mux)

	indexRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	indexRequest.RemoteAddr = "10.42.0.10:443"
	indexRequest.Header.Set(hlabUserIDHeader, "operator")
	indexRecorder := httptest.NewRecorder()
	mux.ServeHTTP(indexRecorder, indexRequest)
	if indexRecorder.Code != http.StatusOK {
		t.Fatalf("index status=%d, want %d", indexRecorder.Code, http.StatusOK)
	}
	cookies := indexRecorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != csrfCookieName || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("csrf cookie=%+v", cookies)
	}

	values := url.Values{"csrf_token": {cookies[0].Value}}
	buildRequest := httptest.NewRequest(http.MethodPost, "/api/builds", strings.NewReader(values.Encode()))
	buildRequest.RemoteAddr = "10.42.0.10:443"
	buildRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	buildRequest.Header.Set("Origin", "https://builder.example.com")
	buildRequest.Header.Set(hlabUserIDHeader, "operator")
	buildRequest.AddCookie(cookies[0])
	buildRecorder := httptest.NewRecorder()
	mux.ServeHTTP(buildRecorder, buildRequest)
	if buildRecorder.Code != http.StatusSeeOther {
		t.Fatalf("build status=%d, want %d", buildRecorder.Code, http.StatusSeeOther)
	}
}

func TestNew_RejectsUnsafeTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()
	for _, cidr := range []string{
		"0.0.0.0/0", "::/0", "127.0.0.0/8", "127.0.0.1/32", "::1/128",
		"169.254.0.0/16", "fe80::/10", "126.0.0.0/7", "fe00::/8",
	} {
		t.Run(cidr, func(t *testing.T) {
			_, err := New(Config{SiteDir: t.TempDir(), TrustedProxyCIDRs: []string{cidr}})
			if err == nil {
				t.Fatalf("New() with trusted proxy CIDR %q succeeded, want error", cidr)
			}
		})
	}

	service, err := New(Config{SiteDir: t.TempDir(), TrustedProxyCIDRs: []string{"10.42.0.0/16"}})
	if err != nil {
		t.Fatalf("New() with pod CIDR failed: %v", err)
	}
	t.Cleanup(func() { _ = service.leaderLock.Close() })
}

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
