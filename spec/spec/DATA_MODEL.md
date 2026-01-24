# Data Model Specification

This document specifies the content and configuration data models.

## Overview

The system uses two main models:

1. **Post** - Represents a single piece of content
2. **Config** - Represents site configuration

Both models are dynamically constructed by merging fragments from plugins.

---

## Post Model

### Required Fields

Every Post MUST have these fields:

| Field | Type | Description |
|-------|------|-------------|
| `path` | Path | Source file path |
| `content` | string | Raw content (after frontmatter extraction) |
| `slug` | string | URL-safe identifier |
| `href` | string | Relative URL path (e.g., `/my-post/`) |

### Standard Optional Fields

These fields SHOULD be supported:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string? | null | Content title |
| `date` | date? | null | Publication date |
| `published` | bool | false | Include in feeds (shadow pages if false) |
| `draft` | bool | false | Work-in-progress (not rendered) |
| `skip` | bool | false | Skip during build (not rendered) |
| `tags` | string[] | [] | Content tags/categories |
| `description` | string? | null | Summary/excerpt |
| `template` | string | "post.html" | Template to render with |
| `config_overrides` | object? | null | Per-post configuration overrides |
| `html` | string? | null | Final rendered HTML |
| `article_html` | string? | null | Rendered content (without template) |

### Field Behaviors

#### `published` - Shadow Pages

Controls whether content appears in feeds, sitemaps, and RSS. **All non-draft, non-skip content is rendered regardless of published status.**

| `published` | `draft` | Behavior |
|-------------|---------|----------|
| `true` | `false` | Rendered + in feeds + in sitemap |
| `false` | `false` | Rendered as "shadow page" + NOT in feeds + NOT in sitemap |
| any | `true` | NOT rendered (draft is for private WIP) |

**Shadow Pages** are posts with `published: false` that are:
- **Rendered** to HTML and accessible via direct URL
- **Excluded** from feeds, sitemaps, and RSS
- **Not discoverable** through normal site navigation

**Use cases for shadow pages:**
- Draft content accessible to reviewers via direct URL
- Private documentation not linked publicly
- Work-in-progress shared with specific people
- Admin pages or development tools
- Staging versions of pages

**Example:**
```yaml
---
title: "Draft Post - For Review"
published: false  # Renders to /draft-post/, but not in feeds
---
This content is accessible at /draft-post/ but won't appear in
the blog feed or sitemap.
```

#### `draft` - True Private Content

Posts with `draft: true` are **never rendered** regardless of other settings. Use this for truly private work-in-progress content that should not be accessible at all.

```yaml
---
title: "Very Early Draft"
draft: true  # Will NOT be rendered at all
---
```

#### `config_overrides`

Allows a post to override **any** configuration value. The overrides are deep-merged with the global configuration when rendering this specific post.

**Syntax:**
```yaml
---
title: Special Post
config_overrides:
  # Override any config key
  markdown:
    highlight:
      theme: monokai

  head:
    meta:
      - name: robots
        content: noindex

  style:
    color_bg: "#000000"

  theme:
    options:
      show_toc: false

  # Disable plugins for this post
  toc:
    enabled: false
---
```

**Merge behavior:**
| Type | Behavior |
|------|----------|
| Scalars | Post value replaces global value |
| Objects | Deep merge (post keys override, others preserved) |
| Arrays | Post array replaces global array |
| Special | `head.meta`, `head.link`, `head.script` arrays are appended |

**Use cases:**
- Disable features for a specific post (TOC, prev/next)
- Change code highlighting theme
- Add post-specific meta tags or scripts
- Override styling for landing pages
- Use different markdown extensions

See [HEAD_STYLE.md](./HEAD_STYLE.md) for detailed head/style override documentation.

#### `slug`

The URL-safe identifier for the content.

**Generation priority:**
1. Explicit `slug` in frontmatter (including empty string)
2. Derived from `title` (if present)
3. Derived from file path (stem without extension)

**Transformation rules:**
- Lowercase
- Replace spaces with hyphens
- Remove special characters except hyphens
- Collapse multiple hyphens
- Strip leading/trailing hyphens

**Examples:**
| Input | Slug |
|-------|------|
| `"Hello World"` | `hello-world` |
| `"What's New?"` | `whats-new` |
| `"Python 3.12"` | `python-312` |
| `path: blog/my-post.md` | `my-post` |

#### Custom Slugs

Slugs can be explicitly set in frontmatter to override automatic generation:

```yaml
---
title: About Me
slug: about  # Explicit slug
---
```

**Custom slug normalization:**
- Leading and trailing slashes are stripped: `/about/` → `about`
- Nested paths are preserved: `/docs/guides/install` → `docs/guides/install`
- Slash alone (`/`) means homepage (empty slug)
- Empty string (`""`) means homepage (empty slug)

**Homepage via custom slug:**

Any post can become the homepage by setting an empty or slash slug:

```yaml
---
title: Welcome
slug: ""      # Becomes homepage at /
published: true
---
```

or equivalently:

```yaml
---
title: Welcome
slug: /       # Also becomes homepage at /
published: true
---
```

**Use cases for custom slugs:**
- Create homepage from content in any directory
- Control output paths independently of file location
- Migrate from other SSGs with different URL structures
- Create SEO-friendly URLs that differ from file names
- Group related content under custom paths

#### Special Case: `index.md` Files

Files named `index.md` receive special slug handling to enable homepage and directory-based URLs:

| File Path | Slug | Href |
|-----------|------|------|
| `./index.md` | `""` (empty) | `/` (homepage) |
| `docs/index.md` | `docs` | `/docs/` |
| `blog/guides/index.md` | `blog/guides` | `/blog/guides/` |

This allows:
- Creating a homepage with custom content instead of requiring a feed with `slug = ""`
- Directory-based URL structures where `docs/index.md` becomes `/docs/` rather than `/docs/index/`

**Note:** The filename check is case-insensitive (`INDEX.MD` also matches).

#### `href`

The relative URL path for the content.

**Format:** `/{slug}/`

Always starts with `/` and ends with `/`.

**Examples:**
| Slug | Href |
|------|------|
| `hello-world` | `/hello-world/` |
| `blog/post-1` | `/blog/post-1/` |

#### `date`

Publication date for the content.

**Accepted formats:**
- ISO 8601: `2024-01-15`
- ISO 8601 with time: `2024-01-15T10:30:00`
- Common formats: `January 15, 2024`, `15/01/2024`

**Auto-detection (optional):**
- From file modification time
- From filename pattern: `2024-01-15-my-post.md`
- From path pattern: `posts/2024/01/my-post.md`

#### `published` vs `draft`

These are related but distinct:

| published | draft | Behavior |
|-----------|-------|----------|
| true | false | Visible in production |
| false | false | Not visible (explicit unpublish) |
| false | true | Work in progress |
| true | true | Invalid (treat as draft) |

Default behavior:
- New posts: `published: false, draft: false`
- Published posts: `published: true, draft: false`

#### `skip`

When true, the post is loaded but not rendered or saved.

Use cases:
- Temporarily exclude content
- Content used only for data (not pages)
- Failed content that shouldn't block build

#### `template`

Can be a string or object:

```yaml
# Simple: use a single template
template: post.html

# Complex: different templates for different contexts
template:
  default: post.html
  card: card.html
  feed: feed-item.html
```

---

## Config Model

### Required Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `output_dir` | Path | `"output"` | Build output directory |
| `hooks` | string[] | `["default"]` | Plugins to load |
| `disabled_hooks` | string[] | `[]` | Plugins to exclude |

### Standard Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | URL? | null | Site base URL |
| `title` | string? | null | Site title |
| `description` | string? | null | Site description |
| `author` | string? | null | Default author |
| `assets_dir` | Path | `"static"` | Static assets directory |
| `templates_dir` | Path | `"templates"` | Templates directory |

### Nested Configuration

Plugins add their own sections:

```toml
[tool-name]
output_dir = "public"
url = "https://example.com"

[tool-name.glob]
glob_patterns = ["posts/**/*.md"]
use_gitignore = true

[tool-name.markdown]
extensions = ["tables", "admonitions"]

[[tool-name.feeds]]
slug = "blog"
filter = "published == True"
```

### Hooks Configuration

The `hooks` field controls which plugins are loaded:

```toml
# Load only default plugins
hooks = ["default"]

# Load default plus custom
hooks = ["default", "my_plugins.reading_time"]

# Load only specific plugins
hooks = [
    "your_ssg.plugins.glob",
    "your_ssg.plugins.load",
    "your_ssg.plugins.render",
    "your_ssg.plugins.save",
]
```

The special value `"default"` expands to the standard plugin set.

### Disabled Hooks

Exclude specific plugins:

```toml
hooks = ["default"]
disabled_hooks = ["toc", "wikilinks"]
```

