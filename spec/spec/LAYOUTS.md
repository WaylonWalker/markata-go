# Layout System Specification

The layout system provides configurable page structures for different content types. It enables documentation sites, blogs, landing pages, and custom layouts with zero-config defaults and progressive customization.

## Design Principles

1. **Zero-config beautiful** - `layout: docs` gives a professional 3-panel documentation layout immediately
2. **Progressive customization** - Enable components, configure them, override per-page
3. **Responsive by default** - All layouts adapt to mobile with collapsible panels
4. **Component-based** - Sidebar, TOC, header, footer are independent, composable components
5. **Theme integration** - Layouts use the existing palette and CSS variable system
6. **Template inheritance** - Layouts extend `base.html` using pongo2 blocks

---

## Layout Types

Markata-go provides four built-in layout presets. Each preset configures components appropriately for its use case.

### Layout Preset Overview

| Layout | Sidebar | TOC | Header | Footer | Best For |
|--------|---------|-----|--------|--------|----------|
| `docs` | Left (nav) | Right | Minimal | Minimal | Documentation sites |
| `blog` | None | Optional | Full | Full | Blog posts, articles |
| `landing` | None | None | Minimal | Full | Home pages, marketing |
| `bare` | None | None | None | None | Embeds, custom pages |

### `docs` Layout

The 3-panel documentation layout optimized for technical content.

```
+------------------------------------------------------------------+
|                         HEADER (minimal)                          |
|  [Logo]              [Nav Links]              [Search] [Theme]    |
+----------+------------------------------------------+-------------+
|          |                                          |             |
|  SIDEBAR |              CONTENT                     |     TOC     |
|  (nav)   |                                          |  (on-page)  |
|          |  # Page Title                            |             |
|  [Home]  |                                          |  - Section  |
|  [Guide] |  Content here...                         |  - Section  |
|  [API]   |                                          |    - Sub    |
|  [FAQ]   |                                          |  - Section  |
|          |                                          |             |
+----------+------------------------------------------+-------------+
|                         FOOTER (minimal)                          |
+------------------------------------------------------------------+
```

**Default configuration:**
```toml
[markata-go.layout]
name = "docs"

[markata-go.layout.docs]
sidebar_position = "left"
sidebar_width = "280px"
toc_position = "right"
toc_width = "220px"
content_max_width = "800px"
header_style = "minimal"
footer_style = "minimal"

# TOC Configuration (new in v0.x.x)
[markata-go.layout.docs.toc]
enabled = true                              # Enable TOC display
auto_enable = true                          # Auto-show based on thresholds
min_toc_links = 3                           # Min headings for auto-show
min_word_count = 500                        # Min words for auto-show
```

### `blog` Layout

Traditional blog layout with optional TOC for long-form content.

```
+------------------------------------------------------------------+
|                           HEADER                                  |
|  [Logo]    [Home] [Blog] [About] [Contact]    [Search] [Theme]   |
+------------------------------------------------------------------+
|                                                                   |
|                          CONTENT                                  |
|                                                                   |
|  # Article Title                                                  |
|  Posted on January 15, 2024 by Author                            |
|  Tags: [go] [static-site]                                        |
|                                                                   |
|  Article content here with full width...                         |
|                                                                   |
|  [Previous Post]                      [Next Post]                |
|                                                                   |
+------------------------------------------------------------------+
|                           FOOTER                                  |
|  Copyright 2024  |  RSS  |  Twitter  |  GitHub                   |
+------------------------------------------------------------------+
```

**Default configuration:**
```toml
[markata-go.layout]
name = "blog"

[markata-go.layout.blog]
content_max_width = "720px"
show_toc = false          # Enable per-post in frontmatter
toc_position = "right"
header_style = "full"
footer_style = "full"
show_author = true
show_date = true
show_tags = true
show_reading_time = true
show_prev_next = true
```

### `landing` Layout

Full-width layout for marketing pages and home pages.

```
+------------------------------------------------------------------+
|  HEADER (minimal, transparent)        [Nav] [CTA Button]         |
+------------------------------------------------------------------+
|                                                                   |
|                     HERO SECTION                                  |
|                                                                   |
|              Your Amazing Product                                 |
|         Build beautiful sites with ease                          |
|                                                                   |
|              [Get Started]  [Learn More]                         |
|                                                                   |
+------------------------------------------------------------------+
|                                                                   |
|                   CONTENT (full width)                            |
|                                                                   |
|  Features, testimonials, pricing, etc.                           |
|                                                                   |
+------------------------------------------------------------------+
|                           FOOTER                                  |
+------------------------------------------------------------------+
```

**Default configuration:**
```toml
[markata-go.layout]
name = "landing"

[markata-go.layout.landing]
content_max_width = "100%"
header_style = "transparent"
footer_style = "full"
hero_enabled = true
```

