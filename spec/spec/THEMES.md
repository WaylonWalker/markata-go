# Themes and Customization Specification

Themes control the visual appearance of the generated site. The system supports:
- **Built-in themes** - Ship with usable defaults
- **Theme packages** - Complete themes with templates, CSS, and assets  
- **User customization** - Override any theme file locally

## Design Principles

1. **Zero-config beautiful** - Sites look good with no customization
2. **Progressive customization** - Pick colors → override CSS → replace templates
3. **Theme packages** - Installable, shareable, complete visual identities
4. **Local overrides** - Any theme file can be overridden by placing it in your project
5. **Readable by default** - Typography, contrast, and spacing optimized for reading
6. **Accessible first** - WCAG 2.1 AA compliant colors and focus states

---

## Customization Philosophy

Markata-go is designed to make customization **easy and intuitive**. Users should be able to:

### Level 1: Zero Effort (Instant Results)
- Choose a built-in color palette in one line of config
- Switch between light/dark modes automatically
- Get beautiful defaults that just work

```toml
# One line to change your entire site's look
[markata-go.theme]
palette = "catppuccin-mocha"
```

### Level 2: Quick Tweaks (Minutes)
- Override specific colors without touching CSS
- Change fonts, spacing, or accent colors
- Toggle features on/off

```toml
[markata-go.theme]
palette = "nord-dark"

[markata-go.theme.palette]
accent = "#88c0d0"  # Override just the accent color

[markata-go.theme.features]
toc = false         # Disable table of contents
```

### Level 3: Template Overrides (Hours)
- Override a single template file (e.g., footer)
- Create custom layouts for specific content types
- Maintain upgradability by only changing what you need

```
my-site/
├── templates/
│   └── partials/
│       └── footer.html    # Your custom footer only
└── markata-go.toml
```

### Level 4: Full Custom Theme (Days)
- Create a complete custom theme
- Share themes as packages
- Extend existing themes

### Design Goals

| Goal | How We Achieve It |
|------|-------------------|
| **Look good immediately** | Carefully designed default theme with professional typography |
| **Easy color changes** | Palette system with 20+ built-in options |
| **No CSS knowledge needed** | Config-driven customization for common changes |
| **CSS power when needed** | Full CSS override capability for advanced users |
| **Don't break on upgrade** | Override system means your customizations survive updates |
| **Consistent across formats** | Same visual identity in HTML, RSS readers, and more |

---

## Layout System

Markata-go uses a hierarchical layout system with different templates for different content types.

### Layout Hierarchy

```
┌─────────────────────────────────────────────────────────────────────┐
│                          base.html                                   │
│  (HTML skeleton, <head>, header, footer, common CSS/JS)             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │   post.html     │  │   feed.html     │  │   page.html     │     │
│  │  Single article │  │  List of posts  │  │  Static pages   │     │
│  │  (blog posts)   │  │  (index, tags)  │  │  (about, etc)   │     │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘     │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Custom Layouts                            │   │
│  │  landing.html, gallery.html, docs.html, etc.                │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### Built-in Layouts

| Layout | Purpose | Used For |
|--------|---------|----------|
| `base.html` | HTML skeleton with blocks | All pages inherit from this |
| `post.html` | Single content item | Blog posts, articles |
| `feed.html` | List of content items | Index pages, tag pages, archives |
| `page.html` | Static pages | About, contact, custom pages |
| `card.html` | Post preview in lists | Used inside feed.html |

### Post vs Feed Layouts

**Post Layout (`post.html`)** - Single content item with full content:
- Article header (title, date, author)
- Full rendered content
- Tags/categories
- Previous/next navigation
- Comments section (optional)
- Related posts (optional)

**Feed Layout (`feed.html`)** - Collection of content items:
- Feed header (title, description)
- List of post cards/previews
- Pagination controls
- Filter/sort options (optional)

### Per-Post Layout Override

Individual posts can specify a custom layout in frontmatter:

```yaml
---
title: "My Product Landing Page"
layout: landing
---
```

This uses `templates/landing.html` instead of the default `post.html`.

**Use cases:**
- Landing pages with custom hero sections
- Gallery posts with image grids
- Documentation pages with sidebars
- Minimal pages without header/footer

### Per-Post Template Override

For complete control, posts can specify a full template:

```yaml
---
title: "Custom Page"
template: custom/special-page.html
---
```

**Layout vs Template:**
- `layout` - Uses a layout that extends `base.html` (recommended)
- `template` - Uses a completely standalone template (full control)

### Per-Post Style Override

Posts can include custom CSS or override palette colors:

```yaml
---
title: "Dark Mode Article"
palette: dracula
custom_css: |
  .post-content {
    font-size: 1.1rem;
  }
---
```

Or reference an external CSS file:

```yaml
---
title: "Styled Article"  
css: /css/special-article.css
---
```

### Per-Post Feature Flags

Toggle theme features for specific posts:

```yaml
---
title: "Minimal Post"
features:
  toc: false
  reading_time: false
  comments: false
  header: false
  footer: false
---
```

---

## RSS and Atom Stylesheets

RSS and Atom feeds SHOULD include XSL stylesheets for beautiful rendering in browsers.

### Why XSL Stylesheets?

When users click an RSS/Atom link, browsers display raw XML by default - ugly and confusing. XSL stylesheets transform this into a readable, styled page that:

1. **Explains what RSS is** - Many users don't know what they're looking at
2. **Shows feed content beautifully** - Styled preview of recent posts
3. **Provides subscription instructions** - How to use the feed
4. **Maintains brand consistency** - Uses your site's colors and typography

### Feed Stylesheet Structure

```
themes/default/
├── templates/
│   ├── rss.xml              # RSS 2.0 feed template
│   ├── atom.xml             # Atom feed template
│   └── feed-styles.xsl      # Shared XSL stylesheet
└── static/
    └── css/
        └── feed.css         # CSS for styled feed pages
```

### XSL Stylesheet Template

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" encoding="UTF-8" />
  
  <xsl:template match="/">
    <html>
      <head>
        <title><xsl:value-of select="/rss/channel/title" /> - RSS Feed</title>
        <style>
          /* Inline styles using palette colors */
          :root {
            --color-bg: {{ palette.bg_primary }};
            --color-text: {{ palette.text_primary }};
            --color-accent: {{ palette.accent }};
          }
          body {
            font-family: system-ui, sans-serif;
            max-width: 50rem;
            margin: 2rem auto;
            padding: 0 1rem;
            background: var(--color-bg);
            color: var(--color-text);
          }
          /* ... more styles ... */
        </style>
      </head>
      <body>
        <header>
          <h1>📡 <xsl:value-of select="/rss/channel/title" /></h1>
          <p class="subtitle">This is an RSS feed. Subscribe by copying the URL into your feed reader.</p>
          <p class="feed-url"><xsl:value-of select="/rss/channel/link" /></p>
        </header>
        
        <section class="about-feeds">
          <h2>What is an RSS feed?</h2>
          <p>RSS feeds allow you to subscribe to websites and receive updates automatically in a feed reader app.</p>
          <details>
            <summary>Popular feed readers</summary>
            <ul>
              <li><a href="https://feedly.com">Feedly</a> (Web, iOS, Android)</li>
              <li><a href="https://netnewswire.com">NetNewsWire</a> (Mac, iOS)</li>
              <li><a href="https://newsblur.com">NewsBlur</a> (Web, iOS, Android)</li>
            </ul>
          </details>
        </section>
        
        <section class="recent-posts">
          <h2>Recent Posts</h2>
          <xsl:for-each select="/rss/channel/item">
            <article>
              <h3><a href="{link}"><xsl:value-of select="title" /></a></h3>
              <time><xsl:value-of select="pubDate" /></time>
              <p><xsl:value-of select="description" /></p>
            </article>
          </xsl:for-each>
        </section>
      </body>
    </html>
  </xsl:template>
</xsl:stylesheet>
```

### RSS Template with Stylesheet Reference

```xml
<?xml version="1.0" encoding="UTF-8"?>
<?xml-stylesheet type="text/xsl" href="/feed-styles.xsl"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>{{ config.title }}</title>
    <link>{{ config.url }}</link>
    <description>{{ config.description }}</description>
    <atom:link href="{{ feed.url }}/rss.xml" rel="self" type="application/rss+xml"/>
    {% for post in feed.posts %}
    <item>
      <title>{{ post.title }}</title>
      <link>{{ post.absolute_url }}</link>
      <guid isPermaLink="true">{{ post.absolute_url }}</guid>
      <pubDate>{{ post.date | rfc822 }}</pubDate>
      <description>{{ post.description | escape }}</description>
    </item>
    {% endfor %}
  </channel>
</rss>
```

### Atom Template with Stylesheet

