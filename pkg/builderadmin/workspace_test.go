package builderadmin

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeWorkspaceFixture(t *testing.T, workspace string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(workspace, "post"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "post", "index.html"), []byte("<h1>hello</h1>"), 0o640); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink("post/index.html", filepath.Join(workspace, "index-link.html")); err != nil {
			t.Fatal(err)
		}
	}
}

func assertWorkspaceRelease(t *testing.T, finalPath string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(finalPath, "post", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "<h1>hello</h1>" {
		t.Fatalf("copied content = %q", content)
	}
}

func TestPromoteWorkspaceReleaseUsesRenameFastPath(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	releases := filepath.Join(root, "releases")
	writeWorkspaceFixture(t, workspace)
	called := false
	finalPath, err := promoteWorkspaceReleaseWithRename(workspace, releases, "release-1", func(oldPath, newPath string) error {
		called = true
		return os.Rename(oldPath, newPath)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("rename fast path was not attempted")
	}
	assertWorkspaceRelease(t, finalPath)
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("same-filesystem rename should consume workspace, stat err = %v", err)
	}
}

func TestPromoteWorkspaceReleaseFallsBackToStagedCopy(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	releases := filepath.Join(root, "releases")
	writeWorkspaceFixture(t, workspace)
	finalPath, err := promoteWorkspaceReleaseWithRename(workspace, releases, "release-1", func(string, string) error {
		return errors.New("cross-device rename")
	})
	if err != nil {
		t.Fatal(err)
	}
	assertWorkspaceRelease(t, finalPath)
	if _, err := os.Stat(workspace); err != nil {
		t.Fatalf("staged fallback should leave source workspace for caller cleanup: %v", err)
	}
	entries, err := os.ReadDir(releases)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".staging-") {
			t.Fatalf("staging directory leaked after commit: %s", entry.Name())
		}
	}
}

func TestStageWorkspaceReleaseCopiesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires Windows developer mode or elevated privileges")
	}
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	releases := filepath.Join(root, "releases")
	writeWorkspaceFixture(t, workspace)
	finalPath, err := stageWorkspaceRelease(workspace, releases, "release-1")
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(finalPath, "index-link.html"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "post/index.html" {
		t.Fatalf("symlink target = %q", target)
	}
}

func TestStageWorkspaceReleaseDoesNotOverwriteExistingRelease(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	releases := filepath.Join(root, "releases")
	finalPath := filepath.Join(releases, "release-1")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(finalPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(finalPath, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := stageWorkspaceRelease(workspace, releases, "release-1"); err == nil {
		t.Fatal("stageWorkspaceRelease() succeeded over an existing release")
	}
	content, err := os.ReadFile(filepath.Join(finalPath, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old" {
		t.Fatalf("existing release was modified: %q", content)
	}
}

func TestStageWorkspaceReleaseRejectsMissingWorkspace(t *testing.T) {
	root := t.TempDir()
	if _, err := stageWorkspaceRelease(filepath.Join(root, "missing"), filepath.Join(root, "releases"), "release-1"); err == nil {
		t.Fatal("stageWorkspaceRelease() succeeded with missing workspace")
	}
}

func TestPrepareBuildUsesConfiguredWorkDir(t *testing.T) {
	root := t.TempDir()
	siteDir := filepath.Join(root, "site")
	workDir := filepath.Join(root, "fast-work", "build")
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service := &Service{cfg: Config{SiteDir: siteDir, WorkDir: workDir, SourceDir: sourceDir}}
	if err := service.prepareBuild(os.Stderr); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		t.Fatalf("configured work dir was not prepared: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(filepath.Join(siteDir, ".build-work")); !os.IsNotExist(err) {
		t.Fatalf("legacy workspace should not be created when WorkDir is configured: %v", err)
	}
}
