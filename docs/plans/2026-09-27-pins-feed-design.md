---
title: "Pins Feed Design"
description: "Design notes for the Markata pinboard feed."
date: 2026-09-27
published: false
tags:
  - documentation
  - feeds
---

# Pins feed design

## Purpose

Give saved links and link posts a useful destination beyond the chronological blog stream. `/pins/` collects every published post with a non-empty `link` frontmatter value.

## Shape

The subscription feed collector supplies an implicit `pins` HTML feed unless the site defines its own `pins` feed. It uses the existing filter, sorting, template, and output pipeline, so link posts remain ordinary Markdown posts and can still be selected by other feeds. The feed has a dedicated theme template and CSS card treatment rather than changing the appearance of every link card site-wide.

The page reads as a field notebook: a restrained editorial heading followed by a responsive, varied-height pinboard of source cards. Cards separate the saved post (internal title/body) from its external destination (domain and outbound link), show optional image, date, and tags, and retain keyboard focus and narrow-screen layouts. Theme variables provide colors so both light and dark palettes remain legible.

## Data and behavior

- Selection: `published == true and link`.
- Order: newest first.
- Output: HTML at `/pins/`; existing feed configuration with slug `pins` wins.
- Cards display optional `image`, `title`, `description`/body, `link`, date, and tags.
- Empty sets render the existing feed empty state.

## Verification

Test implicit feed registration, custom feed precedence, and selection of linked versus unlinked posts. Build the gallery demo to verify the route, card markup, and responsive styles. Document how to override the built-in feed with an explicit feed config.
