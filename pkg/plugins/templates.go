package plugins

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// Output format constants.
const (
	formatHTML     = "html"
	formatTxt      = "txt"
	formatText     = "text"
	formatMarkdown = "markdown"
	formatMD       = "md"
	formatOG       = "og"

	// defaultTemplate is the default template name for posts.
	defaultTemplate = "post.html"
)

// TemplatesPlugin wraps rendered markdown content in HTML templates.
// It operates during the render stage, after markdown has been converted to HTML.
type TemplatesPlugin struct {
	engine       *templates.Engine
	layoutConfig *models.LayoutConfig
	config       *lifecycle.Config
}

// NewTemplatesPlugin creates a new templates plugin.
func NewTemplatesPlugin() *TemplatesPlugin {
	return &TemplatesPlugin{}
}

// Name returns the plugin name.
func (p *TemplatesPlugin) Name() string {
	return PluginNameTemplates
}

// Configure initializes the template engine from the config.
func (p *TemplatesPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Get templates directory from config
	templatesDir := PluginNameTemplates
	if extra, ok := config.Extra["templates_dir"].(string); ok && extra != "" {
		templatesDir = extra
	}

	// Get theme name from config (default to "default")
	themeName := ThemeDefault
	if extra := config.Extra; extra != nil {
		// Check for typed ThemeConfig struct (set by core.go)
		if theme, ok := extra["theme"].(models.ThemeConfig); ok {
			if theme.Name != "" {
				themeName = theme.Name
			}
		}
		// Also check for map[string]interface{} (legacy/dynamic config)
		if theme, ok := extra["theme"].(map[string]interface{}); ok {
			if name, ok := theme["name"].(string); ok && name != "" {
				themeName = name
			}
		}
		// Also check for simple theme string
		if name, ok := extra["theme"].(string); ok && name != "" {
			themeName = name
		}
	}

	// Initialize template engine with theme support
	engine, err := templates.NewEngineWithTheme(templatesDir, themeName)
	if err != nil {
		return fmt.Errorf("failed to initialize template engine: %w", err)
	}
	p.engine = engine

	// Store engine in cache for other plugins to use
	m.Cache().Set("templates.engine", engine)

	// Store this plugin in cache for other plugins to use per-format resolution
	m.Cache().Set("templates.plugin", p)

	// Get layout config if available
	switch lc := config.Extra["layout"].(type) {
	case *models.LayoutConfig:
		p.layoutConfig = lc
	case models.LayoutConfig:
		p.layoutConfig = &lc
	}

	// Store config reference for template preset resolution
	p.config = config

	return nil
}

// resolveTemplate determines the template to use for a post (HTML format).
// This is a convenience wrapper for resolveTemplateForFormat with "html" format.
// Priority: frontmatter per-format -> preset -> simple template -> layout config -> global default
func (p *TemplatesPlugin) resolveTemplate(post *models.Post) string {
	return p.resolveTemplateForFormat(post, "html")
}

// resolveTemplateForFormat determines the template to use for a post and output format.
// Resolution priority:
// 1. Frontmatter per-format override (templates.html, templates.txt, etc.)
// 2. Frontmatter template preset (template: blog → expand to preset)
// 3. Frontmatter simple template (template: post.html → use for current format)
// 4. Layout config (path/feed-based)
// 5. Global default for format (default_templates.html, etc.)
// 6. Hardcoded default (post.html, default.txt, etc.)
func (p *TemplatesPlugin) resolveTemplateForFormat(post *models.Post, format string) string {
	// 1. Check per-format override in frontmatter
	if post.Templates != nil {
		if tmpl, ok := post.Templates[format]; ok && tmpl != "" {
			return tmpl
		}
	}

	// 2. Check if template is a preset name
	if post.Template != "" && p.config != nil {
		presets := getTemplatePresets(p.config)
		if preset, ok := presets[post.Template]; ok {
			tmpl := preset.TemplateForFormat(format)
			if tmpl != "" {
				return tmpl
			}
		}
	}

	// 3. Use template as explicit file (if has extension) and adapt for format
	if post.Template != "" && strings.Contains(post.Template, ".") {
		return adaptTemplateForFormat(post.Template, format)
	}

	// 4. Use template as-is if it doesn't have an extension
	// This might be a preset name that wasn't found, fall through to layout
	if post.Template != "" {
		// For HTML, use the template directly
		if format == formatHTML {
			return post.Template
		}
		// For other formats, try to adapt it
		return adaptTemplateForFormat(post.Template+".html", format)
	}

	// 5. Use layout configuration to determine template
	if p.layoutConfig != nil {
		// Get feed slug for feed-based layout lookup
		feedSlug := post.PrevNextFeed
		if feedSlug == "" {
			if feed, ok := post.Extra["feed"].(string); ok {
				feedSlug = feed
			}
		}

		// Get post path for path-based layout lookup
		postPath := post.Href
		if postPath == "" {
			postPath = "/" + strings.TrimPrefix(post.Path, "/")
		}

		// Resolve layout based on path and feed
		layout := p.layoutConfig.ResolveLayout(postPath, feedSlug)
		if layout != "" {
			baseTemplate := models.LayoutToTemplate(layout)
			return adaptTemplateForFormat(baseTemplate, format)
		}
	}

	// 6. Check global default templates from config
	if p.config != nil {
		defaultTemplates := getDefaultTemplates(p.config)
		if tmpl, ok := defaultTemplates[format]; ok && tmpl != "" {
			return tmpl
		}
	}

	// 7. Fall back to hardcoded defaults per format
	return getHardcodedDefault(format)
}

