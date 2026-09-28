
# Feed views

Built-in feeds expose three presentation views by default: `default`, `simple`, and `calendar`.
`default` is the feed's normal card/grid page and is always required. `simple` is the compact
`/simple/` page, and `calendar` is the interactive `?view=calendar` peer view.

To disable an alternate view for every feed, set site-wide feed defaults:

```toml
[markata-go.feed_defaults]
views = ["default", "calendar"]
```

A feed can override that set:

```toml
[[markata-go.feeds]]
slug = "notes"
views = ["default", "simple"]
```

Omitting `views` inherits the site defaults. Unknown view names are configuration errors, and an
explicit list must include `default`. Disabling `simple` also stops Markata from generating that
feed's `/simple/` HTML output even when `formats.simple_html` is enabled. Disabling `calendar`
removes Calendar controls and avoids loading its CSS, JavaScript, and hidden calendar source data.
Output-format settings for RSS, Atom, JSON, Markdown, text, and sitemaps are independent of views.
