package builderadmin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeReleaseForActivationTest(t *testing.T, path, homepage, contentIndex string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if homepage != "" {
		if err := os.WriteFile(filepath.Join(path, "index.html"), []byte(homepage), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if contentIndex != "" {
		if err := os.WriteFile(filepath.Join(path, defaultReleaseContentIndex), []byte(contentIndex), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateReleaseOutputRequiresHomepage(t *testing.T) {
	release := t.TempDir()
	if err := validateReleaseOutput(release); err == nil || !strings.Contains(err.Error(), "homepage") {
		t.Fatalf("validateReleaseOutput() error = %v, want homepage failure", err)
	}
	if err := os.WriteFile(filepath.Join(release, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseOutput(release); err != nil {
		t.Fatalf("validateReleaseOutput() rejected valid release: %v", err)
	}
}

func TestValidateReleaseOutputRejectsMalformedContentIndex(t *testing.T) {
	release := t.TempDir()
	writeReleaseForActivationTest(t, release, "<html>ok</html>", "not json")
	if err := validateReleaseOutput(release); err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("validateReleaseOutput() error = %v, want invalid JSON failure", err)
	}
}

func TestReplaceCurrentReleaseRejectsInvalidTargetAndKeepsCurrent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require Windows developer mode")
	}
	root := t.TempDir()
	releases := filepath.Join(root, "releases")
	oldRelease := filepath.Join(releases, "old")
	badRelease := filepath.Join(releases, "bad")
	writeReleaseForActivationTest(t, oldRelease, "old", "")
	writeReleaseForActivationTest(t, badRelease, "", "")

	current := filepath.Join(root, "current")
	currentNext := filepath.Join(root, "current.next")
	oldTarget := filepath.Join("releases", "old")
	if err := os.Symlink(oldTarget, current); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "bad"), currentNext); err != nil {
		t.Fatal(err)
	}
	if err := replaceCurrentRelease(currentNext, current); err == nil {
		t.Fatal("replaceCurrentRelease() activated release without homepage")
	}
	got, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	if got != oldTarget {
		t.Fatalf("current target = %q, want unchanged %q", got, oldTarget)
	}
}

func TestReplaceCurrentReleaseAcceptsValidTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require Windows developer mode")
	}
	root := t.TempDir()
	releases := filepath.Join(root, "releases")
	oldRelease := filepath.Join(releases, "old")
	newRelease := filepath.Join(releases, "new")
	writeReleaseForActivationTest(t, oldRelease, "old", "")
	writeReleaseForActivationTest(t, newRelease, "new", `{"documents":[]}`)

	current := filepath.Join(root, "current")
	currentNext := filepath.Join(root, "current.next")
	if err := os.Symlink(filepath.Join("releases", "old"), current); err != nil {
		t.Fatal(err)
	}
	newTarget := filepath.Join("releases", "new")
	if err := os.Symlink(newTarget, currentNext); err != nil {
		t.Fatal(err)
	}
	if err := replaceCurrentRelease(currentNext, current); err != nil {
		t.Fatalf("replaceCurrentRelease() error = %v", err)
	}
	got, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	if got != newTarget {
		t.Fatalf("current target = %q, want %q", got, newTarget)
	}
}
