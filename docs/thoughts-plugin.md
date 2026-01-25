# Thoughts Plugin

The thoughts plugin provides PESOS (Publish Elsewhere, Syndicate to Own Site) functionality for markata-go, enabling you to import and manage microblog content from external platforms like Mastodon, Twitter, and RSS feeds.

## Features

- **PESOS Import**: Import thoughts from Mastodon RSS feeds, Twitter (planned), and other RSS sources
- **Thought Management**: Create, organize, and display short-form content alongside regular posts
- **Syndication**: Post thoughts to external platforms (planned)
- **Feed Integration**: Automatic feed generation for thoughts with customizable filtering
- **Caching**: Intelligent caching to avoid re-fetching external content
- **Template Support**: Dedicated templates for thought display

## Quick Start

### 1. Enable the Plugin

Add `thoughts` to your hooks in `markata-go.toml`:

```toml
hooks = ["default", "thoughts"]
```

### 2. Basic Configuration

```toml
[thoughts]
enabled = true
thoughts_dir = "thoughts"
cache_dir = "cache/thoughts"
max_items = 200
```

### 3. Add External Sources

Configure external sources to import from:

```toml
[thoughts.sources]
mastodon = { type = "mastodon", url = "https://mastodon.social/@username.rss", handle = "username", active = true, max_items = 50 }
blog = { type = "rss", url = "https://myblog.com/feed.xml", active = true, max_items = 20 }
```

### 4. Configure Feeds

Add feeds for your thoughts:

```toml
[[markata-go.feeds]]
slug = "thoughts"
title = "Thoughts"
description = "Quick notes and thoughts"
filter = "template == 'thought.html' or path startswith 'thoughts/'"
sort = "date"
reverse = true
items_per_page = 50
```

## Configuration

### Plugin Settings

| Setting | Type | Default | Description |
|----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `thoughts_dir` | string | `"thoughts"` | Directory for local thought files |
| `cache_dir` | string | `"cache/thoughts"` | Cache directory for external feeds |
| `max_items` | int | `200` | Maximum thoughts to keep |

### External Sources

Configure external sources in `[thoughts.sources]`:

```toml
[thoughts.sources.source_name]
type = "mastodon" | "twitter" | "rss"
url = "https://example.com/feed.rss"
handle = "username"
active = true
max_items = 50
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Source type: `mastodon`, `twitter`, `rss` |
| `url` | string | Required for RSS/Mastodon | Feed URL |
| `handle` | string | No | User handle for attribution |
| `active` | bool | No | Whether to import from this source |
| `max_items` | int | No | Maximum items to import |

### Syndication Configuration

Configure outbound syndication in `[thoughts.syndication]`:

```toml
[thoughts.syndication]
enabled = false

[thoughts.syndication.mastodon]
access_token = "your_token"
instance_url = "https://mastodon.social"
character_limit = 500

[thoughts.syndication.twitter]
api_key = "your_api_key"
api_secret = "your_api_secret"
access_token = "your_access_token"
access_secret = "your_access_secret"
character_limit = 280
```

## Creating Thoughts

### Local Thought Files

Create markdown files in your `thoughts_dir` with the `thought.html` template:

```markdown
---
title: "Just discovered this amazing tool"
date: 2024-01-24T15:30:00Z
published: true
tags: ["tools", "discovery"]
template: "thought.html"
syndicate_to: ["mastodon"]
---

This looks really interesting! Found it while browsing and wanted to share my initial thoughts.

The approach seems solid and the implementation is clean. Thinking about trying it out for my next project.

#tools #discovery
```

### Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Thought title (required) |
| `date` | datetime | Publication date |
| `published` | bool | Whether to publish (default: true) |
| `tags` | array | Tags/categorization |
| `template` | string | Use `"thought.html"` for thoughts |
| `syndicate_to` | array | Platforms to syndicate to |
| `original_url` | string | Original URL for link commentary |
| `thought_type` | string | Type: `"link_commentary"`, `"note"`, etc. |

### Content Guidelines

- Keep thoughts concise (1-3 paragraphs recommended)
- Use hashtags at the end for tagging
- For link commentary, include the original URL
- Thoughts support full markdown formatting

## Feed Integration

Thoughts integrate seamlessly with markata-go's feed system:

### Filtering Thoughts

Use the filter syntax to create custom thought feeds:

```toml
# All thoughts
filter = "template == 'thought.html'"

# Only micro thoughts
filter = "template == 'thought.html' and 'microblog' in tags"

# External thoughts only
filter = "template == 'thought.html' and is_external_thought == true"

