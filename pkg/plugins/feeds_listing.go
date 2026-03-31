// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

const homeFeedTitle = "Home"

const (
	feedVariantPage    = "page"
	feedVariantExport  = "export"
	feedVariantFeed    = DefaultFeedPath
	feedVariantArchive = defaultArchivePrefix
)

// FeedVariantLink describes one publicly accessible output for a feed.
type FeedVariantLink struct {
	Label string
	Href  string
	Kind  string
}

type FeedListingSection struct {
	ID          string
	Title       string
	Description string
	TotalCount  int
	MoreHref    string
	MoreLabel   string
	Pagination  *FeedListingPage
	Feeds       []FeedListingInfo
}

type FeedListingPage struct {
	Number       int
	Feeds        []FeedListingInfo
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
	TotalPages   int
	TotalItems   int
	ItemsPerPage int
	PageURLs     []string
}

// FeedListingInfo contains the data rendered on the /feeds page.
type FeedListingInfo struct {
	Title           string
	Slug            string
	Description     string
	Href            string
	PostCount       int
	LatestPostDate  string
	SubscribeCount  int
	ArchiveCount    int
	DisplayVariants []FeedVariantLink
	PrimaryVariants []FeedVariantLink
	ArchiveVariants []FeedVariantLink
	UtilityVariants []FeedVariantLink
	LatestPostTime  time.Time
	SparklinePoints string
	SparklineTitle  string
	GeneratedBySite bool
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
	feedDefaults := getFeedDefaults(config)

	sections, generatedFeedPages := p.collectFeedSections(feedConfigs, config, &feedsPage, feedDefaults)
	if len(sections) == 0 {
		return nil
	}

	if err := p.renderFeedsPage(config, &feedsPage, sections, feedsPage.SlugPrefix, feedsPage.Title, feedsPage.Description, nil, nil); err != nil {
		return err
	}

	if len(generatedFeedPages) > 0 {
		generatedTitle := "Generated Feeds"
		generatedDescription := "Automatically updated feeds for broader site sections, archives, and collections."
		for i := range generatedFeedPages {
			pageSlug := filepath.ToSlash(filepath.Join(feedsPage.SlugPrefix, "generated"))
			if generatedFeedPages[i].Number > 1 {
				pageSlug = filepath.ToSlash(filepath.Join(pageSlug, "page", fmt.Sprintf("%d", generatedFeedPages[i].Number)))
			}
			section := FeedListingSection{
				ID:          "generated-feeds",
				Title:       generatedTitle,
				Description: generatedDescription,
				TotalCount:  generatedFeedPages[i].TotalItems,
				Feeds:       generatedFeedPages[i].Feeds,
			}
			if err := p.renderFeedsPage(
				config,
				&feedsPage,
				[]FeedListingSection{section},
				pageSlug,
				generatedTitle,
				generatedDescription,
				nil,
				&generatedFeedPages[i],
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *FeedsListingPlugin) collectFeedSections(feedConfigs []models.FeedConfig, config *lifecycle.Config, feedsPage *models.FeedsPageConfig, feedDefaults models.FeedDefaults) ([]FeedListingSection, []FeedListingPage) {
	syndication := getSyndicationConfig(config)
	userDefined := make([]FeedListingInfo, 0, len(feedConfigs))
	generated := make([]FeedListingInfo, 0, len(feedConfigs))
	configuredSlugs := configuredFeedSlugs(config)

	for i := range feedConfigs {
		fc := &feedConfigs[i]
		if fc.IncludePrivate {
			continue
		}

		postCount, latestDate, latestTime := publicFeedStats(fc.Posts)
		display, primary, archive, utility := splitFeedVariants(fc, syndication)
		_, isConfigured := configuredSlugs[fc.Slug]
		info := FeedListingInfo{
			Title:           feedDisplayTitle(fc),
			Slug:            fc.Slug,
			Description:     fc.Description,
			Href:            feedHTMLHref(fc),
			PostCount:       postCount,
			LatestPostDate:  latestDate,
			LatestPostTime:  latestTime,
			SubscribeCount:  feedSubscribeCount(fc, syndication, postCount),
			ArchiveCount:    postCount,
			DisplayVariants: display,
			PrimaryVariants: primary,
			ArchiveVariants: archive,
			UtilityVariants: utility,
			SparklinePoints: buildFeedSparkline(fc.Posts),
			SparklineTitle:  buildFeedSparklineTitle(fc.Posts),
			GeneratedBySite: !isConfigured,
		}

		if isConfigured {
			userDefined = append(userDefined, info)
		} else {
			generated = append(generated, info)
		}
	}

	sort.SliceStable(userDefined, func(i, j int) bool {
		leftOrder, leftOK := configuredSlugs[userDefined[i].Slug]
		rightOrder, rightOK := configuredSlugs[userDefined[j].Slug]
		if leftOK && rightOK {
			return leftOrder < rightOrder
		}
		if leftOK != rightOK {
			return leftOK
		}
		return userDefined[i].Title < userDefined[j].Title
	})
	sort.SliceStable(generated, func(i, j int) bool {
		if generated[i].PostCount != generated[j].PostCount {
			return generated[i].PostCount > generated[j].PostCount
		}
		if !generated[i].LatestPostTime.Equal(generated[j].LatestPostTime) {
			return generated[i].LatestPostTime.After(generated[j].LatestPostTime)
		}
		return generated[i].Title < generated[j].Title
	})

	sections := make([]FeedListingSection, 0, 2)
	generatedPages := make([]FeedListingPage, 0)
	if len(userDefined) > 0 {
		sections = append(sections, FeedListingSection{
			ID:          "configured-feeds",
			Title:       "Curated Feeds",
			Description: "Hand-picked collections grouped around the main themes of the site.",
			TotalCount:  len(userDefined),
			Feeds:       userDefined,
		})
	}
	if len(generated) > 0 {
		generatedPages = paginateFeedListings(
			generated,
			feedDefaults,
			"/"+filepath.ToSlash(filepath.Join(feedsPage.SlugPrefix, "generated"))+"/",
			"/"+filepath.ToSlash(filepath.Join(feedsPage.SlugPrefix, "generated")),
		)
		previewPages := paginateFeedListings(
			generated,
			feedDefaults,
			"/"+feedsPage.SlugPrefix+"/",
			"/"+filepath.ToSlash(filepath.Join(feedsPage.SlugPrefix, "generated")),
		)
		preview := generated
		truncated := false
		if len(previewPages) > 0 {
			preview = previewPages[0].Feeds
			truncated = previewPages[0].TotalPages > 1
		}

		section := FeedListingSection{
			ID:          "generated-feeds",
			Title:       "Generated Feeds",
			Description: "Automatically updated feeds for broader site sections, archives, and collections.",
			TotalCount:  len(generated),
			Feeds:       preview,
			Pagination:  nil,
		}
		if truncated {
			section.Pagination = &previewPages[0]
		}
		sections = append(sections, section)
	}

	return sections, generatedPages
}

func publicFeedStats(posts []*models.Post) (count int, latestDate string, latestTime time.Time) {
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
		return count, "", time.Time{}
	}
	return count, latest.Format("2006-01-02"), latest
}

func feedDisplayTitle(fc *models.FeedConfig) string {
	if fc.Title != "" {
		return fc.Title
	}
	if fc.Slug == "" {
		return homeFeedTitle
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

func splitFeedVariants(fc *models.FeedConfig, syndication models.SyndicationConfig) (display, primary, archive, utility []FeedVariantLink) {
	variants := feedVariantLinks(fc, syndication)
	for _, variant := range variants {
		switch variant.Kind {
		case feedVariantPage, feedVariantExport:
			display = append(display, variant)
		case feedVariantFeed:
			primary = append(primary, variant)
		case feedVariantArchive:
			archive = append(archive, variant)
		default:
			utility = append(utility, variant)
		}
	}
	return display, primary, archive, utility
}

func feedSubscribeCount(fc *models.FeedConfig, syndication models.SyndicationConfig, totalPosts int) int {
	if fc == nil {
		return 0
	}
	if isArchiveFeed(fc) {
		return totalPosts
	}
	if syndication.MaxItems <= 0 || totalPosts < syndication.MaxItems {
		return totalPosts
	}
	return syndication.MaxItems
}

func configuredFeedSlugs(config *lifecycle.Config) map[string]int {
	slugs := make(map[string]int)
	if config == nil || config.Extra == nil {
		return slugs
	}
	feeds, ok := config.Extra["feeds"].([]models.FeedConfig)
	if !ok {
		return slugs
	}
	for i := range feeds {
		slugs[feeds[i].Slug] = i
	}
	return slugs
}

func buildFeedSparkline(posts []*models.Post) string {
	buckets := monthlyPostBuckets(posts)
	if len(buckets) < 2 {
		return ""
	}

	maxValue := 0
	for _, count := range buckets {
		if count > maxValue {
			maxValue = count
		}
	}
	if maxValue == 0 {
		return ""
	}

	const width = 96
	const height = 28
	step := float64(width) / float64(len(buckets)-1)
	points := make([]string, 0, len(buckets))
	for i, count := range buckets {
		x := float64(i) * step
		y := float64(height)
		if maxValue > 0 {
			y = float64(height) - ((float64(count) / float64(maxValue)) * float64(height-4))
		}
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}
	return strings.Join(points, " ")
}

func buildFeedSparklineTitle(posts []*models.Post) string {
	buckets := monthlyPostBuckets(posts)
	if len(buckets) == 0 {
		return ""
	}
	return "Posts published over time"
}

func monthlyPostBuckets(posts []*models.Post) []int {
	counts := map[string]int{}
	months := make([]string, 0)
	seen := map[string]bool{}
	for _, post := range posts {
		if post == nil || post.Private || post.Skip || post.Draft || !post.Published || post.Date == nil {
			continue
		}
		monthKey := post.Date.UTC().Format("2006-01")
		counts[monthKey]++
		if !seen[monthKey] {
			seen[monthKey] = true
			months = append(months, monthKey)
		}
	}
	if len(months) == 0 {
		return nil
	}
	sort.Strings(months)
	buckets := make([]int, 0, len(months))
	for _, month := range months {
		buckets = append(buckets, counts[month])
	}
	return buckets
}

func paginateFeedListings(feeds []FeedListingInfo, defaults models.FeedDefaults, firstPageURL, baseURL string) []FeedListingPage {
	if len(feeds) == 0 {
		return nil
	}

	itemsPerPage := defaults.ItemsPerPage
	if itemsPerPage <= 0 {
		itemsPerPage = 10
	}
	orphanThreshold := defaults.OrphanThreshold
	if orphanThreshold <= 0 {
		orphanThreshold = 3
	}

	pages := make([]FeedListingPage, 0)
	for i := 0; i < len(feeds); i += itemsPerPage {
		end := i + itemsPerPage
		if end > len(feeds) {
			end = len(feeds)
		}
		remaining := len(feeds) - end
		if remaining > 0 && remaining < orphanThreshold {
			end = len(feeds)
		}
		pageNum := len(pages) + 1
		pages = append(pages, FeedListingPage{
			Number:  pageNum,
			Feeds:   feeds[i:end],
			HasPrev: pageNum > 1,
		})
		if end >= len(feeds) {
			break
		}
	}

	totalPages := len(pages)
	pageURLs := make([]string, totalPages)
	for i := 0; i < totalPages; i++ {
		if i == 0 {
			pageURLs[i] = firstPageURL
		} else {
			pageURLs[i] = baseURL + "/page/" + fmt.Sprintf("%d", i+1) + "/"
		}
	}
	for i := range pages {
		pages[i].HasNext = i < totalPages-1
		pages[i].TotalPages = totalPages
		pages[i].TotalItems = len(feeds)
		pages[i].ItemsPerPage = itemsPerPage
		pages[i].PageURLs = pageURLs
		if pages[i].HasPrev {
			if i == 1 {
				pages[i].PrevURL = firstPageURL
			} else {
				pages[i].PrevURL = baseURL + "/page/" + fmt.Sprintf("%d", i) + "/"
			}
		}
		if pages[i].HasNext {
			pages[i].NextURL = baseURL + "/page/" + fmt.Sprintf("%d", i+2) + "/"
		}
	}

	return pages
}

func pathJoinURL(base, suffix string) string {
	if base == "/" {
		return "/" + suffix
	}
	return base + suffix
}

func (p *FeedsListingPlugin) renderFeedsPage(
	config *lifecycle.Config,
	feedsPage *models.FeedsPageConfig,
	sections []FeedListingSection,
	pageSlug string,
	title string,
	description string,
	pageLinks []FeedVariantLink,
	pagination *FeedListingPage,
) error {
	feedsDir := filepath.Join(config.OutputDir, filepath.FromSlash(pageSlug))
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
	syntheticPost := &models.Post{
		Slug:        pageSlug,
		Title:       &title,
		Description: &description,
	}

	ctx := templates.NewContext(syntheticPost, "", modelsConfig)
	ctx.Extra["feed_sections"] = sections
	ctx.Extra["page_links"] = pageLinks
	ctx.Extra["pagination"] = pagination
	totalFeeds := 0
	for _, section := range sections {
		totalFeeds += section.TotalCount
	}
	ctx.Extra["total_feeds"] = totalFeeds

	html, err := engine.Render(feedsPage.Template, ctx)
	if err != nil {
		return fmt.Errorf("rendering feeds template: %w", err)
	}

	outputPath := filepath.Join(feedsDir, "index.html")
	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil { //nolint:gosec // web output must be world-readable
		return fmt.Errorf("writing feeds listing page: %w", err)
	}

	log.Printf("[feeds_listing] Generated /%s/ with %d feeds", pageSlug, totalFeeds)
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
