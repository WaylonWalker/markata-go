package plugins

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

const (
	wellKnownDir           = ".well-known"
	wellKnownGeneratorName = "markata-go"
	wellKnownGeneratorVer  = "unknown"
	wellKnownNodeInfoRel   = "http://nodeinfo.diaspora.software/ns/schema/2.0"
	wellKnownWebfingerRel  = "http://webfinger.net/rel/profile-page"
	wellKnownAvatarRel     = "http://webfinger.net/rel/avatar"
	wellKnownHostMetaRel   = "lrdd"
	wellKnownHostMetaType  = "application/jrd+json"
	wellKnownWebfingerType = "text/html"
)

type wellKnownData struct {
	SiteURL          string
	SiteTitle        string
	SiteDescription  string
	SiteHost         string
	Author           string
	AuthorImageURL   string
	BuildTime        string
	WebfingerSubject string
	WebfingerURL     string
	WebfingerAliases []string
	NodeInfoURL      string
	GeneratorName    string
	GeneratorVersion string
	SSHFingerprint   string
	KeybaseUsername  string
}

func (d wellKnownData) toMap() map[string]interface{} {
	return map[string]interface{}{
		"site_url":          d.SiteURL,
		"site_title":        d.SiteTitle,
		"site_description":  d.SiteDescription,
		"site_host":         d.SiteHost,
		"author":            d.Author,
		"author_image_url":  d.AuthorImageURL,
		"build_time":        d.BuildTime,
		"webfinger_subject": d.WebfingerSubject,
		"webfinger_url":     d.WebfingerURL,
		"webfinger_aliases": d.WebfingerAliases,
		"nodeinfo_url":      d.NodeInfoURL,
		"generator":         d.GeneratorName,
		"generator_version": d.GeneratorVersion,
		"ssh_fingerprint":   d.SSHFingerprint,
		"keybase_username":  d.KeybaseUsername,
	}
}

type wellKnownEntry struct {
	name     string
	path     string
	template string
	fallback func(wellKnownData) string
}

// WellKnownPlugin generates .well-known endpoints from site metadata.
type WellKnownPlugin struct {
	now func() time.Time
}

// NewWellKnownPlugin creates a new WellKnownPlugin.
func NewWellKnownPlugin() *WellKnownPlugin {
	return &WellKnownPlugin{now: time.Now}
}

// Name returns the unique name of the plugin.
func (p *WellKnownPlugin) Name() string {
	return "well_known"
}

// Write generates .well-known files during the write stage.
func (p *WellKnownPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	wellKnownConfig := getWellKnownConfig(config)
	if !wellKnownConfig.IsEnabled() {
		return nil
	}

	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	var engine *templates.Engine
	if cached, ok := m.Cache().Get("templates.engine"); ok && cached != nil {
		if e, ok := cached.(*templates.Engine); ok {
			engine = e
		}
	}

	data := buildWellKnownData(config, wellKnownConfig, p.now())
	entries := resolveWellKnownEntries(wellKnownConfig)

	// Add avatar endpoint if author image is available
	if data.AuthorImageURL != "" {
		entries = append(entries, wellKnownEntries["avatar"])
	}

	for _, entry := range entries {
		content, err := p.renderEntry(entry, engine, config, m, data)
		if err != nil {
			return err
		}
		// Skip empty content (e.g., avatar with no image)
		if content == "" || content == "\n" {
			continue
		}
		if err := writeWellKnownFile(config.OutputDir, entry.path, content); err != nil {
			return err
		}
	}

	return nil
}

func getWellKnownConfig(config *lifecycle.Config) models.WellKnownConfig {
	if config != nil && config.Extra != nil {
		if wk, ok := config.Extra["well_known"].(models.WellKnownConfig); ok {
			return wk
		}
	}
	return models.NewWellKnownConfig()
}

func buildWellKnownData(config *lifecycle.Config, wellKnownConfig models.WellKnownConfig, buildTime time.Time) wellKnownData {
	siteURL := strings.TrimSuffix(getSiteURL(config), "/")
	if siteURL == "" {
		siteURL = DefaultSiteURL
	}

	siteTitle := getSiteTitle(config)
	siteDescription := getSiteDescription(config)
	siteHost := extractHost(siteURL)
	author := getStringFromExtra(config.Extra, "author")
	authorImageURL := getAuthorImageURL(config, siteURL)
	webfingerURL := siteURL + "/" + wellKnownDir + "/webfinger"
	nodeInfoURL := siteURL + "/nodeinfo/2.0"

	webfingerSubject := siteURL
	if author != "" && siteHost != "" {
		webfingerSubject = fmt.Sprintf("acct:%s@%s", models.Slugify(author), siteHost)
	}

	return wellKnownData{
		SiteURL:          siteURL,
		SiteTitle:        siteTitle,
		SiteDescription:  siteDescription,
		SiteHost:         siteHost,
		Author:           author,
		AuthorImageURL:   authorImageURL,
		BuildTime:        buildTime.UTC().Format(time.RFC3339),
		WebfingerSubject: webfingerSubject,
		WebfingerURL:     webfingerURL,
		WebfingerAliases: []string{siteURL},
		NodeInfoURL:      nodeInfoURL,
		GeneratorName:    wellKnownGeneratorName,
		GeneratorVersion: wellKnownGeneratorVer,
		SSHFingerprint:   wellKnownConfig.SSHFingerprint,
		KeybaseUsername:  wellKnownConfig.KeybaseUsername,
	}
}

