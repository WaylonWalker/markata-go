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
		if useCSSVariables, ok := cfgMap["use_css_variables"].(bool); ok {
			p.config.UseCSSVariables = useCSSVariables
		}
		if lightbox, ok := cfgMap["lightbox"].(bool); ok {
			p.config.Lightbox = lightbox
		}
		if selector, ok := cfgMap["lightbox_selector"].(string); ok && selector != "" {
			p.config.LightboxSelector = selector
		}
	}

	return nil
}

// Render processes mermaid code blocks in the rendered HTML for all posts.
func (p *MermaidPlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	posts := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.ArticleHTML == "" {
			return false
		}
		return strings.Contains(post.ArticleHTML, `class="language-mermaid"`)
	})

	if err := m.ProcessPostsSliceConcurrently(posts, p.processPost); err != nil {
		return err
	}

	if p.config.Lightbox {
		needsLightbox := false
		for _, post := range m.Posts() {
			if post.ArticleHTML == "" {
				continue
			}
			if strings.Contains(post.ArticleHTML, `class="mermaid"`) || strings.Contains(post.ArticleHTML, `class="language-mermaid"`) {
				needsLightbox = true
				break
			}
		}

		if needsLightbox {
			config := m.Config()
			if config.Extra == nil {
				config.Extra = make(map[string]interface{})
			}
			config.Extra["glightbox_enabled"] = true
			config.Extra["glightbox_options"] = map[string]interface{}{
				"selector": p.config.LightboxSelector,
			}
			config.Extra["glightbox_cdn"] = true
		}
	}

	return nil
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

	// If we found mermaid blocks or existing mermaid blocks, inject the script
	if foundMermaid || strings.Contains(result, `class="mermaid"`) {
		result = p.injectMermaidScript(result)
	}

	post.ArticleHTML = result
	return nil
}

// injectMermaidScript adds the Mermaid.js initialization script to the HTML.
// The script is only injected once per post.
func (p *MermaidPlugin) injectMermaidScript(htmlContent string) string {
	var script string
	if p.config.UseCSSVariables {
		script = p.cssVariablesScript()
	} else {
		script = `
<script type="module">
  import mermaid from '` + p.config.CDNURL + `';
  let mermaidLightbox = null;
  const ensureMermaidLightbox = () => {
    const diagrams = document.querySelectorAll('.mermaid svg');
    if (!diagrams.length) return;
    diagrams.forEach((svg) => {
      if (svg.dataset.lightboxBound) return;
      svg.dataset.lightboxBound = 'true';
      svg.style.cursor = 'zoom-in';
      svg.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        const svgHtml = svg.outerHTML;
        const openLightbox = () => {
          if (!mermaidLightbox) {
            mermaidLightbox = GLightbox({ selector: false, openEffect: 'fade', closeEffect: 'fade', zoomable: false });
          }
          mermaidLightbox.setElements([{
            content: '<div class="mermaid-lightbox-wrap">' + svgHtml + '</div>',
            width: '90vw',
            height: '90vh'
          }]);
          mermaidLightbox.open();
        };
        if (typeof GLightbox !== 'undefined') {
          openLightbox();
        } else if (window.initGLightbox) {
          window.initGLightbox();
          openLightbox();
        } else {
          window.addEventListener('glightbox-ready', () => { openLightbox(); }, { once: true });
        }
      });
    });
  };
  mermaid.initialize({ startOnLoad: true, theme: '` + p.config.Theme + `' });
  window.initMermaid = () => {
    try {
      mermaid.run();
      setTimeout(ensureMermaidLightbox, 100);
    } catch (_) {
      // no-op
    }
  };
  setTimeout(window.initMermaid, 0);
</script>`
	}

	// Append the script to the end of the content
	return htmlContent + script
}

func (p *MermaidPlugin) cssVariablesScript() string {
	return `
<script type="module">
  import mermaid from '` + p.config.CDNURL + `';
  const rootStyle = getComputedStyle(document.documentElement);
  const css = (name, fallback) => (rootStyle.getPropertyValue(name) || fallback).trim();
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches ||
    document.documentElement.dataset.theme === 'dark';
  const accent = css('--color-primary', '#ffcd11');
  const flowchart = {
    nodeSpacing: 60,
    rankSpacing: 90,
    padding: 12,
  };
  const themeCSS = ` + "`" + `
    .label foreignObject > div { padding: 14px 14px 10px; line-height: 1.2; }
    .nodeLabel { padding: 14px 14px 10px; line-height: 1.2; }
    .mermaid svg { cursor: zoom-in; }
  ` + "`" + `;
  const themeVariables = {
    background: css('--color-background', '#ffffff'),
    primaryColor: css('--color-code-bg', '#0a0a0a'),
    primaryTextColor: css('--color-text', '#1f2937'),
    primaryBorderColor: accent,
    lineColor: accent,
    textColor: css('--color-text', '#1f2937'),
    nodeBkg: css('--color-code-bg', '#0a0a0a'),
    nodeBorder: accent,
    nodeTextColor: css('--color-text', '#1f2937'),
    fontSize: '16px',
    nodePadding: 20,
    nodeTextMargin: 14,
    clusterBkg: isDark ? css('--color-background', '#0f0f0f') : css('--color-surface', '#f9fafb'),
    clusterBorder: accent,
    clusterTextColor: css('--color-text', '#1f2937'),
    titleColor: css('--color-text', '#1f2937'),
    edgeLabelBackground: css('--color-code-bg', '#0a0a0a'),
  };
  let mermaidLightbox = null;
  const ensureMermaidLightbox = () => {
    const diagrams = document.querySelectorAll('.mermaid svg');
    if (!diagrams.length) return;
    diagrams.forEach((svg) => {
      if (svg.dataset.lightboxBound) return;
      svg.dataset.lightboxBound = 'true';
      svg.style.cursor = 'zoom-in';
      svg.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        const svgHtml = svg.outerHTML;
        const openLightbox = () => {
          if (!mermaidLightbox) {
            mermaidLightbox = GLightbox({ selector: false, openEffect: 'fade', closeEffect: 'fade', zoomable: false });
          }
          mermaidLightbox.setElements([{
            content: '<div class="mermaid-lightbox-wrap">' + svgHtml + '</div>',
            width: '90vw',
            height: '90vh'
          }]);
          mermaidLightbox.open();
        };
        if (typeof GLightbox !== 'undefined') {
          openLightbox();
        } else if (window.initGLightbox) {
          window.initGLightbox();
          openLightbox();
        } else {
          window.addEventListener('glightbox-ready', () => { openLightbox(); }, { once: true });
        }
      });
    });
  };
  mermaid.initialize({ startOnLoad: true, theme: 'base', themeVariables, flowchart, themeCSS });
  window.initMermaid = () => {
    try {
      mermaid.run();
      setTimeout(ensureMermaidLightbox, 100);
    } catch (_) {
      // no-op
    }
  };
  setTimeout(window.initMermaid, 0);
</script>`
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
