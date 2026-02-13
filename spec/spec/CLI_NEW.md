# CLI `new` Command Specification

This document specifies the `markata-go new` command behavior, including the
interactive TUI wizard powered by charmbracelet/huh.

## Overview

The `new` command creates new content files with frontmatter templates. It
operates in two modes:

1. **Non-interactive mode** - title provided as argument, flags control all options
2. **Interactive TUI mode** - no title argument, a huh-based wizard guides the user

## Command Signature

```
markata-go new [title] [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--template` | `-t` | Content template to use | `post` |
| `--list` | `-l` | List available templates | `false` |
| `--dir` | | Output directory (overrides template) | Template default |
| `--draft` | | Create as a draft | `false` |
| `--tags` | | Comma-separated list of tags | `""` |
| `--plain` | | Use plain text prompts instead of TUI | Auto-detected |

**Note:** `--draft` defaults to `false`. New posts are created as published by
default (`published: true`, `draft: false`).

## Interactive TUI Wizard

When no title argument is provided, the command enters interactive mode using
charmbracelet/huh forms. The wizard falls back to plain text prompts when:

- The `--plain` flag is specified
- stdin is not a TTY

### Wizard Flow

```
┌─────────────────────────────┐
│  1. Title (text input)      │
├─────────────────────────────┤
│  2. Template (select)       │  ← Pick from available post types
├─────────────────────────────┤
│  3. Directory (select)      │  ← Pick from existing dirs or use default
├─────────────────────────────┤
│  4. Tags (multi-select)     │  ← Existing site tags as suggestions
├─────────────────────────────┤
│  5. Private? (confirm)      │  ← Whether post is private
├─────────────────────────────┤
│  6. Authors (multi-select)  │  ← Only shown if multi-author configured
├─────────────────────────────┤
│  7. Summary + Confirm       │  ← Review before creation
└─────────────────────────────┘
```

### Step Details

#### 1. Title (Required)

- `huh.Input` with placeholder text
- Cannot be empty (validated)
- Generates slug automatically

#### 2. Template Selection

- `huh.Select` showing all available templates
- Default: `post` (or value from `--template` flag)
- Templates sourced from: builtins, config, content-templates/ directory
- Each option shows: `name -> directory/ (source)`

#### 3. Directory Selection

- `huh.Select` presenting:
  1. The template's default directory (first option, marked as default)
  2. All existing content directories discovered from the filesystem
  3. A "Custom..." option that reveals a text input
- Default directories per template type are configurable in config under
  `[content_templates.placement]`
- Default directory pattern: `pages/<template>` for non-standard templates

#### 4. Tag Selection

- `huh.MultiSelect` populated with all existing tags from the site
- Tags are collected by scanning existing content files using the glob patterns
  from config
- Additional custom tags can be typed in (text input after multi-select)
- Tags sorted alphabetically

#### 5. Private Flag

- `huh.Confirm` asking "Is this post private?"
- Default: `No`
- When true, sets `private: true` in frontmatter

#### 6. Authors (Conditional)

Only shown when **all** of these conditions are met:
- The config has `[authors]` section with multiple authors defined
- At least 2 authors exist in config

When shown:
- First asks: "Use default author only?" (confirm)
  - If yes, uses the author marked `default: true` (or first active)
  - If no, shows `huh.MultiSelect` with all configured authors
- Author IDs are written to `authors:` frontmatter field

#### 7. Summary

- `huh.Note` showing a summary of all selections
- `huh.Confirm` to proceed or cancel

## Generated Frontmatter

The generated file includes these fields:

```yaml
---
title: "The Post Title"
slug: the-post-title
date: "2024-01-15"
published: true
draft: false
private: false
tags:
  - tag1
  - tag2
template: post
description: ""
authors:              # Only if multi-author configured
  - author-id
---

# The Post Title

Write your content here...
```

### Defaults

| Field | Default |
|-------|---------|
| `published` | `true` |
| `draft` | `false` |
| `private` | `false` |
| `description` | `""` |
| `tags` | `[]` |
| `template` | Selected template name |

## Configuration

### Default Directories per Template Type

```toml
[content_templates.placement]
post = "posts"
page = "pages"
docs = "docs"
article = "pages/article"
note = "pages/note"
# etc.
```

When no placement is configured for a template, the default directory is
`pages/<template>`.

### Author Configuration

```toml
[authors]
generate_pages = true

[authors.authors.alice]
id = "alice"
name = "Alice"
default = true
active = true

[authors.authors.bob]
id = "bob"
name = "Bob"
active = true
```

## Tag Discovery

Tags are collected from existing content by:

1. Loading config to get glob patterns
2. Scanning all matching markdown files
3. Extracting tags from frontmatter
4. Deduplicating and sorting alphabetically

This runs once at wizard startup. Performance is acceptable for typical sites
(< 1000 files).

## Theme Integration

The TUI wizard uses the site's configured palette for styling (same as `init`
command), falling back to the default Charm theme if no palette is configured.

## Non-Interactive Mode

When title is provided as an argument:

```bash
markata-go new "My Post" --template page --tags "go,web" --dir custom-dir
```

All options come from flags. No prompts are shown. The defaults apply for any
unspecified flags.

## Error Handling

| Error | Behavior |
|-------|----------|
| File already exists | Error with path |
| Invalid template name | Error listing available templates |
| Empty title (interactive) | Validation prevents proceeding |
| Config not found | Use defaults, skip author features |
| No TTY + no --plain | Fall back to plain mode |
