// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// LinkAvatarsConfig holds configuration for the link_avatars plugin.
type LinkAvatarsConfig struct {
	// Enabled controls whether the plugin is active.
	// Default: false
	Enabled bool

	// Selector is the CSS selector for links to enhance.
	// Default: "a[href^='http']"
	Selector string

	// Service is the avatar service provider: "duckduckgo", "google", "custom"
	// Default: "duckduckgo"
	Service string

	// Template is a custom URL template (only used when Service = "custom").
	// Supports placeholders: {origin}, {host}
	Template string

	// IgnoreDomains is a list of domains to skip.
	IgnoreDomains []string

	// IgnoreOrigins is a list of full origins to skip (includes protocol).
	IgnoreOrigins []string

	// IgnoreSelectors is a list of CSS selectors to exclude.
	IgnoreSelectors []string

	// IgnoreClasses is a list of CSS classes to exclude.
	IgnoreClasses []string

	// IgnoreIDs is a list of element IDs to exclude.
	IgnoreIDs []string

	// Size is the avatar icon size in pixels.
	// Default: 16
	Size int

	// Position is where to place the avatar: "before" or "after" link text.
	// Default: "before"
	Position string
}

// defaultLinkAvatarsConfig returns the default configuration.
func defaultLinkAvatarsConfig() LinkAvatarsConfig {
	return LinkAvatarsConfig{
		Enabled:         false,
		Selector:        "a[href^='http']",
		Service:         "duckduckgo",
		Template:        "",
		IgnoreDomains:   []string{},
		IgnoreOrigins:   []string{},
		IgnoreSelectors: []string{},
		IgnoreClasses:   []string{},
		IgnoreIDs:       []string{},
		Size:            16,
		Position:        "before",
	}
}

// LinkAvatarsPlugin adds favicon/avatar icons next to external links.
// It generates client-side JavaScript and CSS assets that enhance links
// at runtime in the browser.
type LinkAvatarsPlugin struct {
	config LinkAvatarsConfig
}

// NewLinkAvatarsPlugin creates a new LinkAvatarsPlugin.
func NewLinkAvatarsPlugin() *LinkAvatarsPlugin {
	return &LinkAvatarsPlugin{config: defaultLinkAvatarsConfig()}
}

// Name returns the unique name of the plugin.
func (p *LinkAvatarsPlugin) Name() string {
	return "link_avatars"
}

// Configure loads plugin configuration from the manager.
func (p *LinkAvatarsPlugin) Configure(m *lifecycle.Manager) error {
	p.config = parseLinkAvatarsConfig(m.Config())
	return nil
}

// Write generates the JavaScript and CSS assets and injects head tags.
func (p *LinkAvatarsPlugin) Write(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	cfg := m.Config()
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}

	// Create assets directory
	assetsDir := filepath.Join(outputDir, "assets", "markata")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("creating link_avatars assets directory: %w", err)
	}

	// Generate JavaScript
	jsContent := p.generateJavaScript()
	jsPath := filepath.Join(assetsDir, "link-avatars.js")
	if err := os.WriteFile(jsPath, []byte(jsContent), 0o644); err != nil { //nolint:gosec // static JS needs world-readable permissions
		return fmt.Errorf("writing link-avatars.js: %w", err)
	}

	// Generate CSS
	cssContent := p.generateCSS()
	cssPath := filepath.Join(assetsDir, "link-avatars.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0o644); err != nil { //nolint:gosec // static CSS needs world-readable permissions
		return fmt.Errorf("writing link-avatars.css: %w", err)
	}

	// Inject head tags
	p.injectHeadTags(cfg)

	return nil
}