// adaptTemplateForFormat adapts a template name for a specific output format.
// For example: post.html → post.txt, post.md, post-og.html
func adaptTemplateForFormat(template, format string) string {
	ext := filepath.Ext(template)
	base := strings.TrimSuffix(template, ext)

	switch format {
	case formatHTML:
		return template
	case formatTxt, formatText:
		return base + ".txt"
	case formatMarkdown, formatMD:
		return base + ".md"
	case formatOG:
		return base + "-og.html"
	default:
		return template
	}
}

// getHardcodedDefault returns the hardcoded default template for a format.
func getHardcodedDefault(format string) string {
	switch format {
	case formatHTML:
		return defaultTemplate
	case formatTxt, formatText:
		return "default.txt"
	case formatMarkdown, formatMD:
		return "raw.txt"
	case formatOG:
		return "og-card.html"
	default:
		return defaultTemplate
	}
}

// getTemplatePresets extracts TemplatePresets from lifecycle.Config.Extra.
func getTemplatePresets(config *lifecycle.Config) map[string]models.TemplatePreset {
	if config.Extra == nil {
		return nil
	}
	if presets, ok := config.Extra["template_presets"].(map[string]models.TemplatePreset); ok {
		return presets
	}
	return nil
}

// getDefaultTemplates extracts DefaultTemplates from lifecycle.Config.Extra.
func getDefaultTemplates(config *lifecycle.Config) map[string]string {
	if config.Extra == nil {
		return nil
	}
	if defaults, ok := config.Extra["default_templates"].(map[string]string); ok {
		return defaults
	}
	return nil
}

