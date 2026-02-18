---
title: "Configuration Reference"
description: "Complete reference for all markata-go configuration options"
date: 2026-01-24
published: true
slug: /docs/guides/configuration/
tags:
  - configuration
  - reference
---

# Configuration

markata-go uses a flexible configuration system that supports multiple file formats, environment variable overrides, and intelligent merging from multiple sources.

## Configuration File Locations

markata-go searches for configuration files in the following order (first found wins):

| Priority | Location | Description |
|----------|----------|-------------|
| 1 | `--config path/to/config.toml` | CLI-specified path |
| 2 | `./markata-go.toml` | Current directory (TOML) |
| 3 | `./markata-go.yaml` | Current directory (YAML) |
| 4 | `./markata-go.yml` | Current directory (YAML alternate) |
| 5 | `./markata-go.json` | Current directory (JSON) |
| 6 | `~/.config/markata-go/config.toml` | User config directory |

If no configuration file is found, markata-go uses default values with any environment variable overrides applied.

## Supported Formats

| Extension | Format | Notes |
|-----------|--------|-------|
| `.toml` | TOML | **Recommended** - Best for nested config, readable |
| `.yaml`, `.yml` | YAML | Good for complex structures |
| `.json` | JSON | Strict, good for programmatic generation |

### TOML Example (Recommended)

```toml
[markata-go]
title = "My Site"
description = "A site built with markata-go"
url = "https://example.com"
output_dir = "public"

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
use_gitignore = true

[markata-go.markdown]
extensions = ["tables", "strikethrough", "tasklist"]
```

### YAML Example

```yaml
markata-go:
  title: My Site
  description: A site built with markata-go
  url: https://example.com
  output_dir: public

  glob:
    patterns:
      - "posts/**/*.md"
      - "pages/*.md"
    use_gitignore: true

  markdown:
    extensions:
      - tables
      - strikethrough
      - tasklist
```

### JSON Example

```json
{
  "markata-go": {
    "title": "My Site",
    "description": "A site built with markata-go",
    "url": "https://example.com",
    "output_dir": "public",
    "glob": {
      "patterns": ["posts/**/*.md", "pages/*.md"],
      "use_gitignore": true
    },
    "markdown": {
      "extensions": ["tables", "strikethrough", "tasklist"]
    }
  }
}
```

## Configuration Namespacing

All configuration lives under the `[markata-go]` namespace. This:

- Avoids conflicts with other tools in shared config files
- Provides clear ownership for each plugin section
- Enables tooling-friendly editor completions

```toml
# Root namespace - site-wide settings
[markata-go]
title = "My Site"
output_dir = "public"

# Plugin namespaces
[markata-go.glob]
patterns = ["pages/**/*.md", "posts/**/*.md"]

[markata-go.markdown]
extensions = ["tables"]

[markata-go.feeds]
# Feed-specific config
```

## Configuration Options Reference

### Core Settings (`[markata-go]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `output_dir` | string | `"output"` | Build output directory |
| `url` | string | `""` | Site base URL (for absolute links) |
| `title` | string | `""` | Site title |
| `description` | string | `""` | Site description |
| `author` | string | `""` | Default author |
| `license` | string `\|` bool | (unset) | Select a license key (see below). Set to `false` to skip the attribution and warning. |
| `assets_dir` | string | `"static"` | Static assets directory |
| `templates_dir` | string | `"templates"` | Templates directory |
| `hooks` | string[] | `["default"]` | Plugins to load |
| `disabled_hooks` | string[] | `[]` | Plugins to exclude |
| `concurrency` | int | `0` | Worker threads (0 = auto based on CPU cores) |

```toml
[markata-go]
output_dir = "public"
url = "https://example.com"
title = "My Site"
description = "A site built with markata-go"
author = "Jane Doe"
assets_dir = "static"
templates_dir = "templates"
hooks = ["default"]
disabled_hooks = ["sitemap"]
concurrency = 4
```

### License configuration

The `license` key controls the attribution shown in the footer and whether the dev server reminds you to pick a license. It accepts either a string key (selects the attribution) or the literal `false` (hides the line and silences the warning). When the footer shows the copyright line, the license appears on the same line next to the copyright symbol; if copyright is disabled, the license renders as its own line.

Supported license keys:

- `all-rights-reserved` – All rights reserved.
- `cc-by-4.0` (recommended) – Creative Commons Attribution 4.0 International (`https://creativecommons.org/licenses/by/4.0/`). The default when running `markata-go config init` or picking the recommended option in the init wizard.
- `cc-by-sa-4.0` – Creative Commons Attribution-ShareAlike 4.0 International (`https://creativecommons.org/licenses/by-sa/4.0/`).
- `cc-by-nc-4.0` – Creative Commons Attribution-NonCommercial 4.0 International (`https://creativecommons.org/licenses/by-nc/4.0/`).
- `cc-by-nd-4.0` – Creative Commons Attribution-NoDerivatives 4.0 International (`https://creativecommons.org/licenses/by-nd/4.0/`).
- `cc-by-nc-sa-4.0` – Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International (`https://creativecommons.org/licenses/by-nc-sa/4.0/`).
- `mit` – MIT License (`https://opensource.org/licenses/MIT`).

Leaving the key absent (the default in older configs) triggers a validation warning and the serve banner/toast until you choose one of the supported strings or set `license = false`.

### Navigation Links (`[[markata-go.nav]]`)

Navigation links appear in the site header and define your site's main navigation.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `label` | string | Required | Display text for the link |
| `url` | string | Required | Link destination (relative or absolute) |
| `external` | bool | `false` | Opens link in new tab with noopener |

```toml
[[markata-go.nav]]
label = "Home"
url = "/"

[[markata-go.nav]]
label = "Blog"
url = "/blog/"

[[markata-go.nav]]
label = "Docs"
url = "/docs/"

[[markata-go.nav]]
label = "GitHub"
url = "https://github.com/WaylonWalker/markata-go"
external = true
```

To add a built-in **Random post** link:

```toml
[markata-go.random_post]
enabled = true
path = "random"                # generates /random/
exclude_tags = ["draft"]       # optional

[[markata-go.nav]]
label = "Random"
url = "/random/"
```

