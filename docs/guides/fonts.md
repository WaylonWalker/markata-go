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

Catalog-defined bundled packs use vendored, stable WOFF2 tiers. This
repository currently ships the assets required by `field-notebook` and
`brush-poster`; other bundled pack definitions remain catalog entries until
their family assets are vendored. Bundled builds do not run Python or FontTools
during `markata build`.

Inspect the catalog with:

```text
markata-go fonts packs
markata-go fonts show system
markata-go fonts doctor
```

## Compare packs on one site

The `fontpack` frontmatter field overrides the site default for one page:

```yaml
---
title: "Brush comparison"
fontpack: brush-poster
---
```

All selected page packs share one generated stylesheet and deduplicated asset
set. The comparison pages in `pages/post/test-*.md` use this mechanism.

Regenerate the comparison pages after changing the catalog with:

```bash
uv run --with pyyaml python scripts/generate_fontpack_demos.py
```