### `bare` Layout

Minimal layout with no chrome - just the content.

```
+------------------------------------------------------------------+
|                                                                   |
|                         CONTENT                                   |
|                                                                   |
|  Pure content, no header, no footer, no navigation               |
|                                                                   |
+------------------------------------------------------------------+
```

**Default configuration:**
```toml
[markata-go.layout]
name = "bare"

[markata-go.layout.bare]
content_max_width = "100%"
```

---

## Configuration

### Global Layout Configuration

Set site-wide layout defaults in `markata-go.toml`:

```toml
[markata-go.layout]
# Default layout for all pages
name = "docs"

# Global component settings (apply to all layouts unless overridden)
[markata-go.layout.defaults]
content_max_width = "800px"
header_sticky = true
footer_sticky = false
```

### Layout-Specific Configuration

Configure each layout preset:

```toml
# Documentation layout
[markata-go.layout.docs]
sidebar_position = "left"     # "left" | "right"
sidebar_width = "280px"
sidebar_collapsible = true
sidebar_default_open = true   # Desktop default
toc_position = "right"        # "left" | "right" (opposite of sidebar)
toc_width = "220px"
toc_collapsible = true
toc_default_open = true
content_max_width = "800px"
header_style = "minimal"      # "full" | "minimal" | "transparent" | "none"
footer_style = "minimal"      # "full" | "minimal" | "none"

# Blog layout
[markata-go.layout.blog]
content_max_width = "720px"
show_toc = false
toc_position = "right"
toc_width = "200px"
header_style = "full"
footer_style = "full"
show_author = true
show_date = true
show_tags = true
show_reading_time = true
show_prev_next = true

# Landing page layout
[markata-go.layout.landing]
content_max_width = "100%"
header_style = "transparent"
header_sticky = true
footer_style = "full"

# Bare layout
[markata-go.layout.bare]
content_max_width = "100%"
```

### Component Configuration

Individual components can be configured globally:

```toml
# Sidebar component
[markata-go.components.sidebar]
enabled = true
position = "left"
width = "280px"
collapsible = true
default_open = true
# Navigation structure (auto-generated from feeds if not specified)
# Or manually defined:
[[markata-go.components.sidebar.nav]]
title = "Getting Started"
href = "/docs/getting-started/"

[[markata-go.components.sidebar.nav]]
title = "Guides"
children = [
    { title = "Configuration", href = "/docs/guides/configuration/" },
    { title = "Themes", href = "/docs/guides/themes/" },
    { title = "Layouts", href = "/docs/guides/layouts/" },
]

[[markata-go.components.sidebar.nav]]
title = "Reference"
children = [
    { title = "CLI", href = "/docs/reference/cli/" },
    { title = "Plugins", href = "/docs/reference/plugins/" },
]

# Table of Contents component
[markata-go.components.toc]
enabled = true
position = "right"
width = "220px"
min_depth = 2          # Start at h2
max_depth = 4          # End at h4
title = "On this page"
collapsible = true
default_open = true
scroll_spy = true      # Highlight current section

# Header component
[markata-go.components.header]
style = "full"         # "full" | "minimal" | "transparent" | "none"
sticky = true
show_logo = true
show_title = true
show_nav = true
show_search = true
show_theme_toggle = true

# Footer component
[markata-go.components.footer]
style = "full"         # "full" | "minimal" | "none"
sticky = false
show_copyright = true
copyright_text = "Copyright 2024 My Company"
show_social_links = true
show_nav_links = true
```

---

## Frontmatter Overrides

Any layout setting can be overridden per-page using frontmatter.

### Basic Override

```yaml
---
title: "API Reference"
layout: docs
---
```

### Component Overrides

```yaml
---
title: "Quick Start Guide"
layout: docs

# Override specific components for this page
layout_config:
  sidebar:
    enabled: true
    position: left
  toc:
    enabled: true
    position: right
    min_depth: 2
    max_depth: 3
---
```

### Disable Components

```yaml
---
title: "Full-Width Demo"
layout: docs

# Disable sidebar and TOC for this page
layout_config:
  sidebar:
    enabled: false
  toc:
    enabled: false
  content_max_width: "100%"
---
```

### Switch Layout

```yaml
---
title: "Welcome to Our Docs"
layout: landing   # Use landing layout instead of default

layout_config:
  hero_enabled: true
  header_style: transparent
---
```

### Feature Flags (Shorthand)

Common overrides have shorthand frontmatter keys:

```yaml
---
title: "My Post"
toc: false              # Shorthand for layout_config.toc.enabled = false
sidebar: false          # Shorthand for layout_config.sidebar.enabled = false
full_width: true        # Shorthand for layout_config.content_max_width = "100%"
---
```

---

