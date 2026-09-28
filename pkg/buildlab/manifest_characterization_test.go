package buildlab

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// TestBuildManifestCharacterization records the complete result for a mixed
// fixture. Keeping the expected records explicit makes changes to walking,
// filtering, hashing, or classification observable in one comparison.
func TestBuildManifestCharacterization(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"content/post.md":            []byte("# Post\n"),
		"assets/site.css":            []byte("body { color: red }\n"),
		"output/index.html":          []byte("generated\n"),
		"nested/output/source":       []byte("keep\n"),
		"nested/cache/source":        []byte("keep too\n"),
		"nested/.markata-fonts.json": []byte("metadata\n"),
	}
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink("../post.md", filepath.Join(root, "content", "latest.md")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("nested", filepath.Join(root, "nested-link")); err != nil {
			t.Fatal(err)
		}
	}

	classes := map[string]OutputClass{
		"assets/site.css":      ClassSemantic,
		"nested/output/source": ClassVolatile,
	}
	got, err := BuildManifest(root, classes, "output", "cache")
	if err != nil {
		t.Fatal(err)
	}

	want := Manifest{Records: []FileRecord{
		manifestRecord(t, root, "assets/site.css", files["assets/site.css"], ClassSemantic),
		manifestRecord(t, root, "nested/cache/source", files["nested/cache/source"], ClassDeterministic),
		manifestRecord(t, root, "nested/output/source", files["nested/output/source"], ClassVolatile),
	}}
	if runtime.GOOS != "windows" {
		want.Records = []FileRecord{
			want.Records[0],
			manifestSymlinkRecord(t, root, "content/latest.md", "../post.md"),
			manifestRecord(t, root, "content/post.md", files["content/post.md"], ClassDeterministic),
			manifestSymlinkRecord(t, root, "nested-link", "nested"),
			want.Records[1], want.Records[2],
		}
	} else {
		want.Records = []FileRecord{
			want.Records[0],
			manifestRecord(t, root, "content/post.md", files["content/post.md"], ClassDeterministic),
			want.Records[1], want.Records[2],
		}
	}
	// BuildManifest sorts records by normalized path.
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest = %+v, want %+v", got, want)
	}
}

func TestBuildManifestCharacterization_MetadataFiles(t *testing.T) {
	root := t.TempDir()
	metadata := []string{".markata-css_minify-cache", ".markata-fontpack-cache", ".markata-fonts.json", ".markata-js_minify-cache"}
	for _, name := range metadata {
		if err := os.WriteFile(filepath.Join(root, name), []byte("ignored"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := BuildManifest(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{Records: []FileRecord{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest = %+v, want %+v", got, want)
	}
}

func TestBuildManifestCharacterization_RootIsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := BuildManifest(path, nil)
	wantError := fmt.Sprintf("path %q escapes root", path)
	if err == nil || err.Error() != wantError {
		t.Fatalf("error = %v, want %q", err, wantError)
	}
}

func manifestRecord(t *testing.T, root, path string, contents []byte, class OutputClass) FileRecord {
	t.Helper()
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(contents)
	return FileRecord{
		Path: path, SHA256: hex.EncodeToString(hash[:]), Size: int64(len(contents)),
		Mode: uint32(info.Mode()), Type: TypeRegular, Class: class,
	}
}

func manifestSymlinkRecord(t *testing.T, root, path, target string) FileRecord {
	t.Helper()
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(target))
	return FileRecord{
		Path: path, SHA256: hex.EncodeToString(hash[:]), Size: int64(len(target)),
		Mode: uint32(info.Mode()), Type: TypeSymlink, Class: ClassDeterministic,
	}
}

func TestBuildManifestCharacterization_MissingRoot(t *testing.T) {
	got, err := BuildManifest(filepath.Join(t.TempDir(), "missing"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Records == nil || len(got.Records) != 0 {
		t.Fatalf("missing root manifest = %+v, want non-nil empty records", got)
	}
}
