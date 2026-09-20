package builderadmin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateBuildOutput_RequiresHomepage(t *testing.T) {
	candidate := t.TempDir()
	if err := validateBuildOutput(candidate, ""); err == nil {
		t.Fatal("validateBuildOutput unexpectedly accepted a candidate without index.html")
	}
	if err := os.WriteFile(filepath.Join(candidate, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateBuildOutput(candidate, ""); err != nil {
		t.Fatalf("validateBuildOutput rejected a valid homepage: %v", err)
	}
}

func TestValidateBuildOutput_RequiresConfiguredContentIndex(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidate")
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidate, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(`[markata-go.content_index]
enabled = true
output = "content-index.json"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateBuildOutput(candidate, configPath); err == nil {
		t.Fatal("validateBuildOutput unexpectedly accepted a missing configured content index")
	}
	if err := os.WriteFile(filepath.Join(candidate, "content-index.json"), []byte(`{"documents":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateBuildOutput(candidate, configPath); err != nil {
		t.Fatalf("validateBuildOutput rejected valid configured artifacts: %v", err)
	}
}

func TestPromoteBuild_RejectsInvalidCandidateAndKeepsCurrent(t *testing.T) {
	siteDir := t.TempDir()
	historyDir := filepath.Join(siteDir, ".builder-admin")
	oldRelease := filepath.Join(siteDir, "releases", "old")
	candidate := filepath.Join(siteDir, ".build-work")
	if err := os.MkdirAll(oldRelease, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldRelease, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "old"), filepath.Join(siteDir, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SiteDir: siteDir, HistoryDir: historyDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })

	if _, _, err := svc.promoteBuild(candidate); err == nil {
		t.Fatal("promoteBuild unexpectedly promoted a candidate without homepage")
	} else if !strings.Contains(err.Error(), "homepage") {
		t.Fatalf("promoteBuild error = %v, want homepage reason", err)
	}
	current, err := os.Readlink(filepath.Join(siteDir, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if current != filepath.Join("releases", "old") {
		t.Fatalf("current target = %q, want old release", current)
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("invalid candidate was not retained: %v", err)
	}
}

func TestPromoteBuild_ValidCandidateBecomesCurrent(t *testing.T) {
	siteDir := t.TempDir()
	historyDir := filepath.Join(siteDir, ".builder-admin")
	oldRelease := filepath.Join(siteDir, "releases", "old")
	candidate := filepath.Join(siteDir, ".build-work")
	if err := os.MkdirAll(oldRelease, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldRelease, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "old"), filepath.Join(siteDir, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidate, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SiteDir: siteDir, HistoryDir: historyDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })

	releaseID, releasePath, err := svc.promoteBuild(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if releaseID == "" || releasePath == "" {
		t.Fatalf("promoteBuild() returned empty release: %q %q", releaseID, releasePath)
	}
	current, err := os.Readlink(filepath.Join(siteDir, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if current != filepath.Join("releases", releaseID) {
		t.Fatalf("current target = %q, want %q", current, filepath.Join("releases", releaseID))
	}
	if _, err := os.Stat(filepath.Join(releasePath, "index.html")); err != nil {
		t.Fatalf("promoted release missing homepage: %v", err)
	}
}

func TestPrepareBuild_FailedCopyDoesNotChangeCurrent(t *testing.T) {
	siteDir := t.TempDir()
	historyDir := filepath.Join(siteDir, ".builder-admin")
	currentFile := filepath.Join(siteDir, "current-file")
	if err := os.WriteFile(currentFile, []byte("not a release directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(currentFile), filepath.Join(siteDir, "current")); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{SiteDir: siteDir, HistoryDir: historyDir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })

	logPath, logFile, err := svc.createLogFile("copy-failure")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	if err := svc.prepareBuild(logFile); err == nil {
		t.Fatal("prepareBuild unexpectedly copied a regular file as a release")
	}
	_ = logPath
	current, err := os.Readlink(filepath.Join(siteDir, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if current != filepath.Base(currentFile) {
		t.Fatalf("current target = %q changed after copy failure", current)
	}
}

func TestCheckFreeSpace_RejectsENOSPCLikeBudget(t *testing.T) {
	if err := checkFreeSpace(10, 11); err == nil {
		t.Fatal("checkFreeSpace accepted an insufficient budget")
	} else if !strings.Contains(err.Error(), "insufficient free space") {
		t.Fatalf("checkFreeSpace error = %v", err)
	}
}

func TestRunBuild_RecordsCandidateValidationFailure(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "successful-no-output.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := New(Config{
		SourceDir:    root,
		SiteDir:      filepath.Join(root, "site"),
		HistoryDir:   filepath.Join(root, "history"),
		BuildTimeout: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.leaderLock.Close() })
	svc.executable = script
	svc.runBuild(context.Background(), queueRequest{QueuedOperation: QueuedOperation{
		ID:          "validation-failure",
		Kind:        "build",
		Label:       "Build",
		TriggerType: "test",
		EnqueuedAt:  time.Now().UTC(),
	}})
	state := svc.snapshotState()
	if len(state.Builds) != 1 {
		t.Fatalf("build records = %d, want one", len(state.Builds))
	}
	if state.Builds[0].Status != "failed" || !strings.Contains(state.Builds[0].Error, "homepage") {
		t.Fatalf("build record = %+v, want homepage validation failure", state.Builds[0])
	}
	logData, err := os.ReadFile(filepath.Join(svc.logDir, state.Builds[0].LogPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "promotion rejected") {
		t.Fatalf("build log = %q, want promotion rejection reason", logData)
	}
	if _, err := os.Stat(filepath.Join(svc.cfg.SiteDir, ".build-work")); err != nil {
		t.Fatalf("invalid candidate was not retained: %v", err)
	}
}
