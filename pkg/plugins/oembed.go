package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Name           string
	Endpoint       string
	URLPrefixes    []string
	RequiresAuth   bool
	SupportsFormat bool
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
		return nil, false, nil
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

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, true, fmt.Errorf("oembed request failed: %s", resp.Status)
	}

	var payload OEmbedResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&payload); err != nil {
		return nil, true, err
	}

	return &payload, true, nil
}

func (r *oembedResolver) matchProvider(rawURL string) *oembedProvider {
	for _, provider := range r.providers {
		for _, prefix := range provider.URLPrefixes {
			if strings.HasPrefix(rawURL, prefix) {
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
	}
}
