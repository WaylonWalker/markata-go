---
title: "@Mentions"
description: "Resolve @handles to links from blogroll feeds and internal posts"
date: 2025-01-26
published: true
tags:
  - documentation
  - mentions
  - social
---

# @Mentions

The mentions plugin transforms `@handle` syntax in your markdown content into HTML links. It resolves handles from two sources:

1. **Blogroll feeds** - External RSS/Atom feeds from your blogroll configuration
2. **Internal posts** - Posts in your site matching filter expressions (like contact pages)

## Basic Usage

Write mentions in your markdown using `@handle` syntax:

```markdown
I was reading @alice's post about static site generators.
Also check out @bob for more great content.
```

When handles are resolved, they become clickable links:

```html
I was reading <a href="/contact/alice/" class="mention">@alice</a>'s post...
```

## Configuration

Configure mentions in your `markata-go.toml`:

```toml
[markata-go.mentions]
enabled = true
css_class = "mention"  # CSS class for links (default: "mention")

# Source handles from internal posts
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"       # Frontmatter field for handle (optional, uses slug if not set)
aliases_field = "aliases"     # Frontmatter field for aliases (optional)
```

## Resolving from Internal Posts

The `from_posts` configuration lets you resolve `@handles` from posts in your site. This is useful for:

- **Contact pages** - Team member or author profiles
- **Contributor pages** - Guest authors or collaborators  
- **Partner pages** - Companies or organizations you work with

### Example: Contact Pages

Create contact pages with handle information in frontmatter:

```yaml
# pages/contact/alice-smith.md
---
title: "Alice Smith"
template: contact
tags:
  - contact
handle: alice
aliases:
  - alices
  - asmith
avatar: /images/alice.jpg
url: https://alicesmith.dev
description: "Software engineer specializing in Go and web development."
---

Alice is a software engineer specializing in Go and web development.
```

Configure the mentions plugin to use these posts:

```toml
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"
aliases_field = "aliases"
```

Now `@alice`, `@alices`, and `@asmith` all link to `/contact/alice-smith/` and display `@alice` as the link text.

The mention hovercard will show the avatar (from the `avatar`, `image`, or `icon` frontmatter field), the post title as the name, and the description as the bio. You can specify which field to use for the avatar with `avatar_field`:

```toml
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"
aliases_field = "aliases"
avatar_field = "icon"   # Use 'icon' field instead of default lookup order
```

### Multiple Sources

You can define multiple `from_posts` sources for different content types:

```toml
# Contact pages
[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "handle"
aliases_field = "aliases"

# Team members
[[markata-go.mentions.from_posts]]
filter = "template == 'team-member.html'"
handle_field = "github_handle"

# Projects (no explicit handle - uses slug)
[[markata-go.mentions.from_posts]]
filter = "'project' in tags"
```

### Fallback to Slug

If `handle_field` is not specified or the frontmatter field is empty, the post's slug is used as the handle.

```toml
[[markata-go.mentions.from_posts]]
filter = "'contributor' in tags"
# No handle_field - uses slug
```

A post at `contributors/jane-doe.md` becomes `@jane-doe`.

## Resolving from Blogroll

If you have a [blogroll](/guides/blogroll/) configured, handles are automatically extracted from your RSS/Atom feeds:

```toml
[markata-go.blogroll]
enabled = true

[[markata-go.blogroll.feeds]]
url = "https://daverupert.com/feed.xml"
title = "Dave Rupert"
handle = "daverupert"
site_url = "https://daverupert.com"
aliases = ["dave", "davatron"]
```

Now `@daverupert`, `@dave`, and `@davatron` all link to `https://daverupert.com`.

## Handle Resolution Priority

When building the handle map, sources are processed in this order:

1. **Blogroll feeds** (if enabled) - First registered wins
2. **From posts** sources - Processed in order they appear in config

If the same handle is registered multiple times, the first registration wins and subsequent duplicates are logged as warnings.

## Styling Mentions

Mention links have the configured CSS class (default: `mention`). Add styles to your CSS:

```css
a.mention {
  color: var(--color-primary);
  font-weight: 500;
}

a.mention::before {
  content: "";  /* Remove @ if you want */
}

a.mention:hover {
  text-decoration: underline;
}
```

## Code Block Protection

Mentions inside fenced code blocks are preserved and not transformed:

````markdown
Check out @alice's post.

```
// @alice is not transformed here
const handle = "@alice";
```
````

## Email Protection

Email addresses are not transformed as mentions. The plugin detects the `@` preceded by word characters and ignores them:

```markdown
Contact me at test@example.com  <!-- Not transformed -->
Follow @example on social       <!-- Transformed -->
```

## Configuration Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable mentions processing |
| `css_class` | string | `"mention"` | CSS class for mention links |
| `from_posts` | array | `[]` | List of internal post sources |

### from_posts Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `filter` | string | required | Filter expression to select posts |
| `handle_field` | string | (slug) | Frontmatter field for handle |
| `aliases_field` | string | - | Frontmatter field for aliases |
| `avatar_field` | string | - | Frontmatter field for avatar URL. If not set, checks `avatar`, `image`, `icon` in order |

## Examples

### Team Directory

```toml
[[markata-go.mentions.from_posts]]
filter = "'team' in tags and published == true"
handle_field = "slack_handle"
aliases_field = "nicknames"
```

### Author Profiles

```toml
[[markata-go.mentions.from_posts]]
filter = "template == 'author.html'"
handle_field = "username"
```

### Combined Sources

```toml
[markata-go.blogroll]
enabled = true

[[markata-go.blogroll.feeds]]
url = "https://external-blog.com/rss"
handle = "external"

[[markata-go.mentions.from_posts]]
filter = "'internal' in tags"
handle_field = "handle"
```

Now you can use `@external` for external links and `@internal-user` for internal profile links in the same post.
