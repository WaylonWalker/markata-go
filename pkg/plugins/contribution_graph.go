// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
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
		optionsMap, _ := options.(map[string]interface{})
		if optionsMap == nil {
			optionsMap = map[string]interface{}{}
		}

		// Auto-detect year from first data point if not specified
		if _, hasYear := optionsMap["year"]; !hasYear {
			if dataSlice, ok := data.([]interface{}); ok && len(dataSlice) > 0 {
				if firstItem, ok := dataSlice[0].(map[string]interface{}); ok {
					if dateStr, ok := firstItem["date"].(string); ok && len(dateStr) >= 4 {
						// Extract year from date string (YYYY-MM-DD format)
						if year, err := strconv.Atoi(dateStr[:4]); err == nil {
							optionsMap["year"] = float64(year)
						}
					}
				}
			}
		}

		// Marshal data and options back to JSON for the script
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return fmt.Sprintf(`<div class="%s contribution-graph-error">
  <p>Contribution Graph Error: Failed to serialize data</p>
  <pre>%s</pre>
</div>`, p.config.ContainerClass, html.EscapeString(err.Error()))
		}
		optionsJSON, err := json.Marshal(optionsMap)
		if err != nil {
			return fmt.Sprintf(`<div class="%s contribution-graph-error">
  <p>Contribution Graph Error: Failed to serialize options</p>
  <pre>%s</pre>
</div>`, p.config.ContainerClass, html.EscapeString(err.Error()))
		}

		// Create the initialization script for this graph
		// Uses Cal-Heatmap Tooltip plugin for hover information
		// Stores paint function for theme change re-rendering
		initScript := fmt.Sprintf(`
  (function() {
    const graphId = '%s';
    const data = %s;
    const options = {%s};
    
    function paintGraph() {
      // Clear existing graph
      const container = document.getElementById(graphId);
      if (!container) return;
      container.innerHTML = '';
      
      // Calculate max value for this graph's scale
      const maxValue = Math.max(1, ...data.map(d => d.value || 0));
      
      // Get theme colors from CSS variables
      const styles = getComputedStyle(document.documentElement);
      const bgColor = styles.getPropertyValue('--color-background').trim();
      const surfaceColor = styles.getPropertyValue('--color-surface').trim();
      const primaryColor = styles.getPropertyValue('--color-primary').trim();
      
      // Use surface color as base, primary as accent
      const baseColor = surfaceColor || bgColor || '#ebedf0';
      const accentColor = primaryColor || '#216e39';
      
      const cal = new CalHeatmap();
      cal.paint(
        {
          itemSelector: '#' + graphId,
          data: {
            source: data,
            x: 'date',
            y: 'value'
          },
          date: options.date,
          domain: options.domain || { type: 'year' },
          subDomain: options.subDomain || { type: 'day' },
          range: options.range,
          scale: {
            color: {
              type: 'linear',
              range: [baseColor, accentColor],
              domain: [0, maxValue]
            }
          }
        },
        [
          [
            Tooltip,
            {
              text: function (date, value, dayjsDate) {
                return (value ? value : 'No') + ' posts on ' + dayjsDate.format('MMM D, YYYY');
              },
            },
          ],
        ]
      );
    }
    
    // Initial paint
    paintGraph();
    
    // Register for theme changes
    if (!window._contributionGraphPainters) {
      window._contributionGraphPainters = [];
    }
    window._contributionGraphPainters.push(paintGraph);
  })();`, graphID, string(dataJSON), p.buildOptionsObject(optionsJSON))
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

	// Handle date.start configuration from "year" option
	// This is crucial for showing historical data
	if year, ok := options["year"].(float64); ok {
		configParts = append(configParts, fmt.Sprintf(`date: { start: new Date('%d-01-01') }`, int(year)))
	}

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

	// Color scale is set dynamically in the init script using getComputedStyle
	// to resolve CSS variables at runtime

	result := strings.Join(configParts, ",\n      ")
	if result != "" {
		result += ","
	}
	return result
}

// buildOptionsObject converts options JSON into a JavaScript object literal for Cal-Heatmap.
// This is used by the reactive theme system to store options that can be re-applied.
func (p *ContributionGraphPlugin) buildOptionsObject(optionsJSON []byte) string {
	var options map[string]interface{}
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		return ""
	}

	var parts []string

	// Handle date.start configuration from "year" option
	if year, ok := options["year"].(float64); ok {
		parts = append(parts, fmt.Sprintf(`date: { start: new Date('%d-01-01') }`, int(year)))
	}

	// Handle domain configuration
	if domain, ok := options["domain"].(string); ok {
		parts = append(parts, fmt.Sprintf(`domain: { type: '%s' }`, domain))
	}

	// Handle subDomain configuration
	if subDomain, ok := options["subDomain"].(string); ok {
		parts = append(parts, fmt.Sprintf(`subDomain: { type: '%s' }`, subDomain))
	}

	// Handle range
	if rangeVal, ok := options["range"].(float64); ok {
		parts = append(parts, fmt.Sprintf(`range: %d`, int(rangeVal)))
	}

	return strings.Join(parts, ", ")
}

// injectCalHeatmapScripts adds the Cal-Heatmap library and initialization scripts to the HTML.
func (p *ContributionGraphPlugin) injectCalHeatmapScripts(htmlContent string, initScripts []string) string {
	// Build the combined script
	// Cal-Heatmap v4 requires d3 as a dependency
	// Tooltip plugin requires popper.js
	script := fmt.Sprintf(`
<style>
.contribution-graph-container {
  width: 100%%;
  overflow-x: auto;
  margin: 1rem 0;
  display: flex;
  justify-content: center;
}
.contribution-graph-container > div {
  flex-shrink: 0;
}
#ch-tooltip {
  background: var(--color-surface, #333);
  color: var(--color-text, #fff);
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  font-size: 0.875rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  z-index: 10000 !important;
}
</style>
<link rel="stylesheet" href="%s/cal-heatmap.css">
<script src="https://d3js.org/d3.v7.min.js"></script>
<script src="https://unpkg.com/@popperjs/core@2"></script>
<script src="%s/cal-heatmap.min.js"></script>
<script src="%s/plugins/Tooltip.min.js"></script>
<script>
document.addEventListener('DOMContentLoaded', function() {
  // Initialize graphs
  %s
  
  // Watch for theme/palette changes and re-paint graphs
  const observer = new MutationObserver(function(mutations) {
    mutations.forEach(function(mutation) {
      if (mutation.attributeName === 'data-palette' || mutation.attributeName === 'class') {
        // Small delay to let CSS variables update
        setTimeout(function() {
          if (window._contributionGraphPainters) {
            window._contributionGraphPainters.forEach(function(paint) {
              paint();
            });
          }
        }, 50);
      }
    });
  });
  
  observer.observe(document.documentElement, { attributes: true });
  observer.observe(document.body, { attributes: true });
});
</script>`, p.config.CDNURL, p.config.CDNURL, p.config.CDNURL, strings.Join(initScripts, "\n"))

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