---

## Model Extension

### How Plugins Extend Models

Plugins contribute model fragments that are merged:

```python
# Plugin A contributes:
class PostFieldsA(BaseModel):
    reading_time: int = 0

# Plugin B contributes:
class PostFieldsB(BaseModel):
    word_count: int = 0
    featured: bool = False

# Core merges into:
class Post(PostFieldsA, PostFieldsB, BasePost):
    pass

# Result: Post has reading_time, word_count, featured, plus base fields
```

### Merge Rules

1. **Field conflicts:** Later plugins override earlier plugins
2. **Type conflicts:** Must be compatible or error
3. **Default conflicts:** Later default wins
4. **Validator conflicts:** All validators run

### Type Mapping

Abstract types map to language-specific types:

| Abstract | Python | TypeScript | Go | Rust |
|----------|--------|------------|-----|------|
| string | str | string | string | String |
| string? | Optional[str] | string \| null | *string | Option<String> |
| int | int | number | int | i64 |
| float | float | number | float64 | f64 |
| bool | bool | boolean | bool | bool |
| date | datetime.date | Date | time.Time | NaiveDate |
| datetime | datetime.datetime | Date | time.Time | DateTime |
| string[] | List[str] | string[] | []string | Vec<String> |
| Path | pathlib.Path | string | string | PathBuf |
| URL | pydantic.AnyUrl | string | *url.URL | Url |
| any | Any | any | interface{} | serde_json::Value |

---

## Validation

### Required vs Optional

```python
# Required field - must be provided
class Post(BaseModel):
    path: Path  # Required

# Optional with None default
class Post(BaseModel):
    title: Optional[str] = None  # Optional

# Optional with value default
class Post(BaseModel):
    published: bool = False  # Optional, defaults to False
```

### Custom Validators

```python
from pydantic import field_validator

class Post(BaseModel):
    date: Optional[date] = None

    @field_validator("date", mode="before")
    @classmethod
    def parse_date(cls, v):
        if isinstance(v, str):
            return parse_date_string(v)
        return v
```

### Computed Fields

Fields derived from other fields:

```python
from pydantic import computed_field

class Post(BaseModel):
    title: Optional[str] = None
    path: Path

    @computed_field
    @property
    def slug(self) -> str:
        if self.title:
            return slugify(self.title)
        return self.path.stem
```

---

## Frontmatter

### Format

YAML between `---` delimiters:

```markdown
---
title: My Post
date: 2024-01-15
tags:
  - python
  - tutorial
published: true
---

Content starts here.
```

### Parsing

1. Check for opening `---` at start of file
2. Find closing `---`
3. Parse YAML between delimiters
4. Remainder is content

### Edge Cases

| Case | Behavior |
|------|----------|
| No frontmatter | All content, empty metadata |
| Empty frontmatter (`---\n---`) | Empty metadata, all content |
| Invalid YAML | Error with file path |
| Extra `---` in content | Only first two delimiters matter |

### Unknown Fields

Frontmatter may contain fields not in the Post model:

**Options:**
1. **Strict:** Error on unknown fields
2. **Ignore:** Silently drop unknown fields (default)
3. **Extra:** Store in an `extra` dict

```python
class Post(BaseModel):
    model_config = ConfigDict(extra="allow")  # Store unknown fields
```

---

## Querying

### Filter Syntax

Filter posts with Python-like expressions:

```python
# Boolean comparison
core.filter("published == True")
core.filter("draft == False")

# Date comparison
core.filter("date <= today")
core.filter("date >= date(2024, 1, 1)")

# String containment
core.filter("'python' in tags")
core.filter("'tutorial' in title.lower()")

# Compound expressions
core.filter("published == True and date <= today")
core.filter("status == 'draft' or 'wip' in tags")

# Negation
core.filter("not skip")
core.filter("not draft")
```

### Built-in Variables

Available in filter expressions:

| Variable | Type | Value |
|----------|------|-------|
| `today` | date | Current date |
| `now` | datetime | Current datetime |
| `True` | bool | Boolean true |
| `False` | bool | Boolean false |
| `None` | null | Null value |

### Map Function

Extract fields from posts:

```python
# Get single field
titles = core.map("title")  # ["Post 1", "Post 2", ...]

# With filter
published_titles = core.map("title", filter="published == True")

# With sort
sorted_titles = core.map("title", sort="date", reverse=True)

# Get post objects
posts = core.map("post", filter="True")
```

