package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// OEmbedResponse represents the JSON oEmbed response fields we care about.
// See: https://oembed.com
type OEmbedResponse struct {
	Type            string `json:"type"`
	Version         string `json:"version"`
	Title           string `json:"title"`
	AuthorName      string `json:"author_name"`
	ProviderName    string `json:"provider_name"`
	ProviderURL     string `json:"provider_url"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_width"`
	ThumbnailHeight int    `json:"thumbnail_height"`
	HTML            string `json:"html"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	CacheAge        int    `json:"cache_age"`
}

// oembedProvider describes a single oEmbed provider.
type oembedProvider struct {
	Name             string
	Endpoint         string
	URLPrefixes      []string
	RequiresAuth     bool
	SupportsFormat   bool
	SupportsDiscover bool
}

// oembedResolver resolves oEmbed data for URLs.
type oembedResolver struct {
	client    *http.Client
	config    models.EmbedsConfig
	providers []oembedProvider
}

func newOEmbedResolver(config models.EmbedsConfig, client *http.Client) *oembedResolver {
	return &oembedResolver{
		client:    client,
		config:    config,
		providers: defaultOEmbedProviders(),
	}
}

func newOEmbedResolverWithProviders(config models.EmbedsConfig, client *http.Client, providers []oembedProvider) *oembedResolver {
	return &oembedResolver{
		client:    client,
		config:    config,
		providers: providers,
	}
}

func (r *oembedResolver) updateConfig(config models.EmbedsConfig) {
	r.config = config
}

func (r *oembedResolver) Resolve(rawURL string) (*OEmbedResponse, bool, error) {
	provider := r.matchProvider(rawURL)
	if provider == nil {
		if !r.config.OEmbedAutoDiscover {
			return nil, false, nil
		}

		endpoint, err := r.discoverEndpoint(rawURL)
		if err != nil {
			if errors.Is(err, errOEmbedDiscoveryDisabled) {
				return nil, false, nil
			}
			return nil, false, err
		}

		payload, err := r.fetchOEmbedResponse(endpoint)
		if err != nil {
			return nil, false, err
		}

		return payload, true, nil
	}

	if provider.SupportsDiscover {
		endpoint, err := r.discoverEndpoint(rawURL)
		if err != nil {
			if errors.Is(err, errOEmbedDiscoveryDisabled) {
				return nil, false, nil
			}
			return nil, false, err
		}

		payload, err := r.fetchOEmbedResponse(endpoint)
		if err != nil {
			return nil, false, err
		}

		return payload, true, nil
	}

	if !r.isProviderEnabled(provider.Name) {
		return nil, true, nil
	}

	endpoint, err := r.buildEndpoint(provider, rawURL)
	if err != nil {
		return nil, true, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, true, err
	}

	req.Header.Set("User-Agent", "markata-go/1.0 (+https://github.com/WaylonWalker/markata-go)")
	req.Header.Set("Accept", "application/json")

	payload, err := r.fetchOEmbedResponse(endpoint)
	if err != nil {
		return nil, true, err
	}

	return payload, true, nil
}

func (r *oembedResolver) matchProvider(rawURL string) *oembedProvider {
	for _, provider := range r.providers {
		for _, prefix := range provider.URLPrefixes {
			if strings.HasPrefix(rawURL, prefix) {
				return &provider
			}
		}
	}

	if r.config.OEmbedAutoDiscover {
		for _, provider := range r.providers {
			if provider.SupportsDiscover {
				return &provider
			}
		}
	}

	return nil
}

func (r *oembedResolver) isProviderEnabled(name string) bool {
	if r.config.OEmbedProviders == nil {
		return true
	}

	providerConfig, ok := r.config.OEmbedProviders[strings.ToLower(name)]
	if !ok {
		return true
	}

	return providerConfig.Enabled
}

func (r *oembedResolver) buildEndpoint(provider *oembedProvider, rawURL string) (string, error) {
	encodedURL := url.QueryEscape(rawURL)
	if provider.SupportsFormat {
		return fmt.Sprintf("%s?url=%s&format=json", provider.Endpoint, encodedURL), nil
	}

	parsed, err := url.Parse(provider.Endpoint)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("url", rawURL)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (r *oembedResolver) fetchOEmbedResponse(endpoint string) (*OEmbedResponse, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "markata-go/1.0 (+https://github.com/WaylonWalker/markata-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oembed request failed: %s", resp.Status)
	}

	var payload OEmbedResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func (r *oembedResolver) discoverEndpoint(rawURL string) (string, error) {
	if !r.config.OEmbedAutoDiscover {
		return "", errOEmbedDiscoveryDisabled
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "markata-go/1.0 (+https://github.com/WaylonWalker/markata-go)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oembed discovery failed: %s", resp.Status)
	}

	const maxBody = 512 * 1024
	limited := io.LimitReader(resp.Body, maxBody)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}

	endpoint := findOEmbedLink(string(body))
	if endpoint == "" {
		return "", fmt.Errorf("oembed discovery: endpoint not found")
	}

	if strings.HasPrefix(endpoint, "//") {
		endpoint = "https:" + endpoint
	}

	if strings.HasPrefix(endpoint, "/") {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", err
		}
		endpoint = parsed.ResolveReference(&url.URL{Path: endpoint}).String()
	}

	return endpoint, nil
}

func findOEmbedLink(html string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<link[^>]*rel=["']alternate["'][^>]*type=["']application/json\+oembed["'][^>]*href=["']([^"']+)["']`),
		regexp.MustCompile(`<link[^>]*type=["']application/json\+oembed["'][^>]*rel=["']alternate["'][^>]*href=["']([^"']+)["']`),
		regexp.MustCompile(`<link[^>]*rel=["']alternate["'][^>]*href=["']([^"']+)["'][^>]*type=["']application/json\+oembed["']`),
	}

	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(html)
		if len(match) > 1 {
			return match[1]
		}
	}

	return ""
}

