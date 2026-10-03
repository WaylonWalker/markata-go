package fontcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"sync"
)

var bundledFingerprint = sync.OnceValues(func() (string, error) {
	return metadataFingerprint(FS())
})

// Fingerprint identifies immutable catalog, lockfile, and manifest metadata.
// Release verification checks the font binaries against these manifests.
func Fingerprint() (string, error) {
	return bundledFingerprint()
}

func metadataFingerprint(source fs.FS) (string, error) {
	paths := []string{"markata-fontpacks.yaml", "markata-fonts.lock.yaml"}
	if err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("inspect font metadata %q: %w", name, err)
		}
		if !entry.IsDir() && path.Base(name) == "manifest.yaml" {
			paths = append(paths, name)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	metadata := make([]struct {
		Path string
		Data []byte
	}, 0, len(paths))
	for _, name := range paths {
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return "", fmt.Errorf("read font metadata %q: %w", name, err)
		}
		metadata = append(metadata, struct {
			Path string
			Data []byte
		}{Path: name, Data: data})
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("encode font metadata fingerprint: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
