# Migration Tool Specification

This document specifies the migration tool for helping users migrate from Python markata to markata-go.

## Overview

The migration tool provides automated assistance for users transitioning from Python markata to markata-go. It handles:

1. **Configuration migration** - Converting Python markata config to markata-go format
2. **Filter expression migration** - Adapting Python-style filter expressions
3. **Template compatibility checking** - Identifying template changes needed
4. **Migration reporting** - Generating actionable migration guidance

## CLI Commands

### `markata-go migrate`

Full migration analysis and transformation.

```bash
markata-go migrate [flags]
```

**Flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input` | `-i` | `markata.toml` | Input config file path |
| `--output` | `-o` | `markata-go.toml` | Output config file path |
| `--dry-run` | `-n` | `false` | Show changes without writing |
| `--format` | `-f` | `toml` | Output format (toml, yaml, json) |

**Example:**
```bash
# Analyze and migrate
markata-go migrate

# Dry run to see what would change
markata-go migrate --dry-run

# Specify input/output files
markata-go migrate -i pyproject.toml -o markata-go.toml
```

### `markata-go migrate config`

Migrate configuration file only.

```bash
markata-go migrate config [flags]
```

**Flags:** Same as `markata-go migrate`

### `markata-go migrate filter`

Check and migrate filter expressions.

```bash
markata-go migrate filter [expression]
```

**Arguments:**
- `expression` - Filter expression to check/migrate (optional, reads from config if omitted)

**Example:**
```bash
# Check a specific expression
markata-go migrate filter "published == 'True'"

# Check all filters in config
markata-go migrate filter
```

### `markata-go migrate templates`

Validate template compatibility.

```bash
markata-go migrate templates [path]
```

**Arguments:**
- `path` - Templates directory to check (default: `templates/`)

---

## Configuration Migration

### Namespace Changes

| Python markata | markata-go | Notes |
|----------------|------------|-------|
| `[markata]` | `[markata-go]` | Root namespace |
| `[markata.feeds]` | `[markata-go.feeds]` | Feed config |
| `[markata.nav]` | `[markata-go.nav]` | Navigation |

### Key Renames

| Python markata | markata-go | Notes |
|----------------|------------|-------|
| `glob_patterns` | `patterns` | Under `[markata-go.glob]` |
| `author_name` | `author` | Root level |
| `site_name` | `title` | Root level |
| `site_description` | `description` | Root level |
| `color_theme` | `theme.palette` | Nested under theme |
| `output` | `output_dir` | Root level |

### Nav Configuration Migration

Python markata uses a map for navigation:

```toml
# Python markata
[markata.nav]
home = "/"
blog = "/blog"
about = "/about"
```

markata-go uses an array with explicit labels:

```toml
# markata-go
[[markata-go.nav]]
label = "Home"
url = "/"

[[markata-go.nav]]
label = "Blog"
url = "/blog"

[[markata-go.nav]]
label = "About"
url = "/about"
```

### Feed Configuration Migration

Python markata feed syntax:

```toml
# Python markata
[[markata.feeds]]
slug = "blog"
filter = "templateKey in ['blog-post', 'til']"
```

markata-go feed syntax:

```toml
# markata-go
[[markata-go.feeds]]
slug = "blog"
filter = "templateKey == 'blog-post' or templateKey == 'til'"
```

### Unsupported Features

The following Python markata features are not yet supported in markata-go:

| Feature | Status | Alternative |
|---------|--------|-------------|
| Custom Python hooks | Not supported | Use Go plugins |
| `jinja_md` with Python | Limited | Use pongo2 templates |
| Rich console output | Not supported | Plain text output |
| `post.markata` access | Not supported | Use template variables |

---

## Filter Expression Migration

### Boolean Literals

Python-style quoted booleans must be converted to unquoted:

| Python markata | markata-go |
|----------------|------------|
| `published == 'True'` | `published == True` |
| `published == 'False'` | `published == False` |
| `draft == 'true'` | `draft == True` |
| `draft != 'false'` | `draft != False` |

### `in` Operator

The `in` operator with lists is not supported. Convert to `or` expressions:

| Python markata | markata-go |
|----------------|------------|
| `templateKey in ['blog-post', 'til']` | `templateKey == 'blog-post' or templateKey == 'til'` |
| `status in ['draft', 'review']` | `status == 'draft' or status == 'review'` |
| `tag in ['python', 'go']` | `tag == 'python' or tag == 'go'` |

### Operator Spacing

Operators require surrounding whitespace:

| Python markata | markata-go |
|----------------|------------|
| `date<=today` | `date <= today` |
| `count>=10` | `count >= 10` |
| `title!='test'` | `title != 'test'` |

### None/Null Values

| Python markata | markata-go |
|----------------|------------|
| `image == None` | `image == None` |
| `image is None` | `image == None` |
| `image is not None` | `image != None` |

### String Comparisons

Both single and double quotes are supported:

| Python markata | markata-go |
|----------------|------------|
| `title == "Hello"` | `title == "Hello"` |
| `title == 'Hello'` | `title == 'Hello'` |

---

## Template Compatibility

### Supported Template Features

markata-go uses pongo2 (Jinja2-like) templates. Most Jinja2 features are supported:

- Variable interpolation: `{{ variable }}`
- Filters: `{{ title|lower }}`
- Conditionals: `{% if condition %}...{% endif %}`
- Loops: `{% for item in items %}...{% endfor %}`
- Includes: `{% include "partial.html" %}`
- Extends: `{% extends "base.html" %}`
- Blocks: `{% block content %}...{% endblock %}`

### Unsupported Template Features

| Feature | Status | Alternative |
|---------|--------|-------------|
| `do` statement | Not supported | Use `{% set %}` |
| `with` statement | Limited | Use `{% set %}` |
| Macros | Not supported | Use includes |
| Call blocks | Not supported | Use includes |
| Python expressions | Not supported | Use pongo2 filters |

### Variable Changes

| Python markata | markata-go | Notes |
|----------------|------------|-------|
| `post.markata.config` | `config` | Direct access |
| `post.markata.feeds` | `feeds` | Direct access |
| `post.content` | `post.content` | Same |
| `post.article_html` | `post.content` | Renamed |

---

## Migration Report Format

### Report Structure

```
================================================================================
                        markata-go Migration Report
