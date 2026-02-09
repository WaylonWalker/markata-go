# Multi-Author System Specification

This document specifies the multi-author support system, including author models, configuration, validation, and template integration.

## Overview

The multi-author system enables sites with multiple contributors while maintaining backward compatibility with single-author configurations. It supports three tiers of contributor attribution: academic CReDiT roles, simplified blog roles, and free-form custom contributions.

## Author Model

### Core Author Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier (used in frontmatter) |
| `name` | string | Yes | Display name |
| `bio` | string? | No | Short biography |
| `email` | string? | No | Contact email |
| `avatar` | string? | No | Path to avatar image |
| `url` | string? | No | Personal website URL |
| `social` | map[string]string | No | Social media handles |
| `guest` | bool | No | Mark as guest author (default: false) |
| `active` | bool | No | Include in author lists (default: true) |
| `default` | bool | No | Fallback author for posts without attribution (default: false) |

### Contributor Role Fields

The system provides three levels of role specification:

#### Level 1: CReDiT Academic Roles

Field: `contributions` (string[])

CReDiT (Contributor Roles Taxonomy) is the ANSI/NISO Z39.104-2022 standard for scholarly contributions. These 14 roles capture detailed academic attribution:

| Role | Description |
|------|-------------|
| `conceptualization` | Ideas, formulation of research questions, hypothesis development |
| `data-curation` | Management, cleaning, and annotation of data |
| `formal-analysis` | Statistical analysis, interpretation, and modeling |
| `funding-acquisition` | Securing financial support for the project |
| `investigation` | Research, data collection, and experimentation |
| `methodology` | Study design, methods development, and protocol creation |
| `project-administration` | Project management, coordination, and logistics |
| `resources` | Provision of materials, reagents, equipment, or facilities |
| `software` | Programming, software development, and implementation |
| `supervision` | Mentorship, oversight, and guidance |
| `validation` | Verification, reproducibility testing, and quality assurance |
| `visualization` | Data visualization, figure creation, and presentation |
| `writing-original-draft` | Initial manuscript or content preparation |
| `writing-review-editing` | Revision, editing, and proofreading |

**When to use:** Academic publications, research documentation, technical specifications, and collaborative scientific work.

#### Level 2: Simple Blog Roles

Field: `role` (string)

Simplified roles for content-focused sites:

| Role | Description |
|------|-------------|
| `author` | Primary content writer |
| `editor` | Editorial oversight and content review |
| `designer` | Visual design and UX contributions |
| `maintainer` | Ongoing maintenance and updates |
| `contributor` | General contribution without primary authorship |
| `reviewer` | Peer review and feedback |
| `translator` | Translation and localization |

**When to use:** Blog posts, documentation, tutorials, and general content sites.

#### Level 3: Custom Contribution

Field: `contribution` (string)

Free-form text describing the author's specific contribution. This overrides other role fields in display contexts.

**When to use:** Unique contribution descriptions, hybrid roles, or when standard taxonomies do not fit.

### Validation Rules

1. **ID uniqueness**: Author IDs must be unique within the site
2. **Name required**: Every author must have a display name
3. **Single default**: Exactly one author may be marked as `default: true` per site
4. **CReDiT validation**: All entries in `contributions` must be valid CReDiT role names
5. **Simple role validation**: The `role` field must be a valid simple role name

**Validation error examples:**

```
Error: only one author can be marked as default, found 2
Error: author john-doe: invalid CReDiT contribution: coding, valid roles are: conceptualization, data-curation, ...
Error: author jane-smith: invalid role: writer, valid roles are: author, editor, designer, ...
```

## Configuration

### Authors Configuration Section

Authors are configured under `[markata-go.authors]` in `markata-go.toml`:

```toml
[markata-go.authors]
generate_pages = true        # Create author bio pages
url_pattern = "/authors/{author}/"  # URL pattern for author pages
feeds_enabled = true         # Generate RSS/Atom/JSON feeds per author
```

### Defining Authors

Authors are defined under `[markata-go.authors.authors.{id}]`:

```toml
[markata-go.authors.authors.jane-doe]
name = "Jane Doe"
bio = "Software engineer and technical writer"
email = "jane@example.com"
avatar = "/images/authors/jane.jpg"
url = "https://janedoe.dev"
default = true

[markata-go.authors.authors.jane-doe.social]
github = "janedoe"
twitter = "jane_doe"
linkedin = "janedoe"

# CReDiT roles for academic attribution
contributions = ["conceptualization", "writing-original-draft", "software"]

[markata-go.authors.authors.guest-author]
name = "Guest Contributor"
bio = "Visiting expert"
guest = true
role = "contributor"  # Simple role
```

### Complete Configuration Example

```toml
[markata-go]
title = "Multi-Author Blog"
url = "https://example.com"

[markata-go.authors]
generate_pages = true
url_pattern = "/authors/{author}/"
feeds_enabled = true

[markata-go.authors.authors.primary-author]
name = "Primary Author"
bio = "Site maintainer and lead writer"
email = "author@example.com"
avatar = "/images/authors/primary.jpg"
url = "https://primary.example.com"
default = true

[markata-go.authors.authors.primary-author.social]
github = "primaryauthor"
twitter = "primary_author"

[markata-go.authors.authors.guest-expert]
name = "Guest Expert"
bio = "Industry specialist"
guest = true
contributions = ["investigation", "writing-review-editing"]
```

## Post Model Extensions

### Author Fields in Frontmatter

Posts support two author fields:

| Field | Type | Description |
|-------|------|-------------|
| `author` | string? | Legacy single author (backward compatible) |
| `authors` | string[] | Multiple author IDs |