// generateJavaScript generates the client-side JavaScript.
func (p *LinkAvatarsPlugin) generateJavaScript() string {
	// Marshal config values for JavaScript - this won't fail with basic Go types
	configJSON, err := json.Marshal(map[string]interface{}{
		"selector":        p.config.Selector,
		"service":         p.config.Service,
		"template":        p.config.Template,
		"ignoreDomains":   p.config.IgnoreDomains,
		"ignoreOrigins":   p.config.IgnoreOrigins,
		"ignoreSelectors": p.config.IgnoreSelectors,
		"ignoreClasses":   p.config.IgnoreClasses,
		"ignoreIds":       p.config.IgnoreIDs,
		"size":            p.config.Size,
		"position":        p.config.Position,
	})
	if err != nil {
		// Fallback to empty config on error (should never happen with basic types)
		configJSON = []byte("{}")
	}

	return `/**
 * Link Avatars - markata-go
 * Adds favicon icons next to external links
 */
(function() {
  'use strict';

  var config = ` + string(configJSON) + `;

  // Service URL templates
  var serviceTemplates = {
    'duckduckgo': 'https://icons.duckduckgo.com/ip3/{host}.ico',
    'google': 'https://www.google.com/s2/favicons?domain={host}&sz=' + config.size
  };

  function getFaviconURL(href) {
    try {
      var url = new URL(href);
      var host = url.hostname;
      var origin = url.origin;
      var template;

      if (config.service === 'custom' && config.template) {
        template = config.template;
      } else {
        template = serviceTemplates[config.service] || serviceTemplates['duckduckgo'];
      }

      return template
        .replace('{host}', host)
        .replace('{origin}', encodeURIComponent(origin));
    } catch (e) {
      return null;
    }
  }

  function shouldIgnoreLink(link) {
    var href = link.href;
    var url;

    try {
      url = new URL(href);
    } catch (e) {
      return true; // Invalid URL
    }

    // Skip same-origin links
    if (url.origin === window.location.origin) {
      return true;
    }

    // Check ignore domains
    for (var i = 0; i < config.ignoreDomains.length; i++) {
      if (url.hostname === config.ignoreDomains[i] ||
          url.hostname.endsWith('.' + config.ignoreDomains[i])) {
        return true;
      }
    }

    // Check ignore origins
    for (var j = 0; j < config.ignoreOrigins.length; j++) {
      if (url.origin === config.ignoreOrigins[j]) {
        return true;
      }
    }

    // Check ignore classes
    for (var k = 0; k < config.ignoreClasses.length; k++) {
      if (link.classList.contains(config.ignoreClasses[k])) {
        return true;
      }
    }

    // Check ignore IDs (link is inside an element with ignored ID)
    for (var l = 0; l < config.ignoreIds.length; l++) {
      if (link.closest('#' + config.ignoreIds[l])) {
        return true;
      }
    }

    // Check ignore selectors
    for (var m = 0; m < config.ignoreSelectors.length; m++) {
      try {
        if (link.matches(config.ignoreSelectors[m])) {
          return true;
        }
      } catch (e) {
        // Invalid selector, skip
      }
    }

    return false;
  }

  function processLink(link) {
    if (link.classList.contains('has-avatar')) {
      return; // Already processed
    }

    if (shouldIgnoreLink(link)) {
      return;
    }

    var faviconURL = getFaviconURL(link.href);
    if (!faviconURL) {
      return;
    }

    link.setAttribute('data-favicon', faviconURL);
    link.style.setProperty('--favicon-url', 'url("' + faviconURL + '")');
    link.classList.add('has-avatar');
    link.classList.add('has-avatar-' + config.position);
  }

  function processLinks() {
    var links = document.querySelectorAll(config.selector);
    links.forEach(processLink);
  }

  // Use Intersection Observer for lazy loading if available
  function observeLinks() {
    if (!('IntersectionObserver' in window)) {
      processLinks();
      return;
    }

    var observer = new IntersectionObserver(function(entries) {
      entries.forEach(function(entry) {
        if (entry.isIntersecting) {
          processLink(entry.target);
          observer.unobserve(entry.target);
        }
      });
    }, { rootMargin: '100px' });

    var links = document.querySelectorAll(config.selector);
    links.forEach(function(link) {
      if (!shouldIgnoreLink(link)) {
        observer.observe(link);
      }
    });
  }

  // Initialize
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', observeLinks);
  } else {
    observeLinks();
  }
})();
`
}

