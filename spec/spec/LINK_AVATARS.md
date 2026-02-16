# Link Avatars Specification

This document specifies the built-in `link_avatars` plugin.

## Goal

Add small favicon/avatar icons next to external links to improve visual identification of link destinations. The feature supports both client-side and build-time modes.

## Lifecycle

- **Stage:** `configure` (reads config, injects head tags), `render` (build-time injection), `write` (generates assets)
- **Determinism:** Build output is deterministic; build-time modes cache icons and reuse them across builds.

## Configuration

Configuration is namespaced under the top-level `markata-go` section.

```toml
[markata-go.link_avatars]
enabled = true

# Mode: "js" (default), "local", or "hosted"
# js: client-side enhancement with runtime favicon fetch
# local: build-time caching, HTML points to site-relative icons
# hosted: build-time caching, HTML points to hosted_base_url
mode = "js"

# CSS selector for links to enhance (default: "a[href^='http']")
selector = "a[href^='http']"

# Avatar service provider (default: "duckduckgo")
# Options: "duckduckgo", "google", "custom"
service = "duckduckgo"

# Custom template URL (only used when service = "custom")
# Supports placeholders: {origin}, {host}
# template = "https://my-service.com/favicon?domain={host}"

# Domains to exclude from avatar display
ignore_domains = ["example.com", "localhost"]

# Full origins to exclude (includes protocol)
ignore_origins = ["https://example.com"]

# CSS selectors to exclude from processing
ignore_selectors = [".no-avatars", "#nav", "a.no-avatar"]

# CSS classes to exclude (links with these classes are skipped)
ignore_classes = ["no-avatar", "internal-link"]

# Element IDs to exclude (links inside elements with these IDs are skipped)
ignore_ids = ["site-nav", "footer-links"]

# Avatar size in pixels (default: 16)
size = 16

# Position of the avatar relative to link text (default: "before")
# Options: "before", "after"
position = "before"

# Hosted base URL for mode = "hosted"
# hosted_base_url = "https://cdn.example.com/markata/link-avatars"
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable the plugin |
| `mode` | string | `"js"` | Avatar mode: "js", "local", or "hosted" |
| `selector` | string | `"a[href^='http']"` | CSS selector for links to enhance |
| `service` | string | `"duckduckgo"` | Avatar service: "duckduckgo", "google", "custom" |
| `template` | string | `""` | Custom URL template (when service="custom") |
| `ignore_domains` | []string | `[]` | Domains to skip (e.g., `["example.com"]`) |
| `ignore_origins` | []string | `[]` | Full origins to skip (e.g., `["https://example.com"]`) |
| `ignore_selectors` | []string | `[]` | CSS selectors to skip |
| `ignore_classes` | []string | `[]` | Classes to skip |
| `ignore_ids` | []string | `[]` | Element IDs to skip |
| `size` | int | `16` | Avatar icon size in pixels |
| `position` | string | `"before"` | Position: "before" or "after" link text |
| `hosted_base_url` | string | `""` | Base URL for hosted mode assets |

### Service Templates

| Service | URL Template |
|---------|--------------|
| `duckduckgo` | `https://icons.duckduckgo.com/ip3/{host}.ico` |
| `google` | `https://www.google.com/s2/favicons?domain={host}&sz={size}` |
| `custom` | User-provided template |

## Behavior

1. **Link Selection**: Links matching the `selector` (default: external links starting with `http`).

2. **Same-Origin Skip**: Links pointing to the same origin as the configured site URL are skipped. If no site URL is configured, only the explicit ignore rules apply.

3. **Ignore Rules Applied**: Links are filtered out based on:
   - Domain matches `ignore_domains`
   - Origin matches `ignore_origins`
   - Link matches any `ignore_selectors`
   - Link has any class in `ignore_classes`
   - Link is inside an element with ID in `ignore_ids`
   - Link contains an `img` or `picture` element

4. **Avatar Injection**:
   - **js mode**: client-side JavaScript sets `data-favicon`, `--favicon-url`, and `has-avatar` classes at runtime.
   - **local/hosted mode**: build-time HTML injection sets `data-favicon`, `--favicon-url`, and `has-avatar` classes.

5. **CSS Styling**: The generated CSS uses `::before` or `::after` pseudo-elements to display the favicon using `background-image`.

6. **Build-Time Caching** (local/hosted): Favicons are downloaded once per host, stored under `assets/markata/link-avatars/`, and reused on subsequent builds.

## Generated Output

When enabled, the plugin generates:

- `{output_dir}/css/link-avatars.css` - Minimal CSS styles
- `{output_dir}/js/link-avatars.js` - Client-side JavaScript (js mode only)
- `{output_dir}/assets/markata/link-avatars/{host}.ico` - Cached icons (local/hosted modes)

And injects into the HTML `<head>` (js mode):

```html
<link rel="stylesheet" href="/css/link-avatars.css">
<script src="/js/link-avatars.js" defer></script>
```

For local/hosted modes, only the CSS link is injected.

### JavaScript Behavior (js mode)

The JavaScript:
- Runs on `DOMContentLoaded`
- Finds links matching the selector
- Filters based on ignore rules
- Sets `data-favicon` attribute and `--favicon-url` CSS variable
- Adds `has-avatar` class
- Handles lazy loading via Intersection Observer for performance

### CSS Styling

The CSS:
- Uses `::before` or `::after` pseudo-element
- Sets `background-image: var(--favicon-url)`
- Applies appropriate sizing and spacing
- Includes fallback for failed favicon loads (hide the pseudo-element)

## Error Handling

- If the plugin is disabled (default), it does nothing.
- If output directories cannot be created or files cannot be written, the plugin returns an error.
- Invalid favicon URLs gracefully fail (favicon simply doesn't display).
- Network errors for favicon fetches do not stop the build; affected links render without avatars.

## Example Usage

### Minimal Configuration

```toml
[markata-go.link_avatars]
enabled = true
```

### Customized Configuration

```toml
[markata-go.link_avatars]
enabled = true
service = "google"
size = 14
position = "after"
ignore_domains = ["localhost", "127.0.0.1"]
ignore_classes = ["no-favicon", "plain-link"]
ignore_selectors = ["nav a", ".footer a"]
```

### Build-Time Local Mode

```toml
[markata-go.link_avatars]
enabled = true
mode = "local"
service = "duckduckgo"
```

### Build-Time Hosted Mode

```toml
[markata-go.link_avatars]
enabled = true
mode = "hosted"
hosted_base_url = "https://cdn.example.com/markata/link-avatars"
```

### Custom Service

```toml
[markata-go.link_avatars]
enabled = true
service = "custom"
template = "https://favicon.splitbee.io/?url={origin}"
```

## CSS Customization

Users can override the default styles:

```css
/* Custom avatar styling */
a.has-avatar::before {
  margin-right: 0.5em;
  opacity: 0.8;
  border-radius: 2px;
}

/* Hide avatars in specific contexts */
.prose a.has-avatar::before {
  display: none;
}
```
