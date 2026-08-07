package fontpacks

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Lockfile struct {
	Schema         string                  `yaml:"schema"`
	CatalogVersion string                  `yaml:"catalog_version"`
	Repository     string                  `yaml:"repository"`
	Revision       string                  `yaml:"revision"`
	Sources        map[string]LockedSource `yaml:"sources"`
}
type LockedSource struct {
	Provider   string                `yaml:"provider"`
	Family     string                `yaml:"family"`
	Repository string                `yaml:"repository"`
	Revision   string                `yaml:"revision"`
	Directory  string                `yaml:"directory"`
	Files      map[string]LockedFile `yaml:"files"`
	License    LockedLicense         `yaml:"license"`
}
type LockedFile struct {
	Source string `yaml:"source"`
	SHA256 string `yaml:"sha256"`
}
type LockedLicense struct {
	ID     string `yaml:"id"`
	File   string `yaml:"file"`
	SHA256 string `yaml:"sha256"`
}

const bundledPerformanceClass = "bundled"

// Verify checks every family/tier referenced by a bundled pack, including the
// generated file hash, license file, WOFF2 signature, and lockfile source.
func Verify(c *Catalog, catalogRoot, lockPath string, selected ...string) error {
	s := &CatalogSource{Catalog: c, FS: os.DirFS(catalogRoot), Root: ".", LockFS: os.DirFS(filepath.Dir(lockPath)), Lock: filepath.Base(lockPath)}
	return VerifySource(s, selected...)
}

// VerifySource verifies provenance and integrity using the source filesystem.
func VerifySource(s *CatalogSource, selected ...string) error {
	c := s.Catalog
	if err := c.Validate(); err != nil {
		return err
	}
	lock, err := loadLock(s)
	if err != nil {
		return err
	}
	packs := c.FontPacks
	if len(selected) > 0 && selected[0] != "" {
		name, pack, err := c.ResolvePack(selected[0])
		if err != nil {
			return err
		}
		packs = map[string]FontPack{name: pack}
	}
	used := map[string]bool{}
	for _, pack := range packs {
		if pack.Performance.Class != bundledPerformanceClass {
			continue
		}
		for _, role := range pack.Roles {
			if role.Source != "" {
				used[role.Source] = true
			}
		}
	}
	for source := range used {
		sourceErr := verifyBundledSource(s, source, lock)
		if sourceErr != nil {
			return sourceErr
		}
	}
	return nil
}

// BundledScope reports the number of bundled packs and unique sources.
func BundledScope(c *Catalog) (packs, sources int) {
	used := map[string]bool{}
	for _, pack := range c.FontPacks {
		if pack.Performance.Class != "bundled" {
			continue
		}
		packs++
		for _, role := range pack.Roles {
			if role.Source != "" {
				used[role.Source] = true
			}
		}
	}
	return packs, len(used)
}

func BundledScopeForPack(pack FontPack) (packs, sources int) {
	used := map[string]bool{}
	if pack.Performance.Class == bundledPerformanceClass {
		for _, role := range pack.Roles {
			if role.Source != "" {
				used[role.Source] = true
			}
		}
	}
	return 1, len(used)
}

