# Feed Calendar View

## Status

Implemented as a default peer view of normal HTML feeds.

## Purpose

The feed calendar view visualizes publishing activity over time. It complements the feed's normal presentation by making posting cadence, gaps, and clusters visible across years and months.

The design is inspired by Jim Nielsen's calendar archive, where List and Calendar are peer views of the same archive and dates with posts are visually marked.

## Availability

Calendar MUST be available on every built-in normal HTML feed without requiring a second feed or a per-feed template override.

The normal feed URL remains canonical for the feed's primary presentation. Calendar is an in-page alternate state and MUST be deep-linkable with the `view=calendar` query parameter:

- `/blog/?view=calendar`
- `/?view=calendar` for a root feed

The existing Simple HTML view remains a separate compact page at `/simple/` and MUST link to Calendar. Built-in alternate primary templates such as `feed-photo-grid.html` MUST expose Calendar as a peer view too.

The dedicated `calendar-feed.html` template remains supported for intentional calendar-first customization, but selecting it MUST NOT be required to use Calendar on ordinary feeds.

## Rendering contract

A normal feed calendar integration MUST:

1. Preserve the existing server-rendered primary feed as the no-JavaScript default.
2. Keep Calendar controls hidden until enhancement initializes successfully.
3. Build calendar source data from the complete `feed.posts` collection, not `page.posts`, so pagination does not truncate history.
4. Switch between the primary feed presentation and Calendar in place.
5. Set `?view=calendar` when Calendar is selected and remove that query value when returning to the primary presentation.
6. Honor an initial `?view=calendar` deep link when enhancement initializes.
7. Include all twelve months for every represented year.
8. Place every day of each month in its correct weekday column.
9. Visually distinguish dates containing one or more posts.
10. Expose all post titles and links for a marked date.
11. Support multiple posts on the same date.
12. Preserve post links as normal anchors.
13. Use theme color/spacing variables rather than site-specific colors.

Posts without a date are omitted from the calendar grid and remain available through the feed's primary presentation. A dedicated `calendar-feed.html` template MAY retain an undated list fallback.

## View navigation

Normal feed headers SHOULD expose the human-facing presentation choices together:

- primary presentation (`Posts`, `Grid`, or another template-appropriate label)
- `Simple`, when simple HTML is enabled
- `Calendar`

The Simple view MUST provide links back to the primary presentation and Calendar.

When the feed sidebar is enabled on post pages, it MUST expose a `calendar` view link. If the user changes the selected/cycled sidebar feed, that link MUST resolve to the selected feed's calendar state.

RSS, Atom, JSON, Markdown, text, and sitemap outputs remain export/subscription formats and are not reclassified as human-facing view modes.

## Date semantics

- Calendar grouping uses the rendered `YYYY-MM-DD` date value from each post.
- Gregorian calendar rules apply, including leap years.
- Month lengths and weekday offsets are computed in the browser from the ISO date components.
- Calendar construction uses local `Date(year, month, day)` values rather than parsing `YYYY-MM-DD` with `Date.parse`, avoiding UTC date rollover surprises.

## Progressive enhancement

The calendar is progressive enhancement over server-rendered feed data:

- With JavaScript disabled, users see the existing primary feed presentation.
- A hidden complete dated source supplies Calendar without duplicating visible content.
- With JavaScript enabled, the view selector becomes available.
- Calendar day controls use native `<details>/<summary>` disclosure so keyboard users can reveal posts for a day without custom key handling.
- Generated calendars include weekday labels and descriptive accessible labels for marked dates.

The dedicated `calendar-feed.html` template MAY continue to use its complete server-rendered list as its no-JavaScript fallback and default to Calendar after enhancement.

## Empty and undated feeds

When no dated posts are available, Calendar enhancement MUST NOT replace the primary feed with an empty calendar. The normal feed remains usable.

## Theme assets

The default theme provides:

- `static/css/calendar-feed.css`
- `static/js/calendar-feed.js`

Built-in feed templates load these through `theme_asset_hashed` so normal asset hashing/caching behavior applies.

## Compatibility

- Feed membership, sorting, pagination, and syndication formats are unchanged.
- Existing primary HTML templates keep their presentation semantics.
- Custom templates do not automatically gain Calendar unless they opt into the calendar data/control hooks.
- The dedicated `calendar-feed.html` template remains supported for compatibility and customization.
