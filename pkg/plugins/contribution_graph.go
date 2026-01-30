// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// ContributionGraphPlugin converts contribution-graph JSON code blocks into
// rendered Cal-Heatmap calendar heatmaps showing GitHub-style activity.
// It runs at the render stage (after markdown conversion).
type ContributionGraphPlugin struct {
	config    models.ContributionGraphConfig
	idCounter uint64
}

// NewContributionGraphPlugin creates a new ContributionGraphPlugin with default settings.
func NewContributionGraphPlugin() *ContributionGraphPlugin {
	return &ContributionGraphPlugin{
		config: models.NewContributionGraphConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *ContributionGraphPlugin) Name() string {
	return "contribution_graph"
}

// Priority returns the plugin's priority for a given stage.
// This plugin runs after render_markdown (which has default priority 0).
func (p *ContributionGraphPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after render_markdown
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "contribution_graph" key.
func (p *ContributionGraphPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for contribution_graph config in Extra
	graphConfig, ok := config.Extra["contribution_graph"]
	if !ok {
		return nil
	}

	// Handle map configuration
	if cfgMap, ok := graphConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if cdnURL, ok := cfgMap["cdn_url"].(string); ok && cdnURL != "" {
			p.config.CDNURL = cdnURL
		}
		if containerClass, ok := cfgMap["container_class"].(string); ok && containerClass != "" {
			p.config.ContainerClass = containerClass
		}
		if theme, ok := cfgMap["theme"].(string); ok && theme != "" {
			p.config.Theme = theme
		}
	}

	return nil
}

// Render processes contribution-graph code blocks in the rendered HTML for all posts.
func (p *ContributionGraphPlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	return m.ProcessPostsConcurrently(p.processPost)
}

// contributionGraphCodeBlockRegex matches <pre><code class="language-contribution-graph"> blocks.
// It captures the JSON content inside.
var contributionGraphCodeBlockRegex = regexp.MustCompile(
	`<pre><code class="language-contribution-graph"[^>]*>([\s\S]*?)</code></pre>`,
)

// processPost processes a single post's HTML for contribution-graph code blocks.
func (p *ContributionGraphPlugin) processPost(post *models.Post) error {
	// Skip posts marked as skip or with no HTML content
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	// Check if there are any contribution-graph code blocks
	if !strings.Contains(post.ArticleHTML, `class="language-contribution-graph"`) {
		return nil
	}

	// Track if we found any contribution-graph blocks and collect initialization scripts
	var initScripts []string

	// Replace contribution-graph code blocks with div elements
	result := contributionGraphCodeBlockRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		// Extract the JSON content
		submatches := contributionGraphCodeBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		// Decode HTML entities in the JSON (goldmark encodes them)
		jsonContent := html.UnescapeString(submatches[1])
		jsonContent = strings.TrimSpace(jsonContent)

		// Validate and parse the JSON content
		var graphConfig map[string]interface{}
		if err := json.Unmarshal([]byte(jsonContent), &graphConfig); err != nil {
			// Invalid JSON, return with an error comment
			return fmt.Sprintf(`<div class="%s contribution-graph-error">
  <p>Contribution Graph Error: Invalid JSON configuration</p>
  <pre>%s</pre>
</div>`, p.config.ContainerClass, html.EscapeString(err.Error()))
		}

		// Generate a unique ID for this graph
		graphID := fmt.Sprintf("contribution-graph-%d", atomic.AddUint64(&p.idCounter, 1))

		// Extract data and options from the config
		data := graphConfig["data"]
		options := graphConfig["options"]
		if options == nil {
			options = map[string]interface{}{}
		}

		// Marshal data and options back to JSON for the script
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return fmt.Sprintf(`<div class="%s contribution-graph-error">
  <p>Contribution Graph Error: Failed to serialize data</p>
  <pre>%s</pre>
</div>`, p.config.ContainerClass, html.EscapeString(err.Error()))
		}
		optionsJSON, err := json.Marshal(options)
		if err != nil {
			return fmt.Sprintf(`<div class="%s contribution-graph-error">
  <p>Contribution Graph Error: Failed to serialize options</p>
  <pre>%s</pre>
</div>`, p.config.ContainerClass, html.EscapeString(err.Error()))
		}

		// Create the initialization script for this graph
		initScript := fmt.Sprintf(`
  (function() {
    const cal = new CalHeatmap();
    cal.paint({
      itemSelector: '#%s',
      data: {
        source: %s,
        x: 'date',
        y: 'value'
      },
      %s
    });
  })();`, graphID, string(dataJSON), p.buildOptionsScript(optionsJSON))
		initScripts = append(initScripts, initScript)

		// Return the container div
		return fmt.Sprintf(`<div class="%s">
  <div id="%s"></div>
</div>`, p.config.ContainerClass, graphID)
	})

	// If we found contribution-graph blocks, inject the scripts
	if len(initScripts) > 0 {
		result = p.injectCalHeatmapScripts(result, initScripts)
	}

	post.ArticleHTML = result
	return nil
}

// buildOptionsScript converts options JSON into Cal-Heatmap configuration.
func (p *ContributionGraphPlugin) buildOptionsScript(optionsJSON []byte) string {
	var options map[string]interface{}
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		return ""
	}

	// Build configuration options string
	var configParts []string

	// Handle domain configuration
	if domain, ok := options["domain"].(string); ok {
		configParts = append(configParts, fmt.Sprintf(`domain: { type: '%s' }`, domain))
	} else {
		configParts = append(configParts, `domain: { type: 'year' }`)
	}

	// Handle subDomain configuration
	if subDomain, ok := options["subDomain"].(string); ok {
		configParts = append(configParts, fmt.Sprintf(`subDomain: { type: '%s' }`, subDomain))
	} else {
		configParts = append(configParts, `subDomain: { type: 'day' }`)
	}

	// Handle cellSize/width
	if cellSize, ok := options["cellSize"].(float64); ok {
		configParts = append(configParts, fmt.Sprintf(`subDomain: { type: 'day', width: %d, height: %d }`, int(cellSize), int(cellSize)))
	}

	// Handle range
	if rangeVal, ok := options["range"].(float64); ok {
		configParts = append(configParts, fmt.Sprintf(`range: %d`, int(rangeVal)))
	}

	// Handle theme
	if p.config.Theme != "" {
		configParts = append(configParts, fmt.Sprintf(`theme: '%s'`, p.config.Theme))
	}

	return strings.Join(configParts, ",\n      ")
}

// injectCalHeatmapScripts adds the Cal-Heatmap library and initialization scripts to the HTML.
func (p *ContributionGraphPlugin) injectCalHeatmapScripts(htmlContent string, initScripts []string) string {
	// Build the combined script
	script := fmt.Sprintf(`
<link rel="stylesheet" href="%s/cal-heatmap.css">
<script src="%s/cal-heatmap.min.js"></script>
<script>
document.addEventListener('DOMContentLoaded', function() {%s
});
</script>`, p.config.CDNURL, p.config.CDNURL, strings.Join(initScripts, ""))

	// Append the script to the end of the content
	return htmlContent + script
}

// SetConfig sets the contribution graph configuration directly.
// This is useful for testing or programmatic configuration.
func (p *ContributionGraphPlugin) SetConfig(config models.ContributionGraphConfig) {
	p.config = config
}

// Config returns the current contribution graph configuration.
func (p *ContributionGraphPlugin) Config() models.ContributionGraphConfig {
	return p.config
}

// Ensure ContributionGraphPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*ContributionGraphPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*ContributionGraphPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*ContributionGraphPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*ContributionGraphPlugin)(nil)
)
