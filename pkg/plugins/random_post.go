// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// RandomPostConfig holds configuration for the random_post plugin.
type RandomPostConfig struct {
	// Enabled controls whether the plugin writes /random/ output.
	// Default: false
	Enabled bool

	// Path is the output path segment for the random endpoint.
	// Default: "random"
	Path string

	// EmitPostsJSON controls whether posts.json is also written.
	// Default: false
	EmitPostsJSON bool

	// ExcludeTags is a case-insensitive denylist of tags.
	ExcludeTags []string
}

func defaultRandomPostConfig() RandomPostConfig {
	return RandomPostConfig{
		Enabled:       false,
		Path:          "random",
		EmitPostsJSON: false,
		ExcludeTags:   []string{},
	}
}

// RandomPostPlugin generates a static /random/ endpoint that redirects client-side
// to a random eligible post.
type RandomPostPlugin struct {
	config RandomPostConfig
}

// NewRandomPostPlugin creates a new RandomPostPlugin.
func NewRandomPostPlugin() *RandomPostPlugin {
	return &RandomPostPlugin{config: defaultRandomPostConfig()}
}

// Name returns the unique name of the plugin.
func (p *RandomPostPlugin) Name() string {
	return "random_post"
}

// Configure loads plugin configuration from the manager.
func (p *RandomPostPlugin) Configure(m *lifecycle.Manager) error {
	p.config = parseRandomPostConfig(m.Config())
	return nil
}

// Write writes the random endpoint output files.
func (p *RandomPostPlugin) Write(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	cfg := m.Config()
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	endpointPath := normalizeRandomPostPath(p.config.Path)
	outDir := filepath.Join(outputDir, filepath.FromSlash(endpointPath))
	indexPath := filepath.Join(outDir, "index.html")
	postsJSONPath := filepath.Join(outDir, "posts.json")

	// Avoid clobbering existing outputs (e.g., a post with slug "random").
	if err := ensureOutputDoesNotExist(indexPath); err != nil {
		return err
	}
	if p.config.EmitPostsJSON {
		if err := ensureOutputDoesNotExist(postsJSONPath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outDir, err)
	}

	hrefs := eligibleRandomPostHrefs(m.Posts(), p.config.ExcludeTags)
	hrefsJSON, err := json.Marshal(hrefs)
	if err != nil {
		return fmt.Errorf("marshaling random posts list: %w", err)
	}

	indexHTML := buildRandomPostIndexHTML(hrefs, string(hrefsJSON))
	if err := os.WriteFile(indexPath, []byte(indexHTML), 0o644); err != nil { //nolint:gosec // static HTML needs world-readable permissions for web serving
		return fmt.Errorf("writing random endpoint %s: %w", indexPath, err)
	}

	if p.config.EmitPostsJSON {
		if err := os.WriteFile(postsJSONPath, hrefsJSON, 0o644); err != nil { //nolint:gosec // static JSON needs world-readable permissions for web serving
			return fmt.Errorf("writing random posts json %s: %w", postsJSONPath, err)
		}
	}

	return nil
}

func parseRandomPostConfig(cfg *lifecycle.Config) RandomPostConfig {
	result := defaultRandomPostConfig()
	if cfg == nil {
		return result
	}

	raw := getRandomPostConfigRaw(cfg.Extra)
	if raw == nil {
		return result
	}

	// Also allow directly providing a typed config.
	if typed, ok := raw.(RandomPostConfig); ok {
		return typed
	}

	m := coerceToMapAny(raw)
	if m == nil {
		return result
	}

	applyRandomPostConfigMap(&result, m)
	return result
}

func getRandomPostConfigRaw(extra map[string]interface{}) any {
	if extra == nil {
		return nil
	}
	if v, ok := extra["random_post"]; ok {
		return v
	}
	// Back-compat: allow a nested "markata-go" map.
	if markataGo, ok := extra["markata-go"].(map[string]any); ok {
		return markataGo["random_post"]
	}
	if markataGo, ok := extra["markata-go"].(map[string]interface{}); ok {
		return markataGo["random_post"]
	}
	return nil
}

func coerceToMapAny(raw any) map[string]any {
	if raw == nil {
		return nil
	}
	if m, ok := raw.(map[string]any); ok {
		return m
	}
	if mm, ok := raw.(map[string]interface{}); ok {
		m := make(map[string]any, len(mm))
		for k, v := range mm {
			m[k] = v
		}
		return m
	}
	return nil
}