// Render wraps markdown content in templates.
// This runs after markdown rendering, using post.ArticleHTML as the body.
// Skips posts that don't need rebuilding (incremental builds).
//
// Uses three-phase processing for incremental optimization:
// Phase 1a: Quick single-threaded pass to classify posts (no disk I/O)
// Phase 1b: Concurrent batch read of cached HTML files for unchanged posts
// Phase 2: Concurrent rendering only for posts that need it
func (p *TemplatesPlugin) Render(m *lifecycle.Manager) error {
	if p.engine == nil {
		return fmt.Errorf("template engine not initialized")
	}

	// Get config for template context
	config := m.Config()

	// Get build cache to check if posts need rebuilding
	cache := GetBuildCache(m)
	changedSlugs := getChangedSlugsMap(cache)

	// Collect private paths for robots.txt template variable
	privatePaths := collectPrivatePaths(m.Posts())

	// Pre-compute feed membership hashes once (O(N)) instead of per-post (O(N^2))
	feedMembershipHashes := precomputeFeedMembershipHashes(config, m)

	// Phase 1a: Classify posts into "cacheable" vs "needs rendering" without disk I/O.
	// For cacheable posts, we collect the source path so we can batch-read HTML later.
	t0 := time.Now()
	var cacheablePosts []cacheablePost
	var postsNeedingRender []*models.Post

	for _, post := range m.Posts() {
		// Skip posts marked to skip or without article HTML
		if post.Skip || post.ArticleHTML == "" {
			continue
		}

		// Check if we can use cached HTML (no disk I/O -- just map lookups)
		if canUseCachedHTML(post, cache, changedSlugs, feedMembershipHashes) {
			cacheablePosts = append(cacheablePosts, cacheablePost{post: post, path: post.Path})
		} else {
			postsNeedingRender = append(postsNeedingRender, post)
		}
	}
	t1 := time.Now()
	log.Printf("[templates] Phase 1a classify: %d cacheable, %d need render (took %v)", len(cacheablePosts), len(postsNeedingRender), t1.Sub(t0))

	// Phase 1b: Batch-read all cached HTML files concurrently.
	// This converts ~2900 sequential os.ReadFile calls into a parallel batch,
	// significantly reducing wall-clock time for the cache restore phase.
	if len(cacheablePosts) > 0 && cache != nil {
		p.batchRestoreCachedHTML(cacheablePosts, cache, &postsNeedingRender)
	}
	t2 := time.Now()
	log.Printf("[templates] Phase 1b batch restore: took %v, %d now need render", t2.Sub(t1), len(postsNeedingRender))

	// Phase 2: Process only posts that need rendering concurrently
	err := m.ProcessPostsSliceConcurrently(postsNeedingRender, func(post *models.Post) error {
		// Render the template
		html, err := p.renderPost(post, config, m, privatePaths)
		if err != nil {
			return err
		}
		post.HTML = html

		// Cache the full HTML for future incremental builds
		if cache != nil && post.InputHash != "" {
			//nolint:errcheck // caching is best-effort, failures are non-fatal
			cache.CacheFullHTML(post.Path, html)
			// Store feed membership hash for future builds
			if membershipHash := lookupFeedMembershipHash(post, feedMembershipHashes); membershipHash != "" {
				cache.SetFeedMembershipHash(post.Path, membershipHash)
			}
		}

		return nil
	})
	t3 := time.Now()
	log.Printf("[templates] Phase 2 render: took %v", t3.Sub(t2))
	return err
}

// cacheablePost pairs a post with its source path for batch cache operations.
type cacheablePost struct {
	post *models.Post
	path string // source path for cache lookup
}

// batchRestoreCachedHTML reads cached HTML files concurrently and assigns them to posts.
// Posts whose cache files are missing or unreadable are moved to postsNeedingRender.
func (p *TemplatesPlugin) batchRestoreCachedHTML(
	posts []cacheablePost,
	cache *buildcache.Cache,
	postsNeedingRender *[]*models.Post,
) {
	type result struct {
		idx  int
		html string
	}

	results := make([]result, 0, len(posts))
	resultCh := make(chan result, len(posts))

	// Use a worker pool with bounded concurrency for parallel file reads
	const numWorkers = 32
	jobs := make(chan int, len(posts))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				html := cache.GetCachedFullHTML(posts[idx].path)
				resultCh <- result{idx: idx, html: html}
			}
		}()
	}

	// Send all jobs
	for i := range posts {
		jobs <- i
	}
	close(jobs)

	// Collect results in background
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for r := range resultCh {
		results = append(results, r)
	}

	// Assign HTML to posts, moving cache misses to postsNeedingRender
	for _, r := range results {
		if r.html != "" {
			posts[r.idx].post.HTML = r.html
		} else {
			*postsNeedingRender = append(*postsNeedingRender, posts[r.idx].post)
		}
	}
}

// canUseCachedHTML checks if a post can use cached HTML without doing any disk I/O.
// This is the "decision" phase that determines cache eligibility.
func canUseCachedHTML(post *models.Post, cache *buildcache.Cache, changedSlugs map[string]bool, feedMembershipHashes map[string]string) bool {
	if cache == nil || post.InputHash == "" {
		return false
	}

	// Check if post itself changed
	if cache.ShouldRebuild(post.Path, post.InputHash, post.Template) {
		return false
	}

	// Check if any dependency changed
	if len(changedSlugs) > 0 {
		for _, dep := range post.Dependencies {
			if changedSlugs[dep] {
				return false
			}
		}
		// Check if this post's slug is in changedSlugs
		if changedSlugs[post.Slug] {
			return false
		}
	}

	// Check if feed membership changed (for sidebar invalidation)
	if currentHash := lookupFeedMembershipHash(post, feedMembershipHashes); currentHash != "" {
		cachedHash := cache.GetFeedMembershipHash(post.Path)
		if cachedHash != currentHash {
			return false
		}
	}

	return true
}

