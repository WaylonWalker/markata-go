// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// ImageZoomPlugin adds optional image zoom/lightbox functionality using GLightbox.
// It runs at the render stage (post_render, after markdown conversion) to add
// data-zoomable attributes to images, and at the write stage to inject the
// required JavaScript and CSS.
type ImageZoomPlugin struct {
	config models.ImageZoomConfig
}

// NewImageZoomPlugin creates a new ImageZoomPlugin with default settings.
func NewImageZoomPlugin() *ImageZoomPlugin {
	return &ImageZoomPlugin{
		config: models.NewImageZoomConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *ImageZoomPlugin) Name() string {
	return "image_zoom"
}

// Priority returns the plugin's priority for a given stage.
// This plugin runs after render_markdown (which has default priority 0) but
// BEFORE templates (which uses PriorityLate=100) so that glightbox_enabled
// is set in config.Extra before templates renders the HTML.
func (p *ImageZoomPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return 50 // After render_markdown (0), before templates (100)
	}
	return lifecycle.PriorityDefault
}

// Configure reads configuration options for the plugin from config.Extra.
// Configuration is expected under the "image_zoom" key.
func (p *ImageZoomPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	// Check for image_zoom config in Extra
	imageZoomConfig, ok := config.Extra["image_zoom"]
	if !ok {
		return nil
	}

	// Handle map configuration
	if cfgMap, ok := imageZoomConfig.(map[string]interface{}); ok {
		if enabled, ok := cfgMap["enabled"].(bool); ok {
			p.config.Enabled = enabled
		}
		if library, ok := cfgMap["library"].(string); ok && library != "" {
			p.config.Library = library
		}
		if selector, ok := cfgMap["selector"].(string); ok && selector != "" {
			p.config.Selector = selector
		}
		if cdn, ok := cfgMap["cdn"].(bool); ok {
			p.config.CDN = cdn
		}
		if autoAllImages, ok := cfgMap["auto_all_images"].(bool); ok {
			p.config.AutoAllImages = autoAllImages
		}
		if openEffect, ok := cfgMap["open_effect"].(string); ok && openEffect != "" {
			p.config.OpenEffect = openEffect
		}
		if closeEffect, ok := cfgMap["close_effect"].(string); ok && closeEffect != "" {
			p.config.CloseEffect = closeEffect
		}
		if slideEffect, ok := cfgMap["slide_effect"].(string); ok && slideEffect != "" {
			p.config.SlideEffect = slideEffect
		}
		if touchNavigation, ok := cfgMap["touch_navigation"].(bool); ok {
			p.config.TouchNavigation = touchNavigation
		}
		if loop, ok := cfgMap["loop"].(bool); ok {
			p.config.Loop = loop
		}
		if draggable, ok := cfgMap["draggable"].(bool); ok {
			p.config.Draggable = draggable
		}
	}

	return nil
}

// Render processes images in the rendered HTML for all posts.
// It adds data-glightbox attributes to images that should be zoomable.
func (p *ImageZoomPlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	posts := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.ArticleHTML == "" {
			return false
		}
		return strings.Contains(post.ArticleHTML, "<img")
	})

	// Process posts
	if err := m.ProcessPostsSliceConcurrently(posts, p.processPost); err != nil {
		return err
	}

	// After processing all posts, check if any need image zoom and set config
	needsZoom := false
	for _, post := range m.Posts() {
		if post.Extra != nil {
			if needs, ok := post.Extra["needs_image_zoom"].(bool); ok && needs {
				needsZoom = true
				break
			}
		}
	}

	if needsZoom {
		// Set GLightbox config so templates can include the CSS/JS
		config := m.Config()
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}

		// Build the GLightbox initialization options
		glightboxOptions := map[string]interface{}{
			"selector":        p.config.Selector,
			"openEffect":      p.config.OpenEffect,
			"closeEffect":     p.config.CloseEffect,
			"slideEffect":     p.config.SlideEffect,
			"touchNavigation": p.config.TouchNavigation,
			"loop":            p.config.Loop,
			"draggable":       p.config.Draggable,
		}

		config.Extra["glightbox_options"] = glightboxOptions
		config.Extra["glightbox_enabled"] = true
		config.Extra["glightbox_cdn"] = p.config.CDN
	}

	return nil
}

// imageZoomImgTagRegex matches <img> tags and captures their attributes.
var imageZoomImgTagRegex = regexp.MustCompile(`<img\s+([^>]*)>`)

// dataZoomableRegex matches the {data-zoomable} attribute marker in alt text or title.
var dataZoomableRegex = regexp.MustCompile(`\{data-zoomable\}`)

// zoomableClassRegex matches the {.zoomable} class marker in alt text or title.
var zoomableClassRegex = regexp.MustCompile(`\{\.zoomable\}`)

// imgSrcRegex extracts the src attribute from an img tag.
var imgSrcRegex = regexp.MustCompile(`src="([^"]+)"`)

