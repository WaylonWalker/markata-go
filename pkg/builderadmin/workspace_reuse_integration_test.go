package builderadmin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuilderAdminBuild_RetainedWorkspaceSuccessFailureAndDeletion(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("cross-filesystem fixture uses Linux tmpfs")
	}
	workRoot, err := os.MkdirTemp("/dev/shm", "markata-workspace-")
	if err != nil {
		t.Skipf("tmpfs unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workRoot) })
	source := t.TempDir()
	script := filepath.Join(source, "worker.sh")
	contents := `#!/bin/sh
set -eu
for output do :; done
sleep 1
mkdir -p "$output"
phase=$(cat phase)
printf '%s' "$phase" > "$output/index.html"
case "$phase" in
 first) printf old > "$output/deleted.html" ;;
 failed) exit 1 ;;
 second|recovered) rm -f "$output/deleted.html"; printf new > "$output/renamed.html" ;;
esac
`
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SourceDir: source, SiteDir: t.TempDir(), WorkDir: filepath.Join(workRoot, "work"), ReleasesKeep: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })
	svc.executable = script
	run := func(phase string) BuildRecord {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, "phase"), []byte(phase), 0o600); err != nil {
			t.Fatal(err)
		}
		svc.runBuild(context.Background(), queueRequest{QueuedOperation: QueuedOperation{ID: "build-" + phase, Kind: "build", EnqueuedAt: time.Now()}})
		svc.pruneScheduleMu.Lock()
		done := svc.pruneDone
		svc.pruneScheduleMu.Unlock()
		if done != nil {
			<-done
		}
		return svc.snapshotState().Builds[0]
	}
	first := run("first")
	if first.Status != "success" {
		t.Fatalf("first build: %+v", first)
	}
	assertWorkspaceContent(t, workspaceReuseMarkerPath(svc.cfg.WorkDir), first.ReleaseID+"\n")
	second := run("second")
	if second.Status != "success" {
		t.Fatalf("warm build: %+v", second)
	}
	log, err := os.ReadFile(filepath.Join(svc.logDir, second.LogPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "reusing build work") {
		t.Fatal("successful warm workspace was not reused")
	}
	if _, err := os.Stat(filepath.Join(second.ReleasePath, "deleted.html")); !os.IsNotExist(err) {
		t.Fatal("deleted output was published")
	}
	assertWorkspaceContent(t, filepath.Join(second.ReleasePath, "renamed.html"), "new")
	assertWorkspaceContent(t, filepath.Join(first.ReleasePath, "index.html"), "first")
	assertWorkspaceContent(t, filepath.Join(first.ReleasePath, "deleted.html"), "old")
	failed := run("failed")
	if failed.Status != "failed" || svc.currentReleaseID() != second.ReleaseID {
		t.Fatal("failed build changed current release")
	}
	if _, err := os.Stat(workspaceReuseMarkerPath(svc.cfg.WorkDir)); !os.IsNotExist(err) {
		t.Fatal("failed workspace retained reusable proof")
	}
	recovered := run("recovered")
	if recovered.Status != "success" {
		t.Fatalf("recovery build: %+v", recovered)
	}
	log, err = os.ReadFile(filepath.Join(svc.logDir, recovered.LogPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "seeding build work") {
		t.Fatal("recovery reused failed output")
	}
	assertWorkspaceContent(t, filepath.Join(second.ReleasePath, "index.html"), "second")
}