func extractHost(siteURL string) string {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		return strings.TrimSpace(siteURL)
	}
	return host
}

// getAuthorImageURL returns the absolute URL of the author's avatar image.
// It checks seo.author_image first, falling back to seo.default_image.
// Relative paths are resolved against the site URL.
func getAuthorImageURL(config *lifecycle.Config, siteURL string) string {
	if config == nil || config.Extra == nil {
		return ""
	}

	// Try to get SEO config
	seoVal, ok := config.Extra["seo"]
	if !ok {
		return ""
	}

	seoMap, ok := seoVal.(map[string]interface{})
	if !ok {
		// Try as SEOConfig struct
		if seo, ok := seoVal.(models.SEOConfig); ok {
			imageURL := seo.AuthorImage
			if imageURL == "" {
				imageURL = seo.DefaultImage
			}
			return resolveImageURL(imageURL, siteURL)
		}
		return ""
	}

	// Get author_image, fallback to default_image
	imageURL := ""
	if authorImage, ok := seoMap["author_image"].(string); ok && authorImage != "" {
		imageURL = authorImage
	} else if defaultImage, ok := seoMap["default_image"].(string); ok && defaultImage != "" {
		imageURL = defaultImage
	}

	return resolveImageURL(imageURL, siteURL)
}

// resolveImageURL converts a potentially relative image URL to an absolute URL.
func resolveImageURL(imageURL, siteURL string) string {
	if imageURL == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}

	// Relative path - resolve against site URL
	if strings.HasPrefix(imageURL, "/") {
		return strings.TrimSuffix(siteURL, "/") + imageURL
	}

	// Relative without leading slash
	return strings.TrimSuffix(siteURL, "/") + "/" + imageURL
}

func resolveWellKnownEntries(wellKnownConfig models.WellKnownConfig) []wellKnownEntry {
	entries := make([]wellKnownEntry, 0)
	seen := make(map[string]bool)

	addEntry := func(entry wellKnownEntry) {
		if entry.name == "" || seen[entry.name] {
			return
		}
		entries = append(entries, entry)
		seen[entry.name] = true
	}

	for _, name := range wellKnownConfig.AutoGenerateList() {
		switch name {
		case "nodeinfo":
			addEntry(wellKnownEntries["nodeinfo"])
			addEntry(wellKnownEntries["nodeinfo-2.0"])
		default:
			if entry, ok := wellKnownEntries[name]; ok {
				addEntry(entry)
			}
		}
	}

	if wellKnownConfig.SSHFingerprint != "" {
		addEntry(wellKnownEntries["sshfp"])
	}
	if wellKnownConfig.KeybaseUsername != "" {
		addEntry(wellKnownEntries["keybase"])
	}

	return entries
}

func (p *WellKnownPlugin) renderEntry(entry wellKnownEntry, engine *templates.Engine, config *lifecycle.Config, m *lifecycle.Manager, data wellKnownData) (string, error) {
	if engine != nil && engine.TemplateExists(entry.template) {
		ctx := templates.NewContext(nil, "", ToModelsConfig(config))
		ctx.Extra["well_known"] = data.toMap()
		ctx = ctx.WithCore(m)
		result, err := engine.Render(entry.template, ctx)
		if err == nil {
			return normalizeWellKnownContent(result), nil
		}
	}

	if entry.fallback == nil {
		return "", fmt.Errorf("no template or fallback for %s", entry.name)
	}

	return normalizeWellKnownContent(entry.fallback(data)), nil
}

func writeWellKnownFile(outputDir, relativePath, content string) error {
	fullPath := filepath.Join(outputDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", fullPath, err)
	}
	//nolint:gosec // G306: Output files need 0644 for web serving
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", fullPath, err)
	}
	return nil
}

func normalizeWellKnownContent(content string) string {
	trimmed := strings.TrimRight(content, "\n")
	return trimmed + "\n"
}

