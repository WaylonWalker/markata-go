# markata-go

A fast, plugin-driven static site generator with a powerful feed system.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

markata-go is a static site generator written in Go that processes Markdown files with YAML frontmatter and generates a complete static website. It features a flexible plugin architecture, a powerful feed system for creating archives and syndication feeds, and Jinja2-like templating.

### Key Features

- **Plugin-driven architecture** - Extensible system with 14+ built-in plugins
- **Powerful feed system** - Define feeds with filtering, sorting, and pagination; generate multiple output formats (HTML, RSS, Atom, JSON) from a single definition
- **Jinja2-like templates** - Familiar template syntax via pongo2 with custom filters
- **9-stage lifecycle** - Configure, Validate, Glob, Load, Transform, Render, Collect, Write, Cleanup
- **Concurrent processing** - Parallel processing with configurable worker count
- **Markdown with extensions** - GFM tables, strikethrough, task lists, admonitions, syntax highlighting, wikilinks, and table of contents generation
- **Live reload development server** - Built-in server with file watching and automatic rebuilds

## Quick Start

```bash
# Install
go install github.com/example/markata-go/cmd/markata-go@latest

# Create a new post
markata-go new "Hello World"

# Build the site
markata-go build

# Serve with live reload
markata-go serve
```

## Configuration

markata-go uses TOML configuration (also supports YAML and JSON). Create a `markata-go.toml` in your project root:

```toml
[markata-go]
# Site metadata
title = "My Site"
description = "A site built with markata-go"
url = "https://example.com"
author = "Your Name"

# Build settings
output_dir = "public"
templates_dir = "templates"
assets_dir = "static"

# Concurrency (0 = auto based on CPU cores)
concurrency = 0

# Plugin configuration
hooks = ["default"]
disabled_hooks = []

# Content discovery
[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
use_gitignore = true

# Markdown extensions
[markata-go.markdown]
extensions = ["tables", "strikethrough", "autolinks", "tasklist"]

# Reading time calculation
[markata-go.reading_time]
words_per_minute = 200

# Auto-generated descriptions
[markata-go.description]
max_length = 160

# Table of contents
[markata-go.toc]
min_level = 2
max_level = 4

# Wikilinks
[markata-go.wikilinks]
warn_broken = true
```

### Environment Variable Overrides

All configuration options can be overridden via environment variables using the `MARKATA_GO_` prefix:

```bash
MARKATA_GO_OUTPUT_DIR=dist markata-go build
MARKATA_GO_URL=https://staging.example.com markata-go build
MARKATA_GO_CONCURRENCY=4 markata-go build
```

## Content

### Frontmatter Format

Posts use YAML frontmatter to define metadata:

```yaml
---
title: "My First Post"
slug: "my-first-post"
date: 2024-01-15
published: true
draft: false
tags: ["go", "static-site"]
description: "An introduction to markata-go"
template: "post.html"
---
```

### Supported Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | - | Post title |
| `slug` | string | auto | URL-safe identifier (auto-generated from title if not set) |
| `date` | date | - | Publication date |
| `published` | bool | `false` | Whether the post is published |
| `draft` | bool | `false` | Whether the post is a draft |
| `tags` | []string | `[]` | List of tags |
| `description` | string | auto | Post description (auto-generated if not set) |
| `template` | string | `post.html` | Template file to use |

Any additional fields in frontmatter are stored in `Extra` and accessible in templates.

### Example Post

```markdown
---
title: "Getting Started with markata-go"
date: 2024-01-15
published: true
tags: ["tutorial", "go"]
featured: true
---

# Getting Started with markata-go

This is a guide to building your first site with markata-go.

## Installation

Install markata-go using Go:

```bash
go install github.com/example/markata-go/cmd/markata-go@latest
```

## Creating Content

Create markdown files in your `posts/` directory...
```

## Feed System

The feed system is markata-go's most powerful feature. Define feeds to create filtered, sorted, paginated collections of posts with multiple output formats.

### Feed Configuration

