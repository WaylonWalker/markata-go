package fontpacks

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func verificationFixture(t *testing.T) (*CatalogSource, *Lockfile, string) {
	t.Helper()
	root := t.TempDir()
	family := filepath.Join(root, "demo")
	if err := os.MkdirAll(family, 0o755); err != nil {
		t.Fatal(err)
	}
	license := []byte("license")
	font := []byte("wOF2 fixture")
	if err := os.WriteFile(filepath.Join(family, "OFL.txt"), license, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "demo.woff2"), font, 0o644); err != nil {
		t.Fatal(err)
	}
	licenseHash, _, _ := AssetSHA256(filepath.Join(family, "OFL.txt"))
	fontHash, _, _ := AssetSHA256(filepath.Join(family, "demo.woff2"))
	manifest := Manifest{ID: "demo", Family: "Demo", Source: ManifestSource{Provider: "test", Repository: "https://example.test/fonts", Revision: "rev1", Directory: "ofl/demo", Files: map[string]string{"normal": strings.Repeat("a", 64)}}, License: License{ID: "OFL-1.1", File: "OFL.txt", SHA256: licenseHash}, Faces: map[string]Face{"normal": {SourceFile: "Demo-Regular.ttf"}}, Tiers: map[string]Tier{"prose-core": {File: "demo.woff2", Profile: "prose-core", SHA256: fontHash, Bytes: int64(len(font))}}}
	data, _ := yaml.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(family, "manifest.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Catalog{SystemStacks: map[string]SystemStack{"sans": {CSS: "system-ui"}}, FontSources: map[string]FontSource{"demo": {Provider: "test", Family: "Demo"}}, FontPacks: map[string]FontPack{"demo": {Performance: Performance{Class: "bundled"}, Roles: map[string]Role{"body": {Source: "demo", Tier: "prose-core"}}}}}
	lock := &Lockfile{Sources: map[string]LockedSource{"demo": {Provider: "test", Family: "Demo", Repository: "https://example.test/fonts", Revision: "rev1", Directory: "ofl/demo", Files: map[string]LockedFile{"normal": {Source: "Demo-Regular.ttf", SHA256: strings.Repeat("a", 64)}}, License: LockedLicense{ID: "OFL-1.1", File: "OFL.txt", SHA256: licenseHash}}}}
	lockData, _ := yaml.Marshal(lock)
	lockPath := filepath.Join(root, "lock.yaml")
	if err := os.WriteFile(lockPath, lockData, 0o644); err != nil {
		t.Fatal(err)
	}
	return &CatalogSource{Catalog: c, FS: os.DirFS(root), Root: ".", LockFS: os.DirFS(root), Lock: "lock.yaml"}, lock, lockPath
}

func TestVerifySourceRejectsChangedLockProvenance(t *testing.T) {
	fields := []string{"Provider", "Family", "Revision", "Directory"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			source, lock, lockPath := verificationFixture(t)
			locked := lock.Sources["demo"]
			switch field {
			case "Provider":
				locked.Provider = "changed"
			case "Family":
				locked.Family = "changed"
			case "Revision":
				locked.Revision = "changed"
			case "Directory":
				locked.Directory = "changed"
			}
			lock.Sources["demo"] = locked
			data, _ := yaml.Marshal(lock)
			if err := os.WriteFile(lockPath, data, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := VerifySource(source, "demo"); err == nil {
				t.Fatalf("changed %s was accepted", field)
			}
		})
	}
}

func TestVerifySourceRejectsChangedSourceHash(t *testing.T) {
	source, lock, lockPath := verificationFixture(t)
	locked := lock.Sources["demo"]
	locked.Files["normal"] = LockedFile{Source: "Demo-Regular.ttf", SHA256: hex.EncodeToString(make([]byte, 32))}
	lock.Sources["demo"] = locked
	data, _ := yaml.Marshal(lock)
	if err := os.WriteFile(lockPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySource(source, "demo"); err == nil {
		t.Fatal("changed source hash was accepted")
	}
}