// getChangedSlugsMap returns a map of slugs that changed in this build.
func getChangedSlugsMap(cache *buildcache.Cache) map[string]bool {
	if cache == nil {
		return nil
	}
	changedSlugs := make(map[string]bool)
	for _, slug := range cache.GetChangedSlugs() {
		changedSlugs[slug] = true
	}
	return changedSlugs
}

// precomputeFeedMembershipHashes builds a map of tag -> membership hash in O(N) time.
// This replaces the per-post O(N) scan that previously made the overall complexity O(N^2).
func precomputeFeedMembershipHashes(config *lifecycle.Config, m *lifecycle.Manager) map[string]string {
	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok {
		return nil
	}
	if components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return nil
	}
	feedSlugs := components.FeedSidebar.Feeds
	if len(feedSlugs) == 0 {
		return nil
	}

	// Collect the tag names we care about
	tagNames := make(map[string]bool, len(feedSlugs))
	for _, feedSlug := range feedSlugs {
		if strings.HasPrefix(feedSlug, "tags/") {
			tagNames[strings.TrimPrefix(feedSlug, "tags/")] = true
		}
	}
	if len(tagNames) == 0 {
		return nil
	}

	// Single pass over all posts: group member slugs by tag
	tagMembers := make(map[string][]string, len(tagNames))
	for _, post := range m.Posts() {
		if !post.Published || post.Draft || post.Skip {
			continue
		}
		for _, t := range post.Tags {
			if tagNames[t] {
				tagMembers[t] = append(tagMembers[t], post.Slug)
			}
		}
	}

	// Compute hash per tag
	result := make(map[string]string, len(tagMembers))
	for tag, slugs := range tagMembers {
		result[tag] = buildcache.ComputeFeedMembershipHash(slugs)
	}
	return result
}

// lookupFeedMembershipHash finds the feed membership hash for a post using the pre-computed map.
// Returns the hash of the first matching tag's membership, or empty string.
func lookupFeedMembershipHash(post *models.Post, feedMembershipHashes map[string]string) string {
	if len(feedMembershipHashes) == 0 {
		return ""
	}
	for _, tag := range post.Tags {
		if h, ok := feedMembershipHashes[tag]; ok {
			return h
		}
	}
	return ""
}

// renderPost renders a single post using the appropriate template.
func (p *TemplatesPlugin) renderPost(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager, privatePaths []string) (string, error) {
	// Determine which template to use
	templateName := p.resolveTemplate(post)

	// Check if template exists, fall back to post.html if not
	if !p.engine.TemplateExists(templateName) {
		templateName = "post.html"
		if !p.engine.TemplateExists(templateName) {
			return post.ArticleHTML, nil
		}
	}

	// Create template context
	modelsConfig := ToModelsConfig(config)
	ctx := templates.NewContext(post, post.ArticleHTML, modelsConfig)
	ctx = ctx.WithCore(m)
	ctx.Set("private_paths", privatePaths)
	if modelsConfig.Garden.IsExportJSON() {
		ctx.Set("graph_json", "/"+modelsConfig.Garden.GetPath()+"/graph.json")
	}

	// Share buttons at the end of posts
	if modelsConfig.Components.Share.IsEnabled() {
		shareButtons := models.BuildShareButtons(modelsConfig.Components.Share, modelsConfig.URL, modelsConfig.Title, post)
		if len(shareButtons) > 0 {
			ctx.Set("share_buttons", shareButtons)
		}
	}

	// Inject feed sidebar posts if configured
	sidebarPosts, sidebarFeed := p.getFeedSidebarPosts(post, config, m)
	if sidebarPosts != nil {
		ctx.Set("sidebar_posts", sidebarPosts)
		if sidebarFeed != nil {
			ctx.Set("sidebar_feed", sidebarFeed)
		}

		// Calculate prev/next within the sidebar feed
		sidebarPrev, sidebarNext := p.getSidebarPrevNext(post, sidebarPosts)
		if sidebarPrev != nil {
			ctx.Set("sidebar_prev", sidebarPrev)
		}
		if sidebarNext != nil {
			ctx.Set("sidebar_next", sidebarNext)
		}
	}

	// Inject discovery feed for per-page feed discovery links
	// If post has a sidebar feed, use that; otherwise use site default
	discoveryFeed := p.getDiscoveryFeed(post, sidebarFeed, m)
	if discoveryFeed != nil {
		ctx.Set("discovery_feed", DiscoveryFeedToMap(discoveryFeed))
	}

	// Render the template
	html, err := p.engine.Render(templateName, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to render template %q for post %q: %w", templateName, post.Path, err)
	}

	return html, nil
}

