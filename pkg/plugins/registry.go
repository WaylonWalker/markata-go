// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// pluginRegistry holds all registered plugin constructors.
// The registry is initialized lazily via ensureRegistryInitialized().
var pluginRegistry = struct {
	sync.RWMutex
	constructors map[string]func() lifecycle.Plugin
	initialized  bool
}{
	constructors: make(map[string]func() lifecycle.Plugin),
}

// ensureRegistryInitialized registers all built-in plugins if not already done.
// This replaces the init() function to satisfy gochecknoinits linter.
func ensureRegistryInitialized() {
	pluginRegistry.Lock()
	defer pluginRegistry.Unlock()
	if !pluginRegistry.initialized {
		registerBuiltinPluginsLocked()
		pluginRegistry.initialized = true
	}
}

// registerBuiltinPluginsLocked registers all built-in plugins with the registry.
// Must be called with pluginRegistry.Lock held.
func registerBuiltinPluginsLocked() {
	pluginRegistry.constructors["build_cache"] = func() lifecycle.Plugin { return NewBuildCachePlugin() }
	pluginRegistry.constructors["glob"] = func() lifecycle.Plugin { return NewGlobPlugin() }
	pluginRegistry.constructors["load"] = func() lifecycle.Plugin { return NewLoadPlugin() }
	pluginRegistry.constructors["jinja_md"] = func() lifecycle.Plugin { return NewJinjaMdPlugin() }
	pluginRegistry.constructors["render_markdown"] = func() lifecycle.Plugin { return NewRenderMarkdownPlugin() }
	pluginRegistry.constructors[PluginNameTemplates] = func() lifecycle.Plugin { return NewTemplatesPlugin() }
	pluginRegistry.constructors["feeds"] = func() lifecycle.Plugin { return NewFeedsPlugin() }
	pluginRegistry.constructors["auto_feeds"] = func() lifecycle.Plugin { return NewAutoFeedsPlugin() }
	pluginRegistry.constructors["publish_feeds"] = func() lifecycle.Plugin { return NewPublishFeedsPlugin() }
	pluginRegistry.constructors["publish_html"] = func() lifecycle.Plugin { return NewPublishHTMLPlugin() }
	pluginRegistry.constructors["sitemap"] = func() lifecycle.Plugin { return NewSitemapPlugin() }
	pluginRegistry.constructors["wikilinks"] = func() lifecycle.Plugin { return NewWikilinksPlugin() }
	pluginRegistry.constructors["toc"] = func() lifecycle.Plugin { return NewTocPlugin() }
	pluginRegistry.constructors["description"] = func() lifecycle.Plugin { return NewDescriptionPlugin() }
	pluginRegistry.constructors["auto_title"] = func() lifecycle.Plugin { return NewAutoTitlePlugin() }
	pluginRegistry.constructors["reading_time"] = func() lifecycle.Plugin { return NewReadingTimePlugin() }
	pluginRegistry.constructors["static_assets"] = func() lifecycle.Plugin { return NewStaticAssetsPlugin() }
	pluginRegistry.constructors["palette_css"] = func() lifecycle.Plugin { return NewPaletteCSSPlugin() }
	pluginRegistry.constructors["prevnext"] = func() lifecycle.Plugin { return NewPrevNextPlugin() }
	pluginRegistry.constructors["heading_anchors"] = func() lifecycle.Plugin { return NewHeadingAnchorsPlugin() }
	pluginRegistry.constructors["redirects"] = func() lifecycle.Plugin { return NewRedirectsPlugin() }
	pluginRegistry.constructors["csv_fence"] = func() lifecycle.Plugin { return NewCSVFencePlugin() }
	pluginRegistry.constructors["mermaid"] = func() lifecycle.Plugin { return NewMermaidPlugin() }
	pluginRegistry.constructors["link_collector"] = func() lifecycle.Plugin { return NewLinkCollectorPlugin() }
	pluginRegistry.constructors["glossary"] = func() lifecycle.Plugin { return NewGlossaryPlugin() }
	pluginRegistry.constructors["md_video"] = func() lifecycle.Plugin { return NewMDVideoPlugin() }
	pluginRegistry.constructors["chartjs"] = func() lifecycle.Plugin { return NewChartJSPlugin() }
	pluginRegistry.constructors["contribution_graph"] = func() lifecycle.Plugin { return NewContributionGraphPlugin() }
	pluginRegistry.constructors["one_line_link"] = func() lifecycle.Plugin { return NewOneLineLinkPlugin() }
	pluginRegistry.constructors["wikilink_hover"] = func() lifecycle.Plugin { return NewWikilinkHoverPlugin() }
	pluginRegistry.constructors["qrcode"] = func() lifecycle.Plugin { return NewQRCodePlugin() }
	pluginRegistry.constructors["youtube"] = func() lifecycle.Plugin { return NewYouTubePlugin() }
	pluginRegistry.constructors["chroma_css"] = func() lifecycle.Plugin { return NewChromaCSSPlugin() }
	pluginRegistry.constructors["overwrite_check"] = func() lifecycle.Plugin { return NewOverwriteCheckPlugin() }
	pluginRegistry.constructors["structured_data"] = func() lifecycle.Plugin { return NewStructuredDataPlugin() }
	pluginRegistry.constructors["pagefind"] = func() lifecycle.Plugin { return NewPagefindPlugin() }
	pluginRegistry.constructors["stats"] = func() lifecycle.Plugin { return NewStatsPlugin() }
	pluginRegistry.constructors["breadcrumbs"] = func() lifecycle.Plugin { return NewBreadcrumbsPlugin() }
	pluginRegistry.constructors["embeds"] = func() lifecycle.Plugin { return NewEmbedsPlugin() }
	pluginRegistry.constructors["blogroll"] = func() lifecycle.Plugin { return NewBlogrollPlugin() }
	pluginRegistry.constructors["mentions"] = func() lifecycle.Plugin { return NewMentionsPlugin() }
	pluginRegistry.constructors["webmentions"] = func() lifecycle.Plugin { return NewWebMentionsPlugin() }
	pluginRegistry.constructors["webmentions_fetch"] = func() lifecycle.Plugin { return NewWebmentionsFetchPlugin() }
	pluginRegistry.constructors["webmentions_leaderboard"] = func() lifecycle.Plugin { return NewWebmentionsLeaderboardPlugin() }
	pluginRegistry.constructors["background"] = func() lifecycle.Plugin { return NewBackgroundPlugin() }
	pluginRegistry.constructors["image_zoom"] = func() lifecycle.Plugin { return NewImageZoomPlugin() }
	pluginRegistry.constructors["static_file_conflicts"] = func() lifecycle.Plugin { return NewStaticFileConflictsPlugin() }
	pluginRegistry.constructors["slug_conflicts"] = func() lifecycle.Plugin { return NewSlugConflictsPlugin() }
	pluginRegistry.constructors["css_bundle"] = func() lifecycle.Plugin { return NewCSSBundlePlugin() }
}

