# Search Specification

This document specifies the search functionality for markata-go sites using Pagefind.

## Overview

Search is **enabled by default** for all markata-go sites. The implementation uses [Pagefind](https://pagefind.app/), a static site search library that generates an optimized search index during the build process.

## Architecture

### Build Process

1. **Write Stage** - HTML files are generated normally
2. **Cleanup Stage** - Pagefind plugin runs after all files are written:
   - Executes `pagefind` CLI to index the output directory
   - Generates search index in `_pagefind/` directory
   - Copies Pagefind UI assets

### Runtime Behavior

1. Pagefind JavaScript loads asynchronously
2. Search UI renders in configured position (navbar by default)
3. User queries are matched against the pre-built index
4. Results display with excerpts and highlighting

## Configuration

### Basic Configuration

```toml
[search]
enabled = true              # Default: true
position = "navbar"         # navbar, sidebar, footer, custom
placeholder = "Search..."   # Search input placeholder text
```

### Advanced Configuration

```toml
[search]
enabled = true
position = "navbar"
placeholder = "Search..."
show_images = true          # Show thumbnails in results (default: true)
excerpt_length = 200        # Characters for result excerpts (default: 200)
ranking_boost_title = 2.0   # Boost title matches (default: 2.0)
ranking_boost_heading = 1.5 # Boost heading matches (default: 1.5)

# Pagefind CLI options
[search.pagefind]
bundle_dir = "_pagefind"    # Output directory for index (default: "_pagefind")
exclude_selectors = [".no-search", "nav", "footer"]  # Elements to exclude
root_selector = "main"      # Element containing searchable content

# Automatic binary installation (new in v0.2.x)
auto_install = true         # Automatically download Pagefind if not in PATH (default: true)
version = "latest"          # Version to install: "latest" or specific like "v1.4.0" (default: "latest")
cache_dir = ""              # Custom cache directory (default: XDG cache ~/.cache/markata-go/bin/)
```

### Feed-Specific Search

For sites with multiple content types, configure feed-specific search instances:

```toml
# Global search (all published content)
[search]
enabled = true
position = "navbar"

# Feed-specific search instances
[[search.feeds]]
name = "guides"
filter = "published && feed:guides"
position = "sidebar"
placeholder = "Search guides..."

[[search.feeds]]
name = "blog"
filter = "published && feed:blog"
position = "navbar"
placeholder = "Search blog..."
```

### Frontmatter Control

Posts can override search behavior:

```yaml
---
title: "My Post"
search_exclude: true        # Exclude from search index
search_instance: "guides"   # Use specific search instance
search_boost: 2.0          # Boost this page in results
---
```

## Data Model

### SearchConfig

```go
// SearchConfig configures site-wide search functionality.
type SearchConfig struct {
    // Enabled controls whether search is active (default: true)
    Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`
    
    // Position controls where search UI appears: "navbar", "sidebar", "footer", "custom"
    Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`
    
    // Placeholder is the search input placeholder text
    Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty" toml:"placeholder,omitempty"`
    
    // ShowImages shows thumbnails in search results
    ShowImages *bool `json:"show_images,omitempty" yaml:"show_images,omitempty" toml:"show_images,omitempty"`
    
    // ExcerptLength is the character limit for result excerpts
    ExcerptLength int `json:"excerpt_length,omitempty" yaml:"excerpt_length,omitempty" toml:"excerpt_length,omitempty"`
    
    // Pagefind configures the Pagefind CLI options
    Pagefind PagefindConfig `json:"pagefind,omitempty" yaml:"pagefind,omitempty" toml:"pagefind,omitempty"`
    
    // Feeds configures feed-specific search instances
    Feeds []SearchFeedConfig `json:"feeds,omitempty" yaml:"feeds,omitempty" toml:"feeds,omitempty"`
}

// PagefindConfig configures Pagefind CLI behavior.
type PagefindConfig struct {
    // BundleDir is the output directory for search index (default: "_pagefind")
    BundleDir string `json:"bundle_dir,omitempty" yaml:"bundle_dir,omitempty" toml:"bundle_dir,omitempty"`
    
    // ExcludeSelectors are CSS selectors for elements to exclude from indexing
    ExcludeSelectors []string `json:"exclude_selectors,omitempty" yaml:"exclude_selectors,omitempty" toml:"exclude_selectors,omitempty"`
    
    // RootSelector is the CSS selector for the searchable content container
    RootSelector string `json:"root_selector,omitempty" yaml:"root_selector,omitempty" toml:"root_selector,omitempty"`
    
    // AutoInstall enables automatic Pagefind binary installation (default: true)
    AutoInstall *bool `json:"auto_install,omitempty" yaml:"auto_install,omitempty" toml:"auto_install,omitempty"`
    
    // Version is the Pagefind version to install: "latest" or specific (default: "latest")
    Version string `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`
    
    // CacheDir is the directory for caching Pagefind binaries (default: XDG cache)
    CacheDir string `json:"cache_dir,omitempty" yaml:"cache_dir,omitempty" toml:"cache_dir,omitempty"`
}