var errOEmbedDiscoveryDisabled = errors.New("oembed discovery disabled")

func defaultOEmbedProviders() []oembedProvider {
	return []oembedProvider{
		{
			Name:           "youtube",
			Endpoint:       "https://www.youtube.com/oembed",
			URLPrefixes:    []string{"https://www.youtube.com/", "https://youtu.be/"},
			RequiresAuth:   false,
			SupportsFormat: true,
		},
		{
			Name:           "vimeo",
			Endpoint:       "https://vimeo.com/api/oembed.json",
			URLPrefixes:    []string{"https://vimeo.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "tiktok",
			Endpoint:       "https://www.tiktok.com/oembed",
			URLPrefixes:    []string{"https://www.tiktok.com/", "https://vm.tiktok.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "flickr",
			Endpoint:       "https://www.flickr.com/services/oembed/",
			URLPrefixes:    []string{"https://www.flickr.com/", "https://flickr.com/"},
			RequiresAuth:   false,
			SupportsFormat: true,
		},
		{
			Name:           "spotify",
			Endpoint:       "https://open.spotify.com/oembed",
			URLPrefixes:    []string{"https://open.spotify.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "soundcloud",
			Endpoint:       "https://soundcloud.com/oembed",
			URLPrefixes:    []string{"https://soundcloud.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "codepen",
			Endpoint:       "https://codepen.io/api/oembed",
			URLPrefixes:    []string{"https://codepen.io/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "codesandbox",
			Endpoint:       "https://codesandbox.io/oembed",
			URLPrefixes:    []string{"https://codesandbox.io/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "jsfiddle",
			Endpoint:       "https://jsfiddle.net/services/oembed/",
			URLPrefixes:    []string{"https://jsfiddle.net/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "observable",
			Endpoint:       "https://api.observablehq.com/oembed",
			URLPrefixes:    []string{"https://observablehq.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "github",
			Endpoint:       "https://github.com/services/oembed",
			URLPrefixes:    []string{"https://gist.github.com/", "https://github.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "slideshare",
			Endpoint:       "https://www.slideshare.net/api/oembed/2",
			URLPrefixes:    []string{"https://www.slideshare.net/", "https://slideshare.net/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "prezi",
			Endpoint:       "https://prezi.com/services/oembed/",
			URLPrefixes:    []string{"https://prezi.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "speakerdeck",
			Endpoint:       "https://speakerdeck.com/oembed.json",
			URLPrefixes:    []string{"https://speakerdeck.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "issuu",
			Endpoint:       "https://issuu.com/oembed",
			URLPrefixes:    []string{"https://issuu.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "datawrapper",
			Endpoint:       "https://api.datawrapper.de/v3/oembed/",
			URLPrefixes:    []string{"https://datawrapper.de/", "https://www.datawrapper.de/", "https://app.datawrapper.de/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "flourish",
			Endpoint:       "https://app.flourish.studio/api/v1/oembed",
			URLPrefixes:    []string{"https://public.flourish.studio/", "https://app.flourish.studio/", "https://flourish.studio/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "infogram",
			Endpoint:       "https://infogram.com/oembed",
			URLPrefixes:    []string{"https://infogram.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "reddit",
			Endpoint:       "https://www.reddit.com/oembed",
			URLPrefixes:    []string{"https://www.reddit.com/", "https://reddit.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "dailymotion",
			Endpoint:       "https://www.dailymotion.com/services/oembed",
			URLPrefixes:    []string{"https://www.dailymotion.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "wistia",
			Endpoint:       "https://fast.wistia.com/oembed.json",
			URLPrefixes:    []string{"https://wistia.com/", "https://fast.wistia.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:           "giphy",
			Endpoint:       "https://giphy.com/services/oembed",
			URLPrefixes:    []string{"https://giphy.com/"},
			RequiresAuth:   false,
			SupportsFormat: false,
		},
		{
			Name:             "oembed",
			Endpoint:         "",
			URLPrefixes:      []string{},
			RequiresAuth:     false,
			SupportsFormat:   false,
			SupportsDiscover: true,
		},
	}
}
