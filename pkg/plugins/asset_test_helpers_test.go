package plugins

import (
	"path/filepath"
	"strings"
)

func containsRenderedAsset(content, assetPath string) bool {
	if strings.Contains(content, assetPath) {
		return true
	}

	extension := filepath.Ext(assetPath)
	prefix := strings.TrimSuffix(assetPath, extension) + "."
	for start := 0; start < len(content); {
		relative := strings.Index(content[start:], prefix)
		if relative < 0 {
			return false
		}
		hashStart := start + relative + len(prefix)
		if len(content)-hashStart >= 8+len(extension) {
			hash := content[hashStart : hashStart+8]
			if isAssetHash(hash) && strings.HasPrefix(content[hashStart+8:], extension) {
				return true
			}
		}
		start = hashStart
	}
	return false
}

func isAssetHash(value string) bool {
	if len(value) != 8 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