// SearchFeedConfig configures a feed-specific search instance.
type SearchFeedConfig struct {
    // Name is the search instance identifier
    Name string `json:"name" yaml:"name" toml:"name"`
    
    // Filter is the filter expression for posts in this search
    Filter string `json:"filter" yaml:"filter" toml:"filter"`
    
    // Position controls where this search UI appears
    Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`
    
    // Placeholder is the search input placeholder text
    Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty" toml:"placeholder,omitempty"`
}
```

### Default Values

```go
func NewSearchConfig() SearchConfig {
    enabled := true
    showImages := true
    return SearchConfig{
        Enabled:       &enabled,
        Position:      "navbar",
        Placeholder:   "Search...",
        ShowImages:    &showImages,
        ExcerptLength: 200,
        Pagefind: PagefindConfig{
            BundleDir:        "_pagefind",
            ExcludeSelectors: []string{},
            RootSelector:     "",
        },
        Feeds: []SearchFeedConfig{},
    }
}
```

## Template Integration

### Data Attributes

Templates add Pagefind data attributes to enable indexing:

```html
<!-- Main content area -->
<main data-pagefind-body>
    <article>
        <h1 data-pagefind-meta="title">{{ post.title }}</h1>
        <div data-pagefind-meta="excerpt" class="description">
            {{ post.description }}
        </div>
        
        <!-- Filter by feed membership -->
        {% for feed in post.feeds %}
        <span data-pagefind-filter="feed" style="display:none">{{ feed }}</span>
        {% endfor %}
        
        <!-- Filter by tags -->
        {% for tag in post.tags %}
        <span data-pagefind-filter="tag" style="display:none">{{ tag }}</span>
        {% endfor %}
        
        <div class="content">
            {{ post.article_html | safe }}
        </div>
    </article>
</main>
```

### Excluding Content

```html
<!-- Exclude specific elements -->
<nav data-pagefind-ignore>
    <!-- Navigation not indexed -->
</nav>

<aside data-pagefind-ignore="all">
    <!-- Sidebar and all descendants excluded -->