**Priority:** When both fields are present, `authors` takes precedence.

### Frontmatter Examples

**Legacy single author (backward compatible):**
```yaml
---
title: "My Post"
author: "jane-doe"
---
```

**Multiple authors:**
```yaml
---
title: "Collaborative Post"
authors: ["jane-doe", "john-smith", "guest-expert"]
---
```

**Both fields (authors takes precedence):**
```yaml
---
title: "Multi-Author Post"
author: "jane-doe"           # Ignored when authors present
authors: ["john-smith"]      # This is used
---
```

### Computed Author Fields

During the build process, these computed fields are added to posts:

| Field | Type | Description |
|-------|------|-------------|
| `author_objects` | Author[] | Resolved author objects from IDs |
| `primary_author` | Author? | First author (for single-author contexts) |

## Template Context

### Global Context

Templates receive author data through the global context:

| Variable | Type | Description |
|----------|------|-------------|
| `authors` | map[string]Author | All configured authors keyed by ID |
| `default_author` | Author? | The author marked as default |

### Post Context

Each post in template loops has:

| Variable | Type | Description |
|----------|------|-------------|
| `post.authors` | string[] | Author IDs from frontmatter |
| `post.author_objects` | Author[] | Resolved author objects |

### Template Examples

**Display all authors for a post:**
```html
{% if post.author_objects %}
<div class="post-authors">
  {% for author in post.author_objects %}
  <div class="author h-card">
    {% if author.avatar %}
    <img class="u-photo" src="{{ author.avatar }}" alt="{{ author.name }}">
    {% endif %}
    <a class="p-name u-url" href="/authors/{{ author.id }}/">{{ author.name }}</a>
    {% if author.bio %}
    <span class="p-note">{{ author.bio }}</span>
    {% endif %}
  </div>
  {% endfor %}
</div>
{% endif %}
```

**Display author with role:**
```html
{% for author in post.author_objects %}
<div class="author">
  <span class="name">{{ author.name }}</span>
  <span class="role">{{ author.get_role_display() }}</span>
</div>
{% endfor %}
```

## Author Pages

When `generate_pages = true`, author bio pages are generated at `{url_pattern}` with `{author}` replaced by the author ID.

### Default URL Pattern

`/authors/{author}/` produces:
- `/authors/jane-doe/`
- `/authors/john-smith/`

### Author Page Template

Author pages use `author.html` template with context:

| Variable | Type | Description |
|----------|------|-------------|
| `author` | Author | Current author being displayed |
| `posts` | Post[] | Posts by this author |

### Author Page Content

Generated author pages include:
1. Author profile (name, bio, avatar, social links)
2. h-card microformat markup
3. List of posts by the author
4. Links to author-specific feeds (if enabled)

## Author Feeds

When `feeds_enabled = true`, each author gets dedicated feeds:

| Feed Type | URL Pattern | Example |
|-----------|-------------|---------|
| RSS | `/authors/{author}/rss.xml` | `/authors/jane-doe/rss.xml` |
| Atom | `/authors/{author}/atom.xml` | `/authors/jane-doe/atom.xml` |
| JSON | `/authors/{author}/feed.json` | `/authors/jane-doe/feed.json` |

Feed entries include only posts where the author appears in the `authors` array.

## Backward Compatibility

### Legacy Single Author Support

Sites using `config.author` continue to work:

```toml
[markata-go]
author = "Jane Doe"  # Still functional
```

When `markata-go.authors` is not configured:
- `config.author` is used as the site-wide author
- Templates display single author context
- Author pages are not generated

### Migration Path

To migrate from single to multi-author:

1. Create author configuration with one default author
2. Move `config.author` data to `authors.authors.{id}.name`
3. Add additional authors as needed
4. Update frontmatter to use `authors` array

**Before:**
```toml
[markata-go]
title = "My Blog"
author = "Jane Doe"
```

**After:**
```toml
[markata-go]
title = "My Blog"

[markata-go.authors]
generate_pages = true

[markata-go.authors.authors.jane-doe]
name = "Jane Doe"
default = true
```

## Structured Data

Author information integrates with Schema.org markup:

```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "author": [
    {
      "@type": "Person",
      "name": "Jane Doe",
      "url": "https://example.com/authors/jane-doe/",
      "image": "https://example.com/images/authors/jane.jpg",
      "sameAs": [
        "https://github.com/janedoe",
        "https://twitter.com/jane_doe"
      ]
    }
  ]
}
```

For posts with multiple authors, the `author` field is an array. For single authors, it is a single object.

## h-Card Microformats

Author pages and post bylines include h-card markup:

```html
<div class="h-card">
  <img class="u-photo" src="/images/authors/jane.jpg" alt="Jane Doe">
  <a class="p-name u-url" href="/authors/jane-doe/">Jane Doe</a>
  <span class="p-note">Software engineer and writer</span>
</div>
```

This enables IndieWeb parsers to extract author information.

## Error Handling

### Missing Authors

When a post references an author ID not in the configuration:

1. Warning is logged during build
2. Missing author is skipped in `author_objects`
3. Post still renders with available authors
4. If no valid authors, falls back to `default_author`

### Validation Failures

Configuration validation runs during the Configure stage. Invalid configurations fail the build with descriptive error messages.

## Performance Considerations

- Author objects are resolved once during the Load stage
- Author pages are generated in parallel
- Feed generation batches posts by author
- Template context caches author lookups

## Future Extensions

Potential enhancements (not yet implemented):

- Author collaboration metrics
- Guest author submission workflows
- Author reputation scoring
- Content co-authorship weighting
- Author team/group support
