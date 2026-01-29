// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// WebmentionIOResponse represents the response from webmention.io API.
type WebmentionIOResponse struct {
	Type     string               `json:"type"`
	Name     string               `json:"name"`
	Children []ReceivedWebMention `json:"children"`
}

// WebmentionsFetchPlugin fetches incoming webmentions from webmention.io.
type WebmentionsFetchPlugin struct {
	config     models.WebMentionsConfig
	siteURL    string
	httpClient *http.Client
	mentions   []ReceivedWebMention
}

// NewWebmentionsFetchPlugin creates a new WebmentionsFetchPlugin.
func NewWebmentionsFetchPlugin() *WebmentionsFetchPlugin {
	return &WebmentionsFetchPlugin{
		config: models.NewWebMentionsConfig(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		mentions: make([]ReceivedWebMention, 0),
	}
}

// Name returns the unique name of the plugin.
func (p *WebmentionsFetchPlugin) Name() string {
	return "webmentions_fetch"
}

// Configure reads configuration options for the plugin.
func (p *WebmentionsFetchPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Get site URL
	if siteURL, ok := config.Extra["url"].(string); ok {
		p.siteURL = siteURL
	}

	// Get webmentions config from Extra as a map
	if wm, ok := config.Extra["webmentions"]; ok {
		// Handle as map[string]interface{} from TOML parsing
		if wmMap, ok := wm.(map[string]interface{}); ok {
			// Extract fields manually
			if enabled, ok := wmMap["enabled"].(bool); ok {
				p.config.Enabled = enabled
			}
			if outgoing, ok := wmMap["outgoing"].(bool); ok {
				p.config.Outgoing = outgoing
			}
			if cacheDir, ok := wmMap["cache_dir"].(string); ok {
				p.config.CacheDir = cacheDir
			}
			if timeout, ok := wmMap["timeout"].(string); ok {
				p.config.Timeout = timeout
			}
			if token, ok := wmMap["webmention_io_token"].(string); ok {
				p.config.WebmentionIOToken = token
			}
			if userAgent, ok := wmMap["user_agent"].(string); ok {
				p.config.UserAgent = userAgent
			}
			if concurrentRequests, ok := wmMap["concurrent_requests"].(int64); ok {
				p.config.ConcurrentRequests = int(concurrentRequests)
			}
		}
	}

	// Set timeout if configured
	if p.config.Timeout != "" {
		if timeout, err := time.ParseDuration(p.config.Timeout); err == nil {
			p.httpClient.Timeout = timeout
		}
	}

	return nil
}

// Priority returns the execution priority for this plugin.
func (p *WebmentionsFetchPlugin) Priority() int {
	return lifecycle.PriorityDefault
}

// FetchMentions fetches all webmentions from webmention.io for the configured domain.
func (p *WebmentionsFetchPlugin) FetchMentions() error {
	// Get token from config or environment variable
	token := p.config.WebmentionIOToken
	if token == "" || strings.HasPrefix(token, "${") {
		// Try environment variable
		token = os.Getenv("WEBMENTION_IO_TOKEN")
	}

	if token == "" {
		return fmt.Errorf("webmention_io_token not configured - set it in config or WEBMENTION_IO_TOKEN environment variable")
	}

	if p.siteURL == "" {
		return fmt.Errorf("site URL not configured")
	}

	// Extract domain from site URL
	domain := extractDomain(p.siteURL)

	// Build API URL
	apiURL := fmt.Sprintf("https://webmention.io/api/mentions.jf2?token=%s&domain=%s&per-page=1000",
		token, domain)

	// Fetch mentions
	ctx, cancel := context.WithTimeout(context.Background(), p.httpClient.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", p.config.UserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch mentions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // Best effort read for error message
		return fmt.Errorf("webmention.io API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var wmResp WebmentionIOResponse
	if err := json.NewDecoder(resp.Body).Decode(&wmResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	p.mentions = wmResp.Children

	// Save to cache
	if err := p.saveMentionsCache(); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}

	return nil
}

// saveMentionsCache saves the fetched mentions to a JSON file.
func (p *WebmentionsFetchPlugin) saveMentionsCache() error {
	if p.config.CacheDir == "" {
		return nil
	}

	// Create cache directory
	if err := os.MkdirAll(p.config.CacheDir, 0o755); err != nil {
		return err
	}

	cacheFile := filepath.Join(p.config.CacheDir, "received_mentions.json")

	data, err := json.MarshalIndent(p.mentions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0o600)
}

// GetMentions returns all fetched mentions.
func (p *WebmentionsFetchPlugin) GetMentions() []ReceivedWebMention {
	return p.mentions
}

// GetMentionsForURL returns mentions for a specific URL.
func (p *WebmentionsFetchPlugin) GetMentionsForURL(targetURL string) []ReceivedWebMention {
	var results []ReceivedWebMention
	for i := range p.mentions {
		if p.mentions[i].Target == targetURL {
			results = append(results, p.mentions[i])
		}
	}
	return results
}

// GroupMentionsByURL groups mentions by their target URL.
func (p *WebmentionsFetchPlugin) GroupMentionsByURL() map[string][]ReceivedWebMention {
	groups := make(map[string][]ReceivedWebMention)
	for i := range p.mentions {
		groups[p.mentions[i].Target] = append(groups[p.mentions[i].Target], p.mentions[i])
	}
	return groups
}

// extractDomain extracts the domain from a URL.
func extractDomain(siteURL string) string {
	// Remove protocol
	domain := siteURL
	if idx := strings.Index(domain, "://"); idx != -1 {
		domain = domain[idx+3:]
	}
	// Remove path
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	// Remove port
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	return domain
}