// getFeedSidebarPosts returns the posts for the feed sidebar if the post belongs to a configured feed.
// It checks if the post's tags match any of the configured feed_sidebar.feeds.
// The function directly computes feed membership from tags since feed_configs may not be
// available during the Render stage (feeds are built during Collect stage, which runs after Render).
func (p *TemplatesPlugin) getFeedSidebarPosts(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
	seriesPosts, seriesFeed := p.getSeriesSidebarPosts(post, config, m)
	if seriesPosts != nil {
		return seriesPosts, seriesFeed
	}

	// Get components config
	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok {
		return nil, nil
	}

	// Check if feed sidebar is enabled
	if components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return nil, nil
	}

	// Get configured feed slugs (e.g., ["tags/daily-note"])
	feedSlugs := components.FeedSidebar.Feeds
	if len(feedSlugs) == 0 {
		return nil, nil
	}

	// Check if this post belongs to any of the configured feeds
	// For tag-based feeds (tags/xxx), check if post has the tag
	for _, feedSlug := range feedSlugs {
		if !strings.HasPrefix(feedSlug, "tags/") {
			continue // Only handle tag feeds for now
		}

		// Extract tag name from feed slug (e.g., "tags/daily-note" -> "daily-note")
		tagName := strings.TrimPrefix(feedSlug, "tags/")

		// Check if post has this tag
		hasTag := false
		for _, postTag := range post.Tags {
			if postTag == tagName {
				hasTag = true
				break
			}
		}

		if !hasTag {
			continue
		}

		// Post belongs to this feed - collect all posts with this tag
		allPosts := m.Posts()
		sidebarPosts := make([]*models.Post, 0)
		for _, feedPost := range allPosts {
			for _, t := range feedPost.Tags {
				if t == tagName && feedPost.Published && !feedPost.Draft && !feedPost.Skip {
					sidebarPosts = append(sidebarPosts, feedPost)
					break
				}
			}
		}

		// Sort by date (newest first)
		sortPostsByDate(sidebarPosts, true)

		// Create a feed config for template display
		feedConfig := &models.FeedConfig{
			Slug:  feedSlug,
			Title: fmt.Sprintf("Posts tagged: %s", tagName),
			Posts: sidebarPosts,
		}

		return sidebarPosts, feedConfig
	}

	return nil, nil
}

func (p *TemplatesPlugin) getSeriesSidebarPosts(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
	if post == nil || config == nil || m == nil {
		return nil, nil
	}

	seriesName := getStringFromExtra(post.Extra, seriesKey)
	if seriesName == "" {
		return nil, nil
	}

	seriesCfg := parseSeriesConfig(config)
	if !seriesCfg.AutoSidebar {
		return nil, nil
	}

	seriesSlug := buildSeriesFeedSlug(seriesCfg.SlugPrefix, slugify(seriesName))

	allPosts := m.Posts()
	seriesPosts := make([]*models.Post, 0)
	for _, feedPost := range allPosts {
		postSeries := getStringFromExtra(feedPost.Extra, seriesKey)
		if postSeries == "" {
			continue
		}
		postSeriesSlug := buildSeriesFeedSlug(seriesCfg.SlugPrefix, slugify(postSeries))
		if postSeriesSlug == seriesSlug {
			seriesPosts = append(seriesPosts, feedPost)
		}
	}

	if len(seriesPosts) == 0 {
		return nil, nil
	}

	seriesSlugValue := slugify(seriesName)
	group := &seriesGroup{
		name:  seriesName,
		slug:  seriesSlugValue,
		posts: seriesPosts,
		cfg:   resolveSeriesOverride(seriesCfg, seriesName, seriesSlugValue),
	}

	sortSeriesPosts(group, false)
	publishedPosts := filterSeriesOutputPosts(group.posts)
	if len(publishedPosts) == 0 {
		return nil, nil
	}

	feedConfig := &models.FeedConfig{
		Slug:  seriesSlug,
		Title: seriesDisplayTitle(group.name, group.cfg),
		Posts: publishedPosts,
	}
	if group.cfg != nil && group.cfg.Description != "" {
		feedConfig.Description = group.cfg.Description
	}

	return publishedPosts, feedConfig
}

