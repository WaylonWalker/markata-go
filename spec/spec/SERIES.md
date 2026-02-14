# Series Specification

Series are **ordered collections of posts meant to be read sequentially**. They build on the existing feed system to provide a first-class guided reading experience, similar to multi-part tutorials, article series, or course chapters.

## Overview

```
                         SERIES FLOW

  Frontmatter                 Auto-Generated Feed              Output
  ┌─────────────┐            ┌──────────────────┐           ┌──────────────────┐
  │ series:      │            │ slug: series/X   │           │ /series/X/       │
  │   "rest-api" │───scan───▶│ type: series     │──render──▶│   index.html     │
  │ series_order:│            │ sort: order/date │           │   rss.xml        │
  │   1          │            │ sidebar: true    │           │   atom.xml       │
  └─────────────┘            └──────────────────┘           │   feed.json      │
                                                             └──────────────────┘
```

## Why Series

1. **Guided reading** - Posts are meant to be read in order, unlike feeds which are browsed
2. **Navigation** - Automatic prev/next within the series, sidebar TOC, position indicator
3. **Discoverability** - Series index pages, RSS/Atom for subscribing to a series
4. **Simplicity** - One frontmatter field (`series`) is all that's needed

---

## Frontmatter

### Required Field

```yaml
---
title: "Building a REST API - Part 1"
series: "building-a-rest-api"
---
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `series` | string | Yes | Series identifier (becomes slug prefix: `series/<value>`) |

A post can belong to **at most one series**. The value is a string, not an array.

### Optional Field

```yaml
---
title: "Building a REST API - Part 1"
series: "building-a-rest-api"
series_order: 1
---
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `series_order` | int | none | Explicit position in series (1-indexed) |

### Ordering Rules

1. If **any** post in the series has `series_order`, sort by `series_order` ascending
2. Posts without `series_order` in a mixed series are placed after ordered posts, sorted by date ascending
3. If **no** post has `series_order`, sort by `date` ascending (oldest first, for sequential reading)
4. Ties broken by file path for deterministic ordering

**Rationale:** Series are read from beginning to end, so ascending order (oldest/lowest first) is the natural default, unlike blog feeds which show newest first.

---

## Auto-Generated Series Feeds

### Series Plugin Behavior

The `series` plugin scans all posts during the **Collect** stage (before feeds). For each unique `series` value found:

1. Creates a `FeedConfig` with:
   - `Slug`: `"series/<series-value>"` (e.g., `series/building-a-rest-api`)
   - `Title`: Derived from the series value (title-cased, hyphens to spaces) or from config override
   - `Type`: `FeedTypeSeries`
   - `Sidebar`: `true`
   - All format defaults from `[markata-go.feeds.defaults]`

2. Collects posts matching that series value

3. Sorts them according to ordering rules above

4. Sets guide navigation (Prev/Next) on each post in the series

5. Injects the feed config into the feed config list (before the feeds plugin runs)

### Configuration Override

Series feeds inherit from `[markata-go.feeds.defaults]`. Individual series can be customized:

```toml
# Override a specific series
[markata-go.series.overrides."building-a-rest-api"]
title = "Building a REST API with Go"
description = "A complete guide to building REST APIs"
items_per_page = 0          # No pagination (show all parts)

[markata-go.series.overrides."building-a-rest-api".formats]
markdown = true             # Also generate markdown output

# Global series defaults (override feed defaults for all series)
[markata-go.series.defaults]
items_per_page = 0          # Series typically show all posts
sidebar = true              # Always show sidebar
```

### Series-Specific Defaults

These defaults apply to all series feeds unless overridden:

| Setting | Series Default | Feed Default |
|---------|---------------|--------------|
| `items_per_page` | 0 (no pagination) | 10 |
| `sidebar` | true | false |
| `type` | `"series"` | `"blog"` |
| `sort` | `"series_order"` or `"date"` | `"date"` |
| `reverse` | false (ascending) | true (descending) |

---

## Output Structure