## Components

### Sidebar Component

The sidebar provides navigation for documentation and multi-page content.

#### Structure

```html
<aside class="layout-sidebar" data-position="left" data-collapsible="true">
  <button class="sidebar-toggle" aria-label="Toggle sidebar">
    <span class="icon-menu"></span>
  </button>
  <nav class="sidebar-nav" aria-label="Documentation">
    <ul class="sidebar-nav-list">
      <li class="sidebar-nav-item">
        <a href="/docs/" class="sidebar-nav-link">Home</a>
      </li>
      <li class="sidebar-nav-item sidebar-nav-item--has-children">
        <button class="sidebar-nav-toggle" aria-expanded="true">
          Guides
          <span class="icon-chevron"></span>
        </button>
        <ul class="sidebar-nav-children">
          <li><a href="/docs/guides/config/">Configuration</a></li>
          <li><a href="/docs/guides/themes/">Themes</a></li>
        </ul>
      </li>
    </ul>
  </nav>
</aside>
```

#### Auto-Generation

If no manual navigation is specified, the sidebar is auto-generated from:

1. **Feed structure** - Posts in the same feed become nav items
2. **Directory structure** - Nested directories become nested nav items
3. **Frontmatter order** - Use `nav_order: 1` to control ordering
4. **Frontmatter groups** - Use `nav_group: "Guides"` to group items

```yaml
---
title: "Configuration Guide"
nav_order: 2
nav_group: "Guides"
---
```

### TOC Component

The table of contents component shows an outline of the current page.

#### Structure

```html
<aside class="layout-toc" data-position="right">
  <nav class="toc" aria-label="Table of Contents">
    <h2 class="toc-title">On this page</h2>
    <ul class="toc-list">
      <li class="toc-item toc-item--level-2">
        <a href="#overview" class="toc-link">Overview</a>
      </li>
      <li class="toc-item toc-item--level-2">
        <a href="#installation" class="toc-link toc-link--active">Installation</a>
        <ul class="toc-list toc-list--nested">
          <li class="toc-item toc-item--level-3">
            <a href="#prerequisites" class="toc-link">Prerequisites</a>
          </li>
        </ul>
      </li>
    </ul>
  </nav>
</aside>
```

#### Scroll Spy

When `scroll_spy: true`, the TOC highlights the currently visible section:

```javascript
// Intersection Observer watches heading visibility
// Updates .toc-link--active class on scroll
```

### Header Component

#### Styles

**Full Header:**
```html
<header class="layout-header layout-header--full">
  <div class="header-container">
    <a href="/" class="header-logo">
      <img src="/logo.svg" alt="Site Logo">
      <span class="header-title">Site Title</span>
    </a>
    <nav class="header-nav" aria-label="Main navigation">
      <a href="/docs/">Docs</a>
      <a href="/blog/">Blog</a>
      <a href="/about/">About</a>
    </nav>
    <div class="header-actions">
      <button class="search-toggle" aria-label="Search">
        <span class="icon-search"></span>
      </button>
      <button class="theme-toggle" aria-label="Toggle theme">
        <span class="icon-sun"></span>
        <span class="icon-moon"></span>
      </button>
    </div>
  </div>
</header>
```

**Minimal Header:**
```html
<header class="layout-header layout-header--minimal">
  <div class="header-container">
    <a href="/" class="header-logo">Site</a>
    <div class="header-actions">
      <button class="search-toggle"></button>
      <button class="theme-toggle"></button>
    </div>
  </div>
</header>
```

**Transparent Header:**
```html
<header class="layout-header layout-header--transparent">
  <!-- Same as full, but with transparent background -->
</header>
```

### Footer Component

#### Styles

**Full Footer:**
```html
<footer class="layout-footer layout-footer--full">
  <div class="footer-container">
    <div class="footer-nav">
      <div class="footer-nav-group">
        <h3>Product</h3>
        <a href="/features/">Features</a>
        <a href="/pricing/">Pricing</a>
      </div>
      <div class="footer-nav-group">
        <h3>Resources</h3>
        <a href="/docs/">Documentation</a>
        <a href="/blog/">Blog</a>
      </div>
    </div>
    <div class="footer-social">
      <a href="https://twitter.com/..." aria-label="Twitter">
        <span class="icon-twitter"></span>
      </a>
      <a href="https://github.com/..." aria-label="GitHub">
        <span class="icon-github"></span>
      </a>
    </div>
    <div class="footer-legal">
      <p>&copy; 2024 My Company. All rights reserved.</p>
      <a href="/privacy/">Privacy</a>
      <a href="/terms/">Terms</a>
    </div>
  </div>
</footer>
```

**Minimal Footer:**
```html
<footer class="layout-footer layout-footer--minimal">
  <div class="footer-container">
    <p>&copy; 2024 My Company</p>
  </div>
</footer>
```