```xml
<?xml version="1.0" encoding="UTF-8"?>
<?xml-stylesheet type="text/xsl" href="/feed-styles.xsl"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>{{ config.title }}</title>
  <link href="{{ config.url }}"/>
  <link href="{{ feed.url }}/atom.xml" rel="self"/>
  <updated>{{ feed.updated | iso8601 }}</updated>
  <id>{{ config.url }}/</id>
  <author>
    <name>{{ config.author }}</name>
  </author>
  {% for post in feed.posts %}
  <entry>
    <title>{{ post.title }}</title>
    <link href="{{ post.absolute_url }}"/>
    <id>{{ post.absolute_url }}</id>
    <updated>{{ post.date | iso8601 }}</updated>
    <summary>{{ post.description | escape }}</summary>
  </entry>
  {% endfor %}
</feed>
```

### Feed Stylesheet Configuration

```toml
[markata-go.feeds]
# Enable styled RSS/Atom feeds (default: true)
xsl_stylesheet = true

# Custom stylesheet path (optional)
xsl_template = "templates/my-feed-styles.xsl"
```

### Feed Stylesheet Theming

The XSL stylesheet SHOULD use the same color palette as the main site:

```toml
[markata-go.theme]
palette = "catppuccin-mocha"  # RSS feed will use same colors
```

The stylesheet template receives palette colors as variables, ensuring visual consistency between your site and feed reader previews.

---

## Integration with Head/Style System

Themes provide the base visual identity. For additional customization, see [HEAD_STYLE.md](./HEAD_STYLE.md):

| System | Purpose | Example |
|--------|---------|---------|
| Theme (`[markata-go.theme]`) | Base templates, CSS, layouts | Selecting "blog" theme |
| Theme Variables (`[markata-go.theme.variables]`) | Quick CSS property overrides | `--color-primary: #8b5cf6` |
| Head Config (`[markata-go.head]`) | Meta tags, scripts, links | Analytics, fonts, OG tags |
| Style Config (`[markata-go.style]`) | Legacy color overrides | `color_bg`, `color_text` |
| Post Overrides (`config_overrides`) | Per-post customization | Custom scripts for one page |

**Resolution order:** Theme defaults → Theme variables → Head/Style config → Post overrides

---

## Theme Structure

A theme is a directory containing templates, CSS, and optionally assets:

```
themes/
└── default/
    ├── theme.toml           # Theme metadata
    ├── templates/
    │   ├── base.html        # Base layout
    │   ├── post.html        # Single post
    │   ├── feed.html        # Feed/index page
    │   ├── card.html        # Post card for feeds
    │   └── partials/
    │       ├── head.html
    │       ├── header.html
    │       ├── footer.html
    │       └── pagination.html
    ├── static/
    │   ├── css/
    │   │   ├── main.css     # Core styles
    │   │   ├── admonitions.css
    │   │   ├── code.css     # Syntax highlighting
    │   │   └── variables.css # CSS custom properties
    │   └── js/
    │       └── main.js      # Optional JavaScript
    └── assets/              # Theme-specific images, fonts
```

### theme.toml

```toml
[theme]
name = "Default"
version = "1.0.0"
description = "Clean, minimal theme with dark mode support"
author = "SSG Team"
license = "MIT"
homepage = "https://github.com/example/ssg-theme-default"

# Minimum SSG version required
min_version = "1.0.0"

# Theme supports these features
[theme.features]
dark_mode = true
responsive = true
syntax_highlighting = true
admonitions = true

# Configurable options this theme exposes
[theme.options]
primary_color = { type = "color", default = "#3b82f6", description = "Primary accent color" }
font_family = { type = "string", default = "system-ui", description = "Body font family" }
max_width = { type = "string", default = "65ch", description = "Content max width" }
show_toc = { type = "boolean", default = true, description = "Show table of contents" }
show_reading_time = { type = "boolean", default = true, description = "Show reading time" }

# Parent theme to inherit from (optional)
[theme.extends]
theme = "base"  # Inherit from another theme
```

---

## Theme Resolution

When looking for a template or static file, the system searches in order:

```
1. Project local:     ./templates/post.html
2. Project theme:     ./themes/[theme]/templates/post.html  
3. Installed theme:   ~/.config/markata-go/themes/[theme]/templates/post.html
4. Built-in theme:    [internal]/themes/[theme]/templates/post.html
5. Default theme:     [internal]/themes/default/templates/post.html
```

This allows users to override any theme file by creating it locally.

### Override Examples

**Override just the footer:**
```
my-site/
├── templates/
│   └── partials/
│       └── footer.html    # Your custom footer
├── posts/
└── markata-go.toml
```

**Override a CSS file:**
```
my-site/
├── static/
│   └── css/
│       └── admonitions.css  # Your custom admonition styles
├── posts/
└── markata-go.toml
```

---

## Configuration

### Selecting a Theme

```toml
[markata-go.theme]
name = "default"           # Theme name
```

### Customizing Theme Options

Themes expose configurable options:

```toml
[markata-go.theme]
name = "default"

# Theme-specific options (defined in theme.toml)
[markata-go.theme.options]
primary_color = "#8b5cf6"   # Purple instead of blue
font_family = "Inter, system-ui"
max_width = "70ch"
show_toc = false
```

### CSS Variable Overrides

For quick color/font changes without modifying CSS files:

```toml
[markata-go.theme]
name = "default"

# Override CSS custom properties
[markata-go.theme.variables]
"--color-primary" = "#8b5cf6"
"--color-primary-dark" = "#7c3aed"
"--font-body" = "Inter, system-ui"
"--font-heading" = "Inter, system-ui"
"--font-mono" = "JetBrains Mono, monospace"
"--content-width" = "70ch"
"--radius" = "0.5rem"
```

These are injected as a `<style>` block in the `<head>`:

```html
<style>
:root {
  --color-primary: #8b5cf6;
  --color-primary-dark: #7c3aed;
  --font-body: Inter, system-ui;
  /* ... */
}
</style>
```

### Custom CSS File

For more extensive customization:

```toml
[markata-go.theme]
name = "default"
custom_css = "my-styles.css"  # Loaded after theme CSS
```

---

## Built-in Themes

Implementations MUST provide at least the `default` theme.

### Default Theme

A clean, minimal theme with:
- Responsive layout
- Dark mode support (prefers-color-scheme)
- Styled admonitions
- Syntax highlighting
- Typography optimized for reading

### Minimal Theme (Optional)

Bare-bones HTML with minimal styling for:
- Maximum customization flexibility
- Fast loading
- Print-friendly

---

## Color Palette System

Color palettes are **separate from layout themes**. This separation allows users to:
- Apply any color palette to any layout theme
- Create custom palettes without touching theme structure
- Switch between light/dark variants easily

### Architecture: Three-Layer Color System

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 1: Raw Colors (Palette Definition)                   │
│  Pure color values with no semantic meaning                 │
│  e.g., red = "#f38ba8", blue = "#89b4fa"                   │
├─────────────────────────────────────────────────────────────┤
│  Layer 2: Semantic Colors (Role-Based)                      │
│  Colors mapped to meaning/purpose                           │
│  e.g., text = red, accent = blue, surface = base           │
├─────────────────────────────────────────────────────────────┤
│  Layer 3: Component Colors (Usage-Specific)                 │
│  Fine-grained component styling                             │
│  e.g., button-bg = accent, link-color = accent             │
└─────────────────────────────────────────────────────────────┘
```

### Palette File Format

Palettes are defined in TOML files within `palettes/` directory:

```toml
# palettes/catppuccin-mocha.toml
[palette]
name = "Catppuccin Mocha"
variant = "dark"              # "dark" | "light"
author = "Catppuccin Team"
license = "MIT"
homepage = "https://catppuccin.com"

# Layer 1: Raw Colors
# These are the source colors - pure values with no semantic meaning
[palette.colors]
rosewater = "#f5e0dc"
flamingo  = "#f2cdcd"
pink      = "#f5c2e7"
mauve     = "#cba6f7"
red       = "#f38ba8"
maroon    = "#eba0ac"
peach     = "#fab387"
yellow    = "#f9e2af"
green     = "#a6e3a1"
teal      = "#94e2d5"
sky       = "#89dceb"
sapphire  = "#74c7ec"
blue      = "#89b4fa"
lavender  = "#b4befe"
text      = "#cdd6f4"
subtext1  = "#bac2de"
subtext0  = "#a6adc8"
overlay2  = "#9399b2"
overlay1  = "#7f849c"
overlay0  = "#6c7086"
surface2  = "#585b70"
surface1  = "#45475a"
surface0  = "#313244"
base      = "#1e1e2e"
mantle    = "#181825"
crust     = "#11111b"

# Layer 2: Semantic Mapping
# Maps raw colors to semantic roles
[palette.semantic]
# Text colors
text-primary   = "text"       # Reference to raw color name
text-secondary = "subtext1"
text-muted     = "overlay1"

# Background colors
bg-primary   = "base"
bg-secondary = "mantle"
bg-surface   = "surface0"
bg-elevated  = "surface1"

# Accent colors
accent         = "mauve"
accent-hover   = "lavender"
link           = "blue"
link-hover     = "sapphire"
link-visited   = "mauve"

# Status colors
success = "green"
warning = "yellow"
error   = "red"
info    = "blue"

# Border colors
border         = "surface1"
border-focus   = "mauve"

