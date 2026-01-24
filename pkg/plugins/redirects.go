// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// Redirect represents a single URL redirect rule.
type Redirect struct {
	// Original is the source path (the old URL).
	Original string

	// New is the destination path (the new URL).
	New string
}

// RedirectsConfig holds configuration for the redirects plugin.
type RedirectsConfig struct {
	// RedirectsFile is the path to the _redirects file.
	// Default: "static/_redirects"
	RedirectsFile string

	// RedirectTemplate is an optional path to a custom template.
	// If empty, the default template is used.
	RedirectTemplate string
}

// RedirectsPlugin generates HTML redirect pages from a _redirects file.
// It creates index.html files at each source path that redirect to the destination.
type RedirectsPlugin struct {
	config RedirectsConfig
}

// NewRedirectsPlugin creates a new RedirectsPlugin with default configuration.
func NewRedirectsPlugin() *RedirectsPlugin {
	return &RedirectsPlugin{
		config: RedirectsConfig{
			RedirectsFile: "static/_redirects",
		},
	}
}

// Name returns the unique name of the plugin.
func (p *RedirectsPlugin) Name() string {
	return "redirects"
}

// Configure loads plugin configuration from the manager.
func (p *RedirectsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Look for redirects configuration in [markata-go.redirects] or [redirects]
	var redirectsConfig map[string]interface{}

	// Try markata-go.redirects first
	if markataGo, ok := config.Extra["markata-go"].(map[string]interface{}); ok {
		if rc, ok := markataGo["redirects"].(map[string]interface{}); ok {
			redirectsConfig = rc
		}
	}

	// Fall back to top-level redirects
	if redirectsConfig == nil {
		if rc, ok := config.Extra["redirects"].(map[string]interface{}); ok {
			redirectsConfig = rc
		}
	}

	if redirectsConfig == nil {
		return nil
	}

	// Extract configuration values
	if rf, ok := redirectsConfig["redirects_file"].(string); ok && rf != "" {
		p.config.RedirectsFile = rf
	}
	if rt, ok := redirectsConfig["redirect_template"].(string); ok && rt != "" {
		p.config.RedirectTemplate = rt
	}

	return nil
}

// Write generates redirect pages for each redirect rule.
func (p *RedirectsPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Read the redirects file
	redirectsContent, err := os.ReadFile(p.config.RedirectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No redirects file, skip silently
			return nil
		}
		return fmt.Errorf("reading redirects file %s: %w", p.config.RedirectsFile, err)
	}

	// Check cache to avoid regeneration
	// Include custom template hash in cache key if configured
	cacheKey := fmt.Sprintf("redirects:%x", hashContent(redirectsContent))
	if p.config.RedirectTemplate != "" {
		if templateContent, err := os.ReadFile(p.config.RedirectTemplate); err == nil {
			cacheKey = fmt.Sprintf("redirects:%x:%x", hashContent(redirectsContent), hashContent(templateContent))
		}
	}
	if cached, ok := m.Cache().Get(cacheKey); ok {
		if cached == "done" {
			return nil
		}
	}

	// Parse redirect rules
	redirects := p.parseRedirects(string(redirectsContent))
	if len(redirects) == 0 {
		return nil
	}

	// Load template
	tmpl, err := p.loadTemplate()
	if err != nil {
		return fmt.Errorf("loading redirect template: %w", err)
	}

	// Generate redirect pages
	for _, redirect := range redirects {
		if err := p.writeRedirect(redirect, tmpl, outputDir, config); err != nil {
			// Log error but continue with other redirects
			fmt.Fprintf(os.Stderr, "warning: failed to write redirect for %s: %v\n", redirect.Original, err)
		}
	}

	// Mark as done in cache
	m.Cache().Set(cacheKey, "done")

	return nil
}

// parseRedirects parses the redirects file content into redirect rules.
func (p *RedirectsPlugin) parseRedirects(content string) []Redirect {
	var redirects []Redirect

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Skip wildcards (not supported for static generation)
		if strings.Contains(line, "*") {
			continue
		}

		// Split by whitespace
		parts := strings.Fields(line)
		if len(parts) < 2 {
			// Malformed line, skip
			continue
		}

		original := parts[0]
		newPath := parts[1]

		// Validate source path starts with /
		if !strings.HasPrefix(original, "/") {
			continue
		}

		// Validate destination: must be absolute path or external URL
		if !strings.HasPrefix(newPath, "/") &&
			!strings.HasPrefix(newPath, "http://") &&
			!strings.HasPrefix(newPath, "https://") {
			continue
		}

		redirects = append(redirects, Redirect{
			Original: original,
			New:      newPath,
		})
	}

	return redirects
}

