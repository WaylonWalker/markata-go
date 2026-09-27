# Pins

Markata-Go automatically creates a visual link collection at `/pins/` when the site contains published link posts.

A **link post** is any public, published post with a non-empty `link` frontmatter value:

```yaml
---
title: A useful article
published: true
date: 2026-09-27
link: https://example.com/article
description: A short note about why this is worth saving.
image: https://example.com/article-card.jpg
---
```

The normal post remains available at its own Markata URL for your notes or commentary. On `/pins/`, the card distinguishes between:

- the external destination (`link`)
- the local post (`Read notes`)
- the destination hostname
- publication date, description, and optional image metadata

## Included posts

A post appears on `/pins/` when all of the following are true:

- `published: true`
- `link` is a non-empty string
- the post is not a draft
- the post is not skipped
- the post is not private

Pins are sorted newest-first when dates are available. Undated pins sort after dated pins.

## Images

The pinboard uses the first available image from these common frontmatter fields:

1. `image`
2. `cover`
3. `cover_image`
4. `featured_image`
5. `thumbnail`
6. `og_image`
7. `social_image`
8. `hero_image`

Images are optional. Text-only link posts still render as cards.

## Theme behavior

The default theme uses a responsive CSS-column pinboard. Cards preserve their natural image height, avoid breaking between columns, and collapse to one column on narrow screens. Theme authors can override `pins.html` to provide a different presentation while keeping the generated pin data.

If a site has no qualifying link posts, Markata-Go does not create `/pins/`.
