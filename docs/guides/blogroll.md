---
title: "Blogroll & Reader Guide"
description: "Create a blogroll of blogs you follow and a reader page showing their latest posts"
date: 2024-01-15
published: true
slug: /docs/guides/blogroll/
tags:
  - documentation
  - blogroll
  - rss
  - feeds
---

# Blogroll & Reader

The blogroll plugin lets you curate a list of blogs you follow and automatically fetch their RSS/Atom feeds to create:

- **`/blogroll/`** - A directory of blogs you follow, organized by category
- **`/reader/`** - A river-of-news style page showing latest posts from all feeds

This is perfect for sharing your reading list, building community connections, or creating a personal feed reader built into your site.

## Quick Start

Add this to your `markata-go.toml`:

```toml
[blogroll]
enabled = true

[[blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
category = "Technology"
```

Run `markata-go build` and you'll have:
- `/blogroll/` - Lists Simon Willison's blog
- `/reader/` - Shows his latest posts

## Configuration

### Basic Settings

```toml
[blogroll]
enabled = true                    # Enable the blogroll plugin
cache_dir = "cache/blogroll"      # Where to cache fetched feeds
cache_duration = "1h"             # How long to cache (default: 1 hour)
timeout = 30                      # HTTP request timeout in seconds
concurrent_requests = 5           # Max parallel feed fetches
max_entries_per_feed = 50         # Global default entries per feed
```

### Pagination Settings

The reader page supports pagination to handle large numbers of entries:

```toml
[blogroll]
enabled = true
items_per_page = 50               # Entries per page (default: 50)
orphan_threshold = 3              # Min entries for separate page (default: 3)
pagination_type = "manual"        # "manual", "htmx", or "js"
```

**Pagination Types:**

| Type | Description |
|------|-------------|
| `manual` | Traditional page links with full page reloads |
| `htmx` | Seamless AJAX-based navigation using HTMX |
| `js` | Client-side JavaScript pagination |

The paginated reader generates:
```
/reader/              # Page 1
/reader/page/2/       # Page 2
/reader/page/3/       # Page 3
/reader/page/2/partial/   # HTMX partial for page 2
```

### Adding Feeds

Add feeds using the `[[blogroll.feeds]]` array:

```toml
[[blogroll.feeds]]
url = "https://example.com/feed.xml"    # Required: RSS/Atom feed URL
title = "Example Blog"                   # Optional: display name (auto-fetched if not set)
description = "A great blog about stuff" # Optional: short description
category = "Technology"                  # Optional: groups feeds on blogroll page
tags = ["python", "web"]                 # Optional: additional labels
site_url = "https://example.com"         # Optional: main website URL
image_url = "https://example.com/logo.png" # Optional: logo/icon
active = true                            # Optional: set false to disable without removing
max_entries = 50                         # Optional: override global max_entries_per_feed
```

### Per-Feed Entry Limits

Override the global `max_entries_per_feed` for individual feeds:

```toml
[blogroll]
enabled = true
max_entries_per_feed = 50         # Global default

[[blogroll.feeds]]
url = "https://prolific-blogger.com/feed.xml"
title = "Prolific Blogger"
max_entries = 100                 # Override: this site posts frequently

[[blogroll.feeds]]
url = "https://micro.blog/user.xml"
title = "Micro Blog"
max_entries = 200                 # Override: many small posts

[[blogroll.feeds]]
url = "https://infrequent-poster.com/feed.xml"
title = "Infrequent Poster"
max_entries = 10                  # Override: rarely posts, save cache space

[[blogroll.feeds]]
url = "https://normal-blog.com/feed.xml"
title = "Normal Blog"
# Uses global default: 50 entries
```

### Feed Configuration Reference

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `url` | Yes | - | RSS or Atom feed URL |
| `title` | No | Auto-fetched | Display name for the feed |
| `description` | No | Auto-fetched | Short description |
| `category` | No | "Uncategorized" | Groups feeds together |
| `tags` | No | `[]` | Additional labels for filtering |
| `site_url` | No | Auto-fetched | Main website URL |
| `image_url` | No | Auto-fetched | Logo or icon URL |
| `active` | No | `true` | Set to `false` to disable |
| `max_entries` | No | Global default | Override max entries for this feed |

### Custom Templates

Override the default templates:

```toml
[blogroll.templates]
blogroll = "blogroll.html"    # Template for /blogroll/ page
reader = "reader.html"        # Template for /reader/ page
```

## Example: Building a Reading List

Here's a complete example with multiple feeds organized by category:

