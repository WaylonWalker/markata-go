package builderadmin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func writeDeltaFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIncrementalPublication_IsolationChangesAndDeletion(t *testing.T) {
	root := t.TempDir()
	work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
	writeDeltaFile(t, work, "same.html", "unchanged")
	writeDeltaFile(t, work, "changed.html", "before")
	writeDeltaFile(t, work, "removed.html", "deleted")
	first, stats, err := stageIncrementalWorkspace(work, releases, "first", "")
	if err != nil {
		t.Fatal(err)
	}
	if stats.CopiedFiles != 3 || stats.LinkedFiles != 0 {
		t.Fatalf("first stats: %+v", stats)
	}
	writeDeltaFile(t, work, "changed.html", "after!") // Same size must still compare bytes.
	if err := os.Remove(filepath.Join(work, "removed.html")); err != nil {
		t.Fatal(err)
	}
	writeDeltaFile(t, work, "new/index.html", "added")
	second, stats, err := stageIncrementalWorkspace(work, releases, "second", "first")
	if err != nil {
		t.Fatal(err)
	}
	if stats.LinkedFiles != 1 || stats.CopiedFiles != 2 || stats.CopiedBytes != 11 || stats.ComparedBytes == 0 {
		t.Fatalf("warm stats: %+v", stats)
	}
	oldInfo, _ := os.Stat(filepath.Join(first, "same.html"))
	newInfo, _ := os.Stat(filepath.Join(second, "same.html"))
	workInfo, _ := os.Stat(filepath.Join(work, "same.html"))
	if !os.SameFile(oldInfo, newInfo) || os.SameFile(workInfo, newInfo) {
		t.Fatal("release linking violated workspace isolation")
	}
	assertWorkspaceContent(t, filepath.Join(first, "changed.html"), "before")
	assertWorkspaceContent(t, filepath.Join(first, "removed.html"), "deleted")
	assertWorkspaceContent(t, filepath.Join(second, "changed.html"), "after!")
	if _, err := os.Stat(filepath.Join(second, "removed.html")); !os.IsNotExist(err) {
		t.Fatal("deleted file survived publication")
	}
	writeDeltaFile(t, work, "same.html", "later failed build")
	writeDeltaFile(t, work, "changed.html", "another failed write")
	assertWorkspaceContent(t, filepath.Join(first, "same.html"), "unchanged")
	assertWorkspaceContent(t, filepath.Join(second, "same.html"), "unchanged")
	assertWorkspaceContent(t, filepath.Join(second, "changed.html"), "after!")
}

func TestIncrementalPublication_LinkFailureCopies(t *testing.T) {
	root := t.TempDir()
	work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
	writeDeltaFile(t, work, "index.html", "same")
	if _, _, err := stageIncrementalWorkspace(work, releases, "first", ""); err != nil {
		t.Fatal(err)
	}
	next, stats, err := stageIncrementalWorkspaceWithLink(work, releases, "next", "first", func(*os.Root, string, string) error { return errors.New("unsupported links") })
	if err != nil {
		t.Fatal(err)
	}
	if stats.CopiedFiles != 1 || stats.LinkedFiles != 0 {
		t.Fatalf("fallback stats: %+v", stats)
	}
	assertWorkspaceContent(t, filepath.Join(next, "index.html"), "same")
	oldInfo, _ := os.Stat(filepath.Join(releases, "first", "index.html"))
	nextInfo, _ := os.Stat(filepath.Join(next, "index.html"))
	if os.SameFile(oldInfo, nextInfo) {
		t.Fatal("fallback unexpectedly shared an inode")
	}
}

func TestIncrementalPublication_PermissionsAndExistingDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	root := t.TempDir()
	work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
	writeDeltaFile(t, work, "index.html", "same")
	if err := os.Chmod(work, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := stageIncrementalWorkspace(work, releases, "first", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(work, "index.html"), 0o660); err != nil {
		t.Fatal(err)
	}
	next, stats, err := stageIncrementalWorkspace(work, releases, "next", "first")
	if err != nil {
		t.Fatal(err)
	}
	if stats.CopiedFiles != 1 || stats.LinkedFiles != 0 {
		t.Fatalf("mode change ignored: %+v", stats)
	}
	info, _ := os.Stat(next)
	child, _ := os.Stat(filepath.Join(next, "index.html"))
	if info.Mode().Perm() != 0o755 || child.Mode().Perm() != 0o660 {
		t.Fatal("published permissions changed")
	}
	if _, _, err := stageIncrementalWorkspace(work, releases, "next", "first"); err == nil {
		t.Fatal("overwrote existing release")
	}
}