# Button colors
button-primary-bg     = "mauve"
button-primary-text   = "base"
button-secondary-bg   = "surface1"
button-secondary-text = "text"

# Layer 3: Component Colors (optional overrides)
[palette.components]
# Code blocks
code-bg         = "surface0"
code-text       = "text"
code-comment    = "overlay1"
code-keyword    = "mauve"
code-string     = "green"
code-number     = "peach"
code-function   = "blue"

# Admonitions
admonition-note-bg     = "surface0"
admonition-note-border = "blue"
admonition-tip-bg      = "surface0"
admonition-tip-border  = "green"
admonition-warn-bg     = "surface0"
admonition-warn-border = "yellow"
admonition-error-bg    = "surface0"
admonition-error-border = "red"
```

### Configuration

#### Selecting a Palette

```toml
[markata-go.theme]
name = "default"              # Layout theme
palette = "catppuccin-mocha"  # Color palette

# Or with auto dark/light mode
[markata-go.theme]
name = "default"
palette = "catppuccin-latte"        # Light mode palette
palette_dark = "catppuccin-mocha"   # Dark mode palette (prefers-color-scheme)
```

#### Inline Palette Customization

Override specific colors without creating a new palette:

```toml
[markata-go.theme]
name = "default"
palette = "catppuccin-mocha"

# Override specific semantic colors
[markata-go.theme.palette]
accent = "#ff79c6"            # Use Dracula's pink as accent
link = "#8be9fd"              # Use Dracula's cyan for links
```

#### Custom Palette Definition

Define a complete custom palette inline:

```toml
[markata-go.theme]
name = "default"

# Full custom palette
[markata-go.theme.palette]
name = "My Brand"
variant = "light"

[markata-go.theme.palette.colors]
brand-primary = "#4f46e5"
brand-secondary = "#7c3aed"
# ... more colors

[markata-go.theme.palette.semantic]
accent = "brand-primary"
link = "brand-secondary"
# ... more mappings
```

### Built-in Palettes

Implementations MUST include at least `default-light` and `default-dark` palettes.
Implementations SHOULD include these popular community palettes (all MIT licensed):

| Palette | Variants | Description |
|---------|----------|-------------|
| `catppuccin-latte` | light | Soothing pastel theme |
| `catppuccin-frappe` | dark | Medium contrast dark |
| `catppuccin-macchiato` | dark | Higher contrast dark |
| `catppuccin-mocha` | dark | Highest contrast dark |
| `nord-light` | light | Arctic, north-bluish |
| `nord-dark` | dark | Arctic, north-bluish |
| `gruvbox-light` | light | Retro groove, warm |
| `gruvbox-dark` | dark | Retro groove, warm |
| `tokyo-night` | dark | Tokyo city lights |
| `tokyo-night-storm` | dark | Stormy variant |
| `tokyo-night-day` | light | Day variant |
| `rose-pine` | dark | Soho vibes, natural |
| `rose-pine-moon` | dark | Softer variant |
| `rose-pine-dawn` | light | Light variant |
| `everforest-light` | light | Nature-inspired green |
| `everforest-dark` | dark | Nature-inspired green |
| `dracula` | dark | Vibrant on dark |
| `solarized-light` | light | Scientifically designed |
| `solarized-dark` | dark | Scientifically designed |
| `kanagawa-wave` | dark | Japanese art inspired |
| `kanagawa-dragon` | dark | Warmer variant |
| `kanagawa-lotus` | light | Light variant |

### Palette Directory Structure

```
palettes/
├── catppuccin-latte.toml
├── catppuccin-frappe.toml
├── catppuccin-macchiato.toml
├── catppuccin-mocha.toml
├── nord-light.toml
├── nord-dark.toml
├── gruvbox-light.toml
├── gruvbox-dark.toml
├── tokyo-night.toml
├── tokyo-night-storm.toml
├── tokyo-night-day.toml
├── rose-pine.toml
├── rose-pine-moon.toml
├── rose-pine-dawn.toml
├── everforest-light.toml
├── everforest-dark.toml
├── dracula.toml
├── solarized-light.toml
├── solarized-dark.toml
├── kanagawa-wave.toml
├── kanagawa-dragon.toml
└── kanagawa-lotus.toml
```

### Palette Resolution Order

1. User's project: `./palettes/{name}.toml`
2. User config: `~/.config/markata-go/palettes/{name}.toml`
3. Built-in: `[internal]/palettes/{name}.toml`

### Default Palette Behavior

If no palette is specified in configuration:
- **Default palette:** `default-light` is used
- **Dark mode:** If `palette_dark` is not specified but the user's system prefers dark mode, and a matching dark variant exists (e.g., `default-dark` for `default-light`), it will be used automatically

```toml
# Explicit light/dark configuration (recommended)
[markata-go.theme]
palette = "catppuccin-latte"        # Light mode
palette_dark = "catppuccin-mocha"   # Dark mode

# Or single palette with no auto-switching
[markata-go.theme]
palette = "dracula"                  # Always use dracula, no light mode
```

### Color Reference Rules

Semantic and component colors follow strict reference rules:

| Layer | Can Reference | Cannot Reference |
|-------|---------------|------------------|
| Raw colors | (literal hex values) | - |
| Semantic colors | Raw colors only | Other semantic colors |
| Component colors | Raw colors, Semantic colors | Other component colors |

**Valid:**
```toml
[palette.semantic]
accent = "mauve"              # References raw color ✓

[palette.components]
button-bg = "accent"          # References semantic color ✓
button-bg = "mauve"           # References raw color ✓
```

**Invalid:**
```toml
[palette.semantic]
accent-hover = "accent"       # Cannot reference semantic ✗

[palette.components]
button-hover = "button-bg"    # Cannot reference component ✗
```

This prevents circular dependencies and ensures predictable color resolution.

### CSS Generation

Palettes are compiled to CSS custom properties:

```css
/* Generated from catppuccin-mocha palette */
:root {
  /* Raw colors */
  --palette-rosewater: #f5e0dc;
  --palette-flamingo: #f2cdcd;
  --palette-pink: #f5c2e7;
  /* ... all 26 Catppuccin colors */
  
  /* Semantic colors */
  --color-text-primary: var(--palette-text);
  --color-text-secondary: var(--palette-subtext1);
  --color-text-muted: var(--palette-overlay1);
  --color-bg-primary: var(--palette-base);
  --color-bg-secondary: var(--palette-mantle);
  --color-bg-surface: var(--palette-surface0);
  --color-accent: var(--palette-mauve);
  --color-link: var(--palette-blue);
  --color-success: var(--palette-green);
  --color-warning: var(--palette-yellow);
  --color-error: var(--palette-red);
  --color-info: var(--palette-blue);
  --color-border: var(--palette-surface1);
  
  /* Component colors */
  --code-bg: var(--palette-surface0);
  --code-text: var(--palette-text);
  --code-keyword: var(--palette-mauve);
  /* ... */
}

/* Dark mode override (if palette_dark specified) */
@media (prefers-color-scheme: dark) {
  :root {
    /* Dark palette colors override here */
  }
}
```

### Contrast Validation

Palettes SHOULD pass WCAG 2.1 contrast requirements:

| Combination | Minimum Ratio | Level |
|-------------|---------------|-------|
| text-primary on bg-primary | 4.5:1 | AA |
| text-secondary on bg-primary | 4.5:1 | AA |
| text-muted on bg-primary | 3:1 | AA Large |
| accent on bg-primary | 3:1 | AA Large |
| link on bg-primary | 4.5:1 | AA |

Implementations SHOULD provide a contrast validation command:

```bash
$ markata-go palette check catppuccin-mocha

Checking palette: Catppuccin Mocha

Contrast Ratios:
  text-primary on bg-primary:     11.8:1 ✓ (AA, AAA)
  text-secondary on bg-primary:    8.2:1 ✓ (AA, AAA)
  text-muted on bg-primary:        4.1:1 ✓ (AA Large)
  accent on bg-primary:            5.2:1 ✓ (AA)
  link on bg-primary:              6.4:1 ✓ (AA, AAA)
  error on bg-primary:             5.8:1 ✓ (AA)
  warning on bg-primary:           9.1:1 ✓ (AA, AAA)
  success on bg-primary:           7.3:1 ✓ (AA, AAA)

All contrast checks passed!
```

### Contrast Ratio Calculation

Implementations MUST use the WCAG 2.1 relative luminance formula for contrast calculations:

```go
// ContrastRatio calculates the WCAG 2.1 contrast ratio between two colors.
// Returns a value between 1:1 (same color) and 21:1 (black/white).
func ContrastRatio(fg, bg color.Color) float64 {
    l1 := RelativeLuminance(fg)
    l2 := RelativeLuminance(bg)
    
    // Ensure l1 is the lighter color
    if l1 < l2 {
        l1, l2 = l2, l1
    }
    
    return (l1 + 0.05) / (l2 + 0.05)
}

