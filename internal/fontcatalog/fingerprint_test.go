package fontcatalog

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestMetadataFingerprintTracksCatalogLockAndManifests(t *testing.T) {
	fixture := fstest.MapFS{
		"markata-fontpacks.yaml":  {Data: []byte("catalog")},
		"markata-fonts.lock.yaml": {Data: []byte("lock")},
		"body/manifest.yaml":      {Data: []byte("manifest")},
		"body/full.woff2":         {Data: []byte("verified asset")},
	}
	first, err := metadataFingerprint(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 {
		t.Fatalf("fingerprint length = %d, want 64", len(first))
	}
	repeated, err := metadataFingerprint(fixture)
	if err != nil || first != repeated {
		t.Fatalf("fingerprint is not stable: %q, %v", repeated, err)
	}
	for _, name := range []string{"markata-fontpacks.yaml", "markata-fonts.lock.yaml", "body/manifest.yaml"} {
		original := fixture[name].Data
		fixture[name].Data = []byte("changed")
		changed, err := metadataFingerprint(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if changed == first {
			t.Fatalf("changing %s did not invalidate fingerprint", name)
		}
		fixture[name].Data = original
	}
	fixture["additional/manifest.yaml"] = &fstest.MapFile{Data: []byte("another family")}
	added, err := metadataFingerprint(fixture)
	if err != nil || added == first {
		t.Fatalf("adding a manifest did not invalidate fingerprint: %q, %v", added, err)
	}
}

func TestMetadataFingerprintRequiresMetadataFiles(t *testing.T) {
	_, err := metadataFingerprint(fstest.MapFS{"markata-fontpacks.yaml": {Data: []byte("catalog")}})
	if err == nil {
		t.Fatal("missing lockfile accepted")
	}
}

func TestBuiltinFingerprintIsStable(t *testing.T) {
	first, err := Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fingerprint()
	if err != nil || first != second {
		t.Fatalf("builtin fingerprint changed: %q, %v", second, err)
	}
	if _, err := fs.Stat(FS(), "markata-fontpacks.yaml"); err != nil {
		t.Fatal(err)
	}
}