### Share Component

The share component provides a configurable "Share this post" experience at the end of article templates. It renders a compact row of icon buttons for each platform and keeps copy-to-clipboard functionality accessible via keyboard and screen readers.

**Placement**: Injected into `post.html` (and theme equivalents) between the article body and ancillary sections (guide navigation, webmentions) so it appears at the end of every article before comments/footers.

**Behavior**:

1. Displays icon buttons in a compact inline row that wraps as needed on smaller viewports.
2. Hover or focus reveals the platform name via tooltip text and `aria-label` attributes.
3. Clicks open the platform share dialog in a new tab (or copy the link when `copy` is enabled).
4. Copy button uses the Clipboard API with a DOM fallback and provides live feedback.
5. Renders a reply row below sharing when the primary author has contact metadata (email and/or supported social handles).
6. Uses CSS variables for colors, spacing, and border radius so palettes can theme the component.

#### Configuration

```toml
[markata-go.components.share]
enabled = true
position = "bottom"            # Controls the style hook (CSS adds `share-panel--bottom`).
title = "Share this post"     # Label shown before the icon row.
platforms = ["twitter", "bluesky", "linkedin", "whatsapp", "facebook", "telegram", "pinterest", "reddit", "hacker_news", "email", "copy"]

[markata-go.components.share.custom]
mastodon = { name = "Mastodon", icon = "mastodon.svg", url = "https://mastodon.social/share?text={{title}}&url={{url}}" }
```

| Option | Type | Description |
| --- | --- | --- |
| `enabled` | `bool` | Flip the entire component. Defaults to `true`. |
| `position` | `string` | Adds a modifier class `share-panel--<position>` (default `bottom`). |
| `title` | `string` | Label shown before the icon row. |
| `platforms` | `[]string` | Ordered list of platform keys to render. Missing list falls back to the built-in order. |
| `custom` | `table` | Keyed definitions (`name`, `icon`, `url`). Icon paths are resolved via `theme_asset_hashed` when they do not start with `/`, `http`, or `data:`. SVGs from open icon sets are supported. |

Valid placeholders in share URLs:

| Placeholder | Description |
| --- | --- |
| `{{title}}` | URL-encoded post title (falls back to site title). |
| `{{url}}` | URL-encoded absolute post URL (`config.url` + `post.href`). |
| `{{excerpt}}` | URL-encoded post description/excerpt when provided. |

The component ships with these built-in platforms:

| Key | Template | Notes |
| --- | --- | --- |
| `twitter` | `https://twitter.com/intent/tweet?text={{title}}&url={{url}}` | Icon: `icons/share/twitter.svg`. |
| `bluesky` | `https://bsky.app/intent/compose?text={{url}}` | Icon: `icons/share/bluesky.svg`. |
| `facebook` | `https://www.facebook.com/sharer/sharer.php?u={{url}}` | Icon: `icons/share/facebook.svg`. |
| `linkedin` | `https://www.linkedin.com/sharing/share-offsite/?url={{url}}` | Icon: `icons/share/linkedin.svg`. |
| `whatsapp` | `https://wa.me/?text={{url}}` | Icon: `icons/share/whatsapp.svg`. |
| `telegram` | `https://t.me/share/url?url={{url}}` | Icon: `icons/share/telegram.svg`. |
| `pinterest` | `https://pinterest.com/pin/create/button/?url={{url}}` | Icon: `icons/share/pinterest.svg`. |
| `reddit` | `https://reddit.com/submit?url={{url}}&title={{title}}` | Icon: `icons/share/reddit.svg`. |
| `hacker_news` | `https://news.ycombinator.com/submitlink?u={{url}}&t={{title}}` | Icon: `icons/share/hacker_news.svg`. |
| `email` | `mailto:?subject={{title}}&body={{url}}` | Icon: `icons/share/email.svg`. |
| `copy` | Copy clipboard | Icon: `icons/share/copy.svg`. |

Custom entries can reuse the built-in keys (e.g., override the icon for `copy`) or introduce new platforms (`mastodon`, `bluesky`, etc.). If the `platforms` list omits a built-in key, that button is hidden.

#### Accessibility & Interaction

- Buttons are focusable, provide `aria-label`s like "Share on Twitter" (copy button reads "Copy link to clipboard"), and update their label to "Link copied" after copying.
- The component loads a tiny Clipboard helper script only once.
- Responsive states keep buttons compact across desktop and mobile so sharing does not dominate the article layout.
- Reply links are keyboard-focusable and only render when author metadata exists, avoiding empty call-to-action areas.

#### Reply Row

When the first entry in `post.author_objects` contains contact data, a compact reply row is rendered directly below share buttons:

