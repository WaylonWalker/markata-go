package config

import (
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// MergeConfigs merges override configuration into base configuration.
// The override values take precedence over base values.
// For nested objects, a deep merge is performed.
// Arrays replace by default (use *_append fields for appending).
func MergeConfigs(base, override *models.Config) *models.Config {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &models.Config{}
	*result = *base

	// String fields - override if non-empty
	if override.OutputDir != "" {
		result.OutputDir = override.OutputDir
	}
	if override.URL != "" {
		result.URL = override.URL
	}
	if override.Title != "" {
		result.Title = override.Title
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Author != "" {
		result.Author = override.Author
	}
	if override.AssetsDir != "" {
		result.AssetsDir = override.AssetsDir
	}
	if override.TemplatesDir != "" {
		result.TemplatesDir = override.TemplatesDir
	}

	// Slice fields - replace if non-nil and non-empty
	if len(override.Hooks) > 0 {
		result.Hooks = override.Hooks
	}
	if len(override.DisabledHooks) > 0 {
		result.DisabledHooks = override.DisabledHooks
	}
	if len(override.Nav) > 0 {
		result.Nav = override.Nav
	}

	// Integer fields - override if non-zero
	if override.Concurrency != 0 {
		result.Concurrency = override.Concurrency
	}

	// Nested structs - deep merge
	result.GlobConfig = mergeGlobConfig(base.GlobConfig, override.GlobConfig)
	result.MarkdownConfig = mergeMarkdownConfig(base.MarkdownConfig, override.MarkdownConfig)
	result.FeedDefaults = mergeFeedDefaults(base.FeedDefaults, override.FeedDefaults)
	result.Footer = mergeFooterConfig(base.Footer, override.Footer)

	// Feeds array - replace if non-empty
	if len(override.Feeds) > 0 {
		result.Feeds = override.Feeds
	}

	// Theme - merge if override has any non-empty values
	result.Theme = mergeThemeConfig(base.Theme, override.Theme)

	// PostFormats - merge if override has any formats enabled
	result.PostFormats = mergePostFormatsConfig(base.PostFormats, override.PostFormats)

	// WebSub - merge
	result.WebSub = mergeWebSubConfig(base.WebSub, override.WebSub)

	// WellKnown - merge
	result.WellKnown = mergeWellKnownConfig(base.WellKnown, override.WellKnown)

	// SEO - merge
	result.SEO = mergeSEOConfig(base.SEO, override.SEO)

	// Components - merge
	result.Components = mergeComponentsConfig(base.Components, override.Components)

	// Layout - merge
	result.Layout = mergeLayoutConfig(base.Layout, override.Layout)

	// Sidebar - merge
	result.Sidebar = mergeSidebarConfig(base.Sidebar, override.Sidebar)

	// Toc - merge
	result.Toc = mergeTocConfig(base.Toc, override.Toc)

	// Header - merge
	result.Header = mergeHeaderLayoutConfig(base.Header, override.Header)

	// Blogroll - merge
	result.Blogroll = mergeBlogrollConfig(base.Blogroll, override.Blogroll)

	// Encryption - merge
	result.Encryption = mergeEncryptionConfig(base.Encryption, override.Encryption)

	// TagAggregator - merge
	result.TagAggregator = mergeTagAggregatorConfig(base.TagAggregator, override.TagAggregator)

	// Mentions - merge
	result.Mentions = mergeMentionsConfig(base.Mentions, override.Mentions)

	// Extra (plugin configs) - merge
	result.Extra = mergeExtra(base.Extra, override.Extra)

	return result
}

// mergeThemeConfig merges ThemeConfig values.
func mergeThemeConfig(base, override models.ThemeConfig) models.ThemeConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Palette != "" {
		result.Palette = override.Palette
	}
	if override.CustomCSS != "" {
		result.CustomCSS = override.CustomCSS
	}
	// Merge variables maps
	if len(override.Variables) > 0 {
		if result.Variables == nil {
			result.Variables = make(map[string]string)
		}
		for k, v := range override.Variables {
			result.Variables[k] = v
		}
	}

	// Merge background config
	result.Background = mergeBackgroundConfig(base.Background, override.Background)

	// Merge font config
	result.Font = mergeFontConfig(base.Font, override.Font)

	// Merge switcher config
	result.Switcher = mergeSwitcherConfig(base.Switcher, override.Switcher)

	return result
}

// mergeSwitcherConfig merges ThemeSwitcherConfig, preferring override values.
func mergeSwitcherConfig(base, override models.ThemeSwitcherConfig) models.ThemeSwitcherConfig {
	result := base

	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.IncludeAll != nil {
		result.IncludeAll = override.IncludeAll
	}
	if len(override.Include) > 0 {
		result.Include = override.Include
	}
	if len(override.Exclude) > 0 {
		result.Exclude = override.Exclude
	}
	if override.Position != "" {
		result.Position = override.Position
	}

	return result
}

// mergeBackgroundConfig merges BackgroundConfig, preferring override values.
func mergeBackgroundConfig(base, override models.BackgroundConfig) models.BackgroundConfig {
	result := base

	// Override enabled if explicitly set
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}

	// Replace backgrounds array if non-empty
	if len(override.Backgrounds) > 0 {
		result.Backgrounds = override.Backgrounds
	}

	// Replace scripts array if non-empty
	if len(override.Scripts) > 0 {
		result.Scripts = override.Scripts
	}

	// Override CSS if non-empty
	if override.CSS != "" {
		result.CSS = override.CSS
	}

	// Article styling fields - override if non-empty
	if override.ArticleBg != "" {
		result.ArticleBg = override.ArticleBg
	}
	if override.ArticleBlurEnabled != nil {
		result.ArticleBlurEnabled = override.ArticleBlurEnabled
	}
	if override.ArticleBlur != "" {
		result.ArticleBlur = override.ArticleBlur
	}
	if override.ArticleShadow != "" {
		result.ArticleShadow = override.ArticleShadow
	}
	if override.ArticleBorder != "" {
		result.ArticleBorder = override.ArticleBorder
	}
	if override.ArticleRadius != "" {
		result.ArticleRadius = override.ArticleRadius
	}

	return result
}

