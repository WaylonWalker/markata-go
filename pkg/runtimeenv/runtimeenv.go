package runtimeenv

import (
	"os"
	"strings"
)

const (
	EnvOffline               = "MARKATA_GO_OFFLINE"
	EnvDisableDotEnv         = "MARKATA_GO_DISABLE_DOTENV"
	EnvEncryptionEnabled     = "MARKATA_GO_ENCRYPTION_ENABLED"
	EnvBundledAssetsCacheDir = "MARKATA_GO_BUNDLED_ASSETS_CACHE_DIR"
	EnvBundledMermaidDir     = "MARKATA_GO_BUNDLED_MERMAID_DIR"
)

// OfflineEnabled reports whether runtime network access should be treated as disabled.
func OfflineEnabled() bool {
	return parseBool(os.Getenv(EnvOffline))
}

// DotEnvDisabled reports whether automatic loading of the current directory's
// .env file should be skipped.
func DotEnvDisabled() bool {
	return parseBool(os.Getenv(EnvDisableDotEnv))
}

// BundledAssetsCacheDir returns the bundled CDN asset cache directory, if configured.
func BundledAssetsCacheDir() string {
	return strings.TrimSpace(os.Getenv(EnvBundledAssetsCacheDir))
}

// BundledMermaidDir returns the bundled Mermaid JS directory, if configured.
func BundledMermaidDir() string {
	return strings.TrimSpace(os.Getenv(EnvBundledMermaidDir))
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
