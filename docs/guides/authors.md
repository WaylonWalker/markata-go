---
title: "Multi-Author Support"
description: "Configure and use multiple authors, CReDiT roles, and contributor attribution"
date: 2026-02-08
published: true
tags:
  - configuration
  - authors
  - frontmatter
---

# Multi-Author Support

markata-go supports sites with multiple contributors through a flexible author system. You can attribute posts to individual authors, generate author bio pages, create per-author feeds, and use standardized contribution roles.

## Quick Start

Add authors to your `markata-go.toml`:

```toml
[markata-go.authors]
generate_pages = true
feeds_enabled = true

[markata-go.authors.authors.jane-doe]
name = "Jane Doe"
bio = "Software engineer and technical writer"
email = "jane@example.com"
default = true

[markata-go.authors.authors.john-smith]
name = "John Smith"
bio = "DevOps specialist"
```

Then use authors in your post frontmatter:

```yaml
---
title: "My Collaborative Post"
authors: ["jane-doe", "john-smith"]
---
```

## Configuring Authors

### Basic Author Setup

Define authors under `[markata-go.authors.authors.{id}]` where `{id}` is a unique identifier:

```toml
[markata-go.authors.authors.alice]
name = "Alice Johnson"
bio = "Frontend developer and accessibility advocate"
avatar = "/images/authors/alice.jpg"
url = "https://alice.dev"

[markata-go.authors.authors.alice.social]
github = "alicejohnson"
twitter = "alice_dev"
```

### Required and Optional Fields

**Required:**
- `name` - Display name

**Optional:**
- `bio` - Short biography (appears on author pages)
- `email` - Contact address
- `avatar` - Path to profile image
- `url` - Personal website
- `social` - Map of platform to username
- `guest` - Mark as guest author
- `active` - Include in author lists (default: true)
- `default` - Fallback author for unattributed posts

### The Default Author

Exactly one author should be marked as `default: true`. This author is used when:

- A post has no `authors` field
- An author ID in frontmatter does not exist
- Rendering single-author contexts (feeds, meta tags)

```toml
[markata-go.authors.authors.primary]
name = "Site Owner"
default = true  # Only one default allowed
```

Validation fails if multiple authors are marked as default.

## Specifying Contributor Roles

markata-go provides three ways to describe what each author contributed:

### 1. CReDiT Academic Roles

Use CReDiT (Contributor Roles Taxonomy) for scholarly or technical content. These 14 standardized roles capture detailed academic contributions:

```toml
[markata-go.authors.authors.researcher]
name = "Dr. Sarah Chen"
contributions = [
  "conceptualization",
  "methodology", 
  "writing-original-draft"
]
```

**Available CReDiT roles:**

| Role | Use for |
|------|---------|
| `conceptualization` | Ideas, research questions, hypotheses |
| `data-curation` | Data management, cleaning, annotation |
| `formal-analysis` | Statistical analysis, modeling |
| `funding-acquisition` | Securing financial support |
| `investigation` | Research, data collection |
| `methodology` | Study design, protocols |
| `project-administration` | Management, coordination |
| `resources` | Materials, tools, facilities |
| `software` | Programming, development |
| `supervision` | Mentorship, oversight |
| `validation` | Verification, reproducibility |
| `visualization` | Data visualization, figures |
| `writing-original-draft` | Initial content creation |
| `writing-review-editing` | Revision, editing, proofreading |

**Example for a research paper:**

```toml
[markata-go.authors.authors.lead-researcher]
name = "Dr. Alice Wong"
contributions = [
  "conceptualization",
  "investigation",
  "writing-original-draft"
]

[markata-go.authors.authors.data-scientist]
name = "Bob Martinez"
contributions = [
  "data-curation",
  "formal-analysis",
  "visualization"
]

[markata-go.authors.authors.advisor]
name = "Prof. Carol Smith"
contributions = [
  "supervision",
  "funding-acquisition",
  "writing-review-editing"
]
```

### 2. Simple Blog Roles

For blogs and content sites, use simple roles:

```toml
[markata-go.authors.authors.editor]
name = "Editorial Team"
role = "editor"
```

**Available simple roles:**

| Role | Use for |
|------|---------|
| `author` | Primary content writer |
| `editor` | Editorial oversight |
| `designer` | Visual design |
| `maintainer` | Ongoing maintenance |
| `contributor` | General contribution |
| `reviewer` | Content review |
| `translator` | Translation work |

### 3. Custom Contribution Text

For specific descriptions, use free-form text:

```toml
[markata-go.authors.authors.specialist]
name = "Domain Expert"
contribution = "Provided industry insights and fact-checking"
```

Custom contributions override other role fields in display contexts.

## Using Authors in Posts

### Multiple Authors

List author IDs in the `authors` array:

```yaml
---
title: "Building Microservices with Go"
authors: ["jane-doe", "john-smith", "guest-expert"]
date: 2026-02-08
---

Content here...
```

Authors appear in the order specified.

### Single Author (Backward Compatible)

The legacy `author` field still works:

```yaml
---
title: "My Solo Post"
author: "jane-doe"
---
```

This is equivalent to `authors: ["jane-doe"]`.

