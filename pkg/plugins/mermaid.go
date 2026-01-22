// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"html"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// MermaidPlugin converts Mermaid code blocks into rendered diagrams.
// It runs at the render stage (post_render, after markdown conversion).
type MermaidPlugin struct {
	config models.MermaidConfig
}

// NewMermaidPlugin creates a new MermaidPlugin with default settings.
func NewMermaidPlugin() *MermaidPlugin {
	return &MermaidPlugin{
		config: models.NewMermaidConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *MermaidPlugin) Name() string {
	return "mermaid"
}

// Priority returns the plugin's priority for a given stage.
// This plugin runs after render_markdown (which has default priority 0).
func (p *MermaidPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after render_markdown
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "mermaid" key.
func (p *MermaidPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for mermaid config in Extra
	mermaidConfig, ok := config.Extra["mermaid"]
	if !ok {
		return nil
	}

	// Handle map configuration
	if cfgMap, ok := mermaidConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if cdnURL, ok := cfgMap["cdn_url"].(string); ok && cdnURL != "" {
			p.config.CDNURL = cdnURL
		}
		if theme, ok := cfgMap["theme"].(string); ok && theme != "" {
			p.config.Theme = theme
		}
	}

	return nil
}

// Render processes mermaid code blocks in the rendered HTML for all posts.
func (p *MermaidPlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	return m.ProcessPostsConcurrently(p.processPost)
}

// mermaidCodeBlockRegex matches <pre><code class="language-mermaid"> blocks.
// It captures the diagram code inside.
var mermaidCodeBlockRegex = regexp.MustCompile(
	`<pre><code class="language-mermaid"[^>]*>([\s\S]*?)</code></pre>`,
)

// processPost processes a single post's HTML for mermaid code blocks.
func (p *MermaidPlugin) processPost(post *models.Post) error {
	// Skip posts marked as skip or with no HTML content
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	// Check if there are any mermaid code blocks
	if !strings.Contains(post.ArticleHTML, `class="language-mermaid"`) {
		return nil
	}

	// Track if we found any mermaid blocks
	foundMermaid := false

	// Replace mermaid code blocks with proper mermaid pre tags
	result := mermaidCodeBlockRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		foundMermaid = true

		// Extract the diagram code
		submatches := mermaidCodeBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		// Decode HTML entities in the diagram code (goldmark encodes them)
		diagramCode := html.UnescapeString(submatches[1])

		// Trim whitespace from the diagram code
		diagramCode = strings.TrimSpace(diagramCode)

		// Return the mermaid pre block
		return `<pre class="mermaid">` + "\n" + diagramCode + "\n</pre>"
	})

	// If we found mermaid blocks, inject the script
	if foundMermaid {
		result = p.injectMermaidScript(result)
	}

	post.ArticleHTML = result
	return nil
}

// injectMermaidScript adds the Mermaid.js initialization script to the HTML.
// The script is only injected once per post.
func (p *MermaidPlugin) injectMermaidScript(htmlContent string) string {
	// Build the script tag
	script := `
<script type="module">
  import mermaid from '` + p.config.CDNURL + `';
  mermaid.initialize({ startOnLoad: true, theme: '` + p.config.Theme + `' });
</script>`

	// Append the script to the end of the content
	return htmlContent + script
}

// SetConfig sets the mermaid configuration directly.
// This is useful for testing or programmatic configuration.
func (p *MermaidPlugin) SetConfig(config models.MermaidConfig) {
	p.config = config
}

// Config returns the current mermaid configuration.
func (p *MermaidPlugin) Config() models.MermaidConfig {
	return p.config
}

// Ensure MermaidPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*MermaidPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*MermaidPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*MermaidPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*MermaidPlugin)(nil)
)
