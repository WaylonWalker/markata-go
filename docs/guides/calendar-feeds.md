---
title: "Calendar Feed Views"
description: "Visualize a Markata-Go feed as year and month calendars with marked publishing days."
date: 2026-09-27
published: true
tags:
  - documentation
  - feeds
  - templates
---

# Calendar Feed Views

Use the built-in `calendar-feed.html` template when you want a feed to communicate *when* you publish, not just list posts newest-first.

The view is inspired by [Jim Nielsen's calendar archive](https://blog.jim-nielsen.com/archive/calendar/): each year contains twelve month grids, and days with posts are visibly marked. Select a marked day to reveal the post title or titles for that date.

## Quick start

```toml
[[markata-go.feeds]]
slug = "archive"
title = "Archive"
description = "Everything I've published, across time."
filter = "published == true"
sort = "date"
reverse = true

[markata-go.feeds.templates]
html = "calendar-feed.html"
```

Build the site normally:

```bash
markata-go build
```

Then open `/archive/`.

## What the view shows

The calendar uses the feed's complete `feed.posts` collection, so it represents the full feed even when the feed normally paginates its HTML output.

For each represented year it renders all twelve months. Every day appears in the correct weekday column. Dates containing posts get an activity marker and an expandable list of titles. If multiple posts share a date, they all appear beneath that day.

The calendar also includes a **List / Calendar** view switch. Calendar is the enhanced default; the list is plain HTML and remains the fallback when JavaScript is unavailable.

## Accessibility and progressive enhancement

The source HTML contains the full dated post list before any JavaScript runs. That means links remain available to crawlers, assistive technology, constrained browsers, and visitors with JavaScript disabled.

Calendar day disclosures use native `<details>` and `<summary>` controls, so marked dates are keyboard-operable without a custom keybinding layer. Weekday labels and day summaries expose useful accessible names while decorative activity marks stay hidden from assistive technology.

## Pagination

You do not need to disable pagination. The calendar template intentionally reads `feed.posts` rather than the current `page.posts` slice.

You can still set `items_per_page` for other feed templates or output behavior without truncating the calendar history.

## Undated posts

Only posts with a valid `date` can be placed on a calendar. Undated posts are not placed in month grids. Give archive content a date when you want it represented in this view.

## Customizing the calendar

Like other built-in templates, `calendar-feed.html` can be copied into your site's `templates/` directory and edited. Its styling and behavior live in the default theme assets:

- `css/calendar-feed.css`
- `js/calendar-feed.js`

The CSS uses theme variables and `currentColor`-derived borders/backgrounds so it follows custom palettes without calendar-specific color configuration.

## Keep a separate normal feed

If you want both a normal card feed and a dedicated calendar URL, define two feeds with the same filter and sort but different slugs/templates:

```toml
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == true"
sort = "date"
reverse = true

[[markata-go.feeds]]
slug = "calendar"
title = "Blog Calendar"
filter = "published == true"
sort = "date"
reverse = true

[markata-go.feeds.templates]
html = "calendar-feed.html"
```

This leaves `/blog/` as the standard card feed and adds `/calendar/` as the activity-oriented archive.