// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"html"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// YouTubePlugin converts YouTube URLs into embedded iframes.
// It runs at the render stage (after markdown conversion).
type YouTubePlugin struct {
	config models.YouTubeConfig
}

// NewYouTubePlugin creates a new YouTubePlugin with default settings.
func NewYouTubePlugin() *YouTubePlugin {
	return &YouTubePlugin{
		config: models.NewYouTubeConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *YouTubePlugin) Name() string {
	return "youtube"
}

// Priority returns the plugin's priority for a given stage.
// This plugin runs after render_markdown (which has default priority 0).
func (p *YouTubePlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after render_markdown
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "youtube" key.
func (p *YouTubePlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for youtube config in Extra
	pluginConfig, ok := config.Extra["youtube"]
	if !ok {
		return nil
	}

	// Handle map configuration
	if cfgMap, ok := pluginConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if privacyEnhanced, ok := cfgMap["privacy_enhanced"].(bool); ok {
			p.config.PrivacyEnhanced = privacyEnhanced
		}
		if containerClass, ok := cfgMap["container_class"].(string); ok && containerClass != "" {
			p.config.ContainerClass = containerClass
		}
		if lazyLoad, ok := cfgMap["lazy_load"].(bool); ok {
			p.config.LazyLoad = lazyLoad
		}
	}

	return nil
}

// Render processes YouTube URLs in the rendered HTML.
func (p *YouTubePlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	return m.ProcessPostsConcurrently(p.processPost)
}

// youtubeURLRegex matches YouTube URLs in various formats.
var youtubeURLRegex = regexp.MustCompile(
	`<p>\s*(https?://(?:www\.|m\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})(?:\S*)?)\s*</p>`,
)

// youtubeInCodeBlockRegex detects if a YouTube URL is inside a code block.
var youtubeInCodeBlockRegex = regexp.MustCompile(
	`<code[^>]*>.*?https?://(?:www\.|m\.)?(?:youtube\.com/watch\?v=|youtu\.be/)[a-zA-Z0-9_-]{11}.*?</code>`,
)

// processPost processes a single post's HTML for YouTube URLs.
func (p *YouTubePlugin) processPost(post *models.Post) error {
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	if !strings.Contains(post.ArticleHTML, "youtube.com") && !strings.Contains(post.ArticleHTML, "youtu.be") {
		return nil
	}

	result := youtubeURLRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		if youtubeInCodeBlockRegex.MatchString(post.ArticleHTML) {
			matchIdx := strings.Index(post.ArticleHTML, match)
			if matchIdx >= 0 {
				beforeMatch := post.ArticleHTML[:matchIdx]
				openCodes := strings.Count(beforeMatch, "<code")
				closeCodes := strings.Count(beforeMatch, "</code>")
				if openCodes > closeCodes {
					return match
				}
			}
		}

		submatches := youtubeURLRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		videoID := submatches[2]
		return p.buildEmbed(videoID)
	})

	post.ArticleHTML = result
	return nil
}

// buildEmbed creates the HTML for a YouTube embed.
func (p *YouTubePlugin) buildEmbed(videoID string) string {
	domain := "www.youtube.com"
	if p.config.PrivacyEnhanced {
		domain = "www.youtube-nocookie.com"
	}

	var sb strings.Builder
	sb.WriteString(`<div class="`)
	sb.WriteString(html.EscapeString(p.config.ContainerClass))
	sb.WriteString("\">\n")
	sb.WriteString(`  <iframe`)
	sb.WriteString("\n")
	sb.WriteString(`    src="https://`)
	sb.WriteString(domain)
	sb.WriteString(`/embed/`)
	sb.WriteString(html.EscapeString(videoID))
	sb.WriteString("\"\n")
	sb.WriteString(`    title="YouTube video player"`)
	sb.WriteString("\n")
	sb.WriteString(`    frameborder="0"`)
	sb.WriteString("\n")
	sb.WriteString(`    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"`)
	sb.WriteString("\n")
	sb.WriteString(`    allowfullscreen`)

	if p.config.LazyLoad {
		sb.WriteString("\n")
		sb.WriteString(`    loading="lazy"`)
	}

	sb.WriteString(">\n")
	sb.WriteString(`  </iframe>`)
	sb.WriteString("\n")
	sb.WriteString(`</div>`)

	return sb.String()
}

// SetConfig sets the plugin configuration directly.
func (p *YouTubePlugin) SetConfig(config models.YouTubeConfig) {
	p.config = config
}

// Config returns the current plugin configuration.
func (p *YouTubePlugin) Config() models.YouTubeConfig {
	return p.config
}

var (
	_ lifecycle.Plugin          = (*YouTubePlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*YouTubePlugin)(nil)
	_ lifecycle.RenderPlugin    = (*YouTubePlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*YouTubePlugin)(nil)
)