// mergeFontConfig merges FontConfig, preferring override values.
func mergeFontConfig(base, override models.FontConfig) models.FontConfig {
	result := base

	if override.Family != "" {
		result.Family = override.Family
	}
	if override.HeadingFamily != "" {
		result.HeadingFamily = override.HeadingFamily
	}
	if override.CodeFamily != "" {
		result.CodeFamily = override.CodeFamily
	}
	if override.Size != "" {
		result.Size = override.Size
	}
	if override.LineHeight != "" {
		result.LineHeight = override.LineHeight
	}
	if len(override.GoogleFonts) > 0 {
		result.GoogleFonts = override.GoogleFonts
	}
	if len(override.CustomURLs) > 0 {
		result.CustomURLs = override.CustomURLs
	}

	return result
}

// mergePostFormatsConfig merges PostFormatsConfig, preferring override values.
func mergePostFormatsConfig(base, override models.PostFormatsConfig) models.PostFormatsConfig {
	result := base

	// HTML uses pointer, so check if override has it set
	if override.HTML != nil {
		result.HTML = override.HTML
	}

	// For bool fields, override if true (since default is false)
	if override.Markdown {
		result.Markdown = true
	}
	if override.Text {
		result.Text = true
	}
	if override.OG {
		result.OG = true
	}

	return result
}

// mergeWebSubConfig merges WebSubConfig values.
func mergeWebSubConfig(base, override models.WebSubConfig) models.WebSubConfig {
	result := base

	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.Hubs != nil {
		result.Hubs = append([]string{}, override.Hubs...)
	}

	return result
}

// mergeWellKnownConfig merges WellKnownConfig values.
func mergeWellKnownConfig(base, override models.WellKnownConfig) models.WellKnownConfig {
	result := base

	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.AutoGenerate != nil {
		result.AutoGenerate = append([]string{}, override.AutoGenerate...)
	}
	if override.SSHFingerprint != "" {
		result.SSHFingerprint = override.SSHFingerprint
	}
	if override.KeybaseUsername != "" {
		result.KeybaseUsername = override.KeybaseUsername
	}

	return result
}

