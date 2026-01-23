---
title: "Site Search"
description: "Add full-text search to your markata-go site using Pagefind"
date: 2024-01-15
published: true
tags:
  - documentation
  - search
  - pagefind
---

# Site Search

markata-go includes built-in full-text search powered by [Pagefind](https://pagefind.app/), a static site search library that generates an optimized search index during the build process.

## Features

- **Enabled by default** - Search works out of the box with no configuration
- **Fast and lightweight** - Only loads the index chunks needed for each query
- **Offline capable** - Works entirely client-side, no server required
- **Theme integrated** - Automatically matches your site's color palette

## Quick Start

Search is enabled by default for all markata-go sites. Just build your site and the search will work:

```bash
# Install Pagefind (one-time setup)
npm install -g pagefind

# Build your site
markata-go build
```

That's it! The search box appears in your navigation bar.

## Installing Pagefind

Pagefind must be installed on your system to generate the search index. Choose your preferred method:

### npm (Recommended)

```bash
npm install -g pagefind
```

### Homebrew (macOS)

```bash
brew install pagefind
```

### Binary Download

Download from [Pagefind releases](https://github.com/CloudCannon/pagefind/releases).

### Skip Search

If Pagefind is not installed, markata-go logs a warning but continues building. The search UI will be hidden.

## Configuration

### Basic Configuration

Configure search in your `markata-go.toml`:

```toml
[search]
enabled = true              # Default: true
position = "navbar"         # Where to show search: navbar, sidebar, footer
placeholder = "Search..."   # Search input placeholder text
```

### Disable Search

To disable search entirely:

```toml
[search]
enabled = false
```

### Advanced Configuration

```toml
[search]
enabled = true
position = "navbar"
placeholder = "Search..."
show_images = true          # Show thumbnails in results (default: true)
excerpt_length = 200        # Characters for result excerpts (default: 200)

# Pagefind CLI options
[search.pagefind]
bundle_dir = "_pagefind"    # Output directory for index (default: "_pagefind")
root_selector = "main"      # Element containing searchable content
exclude_selectors = [       # Elements to exclude from indexing
    ".no-search",
    "nav",
    "footer"
]
```

## Search Positions

The `position` option controls where the search UI appears:

| Position | Description |
|----------|-------------|
| `navbar` | Right side of the navigation bar (default) |
| `sidebar` | In the sidebar (if enabled) |
| `footer` | In the footer section |
| `custom` | Hidden; use for custom placement via templates |

### Custom Placement

For full control over search placement, set `position = "custom"` and include the search component manually in your templates:

```html
{% include "components/search.html" %}
```

## What Gets Indexed

By default, Pagefind indexes:

- **Post content** - The main article body
- **Title** - Displayed in search results
- **Description** - Shown as result excerpt
- **Tags** - Available as filters
- **Feed membership** - Available as filters

### Excluding Content from Search

#### Via Frontmatter

Exclude a post from the search index:

```yaml
---
title: "Draft Post"
search_exclude: true
---
```

#### Via CSS Classes

Content with `data-pagefind-ignore` is excluded:

```html
<div data-pagefind-ignore>
    This content won't be indexed
</div>
```

## Search Results

Results include:

- **Title** - Clickable link to the post
- **Excerpt** - Highlighted text showing where the match was found
- **Thumbnail** - Post image (if available and `show_images` is enabled)

### Filtering Results

Pagefind supports filtering by metadata. Users can click on tags or feeds in results to filter.

## Theming

The search UI automatically inherits your site's theme colors through CSS variables:

```css
:root {
    --pagefind-ui-primary: var(--primary);
    --pagefind-ui-text: var(--text);
    --pagefind-ui-background: var(--surface);
    --pagefind-ui-border: var(--border);
}
```

### Custom Styling

Override Pagefind styles in your custom CSS:

```css
/* Larger search input */
.pagefind-ui__search-input {
    font-size: 1.1rem;
    padding: 0.75rem 1rem;
}

/* Wider results dropdown */
.search--navbar .pagefind-ui__results {
    width: 500px;
}
```

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `/` | Focus search input (when implemented) |
| `Enter` | Go to first result |
| `↑` / `↓` | Navigate results |
| `Escape` | Close results dropdown |

## Performance

Pagefind is highly optimized:

- **Index size**: ~1KB per page indexed
- **Initial load**: Only loads the search UI (~10KB)
- **On search**: Loads only relevant index chunks
- **Caching**: Results cached for repeated queries

### Large Sites

For sites with thousands of pages, Pagefind handles it efficiently:

- Index is split into chunks
- Only relevant chunks are loaded per query
- Typical search latency: 10-50ms

## Troubleshooting

### Search box not appearing

1. **Check if Pagefind is installed:**
   ```bash
   pagefind --version
   ```

2. **Check build logs for warnings:**
   ```
   WARN pagefind CLI not found, skipping search index generation
   ```

3. **Verify search is enabled:**
   ```bash
   markata-go config get search.enabled
   ```

### No search results

1. **Check if content has `data-pagefind-body`:**
   The default post template includes this. Custom templates need it too.

2. **Verify posts are published:**
   Draft posts (`published: false`) may not be indexed.

3. **Check exclude selectors:**
   Ensure your content isn't excluded by `exclude_selectors`.

### Search results missing content

1. **Check `root_selector`:**
   If set, only content within that selector is indexed.

2. **Verify template structure:**
   Content must be within `data-pagefind-body` element.

## Feed-Specific Search

For sites with multiple content types, you can configure feed-specific search instances:

```toml
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

## CI/CD Integration

### GitHub Actions

```yaml
- name: Install Pagefind
  run: npm install -g pagefind

- name: Build site
  run: markata-go build
```

### Netlify

Add to your `netlify.toml`:

```toml
[build]
  command = "npm install -g pagefind && markata-go build"
```

### Vercel

Add to your `package.json`:

```json
{
  "scripts": {
    "build": "npm install -g pagefind && markata-go build"
  }
}
```

## How It Works

1. **Build stage**: markata-go generates HTML files with `data-pagefind-*` attributes
2. **Cleanup stage**: PagefindPlugin runs `pagefind` CLI to index the output directory
3. **Index generation**: Pagefind creates optimized search index in `_pagefind/`
4. **Runtime**: Pagefind JS loads the index and handles search queries client-side

## See Also

- [Configuration Reference](/docs/guides/configuration/) - Full configuration options
- [Themes](/docs/guides/themes/) - Customize search appearance
- [Pagefind Documentation](https://pagefind.app/docs/) - Official Pagefind docs
