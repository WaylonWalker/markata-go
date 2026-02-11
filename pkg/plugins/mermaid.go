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
		return strings.Contains(post.ArticleHTML, `class="language-mermaid"`) ||
			strings.Contains(post.ArticleHTML, `class="mermaid"`)
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
			// Only enable GLightbox loading; do NOT set glightbox_options.
			// The mermaid lightbox uses a separate programmatic GLightbox
			// instance (selector: false) and does not need the template's
			// shared instance. Overwriting glightbox_options would clobber
			// settings from image_zoom or other plugins.
			config.Extra["glightbox_enabled"] = true
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

	// Check if there are any mermaid code blocks (language-mermaid needs conversion)
	// or pre-rendered mermaid blocks (class="mermaid" just needs the script)
	hasLanguageMermaid := strings.Contains(post.ArticleHTML, `class="language-mermaid"`)
	hasPreRendered := strings.Contains(post.ArticleHTML, `class="mermaid"`)
	if !hasLanguageMermaid && !hasPreRendered {
		return nil
	}

	// Track if we found any mermaid blocks
	foundMermaid := false
	result := post.ArticleHTML

	// Replace language-mermaid code blocks with proper mermaid pre tags
	if hasLanguageMermaid {
		result = mermaidCodeBlockRegex.ReplaceAllStringFunc(result, func(match string) string {
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
	}

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
` + p.mermaidLightboxJS() + `
  mermaid.initialize({ startOnLoad: false, theme: '` + p.config.Theme + `' });
  window.initMermaid = async () => {
    try {
      await mermaid.run();
    } catch (e) {
      console.error('mermaid.run failed:', e);
    }
    ensureMermaidLightbox();
  };
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => window.initMermaid());
  } else {
    window.initMermaid();
  }
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
    * { cursor: pointer; }
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
` + p.mermaidLightboxJS() + `
  mermaid.initialize({ startOnLoad: false, theme: 'base', themeVariables, flowchart, themeCSS });
  window.initMermaid = async () => {
    try {
      await mermaid.run();
    } catch (e) {
      console.error('mermaid.run failed:', e);
    }
    ensureMermaidLightbox();
  };
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => window.initMermaid());
  } else {
    window.initMermaid();
  }
