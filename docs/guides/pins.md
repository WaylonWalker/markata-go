---
title: "Pins"
description: "Build a theme-aware link board from ordinary post metadata."
date: 2026-09-27
published: true
slug: /docs/guides/pins/
tags:
  - documentation
  - feeds
  - pins
---

# Pins

Markata includes a built-in `/pins/` page for bookmarking and link posts. Any published post with a non-empty `link` field appears there, newest first:

```yaml
---
title: "A field guide to useful maps"
link: "https://example.com/field-guide"
description: "A thoughtful guide to reading a landscape."
image: "/images/maps.webp"
date: 2026-09-27
published: true
---

A note about why this is worth saving.
```

The pinboard keeps the title linked to your local post and provides a separate source link to the external URL. Images, descriptions, dates, tags, and authored notes remain ordinary post data.

The `link` field is intentionally render-neutral outside the pinboard. Adding `link:` does not inject an external preview into the individual post and does not cause an external metadata request by itself. Existing sites can therefore keep using `link` as ordinary frontmatter without changing post rendering.

## Route ownership

The built-in Pins feed is a convenience route, not a reservation. If your site already has a publishable post with the slug `pins`, that post keeps `/pins/` and Markata does not inject the implicit feed. A draft or skipped `pins` post does not reserve the route.

An explicitly configured feed with the slug `pins` also takes precedence over the built-in definition:

```toml
[[markata-go.feeds]]
slug = "pins"
title = "Reading list"
filter = "published == true and 'saved' in tags"
sort = "date"
reverse = true
```

Because an explicit feed is a deliberate configuration choice, normal output-conflict checks still apply if it targets the same path as a published post.

## Default behavior

The implicit Pins feed uses:

```text
filter: published == true and link
sort: date
reverse: true
format: HTML
path: /pins/
template: pins.html
```

Disabling built-in subscription feeds disables the implicit Pins feed along with the implicit root and archive feeds. You can still define your own `pins` feed when you want different selection, sorting, pagination, templates, or output formats.
