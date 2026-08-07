package fontpacks

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func verificationFixture(t *testing.T) (*CatalogSource, *Lockfile, string, string) {
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
	licenseHash, _, err := AssetSHA256(filepath.Join(family, "OFL.txt"))
	if err != nil {
		t.Fatal(err)
	}
	fontHash, _, err := AssetSHA256(filepath.Join(family, "demo.woff2"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "demo", Family: "Demo", Source: ManifestSource{Provider: "test", Repository: "https://example.test/fonts", Revision: "rev1", Directory: "ofl/demo", Files: map[string]string{"normal": strings.Repeat("a", 64)}}, License: License{ID: "OFL-1.1", File: "OFL.txt", SHA256: licenseHash}, Faces: map[string]Face{"normal": {SourceFile: "Demo-Regular.ttf"}}, Tiers: map[string]Tier{"prose-core": {File: "demo.woff2", Profile: "prose-core", SHA256: fontHash, Bytes: int64(len(font))}}}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "manifest.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Catalog{SystemStacks: map[string]SystemStack{"sans": {CSS: "system-ui"}}, FontSources: map[string]FontSource{"demo": {Provider: "test", Family: "Demo"}}, FontPacks: map[string]FontPack{"demo": {Performance: Performance{Class: "bundled"}, Roles: map[string]Role{"body": {Source: "demo", Tier: "prose-core"}}}}}
	lock := &Lockfile{Sources: map[string]LockedSource{"demo": {Provider: "test", Family: "Demo", Repository: "https://example.test/fonts", Revision: "rev1", Directory: "ofl/demo", Files: map[string]LockedFile{"normal": {Source: "Demo-Regular.ttf", SHA256: strings.Repeat("a", 64)}}, License: LockedLicense{ID: "OFL-1.1", File: "OFL.txt", SHA256: licenseHash}}}}
	lockData, err := yaml.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, "lock.yaml")
	if err := os.WriteFile(lockPath, lockData, 0o644); err != nil {
		t.Fatal(err)
	}
	return &CatalogSource{Catalog: c, FS: os.DirFS(root), Root: ".", LockFS: os.DirFS(root), Lock: "lock.yaml"}, lock, lockPath, filepath.Join(family, "manifest.yaml")
}

func TestVerifySourceRejectsChangedLockProvenance(t *testing.T) {
	fields := []string{"Provider", "Family", "Revision", "Directory"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			source, lock, lockPath, _ := verificationFixture(t)
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
			data, err := yaml.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
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
	source, lock, lockPath, _ := verificationFixture(t)
	locked := lock.Sources["demo"]
	locked.Files["normal"] = LockedFile{Source: "Demo-Regular.ttf", SHA256: hex.EncodeToString(make([]byte, 32))}
	lock.Sources["demo"] = locked
	data, err := yaml.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySource(source, "demo"); err == nil {
		t.Fatal("changed source hash was accepted")
	}
}

func TestVerifySourceRequiresExactManifestLockFileSet(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]LockedFile
	}{
		{name: "missing", files: map[string]LockedFile{}},
		{name: "unexpected", files: map[string]LockedFile{
			"normal": {Source: "Demo-Regular.ttf", SHA256: strings.Repeat("a", 64)},
			"italic": {Source: "Demo-Italic.ttf", SHA256: strings.Repeat("b", 64)},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, lock, lockPath, _ := verificationFixture(t)
			locked := lock.Sources["demo"]
			locked.Files = tc.files
			lock.Sources["demo"] = locked
			data, err := yaml.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(lockPath, data, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := VerifySource(source, "demo"); err == nil {
				t.Fatal("inexact source file set was accepted")
			}
		})
	}
}

func TestVerifySourceRejectsMissingManifestProvenance(t *testing.T) {
	source, _, _, manifestPath := verificationFixture(t)
	// The fixture uses an OS directory filesystem, so rewrite the source file
	// directly after removing its authoritative source.files mapping.
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Source.Files = nil
	data, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySource(source, "demo"); err == nil {
		t.Fatal("manifest without source file provenance was accepted")
	}
}

