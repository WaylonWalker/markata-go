package builderadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newPruneFixture(t *testing.T) *Service {
	t.Helper()
	site := t.TempDir()
	for _, id := range []string{"20261001T010000Z-old", "20261001T020000Z-current"} {
		dir := filepath.Join(site, "releases", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Release</h1>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("releases", "20261001T020000Z-current"), filepath.Join(site, "current")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	return &Service{cfg: Config{SiteDir: site, ReleasesKeep: 1}, leader: true}
}

func TestPruneReleases_PromotionContinuesDuringDeletion(t *testing.T) {
	s := newPruneFixture(t)
	deleting := make(chan string, 1)
	resume := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- s.pruneReleasesWithRemove(func(path string) error { deleting <- path; <-resume; return os.RemoveAll(path) })
	}()
	t.Cleanup(func() {
		close(resume)
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	var detached string
	select {
	case detached = <-deleting:
	case <-time.After(5 * time.Second):
		t.Fatal("prune did not begin deletion")
	}
	if !strings.HasPrefix(filepath.Base(detached), prunedReleasePrefix) {
		t.Fatalf("release was not detached: %s", detached)
	}
	if _, err := os.Stat(filepath.Join(s.cfg.SiteDir, "releases", "20261001T010000Z-old")); !os.IsNotExist(err) {
		t.Fatalf("old release remains discoverable: %v", err)
	}
	work := filepath.Join(s.cfg.SiteDir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "index.html"), []byte("<h1>New release</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	promoted := make(chan error, 1)
	go func() { _, _, err := s.promoteBuild(work); promoted <- err }()
	select {
	case err := <-promoted:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promotion waited for recursive deletion")
	}
	if s.currentReleaseID() == "20261001T020000Z-current" {
		t.Fatal("new release did not become current")
	}
}

func TestPruneReleases_RetriesFailedDeletion(t *testing.T) {
	s := newPruneFixture(t)
	failure := errors.New("disk unavailable")
	if err := s.pruneReleasesWithRemove(func(string) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("error = %v", err)
	}
	detached := filepath.Join(s.cfg.SiteDir, "releases", prunedReleasePrefix+"20261001T010000Z-old")
	if _, err := os.Stat(detached); err != nil {
		t.Fatal(err)
	}
	if views := s.discoverReleases(); len(views) != 1 || !views[0].Current {
		t.Fatalf("releases include detached tree: %+v", views)
	}
	if err := s.pruneReleases(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(detached); !os.IsNotExist(err) {
		t.Fatalf("detached tree not cleaned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.currentReleasePath(), "index.html")); err != nil {
		t.Fatalf("current release damaged: %v", err)
	}
}

func TestPruneReleases_InternalDirectoriesCannotBePublished(t *testing.T) {
	s := newPruneFixture(t)
	for _, id := range []string{".staging-incomplete", ".pruning-old"} {
		if err := os.MkdirAll(filepath.Join(s.cfg.SiteDir, "releases", id), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := s.switchCurrentRelease(id); err == nil {
			t.Fatalf("published internal directory %s", id)
		}
	}
	if got := len(s.discoverReleases()); got != 2 {
		t.Fatalf("discovered %d releases, want 2", got)
	}
	s.cfg.ReleasesKeep = 0
	if err := s.pruneReleases(); err != nil {
		t.Fatal(err)
	}
	if s.currentReleaseID() != "20261001T020000Z-current" {
		t.Fatal("current release pruned")
	}
	if _, err := os.Stat(s.currentReleasePath()); err != nil {
		t.Fatal(err)
	}
}