func withTrailingSlash(value string) string {
	if value == "" {
		return value
	}
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

var wellKnownEntries = map[string]wellKnownEntry{
	"host-meta": {
		name:     "host-meta",
		path:     wellKnownDir + "/host-meta",
		template: "well-known/host-meta.xml",
		fallback: func(data wellKnownData) string {
			return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
  <Subject>%s</Subject>
  <Link rel="%s" type="%s" template="%s?resource={uri}" />
</XRD>`, withTrailingSlash(data.SiteURL), wellKnownHostMetaRel, wellKnownHostMetaType, data.WebfingerURL)
		},
	},
	"host-meta.json": {
		name:     "host-meta.json",
		path:     wellKnownDir + "/host-meta.json",
		template: "well-known/host-meta.json",
		fallback: func(data wellKnownData) string {
			return fmt.Sprintf(`{
  "subject": "%s",
  "links": [
    {
      "rel": "%s",
      "type": "%s",
      "template": "%s?resource={uri}"
    }
  ]
}`, withTrailingSlash(data.SiteURL), wellKnownHostMetaRel, wellKnownHostMetaType, data.WebfingerURL)
		},
	},
	"webfinger": {
		name:     "webfinger",
		path:     wellKnownDir + "/webfinger",
		template: "well-known/webfinger.json",
		fallback: func(data wellKnownData) string {
			// Build links array - always include profile-page
			links := fmt.Sprintf(`{
      "rel": "%s",
      "type": "%s",
      "href": "%s"
    }`, wellKnownWebfingerRel, wellKnownWebfingerType, withTrailingSlash(data.SiteURL))

			// Add avatar link if author image is available
			if data.AuthorImageURL != "" {
				links += fmt.Sprintf(`,
    {
      "rel": "%s",
      "href": "%s"
    }`, wellKnownAvatarRel, data.AuthorImageURL)
			}

			return fmt.Sprintf(`{
  "subject": "%s",
  "aliases": [
    "%s"
  ],
  "links": [
    %s
  ]
}`, data.WebfingerSubject, withTrailingSlash(data.SiteURL), links)
		},
	},
	"nodeinfo": {
		name:     "nodeinfo",
		path:     wellKnownDir + "/nodeinfo",
		template: "well-known/nodeinfo.json",
		fallback: func(data wellKnownData) string {
			return fmt.Sprintf(`{
  "links": [
    {
      "rel": "%s",
      "href": "%s"
    }
  ]
}`, wellKnownNodeInfoRel, data.NodeInfoURL)
		},
	},
	"nodeinfo-2.0": {
		name:     "nodeinfo-2.0",
		path:     "nodeinfo/2.0",
		template: "well-known/nodeinfo-2.0.json",
		fallback: func(data wellKnownData) string {
			return fmt.Sprintf(`{
  "version": "2.0",
  "software": {
    "name": "%s",
    "version": "%s"
  },
  "protocols": [],
  "services": {
    "inbound": [],
    "outbound": []
  },
  "openRegistrations": false,
  "usage": {
    "users": {
      "total": 0
    }
  },
  "metadata": {
    "site": {
      "name": "%s",
      "description": "%s",
      "url": "%s"
    }
  }
}`, data.GeneratorName, data.GeneratorVersion, data.SiteTitle, data.SiteDescription, withTrailingSlash(data.SiteURL))
		},
	},
	"time": {
		name:     "time",
		path:     wellKnownDir + "/time",
		template: "well-known/time.txt",
		fallback: func(data wellKnownData) string {
			return data.BuildTime
		},
	},
	"sshfp": {
		name:     "sshfp",
		path:     wellKnownDir + "/sshfp",
		template: "well-known/sshfp.txt",
		fallback: func(data wellKnownData) string {
			return data.SSHFingerprint
		},
	},
	"keybase": {
		name:     "keybase",
		path:     wellKnownDir + "/keybase.txt",
		template: "well-known/keybase.txt",
		fallback: func(data wellKnownData) string {
			return fmt.Sprintf("keybase: %s", data.KeybaseUsername)
		},
	},
	"avatar": {
		name:     "avatar",
		path:     wellKnownDir + "/avatar",
		template: "well-known/avatar",
		fallback: func(data wellKnownData) string {
			// Return an HTML redirect to the author image
			// This works better than raw image bytes for static hosting
			if data.AuthorImageURL == "" {
				return ""
			}
			return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="refresh" content="0;url=%s">
<link rel="canonical" href="%s">
</head>
<body>
<a href="%s">Avatar</a>
</body>
</html>`, data.AuthorImageURL, data.AuthorImageURL, data.AuthorImageURL)
		},
	},
}

// Ensure WellKnownPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin      = (*WellKnownPlugin)(nil)
	_ lifecycle.WritePlugin = (*WellKnownPlugin)(nil)
)