```toml
[blogroll]
enabled = true
cache_duration = "2h"
max_entries_per_feed = 25

# =============================================================================
# TECHNOLOGY
# =============================================================================

[[blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
description = "Creator of Datasette, Django co-creator, AI/LLM enthusiast"
category = "Technology"
tags = ["python", "ai", "llm", "sqlite"]
site_url = "https://simonwillison.net"

[[blogroll.feeds]]
url = "https://jvns.ca/atom.xml"
title = "Julia Evans"
description = "Making hard things easy to understand"
category = "Technology"
tags = ["linux", "networking", "zines"]
site_url = "https://jvns.ca"

[[blogroll.feeds]]
url = "https://danluu.com/atom.xml"
title = "Dan Luu"
description = "Deep dives into computer systems"
category = "Technology"
tags = ["systems", "performance"]
site_url = "https://danluu.com"

[[blogroll.feeds]]
url = "https://blog.codinghorror.com/rss/"
title = "Coding Horror"
description = "Jeff Atwood on programming and human factors"
category = "Technology"
tags = ["programming", "software"]
site_url = "https://blog.codinghorror.com"

# =============================================================================
# DESIGN
# =============================================================================

[[blogroll.feeds]]
url = "https://alistapart.com/main/feed/"
title = "A List Apart"
description = "For people who make websites"
category = "Design"
tags = ["web", "ux", "accessibility"]
site_url = "https://alistapart.com"

[[blogroll.feeds]]
url = "https://css-tricks.com/feed/"
title = "CSS-Tricks"
description = "Tips, tricks, and techniques on using CSS"
category = "Design"
tags = ["css", "frontend"]
site_url = "https://css-tricks.com"

# =============================================================================
# PERSONAL
# =============================================================================

[[blogroll.feeds]]
url = "https://austinkleon.com/feed/"
title = "Austin Kleon"
description = "Writer and artist"
category = "Personal"
tags = ["creativity", "writing"]
site_url = "https://austinkleon.com"
```

## Adding Simon Willison

Simon Willison is a prolific blogger known for creating Datasette, co-creating Django, and writing extensively about AI/LLMs. His feed URL is:

```
https://simonwillison.net/atom/everything/
```

Add him to your blogroll:

```toml
[[blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
description = "Creator of Datasette, Django co-creator, AI/LLM enthusiast"
category = "Technology"
tags = ["python", "ai", "llm", "sqlite", "datasette"]
site_url = "https://simonwillison.net"
```

Simon also has topic-specific feeds if you want to subscribe to specific content:
- `https://simonwillison.net/atom/entries/` - Blog entries only
- `https://simonwillison.net/atom/links/` - Links/bookmarks only

## Finding Feed URLs

Most blogs have RSS/Atom feeds. Here's how to find them:

1. **Look for feed icons** - Usually in the header, footer, or sidebar
2. **Check common paths:**
   - `/feed/`
   - `/rss/`
   - `/atom.xml`
   - `/feed.xml`
   - `/rss.xml`
   - `/index.xml`
3. **View page source** - Search for `application/rss+xml` or `application/atom+xml`
4. **Use browser extensions** - Feed discovery extensions can help

### Common Feed URL Patterns

| Platform | Feed URL Pattern |
|----------|-----------------|
| WordPress | `/feed/` or `/feed/rss/` |
| Ghost | `/rss/` |
| Substack | `/feed` |
| Medium | `/feed` |
| Jekyll | `/feed.xml` |
| Hugo | `/index.xml` |
| dev.to | `/feed` |
| GitHub Releases | `/releases.atom` |

## Generated Pages

### Blogroll Page (`/blogroll/`)

The blogroll page lists all feeds grouped by category:

```
/blogroll/
  index.html
```

**Default layout:**
- Header with title and feed count
- Feeds grouped by category
- Each feed shows: title, description, post count
- Links to the original site

### Reader Page (`/reader/`)

The reader page shows the latest posts from all feeds in reverse chronological order with pagination:

```
/reader/
  index.html              # Page 1
  partial/
    index.html            # Page 1 partial (HTMX)
  page/
    2/
      index.html          # Page 2
      partial/
        index.html        # Page 2 partial (HTMX)
    3/
      index.html
      partial/
        index.html
```

**Default layout:**
- Header with title
- List of recent entries (newest first)
- Each entry shows: title, source feed, date, description
- Links to the original article
- Pagination navigation (when more than one page)

## Custom Templates

Create custom templates for full control over the appearance.

### Blogroll Template

Create `templates/blogroll.html`:

```html
{% extends "base.html" %}

{% block content %}
<main class="blogroll">
  <h1>{{ title }}</h1>
  <p class="subtitle">{{ feed_count }} blogs I follow</p>

  {% for category in categories %}
  <section class="category" id="{{ category.Slug }}">
    <h2>{{ category.Name }}</h2>
    <div class="feed-grid">
      {% for feed in category.Feeds %}
      <article class="feed-card">
        {% if feed.ImageURL %}
        <img src="{{ feed.ImageURL }}" alt="{{ feed.Title }}" class="feed-icon">
        {% endif %}
        <h3>
          {% if feed.SiteURL %}
          <a href="{{ feed.SiteURL }}" target="_blank" rel="noopener">{{ feed.Title }}</a>
          {% else %}
          {{ feed.Title }}
          {% endif %}
        </h3>
        {% if feed.Description %}
        <p class="description">{{ feed.Description }}</p>
        {% endif %}
        <div class="meta">
          <span class="post-count">{{ feed.EntryCount }} posts</span>
          <a href="{{ feed.FeedURL }}" class="feed-link" title="RSS Feed">
            <svg><!-- RSS icon --></svg>
          </a>
        </div>
      </article>
      {% endfor %}
    </div>
  </section>
  {% endfor %}
</main>
{% endblock %}
```

### Reader Template

Create `templates/reader.html`:

```html
{% extends "base.html" %}

{% block content %}
<main class="reader">
  <h1>{{ title }}</h1>
  <p class="subtitle">Latest posts from blogs I follow</p>

  <ul class="entry-list">
    {% for entry in entries %}
    <li class="entry">
      <article>
        <h2>
          <a href="{{ entry.URL }}" target="_blank" rel="noopener">
            {{ entry.Title }}
          </a>
        </h2>
        <div class="meta">
          <span class="source">{{ entry.FeedTitle }}</span>
          {% if entry.Published %}
          <time datetime="{{ entry.Published|atom_date }}">
            {{ entry.Published|date_format:"Jan 2, 2006" }}
          </time>
          {% endif %}
          {% if entry.ReadingTime > 0 %}
          <span class="reading-time">{{ entry.ReadingTime }} min read</span>
          {% endif %}
        </div>
        {% if entry.Description %}
        <p class="description">{{ entry.Description|striptags|truncate:200 }}</p>
        {% endif %}
      </article>
    </li>
    {% endfor %}
  </ul>
</main>
{% endblock %}
```

## Template Variables

### Blogroll Template Variables

| Variable | Type | Description |
|----------|------|-------------|
| `title` | string | Page title ("Blogroll") |
| `description` | string | Page description |
| `feeds` | []ExternalFeed | All feeds |
| `categories` | []BlogrollCategory | Feeds grouped by category |
| `feed_count` | int | Total number of feeds |

### Reader Template Variables

| Variable | Type | Description |
|----------|------|-------------|
| `title` | string | Page title ("Reader") |
| `description` | string | Page description |
| `entries` | []ExternalEntry | Entries for current page (newest first) |
| `entry_count` | int | Total number of entries across all pages |
| `page` | ReaderPage | Pagination information |
| `pagination_type` | string | Pagination type ("manual", "htmx", "js") |

### ReaderPage Fields

| Field | Type | Description |
|-------|------|-------------|
| `number` | int | Current page number (1-indexed) |
| `has_prev` | bool | True if previous page exists |
| `has_next` | bool | True if next page exists |
| `prev_url` | string | URL of previous page |
| `next_url` | string | URL of next page |
| `total_pages` | int | Total number of pages |
| `total_items` | int | Total number of entries |
| `items_per_page` | int | Entries per page |
| `page_urls` | []string | URLs for all pages |
| `pagination_type` | string | Pagination type |

### ExternalFeed Fields

| Field | Type | Description |
|-------|------|-------------|
| `Title` | string | Feed title |
| `Description` | string | Feed description |
| `SiteURL` | string | Main website URL |
| `FeedURL` | string | RSS/Atom feed URL |
| `ImageURL` | string | Feed logo/icon |
| `Category` | string | Feed category |
| `Tags` | []string | Feed tags |
| `EntryCount` | int | Number of entries |
| `Entries` | []ExternalEntry | Feed entries |
| `LastFetched` | *time.Time | When feed was last fetched |
| `LastUpdated` | *time.Time | Feed's last update date |
| `Error` | string | Error message if fetch failed |

### ExternalEntry Fields

| Field | Type | Description |
|-------|------|-------------|
| `Title` | string | Entry title |
| `URL` | string | Link to full article |
| `Description` | string | Summary or excerpt |
| `Content` | string | Full content (HTML) |
| `Author` | string | Entry author |
| `Published` | *time.Time | Publication date |
| `Updated` | *time.Time | Last update date |
| `Categories` | []string | Entry categories/tags |
| `ImageURL` | string | Featured image |
| `ReadingTime` | int | Estimated reading time (minutes) |
| `FeedURL` | string | Source feed URL |
| `FeedTitle` | string | Source feed title |