// imgAltRegex extracts the alt attribute from an img tag.
var imgAltRegex = regexp.MustCompile(`alt="([^"]*)"`)

// imgClassRegex matches the class attribute in an img tag for replacement.
var imgClassRegex = regexp.MustCompile(`class="([^"]*)"`)

// processPost processes a single post's HTML for images that should be zoomable.
func (p *ImageZoomPlugin) processPost(post *models.Post) error {
	// Skip posts marked as skip or with no HTML content
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	// Check frontmatter for image_zoom setting
	postZoomEnabled := p.config.AutoAllImages
	if post.Extra != nil {
		if imgZoom, ok := post.Extra["image_zoom"]; ok {
			if enabled, ok := imgZoom.(bool); ok {
				postZoomEnabled = enabled
			}
		}
	}

	// Track if we found any zoomable images
	foundZoomable := false

	// Process all img tags
	result := imageZoomImgTagRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		// Extract the attributes
		submatches := imageZoomImgTagRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		attrs := submatches[1]

		// Check if already has glightbox attribute
		if strings.Contains(attrs, "data-glightbox") {
			foundZoomable = true
			return match
		}

		// Check for {data-zoomable} or {.zoomable} markers in alt text
		hasZoomableMarker := dataZoomableRegex.MatchString(attrs) || zoomableClassRegex.MatchString(attrs)

		// Determine if this image should be zoomable
		shouldZoom := hasZoomableMarker || postZoomEnabled

		if !shouldZoom {
			return match
		}

		foundZoomable = true

		// Clean up the markers from alt text if present
		cleanedAttrs := dataZoomableRegex.ReplaceAllString(attrs, "")
		cleanedAttrs = zoomableClassRegex.ReplaceAllString(cleanedAttrs, "")
		cleanedAttrs = strings.TrimSpace(cleanedAttrs)

		// Extract src and alt for the glightbox data attribute
		srcMatch := imgSrcRegex.FindStringSubmatch(cleanedAttrs)
		altMatch := imgAltRegex.FindStringSubmatch(cleanedAttrs)

		src := ""
		alt := ""
		if len(srcMatch) > 1 {
			src = srcMatch[1]
		}
		if len(altMatch) > 1 {
			alt = strings.TrimSpace(altMatch[1])
		}

		// Build the glightbox data attribute
		glightboxAttr := fmt.Sprintf(`data-glightbox="description: %s"`, alt)

		// Add the gallery class and data attribute
		if strings.Contains(cleanedAttrs, `class="`) {
			// Append to existing class
			cleanedAttrs = imgClassRegex.ReplaceAllString(
				cleanedAttrs,
				`class="$1 glightbox"`,
			)
		} else {
			// Add new class attribute
			cleanedAttrs = `class="glightbox" ` + cleanedAttrs
		}

		// Add the data-glightbox attribute
		cleanedAttrs = cleanedAttrs + " " + glightboxAttr

		// Wrap image in anchor for lightbox functionality
		return `<a href="` + src + `" class="glightbox-link"><img ` + cleanedAttrs + `></a>`
	})

	// Store whether this post needs the glightbox library
	if foundZoomable {
		if post.Extra == nil {
			post.Extra = make(map[string]interface{})
		}
		post.Extra["needs_image_zoom"] = true
	}

	post.ArticleHTML = result
	return nil
}

// Write injects GLightbox CSS and JS into posts that need it.
func (p *ImageZoomPlugin) Write(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	// Check if any posts need image zoom
	needsZoom := false
	for _, post := range m.Posts() {
		if post.Extra != nil {
			if needs, ok := post.Extra["needs_image_zoom"].(bool); ok && needs {
				needsZoom = true
				break
			}
		}
	}

	if !needsZoom {
		return nil
	}

	// Store the GLightbox configuration for templates to use
	config := m.Config()
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}

	// Build the GLightbox initialization options
	glightboxOptions := map[string]interface{}{
		"selector":        p.config.Selector,
		"openEffect":      p.config.OpenEffect,
		"closeEffect":     p.config.CloseEffect,
		"slideEffect":     p.config.SlideEffect,
		"touchNavigation": p.config.TouchNavigation,
		"loop":            p.config.Loop,
		"draggable":       p.config.Draggable,
	}

	config.Extra["glightbox_options"] = glightboxOptions
	config.Extra["glightbox_enabled"] = true
	config.Extra["glightbox_cdn"] = p.config.CDN

	return nil
}

// SetConfig sets the image zoom configuration directly.
// This is useful for testing or programmatic configuration.
func (p *ImageZoomPlugin) SetConfig(config models.ImageZoomConfig) {
	p.config = config
}

// Config returns the current image zoom configuration.
func (p *ImageZoomPlugin) Config() models.ImageZoomConfig {
	return p.config
}

// Ensure ImageZoomPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*ImageZoomPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*ImageZoomPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*ImageZoomPlugin)(nil)
	_ lifecycle.WritePlugin     = (*ImageZoomPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*ImageZoomPlugin)(nil)
)
