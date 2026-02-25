package templates

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

var (
	trustedMediaMu      sync.RWMutex
	trustedMediaDomains map[string]struct{}
	videoMimeTypes      = map[string]string{
		".mp4":  "video/mp4",
		".m4v":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".ogv":  "video/ogg",
		".ogg":  "video/ogg",
	}
	posterAliasOrder = []string{
		"poster_image",
		"poster",
		"video_poster",
		"video_thumbnail",
		"thumbnail",
		"thumb",
	}
)

func init() {
	SetTrustedMediaDomains(models.DefaultTrustedMediaDomains)
}

// SetTrustedMediaDomains replaces the trusted domains allowlist for media helpers.
func SetTrustedMediaDomains(domains []string) {
	trustedMediaMu.Lock()
	defer trustedMediaMu.Unlock()
	trustedMediaDomains = map[string]struct{}{}
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		trustedMediaDomains[domain] = struct{}{}
	}
}

// WithSize appends or overwrites the w/h query parameters for trusted URLs.
func WithSize(raw string, width, height int) string {
	if width <= 0 || height <= 0 {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if !isTrustedURL(u) {
		return raw
	}
	qs := u.Query()
	qs.Set("w", strconv.Itoa(width))
	qs.Set("h", strconv.Itoa(height))
	u.RawQuery = qs.Encode()
	return u.String()
}

// IsVideoURL reports whether a URL ends with a known video extension.
func IsVideoURL(raw string) bool {
	ext := extensionFromURL(raw)
	_, ok := videoMimeTypes[ext]
	return ok
}

// VideoMIMEType infers the video MIME type for a URL.
func VideoMIMEType(raw string) string {
	ext := extensionFromURL(raw)
	return videoMimeTypes[ext]
}

// IsTrustedMediaURL reports whether the host/url is trusted for helper modifications.
func IsTrustedMediaURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return isTrustedURL(u)
}

// PosterURLFromMap resolves the first poster alias value or derives a .webp poster for trusted video URLs.
func PosterURLFromMap(data map[string]interface{}, mediaURL string) string {
	if data == nil {
		return ""
	}
	for _, alias := range posterAliasOrder {
		if val := stringFromMap(data, alias); val != "" {
			return val
		}
	}
	if mediaURL == "" || !IsTrustedMediaURL(mediaURL) {
		return ""
	}
	return derivePosterFromVideo(mediaURL)
}

func isTrustedURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	if u.Host == "" {
		return true
	}
	host := strings.ToLower(u.Hostname())
	trustedMediaMu.RLock()
	defer trustedMediaMu.RUnlock()
	_, ok := trustedMediaDomains[host]
	return ok
}

func extensionFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(filepath.Ext(raw))
	}
	return strings.ToLower(filepath.Ext(u.Path))
}

func derivePosterFromVideo(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	ext := filepath.Ext(u.Path)
	if ext == "" {
		return ""
	}
	u.Path = strings.TrimSuffix(u.Path, ext) + ".webp"
	return u.String()
}

func stringFromMap(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}
