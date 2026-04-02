# Feed Patterns Reference

Feeds are filtered, sorted, paginated collections of posts. In markata-go, many archive and index pages are feeds.

## Minimal Feed

```toml
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true
```

## Common Patterns

### Home Feed

```toml
[[markata-go.feeds]]
slug = ""
title = "Home"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5
```

### Blog Archive

```toml
[[markata-go.feeds]]
slug = "blog"
title = "All Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 10
```

### Tag-Like Feed

```toml
[[markata-go.feeds]]
slug = "go"
title = "Go Posts"
filter = "'go' in tags and published == True"
sort = "date"
reverse = true
```

### Docs Feed

```toml
[[markata-go.feeds]]
slug = "docs"
title = "Documentation"
filter = "published == True"
sort = "title"
reverse = false
items_per_page = 0
```

## Global Feed Formats

This is a top-level setting that controls which output formats are generated for all feeds. It is NOT placed inside a `[[markata-go.feeds]]` entry.

```toml
[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = true
markdown = false
text = false
```

## Feed Defaults

If the site uses shared feed defaults, check those before editing each feed:

```toml
[markata-go.feeds.defaults]
items_per_page = 10

[markata-go.feeds.defaults.formats]
html = true
rss = true
atom = true
```

## Template Touchpoints

- list/archive HTML usually uses `feed.html`
- card rendering often happens in a partial
- RSS and Atom can use separate XML templates

## Agent Guidance

- if the task is “change an index page”, check whether that page is backed by a feed first
- if the task is “show only X posts”, check `filter`, `limit`, `offset`, and `items_per_page`
- if the task is “change archive card layout”, change the feed template or card partial before changing content