func applyRandomPostConfigMap(dst *RandomPostConfig, m map[string]any) {
	if dst == nil || m == nil {
		return
	}

	if v, ok := m["enabled"].(bool); ok {
		dst.Enabled = v
	}
	if v, ok := m["path"].(string); ok && strings.TrimSpace(v) != "" {
		dst.Path = v
	}
	if v, ok := m["emit_posts_json"].(bool); ok {
		dst.EmitPostsJSON = v
	}
	if v, ok := m["exclude_tags"]; ok {
		dst.ExcludeTags = coerceStringSlice(v)
	}
}

func coerceStringSlice(v any) []string {
	switch vv := v.(type) {
	case nil:
		return nil
	case []string:
		return append([]string{}, vv...)
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func normalizeRandomPostPath(p string) string {
	s := strings.TrimSpace(p)
	if s == "" {
		s = "random"
	}
	// Normalize to a safe relative path.
	cleaned := path.Clean("/" + strings.Trim(s, "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "" {
		cleaned = "random"
	}
	return cleaned
}

func ensureOutputDoesNotExist(p string) error {
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("random_post output path already exists: %s", p)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking random_post output path %s: %w", p, err)
	}
	return nil
}

func eligibleRandomPostHrefs(posts []*models.Post, excludeTags []string) []string {
	deny := make(map[string]struct{}, len(excludeTags))
	for _, t := range excludeTags {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		deny[tt] = struct{}{}
	}

	uniq := make(map[string]struct{})
	hrefs := make([]string, 0)
	for _, post := range posts {
		if post == nil {
			continue
		}
		if !post.Published || post.Draft || post.Private || post.Skip {
			continue
		}
		if strings.TrimSpace(post.Href) == "" {
			continue
		}
		if len(deny) > 0 {
			excluded := false
			for _, tag := range post.Tags {
				if _, ok := deny[strings.ToLower(strings.TrimSpace(tag))]; ok {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		href := post.Href
		if !strings.HasPrefix(href, "/") {
			href = "/" + href
		}
		if _, ok := uniq[href]; ok {
			continue
		}
		uniq[href] = struct{}{}
		hrefs = append(hrefs, href)
	}

	sort.Strings(hrefs)
	return hrefs
}

func buildRandomPostIndexHTML(hrefs []string, hrefsJSONArray string) string {
	var noscript strings.Builder
	noscript.WriteString("<noscript>")
	noscript.WriteString("<p>JavaScript is required to pick a random post.</p>")
	noscript.WriteString("<p><a href=\"/\">Go home</a></p>")
	if len(hrefs) > 0 {
		noscript.WriteString("<details><summary>Eligible posts</summary><ul>")
		for _, href := range hrefs {
			h := html.EscapeString(href)
			noscript.WriteString("<li><a href=\"")
			noscript.WriteString(h)
			noscript.WriteString("\">")
			noscript.WriteString(h)
			noscript.WriteString("</a></li>")
		}
		noscript.WriteString("</ul></details>")
	}
	noscript.WriteString("</noscript>")

	// Keep HTML stable; the random selection occurs at runtime.
	return "<!doctype html>\n" +
		"<html lang=\"en\">\n" +
		"<head>\n" +
		"  <meta charset=\"utf-8\">\n" +
		"  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" +
		"  <meta name=\"robots\" content=\"noindex, nofollow\">\n" +
		"  <title>Random post</title>\n" +
		"  <style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;margin:2rem;line-height:1.4}code{background:rgba(0,0,0,.06);padding:.1rem .25rem;border-radius:.25rem}</style>\n" +
		"</head>\n" +
		"<body>\n" +
		"  <p>Redirecting to a random post…</p>\n" +
		"  " + noscript.String() + "\n" +
		"  <script>\n" +
		"  (function(){\n" +
		"    var posts = " + hrefsJSONArray + ";\n" +
		"    if (!Array.isArray(posts) || posts.length === 0) {\n" +
		"      return;\n" +
		"    }\n" +
		"    var i = Math.floor(Math.random() * posts.length);\n" +
		"    var href = posts[i];\n" +
		"    if (typeof href !== 'string' || href.length === 0) {\n" +
		"      return;\n" +
		"    }\n" +
		"    var suffix = window.location.search + window.location.hash;\n" +
		"    window.location.replace(href + suffix);\n" +
		"  })();\n" +
		"  </script>\n" +
		"</body>\n" +
		"</html>\n"
}

// Ensure RandomPostPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*RandomPostPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*RandomPostPlugin)(nil)
	_ lifecycle.WritePlugin     = (*RandomPostPlugin)(nil)
)