</aside>
```

### Search UI Component

The search component is included via template:

```html
{% if config.search.enabled %}
{% include "components/search.html" %}
{% endif %}
```

### components/search.html

```html
{# Search component template #}
{# Requires: config.search (SearchConfig) #}

{% with search = config.search %}
{% if search.enabled %}
<div id="search" class="search search--{{ search.position | default:'navbar' }}">
    <div id="pagefind-search"></div>
</div>

<link href="/_pagefind/pagefind-ui.css" rel="stylesheet">
<script src="/_pagefind/pagefind-ui.js" defer></script>
<script>
    window.addEventListener('DOMContentLoaded', (event) => {
        new PagefindUI({
            element: "#pagefind-search",
            showImages: {{ search.show_images | default:true | lower }},
            excerptLength: {{ search.excerpt_length | default:200 }},
            translations: {
                placeholder: "{{ search.placeholder | default:'Search...' }}"
            }
        });
    });
</script>
{% endif %}
{% endwith %}
```

## Plugin Implementation

### PagefindPlugin

The plugin runs in the **Cleanup stage** (after Write) to ensure all HTML files exist:

```go
type PagefindPlugin struct{}

func (p *PagefindPlugin) Name() string {
    return "pagefind"
}

func (p *PagefindPlugin) Cleanup(m *lifecycle.Manager) error {
    config := getSearchConfig(m.Config())
    
    if !config.IsEnabled() {
        return nil
    }
    
    // Check if pagefind is installed
    if _, err := exec.LookPath("pagefind"); err != nil {
        // Log warning but don't fail - search will just not work
        return nil
    }
    
    return p.runPagefind(m.Config())
}

func (p *PagefindPlugin) runPagefind(config *lifecycle.Config) error {
    searchConfig := getSearchConfig(config)
    
    args := []string{
        "--site", config.OutputDir,
        "--output-subdir", searchConfig.Pagefind.BundleDir,
    }
    
    if searchConfig.Pagefind.RootSelector != "" {
        args = append(args, "--root-selector", searchConfig.Pagefind.RootSelector)
    }
    
    for _, selector := range searchConfig.Pagefind.ExcludeSelectors {
        args = append(args, "--exclude-selectors", selector)
    }
    
    cmd := exec.Command("pagefind", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}
```

### Priority

The plugin runs with `PriorityLast` in the Cleanup stage to ensure all files are written first:

```go
func (p *PagefindPlugin) Priority(stage lifecycle.Stage) int {
    if stage == lifecycle.StageCleanup {
        return lifecycle.PriorityLast
    }
    return lifecycle.PriorityDefault
}
```

## CSS Styling

### Default Search Styles

```css
/* Search container positioning */
.search {
    position: relative;
}

.search--navbar {
    display: flex;
    align-items: center;
    margin-left: auto;
}

.search--sidebar {
    margin-bottom: var(--spacing-lg);
}

.search--footer {
    margin-top: var(--spacing-lg);
}

/* Pagefind UI overrides for theme integration */
:root {
    --pagefind-ui-scale: 0.9;
    --pagefind-ui-primary: var(--primary);
    --pagefind-ui-text: var(--text);
    --pagefind-ui-background: var(--surface);
    --pagefind-ui-border: var(--border);
    --pagefind-ui-tag: var(--secondary);
    --pagefind-ui-border-width: 1px;
    --pagefind-ui-border-radius: var(--radius);
    --pagefind-ui-font: var(--font-family);
}

/* Compact mode for navbar */
.search--navbar .pagefind-ui__search-input {
    width: 200px;
    transition: width 0.2s ease;
}

.search--navbar .pagefind-ui__search-input:focus {
    width: 300px;
}

/* Results dropdown */
.search--navbar .pagefind-ui__results {
    position: absolute;
    top: 100%;
    right: 0;
    width: 400px;
    max-height: 80vh;
    overflow-y: auto;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
```

## Accessibility

The search component follows WCAG 2.1 guidelines:

- Proper ARIA labels and roles
- Keyboard navigation support (Tab, Enter, Escape, Arrow keys)
- Focus management for results
- Screen reader announcements for results count
- Sufficient color contrast in all themes

## Performance

### Index Size Optimization

Pagefind automatically optimizes the search index:
- Content is chunked and compressed
- Only relevant chunks are loaded on search
- Typical index size: ~1KB per page indexed

### Loading Strategy

- Pagefind JS/CSS loaded with `defer` attribute
- Search index loaded lazily on first interaction
- Results cached for repeated queries

## Error Handling

### Pagefind Not Found

If `pagefind` CLI is not found in PATH:
1. **Auto-install enabled (default)**: Automatically downloads and caches Pagefind binary
   - Downloads from official GitHub releases
   - Verifies SHA256 checksum before execution
   - Caches in XDG cache directory for subsequent builds
2. **Auto-install disabled**: Warning logged with installation instructions
   - Site functions normally without search
   - Search UI may show placeholder message

### Auto-Install Failures

If automatic installation fails:
1. Network error: Falls back to PATH check, warns user
2. Checksum mismatch: Aborts for security, warns user
3. Unsupported platform: Warns user, suggests manual installation
4. Disk space/permission issues: Warns user with specific error

### Empty Index

If no content is indexed:
1. Warning logged during build
2. Search UI shows "No results" for all queries

### JavaScript Disabled

When JavaScript is disabled:
1. Search UI hidden via CSS
2. Site remains fully navigable
3. Consider adding sitemap link as fallback

## Security

### Binary Verification

All auto-installed Pagefind binaries are verified:
1. Downloaded from official CloudCannon GitHub releases only
2. SHA256 checksum fetched and verified before extraction
3. Binaries are cached with version-specific directories
4. Executable permissions set appropriately per platform

### Supported Platforms

| OS | Architecture | Asset Name |
|----|--------------|------------|
| macOS | x86_64 (Intel) | x86_64-apple-darwin |
| macOS | arm64 (Apple Silicon) | aarch64-apple-darwin |
| Linux | x86_64 | x86_64-unknown-linux-musl |
| Linux | arm64 | aarch64-unknown-linux-musl |
| Windows | x86_64 | x86_64-pc-windows-msvc |
| FreeBSD | x86_64 | x86_64-unknown-freebsd |

## Future Enhancements

1. **Search Analytics** - Track popular queries
2. **Instant Search** - Search-as-you-type with debouncing
3. **Search Suggestions** - Autocomplete based on content
4. **Federated Search** - Search across multiple sites
5. **Voice Search** - Web Speech API integration