// loadTemplate loads the redirect template (custom or default).
func (p *RedirectsPlugin) loadTemplate() (*template.Template, error) {
	if p.config.RedirectTemplate != "" {
		// Load custom template
		content, err := os.ReadFile(p.config.RedirectTemplate)
		if err != nil {
			log.Printf("warning: failed to read custom redirect template %s: %v, using default", p.config.RedirectTemplate, err)
			return template.New("redirect").Parse(defaultRedirectTemplate)
		}
		tmpl, err := template.New("redirect").Parse(string(content))
		if err != nil {
			log.Printf("warning: failed to parse custom redirect template %s: %v, using default", p.config.RedirectTemplate, err)
			return template.New("redirect").Parse(defaultRedirectTemplate)
		}
		return tmpl, nil
	}

	return template.New("redirect").Parse(defaultRedirectTemplate)
}

// writeRedirect writes a single redirect page.
func (p *RedirectsPlugin) writeRedirect(redirect Redirect, tmpl *template.Template, outputDir string, config *lifecycle.Config) error {
	// Calculate output path: output_dir/original_path/index.html
	// Strip leading slash and create directory structure
	relativePath := strings.TrimPrefix(redirect.Original, "/")
	if relativePath == "" {
		// Can't redirect from root
		return nil
	}

	// Clean the path and validate no path traversal
	cleanPath := filepath.Clean(relativePath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected in redirect source: %s", redirect.Original)
	}

	postDir := filepath.Join(outputDir, cleanPath)

	// Create directory
	if err := os.MkdirAll(postDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", postDir, err)
	}

	// Prepare template data
	data := struct {
		Original string
		New      string
		Config   *lifecycle.Config
	}{
		Original: redirect.Original,
		New:      redirect.New,
		Config:   config,
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Write index.html
	outputPath := filepath.Join(postDir, "index.html")
	//nolint:gosec // G306: HTML output files need 0644 for web serving
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}

// hashContent creates a simple hash of content for caching.
func hashContent(content []byte) uint64 {
	var hash uint64 = 5381
	for _, b := range content {
		hash = ((hash << 5) + hash) + uint64(b)
	}
	return hash
}

// defaultRedirectTemplate is the built-in HTML template for redirect pages.
const defaultRedirectTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta http-equiv="Refresh" content="0; url='{{ .New }}'" />
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="canonical" href="{{ .New }}" />
  <meta name="description" content="{{ .Original }} has been moved to {{ .New }}." />
  <title>{{ .Original }} has been moved to {{ .New }}</title>
  <style>
    html {
      font-family: system-ui, sans-serif;
      background: #1f2022;
      color: #eefbfe;
    }
    body {
      margin: 5rem auto;
      max-width: 800px;
      padding: 0 1rem;
    }
    a {
      color: #fb30c4;
      text-decoration-color: #e1bd00c9;
    }
    code {
      background: #2a2a2e;
      padding: 0.2em 0.4em;
      border-radius: 3px;
    }
  </style>
</head>
<body>
  <h1>Page Moved</h1>
  <p>
    <code>{{ .Original }}</code> has moved to
    <a href="{{ .New }}">{{ .New }}</a>
  </p>
</body>
</html>`

// Priority returns the plugin priority for the given stage.
// Redirects should run late in the write stage, after content is written.
func (p *RedirectsPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLate
	}
	return lifecycle.PriorityDefault
}

// SetConfig allows setting the plugin configuration directly (useful for testing).
func (p *RedirectsPlugin) SetConfig(config RedirectsConfig) {
	p.config = config
}

// GetConfig returns the current plugin configuration.
func (p *RedirectsPlugin) GetConfig() RedirectsConfig {
	return p.config
}

// Ensure RedirectsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*RedirectsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*RedirectsPlugin)(nil)
	_ lifecycle.WritePlugin     = (*RedirectsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*RedirectsPlugin)(nil)
)