// mergeFooterConfig merges FooterConfig values.
func mergeFooterConfig(base, override models.FooterConfig) models.FooterConfig {
	result := base

	if override.Text != "" {
		result.Text = override.Text
	}
	if override.ShowCopyright != nil {
		result.ShowCopyright = override.ShowCopyright
	}

	return result
}

// mergeGlobConfig merges GlobConfig values.
func mergeGlobConfig(base, override models.GlobConfig) models.GlobConfig {
	result := base

	if len(override.Patterns) > 0 {
		result.Patterns = override.Patterns
	}

	// UseGitignore is a bool, always use override value if it differs from zero value
	// Since we can't distinguish between "explicitly set to false" and "not set",
	// we always take the override value
	result.UseGitignore = override.UseGitignore

	return result
}

// mergeMarkdownConfig merges MarkdownConfig values.
func mergeMarkdownConfig(base, override models.MarkdownConfig) models.MarkdownConfig {
	result := base

	if len(override.Extensions) > 0 {
		result.Extensions = override.Extensions
	}

	return result
}

// mergeFeedDefaults merges FeedDefaults values.
func mergeFeedDefaults(base, override models.FeedDefaults) models.FeedDefaults {
	result := base

	if override.ItemsPerPage != 0 {
		result.ItemsPerPage = override.ItemsPerPage
	}
	if override.OrphanThreshold != 0 {
		result.OrphanThreshold = override.OrphanThreshold
	}

	result.Formats = mergeFeedFormats(base.Formats, override.Formats)
	result.Templates = mergeFeedTemplates(base.Templates, override.Templates)
	result.Syndication = mergeSyndicationConfig(base.Syndication, override.Syndication)

	return result
}

// mergeFeedFormats merges FeedFormats values.
// If the override has any format enabled, it replaces the base entirely.
// This allows explicitly disabling formats by setting only the desired ones.
func mergeFeedFormats(base, override models.FeedFormats) models.FeedFormats {
	// Check if override has any format set to true
	if override.HTML || override.SimpleHTML || override.RSS || override.Atom || override.JSON || override.Markdown || override.Text || override.Sitemap {
		// Override is "active" - use it entirely
		return override
	}
	// Override has no formats enabled, keep base
	return base
}

// mergeFeedTemplates merges FeedTemplates values.
func mergeFeedTemplates(base, override models.FeedTemplates) models.FeedTemplates {
	result := base

	if override.HTML != "" {
		result.HTML = override.HTML
	}
	if override.SimpleHTML != "" {
		result.SimpleHTML = override.SimpleHTML
	}
	if override.RSS != "" {
		result.RSS = override.RSS
	}
	if override.Atom != "" {
		result.Atom = override.Atom
	}
	if override.JSON != "" {
		result.JSON = override.JSON
	}
	if override.Card != "" {
		result.Card = override.Card
	}

	return result
}

// mergeSyndicationConfig merges SyndicationConfig values.
func mergeSyndicationConfig(base, override models.SyndicationConfig) models.SyndicationConfig {
	result := base

	if override.MaxItems != 0 {
		result.MaxItems = override.MaxItems
	}
	// For IncludeContent, we take the override value
	result.IncludeContent = override.IncludeContent || base.IncludeContent

	return result
}

// MergeSlice merges two slices with optional append behavior.
// If append is true, the override slice is appended to the base slice.
// Otherwise, the override slice replaces the base slice.
func MergeSlice[T any](base, override []T, appendMode bool) []T {
	if len(override) == 0 {
		return base
	}
	if appendMode {
		result := make([]T, len(base)+len(override))
		copy(result, base)
		copy(result[len(base):], override)
		return result
	}
	return override
}

// mergeEncryptionConfig merges EncryptionConfig values.
func mergeEncryptionConfig(base, override models.EncryptionConfig) models.EncryptionConfig {
	result := base

	// Override enabled if explicitly set (even if false)
	// We check if the override has any non-default values to determine if it was explicitly configured
	if override.Enabled || override.DefaultKey != "" || override.DecryptionHint != "" {
		result.Enabled = override.Enabled
		if override.DefaultKey != "" {
			result.DefaultKey = override.DefaultKey
		}
		if override.DecryptionHint != "" {
			result.DecryptionHint = override.DecryptionHint
		}
	}

	return result
}

