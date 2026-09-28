package builderadmin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStageWorkspaceReleaseCopiesAndCommitsAtomically(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	releases := filepath.Join(root, "site", "releases")
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

	finalPath, err := stageWorkspaceRelease(workspace, releases, "release-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(releases, "release-1"); finalPath != want {
		t.Fatalf("final path = %q, want %q", finalPath, want)
	}
	content, err := os.ReadFile(filepath.Join(finalPath, "post", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "<h1>hello</h1>" {
		t.Fatalf("copied content = %q", content)
	}
	if _, err := os.Stat(workspace); err != nil {
		t.Fatalf("source workspace should remain for caller cleanup: %v", err)
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
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(filepath.Join(finalPath, "index-link.html"))
		if err != nil {
			t.Fatal(err)
		}
		if target != "post/index.html" {
			t.Fatalf("symlink target = %q", target)
		}
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
