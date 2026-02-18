# Embeds Plugin Specification

The embeds plugin enables rich embedding of both internal posts and external URLs within markdown content.

## Overview

The embeds plugin processes two types of embed syntax:

1. **Internal embeds** (`![[slug]]`) - Embed another post from the same site as a preview card
2. **External embeds** (`![embed](url)` or `![[https://url]]`) - Embed external URLs with Open Graph metadata

## Syntax

### Internal Embeds

```markdown
![[slug]]           # Basic internal embed
![[slug|Title]]     # Custom display title
```

### External Embeds

```markdown
![embed](https://example.com/article)  # External embed with OG metadata
![[https://example.com/article]]       # Obsidian-style external embed
![[https://example.com/article|Title]] # Obsidian-style with custom title
```

Note: The alt text must be exactly `embed` to trigger external embedding. Regular images with other alt text are not affected.

## Configuration

```toml
[embeds]
enabled = true                                    # Enable/disable the plugin
internal_card_class = "embed-card"               # CSS class for internal cards
external_card_class = "embed-card embed-card-external"  # CSS class for external cards
fetch_external = true                            # Fetch OG metadata from external URLs
cache_dir = ".cache/embeds"                      # Cache directory for external metadata
timeout = 10                                     # HTTP timeout in seconds
fallback_title = "External Link"                 # Title when OG title is unavailable
show_image = true                                # Show OG images in external embeds
```

## Internal Embeds

### Behavior

1. The plugin looks up the target post by slug (case-insensitive)
2. If found, it generates an embed card with:
   - Linked title (from display text or post title)
   - Description (truncated to 200 characters)
   - Date (if available)
3. If not found, it adds a warning comment and preserves the original syntax

### Self-Reference Protection

A post cannot embed itself. Attempting to do so adds a warning comment:

```html
<!-- cannot embed self -->
![[self-slug]]
```

### Generated HTML

```html
<div class="embed-card">
  <a href="/target-post/" class="embed-card-link">
    <div class="embed-card-content">
      <div class="embed-card-title">Target Post Title</div>
      <div class="embed-card-description">Post description...</div>
      <div class="embed-card-meta">Jan 15, 2024</div>
    </div>
  </a>
</div>
```

## External Embeds

### Behavior

1. Validates the URL (must be http or https)
2. Fetches Open Graph metadata from the URL (if enabled)
3. Caches metadata for 7 days
4. Generates an embed card with:
   - OG image (if available and enabled)
   - OG title (or fallback)
   - OG description (truncated)
   - Site name and domain

### Open Graph Metadata Extraction

The plugin extracts:
- `og:title` - Page title
- `og:description` - Page description
- `og:image` - Preview image URL
- `og:site_name` - Website name

Falls back to `<title>` and `<meta name="description">` if OG tags are missing.

### Caching

External metadata is cached as JSON files:
- Location: `.cache/embeds/` (configurable)
- File name: SHA-256 hash of URL (first 8 bytes)
- Expiration: 7 days

Example cache file:
```json
{
  "title": "Article Title",
  "description": "Article description",
  "image": "https://example.com/og-image.jpg",
  "site_name": "Example Site",
  "type": "article",
  "fetched_at": 1705350000
}
```

### Generated HTML

```html
<div class="embed-card embed-card-external">
  <a href="https://example.com/article" class="embed-card-link" target="_blank" rel="noopener noreferrer">
    <div class="embed-card-image">
      <img src="https://example.com/og-image.jpg" alt="" loading="lazy">
    </div>
    <div class="embed-card-content">
      <div class="embed-card-title">Article Title</div>
      <div class="embed-card-description">Article description...</div>
      <div class="embed-card-meta">Example Site &middot; example.com</div>
    </div>
  </a>
</div>
```

## Code Block Protection

Embed syntax inside fenced code blocks is preserved and not processed:

```markdown
Normal embed: ![[my-post]]

` ` `
Code example: ![[my-post]]
` ` `
```

## Lifecycle Stage

The embeds plugin runs in the **Transform** stage with `PriorityEarly` (-100), ensuring it processes content before:
- Wikilinks plugin
- Table of Contents extraction
- Jinja-MD processing

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Internal embed not found | Warning comment + original syntax preserved |
| Self-reference | Warning comment + original syntax preserved |
| External URL invalid | Original syntax preserved |
| Obsidian external URL invalid | Original syntax preserved |
| External fetch fails | Uses fallback title, no image |
| External timeout | Uses fallback title, no image |

## CSS Classes

| Class | Purpose |
|-------|---------|
| `.embed-card` | Base container |
| `.embed-card-external` | External embed modifier |
| `.embed-card-link` | Clickable anchor |
| `.embed-card-image` | Image container (external only) |
| `.embed-card-content` | Text content wrapper |
| `.embed-card-title` | Title element |
| `.embed-card-description` | Description element |
| `.embed-card-meta` | Metadata (date, domain) |

## Performance Considerations

1. **Caching** - External metadata is cached to avoid repeated HTTP requests
2. **Timeout** - Configurable HTTP timeout (default 10s) prevents slow builds
3. **Concurrent Processing** - Posts are processed concurrently
4. **Body Limit** - External pages are limited to 1MB to prevent memory issues
5. **Disable Fetching** - Set `fetch_external = false` to skip HTTP requests entirely

## Related Features

- **Wikilinks** (`[[slug]]`) - Simple internal links without preview cards
- **One-line Links** - Automatic link card for standalone URLs
- **Link Collector** - Tracks outlinks for backlink generation