### Guest Authors

Mark occasional contributors as guests:

```toml
[markata-go.authors.authors.guest-expert]
name = "Visiting Expert"
guest = true  # Shows guest badge on author pages
```

## Author Pages

Enable author bio pages to give each contributor a dedicated profile:

```toml
[markata-go.authors]
generate_pages = true
url_pattern = "/authors/{author}/"  # Optional, this is the default
```

This generates pages at:
- `/authors/jane-doe/`
- `/authors/john-smith/`

Each page includes:
- Author profile with avatar and bio
- Social media links
- List of posts by the author
- h-card microformat markup for IndieWeb compatibility

## Author Feeds

Generate RSS, Atom, and JSON feeds for each author:

```toml
[markata-go.authors]
feeds_enabled = true
```

This creates:
- `/authors/{author}/rss.xml`
- `/authors/{author}/atom.xml`
- `/authors/{author}/feed.json`

Each feed contains only posts where that author appears.

## Templates

### Displaying Authors

Access authors in templates through `post.author_objects`:

```html
{% if post.author_objects %}
<div class="post-authors">
  {% for author in post.author_objects %}
  <div class="author h-card">
    {% if author.avatar %}
    <img class="u-photo" src="{{ author.avatar }}" alt="{{ author.name }}">
    {% endif %}
    <a class="p-name u-url" href="/authors/{{ author.id }}/">
      {{ author.name }}
    </a>
    {% if author.guest %}
    <span class="guest-badge">Guest</span>
    {% endif %}
  </div>
  {% endfor %}
</div>
{% endif %}
```

### Displaying Roles

Show what each author contributed:

```html
{% for author in post.author_objects %}
<div class="author">
  <span class="name">{{ author.name }}</span>
  {% if author.contributions %}
  <span class="contributions">{{ author.contributions|join(", ") }}</span>
  {% endif %}
  {% if author.role %}
  <span class="role">{{ author.role }}</span>
  {% endif %}
</div>
{% endfor %}
```

### Global Authors List

Access all authors in any template:

```html
<h2>Our Team</h2>
{% for id, author in authors %}
  {% if author.active and not author.guest %}
  <a href="/authors/{{ id }}/">{{ author.name }}</a>
  {% endif %}
{% endfor %}
```

## Complete Configuration Example

```toml
[markata-go]
title = "Tech Collaborative"
url = "https://techcollab.example.com"

[markata-go.authors]
generate_pages = true
feeds_enabled = true

# Primary site author
[markata-go.authors.authors.alice]
name = "Alice Chen"
bio = "Founder and lead developer"
email = "alice@example.com"
avatar = "/images/authors/alice.jpg"
url = "https://alicechen.dev"
default = true

[markata-go.authors.authors.alice.social]
github = "alicechen"
twitter = "alicechen_dev"
linkedin = "alicechen"

# Regular contributor
[markata-go.authors.authors.bob]
name = "Bob Smith"
bio = "DevOps engineer and cloud architect"
avatar = "/images/authors/bob.jpg"
role = "author"

[markata-go.authors.authors.bob.social]
github = "bobsmith"
twitter = "bobsmith_ops"

# Guest expert with CReDiT roles
[markata-go.authors.authors.dr-martinez]
name = "Dr. Elena Martinez"
bio = "Security researcher and consultant"
guest = true
contributions = ["investigation", "validation", "writing-review-editing"]

[markata-go.authors.authors.dr-martinez.social]
twitter = "dr_martinez_sec"
```

## Migrating from Single Author

If your site currently uses `config.author`:

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

All existing posts using `author: "Jane Doe"` in frontmatter will continue to work. You can gradually migrate to using author IDs.

## Validation and Error Messages

The build validates author configuration and provides helpful error messages:

```
Error: only one author can be marked as default, found 2
  authors.authors.alice.default = true
  authors.authors.bob.default = true

Error: author dr-martinez: invalid CReDiT contribution: "coding"
  Valid roles: conceptualization, data-curation, formal-analysis, ...

Warning: Post "unknown-author-post.md" references unknown author "unknown-id"
```

## Best Practices

1. **Use consistent IDs**: Author IDs should be lowercase with hyphens (`jane-doe`, not `Jane Doe`)
2. **Mark one default**: Always have exactly one default author for fallback
3. **Provide bios**: Author bios improve SEO and reader engagement
4. **Use avatars**: Consistent avatar sizing (recommended: 200x200px minimum)
5. **Enable pages**: Author pages help readers discover more content from contributors they like
6. **Choose appropriate roles**: Use CReDiT for technical/academic content, simple roles for blogs
7. **Mark guests**: Use `guest = true` for one-time contributors

## Troubleshooting

**Authors not appearing on posts**
- Check that author IDs in frontmatter match configuration IDs exactly
- Verify `active = true` (this is the default)

**Author pages not generated**
- Ensure `generate_pages = true` in `[markata-go.authors]`
- Check that the build completes without errors

**Default author not used**
- Verify exactly one author has `default = true`
- Check validation output for configuration errors

**Feeds missing authors**
- Enable with `feeds_enabled = true`
- Feeds only include posts where the author is explicitly listed
