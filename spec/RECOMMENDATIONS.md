# Spec Review: Recommendations

> Review of the static site generator specification with actionable improvements.

## Overall Assessment

This is a **well-designed spec** inspired by markata and the "spec-as-product" philosophy. The documentation is comprehensive, covering lifecycle, plugins, data models, templates, and content processing. However, there are several areas that need clarification or expansion before implementation.

---

## Strengths

1. **Clear separation of concerns** - Core vs plugins philosophy is solid
2. **Comprehensive lifecycle** - 13 stages cover all SSG needs
3. **Language-agnostic design** - Spec works for Python, TypeScript, Go, Rust, etc.
4. **Detailed test cases** - `tests.yaml` provides excellent behavior specifications

---

## High Priority Recommendations

### 1. Add Concurrency Model

**Problem:** The spec doesn't address parallelism, which is crucial for performance.

**Solution:** Add the following to SPEC.md:

```markdown
## Concurrency

### Post Processing
- `pre_render`, `render`, `post_render` stages SHOULD process posts concurrently
- Use worker pool pattern with configurable concurrency limit
- Default: number of CPU cores

### Thread Safety
- Plugins MUST NOT modify shared state without synchronization
- Each post can be processed independently
- Core.Posts list is read-only during concurrent stages

### Configuration
```toml
[ssg]
concurrency = 4  # 0 = auto (CPU count)
```
```

---

### 2. Document Plugin Discovery/Resolution

**Problem:** PLUGINS.md shows registration but not how plugins are discovered from config strings.

**Solution:** Add to PLUGINS.md:

```markdown
## Plugin Resolution

When loading plugins from the `hooks` config, resolution follows this order:

1. **"default"** - Expands to the built-in plugin set:
   - glob, load, render_markdown, templates, feeds, publish_html, rss, sitemap

2. **Built-in name** (e.g., `"glob"`, `"render_markdown"`)
   - Resolves to internal plugins

3. **Full module path** (language-specific)
   - Python: `"my_package.plugins.custom"`
   - TypeScript: `"my-package/plugins/custom"`
   - Go: `"github.com/user/ssg-plugin"`
   - Rust: `"my_crate::plugins::custom"`

4. **Local path** (e.g., `"./plugins/myplugin"`)
   - Loads from project-relative path

### Example
```toml
[ssg]
hooks = [
    "default",                    # Built-in set
    "./plugins/reading_time",     # Local plugin
]
disabled_hooks = ["rss", "sitemap"]
```
```

---

### 3. Define Core Interface

**Problem:** PLUGINS.md shows plugin interfaces but the Core interface they interact with is undefined.

**Solution:** Add to SPEC.md under "Core Components":

```markdown
## Core Interface

The central orchestrator that plugins interact with.

### Properties
| Property | Type | Description |
|----------|------|-------------|
| `config` | Config | Merged configuration object |
| `posts` | List[Post] | All loaded content items |
| `files` | List[Path] | Discovered content files |
| `cache` | Cache | Persistent key-value cache |

### Query Methods
| Method | Signature | Description |
|--------|-----------|-------------|
| `filter` | `(expr: string) -> List[Post]` | Query posts with expression |
| `map` | `(field: string, filter?: string, sort?: string, reverse?: bool) -> List[Any]` | Extract field from filtered posts |
| `first` | `(filter?: string, sort?: string) -> Post?` | First matching post |
| `last` | `(filter?: string, sort?: string) -> Post?` | Last matching post |
| `one` | `(expr: string) -> Post` | Exactly one match (error if 0 or >1) |

### Lifecycle Methods
| Method | Description |
|--------|-------------|
| `register(plugin)` | Register a plugin |
| `run(stage?)` | Run build up to optional stage |

### Attribute Storage (for plugin communication)
| Method | Description |
|--------|-------------|
| `set(key, value)` | Store a value |
| `get(key) -> Any` | Retrieve a value |
| `has(key) -> bool` | Check if key exists |
```

---

## Medium Priority Recommendations

### 4. Define Error Types

**Problem:** Error handling section mentions error names but doesn't define them.

**Solution:** Add to DATA_MODEL.md:

```markdown
## Error Types

### Structured Error
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
```

---

### 5. Add Development Server Specification

**Problem:** INSTALL.md mentions `serve` command but no spec exists.

