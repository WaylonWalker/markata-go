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

## Slug Guidance

- if omitted, slug is generated from title or filename
- `index.md` gets a directory-based slug
- `slug: ""` or `slug: /` creates the homepage
- duplicate slugs cause output conflicts

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

## Guidance

- Prefer `markata-go new` when creating content so the site's defaults stay consistent.
- Keep frontmatter YAML valid and minimal.
- Use existing frontmatter conventions from nearby content before introducing new fields.
- Do not duplicate the title with an extra H1 in the body unless the repo already does that intentionally.
- If you introduce a new custom field, check whether templates and feeds need to read it.

## Before Adding New Metadata

1. Search nearby content for the same field.
2. Check whether templates or feeds already read it.
3. Prefer existing naming over introducing another alias.
