# Icon Pack Vendoring Specification

The `icon_vendor` plugin downloads pinned icon-pack archives at build time and materializes their SVG files into a local icon root consumed by the `icons` shortcode plugin.

Explicit `[icons.vendor]` configuration is opt-in and strict. Separately, the `icons` plugin MAY invoke the same vendoring machinery on demand for its built-in pinned Lucide fallback. That automatic path is specified in `ICONS.md` and does not change the explicit plugin's error semantics.

## Goals

- Make Zensical-style icon shortcodes practical without manually copying large icon packs.
- Require explicit, pinned pack versions for reproducible builds.
- Reuse markata-go's existing asset downloader and archive cache.
- Preserve pack license text alongside vendored assets.
- Work without network access when the required archive is already cached or the vendored target already exists.
- Validate SVGs before publishing downloaded content into the site tree.
- Provide one vendoring implementation reusable by both explicit configuration and built-in icon fallbacks.

## Configuration

Explicit vendoring is configured below the existing `[icons]` namespace:

```toml
[icons.vendor]
enabled = true
cache_dir = ".markata/cache/icon-packs"
target = "static/.icons"

[[icons.vendor.packs]]
name = "lucide"
version = "1.48.0"
source = "npm"
package = "lucide-static"
archive_path = "package"
icons_path = "icons"
license_path = "LICENSE"
```

`enabled` defaults to `false`. `cache_dir` defaults to `.markata/cache/icon-packs`. `target` defaults to `static/.icons`, which is one of the default roots indexed by the `icons` plugin.

Each explicitly configured pack must have a unique `name` and a pinned `version`.

The `enabled = false` default refers only to eager Configure-stage vendoring. It does not disable the `icons` plugin's built-in on-demand Lucide fallback. Set `[icons] auto_vendor = false` to disable that fallback.

## Pack Sources

### npm

For `source = "npm"`, `package` is required. markata-go fetches the pinned npm package tarball from the npm registry using the package name and version.

Example:

```toml
[[icons.vendor.packs]]
name = "lucide"
version = "1.48.0"
source = "npm"
package = "lucide-static"
icons_path = "icons"
license_path = "LICENSE"
```

### Direct archive URL

For `source = "url"`, `url` is required and must be HTTP or HTTPS. The URL must point to a gzip-compressed tar archive compatible with markata-go's asset downloader.

```toml
[[icons.vendor.packs]]
name = "custom-icons"
version = "2026.09.1"
source = "url"
url = "https://example.com/custom-icons-2026.09.1.tgz"
archive_path = "package"
icons_path = "svg"
license_path = "LICENSE.txt"
```

The version remains mandatory for direct URLs so cache and publication state are explicit even when the URL itself is versioned.

## Archive Paths

Pack paths are interpreted relative to the extracted archive root:

| Field | Default | Meaning |
|---|---|---|
| `archive_path` | `package` | Prefix inside the `.tgz` to extract. |
| `icons_path` | `icons` | Directory under the extracted archive containing SVGs. |
| `license_path` | `LICENSE` | License file under the extracted archive. |

Absolute paths and parent-directory traversal are rejected.

## Materialized Layout

For a pack named `lucide`, files are written below the configured target:

```text
static/.icons/
├── lucide/
│   ├── activity.svg
│   └── smile.svg
└── licenses/
    └── lucide.txt
```

The `icons` plugin then exposes these files through shortcodes such as `:lucide-smile:` and `:lucide/smile:`.

A newly materialized pack replaces the previous directory for that pack so removed upstream icons do not linger across version changes.

## Caching

Downloaded archives are stored in `cache_dir` using the existing markata-go asset downloader. The downloader's archive marker is used to identify a complete cached extraction.

The vendor plugin also stores a publication fingerprint containing the pack name, version, source, URL/package, and archive path settings. When the fingerprint matches and the target still contains SVGs, materialization is skipped.

If the materialized target is removed but the archive remains cached, markata-go recreates the target without another network request.

The built-in Lucide fallback MUST reuse this cache and publication mechanism rather than maintaining a second icon downloader/cache format.

## Offline Behavior

`MARKATA_GO_OFFLINE=1` disables network downloads through the shared asset downloader.

An explicitly configured pack can still be used offline when either:

1. its matching archive is available in the asset cache, or
2. its already-materialized target directory contains SVGs.

If neither is available, explicit vendoring fails instead of silently fetching from the network or producing an incomplete pack.

The built-in automatic Lucide fallback intentionally handles that error differently: it leaves the unresolved shortcode literal and allows the rest of the site build to continue. This prevents merely mentioning a new icon from breaking an otherwise valid offline build.

## SVG Safety

Only `.svg` files under `icons_path` are materialized. Symbolic links are ignored.

Before publication, every SVG is passed through the same validation used by the icon shortcode renderer. Assets containing active-content primitives such as scripts, `foreignObject`, JavaScript URLs, or inline event handlers are rejected and the pack is not published.

This validation does not replace package trust or dependency review; it prevents common active SVG payloads from being inlined into generated pages.

## Licensing

`license_path` is required to exist in the archive. Its contents are copied to `target/licenses/<pack-name>.txt`.

Vendoring fails if the configured license file is absent. markata-go does not interpret or determine compatibility of license terms; site authors remain responsible for choosing packs they are permitted to redistribute.

The built-in Lucide fallback MUST use the same license materialization path as explicit vendoring.

## Lifecycle

The explicit plugin runs during `Configure` with `PriorityEarly`. This ensures explicitly vendored files are present before the `icons` plugin indexes local SVG roots later in Configure/Transform processing.

The plugin is included in the standard and minimal built-in plugin sets but remains inert unless explicitly enabled.

The `icons` plugin's default Lucide fallback may call the vendor implementation later during Transform when it encounters the first unresolved Lucide shortcode. After successful materialization, the `icons` plugin re-indexes its configured roots before retrying that shortcode.
