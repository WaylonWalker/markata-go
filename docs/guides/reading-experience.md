---
title: "Reading Experience"
description: "Margin table of contents, sidenotes, link previews, code chrome, reader mode, serif toggle, series cards, and the other long-form reading features built into the default theme"
date: 2026-09-21
published: true
tags:
  - documentation
  - themes
  - reading
---

# Reading Experience

The default theme ships a set of long-form reading features inspired by the
best documentation and essay sites. Almost all of them work with content you
already have; a few add small opt-in syntax. This page is the map.

| Feature | Needs content changes? | Configuration |
|---------|------------------------|---------------|
| [Margin table of contents with scroll-spy](#margin-table-of-contents) | No | `components.doc_sidebar.enabled = true` |
| [Sidenotes](#sidenotes) | No (uses footnotes) | Always on |
| [Link previews](#link-previews) | No | `wikilink_hover.all_internal_links` |
| [Code chrome](#code-chrome) | Optional `title=` / `{lines}` | Always on |
| [Serif reading font](#serif-reading-font) | No | `theme.reading_font` |
| [Last updated and edit link](#last-updated-and-edit-link) | No | `components.post_meta` |
| [Reader mode](#reader-mode) | No | `components.post_meta.reader_toggle` |
| [Reading callouts](#reading-callouts) | Opt in with `!!! takeaway` / `!!! pullquote` | Always on |
| [Series card and footer navigation](#series-card) | No (uses `series:` / guide feeds) | `components.post_meta.series_card` |
| [Anchors, smooth scrolling, back to top](#anchors-and-back-to-top) | No | Always on |

## Margin table of contents

Enable the document sidebar and every post with headings gets an
"On this page" list:

```toml
[markata-go.components.doc_sidebar]
enabled = true
position = "right"
min_depth = 2
max_depth = 3
```

On wide screens (1201px and up, with room beside the article) the list floats
into the margin next to the article instead of living in the drawer. As you
scroll, the current section is highlighted (`.toc-link--active`, with
`aria-current="location"`). When sidenotes are present the list moves to the
left margin so the two never fight. On narrower screens the list stays in the
sidebar drawer (`Shift+B`).

## Sidenotes

Standard Markdown footnotes become sidenotes automatically:

```markdown
This claim needs a source[^1].

[^1]: The source, which now sits in the margin beside the paragraph.
```

- **Wide screens:** each note sits in the right margin aligned with its
  paragraph. Notes never overlap; later notes are pushed down.
- **Narrow screens:** the footnote number becomes a tap target that opens a
  small popover under the paragraph. Tap anywhere else or press `Escape` to
  close it.
- The end-of-article footnote list is hidden while notes are shown in the
  margin, and shown again when there is no room.
- The margin and popover copies are visual-only. Screen readers use the
  canonical end-of-article footnote list once, so the note is not announced
  twice.
- Keyboard and assistive-technology activation follows the normal footnote
  link to that canonical list; pointer taps use the decorative popover.

`!!! aside` blocks are a different tool: use them for author commentary that
should sit beside the flow, not for citations.

## Link previews

Hovering a wikilink has always shown a preview card. The same preview now
appears on:

- plain Markdown links to other posts: `[my rule](/rule-1/)`
- absolute links to your own site: `[my rule](https://example.com/rule-1/)`
- keyboard focus, not only mouse hover

The data is added at build time (`data-title`, `data-description`,
`data-date`, `data-preview="internal"`) so there is no client-side lookup.
Links that already carry chrome classes (`heading-anchor`, `tag`, `card`,
`footnote-ref`, ...) or that point at files or the home page are left alone.
Add `class="no-preview"` to a link to opt out. Disable the feature entirely:

```toml
[markata-go.wikilink_hover]
all_internal_links = false
```

## Code chrome

Fenced code blocks render inside a `.code-block` wrapper with a header that
shows the language badge, an optional file title, and a copy button.

````markdown
```python title="hello.py" {2}
def hello():
    print("hi")
    return 1
```
````

- `title="..."` shows a file name in the header.
- `{2}`, `{1,3-5}`, or `hl_lines="1 3-5"` highlight lines (`.line.hl`).
- Fences with no language, or languages the highlighter does not know, keep
  plain `<pre><code>` markup so plugins such as Mermaid, Chart.js, and CSV
  tables keep working; the copy button is still added in the browser.

## Serif reading font

Next to the text-size selector there is an `Aa` toggle that switches the
article prose between the fontpack's body font and a zero-download serif stack
(`--font-serif-reading`: Charter, Iowan Old Style, Palatino, Georgia, ...).
Headings, code, and UI keep their fontpack faces. The choice is stored in
`localStorage` under `reading-font`.

Set the default for visitors who have not chosen:

```toml
[markata-go.theme]
reading_font = "serif"   # or "sans" (default)
```

The toggle is hidden together with the text-size control when
`show_text_size_control = false`. Override the serif stack in
`[markata-go.theme.variables]` with `"--font-serif-reading"`.

## Last updated and edit link

```toml
[markata-go.components.post_meta]
show_updated = true                 # default
edit_url = "https://github.com/you/site/edit/main/{path}"
edit_label = "Edit this page"       # default
```

- **Updated date** appears in the byline when a post's modified date is at
  least a day after its `date`. Set it in frontmatter with `modified`
  (aliases: `lastmod`, `updated`, `updated_at`, `last_modified`).
- **Edit link** is off until `edit_url` is set. `{path}` is replaced with the
  post's path relative to the site root, for example `pages/rule/rule-1.md`.

## Reader mode

Reader mode strips the page down to the article: sidebars, series card,
graph, share panel, and copy controls are hidden, and the text sits alone in
a wide measure. Toggle it with:

- the **Reader mode** button in the post header
- the `s` keyboard shortcut
- `?reader=1` in the URL

`Escape` or pressing the button again exits. Hide the button with
`components.post_meta.reader_toggle = false`; the shortcut keeps working.
When reader mode is active, keyboard, series, feed-sidebar, and view-transition
navigation preserve `?reader=1` on same-origin post URLs.

## Reading callouts

Two admonition types are designed for essays rather than warnings:

```markdown
!!! takeaway
    Use fewer, better borders.

!!! pullquote
    The reading column is the product.

!!! pullquote "Jane Author"
    A pullquote with an attribution line.
```

- `takeaway` is a boxed summary with an arrow icon and a stronger top rule.
- `pullquote` is large, centered, and set in the display font. A title, when
  given, renders as an attribution line below the quote.

Existing content is unaffected; these are additions to the
[admonition types](/docs/guides/markdown/#admonition-types).

## Series card

Posts that belong to a series (`series: name` in frontmatter) or to a feed of
`type = "series"` / `type = "guide"` show a card under the header:

- "Part N of M" with the series title linking to the series page
- a progress bar
- "Up next" / "Previously" links

The card only appears when the series has more than one post. The footer of
the post repeats previous / next navigation with a link back to the series.
Turn the card off with `components.post_meta.series_card = false`.

## Anchors and back to top

- Headings carry `id`s and hover anchors. Anchored scroll targets are offset
  below the sticky header (`scroll-margin-top`).
- Scrolling is smooth for in-page links unless the visitor prefers reduced
  motion.
- A floating **back to top** button appears after scrolling roughly one
  screen; `g g` still jumps to the top from the keyboard.

## Printing

Print styles hide sidebars, buttons, the series card, and the back-to-top
control, and print sidenotes as endnotes.

## See also

- [Themes](/docs/guides/themes/) — palettes, fontpacks, and text size presets
- [Keyboard Navigation](/docs/guides/keyboard-navigation/)
- [Sidebars](/docs/guides/sidebars/)
- [Series](/docs/guides/series/)
- [Markdown](/docs/guides/markdown/) — footnotes, admonitions, code blocks