// mergeExtra merges Extra map values (for plugin configs like image_zoom, wikilinks, etc.)// AppendHooks appends hooks to the configuration's Hooks slice.
func AppendHooks(config *models.Config, hooks ...string) {
	config.Hooks = MergeSlice(config.Hooks, hooks, true)
}

// AppendDisabledHooks appends hooks to the configuration's DisabledHooks slice.
func AppendDisabledHooks(config *models.Config, hooks ...string) {
	config.DisabledHooks = MergeSlice(config.DisabledHooks, hooks, true)
}

// AppendGlobPatterns appends patterns to the configuration's GlobConfig.Patterns slice.
func AppendGlobPatterns(config *models.Config, patterns ...string) {
	config.GlobConfig.Patterns = MergeSlice(config.GlobConfig.Patterns, patterns, true)
}

// AppendFeeds appends feeds to the configuration's Feeds slice.
func AppendFeeds(config *models.Config, feeds ...models.FeedConfig) {
	config.Feeds = MergeSlice(config.Feeds, feeds, true)
}

// mergeSEOConfig merges SEOConfig values.
func mergeSEOConfig(base, override models.SEOConfig) models.SEOConfig {
	result := base

	if override.TwitterHandle != "" {
		result.TwitterHandle = override.TwitterHandle
	}
	if override.DefaultImage != "" {
		result.DefaultImage = override.DefaultImage
	}
	if override.LogoURL != "" {
		result.LogoURL = override.LogoURL
	}
	if override.AuthorImage != "" {
		result.AuthorImage = override.AuthorImage
	}
	if override.OGImageService != "" {
		result.OGImageService = override.OGImageService
	}

	// Merge StructuredData config (pointer fields)
	if override.StructuredData.Enabled != nil {
		result.StructuredData.Enabled = override.StructuredData.Enabled
	}
	if override.StructuredData.Publisher != nil {
		result.StructuredData.Publisher = override.StructuredData.Publisher
	}
	if override.StructuredData.DefaultAuthor != nil {
		result.StructuredData.DefaultAuthor = override.StructuredData.DefaultAuthor
	}

	return result
}

// mergeComponentsConfig merges ComponentsConfig values.
//
//nolint:gocyclo // This function merges many component fields; complexity is inherent
func mergeComponentsConfig(base, override models.ComponentsConfig) models.ComponentsConfig {
	result := base

	// Merge Nav component
	if override.Nav.Enabled != nil {
		result.Nav.Enabled = override.Nav.Enabled
	}
	if override.Nav.Position != "" {
		result.Nav.Position = override.Nav.Position
	}
	if override.Nav.Style != "" {
		result.Nav.Style = override.Nav.Style
	}
	if len(override.Nav.Items) > 0 {
		result.Nav.Items = override.Nav.Items
	}

	// Merge Footer component
	if override.Footer.Enabled != nil {
		result.Footer.Enabled = override.Footer.Enabled
	}
	if override.Footer.Text != "" {
		result.Footer.Text = override.Footer.Text
	}
	if override.Footer.ShowCopyright != nil {
		result.Footer.ShowCopyright = override.Footer.ShowCopyright
	}
	if len(override.Footer.Links) > 0 {
		result.Footer.Links = override.Footer.Links
	}

	// Merge DocSidebar component
	if override.DocSidebar.Enabled != nil {
		result.DocSidebar.Enabled = override.DocSidebar.Enabled
	}
	if override.DocSidebar.Position != "" {
		result.DocSidebar.Position = override.DocSidebar.Position
	}
	if override.DocSidebar.Width != "" {
		result.DocSidebar.Width = override.DocSidebar.Width
	}
	if override.DocSidebar.MinDepth != 0 {
		result.DocSidebar.MinDepth = override.DocSidebar.MinDepth
	}
	if override.DocSidebar.MaxDepth != 0 {
		result.DocSidebar.MaxDepth = override.DocSidebar.MaxDepth
	}

	// Merge FeedSidebar component
	if override.FeedSidebar.Enabled != nil {
		result.FeedSidebar.Enabled = override.FeedSidebar.Enabled
	}
	if override.FeedSidebar.Position != "" {
		result.FeedSidebar.Position = override.FeedSidebar.Position
	}
	if override.FeedSidebar.Width != "" {
		result.FeedSidebar.Width = override.FeedSidebar.Width
	}
	if override.FeedSidebar.Title != "" {
		result.FeedSidebar.Title = override.FeedSidebar.Title
	}
	if len(override.FeedSidebar.Feeds) > 0 {
		result.FeedSidebar.Feeds = override.FeedSidebar.Feeds
	}

	// Merge CardRouter component - mappings from override take precedence
	if len(override.CardRouter.Mappings) > 0 {
		if result.CardRouter.Mappings == nil {
			result.CardRouter.Mappings = make(map[string]string)
		}
		for k, v := range override.CardRouter.Mappings {
			result.CardRouter.Mappings[k] = v
		}
	}

	return result
}

