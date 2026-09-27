---
title: "Photo Galleries"
description: "Create a gallery page and a photo-first feed with the default theme."
date: 2026-09-26
published: true
slug: /docs/guides/galleries/
tags:
  - documentation
  - galleries
---

# Photo galleries

Create a gallery page with `template: gallery.html` and a list of images:

```yaml
---
title: "Field Notes"
description: "Photographs from the garden."
date: 2026-09-26
published: true
slug: gallery
template: gallery.html
gallery:
  - src: /images/flowers.webp
    alt: Pink flowers in afternoon light
    caption: Late summer in the garden
    width: 1600
    height: 1067
  - src: /images/leaves.webp
    alt: Rain on broad green leaves
    width: 1200
    height: 1800
---

An optional introduction can follow the frontmatter.
```

Each image needs `src` and meaningful `alt` text. `caption` is optional. Add the original image's `width` and `height` when you know them so the page reserves space while loading. Images without both fields still render. The page uses two columns on small screens, three on medium screens, and four on wide screens. Readers can open an image, use the arrow keys or swipe to move through the gallery, and press Escape to close it. The original image links work without JavaScript.

Relative image paths and trusted media hosts receive an 800-pixel-wide preview. Other image URLs are used as supplied. See [template media settings](/docs/guides/templates/) if your images live on another CDN.

## Photo-first feeds

Use the existing photo grid template for a feed of individual photo posts:

```toml
[[markata-go.feeds]]
slug = "shots"
title = "Shots"
filter = "published == true and 'shots' in tags"
sort = "date"
reverse = true

[markata-go.feeds.templates]
html = "feed-photo-grid.html"
```

Set each post's `image` or `cover_image`, plus a title and description. Add `card_classes: "col-span-2"` to let a selected card span two columns. The grid shows a date and, for longer posts, a word count when those values are available. See [feeds](/docs/guides/feeds/) for filtering and pagination.

## Long post titles

Post pages now get a conservative starting size based on title length. The browser refines the fit after the active font loads and when the viewport changes. Titles remain visible and may wrap when the space beside the byline is narrow. This works with custom font packs and does not require site JavaScript.
