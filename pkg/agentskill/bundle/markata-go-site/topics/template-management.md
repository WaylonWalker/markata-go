# Template Creation And Management

Use this topic when the task changes layout, partials, cards, feed pages, or page-specific rendering.

## Template Engine

Markata-go uses `pongo2`, which is Django/Jinja2-like.

For a fuller offline field list, read `../reference/template-context.md`.

Core syntax:

```django
{{ post.title }}
{% if post.published %}...{% endif %}
{% for tag in post.tags %}{{ tag }}{% endfor %}
{% extends "base.html" %}
{% include "partials/header.html" %}
```

## Search Order

Templates are resolved in this order:

1. project `templates/`
2. `themes/<theme>/templates/` if present
3. `themes/default/templates/` if present
4. embedded default theme templates bundled with markata-go

That means a file in the site's `templates/` directory wins.

## Default Files To Know

- `templates/base.html`
- `templates/post.html`
- `templates/feed.html`
- `templates/layouts/`
- `templates/components/`
- `templates/partials/`

For first sites, assume:

- `post.html` is the single-content template
- `feed.html` is the list/archive template
- `base.html` provides the shared page shell

## High-Value Context Variables

Usually available:

- `post`
- `body`
- `config`
- `feed`
- `page`
- `posts`
- `core`

Convenience aliases are also injected:

- `title`
- `date`
- `tags`
- `slug`
- `href`
- `published`
- `draft`
- `description`
- `article_html`
- `site_title`
- `site_url`
- `site_description`
- `site_author`

## Common Post Fields In Templates

Typical `post` keys used in templates:

- `post.title`
- `post.slug`
- `post.href`
- `post.date`
- `post.tags`
- `post.description`
- `post.article_html`
- `post.structured_data`
- `post.Extra.*` in some templates

The rendered article body is normally output with:

```django
{{ body | safe }}
```

## Feed And Pagination Variables

Useful feed/page fields:

- `feed.title`
- `feed.description`
- `feed.slug`
- `feed.posts`
- `page.number`
- `page.posts`
- `page.has_prev`
- `page.has_next`
- `page.prev_url`
- `page.next_url`
- `page.total_pages`
- `page.pagination_type`

## Built-In Filters Worth Knowing

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
- `striptags`
- `plaintext`
- `safe`
- `absolute_url`
- `theme_asset`
- `theme_asset_hashed`
- `asset_url`

## Common Patterns

### Base Template

```django
<!DOCTYPE html>
<html>
<head>
  <title>{% block title %}{{ site_title }}{% endblock %}</title>
</head>
<body>
  {% block content %}{% endblock %}
</body>
</html>
```

### Post Template

```django
{% extends "base.html" %}

{% block title %}{{ post.title }} | {{ site_title }}{% endblock %}

{% block content %}
<article>
  <h1>{{ post.title }}</h1>
  {% if post.date %}<time>{{ post.date | date:"January 2, 2006" }}</time>{% endif %}
  <div class="post-content">{{ body | safe }}</div>
</article>
{% endblock %}
```

### Feed Template

```django
{% extends "base.html" %}

{% block content %}
<section>
  <h1>{{ feed.title }}</h1>
  {% for post in page.posts %}
    <article>
      <h2><a href="{{ post.href }}">{{ post.title }}</a></h2>
    </article>
  {% endfor %}
</section>
{% endblock %}
```

## Per-Post Template Selection

Common frontmatter options:

```yaml
template: custom/special-page.html
layout: landing
```

Use `layout` when the site has a base-driven layout system. Use `template` when you need an explicit file.

## Layout Resolution

When a site uses the layout system, the effective layout is resolved roughly in this order:

1. frontmatter `layout`
2. feed-based layout config
3. global `[markata-go.layout].name`
4. fallback blog-like layout

Useful config shape:

```toml
[markata-go.layout]
name = "blog"

[markata-go.layout.paths]
"/docs/" = "docs"
"/about/" = "landing"

[markata-go.layout.feeds]
"docs" = "docs"
"blog" = "blog"
```

## Template Presets And Default Templates

Some sites may use config-defined template presets or default per-format templates.

Useful concepts:

- `template_presets`: named bundles for html/text/ansi/markdown/og templates
- `default_templates`: global fallback templates per format

If a post has `template: blog` and the site defines a `blog` template preset, that preset can expand to multiple output-format templates.

If the repo has no local examples, start from:

- `../examples/templates/base.html`
- `../examples/templates/post.html`
- `../examples/templates/feed.html`

## When To Choose Which Mechanism

- small structural change across many pages: layout config
- one reusable view pattern: partial or layout
- single page needs a unique shell: `template`
- many formats need coordinated template mapping: template preset
- small markup tweak: edit the smallest partial involved

## Per-Format Template Rules

Markata-go adapts template names by output format:

- `post.html` -> `post.txt`
- `post.html` -> `post.ansi`
- `post.html` -> `post.md`
- `post.html` -> `post-og.html`

Hardcoded fallbacks include:

- HTML: `post.html`
- text: `default.txt`
- ANSI: `default.ansi`
- markdown/text-ish fallback: `raw.txt`
- OG: `og-card.html`

## Guidance

- Prefer overriding the smallest template that solves the task.
- Use `layout` for base-driven page variants and `template` only when a full standalone template is needed.
- Reuse existing partials and components before creating new ones.
- Keep naming aligned with the site's current template structure.
- Confirm which template is already in use through nearby frontmatter and config before editing.
- For a first site, start by copying and editing `post.html`, `feed.html`, or a single partial instead of rebuilding the entire template tree.
- If a task sounds like sidebar placement, TOC behavior, or page shell selection, check `[markata-go.layout]` before creating new template files.

## Common Tasks

- customize post or feed cards
- add or adjust partials
- create a new layout under `templates/layouts/`
- apply a page-specific template via frontmatter