### Convenience Methods

```python
# Get first matching post
latest = core.first(filter="published == True", sort="date", reverse=True)

# Get last matching post  
oldest = core.last(filter="published == True", sort="date")

# Get exactly one match (error if 0 or >1)
about = core.one(filter="slug == 'about'")
```

---

## Serialization

### To JSON

Posts should serialize to JSON for:
- API responses
- Template data
- Caching

```python
post.model_dump_json()  # Pydantic v2

# Output:
{
  "path": "posts/hello.md",
  "slug": "hello-world",
  "title": "Hello World",
  "date": "2024-01-15",
  "published": true,
  "tags": ["python"],
  "content": "...",
  "html": "..."
}
```

### Excluded Fields

Some fields should not serialize:

```python
class Post(BaseModel):
    path: Path
    content: str
    markata: Any = Field(exclude=True)  # Don't serialize
```

---

## Post Mutation

### During Lifecycle

- Posts are **mutable** during lifecycle stages
- Changes are held in memory only
- Changes are **NOT** persisted back to source files
- Each stage sees mutations from previous stages

### Field Mutation Rules

| Field | Mutable | Notes |
|-------|---------|-------|
| `path` | No | Set at load, immutable |
| `content` | Yes | Can be modified in pre_render |
| `slug` | Yes | If changed, href auto-updates |
| `href` | Computed | Derived from slug |
| `article_html` | Yes | Set in render stage |
| `html` | Yes | Set in post_render stage |
| Custom fields | Yes | Via dynamic field API |

### Dynamic Fields

For fields not in the Post schema, use dynamic field storage:

```python
# Set a dynamic field
post.set("reading_time", "5 min read")

# Get a dynamic field
rt = post.get("reading_time")

# Check if field exists
if post.has("reading_time"): ...

# All dynamic fields
extras = post.extra()  # returns dict
```

### Computed Field Recalculation

When `slug` is modified, `href` is automatically recalculated:

```python
post.slug = "new-slug"
# post.href is now "/new-slug/"
```

---

## Error Types

### Structured Errors

Errors should include contextual information:
- `stage` - which lifecycle stage
- `plugin` - which plugin caused it
- `path` - file path if applicable
- `line` - line number if applicable
- `message` - human-readable description
- `cause` - underlying error

### Standard Error Types

| Error | When |
|-------|------|
| `FrontmatterParseError` | Invalid YAML in frontmatter |
| `FilterExpressionError` | Invalid filter syntax |
| `TemplateNotFoundError` | Template file doesn't exist |
| `TemplateSyntaxError` | Invalid template syntax |
| `ConfigValidationError` | Invalid configuration |
| `PluginNotFoundError` | Plugin couldn't be loaded |
| `CircularTemplateError` | Template inheritance cycle |
| `PathConflictError` | Multiple posts/feeds would write to same path |

---

## Path Conflict Detection

The build system detects when multiple content sources would write to the same output path. This prevents accidental content overwrites.

### Conflict Sources

Conflicts can occur between:
- **Post vs Post**: Two posts with the same slug
- **Post vs Feed**: A post slug matching a feed slug
- **Feed vs Feed**: Two feeds with the same slug

### Example Conflicts

```yaml
# posts/home.md
---
title: Welcome Home
slug: ""          # Would write to /index.html
---

# posts/landing.md
---
title: Landing Page
slug: /           # Also writes to /index.html - CONFLICT!
---
```

```toml
# markata-go.toml
[[feeds]]
slug = "blog"     # Would write to /blog/index.html

# posts/blog.md
---
title: Blog Post
slug: blog        # Also writes to /blog/index.html - CONFLICT!
---
```

### Conflict Resolution

When a conflict is detected:
1. The build fails with a clear error message listing all conflicting paths
2. Each conflict shows the sources that would collide
3. Users must resolve by changing one of the slugs

**Error example:**
```
detected 1 output path conflict(s):
  - public/index.html: post:pages/home.md, post:pages/landing.md
```

### Warning Mode

The overwrite check can be configured to warn instead of fail:

```toml
[plugins.overwrite_check]
warn_only = true  # Warn but don't fail build
```

---

## See Also

- [SPEC.md](./SPEC.md) - Full specification
- [CONFIG.md](./CONFIG.md) - Configuration system
- [PLUGINS.md](./PLUGINS.md) - How plugins extend models
- [CONTENT.md](./CONTENT.md) - Content processing