For a series named `building-a-rest-api`:

```
output/
└── series/
    └── building-a-rest-api/
        ├── index.html              # Series listing page
        ├── simple/
        │   └── index.html          # Simple list view
        ├── rss.xml                 # RSS feed
        ├── atom.xml                # Atom feed
        ├── feed.json               # JSON feed
        └── sitemap.xml             # Sitemap
```

### Series Index Page

The series index page displays posts in order with position indicators:

```
Building a REST API with Go
============================
A complete guide to building REST APIs

Part 1: Project Setup and Structure
Part 2: Database Models and Migrations
Part 3: Handler Implementation
Part 4: Authentication and Middleware
Part 5: Testing and Deployment

5 posts · 45 min total reading time
```

---

## Navigation

### Feed Sidebar

When a post belongs to a series, the feed sidebar **takes precedence** over any other feed sidebar. The sidebar shows all posts in the series as a table of contents, with the current post highlighted.

```
┌──────────────────────────┐
│  Building a REST API     │
│  ────────────────────    │
│  1. Project Setup     ◀  │  ← current post
│  2. Database Models      │
│  3. Handlers             │
│  4. Auth & Middleware    │
│  5. Testing & Deploy     │
│                          │
│  Part 1 of 5             │
└──────────────────────────┘
```

**Precedence rule:** If a post has `series` frontmatter, the series sidebar is shown instead of any feed-based sidebar, even if the post appears in other feeds.

### Prev/Next Navigation

Series posts get prev/next navigation that follows series order:

- **Previous**: Links to the prior post in the series (or disabled if first)
- **Next**: Links to the next post in the series (or disabled if last)
- **Series link**: "View All" links to the series index page

This is rendered using the existing `guide-navigation.html` partial.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `[` | Navigate to previous post in series |
| `]` | Navigate to next post in series |

These use the existing navigation shortcut infrastructure. When a post is in a series, `[`/`]` navigate within the series rather than any other feed.

### Position Indicator

Posts in a series display a position indicator:

```
Part 3 of 7 · Building a REST API
```

Available in templates as:

```jinja2
{{ post.PrevNextContext.Position }} of {{ post.PrevNextContext.Total }}
```

---

## Template Context

### Post Template Variables

When a post belongs to a series, these template variables are available:

| Variable | Type | Description |
|----------|------|-------------|
| `post.series` | string | Series identifier |
| `post.series_order` | int | Explicit position (if set) |
| `post.Prev` | Post | Previous post in series |
| `post.Next` | Post | Next post in series |
| `post.PrevNextFeed` | string | Series feed slug |
| `post.PrevNextContext.FeedSlug` | string | Series feed slug |
| `post.PrevNextContext.FeedTitle` | string | Series title |
| `post.PrevNextContext.Position` | int | 1-indexed position |
| `post.PrevNextContext.Total` | int | Total posts in series |

### Feed Template Variables

The series feed template receives standard feed context:

| Variable | Type | Description |
|----------|------|-------------|
| `feed.title` | string | Series title |
| `feed.description` | string | Series description |
| `feed.posts` | []Post | All posts in order |
| `feed.config.type` | string | `"series"` |

### Discovery Feed

Posts in a series advertise the series feed for RSS/Atom discovery:

```html
<link rel="alternate" type="application/rss+xml"
      title="Building a REST API (RSS)"
      href="/series/building-a-rest-api/rss.xml">
```

---

## Plugin Architecture

### Plugin: `series`

| Property | Value |
|----------|-------|
| **Name** | `series` |
| **Stage** | Collect |
| **Priority** | `PriorityEarly` (-100) |
| **Runs before** | `feeds`, `prevnext` |

The series plugin runs early in the Collect stage so that:
1. Series feeds are created before the feeds plugin processes them
2. The prevnext plugin can use series feeds for navigation

