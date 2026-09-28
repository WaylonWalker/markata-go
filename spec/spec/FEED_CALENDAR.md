# Feed Calendar View

## Status

Proposed and implemented as an additive built-in feed template.

## Purpose

The feed calendar view visualizes publishing activity over time. It complements the normal reverse-chronological feed by making posting cadence, gaps, and clusters visible across years and months.

The design is inspired by Jim Nielsen's calendar archive, where each year contains month calendars and dates with posts are visually marked.

## Configuration

Calendar rendering uses the existing per-feed HTML template override. No new feed data model or output format is required.

```toml
[[markata-go.feeds]]
slug = "archive"
title = "Archive"
filter = "published == true"
sort = "date"
reverse = true

[markata-go.feeds.templates]
html = "calendar-feed.html"
```

`calendar-feed.html` is a built-in template in the default theme and is also present in the legacy root template tree.

## Rendering contract

The template MUST:

1. Read the complete collection from `feed.posts`, not `page.posts`, so pagination does not truncate the calendar history.
2. Render a useful list of dated post links in HTML before enhancement. This is the accessible and no-JavaScript fallback.
3. Enhance that list into year/month calendar grids when JavaScript is available.
4. Include all twelve months for every represented year.
5. Place every day of each month in its correct weekday column.
6. Visually distinguish dates containing one or more posts.
7. Expose all post titles and links for a marked date.
8. Support multiple posts on the same date.
9. Preserve post links as normal anchors so content remains navigable without client-side routing.
10. Use theme color/spacing variables rather than site-specific colors.

Posts without a date are omitted from the calendar grid but remain visible in the fallback list when they can be rendered meaningfully.

## Date semantics

- Calendar grouping uses the rendered `YYYY-MM-DD` date value from each post.
- Gregorian calendar rules apply, including leap years.
- Month lengths and weekday offsets are computed in the browser from the ISO date components.
- Calendar construction uses local `Date(year, month, day)` values rather than parsing `YYYY-MM-DD` with `Date.parse`, avoiding UTC date rollover surprises.

## Progressive enhancement

The calendar is progressive enhancement over server-rendered feed data:

- With JavaScript disabled, users see a complete dated list of posts.
- With JavaScript enabled, the calendar becomes the default view and a `List` / `Calendar` control switches between representations.
- Calendar day controls use native `<details>/<summary>` disclosure so keyboard users can reveal the posts for a day without custom key handling.
- The generated calendar includes weekday labels and descriptive accessible labels for marked dates.

## Empty feeds

When `feed.posts` is empty, the template renders the same useful empty-state guidance as the normal feed template and does not attempt calendar enhancement.

## Theme assets

The default theme provides:

- `static/css/calendar-feed.css`
- `static/js/calendar-feed.js`

The template loads them through `theme_asset_hashed` so normal asset hashing/caching behavior applies.

## Compatibility

This feature is additive:

- Existing feeds continue to use `feed.html` by default.
- Existing feed pagination and syndication formats are unchanged.
- Custom themes can override `calendar-feed.html` or its static assets.
- The existing feed template contract remains unchanged.