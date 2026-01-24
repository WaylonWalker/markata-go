---
title: "Getting Started"
description: "Complete guide to installing markata-go, creating your first site, and building for production"
date: 2024-01-15
published: true
tags:
  - documentation
  - getting-started
  - tutorial
---

# Getting Started with markata-go

This guide will walk you through installing markata-go, creating your first site, and building it for production.

## Installation

markata-go is distributed as a single binary with no dependencies. Choose the installation method that works best for you.

### Quick Install (Recommended)

```bash
# One-liner install script (Linux/macOS)
curl -sSL https://waylonwalker.github.io/markata-go/install.sh | bash

# Using jpillora/installer (Linux/macOS)
curl -sL https://i.jpillora.com/WaylonWalker/markata-go | bash

# Using eget
eget WaylonWalker/markata-go

# Using mise (installs from GitHub releases)
mise use -g github:WaylonWalker/markata-go
```

### Go Install

If you have Go 1.22+ installed:

```bash
go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
```

Note: This installs to `$GOPATH/bin` (usually `~/go/bin`). Ensure this is in your `PATH`.

### Manual Download

Download pre-built binaries from [GitHub Releases](https://github.com/WaylonWalker/markata-go/releases).

Available platforms: Linux (amd64, arm64, armv7), macOS (Intel, Apple Silicon), Windows (amd64), FreeBSD (amd64), Android (arm64/Termux).

See [Installation Guide](installation.md) for detailed installation instructions.

### Verify Installation

```bash
markata-go version
markata-go --help
```

## Quick Start

Get a site up and running in under a minute:

```bash
# 1. Create a project directory
mkdir my-site && cd my-site

# 2. Initialize configuration
markata-go config init

# 3. Create your first post
markata-go new "Hello World"

# 4. Build and serve with live reload
markata-go serve
```

Open http://localhost:8000 to see your site.

## Project Structure

A typical markata-go project looks like this:

```
my-site/
├── markata-go.toml      # Configuration file
├── posts/               # Your markdown content
│   └── hello-world.md
├── pages/               # Static pages (about, contact, etc.)
├── templates/           # Custom templates (optional)
├── static/              # Static assets (images, CSS, JS)
└── public/              # Generated output (created on build)
```

### Directory Overview

| Directory | Purpose |
|-----------|---------|
| `posts/` | Blog posts and articles with dates |
| `pages/` | Static pages without dates |
| `templates/` | Custom HTML templates (overrides theme defaults) |
| `static/` | Static assets copied directly to output |
| `public/` | Generated site (default output directory) |

## Configuration

markata-go uses TOML for configuration. The `config init` command creates a `markata-go.toml` file:

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

# Content discovery
[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
use_gitignore = true

# Markdown extensions
[markata-go.markdown]
extensions = ["tables", "strikethrough", "autolinks", "tasklist"]
```

### Key Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `title` | - | Site title (used in templates and feeds) |
| `url` | - | Base URL for absolute links and sitemaps |
| `output_dir` | `"output"` | Where generated files are written |
| `templates_dir` | `"templates"` | Custom template directory |
| `assets_dir` | `"static"` | Static assets directory |
| `concurrency` | `0` | Worker threads (0 = auto based on CPU) |

### Environment Variable Overrides

Override any config option with environment variables using the `MARKATA_GO_` prefix:

```bash
MARKATA_GO_OUTPUT_DIR=dist markata-go build
MARKATA_GO_URL=https://staging.example.com markata-go build
```

## Creating Content

### Your First Post

Create a new post with the `new` command:

```bash
markata-go new "My First Post"
# Creates: posts/my-first-post.md
```

This generates a markdown file with YAML frontmatter:

```markdown
---
title: "My First Post"
slug: "my-first-post"
date: 2026-01-21
published: false
draft: true
tags: []
template: "post.html"
---

# My First Post

Write your content here...
```

### Frontmatter Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | - | Post title |
| `slug` | string | auto | URL-safe identifier (auto-generated from title) |
| `date` | date | - | Publication date |
| `published` | bool | `false` | Whether the post is publicly visible |
| `draft` | bool | `false` | Mark as work-in-progress |
| `tags` | []string | `[]` | List of tags for categorization |
| `description` | string | auto | Meta description (auto-generated if not set) |
| `template` | string | `post.html` | Template file to use |

Any additional fields are stored in `Extra` and accessible in templates.

### Example Post

```markdown
---
title: "Getting Started with Go"
date: 2026-01-21
published: true
tags: ["go", "tutorial", "programming"]
description: "Learn the basics of Go programming"
featured: true
---

Go is a statically typed, compiled language designed for simplicity and efficiency.

## Installation

Download Go from the official website...

## Hello World

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

## Next Steps

Check out the [[go-concurrency|concurrency guide]] for more advanced topics.
```

### Markdown Features

markata-go supports GitHub Flavored Markdown plus extensions:

- **Tables** - GFM table syntax
- **Strikethrough** - `~~deleted text~~`
- **Task lists** - `- [ ] todo` and `- [x] done`
- **Syntax highlighting** - Fenced code blocks with language
- **Admonitions** - Note, warning, tip blocks
- **Wikilinks** - `[[other-post]]` internal links
- **Table of contents** - Auto-generated from headings

#### Admonitions

```markdown
!!! note "Important"
    This is a note admonition.

!!! warning
    Be careful with this operation.

!!! tip "Pro Tip"
    Use wikilinks for internal navigation.
```

#### Wikilinks

Link to other posts using their slug:

```markdown
Check out [[my-other-post]] for more details.
Or with custom text: [[my-other-post|click here]]
```

## Development Server

Start the development server with live reload:

```bash
markata-go serve
```

The server:
- Watches for file changes
- Automatically rebuilds on save
- Refreshes the browser

### Server Options

```bash
markata-go serve              # Default: localhost:8000
markata-go serve -p 3000      # Custom port
markata-go serve --host 0.0.0.0  # Bind to all interfaces
markata-go serve --no-watch   # Disable file watching
markata-go serve -v           # Verbose logging
```

## Building for Production

Build your site for deployment:

```bash
markata-go build
```

This generates static files in the `output_dir` (default: `public/`).

### Build Options

```bash
markata-go build              # Standard build
markata-go build --clean      # Clean output directory first
markata-go build --dry-run    # Preview what would be built
markata-go build -o dist      # Custom output directory
markata-go build -v           # Verbose output
```

### Output Structure

```
public/
├── index.html           # Home page (from feed)
├── my-first-post/
│   └── index.html       # Post page
├── blog/
│   ├── index.html       # Blog feed page 1
│   ├── page/2/index.html
│   ├── rss.xml          # RSS feed
│   └── atom.xml         # Atom feed
├── tags/
│   └── go/
│       └── index.html   # Tag archive
├── sitemap.xml
└── css/
    └── ...              # Static assets
```

## Feeds

markata-go has a powerful feed system for creating archives, tag pages, and syndication feeds.

### Basic Feed Configuration

```toml
# Home page feed
[[markata-go.feeds]]
slug = ""                    # Empty slug = root index.html
title = "Latest Posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 5

# Blog archive with RSS
[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true
atom = true
```

### Auto-Generated Tag Feeds

Enable automatic tag pages:

```toml
[markata-go.feeds.auto_tags]
enabled = true
slug_prefix = "tags"
```

This creates `/tags/{tag-name}/` pages for each tag in your posts.

## CLI Reference

### Commands

| Command | Description |
|---------|-------------|
| `markata-go build` | Build the static site |
| `markata-go serve` | Development server with live reload |
| `markata-go new <title>` | Create a new post |
| `markata-go config show` | Display resolved configuration |
| `markata-go config init` | Create a new config file |
| `markata-go config validate` | Validate configuration |
| `markata-go config get <key>` | Get a specific config value |

### Global Flags

```bash
-c, --config string   Config file path (default: auto-discover)
-o, --output string   Output directory (overrides config)
-v, --verbose         Verbose output
```

### Examples

```bash
# Create a draft post
markata-go new "Work in Progress" --draft

# Create in a specific directory
markata-go new "About Me" --dir pages

# Create as published
markata-go new "Announcement" --draft=false

# Show config as JSON
markata-go config show --json

# Get nested config value
markata-go config get glob.patterns
```

## Templates

markata-go uses [pongo2](https://github.com/flosch/pongo2), a Django/Jinja2-like template engine.

### Template Variables

In post templates (`post.html`):

| Variable | Description |
|----------|-------------|
| `post.Title` | Post title |
| `post.Slug` | URL slug |
| `post.Href` | Relative URL path |
| `post.Date` | Publication date |
| `post.Tags` | List of tags |
| `post.Description` | Post description |
| `post.ArticleHTML` | Rendered content HTML |
| `post.Extra` | Additional frontmatter fields |
| `config.Title` | Site title |
| `config.URL` | Site base URL |

### Custom Templates

Override default templates by creating files in your `templates/` directory:

```
templates/
├── base.html        # Base layout
├── post.html        # Single post
├── feed.html        # Feed/archive pages
└── partials/
    ├── header.html
    └── footer.html
```

### Template Example

```html
{% extends "base.html" %}

{% block content %}
<article>
  <h1>{{ post.Title }}</h1>
  <time>{{ post.Date|date:"January 2, 2006" }}</time>
  
  <div class="content">
    {{ post.ArticleHTML|safe }}
  </div>
  
  {% if post.Tags %}
  <div class="tags">
    {% for tag in post.Tags %}
    <a href="/tags/{{ tag|slugify }}/">{{ tag }}</a>
    {% endfor %}
  </div>
  {% endif %}
</article>
{% endblock %}
```

## Deployment

The `public/` directory contains static files ready for any hosting platform.

### Popular Hosting Options

**Netlify:**
```bash
# netlify.toml
[build]
  command = "markata-go build"
  publish = "public"
```

**Vercel:**
```json
{
  "buildCommand": "markata-go build",
  "outputDirectory": "public"
}
```

**GitHub Pages:**
```yaml
# .github/workflows/deploy.yml
- name: Build
  run: markata-go build
- name: Deploy
  uses: peaceiris/actions-gh-pages@v3
  with:
    publish_dir: ./public
```

**Manual:**
```bash
# Build and upload
markata-go build
rsync -avz public/ user@server:/var/www/html/
```

## Next Steps

- **[[configuration-guide|Configuration Reference]]** - Full config options
- **[[templates-guide|Templates Guide]]** - Custom templates and filters
- **[[feeds-guide|Feeds System]]** - Advanced feed configuration
- **[[built-in-plugins|Plugins]]** - Built-in plugins and customization
- **[[markdown-features|Markdown Features]]** - Admonitions, wikilinks, and more

## Getting Help

- **GitHub Issues** - Report bugs and request features
- **Documentation** - Full reference at `/docs`
- **Examples** - Sample sites in `/examples`
