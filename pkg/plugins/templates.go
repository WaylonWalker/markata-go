package plugins

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/logging"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

var templatesLog = logging.Component("templates").Phase("render")

// Output format constants.
const (
	formatHTML     = "html"
	formatTxt      = "txt"
	formatText     = "text"
	formatANSI     = "ansi"
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
	if modelsConfig, ok := config.Extra["models_config"].(*models.Config); ok && modelsConfig != nil {
		templates.SetTrustedMediaDomains(modelsConfig.Templates.Media.TrustedDomains)
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
			return post.Template + ".html"
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
	case formatANSI:
		return base + ".ansi"
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
	case formatANSI:
		return "default.ansi"
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
// ensureFeedConfigsCached pre-computes feed configs and caches them if
// the Collect stage (which normally does this) hasn't run yet.
// This allows the feed sidebar auto-discovery to work during the Render stage.
func ensureFeedConfigsCached(config *lifecycle.Config, m *lifecycle.Manager) {
	if _, ok := m.Cache().Get("feed_configs"); ok {
		return // Already cached (e.g., Collect ran first)
	}

	feedConfigs := getFeedConfigs(config)
	if len(feedConfigs) == 0 {
		return
	}

	feedDefaults := getFeedDefaults(config)
	posts := m.Posts()
	fc := newFeedFilterCache(posts)

	for i := range feedConfigs {
		feedCfg := &feedConfigs[i]
		feedCfg.ApplyDefaults(feedDefaults)

		filteredPosts, err := fc.FilterPosts(feedCfg.Filter, feedCfg.IncludePrivate)
		if err != nil {
			continue
		}

		// Sort posts (same logic as feeds.go)
		sortField := feedCfg.Sort
		reverse := feedCfg.Reverse
		if sortField == "" {
			sortField = "date"
			reverse = true
		}
		sorted := cloneFeedPosts(filteredPosts)
		sortPosts(sorted, sortField, reverse)
		feedCfg.Posts = sorted
	}

	m.Cache().Set("feed_configs", feedConfigs)
}

// Phase 1a: Quick single-threaded pass to classify posts (no disk I/O)
// Phase 1b: Concurrent batch read of cached HTML files for unchanged posts
// Phase 2: Concurrent rendering only for posts that need it
func (p *TemplatesPlugin) Render(m *lifecycle.Manager) error {
	if p.engine == nil {
		return fmt.Errorf("template engine not initialized")
	}

	// Get config for template context
	config := m.Config()

	// Ensure feed_configs are available in cache for sidebar auto-discovery.
	// The Collect stage (feeds) runs AFTER Render, so we pre-compute here.
	ensureFeedConfigsCached(config, m)

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
	templatesLog.Printf("Phase 1a classify: %d cacheable, %d need render (took %v)", len(cacheablePosts), len(postsNeedingRender), t1.Sub(t0))

	// Phase 1b: Batch-read all cached HTML files concurrently.
	// This converts ~2900 sequential os.ReadFile calls into a parallel batch,
	// significantly reducing wall-clock time for the cache restore phase.
	if len(cacheablePosts) > 0 && cache != nil {
		p.batchRestoreCachedHTML(cacheablePosts, cache, &postsNeedingRender, m.Concurrency())
	}
	t2 := time.Now()
	templatesLog.Printf("Phase 1b batch restore: took %v, %d now need render", t2.Sub(t1), len(postsNeedingRender))

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
	templatesLog.Printf("Phase 2 render: took %v", t3.Sub(t2))
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
	concurrency int,
) {
	type result struct {
		idx  int
		html string
	}

	results := make([]result, 0, len(posts))
	resultCh := make(chan result, len(posts))

	// Use a worker pool with bounded concurrency for parallel file reads
	numWorkers := concurrency
	if numWorkers > len(posts) {
		numWorkers = len(posts)
	}
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
	modelsConfig := applyPostFormatsToConfig(ToModelsConfig(config), resolvePostFormats(post, config))
	ctx := templates.NewContext(post, post.ArticleHTML, modelsConfig)
	ctx = ctx.WithCore(m)
	ctx.Set("feed_posts", createFeedPostsFunc(m))
	ctx.Set("render_feed", createRenderFeedFunc(m))
	ctx.Set("render_slashes", createRenderSlashesFunc(m))
	ctx.Set("include_post", createIncludePostFunc(m))
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

	postCopyPayloads := buildPostCopyPayloads(post, config, modelsConfig.URL)
	ctx.Set("post_copy_payloads", postCopyPayloads)
	ctx.Set("post_copy_payloads_json", postCopyPayloads.JSON())

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

		// Build JSON for all candidate feeds (for client-side {/} cycling)
		feedsJSON := p.buildSidebarFeedsJSON(post, config, m, sidebarFeed)
		if feedsJSON != "" {
			ctx.Set("sidebar_feeds_json", feedsJSON)
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
// Resolution priority:
//  1. Series – post has a "series" key in Extra
//  2. Explicit frontmatter – post.PrevNextFeed or post.Extra["sidebar_feed"]
//  3. Tag-based feeds – from config feed_sidebar.feeds
//  4. Auto-discovery – find the best (smallest non-excluded) feed containing this post,
//     preferring primary feeds only as a tie-breaker for equally specific candidates.
func (p *TemplatesPlugin) getFeedSidebarPosts(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
	maxPosts := 0
	if config != nil {
		if components, ok := config.Extra["components"].(models.ComponentsConfig); ok {
			maxPosts = components.FeedSidebar.MaxPosts
		}
	}

	// 1. Series
	seriesPosts, seriesFeed := p.getSeriesSidebarPosts(post, config, m)
	if seriesPosts != nil {
		return seriesPosts, seriesFeed
	}

	// 2. Explicit frontmatter: sidebar_feed or PrevNextFeed
	if explicitSlug := p.getExplicitFeedSlug(post); explicitSlug != "" {
		posts, fc := p.feedFromCachedConfigs(explicitSlug, post, m)
		if posts != nil {
			return trimSidebarPosts(posts, post.Slug, maxPosts), fc
		}
	}

	// 3. Tag-based feeds from feed_sidebar config
	tagPosts, tagFeed := p.getTagFeedSidebarPosts(post, config, m)
	if tagPosts != nil {
		return trimSidebarPosts(tagPosts, post.Slug, maxPosts), tagFeed
	}

	// 4. Auto-discovery: find the best feed containing this post
	autoPosts, autoFeed := p.autoDiscoverFeed(post, config, m)
	if autoPosts != nil {
		return trimSidebarPosts(autoPosts, post.Slug, maxPosts), autoFeed
	}

	return nil, nil
}

func trimSidebarPosts(posts []*models.Post, currentSlug string, maxPosts int) []*models.Post {
	if maxPosts <= 0 || len(posts) <= maxPosts {
		return posts
	}

	currentIndex := -1
	for i, p := range posts {
		if p != nil && p.Slug == currentSlug {
			currentIndex = i
			break
		}
	}
	if currentIndex == -1 {
		trimmed := make([]*models.Post, maxPosts)
		copy(trimmed, posts[:maxPosts])
		return trimmed
	}

	half := maxPosts / 2
	start := currentIndex - half
	if start < 0 {
		start = 0
	}
	end := start + maxPosts
	if end > len(posts) {
		end = len(posts)
		start = end - maxPosts
		if start < 0 {
			start = 0
		}
	}

	trimmed := make([]*models.Post, end-start)
	copy(trimmed, posts[start:end])
	return trimmed
}

// getExplicitFeedSlug returns a feed slug from post frontmatter, if any.
// Checks post.Extra["sidebar_feed"] first, then post.PrevNextFeed.
func (p *TemplatesPlugin) getExplicitFeedSlug(post *models.Post) string {
	if post == nil {
		return ""
	}
	if post.Extra != nil {
		if sf, ok := post.Extra["sidebar_feed"].(string); ok && sf != "" {
			return sf
		}
	}
	if post.PrevNextFeed != "" {
		return post.PrevNextFeed
	}
	return ""
}

// feedFromCachedConfigs looks up a feed by slug in cached feed_configs and
// returns its posts if the current post is a member.
func (p *TemplatesPlugin) feedFromCachedConfigs(slug string, post *models.Post, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		return nil, nil
	}
	configs, ok := cached.([]models.FeedConfig)
	if !ok {
		return nil, nil
	}
	fc := GetFeedBySlug(slug, configs)
	if fc == nil || len(fc.Posts) == 0 {
		return nil, nil
	}
	// Verify the post is actually in the feed
	found := false
	for _, fp := range fc.Posts {
		if fp.Slug == post.Slug {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}
	return fc.Posts, fc
}

// getTagFeedSidebarPosts is the original tag-based feed sidebar logic extracted
// from the old getFeedSidebarPosts.
func (p *TemplatesPlugin) getTagFeedSidebarPosts(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
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

type autoFeedCandidate struct {
	fc    *models.FeedConfig
	count int
}

func selectBestAutoFeedCandidate(candidates []autoFeedCandidate) autoFeedCandidate {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.count < best.count || (c.count == best.count && c.fc.Primary && !best.fc.Primary) {
			best = c
		}
	}
	return best
}

// autoDiscoverFeed finds the best feed containing this post from cached feed_configs.
// It prefers smaller/more specific feeds over large catch-all feeds.
// Feeds with slugs like "archive", "all", "subscription-*" are deprioritized.
func (p *TemplatesPlugin) autoDiscoverFeed(post *models.Post, config *lifecycle.Config, m *lifecycle.Manager) ([]*models.Post, *models.FeedConfig) {
	if post == nil || post.Slug == "" {
		return nil, nil
	}

	// Check if feed sidebar is enabled (must be enabled for auto-discovery)
	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok {
		return nil, nil
	}
	if components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return nil, nil
	}

	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		return nil, nil
	}
	configs, ok := cached.([]models.FeedConfig)
	if !ok {
		return nil, nil
	}

	// Slugs to skip in auto-discovery (catch-all or meta feeds)
	skipSlugs := map[string]bool{
		"archive": true, "all": true, "sitemap": true, "search": true,
	}
	// Prefixes to skip
	skipPrefixes := []string{"subscription-", "tags/"}

	var candidates []autoFeedCandidate

	for i := range configs {
		fc := &configs[i]
		if skipSlugs[fc.Slug] {
			continue
		}
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(fc.Slug, prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Check if this post is in the feed
		for _, fp := range fc.Posts {
			if fp.Slug == post.Slug {
				candidates = append(candidates, autoFeedCandidate{fc: fc, count: len(fc.Posts)})
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Pick the smallest feed (most specific). If two candidates are equally
	// specific, prefer the primary feed so configured navigation still wins
	// when the scope is otherwise identical.
	best := selectBestAutoFeedCandidate(candidates)

	return best.fc.Posts, best.fc
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

// sidebarFeedJSON is the JSON structure for a single feed candidate
// used by the client-side feed cycling feature.
type sidebarFeedJSON struct {
	Slug                string            `json:"slug"`
	Title               string            `json:"title"`
	Priority            string            `json:"priority"`
	Primary             bool              `json:"primary,omitempty"`
	ContainsCurrentPost bool              `json:"containsCurrentPost,omitempty"`
	Variants            []sidebarVariant  `json:"variants,omitempty"`
	Posts               []sidebarPostJSON `json:"posts"`
	TotalPosts          int               `json:"totalPosts"`
	Prev                *sidebarPostJSON  `json:"prev,omitempty"`
	Next                *sidebarPostJSON  `json:"next,omitempty"`
}

type sidebarVariant struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

type sidebarPostJSON struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Href   string `json:"href"`
	Active bool   `json:"active,omitempty"`
}

// sidebarFeedsDataJSON is the top-level JSON structure embedded in the page.
type sidebarFeedsDataJSON struct {
	Feeds             []sidebarFeedJSON `json:"feeds"`
	RotationFeedSlugs []string          `json:"rotationFeedSlugs"`
	CurrentFeedIndex  int               `json:"currentFeedIndex"`
	CurrentPostSlug   string            `json:"currentPostSlug"`
}

// getAllCandidateFeeds collects all feeds that contain this post across all
// 4 priority levels. Unlike getFeedSidebarPosts which returns the first match,
// this collects ALL matches so the user can cycle between them with {/}.
func (p *TemplatesPlugin) getAllCandidateFeeds(
	post *models.Post, config *lifecycle.Config, m *lifecycle.Manager, postFormats models.PostFormatsConfig,
) []sidebarFeedJSON {
	if post == nil {
		return nil
	}

	syndication := getSyndicationConfig(config)
	seen := make(map[string]bool)
	var feeds []sidebarFeedJSON

	addFeed := func(fc *models.FeedConfig, posts []*models.Post, priority string) {
		if fc == nil || fc.IncludePrivate || seen[fc.Slug] {
			return
		}
		seen[fc.Slug] = true
		feeds = append(feeds, p.buildSidebarFeedEntry(post, fc, posts, priority, syndication, postFormats))
	}

	// 1. Series
	seriesPosts, seriesFeed := p.getSeriesSidebarPosts(post, config, m)
	if seriesPosts != nil {
		addFeed(seriesFeed, seriesPosts, "series")
	}

	// 2. Explicit frontmatter
	if explicitSlug := p.getExplicitFeedSlug(post); explicitSlug != "" {
		explicitPosts, explicitFc := p.feedFromCachedConfigs(explicitSlug, post, m)
		if explicitPosts != nil {
			addFeed(explicitFc, explicitPosts, "explicit")
		}
	}

	// 3. Tag-based feeds -- check ALL tag feeds, not just the first match
	p.collectTagFeeds(post, config, m, addFeed)

	// 4. Auto-discovery -- include matching feeds, sorted by size (smallest first)
	p.collectAutoDiscoveredFeeds(post, config, m, seen, addFeed)

	return feeds
}

// buildSidebarFeedEntry constructs a sidebarFeedJSON for a single feed,
// windowing the post list around the current post for large feeds.
func (p *TemplatesPlugin) buildSidebarFeedEntry(
	currentPost *models.Post, fc *models.FeedConfig,
	posts []*models.Post, priority string, syndication models.SyndicationConfig, postFormats models.PostFormatsConfig,
) sidebarFeedJSON {
	const maxWindowPosts = 50

	prev, next := p.getSidebarPrevNext(currentPost, posts)

	// Find current post position for windowing
	currentPos := -1
	for i, fp := range posts {
		if fp.Slug == currentPost.Slug {
			currentPos = i
			break
		}
	}

	// Window the posts if the feed is large
	windowedPosts := posts
	if len(posts) > maxWindowPosts && currentPos >= 0 {
		half := maxWindowPosts / 2
		start := currentPos - half
		end := currentPos + half + 1
		if start < 0 {
			end -= start
			start = 0
		}
		if end > len(posts) {
			start -= end - len(posts)
			end = len(posts)
		}
		if start < 0 {
			start = 0
		}
		windowedPosts = posts[start:end]
	}

	feed := sidebarFeedJSON{
		Slug:                fc.Slug,
		Title:               fc.Title,
		Priority:            priority,
		Primary:             fc.Primary,
		ContainsCurrentPost: currentPos >= 0,
		Variants:            buildSidebarVariants(fc, syndication, postFormats),
		TotalPosts:          len(posts),
		Posts:               make([]sidebarPostJSON, 0, len(windowedPosts)),
	}

	for _, fp := range windowedPosts {
		feed.Posts = append(feed.Posts, postToSidebarJSON(fp, fp.Slug == currentPost.Slug, fc.Slug))
	}

	if prev != nil {
		pj := postToSidebarJSON(prev, false, fc.Slug)
		feed.Prev = &pj
	}
	if next != nil {
		nj := postToSidebarJSON(next, false, fc.Slug)
		feed.Next = &nj
	}

	return feed
}

func buildSidebarVariants(fc *models.FeedConfig, syndication models.SyndicationConfig, postFormats models.PostFormatsConfig) []sidebarVariant {
	if fc == nil {
		return nil
	}

	baseHref := "/"
	if fc.Slug != "" {
		baseHref = "/" + strings.Trim(fc.Slug, "/") + "/"
	}

	variants := make([]sidebarVariant, 0, 8)
	add := func(key, label, href string) {
		variants = append(variants, sidebarVariant{Key: key, Label: label, Href: href})
	}
	canonicalVariantHref := func(ext string) string {
		if fc.Slug == "" {
			return "/index." + ext
		}
		return "/" + strings.Trim(fc.Slug, "/") + "." + ext
	}

	if fc.Formats.Markdown && postFormats.Markdown {
		add("md", "md", canonicalVariantHref("md"))
	}
	if fc.Formats.JSON {
		add("json", "json", baseHref+"feed.json")
	}
	if shouldGenerateFeedArchive(fc, syndication) && fc.Formats.RSS {
		add("archive-rss", "archive-rss", feedArchiveURL(fc.Slug, "rss.xml"))
	}
	if fc.Formats.RSS {
		add("rss", "rss", baseHref+"rss.xml")
	}
	if fc.Formats.Atom {
		add("atom", "atom", baseHref+"atom.xml")
	}
	if fc.Formats.HTML && postFormats.IsHTMLEnabled() {
		add("html", "html", baseHref)
	}
	if fc.Formats.SimpleHTML {
		add("simple", "simple", baseHref+"simple/")
	}
	if fc.Formats.Text && postFormats.Text {
		add("txt", "txt", canonicalVariantHref("txt"))
	}

	return variants
}

func appendFeedParamToHref(href, feedSlug string) string {
	if feedSlug == "" {
		return href
	}

	parsed, err := url.Parse(href)
	if err != nil {
		separator := "?"
		if strings.Contains(href, "?") {
			separator = "&"
		}
		return href + separator + "feed=" + url.QueryEscape(feedSlug)
	}

	query := parsed.Query()
	query.Set("feed", feedSlug)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// postToSidebarJSON converts a Post to a sidebarPostJSON.
func postToSidebarJSON(fp *models.Post, active bool, feedSlug string) sidebarPostJSON {
	title := fp.Slug
	if fp.Title != nil {
		title = *fp.Title
	}
	return sidebarPostJSON{
		Slug:   fp.Slug,
		Title:  title,
		Href:   appendFeedParamToHref(fp.Href, feedSlug),
		Active: active,
	}
}

// collectTagFeeds finds all tag-based feeds from the sidebar config that
// contain the given post and calls addFeed for each.
func (p *TemplatesPlugin) collectTagFeeds(
	post *models.Post, config *lifecycle.Config, m *lifecycle.Manager,
	addFeed func(*models.FeedConfig, []*models.Post, string),
) {
	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok {
		return
	}
	if components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return
	}

	for _, feedSlug := range components.FeedSidebar.Feeds {
		if !strings.HasPrefix(feedSlug, "tags/") {
			continue
		}
		tagName := strings.TrimPrefix(feedSlug, "tags/")
		if !postHasTag(post, tagName) {
			continue
		}
		tagPosts := filterPostsByTag(m.Posts(), tagName)
		sortPostsByDate(tagPosts, true)
		tagFc := &models.FeedConfig{
			Slug:  feedSlug,
			Title: fmt.Sprintf("Posts tagged: %s", tagName),
			Posts: tagPosts,
		}
		addFeed(tagFc, tagPosts, "tag")
	}
}

// postHasTag returns true if the post has the given tag.
func postHasTag(post *models.Post, tag string) bool {
	for _, t := range post.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// filterPostsByTag returns all published, non-draft, non-skip posts with the given tag.
func filterPostsByTag(allPosts []*models.Post, tag string) []*models.Post {
	var result []*models.Post
	for _, fp := range allPosts {
		if !fp.Published || fp.Draft || fp.Skip {
			continue
		}
		for _, t := range fp.Tags {
			if t == tag {
				result = append(result, fp)
				break
			}
		}
	}
	return result
}

// collectAutoDiscoveredFeeds finds feeds from the cache that contain the given
// post, sorted by size (smallest first), skipping catch-all feeds. Results are
// capped at maxAutoFeeds.
func (p *TemplatesPlugin) collectAutoDiscoveredFeeds(
	post *models.Post, config *lifecycle.Config, m *lifecycle.Manager,
	seen map[string]bool,
	addFeed func(*models.FeedConfig, []*models.Post, string),
) {
	const maxAutoFeedPosts = 5000
	const maxAutoFeeds = 10

	components, ok := config.Extra["components"].(models.ComponentsConfig)
	if !ok || components.FeedSidebar.Enabled == nil || !*components.FeedSidebar.Enabled {
		return
	}

	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		return
	}
	configs, ok := cached.([]models.FeedConfig)
	if !ok {
		return
	}

	skipSlugs := map[string]bool{
		"archive": true, "all": true, "sitemap": true, "search": true,
	}
	skipPrefixes := []string{"subscription-", "tags/"}

	type autoCandidate struct {
		fc    *models.FeedConfig
		count int
	}
	var candidates []autoCandidate

	for i := range configs {
		fc := &configs[i]
		if skipSlugs[fc.Slug] || seen[fc.Slug] || len(fc.Posts) > maxAutoFeedPosts {
			continue
		}
		if hasAnyPrefix(fc.Slug, skipPrefixes) {
			continue
		}
		for _, fp := range fc.Posts {
			if fp.Slug == post.Slug {
				candidates = append(candidates, autoCandidate{fc: fc, count: len(fc.Posts)})
				break
			}
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].count < candidates[j].count
	})

	if len(candidates) > maxAutoFeeds {
		candidates = candidates[:maxAutoFeeds]
	}

	for _, ac := range candidates {
		addFeed(ac.fc, ac.fc.Posts, "auto")
	}
}

// hasAnyPrefix returns true if s starts with any of the given prefixes.
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// buildSidebarFeedsJSON builds the JSON string for all candidate feeds,
// for embedding in the page for client-side feed cycling.
func (p *TemplatesPlugin) buildSidebarFeedsJSON(
	post *models.Post, config *lifecycle.Config, m *lifecycle.Manager,
	primaryFeed *models.FeedConfig,
) string {
	postFormats := resolvePostFormats(post, config)
	feeds := p.getAllCandidateFeeds(post, config, m, postFormats)
	feeds = filterPublicSidebarFeeds(feeds)
	seen := makeSidebarFeedSet(feeds)
	feeds = p.appendMissingPrimarySidebarFeeds(feeds, seen, post, config, m, postFormats)

	if len(feeds) <= 1 {
		// No point in cycling with only 0 or 1 feed
		return ""
	}

	rotationFeedSlugs := p.buildRotationFeedSlugs(m, seen)

	// Find the index of the primary (currently displayed) feed
	currentIndex := 0
	if primaryFeed != nil {
		for i := range feeds {
			if feeds[i].Slug == primaryFeed.Slug {
				currentIndex = i
				break
			}
		}
	}

	data := sidebarFeedsDataJSON{
		Feeds:             feeds,
		RotationFeedSlugs: rotationFeedSlugs,
		CurrentFeedIndex:  currentIndex,
		CurrentPostSlug:   post.Slug,
	}

	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}

func filterPublicSidebarFeeds(feeds []sidebarFeedJSON) []sidebarFeedJSON {
	publicFeeds := feeds[:0]
	for i := range feeds {
		if feeds[i].Priority == "private" {
			continue
		}
		publicFeeds = append(publicFeeds, feeds[i])
	}
	return publicFeeds
}

func makeSidebarFeedSet(feeds []sidebarFeedJSON) map[string]bool {
	seen := make(map[string]bool, len(feeds))
	for i := range feeds {
		seen[feeds[i].Slug] = true
	}
	return seen
}

func (p *TemplatesPlugin) appendMissingPrimarySidebarFeeds(
	feeds []sidebarFeedJSON, seen map[string]bool, post *models.Post,
	config *lifecycle.Config, m *lifecycle.Manager, postFormats models.PostFormatsConfig,
) []sidebarFeedJSON {
	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		return feeds
	}
	configs, ok := cached.([]models.FeedConfig)
	if !ok {
		return feeds
	}

	syndication := getSyndicationConfig(config)
	for i := range configs {
		fc := &configs[i]
		if seen[fc.Slug] || !fc.Primary || fc.IncludePrivate {
			continue
		}
		feeds = append(feeds, p.buildSidebarFeedEntry(post, fc, fc.Posts, "primary", syndication, postFormats))
		seen[fc.Slug] = true
	}

	return feeds
}

func (p *TemplatesPlugin) buildRotationFeedSlugs(m *lifecycle.Manager, seen map[string]bool) []string {
	rotationFeedSlugs := make([]string, 0, len(seen))
	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		return rotationFeedSlugs
	}
	configs, ok := cached.([]models.FeedConfig)
	if !ok {
		return rotationFeedSlugs
	}

	for i := range configs {
		fc := &configs[i]
		if !fc.Primary || fc.IncludePrivate || !seen[fc.Slug] {
			continue
		}
		rotationFeedSlugs = append(rotationFeedSlugs, fc.Slug)
	}

	return rotationFeedSlugs
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
// Includes all format variants (.txt, .ansi, .md, .og) and excludes the robots post itself.
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
			// For regular posts: /slug.txt, /slug.ansi, /slug.md, /slug.og/
			if post.Slug != "" {
				paths = append(paths,
					"/"+post.Slug+".txt",
					"/"+post.Slug+".ansi",
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
		Templates:    models.NewTemplatesConfig(),
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

	// Copy feeds page config if available
	modelsConfig.FeedsPage = feedsPageConfigFromExtra(config.Extra["feeds_page"])

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

	if viewTransitions, ok := config.Extra["view_transitions"].(models.ViewTransitionsConfig); ok {
		modelsConfig.ViewTransitions = viewTransitions
	} else {
		modelsConfig.ViewTransitions = models.NewViewTransitionsConfig()
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
	if templatesCfg, ok := config.Extra["templates"].(models.TemplatesConfig); ok {
		modelsConfig.Templates = templatesCfg
	} else if templatesCfgPtr, ok := config.Extra["templates"].(*models.TemplatesConfig); ok && templatesCfgPtr != nil {
		modelsConfig.Templates = *templatesCfgPtr
	}

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

	// Copy Theme config if available
	if theme, ok := config.Extra["theme"].(models.ThemeConfig); ok {
		modelsConfig.Theme = theme
	}
	if theme, ok := config.Extra["theme"].(map[string]interface{}); ok {
		modelsConfig.Theme = themeFromMap(theme, modelsConfig.Theme)
	}
	// Copy Head config if available
	if head, ok := config.Extra["head"].(models.HeadConfig); ok {
		modelsConfig.Head = head
	}
	if headMap, ok := config.Extra["head"].(map[string]interface{}); ok {
		modelsConfig.Head = headFromMap(headMap, modelsConfig.Head)
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

func themeFromMap(theme map[string]interface{}, fallback models.ThemeConfig) models.ThemeConfig {
	result := fallback
	if theme == nil {
		return result
	}
	if name, ok := theme["name"].(string); ok && name != "" {
		result.Name = name
	}
	if palette, ok := theme["palette"].(string); ok && palette != "" {
		result.Palette = palette
	}
	if paletteLight, ok := theme["palette_light"].(string); ok && paletteLight != "" {
		result.PaletteLight = paletteLight
	}
	if paletteDark, ok := theme["palette_dark"].(string); ok && paletteDark != "" {
		result.PaletteDark = paletteDark
	}
	if customCSS, ok := theme["custom_css"].(string); ok {
		result.CustomCSS = customCSS
	}
	switch variables := theme["variables"].(type) {
	case map[string]string:
		result.Variables = variables
	case map[string]interface{}:
		result.Variables = make(map[string]string, len(variables))
		for key, value := range variables {
			if v, ok := value.(string); ok {
				result.Variables[key] = v
			}
		}
	}
	return result
}

func headFromMap(head map[string]interface{}, fallback models.HeadConfig) models.HeadConfig {
	result := fallback
	if head == nil {
		return result
	}
	if text, ok := head["text"].(string); ok {
		result.Text = text
	}
	result.Meta = metaTagsFromValue(head["meta"], result.Meta)
	result.Link = linkTagsFromValue(head["link"], result.Link)
	result.Script = scriptTagsFromValue(head["script"], result.Script)
	return result
}

func metaTagsFromValue(value interface{}, fallback []models.MetaTag) []models.MetaTag {
	switch meta := value.(type) {
	case []models.MetaTag:
		return meta
	case []interface{}:
		result := make([]models.MetaTag, 0, len(meta))
		for _, entry := range meta {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			tag := models.MetaTag{}
			if name, ok := m["name"].(string); ok {
				tag.Name = name
			}
			if property, ok := m["property"].(string); ok {
				tag.Property = property
			}
			if content, ok := m["content"].(string); ok {
				tag.Content = content
			}
			result = append(result, tag)
		}
		return result
	default:
		return fallback
	}
}

func linkTagsFromValue(value interface{}, fallback []models.LinkTag) []models.LinkTag {
	switch links := value.(type) {
	case []models.LinkTag:
		return links
	case []interface{}:
		result := make([]models.LinkTag, 0, len(links))
		for _, entry := range links {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			tag := models.LinkTag{}
			if rel, ok := m["rel"].(string); ok {
				tag.Rel = rel
			}
			if href, ok := m["href"].(string); ok {
				tag.Href = href
			}
			if crossorigin, ok := m["crossorigin"].(bool); ok {
				tag.Crossorigin = crossorigin
			}
			result = append(result, tag)
		}
		return result
	default:
		return fallback
	}
}

func scriptTagsFromValue(value interface{}, fallback []models.ScriptTag) []models.ScriptTag {
	switch scripts := value.(type) {
	case []models.ScriptTag:
		return scripts
	case []interface{}:
		result := make([]models.ScriptTag, 0, len(scripts))
		for _, entry := range scripts {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			tag := models.ScriptTag{}
			if src, ok := m["src"].(string); ok {
				tag.Src = src
			}
			result = append(result, tag)
		}
		return result
	default:
		return fallback
	}
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

func feedsPageConfigFromExtra(raw interface{}) models.FeedsPageConfig {
	defaults := models.NewFeedsPageConfig()

	switch feedsPageVal := raw.(type) {
	case models.FeedsPageConfig:
		return feedsPageVal
	case map[string]interface{}:
		feedsPage := models.FeedsPageConfig{
			Enabled:     defaults.Enabled,
			Title:       getStringFromMap(feedsPageVal, "title"),
			Description: getStringFromMap(feedsPageVal, "description"),
			Template:    getStringFromMap(feedsPageVal, "template"),
			SlugPrefix:  getStringFromMap(feedsPageVal, "slug_prefix"),
			Robots:      getStringFromMap(feedsPageVal, "robots"),
		}
		if rawShowPrivate, ok := feedsPageVal["show_private_feeds"].([]interface{}); ok {
			feedsPage.ShowPrivateFeeds = make([]string, 0, len(rawShowPrivate))
			for _, value := range rawShowPrivate {
				if s, ok := value.(string); ok {
					feedsPage.ShowPrivateFeeds = append(feedsPage.ShowPrivateFeeds, s)
				}
			}
		}
		if rawShowPrivate, ok := feedsPageVal["show_private_feeds"].([]string); ok {
			feedsPage.ShowPrivateFeeds = append([]string{}, rawShowPrivate...)
		}
		if enabled, ok := feedsPageVal["enabled"].(bool); ok {
			feedsPage.Enabled = &enabled
		}
		if feedsPage.Title == "" {
			feedsPage.Title = defaults.Title
		}
		if feedsPage.Description == "" {
			feedsPage.Description = defaults.Description
		}
		if feedsPage.Template == "" {
			feedsPage.Template = defaults.Template
		}
		if feedsPage.SlugPrefix == "" {
			feedsPage.SlugPrefix = defaults.SlugPrefix
		}
		return feedsPage
	default:
		return defaults
	}
}
