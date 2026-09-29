package builderadmin

import (
	"os"
	"path/filepath"
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
