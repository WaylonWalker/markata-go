package servefix

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testSource(t *testing.T, source string) (root, path string) {
	t.Helper()
	root = t.TempDir()
	path = filepath.Join(root, "post.md")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func TestPlanAndApplySelectedCategories(t *testing.T) {
	root, path := testSource(t, "---\ndate: 09/26/2026\n---\n# Heading\n[link](//example.com)\n![](photo.png)\n")
	plan, err := PlanFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Edits) < 2 {
		t.Fatalf("edits = %+v", plan.Edits)
	}
	for _, edit := range plan.Edits {
		if edit.Code == "missing-alt-text" {
			t.Fatal("placeholder alt text should not be offered as an automatic fix")
		}
	}
	selected := Selection{Categories: []string{"h1-in-content", "protocol-less-url"}}
	preview, edits, err := Preview(plan, []byte("---\ndate: 09/26/2026\n---\n# Heading\n[link](//example.com)\n![](photo.png)\n"), selected)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 2 || !strings.Contains(string(preview), "## Heading") || !strings.Contains(string(preview), "https://example.com") {
		t.Fatalf("preview %q, edits %+v", preview, edits)
	}
	if !strings.Contains(string(preview), "![](photo.png)") {
		t.Fatal("unselected content changed")
	}
	if _, err := Apply(root, plan, selected); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, preview) {
		t.Fatalf("applied %q != preview %q", actual, preview)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode().Perm(), err)
	}
}

func TestApplyRejectsStaleFile(t *testing.T) {
	root, path := testSource(t, "# Heading\n")
	plan, err := PlanFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Updated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, plan, Selection{All: true}); !errors.Is(err, ErrStale) {
		t.Fatalf("err = %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "# Updated\n" {
		t.Fatalf("stale source overwritten: %q", actual)
	}
}

func TestReplacePathReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "post.md")
	source := filepath.Join(dir, ".markata-fix-temp")
	if err := os.WriteFile(destination, []byte("old content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("new content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replacePath(source, destination); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new content\n" {
		t.Fatalf("destination content = %q", content)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("replacement source remains or stat failed unexpectedly: %v", err)
	}
}

func TestApplyOneByOneAndUnknownID(t *testing.T) {
	root, path := testSource(t, "# Heading\n[link](//example.com)\n")
	plan, err := PlanFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Edits) != 2 {
		t.Fatalf("edits = %+v", plan.Edits)
	}
	if _, err := Apply(root, plan, Selection{IDs: []string{"missing"}}); err == nil {
		t.Fatal("unknown fix ID accepted")
	}
	if _, err := Apply(root, plan, Selection{IDs: []string{plan.Edits[0].ID}}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "//example.com") && !strings.Contains(string(content), "# Heading") {
		t.Fatal("both fixes were applied despite selecting only one")
	}
}

func TestPlanRejectsOutsideRootAndSymlink(t *testing.T) {
	root, _ := testSource(t, "# Heading\n")
	outside, outsidePath := testSource(t, "# Other\n")
	_ = outside
	if _, err := PlanFile(root, outsidePath); err == nil {
		t.Fatal("outside path accepted")
	}
	link := filepath.Join(root, "link.md")
	if err := os.Symlink(outsidePath, link); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanFile(root, link); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestPlanSkipsAmbiguousDateAndFixesDelimiter(t *testing.T) {
	root, path := testSource(t, "----\ndate: 01/02/2026\n---\n")
	plan, err := PlanFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Edits) != 1 || plan.Edits[0].Code != "frontmatter.suspicious_delimiter" {
		t.Fatalf("edits = %+v", plan.Edits)
	}
	if _, err := Apply(root, plan, Selection{All: true}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "---\ndate: 01/02/2026\n---\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestPreviewMultipleEditsOnSameLine(t *testing.T) {
	source := "[a](//a.example.com) [b](//b.example.com)\n"
	root, path := testSource(t, source)
	plan, err := PlanFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Edits) != 2 {
		t.Fatalf("edits = %+v", plan.Edits)
	}
	content, edits, err := Preview(plan, []byte(source), Selection{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 2 || string(content) != "[a](https://a.example.com) [b](https://b.example.com)\n" {
		t.Fatalf("preview = %q, edits = %+v", content, edits)
	}
}