// mergeLayoutConfig merges LayoutConfig values.
func mergeLayoutConfig(base, override models.LayoutConfig) models.LayoutConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if len(override.Paths) > 0 {
		if result.Paths == nil {
			result.Paths = make(map[string]string)
		}
		for k, v := range override.Paths {
			result.Paths[k] = v
		}
	}
	if len(override.Feeds) > 0 {
		if result.Feeds == nil {
			result.Feeds = make(map[string]string)
		}
		for k, v := range override.Feeds {
			result.Feeds[k] = v
		}
	}

	// Merge nested layout configs
	result.Docs = mergeDocsLayoutConfig(base.Docs, override.Docs)
	result.Blog = mergeBlogLayoutConfig(base.Blog, override.Blog)
	result.Landing = mergeLandingLayoutConfig(base.Landing, override.Landing)
	result.Bare = mergeBareLayoutConfig(base.Bare, override.Bare)
	result.Defaults = mergeLayoutDefaults(base.Defaults, override.Defaults)

	return result
}

// mergeDocsLayoutConfig merges DocsLayoutConfig values.
func mergeDocsLayoutConfig(base, override models.DocsLayoutConfig) models.DocsLayoutConfig {
	result := base

	if override.SidebarPosition != "" {
		result.SidebarPosition = override.SidebarPosition
	}
	if override.SidebarWidth != "" {
		result.SidebarWidth = override.SidebarWidth
	}
	if override.SidebarCollapsible != nil {
		result.SidebarCollapsible = override.SidebarCollapsible
	}
	if override.SidebarDefaultOpen != nil {
		result.SidebarDefaultOpen = override.SidebarDefaultOpen
	}
	if override.TocPosition != "" {
		result.TocPosition = override.TocPosition
	}
	if override.TocWidth != "" {
		result.TocWidth = override.TocWidth
	}
	if override.TocCollapsible != nil {
		result.TocCollapsible = override.TocCollapsible
	}
	if override.TocDefaultOpen != nil {
		result.TocDefaultOpen = override.TocDefaultOpen
	}
	if override.ContentMaxWidth != "" {
		result.ContentMaxWidth = override.ContentMaxWidth
	}
	if override.HeaderStyle != "" {
		result.HeaderStyle = override.HeaderStyle
	}
	if override.FooterStyle != "" {
		result.FooterStyle = override.FooterStyle
	}

	return result
}

// mergeBlogLayoutConfig merges BlogLayoutConfig values.
func mergeBlogLayoutConfig(base, override models.BlogLayoutConfig) models.BlogLayoutConfig {
	result := base

	if override.ContentMaxWidth != "" {
		result.ContentMaxWidth = override.ContentMaxWidth
	}
	if override.ShowToc != nil {
		result.ShowToc = override.ShowToc
	}
	if override.TocPosition != "" {
		result.TocPosition = override.TocPosition
	}
	if override.TocWidth != "" {
		result.TocWidth = override.TocWidth
	}
	if override.HeaderStyle != "" {
		result.HeaderStyle = override.HeaderStyle
	}
	if override.FooterStyle != "" {
		result.FooterStyle = override.FooterStyle
	}
	if override.ShowAuthor != nil {
		result.ShowAuthor = override.ShowAuthor
	}
	if override.ShowDate != nil {
		result.ShowDate = override.ShowDate
	}
	if override.ShowTags != nil {
		result.ShowTags = override.ShowTags
	}
	if override.ShowReadingTime != nil {
		result.ShowReadingTime = override.ShowReadingTime
	}
	if override.ShowPrevNext != nil {
		result.ShowPrevNext = override.ShowPrevNext
	}

	return result
}