================================================================================

Configuration File: markata.toml
Generated: 2024-01-15 10:30:00

--------------------------------------------------------------------------------
SUMMARY
--------------------------------------------------------------------------------

  Status: Ready to migrate (with warnings)

  Changes required:    12
  Warnings:            3
  Incompatibilities:   1

--------------------------------------------------------------------------------
CONFIGURATION CHANGES
--------------------------------------------------------------------------------

  [MIGRATE] Namespace: [markata] -> [markata-go]
  [MIGRATE] Key: glob_patterns -> patterns
  [MIGRATE] Key: author_name -> author
  [MIGRATE] Key: output -> output_dir
  [MIGRATE] Nav: map -> array (3 items)

--------------------------------------------------------------------------------
FILTER MIGRATIONS
--------------------------------------------------------------------------------

  Feed: blog
    [MIGRATE] published == 'True' -> published == True
    [MIGRATE] templateKey in ['blog-post', 'til'] -> templateKey == 'blog-post' or templateKey == 'til'

  Feed: archive
    [OK] date <= today (no changes needed)

--------------------------------------------------------------------------------
WARNINGS
--------------------------------------------------------------------------------

  [WARN] Custom hook 'my_plugin' in hooks list - not supported in markata-go
  [WARN] jinja_md blocks use Python expressions - manual review needed
  [WARN] 'post.markata' access in templates - update to use 'config' directly

--------------------------------------------------------------------------------
INCOMPATIBILITIES
--------------------------------------------------------------------------------

  [ERROR] Plugin 'rich_output' is not available in markata-go

--------------------------------------------------------------------------------
NEXT STEPS
--------------------------------------------------------------------------------

  1. Review the warnings above
  2. Run: markata-go migrate -o markata-go.toml
  3. Update templates as noted in warnings
  4. Test with: markata-go build --dry-run
  5. Full build: markata-go build

================================================================================
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Migration successful, no issues |
| 1 | Migration completed with warnings |
| 2 | Migration has incompatibilities |
| 3 | Migration failed (invalid input) |

---

## Data Models

### MigrationResult

```go
type MigrationResult struct {
    // InputFile is the source config file path
    InputFile string

    // OutputFile is the target config file path
    OutputFile string

    // Changes is the list of configuration changes made
    Changes []ConfigChange

    // FilterMigrations is the list of filter expression migrations
    FilterMigrations []FilterMigration

    // Warnings is the list of non-blocking issues
    Warnings []Warning

    // Errors is the list of blocking issues
    Errors []MigrationError

    // TemplateIssues is the list of template compatibility issues
    TemplateIssues []TemplateIssue
}
```

### ConfigChange

```go
type ConfigChange struct {
    // Type is the change type: "namespace", "rename", "transform", "remove"
    Type string

    // Path is the config path (e.g., "markata.nav")
    Path string

    // OldValue is the original value
    OldValue interface{}

    // NewValue is the migrated value
    NewValue interface{}

    // Description explains the change
    Description string
}
```

### FilterMigration

```go
type FilterMigration struct {
    // Feed is the feed name this filter belongs to
    Feed string

    // Original is the original filter expression
    Original string

    // Migrated is the migrated filter expression
    Migrated string

    // Changes lists specific transformations applied
    Changes []string

    // Valid indicates if the migrated filter is valid
    Valid bool

    // Error contains any migration error
    Error string
}
```

### Warning

```go
type Warning struct {
    // Category groups related warnings
    Category string // "config", "filter", "template", "plugin"

    // Message describes the warning
    Message string

    // Path is the config path or file path
    Path string

    // Suggestion provides actionable guidance
    Suggestion string
}
```

### MigrationError

```go
type MigrationError struct {
    // Category groups related errors
    Category string

    // Message describes the error
    Message string

    // Path is the config path or file path
    Path string

    // Fatal indicates if migration cannot continue
    Fatal bool
}
```

### TemplateIssue

```go
type TemplateIssue struct {
    // File is the template file path
    File string

    // Line is the line number
    Line int

    // Issue describes the compatibility issue
    Issue string

    // Severity is "error", "warning", or "info"
    Severity string

    // Suggestion provides fix guidance
    Suggestion string
}
```

---

## Package Structure

```
pkg/migrate/
├── config.go          # Config transformation logic
├── filter.go          # Filter expression migration
├── templates.go       # Template compatibility checking
├── report.go          # Report generation
├── migrate.go         # Main migration orchestration
├── migrate_test.go    # Tests
└── doc.go             # Package documentation
```

---

## See Also

- [CONFIG.md](./CONFIG.md) - Configuration system specification
- [FILTERS.md](./FILTERS.md) - Filter expression syntax (if exists)
- [TEMPLATES.md](./TEMPLATES.md) - Template system specification