</script>`
}

// mermaidLightboxJS returns the shared JavaScript for mermaid diagram lightbox
// with svg-pan-zoom support. This is used by both the standard and CSS variables
// code paths.
func (p *MermaidPlugin) mermaidLightboxJS() string {
	return `
  const SVG_PAN_ZOOM_CDN = 'https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.2/dist/svg-pan-zoom.min.js';
  let mermaidLightbox = null;
  let activePanZoom = null;

  // Lazy-load svg-pan-zoom from CDN, returns a promise
  const loadSvgPanZoom = () => {
    if (typeof svgPanZoom !== 'undefined') return Promise.resolve();
    return new Promise((resolve, reject) => {
      const s = document.createElement('script');
      s.src = SVG_PAN_ZOOM_CDN;
      s.onload = resolve;
      s.onerror = reject;
      document.head.appendChild(s);
    });
  };

  // Initialize svg-pan-zoom on the SVG inside the lightbox.
  // Called after the container has its final layout dimensions.
  // Retries a few times because GLightbox animation or sequence diagram
  // layout may not be settled when slide_after_load fires.
  let _pzRetries = 0;
  const initPanZoom = () => {
    if (activePanZoom) return; // already initialized
    const container = document.querySelector('.glightbox-container .gslide.current .mermaid-lightbox-wrap');
    if (!container) return;
    const svgEl = container.querySelector('svg');
    if (!svgEl) return;

    // svg-pan-zoom needs a viewBox to calculate zoom/pan transforms.
    // Mermaid sets width/height attrs but not always a viewBox.
    if (!svgEl.getAttribute('viewBox')) {
      // Try width/height attributes first (flowcharts, pie, etc.)
      let w = parseFloat(svgEl.getAttribute('width'));
      let h = parseFloat(svgEl.getAttribute('height'));
      // Sequence diagrams: mermaid sets style="max-width: Npx;" with no width/height attrs.
      if (!w && svgEl.style.maxWidth) w = parseFloat(svgEl.style.maxWidth);
      // Last resort: bounding rect (needs layout to be settled)
      if (!w || !h) {
        const rect = svgEl.getBoundingClientRect();
        if (!w) w = rect.width;
        if (!h) h = rect.height;
      }
      if (w > 0 && h > 0) {
        svgEl.setAttribute('viewBox', '0 0 ' + w + ' ' + h);
      } else if (_pzRetries < 10) {
        // SVG has no dimensions yet (lightbox still animating) -- retry
        _pzRetries++;
        setTimeout(initPanZoom, 80);
        return;
      }
    }
    _pzRetries = 0;

    // Remove fixed width/height so SVG fills the container via CSS (100%)
    svgEl.removeAttribute('width');
    svgEl.removeAttribute('height');
    svgEl.removeAttribute('style');

    try {
      // Initialize WITHOUT fit/center -- container may still be animating.
      activePanZoom = svgPanZoom(svgEl, {
        zoomEnabled: true,
        panEnabled: true,
        controlIconsEnabled: false,
        fit: false,
        center: false,
        minZoom: 0.3,
        maxZoom: 10,
        zoomScaleSensitivity: 0.3,
        mouseWheelZoomEnabled: true,
        preventMouseEventsDefault: true,
      });
      // Force a resize + fit + center once the container has settled.
      // requestAnimationFrame ensures we run after the current paint.
      requestAnimationFrame(() => {
        if (!activePanZoom) return;
        activePanZoom.resize();
        activePanZoom.fit();
        activePanZoom.center();
      });
    } catch (_) {
      activePanZoom = null;
    }

    // Add reset/fit buttons
    let toolbar = container.querySelector('.mermaid-lightbox-toolbar');
    if (!toolbar) {
      toolbar = document.createElement('div');
      toolbar.className = 'mermaid-lightbox-toolbar';
      toolbar.innerHTML =
        '<button class="mermaid-pz-btn" data-action="fit" title="Fit to view">Fit</button>' +
        '<button class="mermaid-pz-btn" data-action="zoomin" title="Zoom in">+</button>' +
        '<button class="mermaid-pz-btn" data-action="zoomout" title="Zoom out">&minus;</button>';
      toolbar.addEventListener('click', (ev) => {
        const btn = ev.target.closest('[data-action]');
        if (!btn || !activePanZoom) return;
        ev.preventDefault();
        ev.stopPropagation();
        const action = btn.dataset.action;
        if (action === 'fit') { activePanZoom.resize(); activePanZoom.fit(); activePanZoom.center(); }
        else if (action === 'zoomin') { activePanZoom.zoomIn(); }
        else if (action === 'zoomout') { activePanZoom.zoomOut(); }
      });
      container.prepend(toolbar);
    }
  };

  // Destroy pan-zoom on lightbox close
  const destroyPanZoom = () => {
    if (activePanZoom) {
      try { activePanZoom.destroy(); } catch (_) { /* no-op */ }
      activePanZoom = null;
    }
  };

  let _lbRetries = 0;
  const ensureMermaidLightbox = () => {
    const diagrams = document.querySelectorAll('.mermaid svg');
    if (!diagrams.length) {
      // Mermaid ESM may still be rendering -- retry up to 2s
      if (_lbRetries < 20) { _lbRetries++; setTimeout(ensureMermaidLightbox, 100); }
      return;
    }
    _lbRetries = 0;
    diagrams.forEach((svg) => {
      if (svg.dataset.lightboxBound) return;
      svg.dataset.lightboxBound = 'true';
      svg.style.cursor = 'pointer';
      svg.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        const svgHtml = svg.outerHTML;
        const openLightbox = () => {
          if (!mermaidLightbox) {
            mermaidLightbox = GLightbox({
              selector: false,
              openEffect: 'fade',
              closeEffect: 'fade',
              zoomable: false,
              draggable: false,
            });
            mermaidLightbox.on('slide_after_load', () => {
              destroyPanZoom();
              _pzRetries = 0;
              loadSvgPanZoom().then(() => initPanZoom());
            });
            mermaidLightbox.on('close', destroyPanZoom);
          }
          mermaidLightbox.setElements([{
            content: '<div class="mermaid-lightbox-wrap">' + svgHtml + '</div>',
            width: '90vw',
            height: '90vh'
          }]);
          mermaidLightbox.open();
          // Pan-zoom init is handled by the slide_after_load event above.
          // Pre-load the script so it's ready when the event fires.
          loadSvgPanZoom();
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
`
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
