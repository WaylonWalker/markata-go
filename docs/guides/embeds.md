---
title: "Embeds"
description: "Embed internal posts and external URLs as rich preview cards"
date: 2024-01-24
published: true
tags:
  - documentation
  - plugins
  - content
---

# Embeds

The embeds plugin lets you embed rich preview cards for both internal posts and external URLs in your markdown content.

## Quick Start

### Internal Post Embeds

Embed another post from your site using the `![[slug]]` syntax:

```markdown
Check out this related article:

![[getting-started]]
```

This creates a preview card showing the embedded post's title, description, and date.

### External URL Embeds

Embed external URLs using either `![embed](url)` or Obsidian-style `![[https://url]]`:

```markdown
Here's a great resource:

![embed](https://example.com/article)

![[https://example.com/article]]
![[https://example.com/article|Custom Title]]
```

This fetches metadata and displays a card with the title, description, and image.

> **Note:** Obsidian-style external embeds like `![[https://example.com]]` are not supported yet. Track progress in issue #837.

## Internal Embeds

### Basic Syntax

```markdown
![[slug]]
```

The slug is the URL path of the post you want to embed. For example, if a post lives at `/guides/configuration/`, use:

```markdown
![[guides/configuration]]
```

### Custom Title

Override the display title with a pipe:

```markdown
![[guides/configuration|Configuration Guide]]
```

### What Gets Displayed

Internal embed cards show:
- **Title** - From the post or your custom title
- **Description** - First 200 characters
- **Date** - If the post has a date

### Not Found Behavior

If the embedded post doesn't exist, markata-go leaves the original syntax and adds an HTML comment:

```html
<!-- embed not found: nonexistent-post -->
![[nonexistent-post]]
```

This helps you spot broken embeds without breaking your build.

## External Embeds

### Syntax

```markdown
![embed](https://example.com/article)
![[https://example.com/article]]
![[https://example.com/article|Custom Title]]
```

> **Note:** The alt text must be exactly `embed`. Regular images are not affected. The Obsidian-style form is only recognized for full URLs (`http`/`https`).

### Open Graph Metadata

External embeds fetch and display (in order based on strategy):
- **oEmbed** - Title, provider name, thumbnail image (if available)
- **Open Graph** - `og:title`, `og:description`, `og:image`, `og:site_name`

### Caching

Metadata is cached for 7 days by default (configurable) to avoid repeated HTTP requests. Cache files are stored in `.cache/embeds/` by default.

### Fallback Behavior

If metadata can't be fetched:
- Title shows "External Link" (configurable)
- No image is displayed
- Domain name is shown

## Configuration

Add to your `markata-go.toml`:

```toml
[embeds]
enabled = true

# CSS classes for styling
internal_card_class = "embed-card"
external_card_class = "embed-card embed-card-external"

# External fetch settings
fetch_external = true        # Set false to skip HTTP requests
oembed_enabled = true        # Enable oEmbed resolution
resolution_strategy = "oembed_first"  # oembed_first | og_first | oembed_only
cache_dir = ".cache/embeds"  # Where to store cached metadata
cache_ttl = 604800           # Cache TTL in seconds (default 7 days)
timeout = 10                 # HTTP timeout in seconds

# Display settings
fallback_title = "External Link"  # Title when OG is unavailable
show_image = true                 # Show OG images

[embeds.providers]
youtube = { enabled = true }
vimeo = { enabled = true }
tiktok = { enabled = true }
flickr = { enabled = true }
spotify = { enabled = true }
soundcloud = { enabled = true }
```

### Disabling External Fetching

For faster builds or offline development:

```toml
[embeds]
fetch_external = false
```

External embeds will use the fallback title and show no image.

### Disabling oEmbed

```toml
[embeds]
oembed_enabled = false
resolution_strategy = "og_first"
```

This skips oEmbed providers and falls back to Open Graph metadata only.

## Styling

Embed cards use these CSS classes:

| Class | Description |
|-------|-------------|
| `.embed-card` | Base container |
| `.embed-card-external` | Added for external embeds |
| `.embed-card-link` | The clickable anchor |
| `.embed-card-image` | Image wrapper (external only) |
| `.embed-card-content` | Text content area |
| `.embed-card-title` | Title |
| `.embed-card-description` | Description |
| `.embed-card-meta` | Date or domain |

### Custom Styling

Override the default styles in your custom CSS:

```css
/* Make internal embeds more prominent */
.embed-card:not(.embed-card-external) {
  border-left-width: 6px;
  border-left-color: var(--color-accent);
}

/* Larger images for external embeds */
.embed-card-image {
  width: 250px;
}
```

## Code Blocks

Embed syntax inside code blocks is preserved:

````markdown
Normal embed:
![[my-post]]

Code example (not processed):
```
![[my-post]]
```
````

## Examples

### Blog Post with Related Reading

```markdown
---
title: "Advanced Configuration"
---

# Advanced Configuration

This guide builds on the basics. If you're new, start here:

![[getting-started]]

## Deep Dive

...content...

## Further Reading

Check out these external resources:

![embed](https://gohugo.io/getting-started/)

![embed](https://www.markdownguide.org/)
```

### Documentation with Cross-References

```markdown
# API Reference

## Authentication

For setup instructions, see:

![[guides/authentication|Authentication Guide]]

## Rate Limiting

Related:
![[api/errors]]
![[guides/best-practices]]
```

## Comparison with Wikilinks

| Feature | Embeds (`![[]]`) | Wikilinks (`[[]]`) |
|---------|------------------|-------------------|
| Output | Preview card | Inline link |
| Shows description | Yes | No |
| Shows date | Yes | No |
| Best for | Related content, references | In-text mentions |

Use embeds when you want readers to see a preview. Use wikilinks for inline mentions.

## Troubleshooting

### Embed Not Rendering

1. Check the slug matches the target post's URL path
2. Ensure the target post is published
3. Look for `<!-- embed not found -->` comments in the HTML

### External Embed Shows Fallback

1. Check your internet connection
2. Verify the URL is accessible
3. Some sites block robots - the fallback is expected
4. Increase timeout if the site is slow

### Images Not Showing

1. Ensure `show_image = true` in config
2. The page may not have an `og:image` tag
3. Image URL may be invalid or blocked

### Build Is Slow

External fetching adds latency. Options:

1. Set `fetch_external = false` during development
2. Reduce `timeout` value
3. Let the cache warm up (first build is slowest)