- `email` -> `Reply by email` mailto link (`subject` prefilled with `Re:<post href>` and body with the post URL).
- `social.twitter` -> `X` profile link (`https://x.com/<handle>`).
- `social.bluesky` -> `Bluesky` profile link (`https://bsky.app/profile/<handle>`).
- `social.linkedin` -> `LinkedIn` profile link (`https://linkedin.com/in/<handle>`).
- `social.github` -> `GitHub` profile link (`https://github.com/<handle>`).
- `social.mastodon` -> `Mastodon` link (uses value directly if it starts with `http`, otherwise `https://mastodon.social/@<handle>`).


---

## CSS Structure

### CSS Variables

Layout system uses CSS custom properties for easy theming:

```css
:root {
  /* Layout dimensions */
  --layout-sidebar-width: 280px;
  --layout-toc-width: 220px;
  --layout-content-max-width: 800px;
  --layout-header-height: 64px;
  --layout-footer-min-height: 200px;

  /* Responsive breakpoints */
  --breakpoint-sm: 640px;
  --breakpoint-md: 768px;
  --breakpoint-lg: 1024px;
  --breakpoint-xl: 1280px;

  /* Spacing */
  --layout-gutter: var(--space-6);
  --layout-padding: var(--space-4);

  /* Z-index layers */
  --z-sidebar: 100;
  --z-header: 200;
  --z-overlay: 300;
  --z-modal: 400;

  /* Transitions */
  --transition-sidebar: transform 0.3s ease-in-out;
  --transition-toc: opacity 0.2s ease;
}
```

### Layout Classes

```css
/* Base layout container */
.layout {
  display: grid;
  min-height: 100vh;
  grid-template-rows: auto 1fr auto;
  grid-template-columns: 1fr;
}

/* Docs layout: 3-column */
.layout--docs {
  grid-template-columns: var(--layout-sidebar-width) 1fr var(--layout-toc-width);
  grid-template-areas:
    "header  header  header"
    "sidebar content toc"
    "footer  footer  footer";
}

/* Blog layout: single column */
.layout--blog {
  grid-template-columns: 1fr;
  grid-template-areas:
    "header"
    "content"
    "footer";
}

/* Landing layout: full width */
.layout--landing {
  grid-template-columns: 1fr;
  grid-template-areas:
    "header"
    "content"
    "footer";
}

/* Bare layout: content only */
.layout--bare {
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
  grid-template-areas: "content";
}
```

### Component Classes

```css
/* Header */
.layout-header {
  grid-area: header;
  position: sticky;
  top: 0;
  z-index: var(--z-header);
  background: var(--color-bg-primary);
  border-bottom: 1px solid var(--color-border);
}

.layout-header--sticky {
  position: sticky;
}

.layout-header--transparent {
  background: transparent;
  border-bottom: none;
  position: absolute;
  width: 100%;
}

/* Sidebar */
.layout-sidebar {
  grid-area: sidebar;
  position: sticky;
  top: var(--layout-header-height);
  height: calc(100vh - var(--layout-header-height));
  overflow-y: auto;
  padding: var(--layout-padding);
  border-right: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.layout-sidebar[data-position="right"] {
  grid-area: toc;
  border-right: none;
  border-left: 1px solid var(--color-border);
}

/* Content */
.layout-content {
  grid-area: content;
  max-width: var(--layout-content-max-width);
  margin: 0 auto;
  padding: var(--layout-padding) var(--layout-gutter);
  width: 100%;
}

/* TOC */
.layout-toc {
  grid-area: toc;
  position: sticky;
  top: var(--layout-header-height);
  height: calc(100vh - var(--layout-header-height));
  overflow-y: auto;
  padding: var(--layout-padding);
  border-left: 1px solid var(--color-border);
}

/* Footer */
.layout-footer {
  grid-area: footer;
  background: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
}
```

---

## Responsive Behavior

### Breakpoints

| Breakpoint | Width | Behavior |
|------------|-------|----------|
| `xl` | >= 1280px | Full 3-panel layout |
| `lg` | >= 1024px | TOC hidden, sidebar visible |
| `md` | >= 768px | Sidebar as overlay |
| `sm` | < 768px | All panels as overlays |

### Mobile Behavior

On mobile devices (< 768px):

1. **Sidebar** - Collapses to hamburger menu
2. **TOC** - Collapses to floating button or hides
3. **Header** - Remains sticky with hamburger toggle
4. **Content** - Expands to full width

