---
title: "Image Library"
description: "Build a searchable image inventory and authoring page with markata-go"
date: 2026-09-13
published: true
slug: /docs/guides/image-library/
tags:
  - images
  - configuration
  - authoring
---

# Image Library

markata-go can generate an image inventory and a page for finding images used
by your site. The default output is:

- `/images/` — a responsive, searchable image library
- `/images/index.json` — a versioned machine-readable inventory

The plugin runs by default. It reads public posts and the configured static
asset directory. It does not fetch remote images.

## Configure the output

Add an `images` table under `[markata-go]` when you need to change the
defaults:

```toml
[markata-go.images]
enabled = true
path = "images"
template = "images.html"
export_json = true
include_unreferenced = true
```

| Option | Default | Description |
|---|---|---|
| `enabled` | `true` | Generate the image library page and inventory. |
| `path` | `"images"` | Relative output directory below `output_dir`. |
| `template` | `"images.html"` | Template used for the HTML page. |
| `export_json` | `true` | Write `path/index.json`. |
| `include_unreferenced` | `true` | Include image files found in `assets_dir` even when no public post uses them. |

The output path must stay inside `output_dir`. The same options work in YAML
and JSON configuration files. Environment overrides use these names:

```text
MARKATA_GO_IMAGES_ENABLED
MARKATA_GO_IMAGES_PATH
MARKATA_GO_IMAGES_TEMPLATE
MARKATA_GO_IMAGES_EXPORT_JSON
MARKATA_GO_IMAGES_INCLUDE_UNREFERENCED
```

## Find and copy an image

The library searches image filenames, paths, alt text, and public post titles.
Use the `All`, `Used`, `Unused`, `Cover`, and `Recently added` views, or choose
a sort order from the toolbar.

Each card includes **Copy Markdown** and **Copy URL** actions. Both actions use
the canonical `src` value from the inventory. For example:

```markdown
![Workshop](https://dropper.wayl.one/file/workshop.webp)
```

Trusted Dropper URLs may use resized URLs for previews. Those presentation URLs
never replace the canonical URL in the JSON artifact or copied Markdown.

## Image discovery

The inventory includes:

- Markdown image syntax and supported raw HTML `<img>` elements
- `image`, `cover`, `cover_image`, `og_image`, `social_image`, `thumbnail`,
  `featured_image`, `hero_image`, `avatar`, and `author_image` frontmatter
- Image files below `assets_dir`

Skipped, draft, private, and unpublished posts do not create usage
relationships. Local files are mapped from the asset root to site-root URLs;
for example, `static/images/logo.png` becomes `/images/logo.png`.
An asset referenced only by private posts is omitted from the public inventory,
even when `include_unreferenced` is enabled. Keep private media outside
`assets_dir` when it must not be published by the static asset writer.

The JSON artifact is deterministic. Images are sorted by source URL and usage
links are sorted by post path and then href. A missing local dimension is
reported as `0`.

The writer checks for collisions with posts, feeds, and static files before it
writes. It does not overwrite an existing site file. When you disable the
library or change its path, cleanup removes only files previously recorded as
image-library outputs whose contents are unchanged. A modified or unowned
stale JSON file is preserved and reported as a collision. Output paths that
traverse symlinks below the configured output root are rejected for safety.
Platform-managed ancestors above that root are allowed.

## Custom templates

Copy `images.html` into your site's `templates/` directory to customize the
page. The template receives:

- `image_library` — page counts and presentation-ready image cards
- `image_index` — the canonical `imageindex.Index`
- `image_library_config` — the resolved image configuration

The server-rendered page remains useful when JavaScript is unavailable. The
bundled JavaScript only enhances filtering, sorting, usage expansion, and copy
feedback.

See the [configuration reference](/docs/guides/configuration/) and [built-in
plugin reference](/docs/reference/plugins/) for related settings.
