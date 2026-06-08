# Feed Patterns Reference

Feeds are filtered, sorted, paginated collections of posts. In markata-go, many archive and index pages are feeds.

## Minimal Feed

```toml
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true
```

## Common Patterns

### Home Feed

```toml
[[markata-go.feeds]]
slug = ""
title = "Home"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5
```

### Blog Archive

```toml
[[markata-go.feeds]]
slug = "blog"
title = "All Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 10
```

### Tag-Like Feed

```toml
[[markata-go.feeds]]
slug = "go"
title = "Go Posts"
filter = "'go' in tags and published == True"
sort = "date"
reverse = true
```

### Docs Feed

```toml
[[markata-go.feeds]]
slug = "docs"
title = "Documentation"
filter = "published == True"
sort = "title"
reverse = false
items_per_page = 0
```

## Global Feed Formats

This is a top-level setting that controls which output formats are generated for all feeds. It is NOT placed inside a `[[markata-go.feeds]]` entry.

```toml
[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = true
markdown = false
text = false
```

## Feed Defaults

If the site uses shared feed defaults, check those before editing each feed:

```toml
[markata-go.feeds.defaults]
items_per_page = 10
pagination_type = "manual"

[markata-go.feeds.defaults.formats]
html = true
rss = true
atom = true
sitemap = true

[markata-go.feeds.defaults.syndication]
max_items = 20
include_content = true
```

## Pagination Types

Each feed can set `pagination_type` to control how page navigation works:

- `"manual"` (default): traditional page links with full page reloads
- `"htmx"`: HTMX-powered seamless page loading
- `"htmx-infinite"`: HTMX-powered infinite scroll
- `"js"`: client-side JavaScript pagination

## Auto-Generated Feeds

Markata-go can automatically generate feeds from tags, categories, and date archives without manually defining each one.

```toml
[markata-go.feeds.auto_tags]
enabled = true
slug_prefix = "tags"

[markata-go.feeds.auto_categories]
enabled = true
slug_prefix = "categories"

[markata-go.feeds.auto_archives]
enabled = true
slug_prefix = "archive"
yearly_feeds = true
monthly_feeds = false
```

**Defaults**: `auto_tags` is enabled by default; `auto_categories` and `auto_archives` are disabled by default.

Auto-tag feeds generate one feed per unique tag (e.g., `/tags/python/`). Auto-category feeds generate one feed per unique `category` frontmatter value. Auto-archive feeds generate per-year (and optionally per-month) feeds.

## Subscription Feeds

Two built-in subscription feeds are injected automatically:

- Root feed (slug `""`) at `/rss.xml` and `/atom.xml` (HTML output disabled)
- Archive feed (slug `"archive"`) at `/archive/rss.xml` and `/archive/atom.xml`

To disable:

```toml
[markata-go]
subscription_feeds_disabled = true
```

## Sitemap Generation

Sitemaps are generated automatically:

- A site-level `sitemap.xml` includes the homepage (priority 1.0), all published posts (priority 0.8), and all feed index pages (priority 0.6).
- Per-feed `sitemap.xml` files are generated when the feed format `sitemap = true` (default: enabled).

No additional config is needed. The sitemap plugin runs in the Write stage after all content is rendered.

## Template Touchpoints

- list/archive HTML usually uses `feed.html`
- per-feed HTML can switch to another built-in template with `[markata-go.feeds.templates] html = "feed-photo-grid.html"`
- card rendering often happens in a partial
- RSS and Atom can use separate XML templates

## Feed Sidebar Windowing

Large feeds can make post sidebars expensive and noisy. Use `max_posts` under `[markata-go.components.feed_sidebar]` to cap sidebar entries while keeping the current post visible.

```toml
[markata-go.components.feed_sidebar]
enabled = true
feeds = ["blog", "docs"]
max_posts = 51
```

A value of `0` or an omitted value means no cap. When capped, markata-go renders a contiguous window around the current post when possible.

## Agent Guidance

- if the task is “change an index page”, check whether that page is backed by a feed first
- if the task is “show only X posts”, check `filter`, `limit`, `offset`, and `items_per_page`
- if the task is “make post sidebars shorter” or “speed up huge series sidebars”, check `components.feed_sidebar.max_posts`
- if the task is “change archive card layout”, change the feed template or card partial before changing content
