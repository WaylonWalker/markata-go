package sourcegit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadSourceState(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "post.md"), "one")
	git(t, dir, "init")
	git(t, dir, "config", "user.email", "test@example.invalid")
	git(t, dir, "config", "user.name", "Content Index Test")
	git(t, dir, "add", "post.md")
	git(t, dir, "commit", "-m", "initial")

	state, err := Read(context.Background(), dir)
	if err != nil || state.Commit == "" || state.Dirty == nil || *state.Dirty {
		t.Fatalf("clean state = %#v, err = %v", state, err)
	}

	writeFile(t, filepath.Join(dir, "post.md"), "changed")
	dirtyState, err := Read(context.Background(), dir)
	if err != nil || dirtyState.Dirty == nil || !*dirtyState.Dirty || state.Equal(dirtyState) {
		t.Fatalf("tracked dirty state = %#v, err = %v", state, err)
	}
	writeFile(t, filepath.Join(dir, "post.md"), "changed again")
	changedDirtyState, err := Read(context.Background(), dir)
	if err != nil || dirtyState.Equal(changedDirtyState) {
		t.Fatalf("changed dirty state was not detected: %#v -> %#v, err = %v", dirtyState, changedDirtyState, err)
	}
	state = dirtyState

	git(t, dir, "add", "post.md")
	state, err = Read(context.Background(), dir)
	if err != nil || state.Dirty == nil || !*state.Dirty {
		t.Fatalf("staged dirty state = %#v, err = %v", state, err)
	}

	git(t, dir, "reset", "--hard", "HEAD")
	if err := os.Remove(filepath.Join(dir, "post.md")); err != nil {
		t.Fatal(err)
	}
	state, err = Read(context.Background(), dir)
	if err != nil || state.Dirty == nil || !*state.Dirty {
		t.Fatalf("deleted dirty state = %#v, err = %v", state, err)
	}
	git(t, dir, "restore", "post.md")
	writeFile(t, filepath.Join(dir, "new.md"), "untracked source")
	state, err = Read(context.Background(), dir)
	if err != nil || state.Dirty == nil || !*state.Dirty {
		t.Fatalf("untracked dirty state = %#v, err = %v", state, err)
	}
}

func TestReadSourceStateDetectsTrackedModeChanges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git executable mode bits are not portable on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "post.md")
	writeFile(t, path, "one")
	git(t, dir, "init")
	git(t, dir, "config", "user.email", "test@example.invalid")
	git(t, dir, "config", "user.name", "Content Index Test")
	git(t, dir, "add", "post.md")
	git(t, dir, "commit", "-m", "initial")
	clean, err := Read(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	modeChanged, err := Read(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if clean.Equal(modeChanged) {
		t.Fatal("tracked mode change was not detected")
	}
}

func TestReadSourceStateWithoutGit(t *testing.T) {
	if _, err := Read(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected unavailable Git state error")
	}
}

func TestReadSourceStateDetectsIgnoredMarkdownChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "ignored/\n")
	git(t, dir, "init")
	git(t, dir, "config", "user.email", "test@example.invalid")
	git(t, dir, "config", "user.name", "Content Index Test")
	git(t, dir, "add", ".gitignore")
	git(t, dir, "commit", "-m", "initial")
	if err := os.Mkdir(filepath.Join(dir, "ignored"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "ignored", "post.md"), "one")
	first, err := Read(context.Background(), dir)
	if err != nil || first.Dirty == nil || !*first.Dirty {
		t.Fatalf("ignored source was not marked dirty: %#v, %v", first, err)
	}
	writeFile(t, filepath.Join(dir, "ignored", "post.md"), "two")
	second, err := Read(context.Background(), dir)
	if err != nil || first.Equal(second) {
		t.Fatalf("ignored source change was not detected: %#v -> %#v, %v", first, second, err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	if output, err := Command(context.Background(), dir, args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
