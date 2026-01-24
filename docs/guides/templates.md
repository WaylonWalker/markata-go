---
title: "Templates Guide"
description: "Complete guide to creating and customizing templates with pongo2/Jinja2 syntax"
date: 2024-01-15
published: true
slug: /docs/guides/templates/
tags:
  - documentation
  - templates
---

# Templates Guide

markata-go uses [pongo2](https://github.com/flosch/pongo2), a Django/Jinja2-like template engine for Go. This guide covers everything you need to know about creating and customizing templates.

## Table of Contents

- [Overview](#overview)
- [Template Location](#template-location)
- [Template Syntax](#template-syntax)
- [Available Variables](#available-variables)
- [Built-in Filters](#built-in-filters)
- [Template Inheritance](#template-inheritance)
- [Including Partials](#including-partials)
- [Post vs Feed Templates](#post-vs-feed-templates)
- [Custom Templates Per Post](#custom-templates-per-post)
- [Complete Examples](#complete-examples)

---

## Overview

Templates wrap your rendered Markdown content in HTML layouts. The template system supports:

- **Template inheritance** - Base templates with extendable blocks
- **Includes** - Reusable partial templates
- **Variables and expressions** - Access post, config, and feed data
- **Control flow** - Conditionals and loops
- **Filters** - Transform data for display
- **Custom templates per post** - Override templates in frontmatter

---

## Template Location

Templates are loaded from these locations (in order of priority):

1. **Project templates:** `templates/` directory in project root
2. **Theme templates:** `themes/{theme}/templates/` for custom themes
3. **Default theme:** `themes/default/templates/` as fallback

```
my-site/
├── templates/           # Project templates (highest priority)
│   ├── base.html
│   ├── post.html
│   ├── feed.html
│   └── partials/
│       ├── header.html
│       ├── footer.html
│       └── card.html
├── themes/
│   └── my-theme/
│       └── templates/   # Theme templates
└── posts/
```

Configure the templates directory in `markata-go.toml`:

```toml
[markata-go]
templates_dir = "templates"
```

---

## Template Syntax

### Variables

Access data using double curly braces:

```django
{{ post.Title }}
{{ config.URL }}
{{ post.Date }}
```

### Attribute Access

```django
{{ post.Title }}              {# Direct attribute access #}
{{ post.Extra.custom_field }} {# Nested attributes #}
```

### Filters

Transform values using the pipe (`|`) operator:

```django
{{ post.Title|upper }}
{{ post.Description|truncate:160 }}
{{ post.Date|date_format:"January 2, 2006" }}
{{ post.Tags|join:", " }}
{{ post.ArticleHTML|safe }}
{{ value|default_if_none:"N/A" }}
```

### Comments

```django
{# This is a comment and won't appear in output #}

{#
Multi-line
comment
#}
```

### Control Flow

#### Conditionals

```django
{% if post.Published %}
    <span class="status">Published</span>
{% elif post.Draft %}
    <span class="status">Draft</span>
{% else %}
    <span class="status">Private</span>
{% endif %}
```

#### Loops

```django
{% for tag in post.Tags %}
    <a href="/tags/{{ tag|slugify }}/">{{ tag }}</a>
    {% if not forloop.Last %}, {% endif %}
{% endfor %}

{% for post in posts %}
    <li>{{ post.Title }}</li>
{% empty %}
    <li>No posts found</li>
{% endfor %}
```

#### Loop Variables

Inside `{% for %}` loops, these variables are available:

| Variable | Description |
|----------|-------------|
| `forloop.Counter` | Current iteration (1-indexed) |
| `forloop.Counter0` | Current iteration (0-indexed) |
| `forloop.First` | True if first iteration |
| `forloop.Last` | True if last iteration |
| `forloop.Revcounter` | Iterations remaining (1-indexed) |

---

## Available Variables

### Post Context

When rendering individual posts, these variables are available:

| Variable | Type | Description |
|----------|------|-------------|
| `post` | object | The post being rendered |
| `post.Title` | string | Post title |
| `post.Slug` | string | URL slug |
| `post.Href` | string | Relative URL path (e.g., `/my-post/`) |
| `post.Date` | time | Publication date |
| `post.Published` | bool | Whether the post is published |
| `post.Draft` | bool | Whether the post is a draft |
| `post.Tags` | []string | List of tags |
| `post.Description` | string | Post description |
| `post.Content` | string | Raw Markdown content |
| `post.ArticleHTML` | string | Rendered HTML content (use with `\|safe`) |
| `post.Extra` | map | Additional frontmatter fields |
| `body` | string | Rendered article HTML (alias for `post.ArticleHTML`) |
| `config` | object | Site configuration |

### Config Context

Site configuration is available via `config`:

| Variable | Type | Description |
|----------|------|-------------|
| `config.Title` | string | Site title |
| `config.Description` | string | Site description |
| `config.URL` | string | Site base URL |
| `config.Author` | string | Site author |

Shorthand aliases are also available:

| Alias | Equivalent |
|-------|------------|
| `site_title` | `config.Title` |
| `site_url` | `config.URL` |
| `site_description` | `config.Description` |
| `site_author` | `config.Author` |

### Feed Context

When rendering feeds/archives, these additional variables are available:

| Variable | Type | Description |
|----------|------|-------------|
| `feed` | object | Feed configuration |
| `feed.Title` | string | Feed title |
| `feed.Description` | string | Feed description |
| `feed.Slug` | string | Feed slug |
| `feed.Posts` | []post | All posts in the feed |
| `page` | object | Current page info |
| `page.Number` | int | Current page number |
| `page.Posts` | []post | Posts on this page |
| `page.HasPrev` | bool | Whether there's a previous page |
| `page.HasNext` | bool | Whether there's a next page |
| `page.PrevURL` | string | URL to previous page |
| `page.NextURL` | string | URL to next page |
| `posts` | []post | Posts on current page (alias for `page.Posts`) |

---

## Built-in Filters

### Date Formatting

| Filter | Example | Output |
|--------|---------|--------|
| `date_format` | `{{ date\|date_format:"2006-01-02" }}` | `2024-01-15` |
| `date_format` | `{{ date\|date_format:"January 2, 2006" }}` | `January 15, 2024` |
| `rss_date` | `{{ date\|rss_date }}` | RFC 1123Z format for RSS |
| `atom_date` | `{{ date\|atom_date }}` | RFC 3339 format for Atom |

**Note:** Go uses reference time formatting. Common formats:

| Format | Go Pattern |
|--------|-----------|
| 2024-01-15 | `2006-01-02` |
| January 15, 2024 | `January 2, 2006` |
| Jan 15, 2024 | `Jan 2, 2006` |
| 15 Jan 2024 | `02 Jan 2006` |
| 2024-01-15T10:30:00Z | `2006-01-02T15:04:05Z07:00` |

### String Manipulation

| Filter | Example | Description |
|--------|---------|-------------|
| `slugify` | `{{ "Hello World"\|slugify }}` | Outputs `hello-world` |
| `truncate` | `{{ text\|truncate:100 }}` | Truncate to 100 characters with ellipsis |
| `truncatewords` | `{{ text\|truncatewords:20 }}` | Truncate to 20 words |
| `upper` | `{{ text\|upper }}` | Convert to uppercase |
| `lower` | `{{ text\|lower }}` | Convert to lowercase |
| `title` | `{{ text\|title }}` | Title case |
| `striptags` | `{{ html\|striptags }}` | Remove HTML tags |

### Collections

| Filter | Example | Description |
|--------|---------|-------------|
| `length` | `{{ list\|length }}` | Get length |
| `first` | `{{ list\|first }}` | Get first element |
| `last` | `{{ list\|last }}` | Get last element |
| `join` | `{{ list\|join:", " }}` | Join with separator |
| `reverse` | `{{ list\|reverse }}` | Reverse order |
| `sort` | `{{ list\|sort }}` | Sort alphabetically |

### HTML/Text

| Filter | Example | Description |
|--------|---------|-------------|
| `safe` | `{{ html\|safe }}` | Mark HTML as safe (don't escape) |
| `escape` | `{{ text\|escape }}` | HTML escape (default behavior) |
| `linebreaks` | `{{ text\|linebreaks }}` | Convert newlines to `<p>` and `<br>` |
| `linebreaksbr` | `{{ text\|linebreaksbr }}` | Convert newlines to `<br>` |

### URLs

| Filter | Example | Description |
|--------|---------|-------------|
| `urlencode` | `{{ path\|urlencode }}` | URL encode |
| `absolute_url` | `{{ post.Href\|absolute_url:config.URL }}` | Convert to absolute URL |

### Default Values

| Filter | Example | Description |
|--------|---------|-------------|
| `default_if_none` | `{{ value\|default_if_none:"fallback" }}` | Provide fallback for nil/empty |
| `default` | `{{ value\|default:"fallback" }}` | pongo2 built-in default |

---

## Template Inheritance

Template inheritance lets you create a base layout that child templates extend.

### Base Template

```html
{# templates/base.html #}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ config.Title }}{% endblock %}</title>
    <meta name="description" content="{% block description %}{{ config.Description }}{% endblock %}">
    {% block head %}{% endblock %}
</head>
<body>
    {% block header %}
    <header>
        <nav>
            <a href="/">{{ config.Title }}</a>
            <a href="/blog/">Blog</a>
            <a href="/about/">About</a>
        </nav>
    </header>
    {% endblock %}

    <main>
        {% block content %}{% endblock %}
    </main>

    {% block footer %}
    <footer>
        <p>&copy; {{ config.Author }}</p>
    </footer>
    {% endblock %}

    {% block scripts %}{% endblock %}
</body>
</html>
```

### Extending the Base

```html
{# templates/post.html #}
{% extends "base.html" %}

{% block title %}{{ post.Title }} | {{ config.Title }}{% endblock %}

{% block description %}{{ post.Description|default_if_none:config.Description }}{% endblock %}

{% block head %}
<meta property="og:title" content="{{ post.Title }}">
<meta property="og:type" content="article">
<meta property="og:url" content="{{ config.URL }}{{ post.Href }}">
{% if post.Date %}
<meta property="article:published_time" content="{{ post.Date|atom_date }}">
{% endif %}
{% endblock %}

{% block content %}
<article>
    {{ body|safe }}
</article>
{% endblock %}
```

### Using `{{ block.super }}`

Include parent block content with `{{ block.super }}`:

```html
{% block scripts %}
{{ block.super }}
<script src="/js/post.js"></script>
{% endblock %}
```

---

## Including Partials

Partials are reusable template fragments. Use `{% include %}` to embed them:

```html
{% include "partials/header.html" %}
{% include "partials/card.html" %}
{% include "partials/footer.html" %}
```

### Example Partial: Card

```html
{# templates/partials/card.html #}
<article class="card">
    <a href="{{ post.Href }}">
        {% if post.Extra.cover_image %}
        <img src="{{ post.Extra.cover_image }}" alt="{{ post.Title }}">
        {% endif %}
        <h2>{{ post.Title }}</h2>
    </a>
    {% if post.Description %}
    <p>{{ post.Description }}</p>
    {% endif %}
    <footer>
        {% if post.Date %}
        <time datetime="{{ post.Date|atom_date }}">
            {{ post.Date|date_format:"Jan 2, 2006" }}
        </time>
        {% endif %}
        {% if post.Extra.reading_time %}
        <span>{{ post.Extra.reading_time }}</span>
        {% endif %}
    </footer>
</article>
```

### Example Partial: Navigation

```html
{# templates/partials/nav.html #}
<nav class="main-nav">
    <a href="/" class="logo">{{ config.Title }}</a>
    <ul>
        <li><a href="/blog/">Blog</a></li>
        <li><a href="/tags/">Tags</a></li>
        <li><a href="/about/">About</a></li>
    </ul>
</nav>
```

---

## Post vs Feed Templates

### Post Templates

Post templates render individual content pages. They receive the `post` and `body` variables.

**Default template:** `post.html`

```html
{# templates/post.html #}
{% extends "base.html" %}

{% block title %}{{ post.Title }} | {{ config.Title }}{% endblock %}

{% block content %}
<article class="post">
    <header>
        <h1>{{ post.Title }}</h1>
        {% if post.Date %}
        <time datetime="{{ post.Date|atom_date }}">
            {{ post.Date|date_format:"January 2, 2006" }}
        </time>
        {% endif %}

        {% if post.Tags %}
        <ul class="tags">
            {% for tag in post.Tags %}
            <li><a href="/tags/{{ tag|slugify }}/">{{ tag }}</a></li>
            {% endfor %}
        </ul>
        {% endif %}
    </header>

    <div class="content">
        {{ body|safe }}
    </div>
</article>
{% endblock %}
```

### Feed Templates

Feed templates render lists/archives of posts. They receive `feed`, `page`, and `posts` variables.

**Default template:** `feed.html`

```html
{# templates/feed.html #}
{% extends "base.html" %}

{% block title %}{{ feed.Title }} | {{ config.Title }}{% endblock %}

{% block content %}
<section class="feed">
    <h1>{{ feed.Title }}</h1>
    {% if feed.Description %}
    <p class="description">{{ feed.Description }}</p>
    {% endif %}

    <ul class="post-list">
        {% for post in posts %}
        <li>
            {% include "partials/card.html" %}
        </li>
        {% endfor %}
    </ul>

    {% if page.HasPrev or page.HasNext %}
    <nav class="pagination">
        {% if page.HasPrev %}
        <a href="{{ page.PrevURL }}" class="prev">&larr; Previous</a>
        {% endif %}

        <span>Page {{ page.Number }}</span>

        {% if page.HasNext %}
        <a href="{{ page.NextURL }}" class="next">Next &rarr;</a>
        {% endif %}
    </nav>
    {% endif %}
</section>
{% endblock %}
```

---

## Custom Templates Per Post

Override the default template for specific posts using the `template` frontmatter field.

### Simple Override

```yaml
---
title: "My Landing Page"
template: "landing.html"
---
```

This post will use `templates/landing.html` instead of the default `post.html`.

### Creating a Landing Page Template

```html
{# templates/landing.html #}
{% extends "base.html" %}

{% block title %}{{ post.Title }}{% endblock %}

{% block header %}
{# No header on landing page #}
{% endblock %}

{% block content %}
<div class="landing">
    <section class="hero">
        <h1>{{ post.Title }}</h1>
        {% if post.Description %}
        <p class="tagline">{{ post.Description }}</p>
        {% endif %}
    </section>

    <section class="content">
        {{ body|safe }}
    </section>

    {% if post.Extra.cta_text %}
    <section class="cta">
        <a href="{{ post.Extra.cta_url }}" class="button">
            {{ post.Extra.cta_text }}
        </a>
    </section>
    {% endif %}
</div>
{% endblock %}

{% block footer %}
{# Minimal footer #}
<footer class="minimal">
    <p>&copy; {{ config.Author }}</p>
</footer>
{% endblock %}
```

### Template Fallback

If a specified template doesn't exist, markata-go falls back to:
1. `post.html` (for posts)
2. `feed.html` (for feeds)

---

## Complete Examples

### base.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ config.Title }}{% endblock %}</title>
    <meta name="description" content="{% block description %}{{ config.Description }}{% endblock %}">

    {# Open Graph #}
    <meta property="og:title" content="{% block og_title %}{{ config.Title }}{% endblock %}">
    <meta property="og:description" content="{% block og_description %}{{ config.Description }}{% endblock %}">
    <meta property="og:url" content="{{ config.URL }}{% block og_url %}/{% endblock %}">
    <meta property="og:site_name" content="{{ config.Title }}">

    {# Feeds #}
    <link rel="alternate" type="application/rss+xml" title="{{ config.Title }} RSS" href="{{ config.URL }}/blog/rss.xml">
    <link rel="alternate" type="application/atom+xml" title="{{ config.Title }} Atom" href="{{ config.URL }}/blog/atom.xml">

    {# Styles #}
    <link rel="stylesheet" href="/css/style.css">

    {% block head %}{% endblock %}
</head>
<body>
    <a href="#main" class="skip-link">Skip to content</a>

    {% block header %}
    <header class="site-header">
        <div class="container">
            <a href="/" class="logo">{{ config.Title }}</a>
            <nav class="main-nav">
                <ul>
                    <li><a href="/blog/">Blog</a></li>
                    <li><a href="/tags/">Tags</a></li>
                    <li><a href="/about/">About</a></li>
                </ul>
            </nav>
        </div>
    </header>
    {% endblock %}

    <main id="main">
        <div class="container">
            {% block content %}{% endblock %}
        </div>
    </main>

    {% block footer %}
    <footer class="site-footer">
        <div class="container">
            <p>&copy; {{ config.Author }}. Built with <a href="https://github.com/example/markata-go">markata-go</a>.</p>
        </div>
    </footer>
    {% endblock %}

    {% block scripts %}{% endblock %}
</body>
</html>
```

### post.html

```html
{% extends "base.html" %}

{% block title %}{{ post.Title }} | {{ config.Title }}{% endblock %}
{% block description %}{{ post.Description|default_if_none:config.Description }}{% endblock %}

{% block og_title %}{{ post.Title }}{% endblock %}
{% block og_description %}{{ post.Description|default_if_none:config.Description }}{% endblock %}
{% block og_url %}{{ post.Href }}{% endblock %}

{% block head %}
<meta property="og:type" content="article">
{% if post.Date %}
<meta property="article:published_time" content="{{ post.Date|atom_date }}">
{% endif %}
{% if post.Tags %}
{% for tag in post.Tags %}
<meta property="article:tag" content="{{ tag }}">
{% endfor %}
{% endif %}

{# JSON-LD structured data #}
<script type="application/ld+json">
{
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    "headline": "{{ post.Title }}",
    "description": "{{ post.Description }}",
    "author": {
        "@type": "Person",
        "name": "{{ config.Author }}"
    },
    {% if post.Date %}
    "datePublished": "{{ post.Date|atom_date }}",
    {% endif %}
    "url": "{{ config.URL }}{{ post.Href }}"
}
</script>
{% endblock %}

{% block content %}
<article class="post h-entry">
    <header class="post-header">
        <h1 class="post-title p-name">{{ post.Title }}</h1>

        <div class="post-meta">
            {% if post.Date %}
            <time class="dt-published" datetime="{{ post.Date|atom_date }}">
                {{ post.Date|date_format:"January 2, 2006" }}
            </time>
            {% endif %}

            {% if post.Extra.reading_time %}
            <span class="reading-time">{{ post.Extra.reading_time }}</span>
            {% endif %}
        </div>

        {% if post.Tags %}
        <ul class="post-tags">
            {% for tag in post.Tags %}
            <li>
                <a href="/tags/{{ tag|slugify }}/" rel="tag" class="p-category">{{ tag }}</a>
            </li>
            {% endfor %}
        </ul>
        {% endif %}
    </header>

    <div class="post-content e-content">
        {{ body|safe }}
    </div>

    <footer class="post-footer">
        <div class="author-info">
            <span>Written by</span>
            <a href="/about/" class="p-author h-card" rel="author">{{ config.Author }}</a>
        </div>

        {% if post.Extra.prev or post.Extra.next %}
        <nav class="post-nav">
            {% if post.Extra.prev %}
            <a href="{{ post.Extra.prev.Href }}" class="prev" rel="prev">
                <span>&larr; Previous</span>
                <span class="title">{{ post.Extra.prev.Title }}</span>
            </a>
            {% endif %}
            {% if post.Extra.next %}
            <a href="{{ post.Extra.next.Href }}" class="next" rel="next">
                <span>Next &rarr;</span>
                <span class="title">{{ post.Extra.next.Title }}</span>
            </a>
            {% endif %}
        </nav>
        {% endif %}
    </footer>
</article>
{% endblock %}
```

### feed.html

```html
{% extends "base.html" %}

{% block title %}{{ feed.Title }} | {{ config.Title }}{% endblock %}
{% block description %}{{ feed.Description|default_if_none:config.Description }}{% endblock %}

{% block og_title %}{{ feed.Title }}{% endblock %}
{% block og_url %}/{{ feed.Slug }}/{% endblock %}

{% block head %}
<meta property="og:type" content="website">
{% endblock %}

{% block content %}
<section class="feed">
    <header class="feed-header">
        <h1>{{ feed.Title }}</h1>
        {% if feed.Description %}
        <p class="feed-description">{{ feed.Description }}</p>
        {% endif %}
    </header>

    {% if posts|length > 0 %}
    <ul class="post-list">
        {% for post in posts %}
        <li class="post-item">
            <article class="card h-entry">
                <h2 class="card-title">
                    <a href="{{ post.href }}" class="p-name u-url">{{ post.title }}</a>
                </h2>

                {% if post.description %}
                <p class="card-description p-summary">{{ post.description }}</p>
                {% endif %}

                <footer class="card-meta">
                    {% if post.date %}
                    <time class="dt-published" datetime="{{ post.date|atom_date }}">
                        {{ post.date|date_format:"Jan 2, 2006" }}
                    </time>
                    {% endif %}

                    {% if post.tags %}
                    <ul class="card-tags">
                        {% for tag in post.tags %}
                        <li><a href="/tags/{{ tag|slugify }}/">{{ tag }}</a></li>
                        {% endfor %}
                    </ul>
                    {% endif %}
                </footer>
            </article>
        </li>
        {% endfor %}
    </ul>

    {% if page.HasPrev or page.HasNext %}
    <nav class="pagination" aria-label="Pagination">
        {% if page.HasPrev %}
        <a href="{{ page.PrevURL }}" class="pagination-prev" rel="prev">
            &larr; Newer Posts
        </a>
        {% else %}
        <span class="pagination-prev disabled">&larr; Newer Posts</span>
        {% endif %}

        <span class="pagination-info">Page {{ page.Number }}</span>

        {% if page.HasNext %}
        <a href="{{ page.NextURL }}" class="pagination-next" rel="next">
            Older Posts &rarr;
        </a>
        {% else %}
        <span class="pagination-next disabled">Older Posts &rarr;</span>
        {% endif %}
    </nav>
    {% endif %}

    {% else %}
    <p class="no-posts">No posts found.</p>
    {% endif %}
</section>
{% endblock %}
```

### partials/card.html

```html
<article class="card h-entry">
    {% if post.cover_image %}
    <a href="{{ post.href }}" class="card-image">
        <img src="{{ post.cover_image }}" alt="{{ post.title }}" loading="lazy">
    </a>
    {% endif %}

    <div class="card-body">
        <h2 class="card-title">
            <a href="{{ post.href }}" class="p-name u-url">{{ post.title }}</a>
        </h2>

        {% if post.description %}
        <p class="card-description p-summary">
            {{ post.description|truncate:160 }}
        </p>
        {% endif %}

        <footer class="card-footer">
            {% if post.date %}
            <time class="dt-published" datetime="{{ post.date|atom_date }}">
                {{ post.date|date_format:"Jan 2, 2006" }}
            </time>
            {% endif %}

            {% if post.reading_time %}
            <span class="reading-time">{{ post.reading_time }}</span>
            {% endif %}

            {% if post.tags %}
            <ul class="card-tags">
                {% for tag in post.tags|slice:":3" %}
                <li><a href="/tags/{{ tag|slugify }}/" class="p-category">{{ tag }}</a></li>
                {% endfor %}
            </ul>
            {% endif %}
        </footer>
    </div>
</article>
```

### partials/header.html

```html
<header class="site-header">
    <div class="container">
        <a href="/" class="site-logo">
            {% if config.Extra.logo %}
            <img src="{{ config.Extra.logo }}" alt="{{ config.Title }}">
            {% else %}
            {{ config.Title }}
            {% endif %}
        </a>

        <nav class="main-nav" aria-label="Main navigation">
            <button class="nav-toggle" aria-expanded="false" aria-controls="nav-menu">
                <span class="sr-only">Menu</span>
                <span class="hamburger"></span>
            </button>

            <ul id="nav-menu" class="nav-menu">
                <li><a href="/">Home</a></li>
                <li><a href="/blog/">Blog</a></li>
                <li><a href="/tags/">Tags</a></li>
                <li><a href="/about/">About</a></li>
            </ul>
        </nav>
    </div>
</header>
```

### partials/footer.html

```html
<footer class="site-footer">
    <div class="container">
        <div class="footer-content">
            <div class="footer-section">
                <h3>{{ config.Title }}</h3>
                <p>{{ config.Description }}</p>
            </div>

            <div class="footer-section">
                <h3>Links</h3>
                <ul>
                    <li><a href="/blog/">Blog</a></li>
                    <li><a href="/tags/">Tags</a></li>
                    <li><a href="/about/">About</a></li>
                </ul>
            </div>

            <div class="footer-section">
                <h3>Subscribe</h3>
                <ul>
                    <li><a href="/blog/rss.xml">RSS Feed</a></li>
                    <li><a href="/blog/atom.xml">Atom Feed</a></li>
                    <li><a href="/blog/feed.json">JSON Feed</a></li>
                </ul>
            </div>
        </div>

        <div class="footer-bottom">
            <p>&copy; {{ config.Author }}. Built with <a href="https://github.com/example/markata-go">markata-go</a>.</p>
        </div>
    </div>
</footer>
```

---

## Tips and Best Practices

1. **Always use `|safe` for HTML content** - When outputting rendered HTML (like `body` or `post.ArticleHTML`), use the `safe` filter to prevent double-escaping.

2. **Provide fallbacks with `default_if_none`** - Use fallback values for optional fields to avoid empty output.

3. **Use template inheritance** - Create a solid `base.html` and extend it for consistency across your site.

4. **Keep partials small and focused** - Each partial should do one thing well.

5. **Use semantic HTML** - Include appropriate ARIA attributes and semantic elements for accessibility.

6. **Add microformats** - Include h-entry, h-card, and other microformats for better interoperability.

7. **Cache considerations** - Templates are cached for performance. Clear the cache during development if changes don't appear.

---

## See Also

- [[configuration-guide|Configuration Guide]] - Configure template settings
- [[feeds-guide|Feeds Guide]] - Learn about the feed system
- [[themes-and-styling|Themes Guide]] - Create and customize themes
- [pongo2 Documentation](https://github.com/flosch/pongo2) - Full template engine reference