//nolint:gocyclo // provenance verification has independent checks for each locked artifact.
func verifyBundledSource(s *CatalogSource, source string, lock Lockfile) error {
	manifestPath := filepath.ToSlash(filepath.Join(s.Root, source, "manifest.yaml"))
	data, err := fs.ReadFile(s.FS, manifestPath)
	if err != nil {
		return fmt.Errorf("source %q manifest: %w", source, err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("source %q manifest: %w", source, err)
	}
	locked, ok := lock.Sources[source]
	if !ok {
		return fmt.Errorf("source %q is missing from font lockfile", source)
	}
	if locked.Provider != manifest.Source.Provider || locked.Family != manifest.Family || locked.Repository != manifest.Source.Repository || locked.Revision != manifest.Source.Revision || locked.Directory != manifest.Source.Directory {
		return fmt.Errorf("source %q lock provenance does not match manifest", source)
	}
	if locked.Revision == "" || locked.Directory == "" || locked.Provider == "" || locked.Family == "" || locked.Repository == "" {
		return fmt.Errorf("source %q has incomplete lock resolution", source)
	}
	if locked.License.ID != manifest.License.ID || locked.License.File != manifest.License.File || locked.License.SHA256 != manifest.License.SHA256 {
		return fmt.Errorf("source %q lock/license metadata mismatch", source)
	}
	licensePath := filepath.ToSlash(filepath.Join(s.Root, source, manifest.License.File))
	if err := verifyHashFS(s.FS, licensePath, manifest.License.SHA256); err != nil {
		return fmt.Errorf("source %q license: %w", source, err)
	}
	if err := verifyTiers(s, source, manifest); err != nil {
		return err
	}
	if len(manifest.Source.Files) == 0 {
		return fmt.Errorf("source %q manifest has no source file provenance", source)
	}
	if len(locked.Files) != len(manifest.Source.Files) {
		return fmt.Errorf("source %q lock and manifest source file sets differ", source)
	}
	for name, file := range locked.Files {
		expected, exists := manifest.Source.Files[name]
		if !exists {
			return fmt.Errorf("source %q manifest is missing file %q", source, name)
		}
		if !fullSHA256(expected) || file.Source == "" || !fullSHA256(file.SHA256) {
			return fmt.Errorf("source %q lock file %q has invalid provenance hash", source, name)
		}
		matched := false
		for _, face := range manifest.Faces {
			if face.SourceFile == file.Source {
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("source %q lock file %q is not represented by manifest", source, name)
		}
		if expected != file.SHA256 {
			return fmt.Errorf("source %q file %q hash differs from manifest", source, name)
		}
	}
	for name := range manifest.Source.Files {
		if _, exists := locked.Files[name]; !exists {
			return fmt.Errorf("source %q lock is missing file %q", source, name)
		}
	}
	return nil
}

func verifyTiers(s *CatalogSource, source string, manifest Manifest) error {
	seenHashes := map[string]struct {
		name         string
		unicodeRange []string
	}{}
	for tierName, tier := range manifest.Tiers {
		if previous, ok := seenHashes[tier.SHA256]; ok && !sameStrings(previous.unicodeRange, tier.UnicodeRange) {
			return fmt.Errorf("source %q tiers %q and %q have different unicode ranges but identical content", source, previous.name, tierName)
		}
		seenHashes[tier.SHA256] = struct {
			name         string
			unicodeRange []string
		}{tierName, tier.UnicodeRange}
		path := filepath.ToSlash(filepath.Join(s.Root, source, tier.File))
		info, err := fs.Stat(s.FS, path)
		if err != nil {
			return fmt.Errorf("source %q tier %q: %w", source, tierName, err)
		}
		if tier.Bytes > 0 && tier.Bytes != info.Size() {
			return fmt.Errorf("source %q tier %q byte count is %d, want %d", source, tierName, info.Size(), tier.Bytes)
		}
		if err := verifyHashFS(s.FS, path, tier.SHA256); err != nil {
			return fmt.Errorf("source %q tier %q: %w", source, tierName, err)
		}
		b, err := fs.ReadFile(s.FS, path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(b, []byte("wOF2")) {
			return fmt.Errorf("source %q tier %q is not a WOFF2 file", source, tierName)
		}
	}
	return nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func verifyHashFS(source fs.FS, path, expected string) error {
	if expected == "" {
		return fmt.Errorf("%s has no recorded sha256", path)
	}
	if !fullSHA256(expected) {
		return fmt.Errorf("%s has a non-canonical sha256 %q", path, expected)
	}
	hash, _, err := AssetSHA256FS(source, path)
	if err != nil {
		return err
	}
	if expected != hash {
		return fmt.Errorf("sha256 %q does not match %s", hash, expected)
	}
	return nil
}

var sha256RE = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

func fullSHA256(value string) bool { return sha256RE.MatchString(strings.TrimSpace(value)) }