// RelativeLuminance calculates the relative luminance of a color.
// Based on WCAG 2.1 definition using sRGB color space.
func RelativeLuminance(c color.Color) float64 {
    r, g, b, _ := c.RGBA()
    
    // Convert to 0-1 range
    rLinear := linearize(float64(r) / 65535.0)
    gLinear := linearize(float64(g) / 65535.0)
    bLinear := linearize(float64(b) / 65535.0)
    
    // ITU-R BT.709 coefficients
    return 0.2126*rLinear + 0.7152*gLinear + 0.0722*bLinear
}

// linearize converts sRGB gamma-corrected value to linear RGB.
func linearize(v float64) float64 {
    if v <= 0.04045 {
        return v / 12.92
    }
    return math.Pow((v+0.055)/1.055, 2.4)
}
```

### WCAG Compliance Levels

| Level | Normal Text | Large Text | UI Components |
|-------|-------------|------------|---------------|
| A     | 3:1         | 3:1        | 3:1           |
| AA    | 4.5:1       | 3:1        | 3:1           |
| AAA   | 7:1         | 4.5:1      | 4.5:1         |

**Large text** is defined as:
- 18pt (24px) or larger for normal weight
- 14pt (18.5px) or larger for bold weight

### Required Contrast Checks

Implementations MUST validate these combinations:

```go
type ContrastCheck struct {
    Foreground string  // Semantic color name
    Background string  // Semantic color name
    MinRatio   float64 // Minimum required ratio
    Level      string  // "AA", "AAA", "AA Large"
}

var RequiredChecks = []ContrastCheck{
    // Primary text must be readable
    {"text-primary", "bg-primary", 4.5, "AA"},
    {"text-primary", "bg-surface", 4.5, "AA"},
    {"text-primary", "bg-elevated", 4.5, "AA"},
    
    // Secondary text
    {"text-secondary", "bg-primary", 4.5, "AA"},
    {"text-muted", "bg-primary", 3.0, "AA Large"},
    
    // Interactive elements
    {"link", "bg-primary", 4.5, "AA"},
    {"accent", "bg-primary", 3.0, "AA Large"},
    
    // Status colors (used for UI, so 3:1 minimum)
    {"success", "bg-primary", 3.0, "UI"},
    {"warning", "bg-primary", 3.0, "UI"},
    {"error", "bg-primary", 3.0, "UI"},
    {"info", "bg-primary", 3.0, "UI"},
    
    // Code blocks
    {"code-text", "code-bg", 4.5, "AA"},
    {"code-comment", "code-bg", 3.0, "AA Large"},
    {"code-keyword", "code-bg", 4.5, "AA"},
    
    // Buttons
    {"button-primary-text", "button-primary-bg", 4.5, "AA"},
    {"button-secondary-text", "button-secondary-bg", 4.5, "AA"},
}
```

### Testing Integration

Contrast validation SHOULD be available in multiple contexts:

**1. CLI Command:**
```bash
markata-go palette check <palette-name> [--strict]
```

**2. Go Test Helper:**
```go
func TestPaletteContrast(t *testing.T) {
    palette, err := palettes.Load("catppuccin-mocha")
    require.NoError(t, err)
    
    // Check all required combinations
    results := palette.CheckContrast()
    for _, r := range results {
        if !r.Passed {
            t.Errorf("%s on %s: got %.2f:1, want %.2f:1 (%s)",
                r.Foreground, r.Background, r.Ratio, r.Required, r.Level)
        }
    }
}
```

**3. CI Integration:**
```yaml
# .github/workflows/test.yml
- name: Validate palette contrast
  run: markata-go palette check --all --strict
```

### Accessibility Report

The `palette check` command SHOULD generate an accessibility report:

```bash
$ markata-go palette check catppuccin-mocha --report

=== Accessibility Report: Catppuccin Mocha ===

WCAG 2.1 Level AA Compliance: PASS

Text Readability:
  ✓ Primary text on backgrounds:     11.8:1 - 8.2:1 (excellent)
  ✓ Secondary text on backgrounds:    8.2:1 - 5.4:1 (good)
  ✓ Muted text on backgrounds:        4.1:1 - 3.2:1 (acceptable)

Interactive Elements:
  ✓ Links are distinguishable:        6.4:1 (AA compliant)
  ✓ Focus indicators visible:         5.2:1 (AA compliant)
  
Color Blindness Simulation:
  ✓ Protanopia:   Status colors distinguishable
  ✓ Deuteranopia: Status colors distinguishable  
  ✓ Tritanopia:   Status colors distinguishable

Recommendations:
  - Consider increasing muted text contrast for AAA compliance
  - Warning color could be slightly darker for better contrast

Full report saved to: .markata/palette-report-catppuccin-mocha.html
```

---

## Feature Flags

Themes can expose optional features that users can enable/disable:

```toml
[markata-go.theme]
name = "default"

[markata-go.theme.features]
dark_mode = true              # Enable dark mode toggle
toc = true                    # Show table of contents
reading_time = true           # Show estimated reading time
copy_code = true              # Add copy button to code blocks
back_to_top = true            # Show back-to-top button
search = false                # Disable search (if theme supports it)
comments = false              # Disable comments integration
```

### Feature Implementation

Themes declare supported features in `theme.toml`:

```toml
[theme.features]
dark_mode = { default = true, description = "Dark mode toggle in header" }
toc = { default = true, description = "Table of contents sidebar" }
reading_time = { default = true, description = "Reading time estimate" }
copy_code = { default = true, description = "Copy button on code blocks" }
back_to_top = { default = false, description = "Floating back-to-top button" }
search = { default = false, description = "Site search functionality" }
comments = { default = false, description = "Comments via Giscus/Utterances" }
```

Features are available in templates:

```jinja2
{% if features.toc and post.toc %}
<aside class="toc">
  {{ post.toc | safe }}
</aside>
{% endif %}

{% if features.reading_time %}
<span class="reading-time">{{ post.reading_time }} min read</span>
{% endif %}

{% if features.copy_code %}
<script src="{{ 'js/copy-code.js' | theme_asset }}"></script>
{% endif %}
```

---

## CSS Custom Properties

Built-in themes SHOULD use CSS custom properties for consistency:

### Colors

```css
:root {
  /* Primary brand color */
  --color-primary: #3b82f6;
  --color-primary-light: #60a5fa;
  --color-primary-dark: #2563eb;
  
  /* Semantic colors */
  --color-text: #1f2937;
  --color-text-muted: #6b7280;
  --color-background: #ffffff;
  --color-surface: #f9fafb;
  --color-border: #e5e7eb;
  
  /* Status colors */
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-error: #ef4444;
  --color-info: #3b82f6;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  :root {
    --color-text: #f9fafb;
    --color-text-muted: #9ca3af;
    --color-background: #111827;
    --color-surface: #1f2937;
    --color-border: #374151;
  }
}
```

### Typography

```css
:root {
  /* Font families */
  --font-body: system-ui, -apple-system, sans-serif;
  --font-heading: var(--font-body);
  --font-mono: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
  
  /* Font sizes (modular scale) */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;
  --text-2xl: 1.5rem;
  --text-3xl: 1.875rem;
  --text-4xl: 2.25rem;
  
  /* Line heights */
  --leading-tight: 1.25;
  --leading-normal: 1.5;
  --leading-relaxed: 1.75;
}
```

### Spacing

```css
:root {
  /* Spacing scale */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;
  --space-12: 3rem;
  --space-16: 4rem;
  
  /* Layout */
  --content-width: 65ch;
  --page-width: 1200px;
  --radius: 0.375rem;
  --radius-lg: 0.5rem;
}
```

---

## Admonition Styles

Built-in themes MUST include styles for all admonition types.

### Admonition CSS

```css
/* Base admonition styles */
.admonition {
  margin: var(--space-4) 0;
  padding: var(--space-4);
  border-left: 4px solid var(--admonition-color, var(--color-primary));
  background: var(--admonition-bg, var(--color-surface));
  border-radius: 0 var(--radius) var(--radius) 0;
}

.admonition-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: 600;
  margin-bottom: var(--space-2);
  color: var(--admonition-color, var(--color-primary));
}

.admonition-title::before {
  content: var(--admonition-icon, "");
  font-size: 1.25em;
}

/* Type-specific styles */
.admonition.note {
  --admonition-color: #3b82f6;
  --admonition-bg: #eff6ff;
  --admonition-icon: "ℹ️";
}

.admonition.info {
  --admonition-color: #3b82f6;
  --admonition-bg: #eff6ff;
  --admonition-icon: "ℹ️";
}

.admonition.tip {
  --admonition-color: #10b981;
  --admonition-bg: #ecfdf5;
  --admonition-icon: "💡";
}

.admonition.hint {
  --admonition-color: #10b981;
  --admonition-bg: #ecfdf5;
  --admonition-icon: "💡";
}

.admonition.success {
  --admonition-color: #10b981;
  --admonition-bg: #ecfdf5;
  --admonition-icon: "✅";
}

.admonition.warning {
  --admonition-color: #f59e0b;
  --admonition-bg: #fffbeb;
  --admonition-icon: "⚠️";
}

