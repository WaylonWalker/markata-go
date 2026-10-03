---
title: "Icon Shortcodes"
description: "Use Lucide and local SVG icon packs directly in Markdown"
published: true
slug: /docs/guides/icons/
tags:
  - documentation
  - markdown
  - icons
---

# Icon Shortcodes

markata-go renders Zensical-style icon shortcodes directly in Markdown.

Lucide is ready by default. You can write:

```markdown
A friendly icon :lucide-smile: inline with text.
```

or the slash alias:

```markdown
A friendly icon :lucide/smile: inline with text.
```

No icon directory or configuration is required. The first unresolved Lucide shortcode vendors a pinned `lucide-static` pack through markata-go's existing icon-vendor cache, indexes the resulting local SVGs, and renders the requested icon. Builds that never use a Lucide shortcode do not download the pack.

The default Lucide version is pinned by markata-go rather than following an upstream `latest` tag. Once vendored, later builds reuse the local materialization/cache and do not need to fetch it again.

## Local icons and overrides

Project-local SVG packs still work and take precedence over automatically vendored Lucide icons. Put SVG files under `.icons/` or `static/.icons/`.

For this asset:

```text
static/.icons/lucide/smile.svg
```

both `:lucide-smile:` and `:lucide/smile:` use your local SVG.

Unknown names are left unchanged, so normal emoji shortcodes such as `:smile:` can still be handled by the Markdown emoji extension.

## Configuration

Icon rendering and the default Lucide fallback are enabled by default:

```toml
[icons]
enabled = true
auto_vendor = true
paths = [".icons", "static/.icons"]
```

To disable the automatic Lucide fallback while continuing to use local icons:

```toml
[icons]
auto_vendor = false
```

To use only selected packs:

```toml
[icons]
packs = ["lucide", "simple-icons"]
```

If `packs` is set and does not include `lucide`, the automatic Lucide fallback is not used.

To use one custom icon root:

```toml
[icons]
path = "assets/icons"
```

To disable icon shortcodes entirely:

```toml
[icons]
enabled = false
```

For explicit pinning, additional icon packs, custom sources, or strict build-time vendoring, see [Vendoring Icon Packs](/docs/guides/icon-vendoring/).

## Offline builds

The automatic Lucide fallback reuses markata-go's icon vendor cache. If the pack has already been cached or materialized, it works offline.

With `MARKATA_GO_OFFLINE=1`, markata-go never downloads a missing pack. If a Lucide shortcode cannot be resolved from local or cached assets, the shortcode is left unchanged instead of failing the whole site build.

## Naming

The path below an icon root becomes the logical icon name. Slashes are replaced with hyphens for the canonical Markdown alias:

| SVG file | Canonical shortcode | Slash alias |
|---|---|---|
| `lucide/smile.svg` | `:lucide-smile:` | `:lucide/smile:` |
| `simple-icons/github.svg` | `:simple-icons-github:` | `:simple-icons/github:` |
| `custom/arrows/left.svg` | `:custom-arrows-left:` | `:custom/arrows/left:` |

If two paths flatten to the same hyphenated alias, markata-go leaves that ambiguous alias unresolved. The explicit slash spellings continue to work.

## Code examples stay literal

Icon shortcodes are not expanded or vendored inside inline code or fenced code blocks:

````markdown
This renders: :lucide-smile:

This stays literal: `:lucide-smile:`

```markdown
:lucide-smile:
```
````

## SVG behavior

Icons are inlined into the generated HTML, so rendered pages do not need browser-time requests for SVG files. Fixed source `width` and `height` values are normalized to `1em` so icons follow surrounding text size.

Because inline SVG becomes part of the page, markata-go rejects SVGs containing scripts, `foreignObject`, JavaScript URLs, or inline event handlers. Local and vendored SVGs go through the same safety check.

Shortcodes are treated as decorative (`aria-hidden`). When an icon communicates meaning, include readable text next to it rather than relying on the glyph alone.
