---
title: "Frontmatter Guide"
description: "Complete guide to post metadata, built-in fields, and custom frontmatter in markata-go"
date: 2024-01-15
published: true
template: doc.html
tags:
  - documentation
  - frontmatter
  - content
---

# Frontmatter

Frontmatter is the **metadata block** at the top of your Markdown files that tells markata-go how to process, display, and organize your content. This guide covers everything you need to know about using frontmatter effectively.

## Table of Contents

- [What is Frontmatter?](#what-is-frontmatter)
- [Basic Frontmatter Fields](#basic-frontmatter-fields)
- [Complete Field Reference](#complete-field-reference)
- [Custom Fields (Extra)](#custom-fields-extra)
- [Examples](#examples)
- [Common Patterns](#common-patterns)
- [Frontmatter in Filtering](#frontmatter-in-filtering)

---

## What is Frontmatter?

Frontmatter is **YAML metadata** placed at the very top of a Markdown file, enclosed between two lines of three dashes (`---`). It defines properties like the post's title, date, tags, and any custom data you want to associate with the content.

### Visual Example

```markdown
---
title: "Getting Started with Go"
date: 2024-01-15
published: true
tags:
  - go
  - tutorial
  - beginners
description: "A beginner-friendly introduction to Go programming."
---

# Welcome to Go!

This is where your actual content begins...
```

### How markata-go Parses Frontmatter

When markata-go processes your Markdown files, it:

1. **Detects the opening delimiter** - Looks for `---` at the very start of the file
2. **Extracts the YAML block** - Reads everything until the closing `---`
3. **Parses the YAML** - Converts the metadata into structured data
4. **Separates the content** - Everything after the second `---` becomes the post body

**Important notes:**

- The opening `---` must be on the very first line of the file
- The closing `---` must be on its own line
- Content before the opening `---` or malformed delimiters will cause errors
- Files without frontmatter are valid - they'll use default values

### Valid Frontmatter

```markdown
---
title: My Post
---
Content here
```

### Invalid Frontmatter (Missing Closing Delimiter)

```markdown
---
title: My Post

Content here (ERROR: unclosed frontmatter)
```

### No Frontmatter (Also Valid)

```markdown
# My Post

Content starts immediately - defaults will be used.
```

---

## Basic Frontmatter Fields

These are the core fields that markata-go recognizes and uses directly.

### title (string)

The display title of your post.

```yaml
title: "Understanding Goroutines"
```

- **Type:** string
- **Default:** None (derived from filename if not set)
- **Used for:** Page title, `<title>` tag, feed listings, slug generation

If not provided, the slug is derived from the filename instead.

### slug (string)

The URL-safe identifier for your post. Determines the URL path.

```yaml
slug: "understanding-goroutines"
```

- **Type:** string
- **Default:** Auto-generated from `title` (or filename if no title)
- **Used for:** URL path (`/understanding-goroutines/`)

**Auto-generation rules:**
- Converts to lowercase
- Replaces spaces with hyphens
- Removes special characters
- Collapses multiple hyphens

```yaml
# These titles produce these slugs:
title: "Hello World!"        # slug: hello-world
title: "Go's Best Features"  # slug: gos-best-features
title: "Part 1: The Basics"  # slug: part-1-the-basics
```

### date (date)

The publication date of the post.

```yaml
date: 2024-01-15
```

- **Type:** date (YYYY-MM-DD format recommended)
- **Default:** None
- **Used for:** Sorting, display, feeds, scheduled publishing

**Supported date formats:**

markata-go supports a wide variety of date formats for maximum flexibility:

```yaml
# ISO 8601 formats (recommended)
date: 2024-01-15                    # Date only
date: 2024-01-15T10:30:00           # With time
date: 2024-01-15T10:30:00Z          # With UTC timezone
date: 2024-01-15T10:30:00+05:00     # With timezone offset
date: 2024-01-15 10:30:00           # With space separator
date: 2024-01-15 10:30              # Without seconds

# Single-digit hours (automatically normalized)
date: 2024-01-15 1:30:00            # 1am
date: 2024-01-15 9:30:00            # 9am

# Slash-separated dates
date: 2024/01/15                    # YYYY/MM/DD
date: 2024/01/15 10:30:00           # With time
date: 01/15/2024                    # MM/DD/YYYY (US format)

# Named month formats
date: January 15, 2024              # Full month name
date: Jan 15, 2024                  # Abbreviated month
date: 15 January 2024               # Day first
date: 15 Jan 2024                   # Day first, abbreviated

# European format
date: 15-01-2024                    # DD-MM-YYYY
```

**Note:** Malformed time components are automatically corrected. For example, `2024-01-15 8:011:00` (typo) will be parsed as `2024-01-15 08:11:00`.

### published (boolean)

Whether the post should be included in public feeds and listings.

```yaml
published: true
```

- **Type:** boolean
- **Default:** `false`
- **Used for:** Filtering posts in feeds

**Accepted values:**

```yaml
published: true    # or: yes, on
published: false   # or: no, off
```

### draft (boolean)

Marks the post as a work-in-progress.

```yaml
draft: true
```

- **Type:** boolean
- **Default:** `false`
- **Used for:** Filtering, visual indicators in templates

**Note:** `draft: true` doesn't automatically exclude posts from builds. Use `published: false` or feed filters to exclude drafts.

### tags (list)

A list of tags for categorizing the post.

```yaml
tags:
  - go
  - concurrency
  - tutorial
```

Or inline format:

```yaml
tags: [go, concurrency, tutorial]
```

- **Type:** list of strings
- **Default:** `[]` (empty list)
- **Used for:** Filtering, tag pages, SEO, organization

### description (string)

A brief summary of the post content.

```yaml
description: "Learn how to use goroutines for concurrent programming in Go."
```

- **Type:** string
- **Default:** Auto-generated from first ~160 characters of content
- **Used for:** Meta description, feed summaries, social previews, cards

**Auto-generation:** If not provided, markata-go extracts the first paragraph or ~160 characters from your content (with HTML tags stripped).

### template (string)

The HTML template file to use for rendering this post.

```yaml
template: "tutorial.html"
```

- **Type:** string
- **Default:** `"post.html"`
- **Used for:** Custom layouts per post

Templates are looked up in:
1. `templates/` directory in your project
2. Theme templates
3. Default theme fallback

---

## Complete Field Reference

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `title` | string | None | No | Display title of the post |
| `slug` | string | Auto-generated | No | URL path identifier |
| `date` | date | None | No | Publication date (YYYY-MM-DD) |
| `published` | bool | `false` | No | Whether to include in public feeds |
| `draft` | bool | `false` | No | Whether this is a work-in-progress |
| `tags` | []string | `[]` | No | List of categorization tags |
| `description` | string | Auto-generated | No | Brief summary for SEO/feeds |
| `template` | string | `"post.html"` | No | Template file to use for rendering |
| `skip` | bool | `false` | No | Skip this file during processing entirely |

### Field Details

#### skip

Completely exclude a file from processing:

```yaml
skip: true
```

Use this for files you want to keep in your content directory but never process (notes, drafts not ready for review, etc.).

---

## Custom Fields (Extra)

Any frontmatter field that isn't a built-in field is automatically stored in the `Extra` map. This allows you to add **any custom metadata** to your posts.

### Adding Custom Fields

```yaml
---
title: "Building a REST API"
date: 2024-01-15
published: true

# Custom fields - stored in Extra
author: "Jane Doe"
category: "Backend"
series: "API Development"
series_order: 1
featured: true
cover_image: "/images/api-cover.jpg"
difficulty: "intermediate"
reading_time: "8 min"
---
```

### Accessing Custom Fields in Templates

Custom fields are available via `post.Extra` in templates:

```html
{% if post.Extra.featured %}
<span class="badge">Featured</span>
{% endif %}

{% if post.Extra.author %}
<p class="author">By {{ post.Extra.author }}</p>
{% endif %}

{% if post.Extra.cover_image %}
<img src="{{ post.Extra.cover_image }}" alt="{{ post.Title }}">
{% endif %}

{% if post.Extra.series %}
<div class="series-info">
    Part {{ post.Extra.series_order }} of {{ post.Extra.series }}
</div>
{% endif %}
```

### Common Custom Field Use Cases

#### Author Information

```yaml
author: "Jane Doe"
author_email: "jane@example.com"
author_url: "https://janedoe.dev"
author_avatar: "/images/avatars/jane.jpg"
```

#### Series/Collections

```yaml
series: "Building a Blog with Go"
series_order: 3
series_total: 5
```

#### Visual Elements

```yaml
cover_image: "/images/posts/my-cover.jpg"
thumbnail: "/images/posts/my-thumb.jpg"
og_image: "/images/social/my-post-og.jpg"
hero_video: "https://youtube.com/watch?v=..."
```

#### Content Metadata

```yaml
difficulty: "beginner"          # beginner, intermediate, advanced
reading_time: "5 min read"
word_count: 1250
updated: 2024-02-20
revision: 3
```

#### Categorization

```yaml
category: "Tutorials"
subcategory: "Web Development"
topic: "Go Programming"
```

#### Flags and Features

```yaml
featured: true
pinned: true
sponsored: false
comments_enabled: true
toc: true                       # Enable table of contents
math: true                      # Enable math rendering
mermaid: true                   # Enable diagrams
```

#### External Links

```yaml
canonical_url: "https://example.com/original-post"
github_repo: "https://github.com/user/project"
demo_url: "https://demo.example.com"
```

---

## Examples

### Minimal Frontmatter

The absolute minimum for a publishable post:

```yaml
---
title: "Hello World"
published: true
---

Your content here.
```

### Full Frontmatter Example

A comprehensive example using all built-in fields:

```yaml
---
title: "Complete Guide to Error Handling in Go"
slug: "go-error-handling-guide"
date: 2024-01-15
published: true
draft: false
tags:
  - go
  - error-handling
  - best-practices
  - tutorial
description: "Master error handling in Go with this comprehensive guide covering best practices, custom errors, and common patterns."
template: "tutorial.html"
---

Content begins here...
```

### Blog Post Example

A typical blog post with custom fields:

```yaml
---
title: "Why I Switched from Python to Go"
date: 2024-01-15
published: true
tags:
  - go
  - python
  - opinion
  - programming
description: "My journey from Python to Go and the lessons learned along the way."

# Custom fields
author: "Alex Chen"
category: "Opinion"
featured: true
cover_image: "/images/python-to-go.jpg"
reading_time: "6 min read"
---

# Why I Switched from Python to Go

After 5 years of Python development, I made the switch to Go...
```

### Documentation Page Example

For technical documentation:

```yaml
---
title: "Configuration Reference"
slug: "docs/configuration"
published: true
tags:
  - documentation
  - reference
description: "Complete reference for markata-go configuration options."
template: "docs.html"

# Custom fields
section: "Reference"
order: 10
toc: true
prev_page: "/docs/getting-started/"
next_page: "/docs/plugins/"
---

# Configuration Reference

This page documents all available configuration options...
```

### Landing Page Example

For standalone pages with custom layouts:

```yaml
---
title: "Welcome to My Site"
slug: ""
published: true
template: "landing.html"

# Custom fields
hero_title: "Build Faster with Go"
hero_subtitle: "A static site generator that respects your time"
cta_text: "Get Started"
cta_url: "/docs/getting-started/"
features:
  - title: "Fast Builds"
    description: "Compile thousands of pages in seconds"
    icon: "lightning"
  - title: "Plugin System"
    description: "Extend functionality with Go plugins"
    icon: "puzzle"
  - title: "Feed Generation"
    description: "RSS, Atom, and JSON feeds built-in"
    icon: "rss"
---

Additional content for the landing page...
```

### Tutorial with Series

For posts that are part of a series:

```yaml
---
title: "Building a CLI Tool - Part 2: Adding Commands"
date: 2024-01-20
published: true
tags:
  - go
  - cli
  - tutorial
description: "Learn how to add subcommands to your Go CLI application using cobra."
template: "tutorial.html"

# Series information
series: "Building a CLI Tool in Go"
series_slug: "go-cli-tutorial"
series_order: 2
series_total: 5

# Tutorial metadata
difficulty: "intermediate"
prerequisites:
  - "Basic Go knowledge"
  - "Part 1 of this series"
code_repo: "https://github.com/example/go-cli-tutorial"
---

# Adding Commands

In the previous part, we set up our project structure...
```

---

## Common Patterns

### Draft Workflow

Use `draft` and `published` together for a clear workflow:

```yaml
# Work in progress - not visible anywhere
---
title: "My Draft Post"
draft: true
published: false
---

# Ready for review - still not public
---
title: "My Draft Post"
draft: true
published: false
---

# Published - visible to everyone
---
title: "My Draft Post"
draft: false
published: true
---
```

**Feed filter for published only:**

```toml
[[markata-go.feeds]]
slug = "blog"
filter = "published == True"
```

**Feed filter excluding drafts:**

```toml
[[markata-go.feeds]]
slug = "blog"
filter = "published == True and draft == False"
```

### Scheduled Publishing

Combine `date` with feed filters for scheduled publishing:

```yaml
---
title: "New Year Announcement"
date: 2025-01-01
published: true
---
```

**Feed filter for past/current dates only:**

```toml
[[markata-go.feeds]]
slug = "blog"
filter = "published == True and date <= today"
```

Posts with future dates won't appear until that date arrives. Rebuild your site daily (via CI/CD) to "publish" scheduled posts.

### Custom Templates Per Post

Use different layouts for different types of content:

```yaml
# Regular blog post
---
title: "My Post"
template: "post.html"
---

# Tutorial with sidebar
---
title: "Go Tutorial"
template: "tutorial.html"
---

# Full-width landing page
---
title: "About Me"
template: "landing.html"
---

# Documentation with navigation
---
title: "API Reference"
template: "docs.html"
---
```

### Series of Posts

Organize related posts into a series:

**Post 1:**
```yaml
---
title: "Web Scraping with Go - Part 1: Basics"
date: 2024-01-10
series: "Web Scraping with Go"
series_order: 1
tags: [go, web-scraping, tutorial]
---
```

**Post 2:**
```yaml
---
title: "Web Scraping with Go - Part 2: Handling JavaScript"
date: 2024-01-17
series: "Web Scraping with Go"
series_order: 2
tags: [go, web-scraping, tutorial]
---
```

**Filter for the series:**
```toml
[[markata-go.feeds]]
slug = "series/web-scraping-go"
title = "Web Scraping with Go"
filter = "Extra.series == 'Web Scraping with Go'"
sort = "Extra.series_order"
reverse = false
```

### Canonical URLs for Cross-Posted Content

When you publish the same content elsewhere:

```yaml
---
title: "My Post"
canonical_url: "https://dev.to/username/my-post"
---
```

Use in templates:

```html
{% if post.Extra.canonical_url %}
<link rel="canonical" href="{{ post.Extra.canonical_url }}">
{% else %}
<link rel="canonical" href="{{ config.URL }}{{ post.Href }}">
{% endif %}
```

---

## Frontmatter in Filtering

Frontmatter fields power the feed filtering system. You can filter posts based on any frontmatter value.

### Filter Syntax

Filters use a Python-like expression syntax:

```toml
filter = "published == True"
filter = "'tutorial' in tags"
filter = "date >= '2024-01-01'"
```

### Filtering by Built-in Fields

```toml
# Published posts only
filter = "published == True"

# Exclude drafts
filter = "draft == False"

# Posts with a specific tag
filter = "'go' in tags"

# Posts from 2024
filter = "date >= '2024-01-01' and date < '2025-01-01'"

# Posts with a specific slug prefix
filter = "slug.startswith('tutorials/')"

# Posts using a specific template
filter = "template == 'tutorial.html'"
```

### Filtering by Custom Fields (Extra)

Access custom fields directly by name:

```toml
# Featured posts
filter = "featured == True"

# Posts by specific author
filter = "author == 'Jane Doe'"

# Posts in a specific category
filter = "category == 'Tutorials'"

# Posts in a series
filter = "series == 'Web Scraping with Go'"

# Intermediate difficulty tutorials
filter = "difficulty == 'intermediate'"
```

### Combined Filters

Use `and`, `or`, and parentheses for complex filters:

```toml
# Published tutorials
filter = "published == True and 'tutorial' in tags"

# Featured posts from 2024
filter = "published == True and featured == True and date >= '2024-01-01'"

# Go or Python tutorials
filter = "published == True and ('go' in tags or 'python' in tags)"

# Published, non-draft, with specific category
filter = "published == True and draft == False and category == 'Backend'"
```

### Filtering with Dates

Special date values are available:

```toml
# Posts up to today (no future posts)
filter = "date <= today"

# Posts from the last 30 days
filter = "date >= today - 30"

# Recent posts (within current timestamp)
filter = "date <= now"
```

### Filter Operators Reference

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal to | `published == True` |
| `!=` | Not equal to | `draft != True` |
| `>` | Greater than | `date > '2024-01-01'` |
| `>=` | Greater than or equal | `date >= '2024-01-01'` |
| `<` | Less than | `date < '2025-01-01'` |
| `<=` | Less than or equal | `date <= today` |
| `in` | Value in collection | `'go' in tags` |
| `and` | Logical AND | `published == True and featured == True` |
| `or` | Logical OR | `'go' in tags or 'python' in tags` |
| `not` | Logical NOT | `not draft` |

### String Methods in Filters

```toml
# Slugs starting with a prefix
filter = "slug.startswith('tutorials/')"

# Slugs ending with a suffix
filter = "slug.endswith('-guide')"

# Slugs containing a substring
filter = "slug.contains('api')"

# Case-insensitive comparison
filter = "title.lower().contains('go')"
```

### Complete Filter Examples

**Home page - recent published posts:**
```toml
[[markata-go.feeds]]
slug = ""
title = "Recent Posts"
filter = "published == True and date <= today"
sort = "date"
reverse = true
items_per_page = 5
```

**Tutorials section:**
```toml
[[markata-go.feeds]]
slug = "tutorials"
title = "Tutorials"
filter = "published == True and 'tutorial' in tags"
sort = "date"
reverse = true
```

**Featured posts:**
```toml
[[markata-go.feeds]]
slug = "featured"
title = "Featured"
filter = "published == True and featured == True"
sort = "date"
reverse = true
items_per_page = 6
```

**Author archive:**
```toml
[[markata-go.feeds]]
slug = "authors/jane-doe"
title = "Posts by Jane Doe"
filter = "published == True and author == 'Jane Doe'"
sort = "date"
reverse = true
```

**Beginner-friendly content:**
```toml
[[markata-go.feeds]]
slug = "beginners"
title = "Beginner Guides"
filter = "published == True and difficulty == 'beginner'"
sort = "date"
reverse = true
```

---

## See Also

- [[feeds-guide|Feeds Guide]] - Complete guide to feed filtering and configuration
- [[templates-guide|Templates Guide]] - Using frontmatter in templates
- [[configuration-guide|Configuration Guide]] - Site-wide defaults and settings
