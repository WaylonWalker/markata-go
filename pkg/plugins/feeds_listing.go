// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// FeedVariantLink describes one publicly accessible output for a feed.
type FeedVariantLink struct {
	Label string
	Href  string
	Kind  string
}

// FeedListingInfo contains the data rendered on the /feeds page.
type FeedListingInfo struct {
	Title       string
	Slug        string
	Description string
	Href        string
	PostCount   int
	LatestDate  string
	Variants    []FeedVariantLink
}

// FeedsListingPlugin generates a feeds listing page at /feeds.
type FeedsListingPlugin struct {
	engineMu    sync.RWMutex
	engineCache map[string]*templates.Engine
}

// NewFeedsListingPlugin creates a new FeedsListingPlugin.
func NewFeedsListingPlugin() *FeedsListingPlugin {
	return &FeedsListingPlugin{engineCache: make(map[string]*templates.Engine)}
}

// Name returns the unique name of the plugin.
func (p *FeedsListingPlugin) Name() string {
	return "feeds_listing"
}

// Priority returns the plugin's priority for a given stage.
func (p *FeedsListingPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLate
	}
	return lifecycle.PriorityDefault
}

// Write generates the feeds listing page.
func (p *FeedsListingPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	feedsPage := getFeedsPageConfig(config)
	if !feedsPage.IsEnabled() {
		return nil
	}

	feedConfigs := getCachedFeedConfigs(m)
	if len(feedConfigs) == 0 {
		return nil
	}

	feedInfos := p.collectFeedInfos(feedConfigs, config)
	if len(feedInfos) == 0 {
		return nil
	}

	sort.Slice(feedInfos, func(i, j int) bool {
		return feedInfos[i].Title < feedInfos[j].Title
	})

	return p.renderFeedsPage(config, &feedsPage, feedInfos)
}

func (p *FeedsListingPlugin) collectFeedInfos(feedConfigs []models.FeedConfig, config *lifecycle.Config) []FeedListingInfo {
	syndication := getSyndicationConfig(config)
	infos := make([]FeedListingInfo, 0, len(feedConfigs))

	for i := range feedConfigs {
		fc := &feedConfigs[i]
		if fc.IncludePrivate {
			continue
		}

		postCount, latestDate := publicFeedStats(fc.Posts)
		info := FeedListingInfo{
			Title:       feedDisplayTitle(fc),
			Slug:        fc.Slug,
			Description: fc.Description,
			Href:        feedHTMLHref(fc),
			PostCount:   postCount,
			LatestDate:  latestDate,
			Variants:    feedVariantLinks(fc, syndication),
		}
		infos = append(infos, info)
	}

	return infos
}

func publicFeedStats(posts []*models.Post) (int, string) {
	count := 0
	var latest time.Time
	for _, post := range posts {
		if post == nil || post.Private || post.Skip || post.Draft || !post.Published {
			continue
		}
		count++
		if post.Date != nil && post.Date.After(latest) {
			latest = *post.Date
		}
	}
	if latest.IsZero() {
		return count, ""
	}
	return count, latest.Format("2006-01-02")
}

func feedDisplayTitle(fc *models.FeedConfig) string {
	if fc.Title != "" {
		return fc.Title
	}
	if fc.Slug == "" {
		return "Home"
	}
	return fc.Slug
}

func feedHTMLHref(fc *models.FeedConfig) string {
	if fc.Slug == "" {
		return "/"
	}
	return "/" + fc.Slug + "/"
}