// generateCSS generates the CSS styles for link avatars.
func (p *LinkAvatarsPlugin) generateCSS() string {
	size := p.config.Size
	if size <= 0 {
		size = 16
	}

	return fmt.Sprintf(`/**
 * Link Avatars - markata-go
 * Styles for favicon icons next to external links
 */

/* Common avatar styles */
a.has-avatar {
  position: relative;
}

a.has-avatar::before,
a.has-avatar::after {
  content: '';
  display: inline-block;
  width: %dpx;
  height: %dpx;
  background-image: var(--favicon-url);
  background-size: contain;
  background-repeat: no-repeat;
  background-position: center;
  vertical-align: middle;
  opacity: 0;
  transition: opacity 0.2s ease;
}

/* Show avatar when image loads */
a.has-avatar::before,
a.has-avatar::after {
  opacity: 0.8;
}

/* Avatar before link text */
a.has-avatar-before::before {
  margin-right: 0.35em;
}

a.has-avatar-before::after {
  display: none;
}

/* Avatar after link text */
a.has-avatar-after::after {
  margin-left: 0.35em;
}

a.has-avatar-after::before {
  display: none;
}

/* Hover effect */
a.has-avatar:hover::before,
a.has-avatar:hover::after {
  opacity: 1;
}

/* Hide avatar if image fails to load (fallback) */
a.has-avatar[data-favicon-error]::before,
a.has-avatar[data-favicon-error]::after {
  display: none;
}
`, size, size)
}

// injectHeadTags adds the CSS and JS references to the config head.
func (p *LinkAvatarsPlugin) injectHeadTags(cfg *lifecycle.Config) {
	if cfg.Extra == nil {
		cfg.Extra = make(map[string]interface{})
	}

	// Get existing head config or create new one
	headConfig := getOrCreateHeadConfig(cfg)

	// Add CSS link
	headConfig.Link = append(headConfig.Link, models.LinkTag{
		Rel:  "stylesheet",
		Href: "/assets/markata/link-avatars.css",
	})

	// Add JS script
	headConfig.Script = append(headConfig.Script, models.ScriptTag{
		Src: "/assets/markata/link-avatars.js",
	})

	// Mark link avatars as enabled for templates
	cfg.Extra["link_avatars_enabled"] = true
}

// getOrCreateHeadConfig gets the existing head config or creates a new one.
func getOrCreateHeadConfig(cfg *lifecycle.Config) *models.HeadConfig {
	if cfg.Extra == nil {
		cfg.Extra = make(map[string]interface{})
	}

	// Check if we have direct access to models.Config through Extra
	if modelsConfig, ok := cfg.Extra["_models_config"].(*models.Config); ok {
		return &modelsConfig.Head
	}

	// For standalone lifecycle.Config, we need to use Extra to store head elements
	// The templates plugin will read these
	links, linksOK := cfg.Extra["head_links"].([]models.LinkTag)
	if !linksOK {
		links = []models.LinkTag{}
	}
	scripts, scriptsOK := cfg.Extra["head_scripts"].([]models.ScriptTag)
	if !scriptsOK {
		scripts = []models.ScriptTag{}
	}

	headConfig := &models.HeadConfig{
		Link:   links,
		Script: scripts,
	}

	// Store back
	cfg.Extra["head_links"] = headConfig.Link
	cfg.Extra["head_scripts"] = headConfig.Script

	return headConfig
}

// parseLinkAvatarsConfig parses the configuration from the manager config.
func parseLinkAvatarsConfig(cfg *lifecycle.Config) LinkAvatarsConfig {
	result := defaultLinkAvatarsConfig()
	if cfg == nil || cfg.Extra == nil {
		return result
	}

	raw := getLinkAvatarsConfigRaw(cfg.Extra)
	if raw == nil {
		return result
	}

	// Allow directly providing a typed config
	if typed, ok := raw.(LinkAvatarsConfig); ok {
		return typed
	}

	m := coerceToMapAny(raw)
	if m == nil {
		return result
	}

	applyLinkAvatarsConfigMap(&result, m)
	return result
}

