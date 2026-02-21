# YouTube Plugin Specification

This document specifies the YouTube embed plugin for markata-go.

## Overview

The YouTube plugin automatically converts YouTube URLs in markdown content into responsive embedded iframes. It processes URLs that appear on their own line (in a `<p>` tag) and replaces them with privacy-enhanced video embeds. This plugin is separate from the embeds plugin; oEmbed-based YouTube embeds use Lite YouTube by default.

## Plugin Information

| Property | Value |
|----------|-------|
| **Name** | `youtube` |
| **Stage** | Render (with late priority) |
| **Interfaces** | `Plugin`, `ConfigurePlugin`, `RenderPlugin`, `PriorityPlugin` |
| **Default Enabled** | Yes |

## Configuration

```toml
[markata-go.youtube]
enabled = true              # Enable/disable the plugin (default: true)
privacy_enhanced = true     # Use youtube-nocookie.com (default: true)
container_class = "youtube-embed"  # CSS class for container div (default)
lazy_load = true            # Enable lazy loading of iframe (default: true)
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable YouTube URL conversion |
| `privacy_enhanced` | bool | `true` | Use `youtube-nocookie.com` domain for GDPR compliance |
| `container_class` | string | `"youtube-embed"` | CSS class applied to the container div |
| `lazy_load` | bool | `true` | Add `loading="lazy"` attribute to iframe |

## Supported URL Formats

The plugin recognizes the following YouTube URL patterns:

| Format | Example |
|--------|---------|
| Standard watch URL | `https://www.youtube.com/watch?v=dQw4w9WgXcQ` |
| Without www | `https://youtube.com/watch?v=dQw4w9WgXcQ` |
| Mobile URL | `https://m.youtube.com/watch?v=dQw4w9WgXcQ` |
| Short URL | `https://youtu.be/dQw4w9WgXcQ` |
| With timestamp | `https://youtu.be/dQw4w9WgXcQ?t=1h2m3s` |
| With start param | `https://www.youtube.com/watch?v=dQw4w9WgXcQ&start=123` |

### Video ID Requirements

- Must be exactly 11 characters
- Alphanumeric plus hyphen (`-`) and underscore (`_`)
- Invalid IDs are ignored (URL left unchanged)

## Timestamp Support

The plugin extracts and converts timestamps to the embed `start` parameter:

| Input Format | Converted To |
|--------------|--------------|
| `?t=123` | `?start=123` |
| `?t=1h2m3s` | `?start=3723` |
| `?t=2m30s` | `?start=150` |
| `?t=45s` | `?start=45` |
| `?start=123` | `?start=123` (preserved) |

## Behavior

### URL Detection

1. **Standalone URLs only**: URLs must be on their own line, resulting in a `<p>` tag containing only the URL
2. **Handles autolinked URLs**: Works with both plain URLs and URLs converted to `<a>` tags by the Linkify extension
3. **Preserves inline URLs**: URLs within text are not converted

### Detection Patterns

The plugin matches URLs in two formats after markdown rendering:

```html
<!-- Pattern 1: Plain URL (no Linkify) -->
<p>https://www.youtube.com/watch?v=VIDEO_ID</p>

<!-- Pattern 2: Autolinked URL (with Linkify extension) -->
<p><a href="https://www.youtube.com/watch?v=VIDEO_ID">https://www.youtube.com/watch?v=VIDEO_ID</a></p>
```

### Code Block Protection

URLs inside `<code>` blocks are never converted, allowing documentation of YouTube URLs without embedding:

```markdown
To embed a video, use: `https://youtu.be/VIDEO_ID`
```

## Output

### HTML Structure

```html
<div class="youtube-embed">
  <iframe
    src="https://www.youtube-nocookie.com/embed/VIDEO_ID"
    title="YouTube video player"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen
    loading="lazy">
  </iframe>
</div>
```

## Embeds Plugin Note

When using the embeds plugin with oEmbed (for `![embed](https://youtube.com/...)`), YouTube rich embeds render with Lite YouTube by default instead of the raw oEmbed iframe HTML.

### With Timestamp

```html
<div class="youtube-embed">
  <iframe
    src="https://www.youtube-nocookie.com/embed/VIDEO_ID?start=3723"
    ...>
  </iframe>
</div>
```

### Standard Mode (privacy_enhanced: false)

```html
<div class="youtube-embed">
  <iframe
    src="https://www.youtube.com/embed/VIDEO_ID"
    ...>
  </iframe>
</div>
```

## CSS Styling

The default theme includes responsive styling for YouTube embeds:

```css
.youtube-embed {
  position: relative;
  padding-bottom: 56.25%; /* 16:9 aspect ratio */
  height: 0;
  overflow: hidden;
  max-width: 100%;
  margin: 1.5rem 0;
  border-radius: 0.5rem;
}

.youtube-embed iframe {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border: none;
  border-radius: 0.5rem;
}
```

## Privacy Mode

When `privacy_enhanced` is enabled (default), the plugin uses `youtube-nocookie.com`:

- **No cookies until play**: YouTube does not store cookies until the user clicks play
- **GDPR compliance**: Helps meet privacy regulations
- **Same functionality**: Video playback works identically

To use standard YouTube embeds:

```toml
[markata-go.youtube]
privacy_enhanced = false
```

## Processing Order

1. Plugin runs during the **Render** stage with **late priority**
2. Executes after `render_markdown` (which converts markdown to HTML)
3. Processes `post.ArticleHTML` for each post
4. Uses concurrent processing for performance

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Invalid video ID | URL left unchanged |
| URL in code block | URL left unchanged |
| Inline URL (not standalone) | URL left unchanged |
| Malformed URL | URL left unchanged |
| Empty ArticleHTML | Skipped |
| Skipped post | Skipped |

## Interface Compliance

```go
var (
    _ lifecycle.Plugin          = (*YouTubePlugin)(nil)
    _ lifecycle.ConfigurePlugin = (*YouTubePlugin)(nil)
    _ lifecycle.RenderPlugin    = (*YouTubePlugin)(nil)
    _ lifecycle.PriorityPlugin  = (*YouTubePlugin)(nil)
)
```

## Examples

### Basic Usage

```markdown
Check out this video:

https://www.youtube.com/watch?v=dQw4w9WgXcQ

More content below...
```

### With Timestamp

```markdown
Skip to the good part:

https://youtu.be/dQw4w9WgXcQ?t=1m30s
```

### Short URL

```markdown
https://youtu.be/dQw4w9WgXcQ
```

### NOT Converted (inline)

```markdown
Check out https://youtu.be/dQw4w9WgXcQ for more info.
```

### NOT Converted (code block)

```markdown
To embed: `https://youtu.be/VIDEO_ID`
```

## See Also

- [Configuration Reference](CONFIG.md) - Global configuration options
- [Lifecycle Stages](LIFECYCLE.md) - Plugin execution order
- [Default Plugins](DEFAULT_PLUGINS.md) - All built-in plugins