func feedVariantLinks(fc *models.FeedConfig, syndication models.SyndicationConfig) []FeedVariantLink {
	variants := make([]FeedVariantLink, 0, 10)
	baseHref := feedHTMLHref(fc)

	if fc.Formats.HTML {
		variants = append(variants, FeedVariantLink{Label: "HTML", Href: baseHref, Kind: "page"})
	}
	if fc.Formats.SimpleHTML {
		variants = append(variants, FeedVariantLink{Label: "Simple", Href: pathJoinURL(baseHref, "simple/"), Kind: "page"})
	}
	if fc.Formats.RSS {
		variants = append(variants, FeedVariantLink{Label: "RSS", Href: pathJoinURL(baseHref, "rss.xml"), Kind: "feed"})
	}
	if fc.Formats.Atom {
		variants = append(variants, FeedVariantLink{Label: "Atom", Href: pathJoinURL(baseHref, "atom.xml"), Kind: "feed"})
	}
	if fc.Formats.JSON {
		variants = append(variants, FeedVariantLink{Label: "JSON", Href: pathJoinURL(baseHref, "feed.json"), Kind: "feed"})
	}
	if fc.Formats.Markdown && fc.Slug != "" {
		variants = append(variants, FeedVariantLink{Label: "Markdown", Href: "/" + fc.Slug + ".md", Kind: "export"})
	}
	if fc.Formats.Text && fc.Slug != "" {
		variants = append(variants, FeedVariantLink{Label: "Text", Href: "/" + fc.Slug + ".txt", Kind: "export"})
	}
	if fc.Formats.Sitemap {
		variants = append(variants, FeedVariantLink{Label: "Sitemap", Href: pathJoinURL(baseHref, "sitemap.xml"), Kind: "meta"})
	}
	if shouldGenerateFeedArchive(fc, syndication) {
		if fc.Formats.RSS {
			variants = append(variants, FeedVariantLink{Label: "Archive RSS", Href: feedArchiveURL(fc.Slug, "rss.xml"), Kind: "archive"})
		}
		if fc.Formats.Atom {
			variants = append(variants, FeedVariantLink{Label: "Archive Atom", Href: feedArchiveURL(fc.Slug, "atom.xml"), Kind: "archive"})
		}
		if fc.Formats.JSON {
			variants = append(variants, FeedVariantLink{Label: "Archive JSON", Href: feedArchiveURL(fc.Slug, "feed.json"), Kind: "archive"})
		}
	}

	return variants
}

func pathJoinURL(base, suffix string) string {
	if base == "/" {
		return "/" + suffix
	}
	return base + suffix
}

func (p *FeedsListingPlugin) renderFeedsPage(config *lifecycle.Config, feedsPage *models.FeedsPageConfig, feedInfos []FeedListingInfo) error {
	feedsDir := filepath.Join(config.OutputDir, feedsPage.SlugPrefix)
	if err := os.MkdirAll(feedsDir, 0o755); err != nil {
		return fmt.Errorf("creating feeds directory: %w", err)
	}

	engine, err := p.createTemplateEngine(config)
	if err != nil {
		return err
	}

	if !engine.TemplateExists(feedsPage.Template) {
		log.Printf("[feeds_listing] Warning: template %q not found, skipping feeds listing page", feedsPage.Template)
		return nil
	}

	modelsConfig := ToModelsConfig(config)
	title := feedsPage.Title
	description := feedsPage.Description
	syntheticPost := &models.Post{
		Slug:        feedsPage.SlugPrefix,
		Title:       &title,
		Description: &description,
	}

	ctx := templates.NewContext(syntheticPost, "", modelsConfig)
	ctx.Extra["feed_list"] = feedInfos
	ctx.Extra["total_feeds"] = len(feedInfos)

	html, err := engine.Render(feedsPage.Template, ctx)
	if err != nil {
		return fmt.Errorf("rendering feeds template: %w", err)
	}

	outputPath := filepath.Join(feedsDir, "index.html")
	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil { //nolint:gosec // web output must be world-readable
		return fmt.Errorf("writing feeds listing page: %w", err)
	}

	log.Printf("[feeds_listing] Generated /%s/ with %d feeds", feedsPage.SlugPrefix, len(feedInfos))
	return nil
}

func (p *FeedsListingPlugin) createTemplateEngine(config *lifecycle.Config) (*templates.Engine, error) {
	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	themeName := getThemeName(config)
	cacheKey := templatesDir + ":" + themeName

	p.engineMu.RLock()
	if engine, ok := p.engineCache[cacheKey]; ok {
		p.engineMu.RUnlock()
		return engine, nil
	}
	p.engineMu.RUnlock()

	p.engineMu.Lock()
	defer p.engineMu.Unlock()
	if engine, ok := p.engineCache[cacheKey]; ok {
		return engine, nil
	}

	engine, err := templates.NewEngineWithTheme(templatesDir, themeName)
	if err != nil {
		return nil, err
	}
	p.engineCache[cacheKey] = engine
	return engine, nil
}

var (
	_ lifecycle.Plugin         = (*FeedsListingPlugin)(nil)
	_ lifecycle.WritePlugin    = (*FeedsListingPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*FeedsListingPlugin)(nil)
)
