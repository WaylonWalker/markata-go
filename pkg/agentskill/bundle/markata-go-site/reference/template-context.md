# Template Context Reference

This reference is for agents working in a markata-go site repo without the markata-go source tree.

## Core Template Variables

Usually available in HTML templates:

- `post`: current post as a map-like object
- `body`: rendered article HTML string
- `config`: resolved site configuration
- `feed`: current feed config when rendering a feed page
- `page`: current paginated feed page
- `posts`: current post list for feed/index contexts
- `core`: lifecycle manager-like object exposed to templates in some contexts

## Common `post` Keys

- `post.path`
- `post.content`
- `post.slug`
- `post.href`
- `post.hrefs`
- `post.inlinks`
- `post.outlinks`
- `post.published`
- `post.draft`
- `post.private`
- `post.skip`
- `post.tags`
- `post.template`
- `post.templateKey`
- `post.html`
- `post.article_html`
- `post.title`
- `post.date`
- `post.description`
- `post.author`
- `post.authors`
- `post.author_objects`
- `post.Extra`

## Common Top-Level Aliases

Markata-go also injects convenient top-level aliases:

- `title`
- `date`
- `tags`
- `slug`
- `href`
- `published`
- `draft`
- `private`
- `description`
- `article_html`

## Site Aliases

- `site_title`
- `site_url`
- `site_description`
- `site_author`
- `authors`
- `default_author`
- `default_author_id`

## Feed Keys

- `feed.slug`
- `feed.base_url`
- `feed.title`
- `feed.description`
- `feed.filter`
- `feed.sort`
- `feed.reverse`
- `feed.items_per_page`
- `feed.limit`
- `feed.offset`
- `feed.posts`
- `feed.formats`

## Page Keys

- `page.number`
- `page.posts`
- `page.has_prev`
- `page.has_next`
- `page.prev_url`
- `page.next_url`
- `page.total_pages`
- `page.total_items`
- `page.items_per_page`
- `page.page_urls`
- `page.pagination_type`

## Sidebar/Navigation Context

- `sidebar_items`
- `sidebar_title`
- `resolved_content_sidebar`

## Common Filters

- `date`
- `date_format`
- `rss_date`
- `atom_date`
- `slugify`
- `truncate`
- `truncatewords`
- `default_if_none`
- `length`
- `first`
- `last`
- `join`
- `sort`
- `reverse`
- `selectattr`
- `rejectattr`
- `striptags`
- `plaintext`
- `linebreaks`
- `linebreaksbr`
- `safe`
- `urlencode`
- `absolute_url`
- `theme_asset`
- `theme_asset_hashed`
- `asset_url`
- `google_fonts_url`
- `reading_time`
- `excerpt`

## Safe Usage Pattern

Use `{{ body | safe }}` for already-rendered HTML body content.

Use plain `{{ post.title }}` or `{{ config.title }}` for text values.

Use `post.Extra.<field>` when the site relies on custom frontmatter not covered by built-in fields.