```css
/* Responsive layout */
@media (max-width: 1279px) {
  .layout--docs {
    grid-template-columns: var(--layout-sidebar-width) 1fr;
    grid-template-areas:
      "header  header"
      "sidebar content"
      "footer  footer";
  }

  .layout-toc {
    display: none;
  }
}

@media (max-width: 1023px) {
  .layout--docs {
    grid-template-columns: 1fr;
    grid-template-areas:
      "header"
      "content"
      "footer";
  }

  .layout-sidebar {
    position: fixed;
    left: 0;
    top: var(--layout-header-height);
    transform: translateX(-100%);
    transition: var(--transition-sidebar);
    z-index: var(--z-sidebar);
    height: calc(100vh - var(--layout-header-height));
    width: min(var(--layout-sidebar-width), 85vw);
  }

  .layout-sidebar[data-open="true"] {
    transform: translateX(0);
  }
}

@media (max-width: 767px) {
  :root {
    --layout-header-height: 56px;
    --layout-padding: var(--space-3);
    --layout-gutter: var(--space-3);
  }
}
```

### Mobile Navigation

The hamburger menu button toggles the sidebar:

```html
<button class="mobile-menu-toggle"
        aria-label="Toggle navigation"
        aria-expanded="false"
        aria-controls="sidebar">
  <span class="hamburger-line"></span>
  <span class="hamburger-line"></span>
  <span class="hamburger-line"></span>
</button>
```

JavaScript handles the toggle:

```javascript
// Sidebar toggle behavior
const toggle = document.querySelector('.mobile-menu-toggle');
const sidebar = document.querySelector('.layout-sidebar');
const overlay = document.querySelector('.sidebar-overlay');

toggle?.addEventListener('click', () => {
  const isOpen = sidebar.dataset.open === 'true';
  sidebar.dataset.open = !isOpen;
  toggle.setAttribute('aria-expanded', !isOpen);
  document.body.classList.toggle('sidebar-open', !isOpen);
});

// Close on overlay click
overlay?.addEventListener('click', () => {
  sidebar.dataset.open = 'false';
  toggle.setAttribute('aria-expanded', 'false');
  document.body.classList.remove('sidebar-open');
});

// Close on escape key
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && sidebar.dataset.open === 'true') {
    sidebar.dataset.open = 'false';
    toggle.setAttribute('aria-expanded', 'false');
    document.body.classList.remove('sidebar-open');
  }
});
```

### Overlay

When sidebar is open on mobile, an overlay covers the content:

```css
.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  top: var(--layout-header-height);
  background: rgba(0, 0, 0, 0.5);
  z-index: calc(var(--z-sidebar) - 1);
}

.sidebar-open .sidebar-overlay {
  display: block;
}
```

---

## Template Integration

### Base Layout Template

The layout system extends `base.html` with layout-specific structure:

```jinja2
{# templates/layouts/docs.html #}
{% extends "base.html" %}

{% block body_class %}layout layout--docs{% endblock %}

{% block header %}
{% include "components/header.html" with style=layout.header_style %}
{% endblock %}

{% block main %}
<div class="layout-container">
  {% if layout.sidebar.enabled %}
  {% include "components/sidebar.html" %}
  {% endif %}

  <main class="layout-content" id="main-content">
    {% block content %}{% endblock %}
  </main>

  {% if layout.toc.enabled and post.Extra.toc %}
  {% include "components/toc.html" %}
  {% endif %}
</div>
{% endblock %}

{% block footer %}
{% include "components/footer.html" with style=layout.footer_style %}
{% endblock %}

{% block scripts %}
{{ super() }}
<script src="{{ 'js/layout.js' | theme_asset }}" defer></script>
{% endblock %}
```

### Using Layouts in Post Templates

```jinja2
{# templates/post.html #}
{% extends layout_template %}

{% block content %}
<article class="post">
  <header class="post-header">
    <h1>{{ post.title }}</h1>
    {% if layout.show_date and post.date %}
    <time datetime="{{ post.date | atom_date }}">{{ post.date | date:"January 2, 2006" }}</time>
    {% endif %}
    {% if layout.show_tags and post.tags %}
    <div class="tags">
      {% for tag in post.tags %}
      <a href="/tags/{{ tag | slugify }}/">{{ tag }}</a>
      {% endfor %}
    </div>
    {% endif %}
  </header>

  <div class="post-content">
    {{ body | safe }}
  </div>

  {% if layout.show_prev_next %}
  <nav class="post-nav">
    {% if post.prev %}<a href="{{ post.prev.href }}" class="prev">&larr; {{ post.prev.title }}</a>{% endif %}
    {% if post.next %}<a href="{{ post.next.href }}" class="next">{{ post.next.title }} &rarr;</a>{% endif %}
  </nav>
  {% endif %}
</article>
{% endblock %}
```

### Layout Resolution

The template system resolves layouts in this order:

1. **Frontmatter `layout`** - Post specifies layout name
2. **Feed default** - Feed configuration specifies default layout
3. **Global default** - `[markata-go.layout].name` setting
4. **Fallback** - `blog` layout