// getLinkAvatarsConfigRaw retrieves the raw config from Extra.
func getLinkAvatarsConfigRaw(extra map[string]interface{}) any {
	if extra == nil {
		return nil
	}
	if v, ok := extra["link_avatars"]; ok {
		return v
	}
	// Back-compat: allow a nested "markata-go" map
	if markataGo, ok := extra["markata-go"].(map[string]any); ok {
		return markataGo["link_avatars"]
	}
	if markataGo, ok := extra["markata-go"].(map[string]interface{}); ok {
		return markataGo["link_avatars"]
	}
	return nil
}

// applyLinkAvatarsConfigMap applies map values to the config.
func applyLinkAvatarsConfigMap(dst *LinkAvatarsConfig, m map[string]any) {
	if dst == nil || m == nil {
		return
	}

	applyLinkAvatarsBasicFields(dst, m)
	applyLinkAvatarsIgnoreFields(dst, m)
	applyLinkAvatarsSizeAndPosition(dst, m)
}

// applyLinkAvatarsBasicFields applies basic string/bool fields.
func applyLinkAvatarsBasicFields(dst *LinkAvatarsConfig, m map[string]any) {
	if v, ok := m["enabled"].(bool); ok {
		dst.Enabled = v
	}
	if v, ok := m["selector"].(string); ok && strings.TrimSpace(v) != "" {
		dst.Selector = v
	}
	if v, ok := m["service"].(string); ok && strings.TrimSpace(v) != "" {
		dst.Service = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := m["template"].(string); ok {
		dst.Template = v
	}
}

// applyLinkAvatarsIgnoreFields applies ignore list fields.
func applyLinkAvatarsIgnoreFields(dst *LinkAvatarsConfig, m map[string]any) {
	if v, ok := m["ignore_domains"]; ok {
		dst.IgnoreDomains = coerceStringSlice(v)
	}
	if v, ok := m["ignore_origins"]; ok {
		dst.IgnoreOrigins = coerceStringSlice(v)
	}
	if v, ok := m["ignore_selectors"]; ok {
		dst.IgnoreSelectors = coerceStringSlice(v)
	}
	if v, ok := m["ignore_classes"]; ok {
		dst.IgnoreClasses = coerceStringSlice(v)
	}
	if v, ok := m["ignore_ids"]; ok {
		dst.IgnoreIDs = coerceStringSlice(v)
	}
}

// applyLinkAvatarsSizeAndPosition applies size and position fields.
func applyLinkAvatarsSizeAndPosition(dst *LinkAvatarsConfig, m map[string]any) {
	// Size can come as int, int64, or float64 depending on the source
	if v, ok := m["size"].(int); ok && v > 0 {
		dst.Size = v
	} else if v, ok := m["size"].(int64); ok && v > 0 {
		dst.Size = int(v)
	} else if v, ok := m["size"].(float64); ok && v > 0 {
		dst.Size = int(v)
	}

	if v, ok := m["position"].(string); ok {
		pos := strings.ToLower(strings.TrimSpace(v))
		if pos == "before" || pos == "after" {
			dst.Position = pos
		}
	}
}

// SetConfig sets the link avatars configuration directly (for testing).
func (p *LinkAvatarsPlugin) SetConfig(config LinkAvatarsConfig) {
	p.config = config
}

// Config returns the current configuration (for testing).
func (p *LinkAvatarsPlugin) Config() LinkAvatarsConfig {
	return p.config
}

// Ensure LinkAvatarsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*LinkAvatarsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*LinkAvatarsPlugin)(nil)
	_ lifecycle.WritePlugin     = (*LinkAvatarsPlugin)(nil)
)
