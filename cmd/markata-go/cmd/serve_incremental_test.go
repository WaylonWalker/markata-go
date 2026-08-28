package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildlab"
	"github.com/fsnotify/fsnotify"
)

func TestIncrementalPathsRequireFullRebuild(t *testing.T) {
	if incrementalPathsRequireFullRebuild([]string{"pages/post.md"}, nil) {
		t.Fatal("markdown content change forced a full rebuild")
	}
	for _, path := range []string{"markata-go.toml", "templates/base.html", "static/site.css"} {
		if !incrementalPathsRequireFullRebuild([]string{path}, nil) {
			t.Fatalf("global input %q did not force a full rebuild", path)
		}
	}
	if !incrementalPathsRequireFullRebuild(nil, []string{"pages/removed.md"}) {
		t.Fatal("removed content did not force a full rebuild")
	}
}

func TestServeIncremental_RebuildsFromPersistentState(t *testing.T) {
	repoRoot := moduleRoot(t)
	sourceFixture := filepath.Join(repoRoot, "cmd", "markata-go", "cmd", "testdata", "dag-site")
	fixture := t.TempDir()
	if err := buildlab.CopyFixture(sourceFixture, fixture); err != nil {
		t.Fatal(err)
	}

	writeFixtureFile := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	writeFixtureFile(filepath.Join(fixture, "content", "slides.md"), `---
title: Slides
date: 2026-08-05
published: true
---

# Slides

The generated page must survive every serve rebuild.
`)
	writeFixtureFile(filepath.Join(fixture, "content", "stable.md"), `---
title: Stable page
date: 2026-08-06
published: true
---

# Stable page

This page should remain cached when another post changes.
`)
	const staticSentinel = "static fixture content must not replace generated output"
	writeFixtureFile(filepath.Join(fixture, "static", "slides", "index.html"), staticSentinel)

	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(fixture); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWorkingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	oldCfgFile := cfgFile
	oldMergeConfigFiles := append([]string(nil), mergeConfigFiles...)
	oldOutputDir := outputDir
	oldVerbose := verbose
	oldServeFast := serveFast
	oldServeIncremental := serveIncremental
	oldBuildFast := buildFast
	oldBuildDAG := buildDAG
	oldIsRebuilding := isRebuilding.Load()
	oldRebuildPending := rebuildPending.Load()
	oldBuildStatus := buildStatus.Load()
	serveChangedPathsMu.Lock()
	oldServeChangedPaths := serveChangedPaths
	oldServeForceFullRebuild := serveForceFullRebuild
	oldServeGlobDirty := serveGlobDirty
	serveChangedPaths = make(map[string]fsnotify.Op)
	serveForceFullRebuild = false
	serveGlobDirty = false
	serveChangedPathsMu.Unlock()
	serveCacheMu.Lock()
	oldServeCache := serveCache
	serveCache = nil
	serveCacheMu.Unlock()
	servePostsMu.Lock()
	oldServePosts := servePosts
	servePosts = nil
	servePostsMu.Unlock()
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		mergeConfigFiles = oldMergeConfigFiles
		outputDir = oldOutputDir
		verbose = oldVerbose
		serveFast = oldServeFast
		serveIncremental = oldServeIncremental
		buildFast = oldBuildFast
		buildDAG = oldBuildDAG
		isRebuilding.Store(oldIsRebuilding)
		rebuildPending.Store(oldRebuildPending)
		if oldBuildStatus != nil {
			buildStatus.Store(oldBuildStatus)
		} else {
			buildStatus.Store(BuildStatus{Status: buildStatusBuilding})
		}
		serveChangedPathsMu.Lock()
		serveChangedPaths = oldServeChangedPaths
		serveForceFullRebuild = oldServeForceFullRebuild
		serveGlobDirty = oldServeGlobDirty
		serveChangedPathsMu.Unlock()
		serveCacheMu.Lock()
		serveCache = oldServeCache
		serveCacheMu.Unlock()
		servePostsMu.Lock()
		servePosts = oldServePosts
		servePostsMu.Unlock()
	})

	t.Setenv("SOURCE_DATE_EPOCH", "1780000000")
	cfgFile = filepath.Join(fixture, "markata-go.toml")
	mergeConfigFiles = nil
	outputDir = ""
	verbose = false
	serveFast = false
	serveIncremental = true
	buildFast = false
	buildDAG = false

	m, err := createManager(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	configureRebuildManager(m)
	var wg sync.WaitGroup
	startInitialBuild(m, nil, &wg)
	wg.Wait()
	if status := getBuildStatus(); status.Status != buildStatusSuccess {
		t.Fatalf("initial serve build status = %+v", status)
	}

	slidesOutput := filepath.Join(fixture, "output", "slides", "index.html")
	stableOutput := filepath.Join(fixture, "output", "stable", "index.html")
	initialSlides, err := os.ReadFile(slidesOutput)
	if err != nil {
		t.Fatal(err)
	}
	initialStable, err := os.ReadFile(stableOutput)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(initialSlides, []byte(staticSentinel)) {
		t.Fatal("initial serve build used the conflicting static page")
	}
	initialCache := getServeCache()
	if initialCache == nil {
		t.Fatal("initial serve build did not retain its cache")
	}
	cachePath := filepath.Join(fixture, ".markata", "build-cache.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("initial serve cache was not written: %v", err)
	}
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}

	changedPath := filepath.Join(fixture, "content", "source.md")
	sourceOutput := filepath.Join(fixture, "output", "source", "index.html")
	changes := []struct {
		content string
		marker  string
	}{
		{
			content: "This post changed during the first serve rebuild.\n\nThis post links to [[target]] and embeds it below.\n\n![[target]]\n",
			marker:  "This post changed during the first serve rebuild.",
		},
		{
			content: "This post changed during the second serve rebuild.\n\nThis post links to [[target]] and embeds it below.\n\n![[target]]\n",
			marker:  "This post changed during the second serve rebuild.",
		},
	}
	for i := range changes {
		if err := os.WriteFile(changedPath, []byte("---\ntitle: Source post\ndate: 2026-08-02\npublished: true\n---\n\n# Source post\n\n"+changes[i].content), 0o600); err != nil {
			t.Fatal(err)
		}
		recordServeChangedPath(fsnotify.Event{Name: changedPath, Op: fsnotify.Write})
		doRebuild(context.Background(), nil)
		if status := getBuildStatus(); status.Status != buildStatusSuccess {
			t.Fatalf("serve rebuild %d status = %+v", i+1, status)
		}
		source, err := os.ReadFile(sourceOutput)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(source, []byte(changes[i].marker)) {
			t.Fatalf("serve rebuild %d did not render the changed source", i+1)
		}

		slides, err := os.ReadFile(slidesOutput)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(slides, initialSlides) {
			t.Fatalf("generated slides output changed after serve rebuild %d", i+1)
		}
		if bytes.Contains(slides, []byte(staticSentinel)) {
			t.Fatalf("serve rebuild %d used the conflicting static page", i+1)
		}
		stable, err := os.ReadFile(stableOutput)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(stable, initialStable) {
			t.Fatalf("unchanged stable output changed after serve rebuild %d", i+1)
		}
		cache := getServeCache()
		if cache == nil {
			t.Fatal("serve cache was not retained after rebuild")
		}
		if cache != initialCache {
			t.Fatalf("serve rebuild %d replaced the persistent in-memory cache", i+1)
		}
		if skipped, _ := cache.Stats(); skipped == 0 {
			t.Fatalf("serve rebuild %d did not reuse any cached posts", i+1)
		}
	}
}
