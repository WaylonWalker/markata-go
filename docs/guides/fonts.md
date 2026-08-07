---
title: "Font packs"
description: "Choose offline system typography or a vendored Markata font pack."
date: 2026-08-06
published: true
tags:
  - documentation
  - typography
---

# Font packs

Markata defaults to the offline `system` pack. It makes no external font
requests and emits no font binaries. To choose another pack, add this to the
site configuration:

```yaml
fontpack: system-reader
```

Catalog-defined bundled packs use vendored, stable WOFF2 tiers. The built-in
catalog, manifests, licenses, and font assets are embedded in the executable,
so these packs work from an installed or GoReleaser binary in any working
directory. Bundled builds do not run Python or FontTools during `markata build`.

System packs use local system stacks and transfer zero font files. Custom packs
are loaded from an explicit catalog and may reference local assets. Relative
paths in a custom catalog are resolved relative to the catalog file, not the
current working directory.

Inspect the catalog with:

```text
markata-go fonts packs
markata-go fonts show system
markata-go fonts doctor
markata-go fonts verify
markata-go fonts report
```

## Compare packs on one site

### Painted Sign

`painted-sign` combines Finger Paint display lettering with scratchy Rock Salt
headings and accents over a clean Source Sans 3 prose stack. DM Mono remains
available for code, so the expressive hierarchy does not compromise reading.

The `fontpack` frontmatter field overrides the site default for one page:

```yaml
---
title: "Brush comparison"
fontpack: brush-poster
---
```

All selected page packs share one generated stylesheet and deduplicated asset
set. The comparison pages in `pages/post/test-*.md` use this mechanism.

Markata tracks emitted files in `output/assets/fonts/.markata-fonts.json`.
Switching packs removes obsolete Markata-generated files while preserving
unrelated user-managed files in the same directory.

Coverage is selected per font family. A family gets `latin-ext` only when its
manifest provides that tier; otherwise Markata falls back to that family's
`full` tier, so expressive faces without extended subsets still render all
text. Role `weight`, `style`, `size`, and `optical_size` values are emitted as
CSS properties and applied to the corresponding body, heading, code, lead,
quote, and caption elements.

The maintenance generator reads variable-font `fvar` axes from the source
font, including their real weight and optical-size bounds. To check generated
manifests without rewriting them, run it with `--check` and the pinned
google/fonts checkout.

Regenerate the comparison pages after changing the catalog with:

```bash
uv run --with pyyaml python scripts/generate_fontpack_demos.py
```