### BlogrollCategory Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | string | Category name |
| `Slug` | string | URL-safe identifier |
| `Feeds` | []ExternalFeed | Feeds in this category |

## Caching

The blogroll plugin caches fetched feeds to avoid hitting external servers on every build.

### Cache Configuration

```toml
[blogroll]
cache_dir = "cache/blogroll"    # Cache directory
cache_duration = "1h"           # How long to cache feeds
```

### Cache Behavior

1. On first build, all feeds are fetched and cached
2. On subsequent builds, cached feeds are used if still valid
3. Cache expires after `cache_duration`
4. Delete `cache/blogroll/` to force a fresh fetch

### Cache Duration Examples

```toml
cache_duration = "30m"    # 30 minutes
cache_duration = "1h"     # 1 hour (default)
cache_duration = "6h"     # 6 hours
cache_duration = "24h"    # 1 day
cache_duration = "168h"   # 1 week
```

**Recommendations:**
- Development: `"5m"` - See changes quickly
- Production: `"1h"` to `"6h"` - Balance freshness and build speed
- High-traffic: `"24h"` or more - Reduce external requests

## Error Handling

When a feed fails to fetch, the plugin:

1. Records the error in `feed.Error`
2. Continues processing other feeds
3. Includes the feed in the blogroll (with error indicator)
4. Uses cached data if available

### Handling Errors in Templates

```html
{% for feed in feeds %}
<article class="feed-card {% if feed.Error %}feed-error{% endif %}">
  <h3>{{ feed.Title }}</h3>
  {% if feed.Error %}
  <p class="error">Unable to fetch: {{ feed.Error }}</p>
  {% else %}
  <p>{{ feed.EntryCount }} posts</p>
  {% endif %}
</article>
{% endfor %}
```

## Performance Tips

### Optimize Build Times

1. **Increase cache duration** for production builds
2. **Limit `max_entries_per_feed`** if you only need recent posts
3. **Reduce `concurrent_requests`** if you're hitting rate limits
4. **Disable inactive feeds** with `active = false` instead of removing them

### Example Production Config

```toml
[blogroll]
enabled = true
cache_dir = "cache/blogroll"
cache_duration = "6h"           # Cache for 6 hours
timeout = 15                    # Shorter timeout
concurrent_requests = 3         # Be nice to servers
max_entries_per_feed = 20       # Only recent posts
```

## Configuration Reference

### Full Configuration

```toml
[blogroll]
# Enable/disable the entire feature
enabled = true

# Cache settings
cache_dir = "cache/blogroll"
cache_duration = "1h"

# HTTP settings
timeout = 30
concurrent_requests = 5
max_entries_per_feed = 50

# Pagination settings
items_per_page = 50
orphan_threshold = 3
pagination_type = "manual"    # "manual", "htmx", or "js"

# Custom templates
[blogroll.templates]
blogroll = "blogroll.html"
reader = "reader.html"

# Feeds
[[blogroll.feeds]]
url = "https://example.com/feed.xml"
title = "Example"
description = "Description"
category = "Category"
tags = ["tag1", "tag2"]
site_url = "https://example.com"
image_url = "https://example.com/logo.png"
active = true
max_entries = 50              # Override global max_entries_per_feed
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable blogroll plugin |
| `cache_dir` | string | `"cache/blogroll"` | Cache directory |
| `cache_duration` | string | `"1h"` | Cache TTL (Go duration) |
| `timeout` | int | `30` | HTTP timeout in seconds |
| `concurrent_requests` | int | `5` | Max parallel fetches |
| `max_entries_per_feed` | int | `50` | Global max entries per feed |
| `items_per_page` | int | `50` | Entries per reader page |
| `orphan_threshold` | int | `3` | Min entries for separate page |
| `pagination_type` | string | `"manual"` | Pagination style |
| `feeds` | []Feed | `[]` | List of feeds |
| `templates.blogroll` | string | `"blogroll.html"` | Blogroll template |
| `templates.reader` | string | `"reader.html"` | Reader template |

---

## Next Steps

- [Feeds Guide](/docs/guides/feeds/) - Create feeds from your own content
- [Templates Guide](/docs/guides/templates/) - Customize blogroll appearance
- [Syndication Guide](/docs/guides/syndication-feeds/) - Share your own content via RSS

---

## See Also

- [Configuration Guide](/docs/guides/configuration/) - Full configuration reference
- [Themes Guide](/docs/guides/themes/) - Style your blogroll with themes
