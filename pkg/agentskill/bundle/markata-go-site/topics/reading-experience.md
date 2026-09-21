# Reading Experience

Use this topic when a site owner asks for "better reading", a table of
contents, sidenotes, link previews, code block titles or highlighted lines,
reader mode, a serif option, "last updated", "edit on GitHub", or series
navigation. The default theme already ships all of these; most requests are a
config flag or a small Markdown change, not a template or CSS rewrite.

## Decision Table

| Ask | Do this | Do not |
|-----|---------|--------|
| "Add a table of contents" | `[markata-go.components.doc_sidebar] enabled = true` (`min_depth`/`max_depth` 2..3 for blogs). On wide screens it floats into the margin with scroll-spy automatically. | Write a TOC template or JS |
| "Sidenotes / margin notes" | Use standard footnotes `[^1]`. They become margin notes on wide screens and tap popovers on narrow ones. | Add HTML asides or a plugin |
| "Hover previews on links" | Already on for wikilinks and plain internal links (`/slug/` and same-origin absolute). Opt a link out with `class="no-preview"`; disable with `[markata-go.wikilink_hover] all_internal_links = false`. | Rewrite links as wikilinks |
| "Code block file name / highlight lines" | Info string extras: ```` ```python title="app.py" {2,5-7} ```` or `hl_lines="2 5-7"`. | Wrap fences in HTML |
| "Serif body text" | Visitors have an `Aa` toggle next to text size. Set the site default with `[markata-go.theme] reading_font = "serif"`. Override the stack with `"--font-serif-reading"` in `theme.variables`. | Change `--font-body` (that also changes UI text) |
| "Show last updated" | On by default when frontmatter `modified` (aliases `lastmod`, `updated`, `updated_at`, `last_modified`) is ≥ 1 day after `date`. Turn off with `components.post_meta.show_updated = false`. | Add a byline template |
| "Edit on GitHub link" | `[markata-go.components.post_meta] edit_url = "https://github.com/o/r/edit/main/{path}"`; `{path}` is the source path relative to the site root. | Hardcode per-post URLs |
| "Reader mode" | Exists: header button, `s` shortcut, `?reader=1`. Hide the button with `components.post_meta.reader_toggle = false`. | Build a separate template |
| "Key takeaway box / pull quote" | `!!! takeaway` and `!!! pullquote` admonitions. A pullquote title renders as attribution. | Style blockquotes by hand |
| "Part N of M series navigation" | Add `series: name` to frontmatter or use a `type = "series"` / `type = "guide"` feed; the card and footer prev/next appear when the series has >1 post. Hide with `components.post_meta.series_card = false`. | Hand-write prev/next links |
| "Back to top / smooth anchors" | Built in. | Add JS |

## Config Reference

```toml
[markata-go.theme]
text_size = "large"        # small | medium | large | x-large
reading_font = "sans"      # sans | serif
show_text_size_control = true   # hides both the size select and the Aa toggle when false

[markata-go.components.doc_sidebar]
enabled = true
position = "right"
min_depth = 2
max_depth = 3

[markata-go.components.post_meta]
show_updated = true
reader_toggle = true
series_card = true
edit_url = ""              # empty disables the edit link
edit_label = "Edit this page"

[markata-go.wikilink_hover]
all_internal_links = true
```

## Template Context Added For These Features

Available in post templates:

- `post.modified`, `post.updated_at` (nil unless modified ≥ date + 24h)
- `post.prev`, `post.next`, `post.prevnext` (maps; only populated for
  contexts computed before render, prefer `series_nav`)
- `series_nav` — `{feed_slug, feed_title, href, position, total, is_first,
  is_last, percent, description?, prev?, next?}`; nil when the post is not in
  a series/guide feed
- `post_edit_url` — rendered edit link or empty string
- `config.components.post_meta.*`, `config.theme.reading_font`

## Verification

After changing any of the above, run a clean build of the site and check the
rendered HTML rather than the source:

- TOC: `<aside class="doc-sidebar">` present on posts with headings
- Link previews: `data-preview="internal"` on plain internal anchors
- Code chrome: `<div class="code-block" data-lang=... data-title=...>` and
  `class="line hl"` for highlighted lines
- Series: `Part N of M` text and `<nav class="post-nav guide-navigation">`
- Updated: `<time class="dt-updated">`

Margin TOC, sidenotes, tooltips, copy buttons, and back-to-top are applied by
`reading.js`/`tooltips.js` in the browser, so verify those with a headless
browser at ≥ 1201px wide (margin layout) and at a phone width (drawer and tap
popovers).