# Thoughts from specific source
filter = "template == 'thought.html' and thought_source == 'mastodon'"
```

### Common Feed Configurations

```toml
# Main thoughts feed
[[markata-go.feeds]]
slug = "thoughts"
title = "Thoughts"
filter = "template == 'thought.html'"
sort = "date"
reverse = true
items_per_page = 50

# Micro blog (short thoughts only)
[[markata-go.feeds]]
slug = "micro"
title = "Micro Blog"
filter = "template == 'thought.html' and 'microblog' in tags"
sort = "date"
reverse = true
items_per_page = 20

# Link commentary
[[markata-go.feeds]]
slug = "links"
title = "Link Commentary"
filter = "template == 'thought.html' and thought_type == 'link_commentary'"
sort = "date"
reverse = true
items_per_page = 30
```

## Template Customization

The plugin includes a default `thought.html` template with:

- Responsive design
- Dark mode support
- Source attribution for imported thoughts
- Tag display
- Permalink structure

### Custom Templates

Override the default by creating your own `templates/thought.html`:

```html
<article class="thought {{ post.thought_type }}">
  <header>
    <time datetime="{{ post.date|dateformat }}">{{ post.date|dateformat }}</time>
    {% if post.thought_source %}
    <span class="source">from {{ post.thought_source }}</span>
    {% endif %}
  </header>
  
  <div class="content">
    {{ post.article_html|safe }}
  </div>
  
  {% if post.tags %}
  <footer>
    {% for tag in post.tags %}
    <span class="tag">{{ tag }}</span>
    {% endfor %}
  </footer>
  {% endif %}
</article>
```

### Available Template Variables

Thought posts include all standard post variables plus:

| Variable | Type | Description |
|----------|------|-------------|
| `post.thought_source` | string | Source name for imported thoughts |
| `post.thought_type` | string | Type of thought |
| `post.original_url` | string | Original external URL |
| `post.external_id` | string | ID from external source |
| `post.source_handle` | string | Handle from external source |
| `post.is_external_thought` | bool | True for imported thoughts |
| `post.image_url` | string | Featured image if available |

## Advanced Usage

### Custom Import Logic

The plugin automatically caches external feeds to avoid redundant requests. Cache files are stored in JSON format in your cache directory.

### Thought Types

Organize thoughts by type using the `thought_type` field:

- `"note"` - Simple text thoughts
- `"link_commentary"` - Links with your commentary
- `"status"` - Status updates
- `"reply"` - Replies to other content

### Tag Strategies

Use consistent tagging for better organization:

```markdown
---
tags: ["microblog", "tech", "discovery"]
---
```

Common tag patterns:
- `microblog` - Short status updates
- `link-commentary` - Link shares with commentary
- Platform-specific: `mastodon`, `twitter`
- Topic tags: `tech`, `programming`, `design`

## Troubleshooting

### Common Issues

**Thoughts not appearing:**
- Check that `thoughts.enabled = true` in config
- Verify feed filter includes thought template
- Ensure thoughts have `published: true`

**External feeds not importing:**
- Verify source `active = true`
- Check feed URL is accessible
- Review cache directory permissions

**Feed filtering not working:**
- Test filter expressions with simpler conditions first
- Use quotes around string values: `template == 'thought.html'`
- Check available filter fields in template variables

### Debug Mode

Enable verbose logging to troubleshoot:

```bash
markata-go build --verbose
```

Check cache files to debug import issues:

```bash
ls -la cache/thoughts/
cat cache/thoughts/feed_source_name.json
```

## Examples

See the `examples/` directory for:

- `thoughts-config.toml` - Complete configuration example
- `examples/thoughts/` - Sample thought files
- `templates/thought.html` - Default template

## Migration from Other Platforms

### From Twitter

Export your tweets and convert to thought files:

```bash
# Convert tweets to markdown thoughts
python convert_tweets.py tweets.json thoughts/
```

### From Mastodon

Use the RSS feed import:

```toml
[thoughts.sources.mastodon]
type = "mastodon"
url = "https://mastodon.social/@username.rss"
active = true
```

### From Existing Blog

Filter existing posts:

```toml
[[markata-go.feeds]]
slug = "short-posts"
title = "Quick Thoughts"
filter = "word_count < 200 and template == 'post.html'"
```

## Roadmap

- [ ] Twitter API integration for import
- [ ] Complete syndication implementation
- [ ] Thread support for related thoughts
- [ ] Image/attachment handling
- [ ] Webmention integration
- [ ] ActivityPub support
- [ ] Advanced filtering expressions