```python
def get_layout_template(post, feed, config):
    layout_name = (
        post.frontmatter.get('layout') or
        feed.layout or
        config.layout.name or
        'blog'
    )
    return f"layouts/{layout_name}.html"
```

---

## Examples

### Complete Documentation Site

```toml
# markata-go.toml

[markata-go.layout]
name = "docs"

[markata-go.layout.docs]
sidebar_position = "left"
sidebar_width = "280px"
toc_position = "right"
toc_width = "220px"
content_max_width = "800px"
header_style = "minimal"
footer_style = "minimal"

[markata-go.components.sidebar]
enabled = true
collapsible = true

[[markata-go.components.sidebar.nav]]
title = "Getting Started"
href = "/docs/"

[[markata-go.components.sidebar.nav]]
title = "Guides"
children = [
    { title = "Configuration", href = "/docs/guides/configuration/" },
    { title = "Themes", href = "/docs/guides/themes/" },
    { title = "Layouts", href = "/docs/guides/layouts/" },
    { title = "Plugins", href = "/docs/guides/plugins/" },
]

[[markata-go.components.sidebar.nav]]
title = "Reference"
children = [
    { title = "CLI Commands", href = "/docs/reference/cli/" },
    { title = "Plugin API", href = "/docs/reference/plugins/" },
    { title = "Config Options", href = "/docs/reference/config/" },
]

[markata-go.components.toc]
enabled = true
min_depth = 2
max_depth = 4
scroll_spy = true

[markata-go.components.header]
style = "minimal"
sticky = true
show_search = true
show_theme_toggle = true
```

### Blog with Optional TOC

```toml
# markata-go.toml

[markata-go.layout]
name = "blog"

[markata-go.layout.blog]
content_max_width = "720px"
show_toc = false       # Disabled by default
toc_position = "right"
show_author = true
show_date = true
show_tags = true
show_reading_time = true
show_prev_next = true

[markata-go.components.header]
style = "full"
sticky = true

[markata-go.components.footer]
style = "full"
show_copyright = true
show_social_links = true
```

Individual posts can enable TOC:

```yaml
---
title: "Complete Guide to Go Interfaces"
date: 2024-01-15
toc: true    # Enable TOC for this long post
---
```

### Landing Page

```toml
# markata-go.toml for landing page sections

[markata-go.layout]
name = "landing"

[markata-go.layout.landing]
content_max_width = "100%"
header_style = "transparent"
hero_enabled = true
```

```yaml
---
title: "Welcome to Markata-Go"
layout: landing
hero:
  title: "Beautiful static sites in Go"
  subtitle: "Zero-config, blazing fast, infinitely customizable"
  cta_primary:
    text: "Get Started"
    href: "/docs/getting-started/"
  cta_secondary:
    text: "View on GitHub"
    href: "https://github.com/..."
---
```

### Mixed Layouts in One Site

```yaml
# docs/index.md - uses docs layout
---
title: "Documentation"
layout: docs
---

# docs/api/index.md - uses docs layout (inherited)
---
title: "API Reference"
---

# blog/my-post.md - uses blog layout
---
title: "Announcing v2.0"
layout: blog
---

# index.md - uses landing layout
---
title: "Home"
layout: landing
---
```

---

## Accessibility

### ARIA Landmarks

```html
<header role="banner">...</header>
<nav role="navigation" aria-label="Main">...</nav>
<main role="main" id="main-content">...</main>
<aside role="complementary" aria-label="Table of Contents">...</aside>
<footer role="contentinfo">...</footer>
```

### Skip Links

```html
<a href="#main-content" class="skip-link">Skip to main content</a>
<a href="#sidebar" class="skip-link">Skip to navigation</a>
```

### Keyboard Navigation

- **Tab** - Navigate between interactive elements
- **Escape** - Close open menus/overlays
- **Arrow keys** - Navigate within menus
- **Enter/Space** - Activate buttons and links

### Focus Management

```css
/* Visible focus indicators */
:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Skip link */
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  padding: var(--space-2) var(--space-4);
  background: var(--color-bg-primary);
  z-index: 9999;
}

.skip-link:focus {
  top: 0;
}
```

---

## Configuration Reference

### Layout Config Model

