package serveadmin

import (
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

var (
	rebuildChannel   chan struct{}
	buildStatusValue atomic.Value // stores string status: "building", "success", "error"
	buildMessage     atomic.Value // stores string message
	contentDir       = "content"
	configPath       string
	watchEnabled     atomic.Bool
	siteConfigMu     sync.RWMutex
	siteConfig       *models.Config
	sitePostsMu      sync.RWMutex
	sitePosts        []*models.Post
)

func SetContentDir(dir string) {
	contentDir = dir
}

func GetContentDir() string {
	return contentDir
}

func SetConfigPath(path string) {
	configPath = path
}

func GetConfigPath() string {
	return configPath
}

func SetWatchEnabled(enabled bool) {
	watchEnabled.Store(enabled)
}

func IsWatchEnabled() bool {
	return watchEnabled.Load()
}

func SetSiteConfig(cfg *models.Config) {
	siteConfigMu.Lock()
	defer siteConfigMu.Unlock()
	siteConfig = cfg
}

func GetSiteConfig() *models.Config {
	siteConfigMu.RLock()
	defer siteConfigMu.RUnlock()
	return siteConfig
}

func SetSitePosts(posts []*models.Post) {
	sitePostsMu.Lock()
	defer sitePostsMu.Unlock()
	if posts == nil {
		sitePosts = nil
		return
	}
	sitePosts = append([]*models.Post(nil), posts...)
}

func GetSitePosts() []*models.Post {
	sitePostsMu.RLock()
	defer sitePostsMu.RUnlock()
	if sitePosts == nil {
		return nil
	}
	return append([]*models.Post(nil), sitePosts...)
}

func ResolveContentPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	base := contentDir
	if base == "" {
		base = "."
	}
	return filepath.Clean(filepath.Join(base, path))
}

func SetRebuildChannel(ch chan struct{}) {
	rebuildChannel = ch
}

func SetBuildStatus(status, message string) {
	buildStatusValue.Store(status)
	buildMessage.Store(message)
}

func TriggerRebuild() {
	if rebuildChannel != nil {
		select {
		case rebuildChannel <- struct{}{}:
		default:
		}
	}
}

func GetBuildStatus() map[string]interface{} {
	status := buildStatusValue.Load()
	if status == nil {
		status = "success"
	}
	message := buildMessage.Load()
	if message == nil {
		message = "Ready"
	}
	return map[string]interface{}{
		"status":  status,
		"message": message,
	}
}
