package sourcegit

import (
	"context"
	"os"
	"path/filepath"
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
	state, err = Read(context.Background(), dir)
	if err != nil || state.Dirty == nil || !*state.Dirty {
		t.Fatalf("tracked dirty state = %#v, err = %v", state, err)
	}

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

func TestReadSourceStateWithoutGit(t *testing.T) {
	if _, err := Read(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected unavailable Git state error")
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
