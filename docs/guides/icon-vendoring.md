---
title: "Vendoring Icon Packs"
description: "Download pinned SVG icon packs for local icon shortcodes"
published: true
slug: /docs/guides/icon-vendoring/
tags:
  - documentation
  - markdown
  - icons
  - assets
---

# Vendoring Icon Packs

The icon shortcode feature works entirely from local SVG files. If you do not want to copy those files into your site by hand, markata-go can optionally download a pinned icon-pack archive during the build and populate `static/.icons/` for you.

Vendoring is disabled by default. Enabling icon shortcodes alone does **not** add network access to your build. Even after vendoring is enabled, each downloaded pack must be explicitly listed with a pinned version.

## Example: Lucide from npm

```toml
[icons.vendor]
enabled = true

[[icons.vendor.packs]]
name = "lucide"
version = "0.468.0"
source = "npm"
package = "lucide-static"
icons_path = "icons"
license_path = "LICENSE"
```

On the first build, markata-go downloads and caches the pinned npm tarball, copies its SVGs to `static/.icons/lucide/`, and writes the pack license to `static/.icons/licenses/lucide.txt`.

You can then use the local icons normally:

```markdown
:lucide-smile: Hello!

:lucide/github: Source code
```

The first form follows Zensical's hyphenated shortcode spelling. The slash form is a markata-go alias for the same local asset.

## Pin versions

Every vendored pack requires an explicit version. This keeps the cache key and generated icon tree deterministic rather than silently following `latest`.

When you want to upgrade a pack, change its configured version and rebuild. markata-go replaces that pack's materialized icon directory so icons removed upstream do not remain as stale files.

## Custom cache and target

```toml
[icons.vendor]
enabled = true
cache_dir = ".cache/markata-icons"
target = "static/.icons"
```

Defaults are:

- cache: `.markata/cache/icon-packs`
- target: `static/.icons`

The default target is already one of the directories scanned by the icon shortcode plugin.

## Pack-specific archive layout

Different packages arrange SVGs and licenses differently. Configure those archive-relative paths per pack:

```toml
[[icons.vendor.packs]]
name = "my-pack"
version = "2.4.0"
source = "npm"
package = "my-icon-package"
archive_path = "package"
icons_path = "dist/icons"
license_path = "LICENSE.md"
```

`archive_path` defaults to `package`, which matches standard npm tarballs. `icons_path` defaults to `icons`, and `license_path` defaults to `LICENSE`.

## Direct `.tgz` archives

You can also use a versioned HTTP or HTTPS tarball directly:

```toml
[[icons.vendor.packs]]
name = "house-icons"
version = "2026.09.1"
source = "url"
url = "https://example.com/house-icons-2026.09.1.tgz"
archive_path = "package"
icons_path = "svg"
license_path = "LICENSE.txt"
```

Direct sources use the same archive cache and extraction safety checks as other markata-go vendored assets.

## Offline builds

The vendor plugin reuses markata-go's asset cache. After a successful online build, deleting `static/.icons/` and rebuilding can recreate the files from the cache without downloading the archive again.

To prohibit runtime network access entirely:

```bash
MARKATA_GO_OFFLINE=1 markata-go build
```

In offline mode, a pack works when its archive is already cached or its previously materialized icon directory is still present. If neither is available, the build fails instead of making a network request.

## Licenses

Each configured pack must provide the file named by `license_path`. markata-go copies it to:

```text
static/.icons/licenses/<pack-name>.txt
```

That preserves the upstream license alongside the files being redistributed. You are still responsible for confirming that a pack's license is appropriate for your site.

## SVG safety

Downloaded SVGs are checked before publication. markata-go rejects a pack if an SVG contains common active-content constructs such as scripts, `foreignObject`, JavaScript URLs, or inline event handlers.

Only `.svg` files under the configured icon directory are copied; symlinks are ignored.

## Multiple packs

Add more `[[icons.vendor.packs]]` blocks as needed. Pack names must be unique because the name becomes the first directory component and shortcode namespace.

```toml
[icons.vendor]
enabled = true

[[icons.vendor.packs]]
name = "lucide"
version = "0.468.0"
source = "npm"
package = "lucide-static"

[[icons.vendor.packs]]
name = "brand"
version = "1.3.0"
source = "url"
url = "https://example.com/brand-icons-1.3.0.tgz"
icons_path = "svg"
license_path = "LICENSE.txt"
```

Because vendoring only populates local files, the shortcode renderer remains deterministic and does not perform browser-time requests to an icon service or CDN.