// RegisterPluginConstructor registers a plugin constructor with the given name.
// This allows third-party plugins to be registered and used by name.
func RegisterPluginConstructor(name string, constructor func() lifecycle.Plugin) {
	ensureRegistryInitialized()
	pluginRegistry.Lock()
	defer pluginRegistry.Unlock()
	pluginRegistry.constructors[name] = constructor
}

// PluginByName returns a new instance of a plugin by its name.
// Returns the plugin and true if found, or nil and false if not found.
func PluginByName(name string) (lifecycle.Plugin, bool) {
	ensureRegistryInitialized()
	pluginRegistry.RLock()
	defer pluginRegistry.RUnlock()

	constructor, ok := pluginRegistry.constructors[name]
	if !ok {
		return nil, false
	}

	return constructor(), true
}

// RegisteredPlugins returns a list of all registered plugin names.
func RegisteredPlugins() []string {
	ensureRegistryInitialized()
	pluginRegistry.RLock()
	defer pluginRegistry.RUnlock()

	names := make([]string, 0, len(pluginRegistry.constructors))
	for name := range pluginRegistry.constructors {
		names = append(names, name)
	}

	return names
}

// DefaultPlugins returns all standard plugins in their recommended execution order.
// This is the typical set of plugins for a complete markata build.
func DefaultPlugins() []lifecycle.Plugin {
	return []lifecycle.Plugin{
		// Build cache plugin (Configure very early, Cleanup very late)
		NewBuildCachePlugin(),

		// Configure/Glob stage plugins
		NewGlobPlugin(),
		NewBackgroundPlugin(), // Configure background decorations early

		// Load stage plugins
		NewLoadPlugin(),

		// Transform stage plugins (in order)
		NewAutoTitlePlugin(),              // Auto-generate titles first
		NewDescriptionPlugin(),            // Auto-generate descriptions early
		NewStructuredDataPlugin(),         // Generate structured data (needs title, description)
		NewReadingTimePlugin(),            // Calculate reading time
		NewStatsPlugin(),                  // Calculate comprehensive content stats
		NewBreadcrumbsPlugin(),            // Generate breadcrumb navigation
		NewEmbedsPlugin(),                 // Process embed syntax (before wikilinks)
		NewWikilinksPlugin(),              // Process wikilinks before rendering
		NewMentionsPlugin(),               // Process @mentions (after blogroll config is loaded)
		NewWebmentionsFetchPlugin(),       // Load cached webmentions and attach to posts
		NewWebmentionsLeaderboardPlugin(), // Calculate top posts by webmentions (after fetch)
		NewTocPlugin(),                    // Extract TOC before rendering
		NewJinjaMdPlugin(),                // Process Jinja templates in markdown

		// Render stage plugins
		NewRenderMarkdownPlugin(),
		NewHeadingAnchorsPlugin(),    // Add anchors after markdown rendering
		NewImageZoomPlugin(),         // Process image zoom attributes
		NewContributionGraphPlugin(), // Process contribution graph code blocks
		NewMDVideoPlugin(),           // Convert video images to video tags
		NewYouTubePlugin(),           // Convert YouTube URLs to embeds
		NewLinkCollectorPlugin(),     // Collect links after markdown rendering
		NewTemplatesPlugin(),

		// Collect stage plugins
		NewSlugConflictsPlugin(), // Detect slug conflicts (runs first in Collect)
		NewFeedsPlugin(),
		NewAutoFeedsPlugin(),
		NewBlogrollPlugin(),            // Fetch external feeds for blogroll
		NewStatsPlugin(),               // Aggregate stats after feeds are built (runs Collect)
		NewPrevNextPlugin(),            // Calculate prev/next after feeds are built
		NewOverwriteCheckPlugin(),      // Detect conflicting output paths
		NewStaticFileConflictsPlugin(), // Detect static files that would clobber generated content

		// Write stage plugins
		NewStaticAssetsPlugin(), // Copy static assets first
		NewPaletteCSSPlugin(),   // Generate palette CSS (overwrites variables.css)
		NewChromaCSSPlugin(),    // Generate syntax highlighting CSS
		NewCSSBundlePlugin(),    // Bundle CSS files (runs after CSS generators)
		NewPublishFeedsPlugin(),
		NewPublishHTMLPlugin(),
		NewRedirectsPlugin(), // Generate redirect pages
		NewSitemapPlugin(),

		// Cleanup stage plugins
		NewPagefindPlugin(), // Generate search index (requires all HTML written first)
	}
}