.admonition.caution {
  --admonition-color: #f59e0b;
  --admonition-bg: #fffbeb;
  --admonition-icon: "⚠️";
}

.admonition.danger {
  --admonition-color: #ef4444;
  --admonition-bg: #fef2f2;
  --admonition-icon: "🚨";
}

.admonition.error {
  --admonition-color: #ef4444;
  --admonition-bg: #fef2f2;
  --admonition-icon: "❌";
}

.admonition.bug {
  --admonition-color: #ef4444;
  --admonition-bg: #fef2f2;
  --admonition-icon: "🐛";
}

.admonition.example {
  --admonition-color: #8b5cf6;
  --admonition-bg: #f5f3ff;
  --admonition-icon: "📝";
}

.admonition.quote {
  --admonition-color: #6b7280;
  --admonition-bg: #f9fafb;
  --admonition-icon: "💬";
}

.admonition.abstract {
  --admonition-color: #06b6d4;
  --admonition-bg: #ecfeff;
  --admonition-icon: "📋";
}

/* Dark mode adjustments */
@media (prefers-color-scheme: dark) {
  .admonition.note { --admonition-bg: #1e3a5f; }
  .admonition.info { --admonition-bg: #1e3a5f; }
  .admonition.tip { --admonition-bg: #064e3b; }
  .admonition.hint { --admonition-bg: #064e3b; }
  .admonition.success { --admonition-bg: #064e3b; }
  .admonition.warning { --admonition-bg: #451a03; }
  .admonition.caution { --admonition-bg: #451a03; }
  .admonition.danger { --admonition-bg: #450a0a; }
  .admonition.error { --admonition-bg: #450a0a; }
  .admonition.bug { --admonition-bg: #450a0a; }
  .admonition.example { --admonition-bg: #2e1065; }
  .admonition.quote { --admonition-bg: #1f2937; }
  .admonition.abstract { --admonition-bg: #083344; }
}
```

### Aside (Sidebar) Styles

```css
/* Aside - Sidebar/Marginal Note */
aside.admonition.aside {
  --admonition-color: #6b7280;
  --admonition-bg: #f9fafb;
  border-left: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: var(--text-sm);
  max-width: 280px;
}

/* Default: float right */
aside.admonition.aside {
  float: right;
  margin: 0 0 var(--space-4) var(--space-4);
  clear: right;
}

/* Inline left modifier */
aside.admonition.aside.aside-inline {
  float: left;
  margin: 0 var(--space-4) var(--space-4) 0;
  clear: left;
}

/* Inline end (right) modifier - explicit */
aside.admonition.aside.aside-inline-end {
  float: right;
  margin: 0 0 var(--space-4) var(--space-4);
  clear: right;
}

/* Aside title styling */
aside.admonition.aside .admonition-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}

aside.admonition.aside .admonition-title::before {
  content: none; /* No icon for asides */
}

/* Responsive: full width on small screens */
@media (max-width: 768px) {
  aside.admonition.aside,
  aside.admonition.aside.aside-inline,
  aside.admonition.aside.aside-inline-end {
    float: none;
    max-width: 100%;
    margin: var(--space-4) 0;
  }
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  aside.admonition.aside {
    --admonition-bg: #1f2937;
  }
}
```

### Chat/Conversation Styles

```css
/* Chat container */
.chat-container {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: var(--space-4) 0;
  padding: var(--space-4);
  background: var(--color-surface);
  border-radius: var(--radius-lg);
}

/* Individual message */
.chat-message {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
  max-width: 85%;
}

/* Left-aligned messages */
.chat-message.chat-left {
  align-self: flex-start;
  flex-direction: row;
}

/* Right-aligned messages */
.chat-message.chat-right {
  align-self: flex-end;
  flex-direction: row-reverse;
}

/* Avatar */
.chat-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background-color: var(--color-primary);
  background-size: cover;
  background-position: center;
  flex-shrink: 0;
}

/* Message bubble */
.chat-bubble {
  padding: var(--space-3);
  border-radius: var(--radius-lg);
  position: relative;
}

/* Left bubble styling */
.chat-left .chat-bubble {
  background: #e5e7eb;
  color: #1f2937;
  border-bottom-left-radius: var(--radius);
}

/* Right bubble styling */
.chat-right .chat-bubble {
  background: var(--color-primary);
  color: white;
  border-bottom-right-radius: var(--radius);
}

/* Author name */
.chat-author {
  font-size: var(--text-xs);
  font-weight: 600;
  margin-bottom: var(--space-1);
  opacity: 0.8;
}

.chat-left .chat-author {
  color: var(--color-text-muted);
}

.chat-right .chat-author {
  color: rgba(255, 255, 255, 0.8);
}

/* Message content */
.chat-content {
  font-size: var(--text-sm);
  line-height: var(--leading-normal);
}

.chat-content p {
  margin: 0;
}

.chat-content p + p {
  margin-top: var(--space-2);
}

/* Timestamp */
.chat-timestamp {
  font-size: var(--text-xs);
  margin-top: var(--space-1);
  opacity: 0.6;
}

.chat-left .chat-timestamp {
  color: var(--color-text-muted);
}

.chat-right .chat-timestamp {
  color: rgba(255, 255, 255, 0.7);
}

/* System messages */
.chat-message.chat-system {
  align-self: center;
  max-width: 100%;
}

.chat-system .chat-content {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  background: transparent;
  padding: var(--space-2) var(--space-4);
  text-align: center;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  .chat-container {
    background: var(--color-surface);
  }
  
  .chat-left .chat-bubble {
    background: #374151;
    color: #f9fafb;
  }
  
  .chat-left .chat-author {
    color: #9ca3af;
  }
}

/* Responsive adjustments */
@media (max-width: 480px) {
  .chat-message {
    max-width: 95%;
  }
  
  .chat-avatar {
    width: 28px;
    height: 28px;
  }
}
```

### Collapsible Admonitions

```css
/* Collapsible admonitions use <details>/<summary> */
details.admonition {
  border-left: 4px solid var(--admonition-color, var(--color-primary));
}

details.admonition > summary {
  cursor: pointer;
  list-style: none;
}

details.admonition > summary::before {
  content: "▶";
  display: inline-block;
  margin-right: var(--space-2);
  transition: transform 0.2s;
}

details.admonition[open] > summary::before {
  transform: rotate(90deg);
}
```

---

## Code Block Styles

### Base Code Styles

```css
/* Inline code */
code {
  font-family: var(--font-mono);
  font-size: 0.9em;
  padding: 0.125em 0.375em;
  background: var(--color-surface);
  border-radius: var(--radius);
}

/* Code blocks */
pre {
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  padding: var(--space-4);
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  overflow-x: auto;
}

pre code {
  padding: 0;
  background: none;
}

/* Line numbers (optional) */
pre.line-numbers {
  padding-left: 3.5em;
  position: relative;
}

pre.line-numbers::before {
  content: attr(data-line-numbers);
  position: absolute;
  left: 0;
  top: var(--space-4);
  width: 3em;
  text-align: right;
  color: var(--color-text-muted);
  border-right: 1px solid var(--color-border);
  padding-right: var(--space-2);
  user-select: none;
}

/* Language label */
pre[data-language]::after {
  content: attr(data-language);
  position: absolute;
  top: var(--space-2);
  right: var(--space-2);
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  text-transform: uppercase;
}
```

### Syntax Highlighting Themes

Implementations SHOULD support multiple syntax highlighting themes:

| Theme | Description |
|-------|-------------|
| `github-light` | GitHub's light theme |
| `github-dark` | GitHub's dark theme |
| `monokai` | Classic dark theme |
| `one-dark` | Atom's One Dark |
| `dracula` | Dracula theme |
| `nord` | Nord color palette |
| `solarized-light` | Solarized light |
| `solarized-dark` | Solarized dark |

Configuration:

```toml
[markata-go.markdown.highlight]
theme = "github-dark"
line_numbers = false
```

---

## Template Requirements

### Base Template (`base.html`)

Every theme MUST provide a base template with these blocks:

```jinja2
<!DOCTYPE html>
<html lang="{{ config.lang | default('en') }}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  
  {% block meta %}
  <title>{% block title %}{{ config.title }}{% endblock %}</title>
  <meta name="description" content="{% block description %}{{ config.description }}{% endblock %}">
  {% endblock %}
  
  {% block head %}
  {# Theme CSS #}
  <link rel="stylesheet" href="{{ 'css/main.css' | theme_asset }}">
  
  {# CSS variable overrides from config #}
  {% if config.theme.variables %}
  <style>:root { {% for k, v in config.theme.variables.items() %}{{ k }}: {{ v }}; {% endfor %} }</style>
  {% endif %}
  
  {# Custom CSS #}
  {% if config.theme.custom_css %}
  <link rel="stylesheet" href="{{ config.theme.custom_css | asset_url }}">
  {% endif %}
  {% endblock %}
</head>
<body>
  {% block header %}
  {% include "partials/header.html" %}
  {% endblock %}
  
  <main>
    {% block content %}{% endblock %}
  </main>
  
  {% block footer %}
  {% include "partials/footer.html" %}
  {% endblock %}
  
  {% block scripts %}{% endblock %}
</body>
</html>
```

### Post Template (`post.html`)

```jinja2
{% extends "base.html" %}

{% block title %}{{ post.title }} | {{ config.title }}{% endblock %}
{% block description %}{{ post.description | default(post.content | truncate(160)) }}{% endblock %}

{% block meta %}
{{ super() }}
<meta property="og:title" content="{{ post.title }}">
<meta property="og:type" content="article">
<meta property="og:url" content="{{ post.absolute_url }}">
{% if post.date %}
<meta property="article:published_time" content="{{ post.date | isoformat }}">
{% endif %}
{% endblock %}

{% block content %}
<article class="post">
  <header class="post-header">
    <h1>{{ post.title }}</h1>
    {% if post.date or post.reading_time %}
    <div class="post-meta">
      {% if post.date %}<time datetime="{{ post.date | isoformat }}">{{ post.date | date }}</time>{% endif %}
      {% if post.reading_time %}<span>{{ post.reading_time }} min read</span>{% endif %}
    </div>
    {% endif %}
  </header>
  
  <div class="post-content">
    {{ body | safe }}
  </div>
  
  {% if post.tags %}
  <footer class="post-footer">
    <div class="tags">
      {% for tag in post.tags %}
      <a href="/tags/{{ tag | slugify }}/" class="tag">{{ tag }}</a>
      {% endfor %}
    </div>
  </footer>
  {% endif %}
</article>

{% if post.prev or post.next %}
<nav class="post-nav">
  {% if post.prev %}<a href="{{ post.prev.href }}" class="prev">← {{ post.prev.title }}</a>{% endif %}
  {% if post.next %}<a href="{{ post.next.href }}" class="next">{{ post.next.title }} →</a>{% endif %}
</nav>
{% endif %}
{% endblock %}
```

### Feed Template (`feed.html`)

```jinja2
{% extends "base.html" %}

{% block title %}{{ feed.title | default(config.title) }}{% endblock %}

{% block content %}
<div class="feed">
  {% if feed.title %}
  <header class="feed-header">
    <h1>{{ feed.title }}</h1>
    {% if feed.description %}<p>{{ feed.description }}</p>{% endif %}
  </header>
  {% endif %}
  
  <div class="posts">
    {% for post in feed.posts %}
    {% include "card.html" %}
    {% endfor %}
  </div>
  
  {% if feed.pagination.total_pages > 1 %}
  {% include "partials/pagination.html" %}
  {% endif %}
</div>
{% endblock %}
```

### Card Template (`card.html`)

```jinja2
<article class="card">
  <h2><a href="{{ post.href }}">{{ post.title }}</a></h2>
  {% if post.description %}
  <p class="card-description">{{ post.description }}</p>
  {% endif %}
  <div class="card-meta">
    {% if post.date %}<time datetime="{{ post.date | isoformat }}">{{ post.date | date }}</time>{% endif %}
    {% if post.reading_time %}<span>{{ post.reading_time }} min</span>{% endif %}
  </div>
</article>
```

---

## Template Filters

Themes should have access to these filters:

| Filter | Description | Example |
|--------|-------------|---------|
| `theme_asset` | URL to theme static file | `{{ 'css/main.css' \| theme_asset }}` |
| `asset_url` | URL to project static file | `{{ 'images/logo.png' \| asset_url }}` |
| `date` | Format date | `{{ post.date \| date('%B %d, %Y') }}` |
| `isoformat` | ISO 8601 date | `{{ post.date \| isoformat }}` |
| `truncate` | Truncate text | `{{ text \| truncate(160) }}` |
| `slugify` | Generate slug | `{{ title \| slugify }}` |
| `safe` | Mark as safe HTML | `{{ body \| safe }}` |

---

## Installing Themes

### From Package Manager

```bash
# Python
pip install [name]-theme-blog

# npm
npm install @[name]/theme-blog

# Direct download
[name] theme install https://github.com/user/theme-blog
```

### Manual Installation

Download theme to `~/.config/[name]/themes/` or `./themes/`:

```bash
mkdir -p themes
git clone https://github.com/user/theme-blog themes/blog
```

### Configuration

```toml
[markata-go.theme]
name = "blog"  # Matches directory name
```

---

## Creating a Theme

### Scaffold a New Theme

```bash
[name] theme new my-theme
```

Creates:
```
themes/my-theme/
├── theme.toml
├── templates/
│   ├── base.html
│   ├── post.html
│   ├── feed.html
│   └── card.html
└── static/
    └── css/
        ├── main.css
        └── variables.css
```

### Extending a Theme

Inherit from an existing theme and override only what you need:

```toml
# themes/my-theme/theme.toml
[theme]
name = "My Theme"

[theme.extends]
theme = "default"
```

Then only include the files you want to override:

```
themes/my-theme/
├── theme.toml
├── templates/
│   └── partials/
│       └── footer.html    # Custom footer only
└── static/
    └── css/
        └── variables.css  # Custom colors only
```

---

## CLI Commands

### `theme list`

List available themes:

```bash
$ markata-go theme list

Installed themes:
  default     Clean, minimal theme with dark mode (built-in)
  minimal     Bare-bones HTML for maximum customization (built-in)
  blog        Feature-rich blog theme (~/.config/markata-go/themes/blog)
  
Current theme: default
```

### `theme info`

Show theme details:

```bash
$ markata-go theme info blog

Name: Blog Theme
Version: 2.1.0
Author: Theme Author
License: MIT
Homepage: https://github.com/user/theme-blog

Features:
  ✓ Dark mode
  ✓ Responsive
  ✓ Syntax highlighting
  ✓ Admonitions

Options:
  primary_color   Color   #3b82f6   Primary accent color
  font_family     String  system-ui Body font family
  show_toc        Boolean true      Show table of contents
```

### `theme install`

Install a theme:

```bash
$ markata-go theme install https://github.com/user/theme-blog
Installing theme from https://github.com/user/theme-blog...
Theme 'blog' installed to ~/.config/markata-go/themes/blog
```

### `theme new`

Create a new theme:

```bash
$ markata-go theme new my-theme
Created theme scaffold in themes/my-theme/
```

### `palette list`

List available color palettes from all sources.

**Usage:**
```bash
markata-go palette list [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--variant` | `-v` | Filter by variant (`light` or `dark`) |
| `--json` | | Output as JSON |

**Output:**
```bash
$ markata-go palette list

Built-in palettes:
  default-light         Light   Clean, minimal light theme
  default-dark          Dark    Clean, minimal dark theme
  catppuccin-latte      Light   Soothing pastel theme
  catppuccin-frappe     Dark    Medium contrast dark
  catppuccin-macchiato  Dark    Higher contrast dark
  catppuccin-mocha      Dark    Highest contrast dark
  nord-light            Light   Arctic, north-bluish
  nord-dark             Dark    Arctic, north-bluish
  gruvbox-light         Light   Retro groove, warm
  gruvbox-dark          Dark    Retro groove, warm
  dracula               Dark    Vibrant on dark
  rose-pine             Dark    Soho vibes, natural
  tokyo-night           Dark    Tokyo city lights
  solarized-dark        Dark    Scientifically designed

User palettes:
  my-brand              Light   ./palettes/my-brand.toml

Current: catppuccin-mocha (from markata-go.toml)
```

**Filtered output:**
```bash
$ markata-go palette list --variant light

Light palettes:
  default-light         Clean, minimal light theme
  catppuccin-latte      Soothing pastel theme
  nord-light            Arctic, north-bluish
  gruvbox-light         Retro groove, warm
  my-brand              ./palettes/my-brand.toml
```

**JSON output:**
```bash
$ markata-go palette list --json
```
```json
{
  "current": "catppuccin-mocha",
  "palettes": [
    {
      "name": "catppuccin-mocha",
      "variant": "dark",
      "description": "Highest contrast dark",
      "source": "built-in",
      "author": "Catppuccin Team"
    }
  ]
}
```

**Implementation notes:**
- Scan directories in order: `./palettes/`, `~/.config/markata-go/palettes/`, built-in
- Read `[palette]` section from each TOML file for metadata
- Current palette determined from `markata-go.toml` config

---

### `palette info`

Show detailed information about a specific palette.

**Usage:**
```bash
markata-go palette info <name> [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--json` | | Output as JSON |
| `--colors` | `-c` | Show only raw colors |
| `--semantic` | `-s` | Show only semantic mappings |

**Output:**
```bash
$ markata-go palette info catppuccin-mocha

Name:     Catppuccin Mocha
Variant:  dark
Author:   Catppuccin Team
License:  MIT
Homepage: https://catppuccin.com
Source:   [built-in]

Raw Colors (26):
  rosewater  #f5e0dc    flamingo   #f2cdcd    pink       #f5c2e7
  mauve      #cba6f7    red        #f38ba8    maroon     #eba0ac
  peach      #fab387    yellow     #f9e2af    green      #a6e3a1
  teal       #94e2d5    sky        #89dceb    sapphire   #74c7ec
  blue       #89b4fa    lavender   #b4befe    text       #cdd6f4
  subtext1   #bac2de    subtext0   #a6adc8    overlay2   #9399b2
  overlay1   #7f849c    overlay0   #6c7086    surface2   #585b70
  surface1   #45475a    surface0   #313244    base       #1e1e2e
  mantle     #181825    crust      #11111b

Semantic Mappings:
  text-primary    -> text (#cdd6f4)
  text-secondary  -> subtext1 (#bac2de)
  text-muted      -> overlay1 (#7f849c)
  bg-primary      -> base (#1e1e2e)
  bg-secondary    -> mantle (#181825)
  bg-surface      -> surface0 (#313244)
  bg-elevated     -> surface1 (#45475a)
  accent          -> mauve (#cba6f7)
  accent-hover    -> lavender (#b4befe)
  link            -> blue (#89b4fa)
  link-hover      -> sapphire (#74c7ec)
  link-visited    -> mauve (#cba6f7)
  success         -> green (#a6e3a1)
  warning         -> yellow (#f9e2af)
  error           -> red (#f38ba8)
  info            -> blue (#89b4fa)
  border          -> surface1 (#45475a)
  border-focus    -> mauve (#cba6f7)

Component Colors:
  code-bg         -> surface0 (#313244)
  code-text       -> text (#cdd6f4)
  code-keyword    -> mauve (#cba6f7)
  code-string     -> green (#a6e3a1)
  code-function   -> blue (#89b4fa)
  ...
```

**Implementation notes:**
- Load palette TOML file
- Resolve all semantic references to actual hex values
- Display in three columns for raw colors (fits 80-char terminal)

---

### `palette check`

Validate palette contrast ratios against WCAG 2.1 accessibility guidelines.

**Usage:**
```bash
markata-go palette check [name] [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--strict` | | Require AAA compliance (7:1 for text) |
| `--all` | `-a` | Check all installed palettes |
| `--json` | | Output as JSON |
| `--report` | `-r` | Generate detailed HTML report |
| `--fix` | | Suggest color adjustments to pass |

**Basic check:**
```bash
$ markata-go palette check catppuccin-mocha

Checking palette: Catppuccin Mocha

Text Contrast (AA requires 4.5:1):
  text-primary on bg-primary:      11.79:1  [PASS] AA, AAA
  text-secondary on bg-primary:     8.21:1  [PASS] AA, AAA
  text-muted on bg-primary:         4.08:1  [PASS] AA Large (3:1)
  text-primary on bg-surface:       9.42:1  [PASS] AA, AAA
  text-primary on bg-elevated:      7.51:1  [PASS] AA, AAA

Interactive Elements (AA requires 4.5:1, UI 3:1):
  link on bg-primary:               6.41:1  [PASS] AA, AAA
  link-hover on bg-primary:         5.87:1  [PASS] AA
  accent on bg-primary:             5.18:1  [PASS] AA
  border-focus on bg-primary:       5.18:1  [PASS] UI

Status Colors (UI requires 3:1):
  success on bg-primary:            7.28:1  [PASS]
  warning on bg-primary:            9.14:1  [PASS]
  error on bg-primary:              5.84:1  [PASS]
  info on bg-primary:               6.41:1  [PASS]

Code Blocks (AA requires 4.5:1):
  code-text on code-bg:             9.42:1  [PASS] AA, AAA
  code-comment on code-bg:          3.25:1  [PASS] AA Large
  code-keyword on code-bg:          4.12:1  [PASS] AA Large
  code-string on code-bg:           5.79:1  [PASS] AA
  code-function on code-bg:         5.10:1  [PASS] AA

Buttons:
  button-primary-text on button-primary-bg:    11.79:1  [PASS] AA, AAA
  button-secondary-text on button-secondary-bg: 9.42:1  [PASS] AA, AAA

Summary: 19/19 checks passed
WCAG 2.1 Level AA: COMPLIANT
```

**Strict mode (AAA):**
```bash
$ markata-go palette check catppuccin-mocha --strict

Checking palette: Catppuccin Mocha (AAA strict mode)

Text Contrast (AAA requires 7:1):
  text-primary on bg-primary:      11.79:1  [PASS] AAA
  text-secondary on bg-primary:     8.21:1  [PASS] AAA
  text-muted on bg-primary:         4.08:1  [FAIL] AAA requires 4.5:1
  ...

Summary: 15/19 checks passed
WCAG 2.1 Level AAA: NOT COMPLIANT

Failed checks:
  - text-muted on bg-primary: 4.08:1 (need 4.5:1, adjust to #8a8faa)
  - code-comment on code-bg: 3.25:1 (need 4.5:1, adjust to #9399b2)
```

**Check all palettes:**
```bash
$ markata-go palette check --all

Checking all installed palettes...

  default-light         [PASS] AA compliant (19/19)
  default-dark          [PASS] AA compliant (19/19)
  catppuccin-latte      [PASS] AA compliant (19/19)
  catppuccin-mocha      [PASS] AA compliant (19/19)
  nord-dark             [PASS] AA compliant (19/19)
  gruvbox-dark          [PASS] AA compliant (19/19)
  dracula               [WARN] 1 marginal (18/19, code-comment: 3.01:1)
  rose-pine             [PASS] AA compliant (19/19)
  tokyo-night           [PASS] AA compliant (19/19)
  solarized-dark        [PASS] AA compliant (19/19)
  my-brand              [FAIL] 3 failures (16/19)

Summary: 10/11 palettes fully AA compliant
         1 palette has failures (run with palette name for details)
```

**Generate report:**
```bash
$ markata-go palette check catppuccin-mocha --report

=== Accessibility Report: Catppuccin Mocha ===

WCAG 2.1 Level AA Compliance: PASS

Text Readability:
  Excellent (7:1+):  text-primary, text-secondary
  Good (4.5:1+):     link, accent, status colors
  Acceptable (3:1+): text-muted, code-comment

Interactive Elements:
  [PASS] Links are distinguishable from text
  [PASS] Focus indicators meet 3:1 contrast
  [PASS] Buttons have sufficient contrast

Color Blindness Simulation:
  [PASS] Protanopia:   Status colors distinguishable
  [PASS] Deuteranopia: Status colors distinguishable
  [PASS] Tritanopia:   Status colors distinguishable

Recommendations:
  - Consider increasing text-muted contrast for AAA compliance
  - code-comment is marginal; consider lightening to #9399b2

Report saved: .markata/palette-report-catppuccin-mocha.html
```

**JSON output:**
```bash
$ markata-go palette check catppuccin-mocha --json
```
```json
{
  "palette": "catppuccin-mocha",
  "compliant": true,
  "level": "AA",
  "checks": [
    {
      "foreground": "text-primary",
      "background": "bg-primary",
      "fg_color": "#cdd6f4",
      "bg_color": "#1e1e2e",
      "ratio": 11.79,
      "required": 4.5,
      "level": "AA",
      "passed": true,
      "grades": ["AA", "AAA"]
    }
  ],
  "summary": {
    "total": 19,
    "passed": 19,
    "failed": 0
  }
}
```

**Implementation notes:**
- Use WCAG 2.1 relative luminance formula (see Contrast Ratio Calculation section)
- Check all combinations defined in RequiredChecks
- `--fix` uses binary search to find minimum luminance adjustment while preserving hue

**Exit codes:**

| Scenario | Exit Code |
|----------|-----------|
| All checks pass | 0 |
| Any AA failure (default mode) | 1 |
| AA passes, AAA fails (no `--strict`) | 0 |
| AA passes, AAA fails (with `--strict`) | 1 |
| `--all`: All palettes pass | 0 |
| `--all`: Any palette fails | 1 |

---

### `palette preview`

Generate an HTML preview page showing the palette applied to sample content.

**Usage:**
```bash
markata-go palette preview <name> [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output file path (default: `.markata/palette-preview-{name}.html`) |
| `--open` | | Open in default browser after generating |
| `--compare` | `-c` | Compare with another palette side-by-side |

**Basic preview:**
```bash
$ markata-go palette preview catppuccin-mocha

Generated: .markata/palette-preview-catppuccin-mocha.html

Preview includes:
  - Color swatches with names and hex values
  - Typography samples (headings, body, code)
  - Code block with syntax highlighting
  - All admonition types
  - Button and form components
  - Card and navigation examples

Open in browser to view.
```

**Open automatically:**
```bash
$ markata-go palette preview catppuccin-mocha --open

Generated: .markata/palette-preview-catppuccin-mocha.html
Opening in default browser...
```

**Compare two palettes:**
```bash
$ markata-go palette preview catppuccin-mocha --compare catppuccin-latte

Generated: .markata/palette-compare-catppuccin-mocha-vs-catppuccin-latte.html

Side-by-side comparison of:
  - catppuccin-mocha (dark)
  - catppuccin-latte (light)
```

**Preview HTML structure:**

The generated HTML includes:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <title>Palette Preview: Catppuccin Mocha</title>
  <style>
    :root {
      /* All palette CSS variables injected here */
      --palette-base: #1e1e2e;
      --color-text-primary: var(--palette-text);
      /* ... */
    }
  </style>
</head>
<body>
  <header>
    <h1>Catppuccin Mocha</h1>
    <p>Dark variant by Catppuccin Team</p>
  </header>
  
  <section id="colors">
    <h2>Raw Colors</h2>
    <!-- Color swatches grid -->
  </section>
  
  <section id="typography">
    <h2>Typography</h2>
    <!-- Heading samples, body text, links -->
  </section>
  
  <section id="code">
    <h2>Code Blocks</h2>
    <!-- Sample code with syntax highlighting -->
  </section>
  
  <section id="admonitions">
    <h2>Admonitions</h2>
    <!-- All admonition types -->
  </section>
  
  <section id="components">
    <h2>Components</h2>
    <!-- Buttons, cards, forms -->
  </section>
  
  <section id="contrast">
    <h2>Contrast Ratios</h2>
    <!-- Visual contrast check results -->
  </section>
</body>
</html>
```

**Implementation notes:**
- Use embedded HTML template
- Inject palette as CSS custom properties
- Include sample content demonstrating all components
- For `--compare`, use CSS grid with two columns

---

### `palette new`

Create a new custom palette from scratch or based on an existing one.

**Usage:**
```bash
markata-go palette new <name> [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--variant` | `-v` | Variant type: `light` or `dark` (default: `dark`) |
| `--from` | `-f` | Base palette to copy from |
| `--output` | `-o` | Output directory (default: `./palettes/`) |
| `--minimal` | `-m` | Create minimal palette (semantic only, no components) |

**Create from scratch:**
```bash
$ markata-go palette new my-brand --variant light

Created: palettes/my-brand.toml

The palette has been initialized with placeholder colors.
Edit the file to customize your brand colors.

Next steps:
  1. Edit palettes/my-brand.toml
  2. Validate: markata-go palette check my-brand
  3. Preview:  markata-go palette preview my-brand --open
  4. Use:      Add 'palette = "my-brand"' to markata-go.toml
```

**Create from existing palette:**
```bash
$ markata-go palette new my-brand --from catppuccin-mocha

Created: palettes/my-brand.toml (based on catppuccin-mocha)

The palette has been copied from catppuccin-mocha.
Customize the colors to match your brand.

Changes from base:
  - Updated [palette] metadata (name, author)
  - All colors preserved - edit as needed
```

**Create minimal palette:**
```bash
$ markata-go palette new my-brand --minimal --variant dark

Created: palettes/my-brand.toml (minimal)

Minimal palette with only semantic colors.
Component colors will inherit from semantic defaults.
```

**Generated palette structure:**

Full palette (default):
```toml
# palettes/my-brand.toml
# Generated by markata-go palette new

[palette]
name = "My Brand"
variant = "dark"
author = "Your Name"
license = "MIT"
homepage = ""

# Layer 1: Raw Colors
# Define your brand colors here
[palette.colors]
# Primary brand colors
primary     = "#3b82f6"   # TODO: Replace with your primary color
secondary   = "#8b5cf6"   # TODO: Replace with your secondary color

# Neutral colors (dark variant)
gray-900    = "#111827"
gray-800    = "#1f2937"
gray-700    = "#374151"
gray-600    = "#4b5563"
gray-500    = "#6b7280"
gray-400    = "#9ca3af"
gray-300    = "#d1d5db"
gray-200    = "#e5e7eb"
gray-100    = "#f3f4f6"

# Status colors
green       = "#10b981"
yellow      = "#f59e0b"
red         = "#ef4444"
blue        = "#3b82f6"

# Layer 2: Semantic Mapping
[palette.semantic]
text-primary   = "gray-100"
text-secondary = "gray-300"
text-muted     = "gray-500"

bg-primary   = "gray-900"
bg-secondary = "gray-800"
bg-surface   = "gray-800"
bg-elevated  = "gray-700"

accent       = "primary"
accent-hover = "secondary"
link         = "primary"
link-hover   = "secondary"
link-visited = "secondary"

success = "green"
warning = "yellow"
error   = "red"
info    = "blue"

border       = "gray-700"
border-focus = "primary"

# Layer 3: Component Colors (optional)
[palette.components]
code-bg       = "gray-800"
code-text     = "gray-100"
code-comment  = "gray-500"
code-keyword  = "secondary"
code-string   = "green"
code-number   = "primary"
code-function = "primary"

# Add more component colors as needed
```

Minimal palette (`--minimal`):
```toml
# palettes/my-brand.toml
# Minimal palette - component colors inherit from semantic

[palette]
name = "My Brand"
variant = "dark"
author = "Your Name"

[palette.colors]
primary   = "#3b82f6"
secondary = "#8b5cf6"
bg        = "#111827"
surface   = "#1f2937"
text      = "#f3f4f6"
muted     = "#6b7280"
green     = "#10b981"
yellow    = "#f59e0b"
red       = "#ef4444"

[palette.semantic]
text-primary   = "text"
text-secondary = "text"
text-muted     = "muted"
bg-primary     = "bg"
bg-surface     = "surface"
accent         = "primary"
link           = "primary"
success        = "green"
warning        = "yellow"
error          = "red"
info           = "primary"
border         = "muted"
```

**Implementation notes:**
- Create `./palettes/` directory if it doesn't exist
- Use template with TODO comments for guidance
- `--from` copies and modifies metadata
- Validate name doesn't conflict with built-in palettes

---

### `palette export`

Export a palette to different formats.

**Usage:**
```bash
markata-go palette export <name> [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Output format: `css`, `scss`, `json`, `tailwind` |
| `--output` | `-o` | Output file (default: stdout) |

**Export as CSS:**
```bash
$ markata-go palette export catppuccin-mocha --format css

:root {
  /* Raw palette colors */
  --palette-rosewater: #f5e0dc;
  --palette-flamingo: #f2cdcd;
  --palette-pink: #f5c2e7;
  --palette-mauve: #cba6f7;
  /* ... */
  
  /* Semantic colors */
  --color-text-primary: var(--palette-text);
  --color-text-secondary: var(--palette-subtext1);
  --color-bg-primary: var(--palette-base);
  --color-accent: var(--palette-mauve);
  /* ... */
  
  /* Component colors */
  --code-bg: var(--palette-surface0);
  --code-text: var(--palette-text);
  /* ... */
}
```

**Export as Tailwind config:**
```bash
$ markata-go palette export catppuccin-mocha --format tailwind

module.exports = {
  theme: {
    extend: {
      colors: {
        'palette': {
          'rosewater': '#f5e0dc',
          'flamingo': '#f2cdcd',
          'pink': '#f5c2e7',
          'mauve': '#cba6f7',
          // ...
        },
        'text': {
          'primary': '#cdd6f4',
          'secondary': '#bac2de',
          'muted': '#7f849c',
        },
        'bg': {
          'primary': '#1e1e2e',
          'secondary': '#181825',
          'surface': '#313244',
        },
        // ...
      }
    }
  }
}
```

**Export as JSON:**
```bash
$ markata-go palette export catppuccin-mocha --format json
```
```json
{
  "name": "Catppuccin Mocha",
  "variant": "dark",
  "colors": {
    "rosewater": "#f5e0dc",
    "flamingo": "#f2cdcd",
    "pink": "#f5c2e7",
    "mauve": "#cba6f7",
    "red": "#f38ba8",
    "text": "#cdd6f4",
    "base": "#1e1e2e"
  },
  "semantic": {
    "text-primary": "#cdd6f4",
    "text-secondary": "#bac2de",
    "text-muted": "#7f849c",
    "bg-primary": "#1e1e2e",
    "bg-secondary": "#181825",
    "accent": "#cba6f7",
    "link": "#89b4fa",
    "link-hover": "#74c7ec",
    "success": "#a6e3a1",
    "warning": "#f9e2af",
    "error": "#f38ba8"
  },
  "components": {
    "code-bg": "#313244",
    "code-text": "#cdd6f4",
    "code-keyword": "#cba6f7",
    "button-primary-bg": "#cba6f7",
    "button-primary-text": "#1e1e2e"
  }
}
```

**Export as SCSS:**
```bash
$ markata-go palette export catppuccin-mocha --format scss -o _palette.scss

// _palette.scss - Generated from catppuccin-mocha

// Raw colors
$palette-rosewater: #f5e0dc;
$palette-flamingo: #f2cdcd;
// ...

// Semantic colors
$color-text-primary: $palette-text;
$color-bg-primary: $palette-base;
// ...

// As a map for programmatic access
$palette: (
  'rosewater': #f5e0dc,
  'flamingo': #f2cdcd,
  // ...
);
```

**Implementation notes:**
- Read palette TOML
- Transform to target format
- Resolve references for formats that don't support variables

---

## See Also

- [TEMPLATES.md](./TEMPLATES.md) - Template system details
- [CONFIG.md](./CONFIG.md) - Theme configuration
- [CONTENT.md](./CONTENT.md) - Admonition syntax
- [SPEC.md](./SPEC.md) - Core specification