func TestIncrementalPublication_ConfinesBaselineAndCopiesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges")
	}
	root := t.TempDir()
	work, releases, outside := filepath.Join(root, "work"), filepath.Join(root, "releases"), filepath.Join(root, "outside")
	writeDeltaFile(t, work, "nested/index.html", "same")
	writeDeltaFile(t, outside, "index.html", "same")
	if err := os.MkdirAll(filepath.Join(releases, "first"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(releases, "first", "nested")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nested/index.html", filepath.Join(work, "alias.html")); err != nil {
		t.Fatal(err)
	}
	next, stats, err := stageIncrementalWorkspace(work, releases, "next", "first")
	if err != nil {
		t.Fatal(err)
	}
	if stats.LinkedFiles != 0 || stats.CopiedFiles != 1 {
		t.Fatalf("baseline escaped root: %+v", stats)
	}
	target, err := os.Readlink(filepath.Join(next, "alias.html"))
	if err != nil || target != "nested/index.html" {
		t.Fatalf("symlink changed: %q %v", target, err)
	}
}

func TestIncrementalPublication_FailedStageLeavesReleasesIntact(t *testing.T) {
	root := t.TempDir()
	work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
	writeDeltaFile(t, work, "index.html", "same")
	if _, _, err := stageIncrementalWorkspace(work, releases, "first", ""); err != nil {
		t.Fatal(err)
	}
	_, _, err := stageIncrementalWorkspaceWithLink(work, releases, "failed", "first", func(*os.Root, string, string) error {
		if err := os.Remove(filepath.Join(work, "index.html")); err != nil {
			return err
		}
		return errors.New("link failed")
	})
	if err == nil {
		t.Fatal("missing workspace file did not fail publication")
	}
	assertWorkspaceContent(t, filepath.Join(releases, "first", "index.html"), "same")
	entries, err := os.ReadDir(releases)
	if err != nil || len(entries) != 1 || entries[0].Name() != "first" {
		t.Fatalf("failed stage exposed output: %v %v", entries, err)
	}
}

func BenchmarkWorkspacePublication(b *testing.B) {
	for _, delta := range []bool{false, true} {
		name := "full-copy"
		if delta {
			name = "unchanged-delta"
		}
		b.Run(name, func(b *testing.B) {
			root := b.TempDir()
			work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
			if err := os.MkdirAll(work, 0o755); err != nil {
				b.Fatal(err)
			}
			data := bytes.Repeat([]byte("benchmark"), 32<<10)
			for i := range 128 {
				if err := os.WriteFile(filepath.Join(work, fmt.Sprintf("file-%d", i)), data, 0o644); err != nil {
					b.Fatal(err)
				}
			}
			if _, _, err := stageIncrementalWorkspace(work, releases, "baseline", ""); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				id := fmt.Sprintf("file-%d", i)
				var path string
				var err error
				if delta {
					var stats workspacePublicationStats
					path, stats, err = stageIncrementalWorkspace(work, releases, id, "baseline")
					b.ReportMetric(float64(stats.CopiedBytes), "copied-bytes/op")
					b.ReportMetric(float64(stats.LinkedFiles), "linked-files/op")
				} else {
					path, err = stageWorkspaceRelease(work, releases, id)
				}
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				if err := os.RemoveAll(path); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
			}
		})
	}
}

func TestIncrementalPublication_RejectsOverlappingTrees(t *testing.T) {
	root := t.TempDir()
	for _, pair := range [][2]string{{root, root}, {root, filepath.Join(root, "releases")}, {filepath.Join(root, "work"), root}} {
		if _, _, err := stageIncrementalWorkspace(pair[0], pair[1], "next", ""); err == nil {
			t.Fatal("accepted overlapping workspace and releases")
		}
	}
}

func TestIncrementalPublication_BoundedConcurrencyAndAtomicVisibility(t *testing.T) {
	root := t.TempDir()
	work, releases := filepath.Join(root, "work"), filepath.Join(root, "releases")
	for i := range workspacePublicationWorkers * 3 {
		writeDeltaFile(t, work, fmt.Sprintf("dir/file-%d", i), "same")
	}
	if _, _, err := stageIncrementalWorkspace(work, releases, "first", ""); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{}, workspacePublicationWorkers*3)
	resume := make(chan struct{})
	var active, maximum atomic.Int32
	done := make(chan error, 1)
	go func() {
		_, _, err := stageIncrementalWorkspaceWithLink(work, releases, "next", "first", func(root *os.Root, old, next string) error {
			n := active.Add(1)
			defer active.Add(-1)
			for previous := maximum.Load(); n > previous; previous = maximum.Load() {
				if maximum.CompareAndSwap(previous, n) {
					break
				}
			}
			entered <- struct{}{}
			<-resume
			return root.Link(old, next)
		})
		done <- err
	}()
	t.Cleanup(func() {
		close(resume)
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	for range workspacePublicationWorkers {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("publication did not overlap file operations")
		}
	}
	if _, err := os.Stat(filepath.Join(releases, "next")); !os.IsNotExist(err) {
		t.Fatal("incomplete release became visible")
	}
	if maximum.Load() != workspacePublicationWorkers {
		t.Fatalf("concurrency = %d", maximum.Load())
	}
}
