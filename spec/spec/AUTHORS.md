# Authors Specification

## Overview

The authors system allows sites to define a registry of author profiles in the configuration and reference them from post frontmatter. This enables multi-author support where each post can have one or more authors with metadata (name, role, bio, avatar, URL, social links).

## Data Model

### Author Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | (key in map) | Unique identifier for the author |
| `name` | string | **required** | Display name |
| `bio` | string? | null | Short biography |
| `email` | string? | null | Contact email |
| `avatar` | string? | null | Avatar image URL or path |
| `url` | string? | null | Personal website URL |
| `social` | map[string]string | {} | Social media links (platform -> URL) |
| `guest` | bool | false | Whether this is a guest author |
| `active` | bool | false | Whether the author is currently active |
| `default` | bool | false | Whether this is the default author for posts without explicit authors |
| `contributions` | string[] | [] | CReDiT taxonomy roles |
| `role` | string? | null | Simple role label (e.g., "author", "editor") |
| `contribution` | string? | null | Free-text contribution description |
| `details` | string? | null | Per-post description of what the author did (shown as tooltip on hover) |

### Post Author Fields

| Field | Type | Description |
|-------|------|-------------|
| `authors` | string[] or AuthorRef[] | List of author IDs or extended author references from frontmatter |
| `author` | string? | Legacy single-author field from frontmatter |
| `author_objects` | Author[] | Computed: resolved Author structs (not serialized) |
| `author_role_overrides` | map[string]string | Per-post role overrides keyed by author ID (not serialized) |
| `author_details_overrides` | map[string]string | Per-post details overrides keyed by author ID (not serialized) |

### AuthorsConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `generate_pages` | bool | false | Whether to generate individual author profile pages |
| `url_pattern` | string | "" | URL pattern for author pages (e.g., `/authors/{id}/`) |
| `feeds_enabled` | bool | false | Whether to generate per-author feeds |
| `authors` | map[string]Author | {} | Author registry keyed by ID |

## Configuration

Authors are configured under `[markata-go.authors]` in the site configuration:

```toml
[markata-go.authors]
generate_pages = false
feeds_enabled = false

[markata-go.authors.authors.waylon]
name = "Waylon Walker"
role = "author"
url = "https://waylonwalker.com"
active = true
default = true

[markata-go.authors.authors.guest]
name = "Guest Writer"
role = "editor"
guest = true
active = true
```

## Frontmatter

Posts reference authors by ID using a simple string format:

```yaml
---
title: My Post
authors:
  - waylon
  - guest
---
```

### Per-Post Role Overrides

Authors can play different roles on different posts. Use the extended format to specify per-post roles:

```yaml
---
title: Collaborative Post
authors:
  - id: waylon
    role: author
  - id: codex
    role: pair programmer
---
```

### Per-Post Details

Authors can have per-post details describing their specific contribution. Details are shown as a tooltip on hover in the byline:

```yaml
---
title: Collaborative Post
authors:
  - id: waylon
    role: author
    details: wrote the introduction and conclusion
  - id: codex
    role: pair programmer
    details: wrote the code examples
  - id: kimmi
    role: outliner
    details: outlined the post structure
---
```

The `details` field is optional and independent of `role`. Both can be set together or separately.

Mixed formats are supported -- strings and extended references can be combined:

```yaml
---
title: Mixed Format Example
authors:
  - waylon
  - id: codex
    role: editor
---
```

When a per-post role is specified, it overrides the author's config-level role for that post only. The `role_display` in templates will reflect the per-post override.

Legacy single-author field is also supported:

```yaml
---
title: My Post
author: "Jane Doe"
---
```

### Priority

1. `authors` array (new multi-author format) takes precedence
2. `author` string (legacy single-author format) used as fallback
3. If neither is specified, the default author from config is assigned

## Plugin: authors

- **Stage:** Transform
- **Priority:** `PriorityFirst + 1` (runs right after `auto_title`)
- **Purpose:** Resolves author IDs in post frontmatter against the site-wide authors config

### Behavior

1. Reads `config.Authors.Authors` map from the models config
2. Identifies the default author (the one with `default: true`)
3. For each non-skipped post:
   - Calls `post.GetAuthors()` to get author IDs (from `authors` array or `author` string)
   - If no author IDs: assigns the default author (if one exists)
   - If author IDs exist: resolves each ID against the config map
    - Checks `post.AuthorRoleOverrides` for per-post role overrides
    - Checks `post.AuthorDetailsOverrides` for per-post details overrides
    - If a per-post role override exists for an author ID, clones the Author struct and sets the overridden Role (clearing Contribution so `GetRoleDisplay()` uses the new Role)
    - If a per-post details override exists for an author ID, sets the Details field on the (possibly already cloned) Author struct
   - Populates `post.AuthorObjects` with the resolved (possibly role-overridden) Author structs
4. Logs a warning for unknown author IDs

### Template Access

In templates, resolved authors are available as:

- `post.author_objects` - Array of author maps with all fields
- `post.authors` - Array of author ID strings
- `post.author` - Legacy single author string (if set)
- `authors` - Top-level map of all configured authors (ID -> author map)

Each author object in templates has:

| Key | Type | Description |
|-----|------|-------------|
| `id` | string | Author ID |
| `name` | string | Display name |
| `bio` | string | Biography (if set) |
| `email` | string | Email (if set) |
| `avatar` | string | Avatar URL (if set) |
| `url` | string | Website URL (if set) |
| `role` | string | Role label (if set) |
| `role_display` | string | Pre-computed display string for role/contributions |
| `guest` | bool | Guest author flag |
| `active` | bool | Active flag |
| `default` | bool | Default author flag |
| `social` | map | Social links (if set) |
| `contributions` | []string | CReDiT roles (if set) |
| `contribution` | string | Contribution text (if set) |
| `details` | string | Per-post details text (if set, shown as tooltip) |

## Build Cache

The `Authors` and `Author` fields are included in `CachedPostData` so that author information survives build cache round-trips. The `AuthorObjects` field is NOT cached -- it is recomputed by the authors plugin on every build since it depends on the current config.

## Template Components

### post_byline.html

A reusable component (`templates/components/post_byline.html`) renders the author byline:

- **Multi-author**: Shows each author name (linked if URL set) with role in parentheses, comma-separated
- **Single author fallback**: Shows the site-level `config.author` as a linked name
- **Date and reading time**: Shown alongside author info when available
- **No avatars in byline**: Avatars are intentionally omitted to avoid visual duplication with linked author names

This component is included in all layout templates (docs, blog) and the standalone post template.

### h-card.html

The existing `templates/components/h-card.html` provides IndieWeb microformats2 markup for authors, used in the site footer area.

## Validation

`Author.Validate()` checks:
- `Name` must not be empty
- If `Email` is set, it must contain `@`

## Default Author Resolution

`models.GetDefaultAuthor(authorMap)` returns the first author in the map with `Default: true`. If no author has `Default: true`, returns nil (no default assignment).