```go
type LayoutConfig struct {
    // Layout preset name
    Name string `toml:"name" default:"blog"`

    // Layout-specific settings
    Docs    DocsLayoutConfig    `toml:"docs"`
    Blog    BlogLayoutConfig    `toml:"blog"`
    Landing LandingLayoutConfig `toml:"landing"`
    Bare    BareLayoutConfig    `toml:"bare"`

    // Global defaults
    Defaults LayoutDefaults `toml:"defaults"`
}

type LayoutDefaults struct {
    ContentMaxWidth string `toml:"content_max_width" default:"800px"`
    HeaderSticky    bool   `toml:"header_sticky" default:"true"`
    FooterSticky    bool   `toml:"footer_sticky" default:"false"`
}

type DocsLayoutConfig struct {
    SidebarPosition    string `toml:"sidebar_position" default:"left"`
    SidebarWidth       string `toml:"sidebar_width" default:"280px"`
    SidebarCollapsible bool   `toml:"sidebar_collapsible" default:"true"`
    SidebarDefaultOpen bool   `toml:"sidebar_default_open" default:"true"`
    TocPosition        string `toml:"toc_position" default:"right"`
    TocWidth           string `toml:"toc_width" default:"220px"`
    TocCollapsible     bool   `toml:"toc_collapsible" default:"true"`
    TocDefaultOpen     bool   `toml:"toc_default_open" default:"true"`
    ContentMaxWidth    string `toml:"content_max_width" default:"800px"`
    HeaderStyle        string `toml:"header_style" default:"minimal"`
    FooterStyle        string `toml:"footer_style" default:"minimal"`
}

type BlogLayoutConfig struct {
    ContentMaxWidth string `toml:"content_max_width" default:"720px"`
    ShowToc         bool   `toml:"show_toc" default:"false"`
    TocPosition     string `toml:"toc_position" default:"right"`
    TocWidth        string `toml:"toc_width" default:"200px"`
    HeaderStyle     string `toml:"header_style" default:"full"`
    FooterStyle     string `toml:"footer_style" default:"full"`
    ShowAuthor      bool   `toml:"show_author" default:"true"`
    ShowDate        bool   `toml:"show_date" default:"true"`
    ShowTags        bool   `toml:"show_tags" default:"true"`
    ShowReadingTime bool   `toml:"show_reading_time" default:"true"`
    ShowPrevNext    bool   `toml:"show_prev_next" default:"true"`
}

type LandingLayoutConfig struct {
    ContentMaxWidth string `toml:"content_max_width" default:"100%"`
    HeaderStyle     string `toml:"header_style" default:"transparent"`
    HeaderSticky    bool   `toml:"header_sticky" default:"true"`
    FooterStyle     string `toml:"footer_style" default:"full"`
    HeroEnabled     bool   `toml:"hero_enabled" default:"true"`
}

type BareLayoutConfig struct {
    ContentMaxWidth string `toml:"content_max_width" default:"100%"`
}
```

### Component Config Models

```go
type SidebarConfig struct {
    Enabled     bool           `toml:"enabled" default:"true"`
    Position    string         `toml:"position" default:"left"`
    Width       string         `toml:"width" default:"280px"`
    Collapsible bool           `toml:"collapsible" default:"true"`
    DefaultOpen bool           `toml:"default_open" default:"true"`
    Nav         []NavItem      `toml:"nav"`
}

type NavItem struct {
    Title    string    `toml:"title"`
    Href     string    `toml:"href"`
    Children []NavItem `toml:"children"`
}

type TocConfig struct {
    Enabled     bool   `toml:"enabled" default:"true"`
    Position    string `toml:"position" default:"right"`
    Width       string `toml:"width" default:"220px"`
    MinDepth    int    `toml:"min_depth" default:"2"`
    MaxDepth    int    `toml:"max_depth" default:"4"`
    Title       string `toml:"title" default:"On this page"`
    Collapsible bool   `toml:"collapsible" default:"true"`
    DefaultOpen bool   `toml:"default_open" default:"true"`
    ScrollSpy   bool   `toml:"scroll_spy" default:"true"`
}

type HeaderConfig struct {
    Style           string `toml:"style" default:"full"`
    Sticky          bool   `toml:"sticky" default:"true"`
    ShowLogo        bool   `toml:"show_logo" default:"true"`
    ShowTitle       bool   `toml:"show_title" default:"true"`
    ShowNav         bool   `toml:"show_nav" default:"true"`
    ShowSearch      bool   `toml:"show_search" default:"true"`
    ShowThemeToggle bool   `toml:"show_theme_toggle" default:"true"`
}

type FooterConfig struct {
    Style           string `toml:"style" default:"full"`
    Sticky          bool   `toml:"sticky" default:"false"`
    ShowCopyright   bool   `toml:"show_copyright" default:"true"`
    CopyrightText   string `toml:"copyright_text"`
    ShowSocialLinks bool   `toml:"show_social_links" default:"true"`
    ShowNavLinks    bool   `toml:"show_nav_links" default:"true"`
}
```

---

## See Also

- [THEMES.md](./THEMES.md) - Color palettes and theme customization
- [TEMPLATES.md](./TEMPLATES.md) - Template system and blocks
- [CONFIG.md](./CONFIG.md) - Configuration system
- [HEAD_STYLE.md](./HEAD_STYLE.md) - Head/style injection