// getSidebarPrevNext finds the previous and next posts relative to the current post
// within the sidebar posts list. The sidebar posts are sorted by date (newest first),
// so "prev" is the newer post (earlier in the list) and "next" is the older post.
func (p *TemplatesPlugin) getSidebarPrevNext(currentPost *models.Post, sidebarPosts []*models.Post) (prev, next *models.Post) {
	if len(sidebarPosts) == 0 {
		return nil, nil
	}

	// Find current post's position in sidebar posts
	position := -1
	for i, post := range sidebarPosts {
		if post.Slug == currentPost.Slug {
			position = i
			break
		}
	}

	if position == -1 {
		return nil, nil
	}

	// Since posts are sorted newest first:
	// - prev (newer) is at position-1
	// - next (older) is at position+1
	if position > 0 {
		prev = sidebarPosts[position-1]
	}
	if position < len(sidebarPosts)-1 {
		next = sidebarPosts[position+1]
	}

	return prev, next
}

// getDiscoveryFeed returns the discovery feed for a post.
// If the post has a sidebar feed, that feed is used for discovery.
// Otherwise, the site default feed (root subscription feed) is used.
func (p *TemplatesPlugin) getDiscoveryFeed(post *models.Post, sidebarFeed *models.FeedConfig, m *lifecycle.Manager) *DiscoveryFeed {
	// Get feed configs from cache
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	return GetDiscoveryFeed(post, sidebarFeed, feedConfigs)
}

// sortPostsByDate sorts posts by date.
// If reverse is true, sorts newest first.
func sortPostsByDate(posts []*models.Post, reverse bool) {
	sort.Slice(posts, func(i, j int) bool {
		// Handle nil dates
		if posts[i].Date == nil && posts[j].Date == nil {
			return false
		}
		if posts[i].Date == nil {
			return !reverse
		}
		if posts[j].Date == nil {
			return reverse
		}
		if reverse {
			return posts[i].Date.After(*posts[j].Date)
		}
		return posts[i].Date.Before(*posts[j].Date)
	})
}

// collectPrivatePaths returns a list of paths (hrefs) for all private posts.
// These paths are used in robots.txt templates to add Disallow directives.
// Includes all format variants (.txt, .md, .og) and excludes the robots post itself.
func collectPrivatePaths(posts []*models.Post) []string {
	var paths []string
	for _, post := range posts {
		if post.Private && !post.Draft && !post.Skip {
			// Skip the robots post itself to avoid self-reference
			if post.Slug == "robots" {
				continue
			}
			// Add base href (e.g., /slug/)
			paths = append(paths, post.Href)
			// Add format variants
			// For regular posts: /slug.txt, /slug.md, /slug.og/
			if post.Slug != "" {
				paths = append(paths,
					"/"+post.Slug+".txt",
					"/"+post.Slug+".md",
					"/"+post.Slug+".og/",
				)
			}
		}
	}
	return paths
}

// Priority returns the plugin priority for the given stage.
// Templates should run late in the render stage, after markdown rendering.
func (p *TemplatesPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLate // Run after markdown rendering
	}
	return lifecycle.PriorityDefault
}

// Engine returns the template engine for use by other plugins.
func (p *TemplatesPlugin) Engine() *templates.Engine {
	return p.engine
}

// ToModelsConfig converts lifecycle.Config to models.Config for template context.
// This is exported for use by other plugins that need to build template contexts
// with the full config (e.g., publish_feeds, blogroll).
// Note: This is not cached because lifecycle.Config.Extra is mutable and may be
// modified by plugins throughout the build process (e.g., image_zoom sets glightbox_enabled).
func ToModelsConfig(config *lifecycle.Config) *models.Config {
	return toModelsConfigUncached(config)
}