If you use sidebar navigation, add the same `url = "/random/"` link to `[[markata-go.sidebar.nav]]` (or to `[[markata-go.components.nav.items]]` if you're using the components system).

### SEO Configuration (`[markata-go.seo]`)

Configure SEO metadata and structured data generation for better search engine visibility.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `twitter_handle` | string | `""` | Twitter/X username (without @) for twitter:site |
| `default_image` | string | `""` | Default Open Graph image URL |
| `logo_url` | string | `""` | Site logo URL for Schema.org |

```toml
[markata-go.seo]
twitter_handle = "waylonwalker"
default_image = "/static/og-default.png"
logo_url = "/static/logo.png"

[markata-go.seo.structured_data]
enabled = true

[markata-go.seo.structured_data.publisher]
type = "Organization"
name = "My Company"
url = "https://example.com"
logo = "/static/logo.png"

[markata-go.seo.structured_data.default_author]
type = "Person"
name = "Jane Doe"
url = "https://example.com/about/"
```

#### Structured Data Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable JSON-LD Schema.org generation |
| `publisher` | object | `nil` | Publisher information |
| `default_author` | object | `nil` | Default author for posts |

When enabled, markata-go generates JSON-LD structured data for:
- `WebSite` schema on the home page
- `Article` or `BlogPosting` schema on posts
- Breadcrumb navigation schema

### Theme Settings (`[markata-go.theme]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `"default"` | Theme name |
| `palette` | string | `"default-light"` | Color palette to use |
| `palette_dark` | string | `""` | Dark mode palette (for prefers-color-scheme) |
| `custom_css` | string | `""` | Custom CSS file path (relative to static/) |
| `variables` | map | `{}` | CSS variable overrides |

```toml
[markata-go.theme]
name = "default"
palette = "catppuccin-mocha"

# Optional: different palette for dark mode
palette_dark = "catppuccin-mocha"

# Optional: override specific CSS variables
[markata-go.theme.variables]
"--color-primary" = "#8b5cf6"
"--content-width" = "800px"

# Optional: add custom CSS file
custom_css = "my-styles.css"
```

**Available palettes:** `default-light`, `default-dark`, `catppuccin-mocha`, `catppuccin-latte`, `nord-dark`, `gruvbox-dark`, `dracula`, `rose-pine`, `solarized-dark`, `tokyo-night`

See the [[themes-and-styling|Themes Guide]] for detailed customization options.

### Aesthetic Settings (`[markata-go]`)

Aesthetics control the non-color visual aspects of your site: border radius, spacing, shadows, and borders. While palettes control *what colors* you use, aesthetics control *how things are shaped*.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `aesthetic` | string | `"balanced"` | Aesthetic preset to use |

```toml
[markata-go]
aesthetic = "elevated"
```

**Available aesthetics:**

| Aesthetic | Description |
|-----------|-------------|
| `balanced` | Default. Comfortable rounding, subtle shadows, normal spacing |
| `brutal` | Sharp corners, thick borders, tight spacing, no shadows |
| `minimal` | No rounding, maximum whitespace, no shadows, hairline borders |
| `elevated` | Generous rounding, layered shadows, generous spacing |
| `precision` | Subtle corners, compact spacing, hairline borders, minimal shadows |

#### Aesthetic Overrides (`[markata-go.aesthetic_overrides]`)

Fine-tune individual aesthetic values without creating a full custom aesthetic:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `border_radius` | string | varies | CSS border-radius value (e.g., `"8px"`, `"0"`) |
| `spacing_scale` | float | varies | Spacing multiplier (e.g., `1.0`, `1.5`) |
| `border_width` | string | varies | CSS border-width value (e.g., `"1px"`, `"3px"`) |
| `border_style` | string | varies | CSS border-style value (e.g., `"solid"`, `"none"`) |
| `shadow_intensity` | float | varies | Shadow opacity multiplier (0.0-1.0) |
| `shadow_size` | string | varies | Shadow size: `"none"`, `"sm"`, `"md"`, `"lg"`, `"xl"` |

```toml
[markata-go]
aesthetic = "balanced"

# Override specific values
[markata-go.aesthetic_overrides]
border_radius = "16px"
shadow_intensity = 0.8
spacing_scale = 1.2
```

**Combining with palettes:**

```toml
[markata-go]
aesthetic = "elevated"

[markata-go.theme]
palette = "catppuccin-mocha"
```

See the [[themes-and-styling|Themes Guide]] for detailed aesthetic customization and the full list of built-in presets.

### Layout Components (`[markata-go.components]`)

markata-go includes a configurable layout components system for navigation, footer, and sidebars. Each component can be enabled/disabled and positioned via configuration.

#### Navigation Component (`[markata-go.components.nav]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Show navigation |
| `position` | string | `"header"` | Position: `"header"` or `"sidebar"` |
| `style` | string | `"horizontal"` | Style: `"horizontal"` or `"vertical"` |
| `items` | array | `[]` | Navigation links (overrides top-level nav) |

```toml
[markata-go.components.nav]
enabled = true
position = "header"
style = "horizontal"

[[markata-go.components.nav.items]]
label = "Home"
url = "/"

[[markata-go.components.nav.items]]
label = "Blog"
url = "/blog/"

[[markata-go.components.nav.items]]
label = "GitHub"
url = "https://github.com/user"
external = true
```

#### Footer Component (`[markata-go.components.footer]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Show footer |
| `text` | string | `""` | Custom footer text |
| `show_copyright` | bool | `true` | Show copyright line |
| `links` | array | `[]` | Footer links |

```toml
[markata-go.components.footer]
enabled = true
text = "Thanks for reading!"
show_copyright = true

[[markata-go.components.footer.links]]
label = "RSS"
url = "/rss.xml"

[[markata-go.components.footer.links]]
label = "GitHub"
url = "https://github.com/user"
external = true
```

#### Document Sidebar (`[markata-go.components.doc_sidebar]`)

Table of contents sidebar for long-form content.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Show TOC sidebar |
| `position` | string | `"right"` | Position: `"left"` or `"right"` |
| `width` | string | `"250px"` | Sidebar width |
| `min_depth` | int | `2` | Minimum heading level (h2 = 2) |
| `max_depth` | int | `4` | Maximum heading level (h4 = 4) |

```toml
[markata-go.components.doc_sidebar]
enabled = true
position = "right"
width = "280px"
min_depth = 2
max_depth = 4
```

#### Feed Sidebar (`[markata-go.components.feed_sidebar]`)

Series/collection navigation sidebar for posts in the same feed.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Show feed sidebar |
| `position` | string | `"left"` | Position: `"left"` or `"right"` |
| `width` | string | `"250px"` | Sidebar width |
| `title` | string | `""` | Custom title (defaults to feed title) |
| `feeds` | array | `[]` | Feed slugs to show navigation for |

```toml
[markata-go.components.feed_sidebar]
enabled = true
position = "left"
width = "250px"
title = "In this series"
feeds = ["tutorials", "guides"]
```

**Responsive behavior:** Sidebars are hidden on mobile (< 768px) and shown inline on tablets (768px - 1024px).

#### Share Component (`[markata-go.components.share]`)

Add a "Share this post" grid to the end of every article so readers can send your content to social platforms or copy the link.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Show the component. |
| `title` | string | `"Share this post"` | Heading text above the buttons. |
| `position` | string | `"bottom"` | Modifier used by CSS (applied as `share-panel--<position>`). |
| `platforms` | array | `["twitter","bluesky","linkedin","whatsapp","signal","facebook","telegram","pinterest","reddit","hacker_news","email","copy"]` | Ordered list of platform keys to render. Omit entries to hide them. |
| `custom` | table | `{}` | Custom platform definitions (`name`, `icon`, `url`). Icon paths without `/`, `http`, or `data:` are resolved via `theme_asset_hashed` (e.g., `icons/share/mastodon.svg`). |

```toml
[markata-go.components.share]
enabled = true
title = "Share this post"
  platforms = ["twitter", "mastodon", "linkedin", "copy"]

[markata-go.components.share.custom]
mastodon = { name = "Mastodon", icon = "mastodon.svg", url = "https://mastodon.social/share?text={{title}}&url={{url}}" }
```

Supported placeholders for share URLs:

- `{{title}}` → URL-encoded post title (falls back to site title or slug).
- `{{url}}` → URL-encoded absolute post URL (`config.url` + `post.href`).
- `{{excerpt}}` → URL-encoded post description or excerpt when provided.

Copying uses the Clipboard API with a DOM fallback and updates the button label/tooltip to "Link copied" for 2 seconds. Built-in platforms include Twitter, Bluesky, LinkedIn, WhatsApp, Signal, Facebook, Telegram, Pinterest, Reddit, Hacker News, Email, and Copy Link, each with a coordinated icon from `pkg/themes/default/static/icons/share/`.

### IndieAuth Settings (`[markata-go.indieauth]`)

[IndieAuth](https://indieauth.net/) is a decentralized identity protocol that allows you to use your own domain to sign in to websites. markata-go can add the necessary `<link>` tags to your site's HTML head.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable IndieAuth link tags in HTML head |
| `authorization_endpoint` | string | `""` | URL of your authorization endpoint |
| `token_endpoint` | string | `""` | URL of your token endpoint |
| `me_url` | string | `""` | Your profile URL for `rel="me"` links |

```toml
[markata-go.indieauth]
enabled = true
authorization_endpoint = "https://indieauth.com/auth"
token_endpoint = "https://tokens.indieauth.com/token"
me_url = "https://github.com/yourusername"
```

When enabled, this adds the following link tags to your site's `<head>`:

```html
<link rel="authorization_endpoint" href="https://indieauth.com/auth">
<link rel="token_endpoint" href="https://tokens.indieauth.com/token">
<link rel="me" href="https://github.com/yourusername">
```

### Webmention Settings (`[markata-go.webmention]`)

[Webmention](https://www.w3.org/TR/webmention/) is a web standard for conversations and interactions across websites. markata-go can add the webmention endpoint link tag to your site.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Webmention link tag in HTML head |
| `endpoint` | string | `""` | URL of your Webmention endpoint |

```toml
[markata-go.webmention]
enabled = true
endpoint = "https://webmention.io/example.com/webmention"
```

When enabled, this adds the following link tag to your site's `<head>`:

```html
<link rel="webmention" href="https://webmention.io/example.com/webmention">
```

**Popular Webmention services:**
- [webmention.io](https://webmention.io/) - Free hosted webmention service
- [Bridgy](https://brid.gy/) - Connects social media interactions to webmentions

### WebSub Settings (`[markata-go.websub]`)

[WebSub](https://www.w3.org/TR/websub/) enables near-real-time feed delivery by advertising hub URLs. When enabled, markata-go adds discovery links to HTML pages and RSS/Atom feeds.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable WebSub discovery links |
| `hubs` | array | `[]` | List of WebSub hub URLs |

```toml
[markata-go.websub]
enabled = true
hubs = ["https://hub.example.com/"]
```

**Output:**
- HTML `<head>` includes `<link rel="hub" href="...">` for each hub
- RSS/Atom feeds include `rel="hub"` and `rel="self"` links

### Head Configuration (`[markata-go.head]`)

The head configuration allows you to customize elements in the HTML `<head>` section, including custom meta tags, links, scripts, and feed alternate links.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `text` | string | `""` | Raw HTML to include in head (use with caution) |
| `meta` | array | `[]` | Custom meta tags |
| `link` | array | `[]` | Custom link tags |
| `script` | array | `[]` | Custom script tags |
| `alternate_feeds` | array | `[]` | Feed alternate links (RSS, Atom, JSON) |

#### Alternate Feed Links

By default, markata-go includes RSS and Atom feed links in the `<head>`. You can customize which feeds are advertised:

```toml
[markata-go.head]
# Customize which feeds get <link rel="alternate"> tags
[[markata-go.head.alternate_feeds]]
type = "rss"
title = "RSS Feed"
href = "/rss.xml"

[[markata-go.head.alternate_feeds]]
type = "atom"
title = "Atom Feed"
href = "/atom.xml"

[[markata-go.head.alternate_feeds]]
type = "json"
title = "JSON Feed"
href = "/feed.json"
```

**Supported feed types:**
| Type | MIME Type | Description |
|------|-----------|-------------|
| `rss` | `application/rss+xml` | RSS 2.0 feed |
| `atom` | `application/atom+xml` | Atom 1.0 feed |
| `json` | `application/feed+json` | JSON Feed |

**Example: Only advertise JSON Feed:**

```toml
[[markata-go.head.alternate_feeds]]
type = "json"
title = "JSON Feed"
href = "/feed.json"
```

**Example: Feed per section:**

```toml
[[markata-go.head.alternate_feeds]]
type = "rss"
title = "Blog RSS"
href = "/blog/rss.xml"

[[markata-go.head.alternate_feeds]]
type = "rss"
title = "Tutorials RSS"
href = "/tutorials/rss.xml"
```

#### Custom Meta Tags

```toml
[[markata-go.head.meta]]
name = "author"
content = "Jane Doe"

[[markata-go.head.meta]]
property = "og:site_name"
content = "My Site"
```

#### Custom Link Tags

```toml
[[markata-go.head.link]]
rel = "icon"
href = "/favicon.ico"

[[markata-go.head.link]]
rel = "preconnect"
href = "https://fonts.googleapis.com"
crossorigin = true
```

#### Custom Script Tags

```toml
[[markata-go.head.script]]
src = "/js/analytics.js"
```

### Glob Settings (`[markata-go.glob]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `patterns` | string[] | `["pages/**/*.md", "posts/**/*.md"]` | Glob patterns to find content files |
| `use_gitignore` | bool | `true` | Respect .gitignore when finding files |

```toml
[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md", "docs/**/*.md"]
use_gitignore = true
```

### Markdown Settings (`[markata-go.markdown]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `extensions` | string[] | `[]` | Markdown extensions to enable |

Available extensions:
- `tables` - GFM tables
- `strikethrough` - ~~strikethrough~~ text
- `autolinks` - Automatic URL linking
- `tasklist` - Task list checkboxes

```toml
[markata-go.markdown]
extensions = ["tables", "strikethrough", "autolinks", "tasklist"]
```

#### Syntax Highlighting (`[markata-go.markdown.highlight]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable syntax highlighting |
| `theme` | string | `""` | Chroma theme (empty = auto from palette) |
| `line_numbers` | bool | `false` | Show line numbers in code blocks |

```toml
[markata-go.markdown.highlight]
enabled = true
theme = "github-dark"    # Or leave empty for auto-detection
line_numbers = false
```

### Mermaid Settings (`[markata-go.mermaid]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Whether mermaid diagram processing is active |
| `mode` | string | `"client"` | Rendering mode: `client` (browser-side), `cli` (mmdc), or `chromium` (headless Chrome) |
| `cdn_url` | string | mermaid CDN URL | URL for the Mermaid.js library |
| `theme` | string | `"default"` | Mermaid theme: `default`, `dark`, `forest`, `neutral`. Ignored when `use_css_variables` is true. |
| `use_css_variables` | bool | `true` | Derive diagram colors from site CSS custom properties. Reads `--color-background`, `--color-text`, `--color-primary`, `--color-code-bg`, and `--color-surface` with hardcoded fallbacks. |
| `lightbox` | bool | `true` | Enable click-to-zoom lightbox overlay with interactive pan and zoom via svg-pan-zoom. |
| `lightbox_selector` | string | `".glightbox-mermaid"` | CSS selector for lightbox links (backward compatibility; programmatic API does not use it). |

#### CLI Mode Settings (`[markata-go.mermaid.cli]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mmdc_path` | string | `""` | Path to the mmdc binary. Auto-detected if empty. |
| `extra_args` | string | `""` | Additional command-line arguments passed to mmdc. |

#### Chromium Mode Settings (`[markata-go.mermaid.chromium]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `browser_path` | string | `""` | Path to Chrome/Chromium binary. Auto-detected if empty. |
| `timeout` | int | `30` | Maximum seconds to wait for a diagram to render. |
| `max_concurrent` | int | `4` | Maximum number of parallel diagram renders. |
| `no_sandbox` | bool | `false` | Disable Chromium sandbox. Required in containers (Docker, Distrobox, etc.). |

```toml
[markata-go.mermaid]
enabled = true
theme = "default"
use_css_variables = true    # Diagrams automatically match your site palette
lightbox = true             # Click any diagram to zoom, pan, and explore
```

To pre-render diagrams to static SVGs using Chromium (no client-side JS needed):

```toml
[markata-go.mermaid]
mode = "chromium"

[markata-go.mermaid.chromium]
timeout = 30
max_concurrent = 4
# no_sandbox = true         # Required inside Docker or other containers
```

#### Chromium in Containers (Docker, Distrobox, Podman)

Chromium's sandbox requires kernel capabilities that most containers restrict.
Set `no_sandbox = true` when running inside Docker, Distrobox, Podman, or
similar environments:

```toml
[markata-go.mermaid.chromium]
no_sandbox = true
```

**Installing Chrome without sudo:** If your container does not have Chrome or
Chromium installed, download `chrome-headless-shell` from
[Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/).
This is a lightweight headless binary (~180 MB) that requires no GUI
dependencies and no root access:

```bash
# Download and extract (adjust version as needed)
cd /tmp
curl -fsSL https://storage.googleapis.com/chrome-for-testing-public/LATEST_RELEASE_STABLE -o version.txt
VERSION=$(cat version.txt)
curl -fsSL "https://storage.googleapis.com/chrome-for-testing-public/${VERSION}/linux64/chrome-headless-shell-linux64.zip" -o headless.zip
unzip headless.zip

# Install to ~/.local (no sudo needed)
mkdir -p ~/.local/lib/chrome-headless-shell
mv chrome-headless-shell-linux64/* ~/.local/lib/chrome-headless-shell/

# Create a wrapper script so markata-go auto-detects it
mkdir -p ~/.local/bin
cat > ~/.local/bin/headless-shell << 'EOF'
#!/bin/sh
exec "$HOME/.local/lib/chrome-headless-shell/chrome-headless-shell" "$@"
EOF
chmod +x ~/.local/bin/headless-shell

# Verify
headless-shell --version
```

markata-go auto-detects `headless-shell` in your `PATH`. You can also set the
path explicitly:

```toml
[markata-go.mermaid.chromium]
browser_path = "/home/you/.local/lib/chrome-headless-shell/chrome-headless-shell"
no_sandbox = true
```

To disable the lightbox overlay (diagrams render inline only):

```toml
[markata-go.mermaid]
lightbox = false
```

To use a fixed Mermaid theme instead of CSS variables:

```toml
[markata-go.mermaid]
use_css_variables = false
theme = "dark"
```

See the [plugin reference](../reference/plugins.md#mermaid) for full details on lightbox interaction, CSS classes, and theming.

### Image Optimization Settings (`[markata-go.image_optimization]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable image optimization |
| `formats` | string[] | `"avif", "webp"` | Output formats to generate |
| `quality` | int | `80` | Default quality for all formats |
| `avif_quality` | int | `80` | AVIF quality override |
| `webp_quality` | int | `80` | WebP quality override |
| `widths` | int[] | `[480, 768, 1200]` | Responsive widths to generate |
| `sizes` | string | `"100vw"` | Sizes attribute for responsive sources |
| `cache_dir` | string | `.markata/image-cache` | Cache directory for encode metadata |
| `avifenc_path` | string | `""` | Path to `avifenc` (auto-detect if empty) |
| `cwebp_path` | string | `""` | Path to `cwebp` (auto-detect if empty) |

```toml
[markata-go.image_optimization]
enabled = true
formats = ["avif", "webp"]
quality = 80
widths = [480, 768, 1200]
sizes = "100vw"
```

The plugin only processes local images and skips external URLs or data URIs. Missing encoders do not fail the build.

### Layout System (`[markata-go.layout]`)

The layout system controls page structure including sidebars, table of contents, headers, and footers. Different layouts can be assigned to different content types.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `"blog"` | Default layout: `"docs"`, `"blog"`, `"landing"`, `"bare"` |
| `paths` | map | `{}` | Map URL prefixes to layouts |
| `feeds` | map | `{}` | Map feed slugs to layouts |

```toml
[markata-go.layout]
name = "docs"  # Default layout for all pages

# Path-based layout selection
[markata-go.layout.paths]
"/docs/" = "docs"
"/blog/" = "blog"
"/about/" = "landing"

# Feed-based layout selection
[markata-go.layout.feeds]
"docs" = "docs"
"blog" = "blog"
```

#### Documentation Layout (`[markata-go.layout.docs]`)

Three-panel layout with sidebar navigation, content, and table of contents.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sidebar_position` | string | `"left"` | Sidebar position: `"left"` or `"right"` |
| `sidebar_width` | string | `"280px"` | Sidebar width |
| `sidebar_collapsible` | bool | `true` | Allow sidebar collapse |
| `sidebar_default_open` | bool | `true` | Sidebar open by default |
| `toc_position` | string | `"right"` | TOC position: `"left"` or `"right"` |
| `toc_width` | string | `"220px"` | TOC width |
| `toc_collapsible` | bool | `true` | Allow TOC collapse |
| `toc_default_open` | bool | `true` | TOC open by default |
| `content_max_width` | string | `"800px"` | Maximum content width |
| `header_style` | string | `"minimal"` | Header: `"full"`, `"minimal"`, `"transparent"`, `"none"` |
| `footer_style` | string | `"minimal"` | Footer: `"full"`, `"minimal"`, `"none"` |

```toml
[markata-go.layout.docs]
sidebar_position = "left"
sidebar_width = "280px"
toc_position = "right"
toc_width = "220px"
content_max_width = "800px"
header_style = "minimal"
footer_style = "minimal"
```

#### Blog Layout (`[markata-go.layout.blog]`)

Single-column layout optimized for reading long-form content.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `content_max_width` | string | `"720px"` | Maximum content width |
| `show_toc` | bool | `false` | Show table of contents |
| `toc_position` | string | `"right"` | TOC position if enabled |
| `toc_width` | string | `"200px"` | TOC width |
| `header_style` | string | `"full"` | Header style |
| `footer_style` | string | `"full"` | Footer style |
| `show_author` | bool | `true` | Display post author |
| `show_date` | bool | `true` | Display post date |
| `show_tags` | bool | `true` | Display post tags |
| `show_reading_time` | bool | `true` | Display estimated reading time |
| `show_prev_next` | bool | `true` | Display prev/next navigation |

```toml
[markata-go.layout.blog]
content_max_width = "720px"
show_toc = true
show_author = true
show_date = true
show_tags = true
show_reading_time = true
show_prev_next = true
```

#### Landing Layout (`[markata-go.layout.landing]`)

Full-width layout for marketing pages and home pages.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `content_max_width` | string | `"100%"` | Maximum content width |
| `header_style` | string | `"transparent"` | Header style |
| `header_sticky` | bool | `true` | Sticky header |
| `footer_style` | string | `"full"` | Footer style |
| `hero_enabled` | bool | `true` | Enable hero section |

```toml
[markata-go.layout.landing]
content_max_width = "100%"
header_style = "transparent"
header_sticky = true
hero_enabled = true
```

### Sidebar Configuration (`[markata-go.sidebar]`)

Configure the sidebar navigation component for documentation and guides.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Show sidebar |
| `position` | string | `"left"` | Position: `"left"` or `"right"` |
| `width` | string | `"280px"` | Sidebar width |
| `collapsible` | bool | `true` | Allow collapse |
| `default_open` | bool | `true` | Open by default |
| `title` | string | `""` | Sidebar title/header |
| `multi_feed` | bool | `false` | Multi-feed mode with sections |
| `feeds` | string[] | `[]` | Feed slugs for multi-feed mode |

```toml
[markata-go.sidebar]
enabled = true
position = "left"
width = "280px"
title = "Documentation"

# Manual navigation structure
[[markata-go.sidebar.nav]]
title = "Getting Started"
href = "/docs/"

[[markata-go.sidebar.nav]]
title = "Guides"
children = [
    { title = "Configuration", href = "/docs/guides/configuration/" },
    { title = "Templates", href = "/docs/guides/templates/" },
    { title = "Themes", href = "/docs/guides/themes/" },
    { title = "Feeds", href = "/docs/guides/feeds/" },
]

[[markata-go.sidebar.nav]]
title = "Reference"
children = [
    { title = "CLI", href = "/docs/reference/cli/" },
    { title = "Plugins", href = "/docs/reference/plugins/" },
]
```

#### Path-Specific Sidebars (`[markata-go.sidebar.paths]`)

Configure different sidebar content for different URL paths.

```toml
[markata-go.sidebar.paths."/docs/"]
title = "Documentation"
feed = "docs"  # Auto-generate from this feed

[markata-go.sidebar.paths."/guides/"]
title = "Guides"
[markata-go.sidebar.paths."/guides/".auto_generate]
directory = "guides"
order_by = "nav_order"
```

#### Multi-Feed Sidebars

Show posts from multiple feeds in collapsible sections.

```toml
[markata-go.sidebar]
multi_feed = true
feeds = ["docs", "guides", "tutorials"]

[[markata-go.sidebar.feed_sections]]
feed = "docs"
title = "Documentation"
collapsed = false

[[markata-go.sidebar.feed_sections]]
feed = "guides"
title = "Guides"
collapsed = true
max_items = 10
```

### Table of Contents (`[markata-go.toc]`)

Configure the table of contents component for document navigation.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Show TOC |
| `auto_enable` | bool | `false` | Auto-show based on thresholds |
| `min_toc_links` | int | `3` | Min headings for auto-show |
| `min_word_count` | int | `500` | Min words for auto-show |
| `position` | string | `"right"` | Position: `"left"` or `"right"` |
| `width` | string | `"220px"` | TOC width |
| `min_depth` | int | `2` | Minimum heading level (h2 = 2) |
| `max_depth` | int | `4` | Maximum heading level (h4 = 4) |
| `title` | string | `"On this page"` | TOC section title |
| `collapsible` | bool | `true` | Allow collapse |
| `default_open` | bool | `true` | Open by default |
| `scroll_spy` | bool | `true` | Highlight current section |

```toml
[markata-go.toc]
enabled = true
auto_enable = true          # Auto-show TOC for long posts
min_toc_links = 3           # Show if 3+ headings
min_word_count = 500        # Show if 500+ words
position = "right"
width = "220px"
title = "On this page"
min_depth = 2
max_depth = 4
scroll_spy = true
```

### Header Configuration (`[markata-go.header]`)

Configure the header component for layouts.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `style` | string | `"full"` | Style: `"full"`, `"minimal"`, `"transparent"`, `"none"` |
| `sticky` | bool | `true` | Stick to top when scrolling |
| `show_logo` | bool | `true` | Display site logo |
| `show_title` | bool | `true` | Display site title |
| `show_nav` | bool | `true` | Display navigation links |
| `show_search` | bool | `true` | Display search box |
| `show_theme_toggle` | bool | `true` | Display theme toggle button |

```toml
[markata-go.header]
style = "full"
sticky = true
show_logo = true
show_title = true
show_nav = true
show_search = true
show_theme_toggle = true
```

### Footer Configuration (`[markata-go.footer_layout]`)

Configure the footer component for layouts.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `style` | string | `"full"` | Style: `"full"`, `"minimal"`, `"none"` |
| `sticky` | bool | `false` | Stick to bottom |
| `show_copyright` | bool | `true` | Display copyright notice |
| `copyright_text` | string | `""` | Custom copyright text |
| `show_social_links` | bool | `true` | Display social media links |
| `show_nav_links` | bool | `true` | Display navigation links |

```toml
[markata-go.footer_layout]
style = "full"
sticky = false
show_copyright = true
copyright_text = "© 2024 My Site. All rights reserved."
show_social_links = true
show_nav_links = true
```

### Link Avatars (`[markata-go.link_avatars]`)

Add favicon icons next to external links.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable link avatars |
| `mode` | string | `"js"` | `"js"`, `"local"`, or `"hosted"` |
| `selector` | string | `"a[href^='http']"` | CSS selector for links to enhance |
| `service` | string | `"duckduckgo"` | Favicon service (`duckduckgo`, `google`, `custom`) |
| `template` | string | `""` | Custom service template (used when `service = "custom"`) |
| `size` | int | `16` | Icon size in pixels |
| `position` | string | `"before"` | `"before"` or `"after"` |
| `ignore_domains` | string[] | `[]` | Domains to skip |
| `ignore_origins` | string[] | `[]` | Full origins to skip |
| `ignore_selectors` | string[] | `[]` | CSS selectors to skip |
| `ignore_classes` | string[] | `[]` | CSS classes to skip |
| `ignore_ids` | string[] | `[]` | Element IDs to skip |
| `hosted_base_url` | string | `""` | Base URL for hosted mode assets |

```toml
[markata-go.link_avatars]
enabled = true
mode = "local"
service = "duckduckgo"
size = 16
position = "before"
ignore_domains = ["localhost", "127.0.0.1"]
```

### Blogroll Configuration (`[markata-go.blogroll]`)

Configure the blogroll and RSS reader functionality to display feeds from blogs you follow.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable blogroll plugin |
| `cache_dir` | string | `"cache/blogroll"` | Cache directory |
| `cache_duration` | string | `"1h"` | Cache TTL (Go duration format) |
| `timeout` | int | `30` | HTTP timeout in seconds |
| `concurrent_requests` | int | `5` | Max parallel feed fetches |
| `max_entries_per_feed` | int | `50` | Max entries per feed |

```toml
[markata-go.blogroll]
enabled = true
cache_dir = "cache/blogroll"
cache_duration = "1h"
timeout = 30
concurrent_requests = 5
max_entries_per_feed = 50

# Custom templates
[markata-go.blogroll.templates]
blogroll = "blogroll.html"
reader = "reader.html"
```

#### Adding Feeds (`[[markata-go.blogroll.feeds]]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | RSS or Atom feed URL |
| `title` | string | No | Display name (auto-fetched if not set) |
| `description` | string | No | Short description |
| `category` | string | No | Group feeds together |
| `tags` | string[] | No | Additional labels |
| `site_url` | string | No | Main website URL |
| `image_url` | string | No | Logo or icon URL |
| `active` | bool | No | Set `false` to disable |

```toml
# Technology blogs
[[markata-go.blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
description = "Creator of Datasette, Django co-creator"
category = "Technology"
tags = ["python", "ai", "llm"]

[[markata-go.blogroll.feeds]]
url = "https://jvns.ca/atom.xml"
title = "Julia Evans"
description = "Making hard things easy to understand"
category = "Technology"
tags = ["linux", "networking"]

# Design blogs
[[markata-go.blogroll.feeds]]
url = "https://css-tricks.com/feed/"
title = "CSS-Tricks"
category = "Design"
tags = ["css", "frontend"]
```

**Generated pages:**
- `/blogroll/` - Directory of all feeds grouped by category
- `/reader/` - River-of-news style page with latest posts from all feeds

See the [Blogroll Guide](/docs/guides/blogroll/) for detailed configuration and customization.

### Post Output Formats (`[markata-go.post_formats]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `html` | bool | `true` | Generate standard HTML pages |
| `markdown` | bool | `true` | Generate raw markdown source |
| `text` | bool | `true` | Generate plain text output |
| `og` | bool | `true` | Generate OpenGraph card HTML |

```toml
[markata-go.post_formats]
html = true       # /slug/index.html (default)
markdown = true   # /slug.md (canonical) - enabled by default
text = true       # /slug.txt (canonical) - enabled by default
og = true         # /slug/og/index.html (social card)
```

**Reversed Redirects for txt/md**: For `.txt` and `.md` formats, content is placed at the canonical URL (`/slug.txt`, `/slug.md`) with backwards-compatible redirects at `/slug/index.txt` and `/slug/index.txt/index.html`. This supports standard web txt files like `robots.txt`, `llms.txt`, and `humans.txt`.

**Canonical URLs and Alternate Links**: When post formats are enabled, markata-go automatically adds:

- `<link rel="canonical">` pointing to the post's primary URL (for SEO)
- `<link rel="alternate">` for each enabled format:
  - Markdown: `type="text/markdown"` linking to `/slug.md`
  - OG Card: `type="text/html"` linking to `og/`

**Visible Format Links**: When alternate formats are enabled, posts and feeds display visible links allowing visitors to access content in their preferred format.

OG card pages automatically include:
- `<link rel="canonical">` pointing back to the original post
- `<meta name="robots" content="noindex, nofollow">` to prevent search engine indexing

See the [[post-formats|Post Output Formats Guide]] for detailed usage including social image generation and content negotiation.

### Well-Known Files (`[markata-go.well_known]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable auto-generated `.well-known` endpoints |
| `auto_generate` | array | `[...]` | Entries to auto-generate from site metadata |
| `ssh_fingerprint` | string | `""` | SSH fingerprint for `/.well-known/sshfp` |
| `keybase_username` | string | `""` | Keybase username for `/.well-known/keybase.txt` |

```toml
[markata-go.well_known]
enabled = true
auto_generate = ["host-meta", "host-meta.json", "webfinger", "nodeinfo", "time"]

# Optional entries requiring config
ssh_fingerprint = "SHA256:abcdef..."
keybase_username = "username"
```

**Generated files:**
- `/.well-known/host-meta`
- `/.well-known/host-meta.json`
- `/.well-known/webfinger`
- `/.well-known/nodeinfo` and `/nodeinfo/2.0`
- `/.well-known/time`
- `/.well-known/sshfp` (when `ssh_fingerprint` is set)
- `/.well-known/keybase.txt` (when `keybase_username` is set)

Templates live under `templates/well-known/` and can be overridden per theme.

### Content Templates (`[content_templates]`)

Content templates configure the `markata-go new` command, controlling default frontmatter and output directories for different content types.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `directory` | string | `"content-templates"` | Directory for user-defined template files |
| `placement` | object | `{post:"posts",page:"pages",docs:"docs"}` | Map of template names to output directories |
| `templates` | array | `[]` | Custom template definitions |

```toml
[content_templates]
directory = "content-templates"

# Override default directory placement
[content_templates.placement]
post = "blog"          # markata-go new -t post creates in blog/
page = "pages"
docs = "documentation"

# Define custom templates
[[content_templates.templates]]
name = "tutorial"
directory = "tutorials"
body = "## Prerequisites\n\n## Steps\n\n## Summary"

[content_templates.templates.frontmatter]
templateKey = "tutorial"
series = ""

[[content_templates.templates]]
name = "recipe"
directory = "recipes"
body = "## Ingredients\n\n## Instructions"

[content_templates.templates.frontmatter]
templateKey = "recipe"
prep_time = ""
cook_time = ""
servings = 4
```

**File-based Templates:**

Create markdown files in the `content-templates/` directory (or your configured directory):

```markdown
---
# content-templates/changelog.md
templateKey: changelog
_directory: changelogs
version: ""
---

## Added

## Changed

## Fixed
```

The `_directory` field in frontmatter specifies the output directory and is removed from generated content.

**Usage:**

```bash
markata-go new --list                     # List all templates
markata-go new "My Post"                  # Use default (post) template
markata-go new "Tutorial" -t tutorial     # Use custom template
markata-go new "Recipe" -t recipe --dir custom  # Override directory
```

See [[cli-reference|CLI Reference]] for complete `new` command documentation.

### Search Settings (`[search]`)

Site-wide search is enabled by default using [Pagefind](https://pagefind.app/).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable search |
| `position` | string | `"navbar"` | Where to show search: `navbar`, `sidebar`, `footer`, `custom` |
| `placeholder` | string | `"Search..."` | Search input placeholder text |
| `show_images` | bool | `true` | Show thumbnails in search results |
| `excerpt_length` | int | `200` | Characters for result excerpts |

```toml
[search]
enabled = true
position = "navbar"
placeholder = "Search..."
show_images = true
excerpt_length = 200

# Pagefind CLI options
[search.pagefind]
bundle_dir = "_pagefind"    # Output directory for search index
root_selector = "main"       # CSS selector for searchable content
exclude_selectors = [".no-search", "nav", "footer"]  # Elements to exclude
```

**Requirements:** Pagefind CLI must be installed. Install the standalone binary from [GitHub releases](https://github.com/Pagefind/pagefind/releases), or enable `auto_install = true` under `[search.pagefind]` to let markata-go download it automatically. If not installed, search is skipped with a warning.

See the [[search|Search Guide]] for detailed usage and customization.

### Feed Defaults (`[markata-go.feed_defaults]`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `items_per_page` | int | `10` | Default items per page |
| `orphan_threshold` | int | `3` | Minimum items for a separate page |

```toml
[markata-go.feed_defaults]
items_per_page = 10
orphan_threshold = 3

[markata-go.feed_defaults.formats]
html = true
simple_html = true
rss = true
atom = true
json = true
sitemap = true
markdown = false
text = false

[markata-go.feed_defaults.templates]
html = "feed.html"
simple_html = "simple-feed.html"
rss = "feed.xml"
atom = "atom.xml"
json = "feed.json"
card = "card.html"
sitemap = "sitemap.xml"

[markata-go.feed_defaults.syndication]
max_items = 20
include_content = true
```

### Feed Format Options

| Format | Output Path | Description |
|--------|-------------|-------------|
| `html` | `/{slug}/index.html` | Paginated HTML pages |
| `simple_html` | `/{slug}/simple/index.html` | Compact HTML list |
| `rss` | `/{slug}/rss.xml` | RSS 2.0 feed |
| `atom` | `/{slug}/atom.xml` | Atom 1.0 feed |
| `json` | `/{slug}/feed.json` | JSON Feed |
| `sitemap` | `/{slug}/sitemap.xml` | Sitemap XML |
| `markdown` | `/{slug}.md` | Markdown output |
| `text` | `/{slug}.txt` | Plain text output |

### Feed Configuration (`[[markata-go.feeds]]`)

Each feed is defined as an array item. The default config ships with an archive feed at `/archive/`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slug` | string | `""` | URL-safe identifier (empty = root index) |
| `title` | string | `""` | Feed title |
| `description` | string | `""` | Feed description |
| `filter` | string | `""` | Filter expression for selecting posts |
| `sort` | string | `""` | Field to sort posts by |
| `reverse` | bool | `false` | Reverse sort order |
| `items_per_page` | int | inherited | Items per page (inherits from defaults) |
| `orphan_threshold` | int | inherited | Orphan threshold (inherits from defaults) |
| `formats` | object | inherited | Output formats (inherits from defaults) |
| `templates` | object | inherited | Templates (inherits from defaults) |

```toml
# Main blog feed
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
description = "All blog posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 10

[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = true

# Home page feed (empty slug = root index.html)
[[markata-go.feeds]]
slug = ""
title = "Latest Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5

[markata-go.feeds.formats]
html = true

# Featured posts feed
[[markata-go.feeds]]
slug = "featured"
title = "Featured"
description = "Featured posts"
filter = "published == True and featured == True"
sort = "date"
reverse = true
```

### Series Settings (`[markata-go.series]`)

Series automatically generate ordered feed pages and navigation from `series` frontmatter.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slug_prefix` | string | `"series"` | URL prefix for series feeds (e.g., `/series/<name>/`) |
| `auto_sidebar` | bool | `true` | Auto-enable the series sidebar on series posts |
| `defaults` | object | see below | Default settings for all series feeds |
| `overrides` | map[string]object | `{}` | Per-series overrides (keyed by series slug or name) |

```toml
[markata-go.series]
slug_prefix = "series"
auto_sidebar = true

[markata-go.series.defaults]
items_per_page = 0
sidebar = true

[markata-go.series.defaults.formats]
html = true
simple_html = true
rss = true
atom = true
json = true
sitemap = true

[markata-go.series.overrides."building-a-rest-api"]
title = "Building a REST API with Go"
description = "A complete guide from zero to production"

[markata-go.series.overrides."building-a-rest-api".formats]
markdown = true
```

## Environment Variable Overrides

All configuration options can be overridden via environment variables using the `MARKATA_GO_` prefix.

### Variable Naming Convention

Environment variables follow the pattern: `MARKATA_GO_{SECTION}_{KEY}`

- Keys are UPPERCASE
- Underscores separate nested keys
- Section names use underscores

### Examples

```bash
# Core settings
export MARKATA_GO_OUTPUT_DIR=dist
export MARKATA_GO_URL=https://example.com
export MARKATA_GO_TITLE="My Site"
export MARKATA_GO_CONCURRENCY=4

# Glob settings
export MARKATA_GO_GLOB_PATTERNS="posts/**/*.md,pages/*.md"
export MARKATA_GO_GLOB_USE_GITIGNORE=true

# Markdown settings
export MARKATA_GO_MARKDOWN_EXTENSIONS="tables,strikethrough"

# Feed defaults
export MARKATA_GO_FEED_DEFAULTS_ITEMS_PER_PAGE=20
export MARKATA_GO_FEEDS_DEFAULTS_ORPHAN_THRESHOLD=5
export MARKATA_GO_FEED_DEFAULTS_FORMATS_HTML=true
export MARKATA_GO_FEED_DEFAULTS_FORMATS_RSS=true
export MARKATA_GO_FEED_DEFAULTS_SYNDICATION_MAX_ITEMS=50
export MARKATA_GO_FEED_DEFAULTS_SYNDICATION_INCLUDE_CONTENT=false

# Blogroll settings
export MARKATA_GO_BLOGROLL_ENABLED=false
```

### Value Formats

| Type | Format | Examples |
|------|--------|----------|
| String | Plain text | `MARKATA_GO_TITLE="My Site"` |
| Integer | Numeric | `MARKATA_GO_CONCURRENCY=4` |
| Boolean | `true`/`false`/`1`/`0`/`yes`/`no` | `MARKATA_GO_GLOB_USE_GITIGNORE=true` |
| List | Comma-separated | `MARKATA_GO_HOOKS="glob,load,render"` |

### Build with Environment Overrides

```bash
# Override output directory for a single build
MARKATA_GO_OUTPUT_DIR=dist markata-go build

# Build for staging environment
MARKATA_GO_URL=https://staging.example.com markata-go build

# Increase concurrency
MARKATA_GO_CONCURRENCY=8 markata-go build
```

## Multiple Config Files (Config Merging)

You can merge multiple configuration files together using the `--merge-config` (or `-m`) flag. This is useful for:

- **Environment-specific overrides** (e.g., `markata-go.local.toml`)
- **Fast build configurations** (e.g., `fast-markata-go.toml`)
- **Temporary overrides** without modifying your main config

### How Config Merging Works

Configs are merged in order, with later configs taking precedence:

1. **Defaults** - Built-in default values
2. **Base config** - Your main config file (`markata-go.toml`)
3. **Merge configs** - Each `--merge-config` file, in order
4. **Environment variables** - `MARKATA_GO_*` vars (highest precedence)

### Example: Fast Build Config

Create a `fast-markata-go.toml` for quick development builds:

```toml
[markata-go]
# Only process specific files for fast builds
glob_patterns = ["posts/draft-*.md"]

[markata-go.blogroll]
# Disable blogroll for faster builds
enabled = false
```

Use it with:

```bash
# Merge fast config with base config
markata-go build --merge-config fast-markata-go.toml

# Or use the short flag
markata-go build -m fast-markata-go.toml
```

### Example: Multiple Merge Configs

You can specify multiple merge configs. They are applied in order:

```bash
# Base config + local overrides + debug settings
markata-go build -m markata-go.local.toml -m debug.toml
```

### Example: No Base Config

You can use merge configs without a base config (uses defaults as base):

```bash
# Just use defaults + your override file
markata-go build -c "" -m fast-markata-go.toml
```

### Common Merge Config Patterns

**Local Development** (`markata-go.local.toml` - add to `.gitignore`):

```toml
[markata-go]
url = "http://localhost:8080"
concurrency = 2

[markata-go.blogroll]
enabled = false
```

**Fast Build** (`fast.toml`):

```toml
[markata-go]
# Limit to specific files
glob_patterns = ["posts/current/**/*.md"]

[markata-go.blogroll]
enabled = false
```

**Note on Boolean Fields**: Merging `enabled = false` into a base config with `enabled = true` requires environment variables. Use `MARKATA_GO_BLOGROLL_ENABLED=false` instead.

## Configuration CLI Commands

### `config show`

Display the resolved configuration:

```bash
# Show as YAML (default)
markata-go config show

# Show as JSON
markata-go config show --json

# Show as TOML
markata-go config show --toml
```

### `config get`

Get a specific configuration value:

```bash
# Get top-level value
markata-go config get output_dir

# Get nested value with dot notation
markata-go config get glob.patterns
markata-go config get feed_defaults.items_per_page
markata-go config get feed_defaults.formats.html
```

### `config set`

Update a specific configuration value in-place:

```bash
# Set top-level value
markata-go config set url "https://example.com"

# Set nested value
markata-go config set glob.patterns '["posts/**/*.md", "pages/**/*.md"]'
```

`config set` preserves file permissions for TOML, YAML, and JSON files. In CGO-enabled builds, TOML/YAML edits use tree-sitter byte ranges so existing comments, ordering, and formatting stay intact.

### `config validate`

Validate the configuration file:

```bash
# Validate default config file
markata-go config validate

# Validate specific config file
markata-go config validate -c custom.toml
```

Example output:

```
✓ Configuration is valid
```

Or with errors:

```
✗ Configuration errors:
  - url: URL must include a scheme (e.g., https://)
  - concurrency: must be >= 0 (0 means auto-detect)
  - glob.patterns: no glob patterns specified, no files will be processed (warning)
```

### `config init`

Generate a starter configuration file:

```bash
# Create markata-go.toml with defaults
markata-go config init

# Create YAML config
markata-go config init site.yaml

# Overwrite existing file
markata-go config init --force
```

## Configuration Merging

markata-go merges configuration from multiple sources in order of increasing precedence:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CONFIGURATION RESOLUTION                          │
├─────────────────────────────────────────────────────────────────────┤
│  1. Built-in defaults                                                │
│  2. User config file (~/.config/markata-go/config.toml)             │
│  3. Local config file (./markata-go.toml)                           │
│  4. Environment variables (MARKATA_GO_*)                            │
│  5. CLI arguments (--output-dir, --config, etc.)                    │
│                                                                      │
│  Later sources OVERRIDE earlier sources                              │
│  Nested objects are DEEP MERGED                                     │
│  Arrays REPLACE (not append)                                        │
└─────────────────────────────────────────────────────────────────────┘
```

### Merge Behavior

**Scalar values:** Later wins

```toml
# User config: output_dir = "dist"
# Local config: output_dir = "public"
# Result: output_dir = "public"
```

**Objects:** Deep merge

```toml
# User config:
[markata-go.feed_defaults.formats]
html = true
rss = true

# Local config:
[markata-go.feed_defaults.formats]
atom = true

# Result:
# html = true   (from user)
# rss = true    (from user)
# atom = true   (from local)
```

**Arrays:** Replace (not append)

```toml
# User config: patterns = ["**/*.md"]
# Local config: patterns = ["posts/*.md", "pages/*.md"]
# Result: patterns = ["posts/*.md", "pages/*.md"]
```

## Default Values

When no configuration is provided, markata-go uses these defaults:

```toml
[markata-go]
output_dir = "output"
templates_dir = "templates"
assets_dir = "static"
hooks = ["default"]
concurrency = 0  # Auto-detect based on CPU cores

[markata-go.glob]
patterns = ["pages/**/*.md", "posts/**/*.md"]
use_gitignore = true

[markata-go.feed_defaults]
items_per_page = 10
orphan_threshold = 3

[markata-go.feed_defaults.formats]
html = true
rss = true
atom = false
json = false
markdown = false
text = false
```

## Authors Configuration

The authors system lets you define author profiles in your configuration and reference them from post frontmatter. This enables multi-author support where each post can have one or more authors with rich metadata.

### Basic Setup

```toml
[markata-go.authors]
generate_pages = false    # Generate individual author profile pages
feeds_enabled = false     # Generate per-author feeds

[markata-go.authors.authors.jane]
name = "Jane Doe"
role = "author"
url = "https://janedoe.com"
active = true
default = true            # Assigned to posts with no explicit author

[markata-go.authors.authors.bob]
name = "Bob Smith"
role = "editor"
active = true
```

### Author Fields

Each author entry supports these fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | **required** | Display name |
| `bio` | string | - | Short biography |
| `email` | string | - | Contact email (must contain `@` if set) |
| `avatar` | string | - | Avatar image URL or path |
| `url` | string | - | Personal website URL |
| `role` | string | - | Role label (e.g., "author", "editor") |
| `guest` | bool | `false` | Whether this is a guest author |
| `active` | bool | `false` | Whether the author is currently active |
| `default` | bool | `false` | Default author for posts without explicit authors |
| `contribution` | string | - | Free-text contribution description |
| `contributions` | []string | `[]` | CReDiT taxonomy roles |

### Referencing Authors in Posts

Use author IDs in post frontmatter:

```yaml
---
title: "Our Collaborative Post"
authors:
  - jane
  - bob
---
```

The legacy single-author field is also supported:

```yaml
---
title: "Solo Post"
author: "Jane Doe"
---
```

**Priority:** `authors` array takes precedence over `author` string. If neither is specified, the default author (the one with `default = true`) is assigned automatically.

### Author Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `generate_pages` | bool | `false` | Generate individual author profile pages |
| `url_pattern` | string | `""` | URL pattern for author pages (e.g., `/authors/{id}/`) |
| `feeds_enabled` | bool | `false` | Generate per-author feeds |

### Full Author Example

```toml
[markata-go.authors]
generate_pages = false
feeds_enabled = false

[markata-go.authors.authors.waylon]
name = "Waylon Walker"
bio = "Python and Go developer, blogger"
email = "waylon@example.com"
avatar = "/images/waylon.jpg"
url = "https://waylonwalker.com"
role = "author"
active = true
default = true
contributions = ["Writing", "Original Draft"]

[markata-go.authors.authors.guest]
name = "Guest Writer"
role = "editor"
guest = true
active = true
contribution = "Technical review"
```

**Related:** See the [Frontmatter Guide](frontmatter.md) for details on the `authors` and `author` fields.

## Encryption Settings

Encryption protects private post content with AES-256-GCM client-side encryption. It is **enabled by default**.

```toml
[encryption]
enabled = true                             # default: true
default_key = "default"                    # default: "default"
decryption_hint = "DM me for access"       # optional hint shown to visitors

[encryption.private_tags]
diary = "personal"                         # tag "diary" auto-encrypts with key "personal"
draft-ideas = "default"                    # tag "draft-ideas" encrypts with default key
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable encryption processing |
| `default_key` | string | `"default"` | Key name when post doesn't specify one |
| `decryption_hint` | string | `""` | Help text shown next to password prompt |
| `private_tags` | map | `{}` | Maps tag names to encryption key names |

Encryption keys are loaded from environment variables:

```bash
MARKATA_GO_ENCRYPTION_KEY_DEFAULT=my-password
MARKATA_GO_ENCRYPTION_KEY_PERSONAL=another-password
```

**Related:** See the [Encryption Guide](encryption.md) for complete documentation on making posts private.

---

## Complete Configuration Example

Here's a comprehensive example showing all available options:

```toml
[markata-go]
# Site metadata
title = "My Awesome Blog"
description = "A blog about technology and life"
url = "https://example.com"
author = "Jane Doe"

# Directory configuration
output_dir = "public"
templates_dir = "templates"
assets_dir = "static"

# Plugin configuration
hooks = ["default"]
disabled_hooks = []

# Performance
concurrency = 0  # Auto-detect

# Content discovery
[markata-go.glob]
patterns = [
    "posts/**/*.md",
    "pages/*.md",
    "docs/**/*.md"
]
use_gitignore = true

# Markdown processing
[markata-go.markdown]
extensions = ["tables", "strikethrough", "autolinks", "tasklist"]

# Feed defaults (inherited by all feeds)
[markata-go.feed_defaults]
items_per_page = 10
orphan_threshold = 3

[markata-go.feed_defaults.formats]
html = true
rss = true
atom = false
json = false

[markata-go.feed_defaults.templates]
html = "feed.html"
rss = "rss.xml"
atom = "atom.xml"
json = "feed.json"
card = "partials/card.html"

[markata-go.feed_defaults.syndication]
max_items = 20
include_content = true

# Home page feed
[[markata-go.feeds]]
slug = ""
title = "Latest Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5

[markata-go.feeds.formats]
html = true

# Blog archive
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
description = "All blog posts"
filter = "published == True"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = true

# Featured posts
[[markata-go.feeds]]
slug = "featured"
title = "Featured Posts"
description = "Hand-picked featured content"
filter = "published == True and featured == True"
sort = "date"
reverse = true
items_per_page = 6

[markata-go.feeds.formats]
html = true
rss = true

# Authors
[markata-go.authors]
generate_pages = false
feeds_enabled = false

[markata-go.authors.authors.jane]
name = "Jane Doe"
role = "author"
url = "https://janedoe.com"
active = true
default = true

# CSS minification
[markata-go.css_minify]
enabled = true
exclude = []
preserve_comments = ["Copyright"]

# JS minification
[markata-go.js_minify]
enabled = true
exclude = []
```

## Common Configuration Patterns

### Minimal Blog

```toml
[markata-go]
title = "My Blog"
url = "https://myblog.com"
output_dir = "public"

[markata-go.glob]
patterns = ["posts/**/*.md"]

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true
```

### Documentation Site

```toml
[markata-go]
title = "Project Docs"
url = "https://docs.example.com"
output_dir = "site"

[markata-go.glob]
patterns = ["docs/**/*.md"]

[markata-go.markdown]
extensions = ["tables", "tasklist"]

[[markata-go.feeds]]
slug = ""
title = "Documentation"
filter = "true"
sort = "title"

[markata-go.feeds.formats]
html = true
```

### Multi-Section Site

```toml
[markata-go]
title = "My Site"
url = "https://example.com"
output_dir = "public"

[markata-go.glob]
patterns = ["content/**/*.md"]

# Blog section
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "'blog' in tags"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true

# Tutorials section
[[markata-go.feeds]]
slug = "tutorials"
title = "Tutorials"
filter = "'tutorial' in tags"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true

# Projects section
[[markata-go.feeds]]
slug = "projects"
title = "Projects"
filter = "'project' in tags"
sort = "title"

[markata-go.feeds.formats]
html = true
```

### CI/CD Deployment

```toml
[markata-go]
title = "My Site"
# URL set via environment variable in CI
output_dir = "dist"

[markata-go.glob]
patterns = ["content/**/*.md"]
```

Then in CI:

```bash
MARKATA_GO_URL=https://example.com markata-go build
```

## CSS and JS Minification

markata-go automatically minifies CSS and JS files in the output directory to reduce file sizes.

### CSS Minification

```toml
[markata-go.css_minify]
enabled = true                      # Enable CSS minification (default: true)
exclude = ["variables.css"]         # Files to skip
preserve_comments = ["Copyright"]   # Comment substrings to preserve
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable/disable CSS minification |
| `exclude` | `[]` | File names or glob patterns to skip (e.g., `"vendor-*.css"`) |
| `preserve_comments` | `[]` | Substrings that identify comments to keep |

### JS Minification

```toml
[markata-go.js_minify]
enabled = true                      # Enable JS minification (default: true)
exclude = ["pagefind-ui.js"]        # Files to skip
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable/disable JS minification |
| `exclude` | `[]` | File names or glob patterns to skip (e.g., `"vendor-*.js"`) |

Files with `.min.js` extension are automatically skipped (already minified).

Both plugins run at the very end of the Write stage (`PriorityLast`), after all CSS/JS-generating plugins have finished.

---

## Validation

markata-go validates configuration on load and can report both errors and warnings:

**Errors** (prevent build):
- Invalid URL format (missing scheme or host)
- Negative `items_per_page` or `orphan_threshold`
- Negative `concurrency`

**Warnings** (build continues):
- Empty glob patterns (no files will be processed)
- Feed with no output formats enabled

Run validation explicitly:

```bash
markata-go config validate
```

## See Also

- [Getting Started](/docs/getting-started/) - Quick start guide
- [Themes Guide](/docs/guides/themes/) - Theme and palette customization
- [Feeds Guide](/docs/guides/feeds/) - Detailed feed configuration
- [Blogroll Guide](/docs/guides/blogroll/) - RSS reader and blogroll setup
- [Templates Guide](/docs/guides/templates/) - Template configuration and usage
- [Sidebars Guide](/docs/guides/sidebars/) - Sidebar navigation setup
- [Frontmatter Guide](/docs/guides/frontmatter/) - Post metadata reference
- [CLI Reference](/docs/reference/cli/) - Command-line interface reference
- [Plugin Reference](/docs/reference/plugins/) - Plugin configuration and development
