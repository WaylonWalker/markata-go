---
title: "Icon Shortcodes"
description: "Use Zensical-style local SVG icon packs in Markdown"
published: true
slug: /docs/guides/icons/
tags:
  - documentation
  - markdown
  - icons
---

# Icon Shortcodes

markata-go can render local SVG icon packs directly in Markdown. Put SVG files under `.icons/` or `static/.icons/`, then reference them with Zensical-style shortcodes.

For this asset:

```text
static/.icons/lucide/smile.svg
```

use the canonical Zensical Markdown spelling:

```markdown
A friendly icon :lucide-smile: inline with text.
```

markata-go also supports the slash spelling from issue #895:

```markdown
A friendly icon :lucide/smile: inline with text.
```

Both forms resolve to the same SVG. Unknown names are left unchanged, so normal emoji shortcodes such as `:smile:` can still be handled by the Markdown emoji extension.

## Configuration

Icon rendering is enabled by default and is a no-op when no icon directories exist.

```toml
[icons]
enabled = true
paths = [".icons", "static/.icons"]
```

To use only selected packs:

```toml
[icons]
packs = ["lucide", "simple-icons"]
```

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

## Naming

The path below the icon root becomes the logical icon name. Slashes are replaced with hyphens for the canonical Markdown alias:

| SVG file | Canonical shortcode | Slash alias |
|---|---|---|
| `lucide/smile.svg` | `:lucide-smile:` | `:lucide/smile:` |
| `simple-icons/github.svg` | `:simple-icons-github:` | `:simple-icons/github:` |
| `custom/arrows/left.svg` | `:custom-arrows-left:` | `:custom/arrows/left:` |

If two paths flatten to the same hyphenated alias, markata-go leaves that ambiguous alias unresolved. The explicit slash spellings continue to work.

## Code examples stay literal

Icon shortcodes are not expanded inside inline code or fenced code blocks:

````markdown
This renders: :lucide-smile:

This stays literal: `:lucide-smile:`

```markdown
:lucide-smile:
```
````

## SVG behavior

Icons are inlined into the generated HTML, so builds do not need a browser-time request for the SVG. Fixed source `width` and `height` values are normalized to `1em` so icons follow surrounding text size.

Because inline SVG becomes part of the page, markata-go rejects SVGs containing scripts, `foreignObject`, JavaScript URLs, or inline event handlers. Icon assets should be ordinary static SVG files from a trusted pack.

Shortcodes are treated as decorative (`aria-hidden`). When an icon communicates meaning, include readable text next to it rather than relying on the glyph alone.

## Vendored icon packs

This feature only resolves local SVGs; it does not download packs during rendering. Automatic download/vendor support is tracked separately in issue #898. Keeping those responsibilities separate makes normal Markdown rendering deterministic and offline-friendly.