### Processing Steps

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SERIES PLUGIN FLOW                            │
├─────────────────────────────────────────────────────────────────────┤
│  1. SCAN                                                             │
│     - Iterate all posts                                              │
│     - Collect posts with series frontmatter                          │
│     - Group by series name                                           │
│                                                                      │
│  2. CONFIGURE                                                        │
│     - For each series group:                                         │
│       - Create FeedConfig with type=series                           │
│       - Apply series defaults                                        │
│       - Apply per-series overrides from config                       │
│                                                                      │
│  3. ORDER                                                            │
│     - Sort posts within each series                                  │
│     - Determine sort mode (explicit order vs date)                   │
│     - Set position metadata on each post                             │
│                                                                      │
│  4. NAVIGATE                                                         │
│     - Set Prev/Next pointers on each post                            │
│     - Set PrevNextFeed to series slug                                │
│     - Set PrevNextContext with position info                         │
│                                                                      │
│  5. INJECT                                                           │
│     - Append series FeedConfigs to config.Extra["feeds"]             │
│     - Set series sidebar metadata on posts                           │
│     - Store series metadata in cache for templates                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Integration with Existing Plugins

| Plugin | Integration |
|--------|-------------|
| `feeds` | Series FeedConfigs are injected before feeds processes them; feeds handles filtering, pagination, and output generation |
| `prevnext` | Series already sets Prev/Next, so prevnext skips posts that already have series navigation |
| `publish_feeds` | Writes series feed outputs (HTML, RSS, etc.) using standard feed publishing |
| `publish_html` | Renders series posts with sidebar and navigation context |
| `auto_feeds` | Series feeds coexist with auto-generated tag/category feeds |

### PrevNext Integration

The `prevnext` plugin must respect series navigation:

- If a post already has `PrevNextFeed` set by the series plugin, `prevnext` **skips** that post
- This ensures series navigation takes precedence regardless of prevnext strategy

---

## Configuration Reference

### Full Configuration Example

```toml
# =============================================================================
# SERIES CONFIGURATION
# =============================================================================

# Global series defaults
[markata-go.series]
slug_prefix = "series"             # URL prefix: /series/<name>/ (default: "series")
auto_sidebar = true                # Auto-enable sidebar for series posts (default: true)

# Series-specific defaults (override feed defaults for series feeds)
[markata-go.series.defaults]
items_per_page = 0                 # Show all posts (no pagination)
sidebar = true

[markata-go.series.defaults.formats]
html = true
simple_html = true
rss = true
atom = true
json = true
sitemap = true

# Override specific series
[markata-go.series.overrides."building-a-rest-api"]
title = "Building a REST API with Go"
description = "A complete guide from zero to production"

[markata-go.series.overrides."building-a-rest-api".formats]
markdown = true                    # Also generate markdown for this series
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slug_prefix` | string | `"series"` | URL prefix for series feeds |
| `auto_sidebar` | bool | `true` | Auto-enable feed sidebar for series posts |
| `defaults` | object | see below | Default settings for all series feeds |
| `overrides` | map[string]object | `{}` | Per-series configuration overrides |

---

## Edge Cases

### Post in Series but Not Published

- Post is included in series ordering for context
- Post is **excluded** from series feed output (not in HTML/RSS/etc.)
- Prev/Next skip unpublished posts (link to next published post)

### Empty Series

- If all posts in a series are unpublished/draft, no series feed is generated
- A warning is logged

### Single-Post Series

- Valid but unusual; series feed is generated with one post
- No prev/next navigation (both disabled)

### Duplicate series_order

- Posts with the same `series_order` are sub-sorted by date, then path
- A warning is logged about duplicate order values

### Series Value Normalization

- Series values are slugified: `"Building a REST API"` becomes `"building-a-rest-api"`
- This ensures consistent feed slugs regardless of how the value is written in frontmatter

---

## See Also

- [FEEDS.md](./FEEDS.md) - Feed system specification
- [DATA_MODEL.md](./DATA_MODEL.md) - Post model
- [DEFAULT_PLUGINS.md](./DEFAULT_PLUGINS.md) - Plugin registry
- [TEMPLATES.md](./TEMPLATES.md) - Template system
