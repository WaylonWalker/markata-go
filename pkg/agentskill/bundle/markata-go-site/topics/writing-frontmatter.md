# Writing And Frontmatter

Use this topic when creating or editing posts, pages, docs, or frontmatter fields.

## Preferred Commands

- `markata-go new`
- `markata-go new --list`
- `markata-go list posts`

## Frontmatter Format

Markata-go uses YAML frontmatter between `---` delimiters at the top of the file.

Example:

```markdown
---
title: "Getting Started with Go"
date: 2026-04-01
published: true
draft: false
tags:
  - go
  - tutorial
description: "A beginner-friendly guide."
template: post
---

Write your content here.
```

## Important Default Behavior

- files without frontmatter are still valid
- `published` defaults to `false` if omitted in raw content
- the `new` command creates content with `published: true` and `draft: false`
- new starter content does not add a default H1 in the body because templates usually render the title as the page H1

## Behavior Rules Agents Should Remember

- `published: true` means the post is eligible for public feeds and listings
- `published: false` can still render a shadow page reachable by direct URL
- `draft: true` means the post should not render at all
- `skip: true` means the file is ignored during processing
- `slug: ""` or `slug: /` makes a content file the homepage
- duplicate slugs create output conflicts and should be treated as a build blocker

## Common Built-In Fields

- `title`
- `slug`
- `date`
- `published`
- `draft`
- `private`
- `tags`
- `description`
- `template`
- `layout`
- `skip`
- `authors`
- `author`
- `modified`

## Slug Guidance

- if omitted, slug is generated from title or filename
- `index.md` gets a directory-based slug
- `slug: ""` or `slug: /` creates the homepage
- duplicate slugs cause output conflicts

## Field Effects

- `title`: affects page title, feed listings, and slug generation
- `slug`: affects final URL path and output location
- `date`: affects sorting, scheduling, and feed ordering
- `published`: affects discoverability in feeds, sitemaps, and public listings
- `draft`: affects whether output is rendered at all
- `tags`: affects filtering and tag/archive pages
- `description`: affects meta tags, SEO, and some feed metadata, but not the main excerpt generation path
- `template`: selects a specific template file
- `layout`: selects a layout-driven page shell when the site uses layout config
- `authors` or `author`: affect bylines and author objects in templates
- custom fields: appear in `post.Extra` for templates and plugin logic

## Recommended Content Workflow

Use `markata-go new` when possible because it knows current built-in content templates such as:

- `post`
- `page`
- `docs`
- `article`
- `note`
- `photo`
- `video`
- `link`
- `quote`
- `guide`
- `inline`
- `contact`
- `author`

## Common Site Patterns

### Public blog post

```yaml
---
title: "My Post"
date: 2026-04-01
published: true
draft: false
tags:
  - go
description: "Short SEO summary"
---
```

### Shadow page

```yaml
---
title: "Reviewer Notes"
published: false
draft: false
---
```

### Truly private draft

```yaml
---
title: "Unpublished Draft"
draft: true
---
```

### Homepage from content

```yaml
---
title: "Welcome"
slug: ""
published: true
layout: landing
---
```

### Page with custom template

```yaml
---
title: "API Landing"
published: true
template: "landing.html"
---
```

## Guidance

- Prefer `markata-go new` when creating content so the site's defaults stay consistent.
- Keep frontmatter YAML valid and minimal.
- Use existing frontmatter conventions from nearby content before introducing new fields.
- Do not duplicate the title with an extra H1 in the body unless the repo already does that intentionally.
- If you introduce a new custom field, check whether templates and feeds need to read it.
- Use `draft: true` for content that must not render at all; use `published: false` for shadow pages.
- Prefer `authors` over legacy `author` when the site already uses structured author config.

## Before Adding New Metadata

1. Search nearby content for the same field.
2. Check whether templates or feeds already read it.
3. Prefer existing naming over introducing another alias.
4. Decide whether the field belongs in frontmatter, config, or computed plugin output.
