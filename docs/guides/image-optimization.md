---
title: "Image Optimization"
description: "Generate AVIF and WebP variants for local images"
date: 2026-02-15
published: true
tags:
  - documentation
  - images
  - performance
---

# Image Optimization

The `image_optimization` plugin generates AVIF and WebP variants for local images and rewrites HTML to use a `<picture>` element with the original image as the fallback.

## Quick Start

Enable the plugin in your hooks:

```toml
[markata-go]
hooks = ["default", "image_optimization"]

[markata-go.image_optimization]
enabled = true
```

## Requirements

- `avifenc` for AVIF output
- `cwebp` for WebP output

If a tool is missing, the build continues and that format is skipped.

## Example Output

Input HTML:

```html
<img src="/images/cat.jpg" alt="Cat">
```

Output HTML:

```html
<picture>
  <source type="image/avif" srcset="/images/cat.avif">
  <source type="image/webp" srcset="/images/cat.webp">
  <img src="/images/cat.jpg" alt="Cat">
</picture>
```

The generated `.avif` and `.webp` files are written next to the original image in the output folder.

## Configuration

```toml
[markata-go.image_optimization]
enabled = true
formats = ["avif", "webp"]
quality = 80
avif_quality = 80
webp_quality = 80
cache_dir = ".markata/image-cache"
avifenc_path = ""
cwebp_path = ""
```

### Notes

- Only local images are processed (relative paths and site-root paths).
- External URLs, protocol-relative URLs, and data URIs are skipped.
- Images already inside `<picture>` are left unchanged.

For full details, see the [plugin reference](/docs/reference/plugins/#image_optimization).
