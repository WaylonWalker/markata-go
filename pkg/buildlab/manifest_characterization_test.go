package buildlab

import (
	"crypto/sha256"
	"encoding/hex"
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
		manifestRecord("assets/site.css", files["assets/site.css"], ClassSemantic),
		manifestRecord("nested/cache/source", files["nested/cache/source"], ClassDeterministic),
		manifestRecord("nested/output/source", files["nested/output/source"], ClassVolatile),
	}}
	if runtime.GOOS != "windows" {
		target := "../post.md"
		hash := sha256.Sum256([]byte(target))
		want.Records = []FileRecord{
			want.Records[0],
			{
				Path: "content/latest.md", SHA256: hex.EncodeToString(hash[:]),
				Size: int64(len(target)), Mode: uint32(os.ModeSymlink | 0o777), Type: TypeSymlink,
				Class: ClassDeterministic,
			},
			manifestRecord("content/post.md", files["content/post.md"], ClassDeterministic),
			want.Records[1], want.Records[2],
		}
	} else {
		want.Records = []FileRecord{
			want.Records[0],
			manifestRecord("content/post.md", files["content/post.md"], ClassDeterministic),
			want.Records[1], want.Records[2],
		}
	}
	// BuildManifest sorts records by normalized path.
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest = %+v, want %+v", got, want)
	}
}

func manifestRecord(path string, contents []byte, class OutputClass) FileRecord {
	hash := sha256.Sum256(contents)
	return FileRecord{
		Path: path, SHA256: hex.EncodeToString(hash[:]), Size: int64(len(contents)),
		Mode: 0o600, Type: TypeRegular, Class: class,
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
