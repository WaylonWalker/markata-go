package builderadmin

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaimReusableWorkspaceConsumesMatchingMarker(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(workDir, "release-a"); err != nil {
		t.Fatal(err)
	}

	reusable, err := claimReusableWorkspace(workDir, "release-a")
	if err != nil {
		t.Fatal(err)
	}
	if !reusable {
		t.Fatal("matching workspace was not reusable")
	}
	if _, err := os.Stat(workspaceReuseMarkerPath(workDir)); !os.IsNotExist(err) {
		t.Fatalf("workspace marker remained after claim: %v", err)
	}

	reusable, err = claimReusableWorkspace(workDir, "release-a")
	if err != nil {
		t.Fatal(err)
	}
	if reusable {
		t.Fatal("claimed workspace was reusable again without a successful promotion")
	}
}

func TestClaimReusableWorkspaceRejectsRollbackOrMismatch(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(workDir, "release-new"); err != nil {
		t.Fatal(err)
	}

	reusable, err := claimReusableWorkspace(workDir, "release-old")
	if err != nil {
		t.Fatal(err)
	}
	if reusable {
		t.Fatal("workspace from a different release was reused")
	}
}

func TestFailedBuildCannotReuseClaimedWorkspace(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "index.html"), []byte("clean"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(workDir, "release-a"); err != nil {
		t.Fatal(err)
	}

	reusable, err := claimReusableWorkspace(workDir, "release-a")
	if err != nil || !reusable {
		t.Fatalf("claim = %t, %v", reusable, err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "index.html"), []byte("partial failed build"), 0o644); err != nil {
		t.Fatal(err)
	}

	reusable, err = claimReusableWorkspace(workDir, "release-a")
	if err != nil {
		t.Fatal(err)
	}
	if reusable {
		t.Fatal("failed build workspace was treated as a clean reusable base")
	}
}

func TestMarkReusableWorkspaceRequiresRetainedDirectory(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "missing-work")
	marker := workspaceReuseMarkerPath(workDir)
	if err := os.WriteFile(marker, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := markReusableWorkspace(workDir, "release-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("stale marker remained when same-filesystem promotion consumed workspace: %v", err)
	}
}

func TestInvalidateReusableWorkspace(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(workDir, "release-a"); err != nil {
		t.Fatal(err)
	}
	if err := invalidateReusableWorkspace(workDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workspaceReuseMarkerPath(workDir)); !os.IsNotExist(err) {
		t.Fatalf("workspace marker remained after invalidation: %v", err)
	}
}

func TestPrepareBuild_ReusesRetainedWorkspaceAndReseedsAfterFailure(t *testing.T) {
	svc, release := newReusableWorkspaceService(t)
	if err := svc.prepareBuild(io.Discard); err != nil {
		t.Fatal(err)
	}
	workFile := filepath.Join(svc.cfg.WorkDir, "index.html")
	before, err := os.Stat(workFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(svc.cfg.WorkDir, "release-a"); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	if err := svc.prepareBuild(&logs); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(workFile)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !strings.Contains(logs.String(), "reusing build work") {
		t.Fatal("warm preparation recopied the release")
	}
	// A failed writer leaves partial output. The next preparation must discard it.
	if err := os.WriteFile(workFile, []byte("partial failed output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.prepareBuild(io.Discard); err != nil {
		t.Fatal(err)
	}
	assertWorkspaceContent(t, workFile, "live")
	assertWorkspaceContent(t, filepath.Join(release, "index.html"), "live")
}

func TestPrepareBuild_RollbackReseedsRetainedWorkspace(t *testing.T) {
	svc, release := newReusableWorkspaceService(t)
	if err := svc.prepareBuild(io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(svc.cfg.WorkDir, "different-release"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(svc.cfg.WorkDir, "index.html"), []byte("different release"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.prepareBuild(io.Discard); err != nil {
		t.Fatal(err)
	}
	assertWorkspaceContent(t, filepath.Join(svc.cfg.WorkDir, "index.html"), "live")
	assertWorkspaceContent(t, filepath.Join(release, "index.html"), "live")
}

func TestPrepareBuild_NoCurrentReleaseClearsStaleProof(t *testing.T) {
	root := t.TempDir()
	svc := &Service{cfg: Config{SiteDir: filepath.Join(root, "site"), WorkDir: filepath.Join(root, "work")}}
	if err := os.MkdirAll(svc.cfg.WorkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(svc.cfg.WorkDir, "stale.html"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := markReusableWorkspace(svc.cfg.WorkDir, "old-release"); err != nil {
		t.Fatal(err)
	}
	if err := svc.prepareBuild(io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(svc.cfg.WorkDir, "stale.html"), workspaceReuseMarkerPath(svc.cfg.WorkDir)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("stale output or proof remains: %s: %v", path, err)
		}
	}
}

func newReusableWorkspaceService(t *testing.T) (*Service, string) {
	t.Helper()
	if _, err := exec.LookPath("cp"); err != nil {
		t.Skip("release seeding requires cp")
	}
	root := t.TempDir()
	site := filepath.Join(root, "site")
	release := filepath.Join(site, "releases", "release-a")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, "index.html"), []byte("live"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "release-a"), filepath.Join(site, "current")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return &Service{cfg: Config{SiteDir: site, WorkDir: filepath.Join(root, "work")}}, release
}

func assertWorkspaceContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("%s = %q, error=%v; want %q", path, data, err, want)
	}
}
