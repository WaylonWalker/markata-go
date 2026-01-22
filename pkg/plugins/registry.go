// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"sync"

	"github.com/example/markata-go/pkg/lifecycle"
)

// pluginRegistry holds all registered plugin constructors.
var pluginRegistry = struct {
	sync.RWMutex
	constructors map[string]func() lifecycle.Plugin
}{
	constructors: make(map[string]func() lifecycle.Plugin),
}

func init() {
	// Register all built-in plugins
	registerBuiltinPlugins()
}

// registerBuiltinPlugins registers all built-in plugins with the registry.
func registerBuiltinPlugins() {
	RegisterPluginConstructor("glob", func() lifecycle.Plugin { return NewGlobPlugin() })
	RegisterPluginConstructor("load", func() lifecycle.Plugin { return NewLoadPlugin() })
	RegisterPluginConstructor("jinja_md", func() lifecycle.Plugin { return NewJinjaMdPlugin() })
	RegisterPluginConstructor("render_markdown", func() lifecycle.Plugin { return NewRenderMarkdownPlugin() })
	RegisterPluginConstructor("templates", func() lifecycle.Plugin { return NewTemplatesPlugin() })
	RegisterPluginConstructor("feeds", func() lifecycle.Plugin { return NewFeedsPlugin() })
	RegisterPluginConstructor("auto_feeds", func() lifecycle.Plugin { return NewAutoFeedsPlugin() })
	RegisterPluginConstructor("publish_feeds", func() lifecycle.Plugin { return NewPublishFeedsPlugin() })
	RegisterPluginConstructor("publish_html", func() lifecycle.Plugin { return NewPublishHTMLPlugin() })
	RegisterPluginConstructor("sitemap", func() lifecycle.Plugin { return NewSitemapPlugin() })
	RegisterPluginConstructor("wikilinks", func() lifecycle.Plugin { return NewWikilinksPlugin() })
	RegisterPluginConstructor("toc", func() lifecycle.Plugin { return NewTocPlugin() })
	RegisterPluginConstructor("description", func() lifecycle.Plugin { return NewDescriptionPlugin() })
	RegisterPluginConstructor("auto_title", func() lifecycle.Plugin { return NewAutoTitlePlugin() })
	RegisterPluginConstructor("reading_time", func() lifecycle.Plugin { return NewReadingTimePlugin() })
	RegisterPluginConstructor("static_assets", func() lifecycle.Plugin { return NewStaticAssetsPlugin() })
	RegisterPluginConstructor("palette_css", func() lifecycle.Plugin { return NewPaletteCSSPlugin() })
	RegisterPluginConstructor("prevnext", func() lifecycle.Plugin { return NewPrevNextPlugin() })
	RegisterPluginConstructor("heading_anchors", func() lifecycle.Plugin { return NewHeadingAnchorsPlugin() })
	RegisterPluginConstructor("redirects", func() lifecycle.Plugin { return NewRedirectsPlugin() })
	RegisterPluginConstructor("csv_fence", func() lifecycle.Plugin { return NewCSVFencePlugin() })
	RegisterPluginConstructor("mermaid", func() lifecycle.Plugin { return NewMermaidPlugin() })
	RegisterPluginConstructor("link_collector", func() lifecycle.Plugin { return NewLinkCollectorPlugin() })
	RegisterPluginConstructor("glossary", func() lifecycle.Plugin { return NewGlossaryPlugin() })
}

// RegisterPluginConstructor registers a plugin constructor with the given name.
// This allows third-party plugins to be registered and used by name.
func RegisterPluginConstructor(name string, constructor func() lifecycle.Plugin) {
	pluginRegistry.Lock()
	defer pluginRegistry.Unlock()
	pluginRegistry.constructors[name] = constructor
}

// PluginByName returns a new instance of a plugin by its name.
// Returns the plugin and true if found, or nil and false if not found.
func PluginByName(name string) (lifecycle.Plugin, bool) {
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
		// Configure/Glob stage plugins
		NewGlobPlugin(),

		// Load stage plugins
		NewLoadPlugin(),

		// Transform stage plugins (in order)
		NewAutoTitlePlugin(),   // Auto-generate titles first
		NewDescriptionPlugin(), // Auto-generate descriptions early
		NewReadingTimePlugin(), // Calculate reading time
		NewWikilinksPlugin(),   // Process wikilinks before rendering
		NewTocPlugin(),         // Extract TOC before rendering
		NewJinjaMdPlugin(),     // Process Jinja templates in markdown

		// Render stage plugins
		NewRenderMarkdownPlugin(),
		NewHeadingAnchorsPlugin(), // Add anchors after markdown rendering
		NewLinkCollectorPlugin(),  // Collect links after markdown rendering
		NewTemplatesPlugin(),

		// Collect stage plugins
		NewFeedsPlugin(),
		NewAutoFeedsPlugin(),
		NewPrevNextPlugin(), // Calculate prev/next after feeds are built

		// Write stage plugins
		NewStaticAssetsPlugin(), // Copy static assets first
		NewPaletteCSSPlugin(),   // Generate palette CSS (overwrites variables.css)
		NewPublishFeedsPlugin(),
		NewPublishHTMLPlugin(),
		NewRedirectsPlugin(), // Generate redirect pages
		NewSitemapPlugin(),
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
		NewWikilinksPlugin(),
		NewTocPlugin(),
		NewJinjaMdPlugin(),
	}
}

// PluginsByNames creates plugin instances from a list of names.
// Unknown plugin names are skipped with a warning returned.
func PluginsByNames(names []string) ([]lifecycle.Plugin, []string) {
	plugins := make([]lifecycle.Plugin, 0, len(names))
	warnings := make([]string, 0)

	for _, name := range names {
		plugin, ok := PluginByName(name)
		if !ok {
			warnings = append(warnings, "unknown plugin: "+name)
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins, warnings
}
