package fontpacks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Lockfile struct {
	Sources map[string]LockedSource `yaml:"sources"`
}
type LockedSource struct {
	Provider  string                `yaml:"provider"`
	Family    string                `yaml:"family"`
	Revision  string                `yaml:"revision"`
	Directory string                `yaml:"directory"`
	Files     map[string]LockedFile `yaml:"files"`
	License   LockedLicense         `yaml:"license"`
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

// Verify checks every family/tier referenced by a bundled pack, including the
// generated file hash, license file, WOFF2 signature, and lockfile source.
func Verify(c *Catalog, catalogRoot, lockPath string, selected ...string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("read font lockfile: %w", err)
	}
	var lock Lockfile
	if err := yaml.Unmarshal(lockData, &lock); err != nil {
		return fmt.Errorf("parse font lockfile: %w", err)
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
		if pack.Performance.Class != "bundled" {
			continue
		}
		for _, role := range pack.Roles {
			if role.Source != "" {
				used[role.Source] = true
			}
		}
	}
	for source := range used {
		manifestPath := filepath.Join(catalogRoot, source, "manifest.yaml")
		data, err := os.ReadFile(manifestPath)
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
		if locked.Revision == "" || locked.Directory == "" {
			return fmt.Errorf("source %q has incomplete lock resolution", source)
		}
		licensePath := filepath.Join(catalogRoot, source, manifest.License.File)
		if err := verifyHash(licensePath, manifest.License.SHA256); err != nil {
			return fmt.Errorf("source %q license: %w", source, err)
		}
		if locked.License.SHA256 != manifest.License.SHA256 {
			return fmt.Errorf("source %q lock/license hash mismatch", source)
		}
		for tierName, tier := range manifest.Tiers {
			path := filepath.Join(catalogRoot, source, tier.File)
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("source %q tier %q: %w", source, tierName, err)
			}
			if tier.Bytes > 0 && tier.Bytes != info.Size() {
				return fmt.Errorf("source %q tier %q byte count is %d, want %d", source, tierName, info.Size(), tier.Bytes)
			}
			if err := verifyHash(path, tier.SHA256); err != nil {
				return fmt.Errorf("source %q tier %q: %w", source, tierName, err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !bytes.HasPrefix(b, []byte("wOF2")) {
				return fmt.Errorf("source %q tier %q is not a WOFF2 file", source, tierName)
			}
		}
	}
	return nil
}

func verifyHash(path, expected string) error {
	if expected == "" {
		return fmt.Errorf("%s has no recorded sha256", path)
	}
	hash, _, err := AssetHash(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(expected, hash) {
		return fmt.Errorf("sha256 %q does not match %s", hash, expected)
	}
	return nil
}