func TestVerifySourceAcceptsMatchingManifestProvenance(t *testing.T) {
	source, lock, lockPath, manifestPath := verificationFixture(t)
	_ = lock
	_ = lockPath
	_ = manifestPath
	if err := VerifySource(source, "demo"); err != nil {
		t.Fatalf("matching provenance rejected: %v", err)
	}
}

func TestVerifySourceRejectsDuplicateTierContent(t *testing.T) {
	source, _, lockPath, manifestPath := verificationFixture(t)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Tiers["latin-ext"] = Tier{File: "demo.woff2", Profile: "latin-ext", SHA256: manifest.Tiers["prose-core"].SHA256, Bytes: manifest.Tiers["prose-core"].Bytes, UnicodeRange: []string{"U+0100"}}
	data, err = yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// The lock remains valid; the duplicate tier must be rejected before asset checks.
	if err := VerifySource(source, "demo"); err == nil {
		t.Fatalf("duplicate tier content was accepted (lock %s)", lockPath)
	}
}

func TestValidateRoleCapabilitiesChecksStaticAndVariableFaces(t *testing.T) {
	c := &Catalog{FontSources: map[string]FontSource{"demo": {Family: "Demo"}}}
	packs := map[string]FontPack{"pack": {Performance: Performance{Class: bundledPerformanceClass}, Roles: map[string]Role{
		"heading": {Source: "demo", Weight: 700},
	}}}
	static := map[string]Manifest{"demo": {Faces: map[string]Face{"normal": {Style: "normal", Weight: []float64{400, 400}}}}}
	if err := ValidateRoleCapabilities(c, packs, static); err == nil {
		t.Fatal("unsupported static weight was accepted")
	}
	variable := map[string]Manifest{"demo": {Faces: map[string]Face{"normal": {Style: "normal", Variable: true, Weight: []float64{300, 800}, Axes: map[string][]float64{"wght": {300, 800}}}}}}
	if err := ValidateRoleCapabilities(c, packs, variable); err != nil {
		t.Fatalf("variable weight rejected: %v", err)
	}
	packs["pack"].Roles["heading"] = Role{Source: "demo", Style: "italic", Weight: 400}
	if err := ValidateRoleCapabilities(c, packs, variable); err == nil {
		t.Fatal("unsupported italic style was accepted")
	}
}

func TestValidateRoleCapabilitiesChecksOpticalSizeAxisAndRange(t *testing.T) {
	c := &Catalog{FontSources: map[string]FontSource{"demo": {Family: "Demo"}}}
	packs := map[string]FontPack{"pack": {Performance: Performance{Class: bundledPerformanceClass}, Roles: map[string]Role{
		"heading": {Source: "demo", OpticalSize: 72},
	}}}
	valid := map[string]Manifest{"demo": {Faces: map[string]Face{"normal": {
		Style: "normal", Variable: true, Axes: map[string][]float64{"opsz": {9, 144}},
	}}}}
	if err := ValidateRoleCapabilities(c, packs, valid); err != nil {
		t.Fatalf("supported optical size rejected: %v", err)
	}

	for _, test := range []struct {
		name        string
		axis        []float64
		opticalSize float64
	}{
		{name: "missing axis", opticalSize: 72},
		{name: "too low", axis: []float64{9, 144}, opticalSize: 8},
		{name: "too high", axis: []float64{9, 144}, opticalSize: 145},
	} {
		name, axis := test.name, test.axis
		role := packs["pack"].Roles["heading"]
		role.OpticalSize = test.opticalSize
		packs["pack"].Roles["heading"] = role
		manifest := map[string]Manifest{"demo": {Faces: map[string]Face{"normal": {
			Style: "normal", Variable: true, Axes: map[string][]float64{},
		}}}}
		if axis != nil {
			manifest["demo"].Faces["normal"] = Face{Style: "normal", Variable: true, Axes: map[string][]float64{"opsz": axis}}
		}
		if err := ValidateRoleCapabilities(c, packs, manifest); err == nil {
			t.Errorf("%s optical-size capability was accepted", name)
		}
	}
}