// MinimalPlugins returns a minimal set of plugins for basic builds.
// This includes only essential plugins for rendering posts without feeds.
func MinimalPlugins() []lifecycle.Plugin {
	return []lifecycle.Plugin{
		NewGlobPlugin(),
		NewLoadPlugin(),
		NewRenderMarkdownPlugin(),
		NewTemplatesPlugin(),
		NewPublishHTMLPlugin(),
	}
}

// TransformPlugins returns only the transform-stage plugins.
// Useful for adding to a custom plugin set.
func TransformPlugins() []lifecycle.Plugin {
	return []lifecycle.Plugin{
		NewAutoTitlePlugin(),
		NewDescriptionPlugin(),
		NewReadingTimePlugin(),
		NewStatsPlugin(),
		NewBreadcrumbsPlugin(),
		NewEmbedsPlugin(),
		NewWikilinksPlugin(),
		NewMentionsPlugin(),
		NewTocPlugin(),
		NewJinjaMdPlugin(),
	}
}

// ByNames creates plugin instances from a list of names.
// Unknown plugin names are skipped with a warning returned.
func ByNames(names []string) (pluginList []lifecycle.Plugin, warnings []string) {
	pluginList = make([]lifecycle.Plugin, 0, len(names))
	warnings = make([]string, 0)

	for _, name := range names {
		plugin, ok := PluginByName(name)
		if !ok {
			warnings = append(warnings, "unknown plugin: "+name)
			continue
		}
		pluginList = append(pluginList, plugin)
	}

	return pluginList, warnings
}