**Solution:** Add to SPEC.md or create CLI.md:

```markdown
## Development Server

### Features
- HTTP server for previewing the built site
- File watching for automatic rebuilds
- Live reload for browser refresh

### Configuration
```toml
[ssg.serve]
port = 3000
host = "localhost"
livereload = true
open_browser = false
debounce_ms = 100  # wait before rebuilding after file change
```

### Rebuild Triggers
| Change Type | Action |
|-------------|--------|
| Content file (*.md) | Incremental rebuild |
| Template file | Full rebuild |
| Config file | Full rebuild |
| Static asset | Copy only |

### Live Reload
- Inject reload script into HTML pages during serve
- Use WebSocket or Server-Sent Events
- Trigger reload after successful build
```

---

### 6. Clarify Post Mutation Behavior

**Problem:** Can plugins mutate posts directly? When are changes persisted?

**Solution:** Add to DATA_MODEL.md:

```markdown
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

### Dynamic Fields
For fields not in the Post schema, use dynamic field storage:

```python
# Set a dynamic field
post.set("reading_time", "5 min read")

# Get a dynamic field
rt = post.get("reading_time")

# Check if field exists
if post.has("reading_time"): ...
```
```

---

### 7. Add Incremental Build Specification

**Problem:** No specification for incremental/watch builds.

**Solution:** Add to LIFECYCLE.md:

```markdown
## Incremental Builds

### Change Detection
Track file state in cache:
- Path
- Modification time
- Content hash

### Rebuild Strategy
| Change | Rebuild Scope |
|--------|---------------|
| Single post content | That post only |
| Post frontmatter | That post + feeds containing it |
| Template file | All posts using that template |
| Config file | Full rebuild |
| Plugin change | Full rebuild |

### Cache Keys
```
post:{path}:mtime     → file modification time
post:{path}:hash      → content hash
post:{path}:rendered  → cached article_html
template:{path}:mtime → template modification time
```
```

---

## Low Priority Recommendations

### 8. Document Template Engine Differences

**Problem:** TEMPLATES.md shows Jinja syntax but other engines have differences.

**Solution:** Add to TEMPLATES.md:

```markdown
## Template Engine Compatibility

This spec uses Jinja2-style syntax. Other engines have differences:

### Date Formatting
| Engine | Syntax |
|--------|--------|
| Jinja2 (Python) | `{{ date \| strftime('%Y-%m-%d') }}` |
| Nunjucks (JS) | `{{ date \| date('YYYY-MM-DD') }}` |
| pongo2 (Go) | `{{ date \| date:"2006-01-02" }}` |
| Tera (Rust) | `{{ date \| date(format="%Y-%m-%d") }}` |

### Struct/Object Access
| Engine | Syntax |
|--------|--------|
| Jinja2 | `post.title` |
| pongo2 | `post.Title` (exported fields) |

Implementations should document any deviations from Jinja2 syntax.
```

---

### 9. Add Full CLI Specification

**Problem:** Commands are mentioned but not fully specified.

**Solution:** Add to SPEC.md or create CLI.md:

```markdown
## CLI Commands

### `build`
Build the static site.

```bash
ssg build [flags]

Flags:
  -c, --config string   Config file path (default "ssg.toml")
  -o, --output string   Output directory (overrides config)
  -v, --verbose         Verbose output
  --clean               Remove output dir before build
  --drafts              Include draft posts
  --future              Include future-dated posts
```

### `serve`
Start development server with live reload.

```bash
ssg serve [flags]

Flags:
  -p, --port int        Server port (default 3000)
  -H, --host string     Bind address (default "localhost")
  --no-reload           Disable live reload
  --no-watch            Disable file watching
```

### `new <path>`
Create a new post with frontmatter template.

```bash
ssg new posts/my-new-post.md [flags]

Flags:
  -t, --title string    Post title
  --edit                Open in $EDITOR after creation
```

### `validate`
Validate configuration and content without building.

```bash
ssg validate [flags]

Flags:
  --strict              Treat warnings as errors
```

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Build error |
| 2 | Config error |
| 3 | Validation error |
```

---

### 10. Add Asset Pipeline Specification

**Problem:** `copy_assets` plugin mentioned but no processing spec.

**Solution:** Add to SPEC.md under Standard Plugins:

```markdown
### Asset Processing (`assets`)

#### Copy Assets
- Copy all files from `assets_dir` to `output_dir`
- Preserve directory structure
- Skip files matching ignore patterns

#### Configuration
```toml
[ssg.assets]
dir = "static"
ignore = ["*.scss", "*.ts", ".DS_Store"]
```

#### Optional: Asset Fingerprinting
Add content hash to filenames for cache busting:

```toml
[ssg.assets]
fingerprint = true
fingerprint_extensions = [".css", ".js"]
```

- `style.css` → `style.a1b2c3d4.css`
- Update references in HTML automatically
```

---

## Additional Test Cases

Add these to `tests.yaml`:

```yaml
# =============================================================================
# CONCURRENCY
# =============================================================================
concurrency:
  - name: "concurrent post rendering produces correct results"
    input:
      posts_count: 100
      concurrency: 8
    output:
      all_posts_rendered: true

# =============================================================================
# EDGE CASES
# =============================================================================
edge_cases:
  - name: "large post (10k+ words)"
    input:
      content_words: 10000
    output:
      renders_successfully: true

  - name: "unicode filename"
    input:
      path: "posts/日本語の記事.md"
    output:
      loads_successfully: true

  - name: "circular template inheritance"
    input:
      templates:
        a.html: "{% extends 'b.html' %}"
        b.html: "{% extends 'a.html' %}"
    error: true
    error_type: "CircularTemplateError"

  - name: "plugin error isolation"
    input:
      plugins:
        - { name: "failing_plugin", throws: true }
        - { name: "good_plugin" }
      continue_on_error: true
    output:
      good_plugin_ran: true
      build_completed: true

  - name: "empty content directory"
    input:
      files: []
    output:
      posts_count: 0
      build_succeeds: true
```

---

## Updated Implementation Checklist

```markdown
## Implementation Checklist

An implementation is complete when it:

### Core
- [ ] Runs lifecycle stages in order
- [ ] Supports concurrent post processing
- [ ] Loads plugins from configuration
- [ ] Resolves plugin names to implementations
- [ ] Merges config from multiple sources
- [ ] Provides thread-safe attribute storage

### Content
- [ ] Discovers content files via glob patterns
- [ ] Parses YAML frontmatter
- [ ] Handles unicode filenames
- [ ] Supports `filter()` with expressions
- [ ] Supports `map()` for field extraction

### Rendering
- [ ] Renders markdown to HTML
- [ ] Processes templates
- [ ] Supports template inheritance
- [ ] Detects circular template inheritance

### Output
- [ ] Generates feeds with filtering/sorting/pagination
- [ ] Writes output to configured directory
- [ ] Copies static assets
- [ ] Generates RSS feed
- [ ] Generates sitemap

### Caching
- [ ] Caches expensive operations
- [ ] Supports incremental builds
- [ ] Invalidates cache on config change

### CLI
- [ ] `build` command with all flags
- [ ] `serve` command with live reload
- [ ] `new` command for post creation
- [ ] `validate` command for checking

### Plugins
- [ ] Allows single-file plugin creation
- [ ] Plugin can extend config schema
- [ ] Plugin can extend post model
- [ ] Plugin can add CLI commands
- [ ] Plugin errors don't crash build (configurable)

### Testing
- [ ] Passes all tests in `tests.yaml`
- [ ] Handles edge cases (large files, unicode, etc.)
```

---

## Summary

| Priority | Issue | Action |
|----------|-------|--------|
| **High** | No concurrency model | Add to SPEC.md |
| **High** | Plugin discovery unclear | Add to PLUGINS.md |
| **High** | Core interface undefined | Add to SPEC.md |
| **Medium** | Error types undefined | Add to DATA_MODEL.md |
| **Medium** | No dev server spec | Add to SPEC.md or CLI.md |
| **Medium** | Post mutation unclear | Add to DATA_MODEL.md |
| **Medium** | No incremental build spec | Add to LIFECYCLE.md |
| **Low** | Template engine differences | Add to TEMPLATES.md |
| **Low** | CLI not fully specified | Add to SPEC.md or CLI.md |
| **Low** | Asset pipeline vague | Add to SPEC.md |

Addressing these recommendations will make the spec implementation-ready and reduce ambiguity for developers and AI agents.
