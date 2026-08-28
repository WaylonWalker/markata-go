package buildlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type OutputClass string

const (
	ClassDeterministic          OutputClass = "deterministic"
	ClassSemantic               OutputClass = "semantic"
	ClassSecureNondeterministic OutputClass = "secure-nondeterministic"
	ClassVolatile               OutputClass = "volatile"
)

type FileType string

const (
	TypeRegular FileType = "regular"
	TypeSymlink FileType = "symlink"
)

type FileRecord struct {
	Path   string      `json:"path"`
	SHA256 string      `json:"sha256"`
	Size   int64       `json:"size"`
	Mode   uint32      `json:"mode"`
	Type   FileType    `json:"type"`
	Class  OutputClass `json:"class"`
}

type Manifest struct {
	Records []FileRecord `json:"records"`
}

func normalizeRelative(root, name string) (string, error) {
	rel, err := filepath.Rel(root, name)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("path %q escapes root", name)
	}
	return rel, nil
}

// BuildManifest walks root without following symlinks and hashes publishable
// regular files as a stream. Known transient Markata-Go metadata files are not
// part of the output contract. classes maps normalized paths to output classes.
func BuildManifest(root string, classes map[string]OutputClass) (Manifest, error) {
	records := make([]FileRecord, 0)
	if _, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return Manifest{Records: []FileRecord{}}, nil
		}
		return Manifest{}, err
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			rel, err := normalizeRelative(root, path)
			if err != nil {
				return err
			}
			if isBuildMetadataPath(rel) {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := normalizeRelative(root, path)
		if err != nil {
			return err
		}
		if isBuildMetadataPath(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		class := classes[rel]
		if class == "" {
			class = ClassDeterministic
		}
		r := FileRecord{Path: rel, Mode: uint32(info.Mode()), Class: class}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			target, e := os.Readlink(path)
			if e != nil {
				return e
			}
			h := sha256.Sum256([]byte(target))
			r.SHA256 = hex.EncodeToString(h[:])
			r.Size = int64(len(target))
			r.Type = TypeSymlink
		case info.Mode().IsRegular():
			f, e := os.Open(path)
			if e != nil {
				return e
			}
			h := sha256.New()
			n, e := io.Copy(h, f)
			closeErr := f.Close()
			if e != nil {
				return e
			}
			if closeErr != nil {
				return closeErr
			}
			r.SHA256 = hex.EncodeToString(h.Sum(nil))
			r.Size = n
			r.Type = TypeRegular
		default:
			return nil
		}
		records = append(records, r)
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return Manifest{Records: records}, nil
}

func isBuildMetadataPath(path string) bool {
	switch filepath.Base(filepath.FromSlash(path)) {
	case ".markata-css_minify-cache", ".markata-fontpack-cache", ".markata-fonts.json", ".markata-js_minify-cache":
		return true
	default:
		return false
	}
}

// CollectManifest is an alias for BuildManifest.
func CollectManifest(root string, classes map[string]OutputClass) (Manifest, error) {
	return BuildManifest(root, classes)
}

// CanonicalJSON returns stable JSON with records sorted by path.
func (m Manifest) CanonicalJSON() ([]byte, error) {
	c := Manifest{Records: append([]FileRecord(nil), m.Records...)}
	if c.Records == nil {
		c.Records = []FileRecord{}
	}
	sort.Slice(c.Records, func(i, j int) bool { return c.Records[i].Path < c.Records[j].Path })
	return json.Marshal(c)
}
func (m Manifest) Digest() (string, error) {
	b, err := m.CanonicalJSON()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Compare compares this manifest with got.
func (m Manifest) Compare(got Manifest, comparators map[string]func(FileRecord, FileRecord) bool) ManifestDiff {
	return CompareManifests(m, got, comparators)
}
func (m Manifest) Record(path string) (FileRecord, bool) {
	for _, r := range m.Records {
		if r.Path == path {
			return r, true
		}
	}
	return FileRecord{}, false
}

type ManifestDiff struct{ Missing, Extra, Changed []FileRecord }

func (d ManifestDiff) Equal() bool {
	return len(d.Missing) == 0 && len(d.Extra) == 0 && len(d.Changed) == 0
}

// Compare checks paths and types always. Deterministic output also checks size
// and bytes; semantic output can be exempted by comparator, while the two
// nondeterministic classes only check path and type.
func CompareManifests(want, got Manifest, comparators map[string]func(FileRecord, FileRecord) bool) ManifestDiff {
	w := map[string]FileRecord{}
	g := map[string]FileRecord{}
	for _, r := range want.Records {
		w[r.Path] = r
	}
	for _, r := range got.Records {
		g[r.Path] = r
	}
	d := ManifestDiff{}
	for p, r := range w {
		x, ok := g[p]
		if !ok {
			d.Missing = append(d.Missing, r)
			continue
		}
		if x.Type != r.Type {
			d.Changed = append(d.Changed, r)
			continue
		}
		if x.Class != r.Class {
			d.Changed = append(d.Changed, r)
			continue
		}
		if r.Class == ClassSecureNondeterministic || r.Class == ClassVolatile {
			continue
		}
		if r.Class == ClassSemantic {
			if f := comparators[p]; f != nil {
				if f(r, x) {
					continue
				}
			}
		}
		if x.SHA256 != r.SHA256 || x.Size != r.Size || x.Mode != r.Mode {
			d.Changed = append(d.Changed, r)
		}
	}
	for p, r := range g {
		if _, ok := w[p]; !ok {
			d.Extra = append(d.Extra, r)
		}
	}
	sort.Slice(d.Missing, func(i, j int) bool { return d.Missing[i].Path < d.Missing[j].Path })
	sort.Slice(d.Extra, func(i, j int) bool { return d.Extra[i].Path < d.Extra[j].Path })
	sort.Slice(d.Changed, func(i, j int) bool { return d.Changed[i].Path < d.Changed[j].Path })
	return d
}
