# Link Avatars Specification

This document specifies the built-in `link_avatars` plugin.

## Goal

Add small favicon/avatar icons next to external links to improve visual identification of link destinations. The feature is implemented entirely client-side for zero build-time overhead.

## Lifecycle

- **Stage:** `configure` (reads config), `write` (generates assets, injects head tags)
- **Determinism:** Build output is deterministic; favicon loading happens at runtime in the browser.

## Configuration

Configuration is namespaced under the top-level `markata-go` section.

```toml
[markata-go.link_avatars]
enabled = true

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
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable the plugin |
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

### Service Templates

| Service | URL Template |
|---------|--------------|
| `duckduckgo` | `https://icons.duckduckgo.com/ip3/{host}.ico` |
| `google` | `https://www.google.com/s2/favicons?domain={host}&sz={size}` |
| `custom` | User-provided template |

## Behavior

1. **Link Selection**: The JavaScript finds all links matching the `selector` (default: external links starting with `http`).

2. **Same-Origin Skip**: Links pointing to the same origin as the current page are automatically skipped.

3. **Ignore Rules Applied**: Links are filtered out based on:
   - Domain matches `ignore_domains`
   - Origin matches `ignore_origins`
   - Link matches any `ignore_selectors`
   - Link has any class in `ignore_classes`
   - Link is inside an element with ID in `ignore_ids`

4. **Avatar Injection**: For qualifying links:
   - A `data-favicon` attribute is set with the favicon URL
   - A CSS custom property `--favicon-url` is set for styling
   - The link gets a `has-avatar` class for CSS targeting

5. **CSS Styling**: The generated CSS uses `::before` or `::after` pseudo-elements to display the favicon using `background-image`.

## Generated Output

When enabled, the plugin generates:

- `{output_dir}/assets/markata/link-avatars.js` - Client-side JavaScript
- `{output_dir}/assets/markata/link-avatars.css` - Minimal CSS styles

And injects into the HTML `<head>`:

```html
<link rel="stylesheet" href="/assets/markata/link-avatars.css">
<script src="/assets/markata/link-avatars.js" defer></script>
```

### JavaScript Behavior

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
- Network errors for favicons don't break the page.

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