// mergeLandingLayoutConfig merges LandingLayoutConfig values.
func mergeLandingLayoutConfig(base, override models.LandingLayoutConfig) models.LandingLayoutConfig {
	result := base

	if override.ContentMaxWidth != "" {
		result.ContentMaxWidth = override.ContentMaxWidth
	}
	if override.HeaderStyle != "" {
		result.HeaderStyle = override.HeaderStyle
	}
	if override.HeaderSticky != nil {
		result.HeaderSticky = override.HeaderSticky
	}
	if override.FooterStyle != "" {
		result.FooterStyle = override.FooterStyle
	}
	if override.HeroEnabled != nil {
		result.HeroEnabled = override.HeroEnabled
	}

	return result
}

// mergeBareLayoutConfig merges BareLayoutConfig values.
func mergeBareLayoutConfig(base, override models.BareLayoutConfig) models.BareLayoutConfig {
	result := base

	if override.ContentMaxWidth != "" {
		result.ContentMaxWidth = override.ContentMaxWidth
	}

	return result
}

// mergeLayoutDefaults merges LayoutDefaults values.
func mergeLayoutDefaults(base, override models.LayoutDefaults) models.LayoutDefaults {
	result := base

	if override.ContentMaxWidth != "" {
		result.ContentMaxWidth = override.ContentMaxWidth
	}
	if override.HeaderSticky != nil {
		result.HeaderSticky = override.HeaderSticky
	}
	if override.FooterSticky != nil {
		result.FooterSticky = override.FooterSticky
	}

	return result
}

// mergeSidebarConfig merges SidebarConfig values.
func mergeSidebarConfig(base, override models.SidebarConfig) models.SidebarConfig {
	result := base

	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.Position != "" {
		result.Position = override.Position
	}
	if override.Width != "" {
		result.Width = override.Width
	}
	if override.Collapsible != nil {
		result.Collapsible = override.Collapsible
	}
	if override.DefaultOpen != nil {
		result.DefaultOpen = override.DefaultOpen
	}
	if len(override.Nav) > 0 {
		result.Nav = override.Nav
	}
	if override.Title != "" {
		result.Title = override.Title
	}

	return result
}

// mergeTocConfig merges TocConfig values.
func mergeTocConfig(base, override models.TocConfig) models.TocConfig {
	result := base

	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.Position != "" {
		result.Position = override.Position
	}
	if override.Width != "" {
		result.Width = override.Width
	}
	if override.MinDepth != 0 {
		result.MinDepth = override.MinDepth
	}
	if override.MaxDepth != 0 {
		result.MaxDepth = override.MaxDepth
	}
	if override.Collapsible != nil {
		result.Collapsible = override.Collapsible
	}
	if override.DefaultOpen != nil {
		result.DefaultOpen = override.DefaultOpen
	}
	if override.Title != "" {
		result.Title = override.Title
	}

	return result
}

// mergeHeaderLayoutConfig merges HeaderLayoutConfig values.
func mergeHeaderLayoutConfig(base, override models.HeaderLayoutConfig) models.HeaderLayoutConfig {
	result := base

	if override.Style != "" {
		result.Style = override.Style
	}
	if override.Sticky != nil {
		result.Sticky = override.Sticky
	}
	if override.ShowLogo != nil {
		result.ShowLogo = override.ShowLogo
	}
	if override.ShowTitle != nil {
		result.ShowTitle = override.ShowTitle
	}
	if override.ShowNav != nil {
		result.ShowNav = override.ShowNav
	}
	if override.ShowSearch != nil {
		result.ShowSearch = override.ShowSearch
	}
	if override.ShowThemeToggle != nil {
		result.ShowThemeToggle = override.ShowThemeToggle
	}

	return result
}