```toml
# Feed defaults - inherited by all feeds
[markata-go.feeds.defaults]
items_per_page = 10
orphan_threshold = 3

[markata-go.feeds.defaults.formats]
html = true
rss = true
atom = false
json = false

[markata-go.feeds.defaults.templates]
html = "feed.html"
card = "partials/card.html"

# Syndication settings (RSS/Atom/JSON)
[markata-go.feeds.syndication]
max_items = 20
include_content = false

# Auto-generated tag feeds
[markata-go.feeds.auto_tags]
enabled = true
slug_prefix = "tags"

[markata-go.feeds.auto_tags.formats]
html = true
rss = true
```

### Defining Feeds

```toml
# Main blog feed
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
description = "All blog posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 10

[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = true

# Home page (empty slug = root index.html)
[[markata-go.feeds]]
slug = ""
title = "Latest Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5

[markata-go.feeds.formats]
html = true

# Featured posts
[[markata-go.feeds]]
slug = "featured"
title = "Featured"
description = "Featured posts"
filter = "published == True and featured == True"
sort = "date"
reverse = true
```

### Multiple Output Formats

Each feed can generate multiple formats from a single definition:

| Format | Output | Description |
|--------|--------|-------------|
| `html` | `/{slug}/index.html` | Paginated HTML pages |
| `rss` | `/{slug}/rss.xml` | RSS 2.0 feed |
| `atom` | `/{slug}/atom.xml` | Atom 1.0 feed |
| `json` | `/{slug}/feed.json` | JSON Feed |
| `markdown` | `/{slug}/index.md` | Markdown output |
| `text` | `/{slug}/index.txt` | Plain text output |

### Auto-Generated Tag Feeds

When `auto_tags.enabled = true`, markata-go automatically creates feeds for each tag found in your posts:

```
/tags/go/index.html
/tags/go/rss.xml
/tags/tutorial/index.html
/tags/tutorial/rss.xml
```

## Templates

markata-go uses [pongo2](https://github.com/flosch/pongo2), a Django/Jinja2-like template engine for Go.

### Template Syntax

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ post.Title|default_if_none:config.Title }}</title>
    <meta name="description" content="{{ post.Description }}">
</head>
<body>
    <article>
        <h1>{{ post.Title }}</h1>
        <time datetime="{{ post.Date|atom_date }}">
            {{ post.Date|date_format:"January 2, 2006" }}
        </time>
        
        {% if post.Tags %}
        <div class="tags">
            {% for tag in post.Tags %}
            <a href="/tags/{{ tag|slugify }}/">{{ tag }}</a>
            {% endfor %}
        </div>
        {% endif %}
        
        <div class="content">
            {{ post.ArticleHTML|safe }}
        </div>
    </article>