// toModelsConfigUncached is the actual conversion logic, used by the cache.
func toModelsConfigUncached(config *lifecycle.Config) *models.Config {
	if config == nil {
		return nil
	}
	// Convert lifecycle.Config to models.Config
	modelsConfig := &models.Config{
		OutputDir:    config.OutputDir,
		Title:        getStringFromExtra(config.Extra, "title"),
		URL:          getStringFromExtra(config.Extra, "url"),
		Description:  getStringFromExtra(config.Extra, "description"),
		Author:       getStringFromExtra(config.Extra, "author"),
		TemplatesDir: getStringFromExtra(config.Extra, "templates_dir"),
	}

	// Copy nav items if available
	if navItems, ok := config.Extra["nav"].([]models.NavItem); ok {
		modelsConfig.Nav = navItems
	}

	// Copy footer config if available
	if footer, ok := config.Extra["footer"].(models.FooterConfig); ok {
		modelsConfig.Footer = footer
	}

	// Copy layout config if available
	switch layoutVal := config.Extra["layout"].(type) {
	case *models.LayoutConfig:
		modelsConfig.Layout = *layoutVal
	case models.LayoutConfig:
		modelsConfig.Layout = layoutVal
	}

	// Copy sidebar config if available
	if sidebar, ok := config.Extra["sidebar"].(models.SidebarConfig); ok {
		modelsConfig.Sidebar = sidebar
	}

	// Copy toc config if available
	if toc, ok := config.Extra["toc"].(models.TocConfig); ok {
		modelsConfig.Toc = toc
	}

	// Copy header config if available
	if header, ok := config.Extra["header"].(models.HeaderLayoutConfig); ok {
		modelsConfig.Header = header
	}

	// Copy SEO config if available
	switch seoVal := config.Extra["seo"].(type) {
	case models.SEOConfig:
		modelsConfig.SEO = seoVal
	case map[string]interface{}:
		modelsConfig.SEO = models.SEOConfig{
			TwitterHandle: getStringFromMap(seoVal, "twitter_handle"),
			DefaultImage:  getStringFromMap(seoVal, "default_image"),
			LogoURL:       getStringFromMap(seoVal, "logo_url"),
			AuthorImage:   getStringFromMap(seoVal, "author_image"),
		}
	}

	// Copy Search config if available, otherwise use defaults
	// This ensures search is enabled by default with position "navbar"
	if search, ok := config.Extra["search"].(models.SearchConfig); ok {
		modelsConfig.Search = search
	} else {
		modelsConfig.Search = models.NewSearchConfig()
	}

	// Copy remaining plugin configs
	copyPluginConfigs(config, modelsConfig)

	// Copy the entire Extra map so templates can access dynamic plugin config
	// (e.g., glightbox_enabled, glightbox_options set by image_zoom plugin)
	if config.Extra != nil {
		modelsConfig.Extra = make(map[string]any)
		for k, v := range config.Extra {
			modelsConfig.Extra[k] = v
		}
	}

	return modelsConfig
}

// copyPluginConfigs copies plugin-specific config sections from lifecycle.Config to models.Config.
func copyPluginConfigs(config *lifecycle.Config, modelsConfig *models.Config) {
	// Copy Components config if available
	if components, ok := config.Extra["components"].(models.ComponentsConfig); ok {
		modelsConfig.Components = components
	}

	// Copy PostFormats config if available
	if postFormats, ok := config.Extra["post_formats"].(models.PostFormatsConfig); ok {
		modelsConfig.PostFormats = postFormats
	}

	// Copy WebSub config if available
	if websub, ok := config.Extra["websub"].(models.WebSubConfig); ok {
		modelsConfig.WebSub = websub
	}

	// Copy Head config if available
	if head, ok := config.Extra["head"].(models.HeadConfig); ok {
		modelsConfig.Head = head
	}

	// Copy Theme config if available
	if theme, ok := config.Extra["theme"].(models.ThemeConfig); ok {
		modelsConfig.Theme = theme
	}

	// Copy Assets config if available
	if assetsConfig, ok := config.Extra["assets"].(models.AssetsConfig); ok {
		modelsConfig.Assets = assetsConfig
	}
	if assetsConfig, ok := config.Extra["assets"].(*models.AssetsConfig); ok && assetsConfig != nil {
		modelsConfig.Assets = *assetsConfig
	}

	// Copy Tags config if available
	if tags, ok := config.Extra["tags"].(models.TagsConfig); ok {
		modelsConfig.Tags = tags
	} else {
		modelsConfig.Tags = models.NewTagsConfig()
	}

	// Copy Garden config if available
	if garden, ok := config.Extra["garden"].(models.GardenConfig); ok {
		modelsConfig.Garden = garden
	} else {
		modelsConfig.Garden = models.NewGardenConfig()
	}
}

// getStringFromExtra safely gets a string value from the Extra map.
func getStringFromExtra(extra map[string]interface{}, key string) string {
	if extra == nil {
		return ""
	}
	if v, ok := extra[key].(string); ok {
		return v
	}
	return ""
}

// getStringFromMap safely gets a string value from a map.
func getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