// mergeBlogrollConfig merges BlogrollConfig values.
func mergeBlogrollConfig(base, override models.BlogrollConfig) models.BlogrollConfig {
	result := base

	// Enabled - use override if true (since default is false)
	if override.Enabled {
		result.Enabled = true
	}

	// String fields - override if non-empty
	if override.BlogrollSlug != "" {
		result.BlogrollSlug = override.BlogrollSlug
	}
	if override.ReaderSlug != "" {
		result.ReaderSlug = override.ReaderSlug
	}
	if override.CacheDir != "" {
		result.CacheDir = override.CacheDir
	}
	if override.CacheDuration != "" {
		result.CacheDuration = override.CacheDuration
	}
	if override.FallbackImageService != "" {
		result.FallbackImageService = override.FallbackImageService
	}
	if override.PaginationType != "" {
		result.PaginationType = override.PaginationType
	}

	// Int fields - override if non-zero
	if override.Timeout != 0 {
		result.Timeout = override.Timeout
	}
	if override.ConcurrentRequests != 0 {
		result.ConcurrentRequests = override.ConcurrentRequests
	}
	if override.MaxEntriesPerFeed != 0 {
		result.MaxEntriesPerFeed = override.MaxEntriesPerFeed
	}
	if override.ItemsPerPage != 0 {
		result.ItemsPerPage = override.ItemsPerPage
	}
	if override.OrphanThreshold != 0 {
		result.OrphanThreshold = override.OrphanThreshold
	}

	// Feeds - replace if non-empty
	if len(override.Feeds) > 0 {
		result.Feeds = override.Feeds
	}

	// Templates - merge
	if override.Templates.Blogroll != "" {
		result.Templates.Blogroll = override.Templates.Blogroll
	}
	if override.Templates.Reader != "" {
		result.Templates.Reader = override.Templates.Reader
	}

	return result
}

// mergeExtra merges Extra map values (for plugin configs like image_zoom, wikilinks, etc.)
func mergeExtra(base, override map[string]any) map[string]any {
	if base == nil && override == nil {
		return nil
	}

	result := make(map[string]any)

	// Copy base values
	for k, v := range base {
		result[k] = v
	}

	// Override with values from override
	for k, v := range override {
		result[k] = v
	}

	return result
}

// mergeTagAggregatorConfig merges TagAggregatorConfig values.
func mergeTagAggregatorConfig(base, override models.TagAggregatorConfig) models.TagAggregatorConfig {
	result := base

	// Enabled - override if set
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}

	// Synonyms - merge maps (override values take precedence)
	if len(override.Synonyms) > 0 {
		if result.Synonyms == nil {
			result.Synonyms = make(map[string][]string)
		}
		for k, v := range override.Synonyms {
			result.Synonyms[k] = v
		}
	}

	// Additional - merge maps (override values take precedence)
	if len(override.Additional) > 0 {
		if result.Additional == nil {
			result.Additional = make(map[string][]string)
		}
		for k, v := range override.Additional {
			result.Additional[k] = v
		}
	}

	// GenerateReport - override if true
	if override.GenerateReport {
		result.GenerateReport = true
	}

	return result
}

// mergeMentionsConfig merges MentionsConfig values.
func mergeMentionsConfig(base, override models.MentionsConfig) models.MentionsConfig {
	result := base

	// Enabled - override if explicitly set
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}

	// CSSClass - override if set
	if override.CSSClass != "" {
		result.CSSClass = override.CSSClass
	}

	// FromPosts - override if set (don't merge, replace entirely)
	if len(override.FromPosts) > 0 {
		result.FromPosts = override.FromPosts
	}

	// CacheDir - override if set
	if override.CacheDir != "" {
		result.CacheDir = override.CacheDir
	}

	// CacheDuration - override if set
	if override.CacheDuration != "" {
		result.CacheDuration = override.CacheDuration
	}

	// Timeout - override if set
	if override.Timeout > 0 {
		result.Timeout = override.Timeout
	}

	// ConcurrentRequests - override if set
	if override.ConcurrentRequests > 0 {
		result.ConcurrentRequests = override.ConcurrentRequests
	}

	return result
}