</body>
</html>
```

### Available Variables

#### Post Context (`post`)
- `post.Title` - Post title
- `post.Slug` - URL slug
- `post.Href` - Relative URL path (e.g., `/my-post/`)
- `post.Date` - Publication date
- `post.Published` - Whether published
- `post.Draft` - Whether draft
- `post.Tags` - List of tags
- `post.Description` - Post description
- `post.Content` - Raw markdown content
- `post.ArticleHTML` - Rendered HTML content
- `post.HTML` - Full rendered HTML with template
- `post.Extra` - Additional frontmatter fields

#### Config Context (`config`)
- `config.Title` - Site title
- `config.Description` - Site description
- `config.URL` - Site base URL
- `config.Author` - Site author

#### Feed Context (`feed`)
- `feed.Title` - Feed title
- `feed.Description` - Feed description
- `feed.Slug` - Feed slug
- `feed.Posts` - List of posts in feed
- `feed.Pages` - Paginated pages
- `feed.Pages[n].Number` - Page number
- `feed.Pages[n].Posts` - Posts on page
- `feed.Pages[n].HasPrev` / `HasNext` - Pagination flags
- `feed.Pages[n].PrevURL` / `NextURL` - Pagination URLs

### Built-in Filters

#### Date Formatting
- `{{ date|rss_date }}` - RFC 1123Z format for RSS
- `{{ date|atom_date }}` - RFC 3339 format for Atom
- `{{ date|date_format:"2006-01-02" }}` - Custom Go date format

#### String Manipulation
- `{{ text|slugify }}` - Convert to URL-safe slug
- `{{ text|truncate:100 }}` - Truncate to character limit
- `{{ text|truncatewords:20 }}` - Truncate to word limit

#### Collections
- `{{ list|length }}` - Get length
- `{{ list|first }}` - Get first element
- `{{ list|last }}` - Get last element
- `{{ list|join:", " }}` - Join with separator
- `{{ list|reverse }}` - Reverse order
- `{{ list|sort }}` - Sort alphabetically

#### HTML/Text
- `{{ html|striptags }}` - Remove HTML tags
- `{{ text|linebreaks }}` - Convert newlines to `<p>` and `<br>`
- `{{ text|linebreaksbr }}` - Convert newlines to `<br>`

#### URLs
- `{{ path|urlencode }}` - URL encode
- `{{ path|absolute_url:config.URL }}` - Convert to absolute URL

#### Default Values
- `{{ value|default_if_none:"fallback" }}` - Provide fallback for nil/empty

### Template Inheritance

```html
{# base.html #}
<!DOCTYPE html>
<html>
<head>
    <title>{% block title %}{{ config.Title }}{% endblock %}</title>
</head>
<body>
    {% block content %}{% endblock %}
</body>
</html>
```

```html
{# post.html #}
{% extends "base.html" %}

{% block title %}{{ post.Title }} | {{ config.Title }}{% endblock %}

{% block content %}
<article>
    {{ post.ArticleHTML|safe }}
</article>
{% endblock %}
```

## Plugins

### Built-in Plugins

| Plugin | Stage | Description |
|--------|-------|-------------|
| `glob` | Glob | Discovers content files using glob patterns |
| `load` | Load | Parses markdown files and frontmatter |
| `frontmatter` | Load | Extracts YAML frontmatter |
| `description` | Transform | Auto-generates descriptions from content |
| `reading_time` | Transform | Calculates reading time |
| `wikilinks` | Transform | Processes `[[wikilink]]` syntax |
| `toc` | Transform | Generates table of contents |
| `jinja_md` | Transform | Processes Jinja templates in markdown |
| `admonitions` | Transform | Converts admonition blocks |
| `render_markdown` | Render | Converts markdown to HTML |
| `templates` | Render | Applies HTML templates |
| `feeds` | Collect | Builds feed collections |
| `auto_feeds` | Collect | Generates tag feeds |
| `publish_feeds` | Write | Writes feed output files |
| `publish_html` | Write | Writes HTML post files |
| `sitemap` | Write | Generates sitemap.xml |

### Plugin Lifecycle Stages

```
1. Configure  - Load config and initialize plugins
2. Validate   - Validate configuration
3. Glob       - Discover content files
4. Load       - Parse files into posts
5. Transform  - Pre-render processing (jinja-md, wikilinks, etc.)
6. Render     - Convert markdown to HTML
7. Collect    - Build feeds and navigation
8. Write      - Output files to disk
9. Cleanup    - Release resources
```

### Disabling Plugins

```toml
[markata-go]
disabled_hooks = ["sitemap", "auto_feeds"]
```

## CLI Commands

### `markata-go build`

Build the static site.

```bash
markata-go build              # Standard build
markata-go build --clean      # Clean output directory first
markata-go build --dry-run    # Show what would be built
markata-go build -v           # Verbose output
markata-go build -o dist      # Custom output directory
```

### `markata-go serve`

Development server with live reload.

```bash
markata-go serve              # Serve on localhost:8000
markata-go serve -p 3000      # Custom port
markata-go serve --host 0.0.0.0  # Bind to all interfaces
markata-go serve --no-watch   # Disable file watching
markata-go serve -v           # Verbose logging
```

### `markata-go new <title>`

Create a new post.

```bash
markata-go new "My First Post"         # Creates posts/my-first-post.md
markata-go new "Hello World" --dir blog  # Creates blog/hello-world.md
markata-go new "Draft" --draft          # Create as draft (default)
markata-go new "Published" --draft=false # Create as published
```

### `markata-go config`

Configuration management commands.

```bash
markata-go config show           # Show resolved config (YAML)
markata-go config show --json    # Show as JSON
markata-go config show --toml    # Show as TOML

markata-go config get output_dir          # Get specific value
markata-go config get glob.patterns       # Nested values with dot notation
markata-go config get feed_defaults.items_per_page

markata-go config validate       # Validate configuration
markata-go config validate -c custom.toml

markata-go config init           # Create markata-go.toml
markata-go config init site.yaml # Create YAML config
markata-go config init --force   # Overwrite existing
```

### Global Flags

```bash
-c, --config string   Config file path (default: auto-discover)
-o, --output string   Output directory (overrides config)
-v, --verbose         Verbose output
```

## Advanced Topics

### Jinja in Markdown

The `jinja_md` plugin allows you to use Jinja2 template syntax within your markdown content:

```markdown
---
title: "Dynamic Content"
items:
  - First item
  - Second item
  - Third item
---

# {{ post.Title }}

Published on {{ post.Date|date_format:"January 2, 2006" }}

## Items

{% for item in post.Extra.items %}
- {{ item }}
{% endfor %}

{% if post.Tags|length > 0 %}
Tags: {{ post.Tags|join:", " }}
{% endif %}
```

### Wikilinks

Link to other posts using wikilink syntax:

```markdown
Check out my [[other-post|Other Post]] for more details.

Or use the slug directly: [[getting-started]]
```

### Custom Templates

Override the default template for specific posts:

```yaml
---
title: "Landing Page"
template: "landing.html"
---
```

### Filter Expressions

Feeds use a Python-like filter expression syntax:

```toml
# Boolean comparisons
filter = "published == True"
filter = "draft == False"

# String comparisons
filter = "slug == 'about'"

# Tag filtering (using 'in' operator)
filter = "'tutorial' in tags"

# Combined expressions
filter = "published == True and featured == True"
filter = "published == True and 'go' in tags"

# Date comparisons
filter = "date >= '2024-01-01'"

# Negation
filter = "not draft"
```

## Project Structure

```
markata-go/
├── cmd/
│   └── markata-go/
│       ├── main.go           # Entry point
│       └── cmd/
│           ├── root.go       # Root command and global flags
│           ├── build.go      # Build command
│           ├── serve.go      # Serve command with live reload
│           ├── new.go        # New post command
│           └── config.go     # Config management commands
├── pkg/
│   ├── models/               # Data models
│   │   ├── post.go          # Post struct
│   │   ├── config.go        # Config struct
│   │   ├── feed.go          # Feed and pagination structs
│   │   └── errors.go        # Error types
│   ├── filter/               # Filter expression parser
│   │   ├── lexer.go         # Tokenizer
│   │   ├── parser.go        # Expression parser
│   │   ├── evaluator.go     # Expression evaluator
│   │   └── filter.go        # High-level filter API
│   ├── config/               # Configuration loading
│   │   ├── loader.go        # Config file discovery and loading
│   │   ├── parser.go        # TOML/YAML/JSON parsing
│   │   ├── merge.go         # Config merging
│   │   ├── env.go           # Environment variable overrides
│   │   ├── validate.go      # Config validation
│   │   └── defaults.go      # Default values
│   ├── lifecycle/            # Build lifecycle manager
│   │   ├── manager.go       # Lifecycle orchestration
│   │   ├── stages.go        # Stage definitions
│   │   ├── plugin.go        # Plugin interface
│   │   └── hooks.go         # Hook management
│   ├── plugins/              # Built-in plugins
│   │   ├── registry.go      # Plugin registration
│   │   ├── glob.go          # File discovery
│   │   ├── load.go          # File loading
│   │   ├── frontmatter.go   # Frontmatter parsing
│   │   ├── render_markdown.go # Markdown rendering
│   │   ├── templates.go     # Template processing
│   │   ├── feeds.go         # Feed building
│   │   ├── auto_feeds.go    # Auto tag feeds
│   │   ├── publish_feeds.go # Feed output
│   │   ├── publish_html.go  # HTML output
│   │   ├── jinja_md.go      # Jinja in markdown
│   │   ├── wikilinks.go     # Wikilink processing
│   │   ├── toc.go           # Table of contents
│   │   ├── description.go   # Auto descriptions
│   │   ├── reading_time.go  # Reading time calc
│   │   ├── admonitions.go   # Admonition blocks
│   │   ├── sitemap.go       # Sitemap generation
│   │   ├── rss.go           # RSS feed generation
│   │   ├── atom.go          # Atom feed generation
│   │   └── jsonfeed.go      # JSON feed generation
│   └── templates/            # Template engine
│       ├── engine.go        # Pongo2 wrapper
│       ├── context.go       # Template context
│       └── filters.go       # Custom filters
├── markata-go.toml           # Example configuration
└── go.mod                    # Go module definition
```

## License

MIT License

Copyright (c) 2024

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
