// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

const pinsSlug = "pins"

// PinInfo contains the public metadata rendered for a link post on /pins/.
type PinInfo struct {
	Title       string
	Destination string
	Domain      string
	Href        string
	Description string
	Image       string
	Date        string
}

// Write generates the built-in visual link-post feed at /pins/. AutoFeedsPlugin
// owns other automatically generated collection pages, so keeping pins here
// avoids adding a second collection lifecycle while still making the page
// available without configuration.
func (p *AutoFeedsPlugin) Write(m *lifecycle.Manager) error {
	pins := collectPins(m.Posts())
	if len(pins) == 0 {
		return nil
	}

	config := m.Config()
	if config == nil {
		return nil
	}

	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	pinsDir := filepath.Join(outputDir, pinsSlug)
	if err := os.MkdirAll(pinsDir, 0o755); err != nil {
		return fmt.Errorf("creating pins directory: %w", err)
	}

	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}
	engine, err := templates.NewEngineWithTheme(templatesDir, getThemeName(config))
	if err != nil {
		return fmt.Errorf("creating pins template engine: %w", err)
	}
	if !engine.TemplateExists("pins.html") {
		log.Printf("[pins] warning: template %q not found, skipping pins page", "pins.html")
		return nil
	}

	title := "Pins"
	description := "Links worth keeping, collected from link posts."
	syntheticPost := &models.Post{
		Slug:        pinsSlug,
		Href:        "/" + pinsSlug + "/",
		Title:       &title,
		Description: &description,
		Published:   true,
	}
	ctx := templates.NewContext(syntheticPost, "", ToModelsConfig(config))
	ctx.Extra["pins"] = pins
	ctx.Extra["total_pins"] = len(pins)

	html, err := engine.Render("pins.html", ctx)
	if err != nil {
		return fmt.Errorf("rendering pins template: %w", err)
	}

	outputPath := filepath.Join(pinsDir, "index.html")
	//nolint:gosec // generated site output must be readable by the web server.
	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("writing pins page: %w", err)
	}

	log.Printf("[pins] Generated /pins/ with %d links", len(pins))
	return nil
}

func collectPins(posts []*models.Post) []PinInfo {
	type datedPin struct {
		pin  PinInfo
		post *models.Post
	}

	collected := make([]datedPin, 0)
	for _, post := range posts {
		if post == nil || !post.Published || post.Draft || post.Skip || post.Private {
			continue
		}

		destination := strings.TrimSpace(getPostExtraString(post, "link"))
		if destination == "" {
			continue
		}

		title := strings.TrimSpace(post.PlainTitle())
		if title == "" {
			title = post.Slug
		}
		description := ""
		if post.Description != nil {
			description = strings.TrimSpace(*post.Description)
		}

		pin := PinInfo{
			Title:       title,
			Destination: destination,
			Domain:      pinDomain(destination),
			Href:        post.Href,
			Description: description,
			Image:       getPostExtraString(post, "image", "cover", "cover_image", "featured_image", "thumbnail", "og_image", "social_image", "hero_image"),
		}
		if pin.Href == "" && post.Slug != "" {
			pin.Href = "/" + strings.Trim(post.Slug, "/") + "/"
		}
		if post.Date != nil {
			pin.Date = post.Date.Format("Jan 2, 2006")
		}
		collected = append(collected, datedPin{pin: pin, post: post})
	}

	sort.SliceStable(collected, func(i, j int) bool {
		left, right := collected[i].post.Date, collected[j].post.Date
		if left == nil && right == nil {
			return strings.ToLower(collected[i].pin.Title) < strings.ToLower(collected[j].pin.Title)
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return strings.ToLower(collected[i].pin.Title) < strings.ToLower(collected[j].pin.Title)
		}
		return left.After(*right)
	})

	pins := make([]PinInfo, 0, len(collected))
	for _, item := range collected {
		pins = append(pins, item.pin)
	}
	return pins
}

func pinDomain(rawURL string) string {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	if host != "" {
		return host
	}
	return rawURL
}

var _ lifecycle.WritePlugin = (*AutoFeedsPlugin)(nil)
