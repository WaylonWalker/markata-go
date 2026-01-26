# Mentions Plugin Specification

The mentions plugin transforms `@handle` syntax in markdown content into HTML links, resolving handles from blogroll feeds and internal posts.

## Overview

The mentions plugin processes `@handle` patterns in post content and replaces them with HTML anchor links. Handles are resolved from two sources:

1. **Blogroll feeds** - External RSS/Atom feed entries with handles
2. **Internal posts** - Posts matching filter expressions (e.g., contact pages, team pages)

## Syntax

```markdown
@handle           # Basic mention
@alice-smith      # Hyphenated handle
@alice123         # Handle with numbers
```

Handle pattern: `@[a-zA-Z][a-zA-Z0-9_-]*`

## Configuration

```toml
[markata-go.mentions]
enabled = true                    # Enable/disable the plugin (default: true)
css_class = "mention"             # CSS class for links (default: "mention")

# Source handles from internal posts
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"      # Filter expression to select posts
handle_field = "handle"           # Frontmatter field for handle (optional, uses slug if not set)
aliases_field = "aliases"         # Frontmatter field for aliases (optional)
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable mentions processing |
| `css_class` | string | `"mention"` | CSS class applied to mention links |
| `from_posts` | array | `[]` | List of internal post sources |

### from_posts Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `filter` | string | Yes | Filter expression to select posts |
| `handle_field` | string | No | Frontmatter field containing the handle (defaults to post slug) |
| `aliases_field` | string | No | Frontmatter field containing handle aliases |

## Handle Resolution

### Resolution Order

Handles are registered in this order (first registration wins):

1. **Blogroll feeds** - If blogroll is enabled, handles from feeds are registered first
2. **from_posts sources** - Processed in the order they appear in configuration

If a handle or alias is already registered, subsequent registrations are ignored with a warning.

### From Blogroll

When blogroll is enabled, handles are extracted from feed configurations:

```toml
[[markata-go.blogroll.feeds]]
url = "https://example.com/feed.xml"
title = "Example Blog"
handle = "example"                  # Primary handle
site_url = "https://example.com"    # Link target
aliases = ["ex", "exblog"]          # Additional handles
```

The `site_url` becomes the link target for `@example`, `@ex`, and `@exblog`.

### From Internal Posts

Internal posts are filtered using the filter expression, then handles are extracted:

1. **Handle extraction**: Uses `handle_field` frontmatter value, or falls back to post slug
2. **Alias extraction**: Uses `aliases_field` frontmatter value if specified
3. **Link target**: The post's permalink URL

Example post frontmatter:

```yaml
---
title: "Alice Smith"
slug: alice-smith
tags:
  - contact
handle: alice
aliases:
  - alices
  - asmith
---
```

With config:

```toml
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"
aliases_field = "aliases"
```

This registers:
- `@alice` → `/contact/alice-smith/`
- `@alices` → `/contact/alice-smith/`
- `@asmith` → `/contact/alice-smith/`

### Slug Fallback

If `handle_field` is empty or the specified field is not present in frontmatter, the post's slug is used:

```toml
[[markata-go.mentions.from_posts]]
filter = "'project' in tags"
# No handle_field - uses slug
```

A post with `slug: my-project` registers `@my-project`.

## Transform Behavior

### Processing

1. Extract code blocks and store them (to protect from transformation)
2. Find all `@handle` patterns in content
3. For each handle:
   - Look up in handle map
   - If found: replace with anchor link
   - If not found: leave as-is
4. Restore code blocks

### Email Protection

Email addresses are not transformed. The regex matches `@` only when:
- NOT preceded by word characters (letters, digits, underscore)

```markdown
test@example.com     <!-- Not transformed (email) -->
Follow @example      <!-- Transformed -->
Say hello@example    <!-- Not transformed (no space before @) -->
```

### Code Block Protection

Mentions inside fenced code blocks are preserved:

````markdown
Check out @alice's post.

```
// @alice is not transformed here
const handle = "@alice";
```
````

### Generated HTML

```html
<a href="/contact/alice/" class="mention">@alice</a>
```

Attributes:
- `href` - Link target URL
- `class` - Configured CSS class (default: "mention")

The `@` symbol is included in the link text.

## Lifecycle Stage

The mentions plugin runs in the **Transform** stage with default priority (0).

## Data Model

### MentionsConfig

```go
type MentionsConfig struct {
    Enabled   *bool               // Enable/disable (default: true)
    CSSClass  string              // CSS class (default: "mention")
    FromPosts []MentionPostSource // Internal post sources
}
```

### MentionPostSource

```go
type MentionPostSource struct {
    Filter       string // Filter expression (required)
    HandleField  string // Frontmatter field for handle (optional)
    AliasesField string // Frontmatter field for aliases (optional)
}
```

### Internal Handle Map

```go
// handleMap maps handles to URLs
type handleMap map[string]string

// Example:
// "alice" → "/contact/alice/"
// "alices" → "/contact/alice/"
// "daverupert" → "https://daverupert.com"
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Unknown handle | Left as plain text (`@unknown`) |
| Duplicate handle registration | Warning logged, first registration wins |
| Invalid filter expression | Error logged, source skipped |
| Missing handle_field in frontmatter | Falls back to slug |
| Empty aliases_field | No aliases registered for that post |

## CSS Classes

| Class | Purpose |
|-------|---------|
| `.mention` | Default class for all mention links |

Custom class can be configured via `css_class` option.

## Examples

### Team Directory

```toml
[[markata-go.mentions.from_posts]]
filter = "'team' in tags and published == true"
handle_field = "slack_handle"
aliases_field = "nicknames"
```

### Multiple Sources

```toml
# External blogroll
[markata-go.blogroll]
enabled = true

[[markata-go.blogroll.feeds]]
url = "https://external.com/feed.xml"
handle = "external"
site_url = "https://external.com"

# Internal contacts
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"
aliases_field = "aliases"

# Internal projects
[[markata-go.mentions.from_posts]]
filter = "'project' in tags"
# Uses slug as handle
```

### Minimal Configuration

```toml
# Just enable with defaults - uses blogroll handles only
[markata-go.mentions]
enabled = true
```

## Related Features

- **Blogroll** - Source of external handles
- **Wikilinks** (`[[slug]]`) - Internal links by slug
- **Filter Expressions** - Query language for selecting posts
