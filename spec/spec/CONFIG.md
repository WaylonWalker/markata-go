# Configuration System Specification

The configuration system is designed to be:
- **Discoverable** - Plugins declare their config with descriptions
- **Flexible** - Multiple file formats and locations
- **Hierarchical** - Namespaced under the tool name
- **Mergeable** - Multiple sources combine intelligently

## Configuration Discovery

### File Locations

The system searches for configuration files in this order (first found wins):

```
1. CLI-specified:        --config path/to/config.toml
2. Current directory:    ./[name].toml (or .yaml, .yml, .json, .jsonc)
3. Current directory:    ./[name]/config.toml
4. pyproject.toml:       ./pyproject.toml (under [tool.name] section)
5. package.json:         ./package.json (under "name" key)
6. User config dir:      ~/.config/[name]/config.toml
7. User home:            ~/.[name].toml
8. User home dotdir:     ~/.[name]/config.toml
```

### Supported Formats

| Extension | Format | Notes |
|-----------|--------|-------|
| `.toml` | TOML | Recommended, best for nested config |
| `.yaml`, `.yml` | YAML | Good for complex structures |
| `.json` | JSON | Strict, good for programmatic generation |
| `.jsonc` | JSON with comments | JSON + `//` and `/* */` comments |

## Config Composition

markata-go supports config composition through an `include` key under `[markata-go]`.

```toml
[markata-go]
include = [
  "config/base/*.toml",
  "config/feeds/*.toml",
  "config/rss.toml",
]
```

`include` supports explicit file paths, glob patterns, recursive includes, and paths resolved relative to the file that declared the include.

### Composition Order

Resolved precedence is:

1. built-in defaults
2. the root config selected by discovery or `--config`
3. included files in declaration order
4. glob matches in lexicographic order
5. environment variable overrides

Later values win over earlier values.

### Repeated Includes And Cycles

- A file included more than once in the same resolution graph is loaded once.
- Include cycles are rejected with a clear error that shows the cycle path.

### Merge Semantics

- scalar values: last explicit value wins
- explicit `false`, `0`, and `""` count as real overrides
- tables/maps: deep merge
- arrays of scalars: replace
- `[[markata-go.feeds]]`: merge by `slug`

For feeds, a new `slug` appends a new feed. A repeated `slug` merges into the existing feed, and later fragments win on conflicts.

### Format Examples

**TOML (recommended):**
```toml
[my-ssg]
output_dir = "public"
url = "https://example.com"

[my-ssg.feeds.defaults]
items_per_page = 10

[[my-ssg.feeds]]
slug = "blog"
filter = "published == True"
```

**YAML:**
```yaml
my-ssg:
  output_dir: public
  url: https://example.com

  feeds:
    defaults:
      items_per_page: 10

    items:
      - slug: blog
        filter: "published == True"
```

**JSON:**
```json
{
  "my-ssg": {
    "output_dir": "public",
    "url": "https://example.com",
    "feeds": {
      "defaults": {
        "items_per_page": 10
      },
      "items": [
        {
          "slug": "blog",
          "filter": "published == True"
        }
      ]
    }
  }
}
```

---

## Configuration Namespacing

All configuration lives under the tool name namespace:

```toml
# Root namespace - minimal, mostly metadata
[my-ssg]
output_dir = "public"
url = "https://example.com"

# Plugin namespaces
[my-ssg.glob]
patterns = ["**/*.md"]

[my-ssg.markdown]
extensions = ["tables", "admonitions"]

[my-ssg.feeds]
# Feed-specific config

[my-ssg.serve]
port = 3000
```

### Why Namespacing?

1. **Avoids conflicts** with other tools in shared config files (`pyproject.toml`)
2. **Clear ownership** - each plugin owns its namespace
3. **Tooling friendly** - editors can provide completions per-namespace
4. **Discoverable** - `my-ssg config list` shows all namespaces

### Root-Level Fields

Only essential, cross-cutting concerns live at the root:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `output_dir` | Path | `"output"` | Build output directory |
| `url` | URL? | null | Site base URL (needed by many plugins) |
| `title` | string? | null | Site title |
| `description` | string? | null | Site description |
| `author` | string? | null | Default author |
| `license` | string `\|` bool | `(unset)` | Select from the supported license keys below or set to `false` to disable the footer attribution and associated warning. |
| `lang` | string | `"en"` | Site language |
| `hooks` | string[] | `["default"]` | Plugins to load |
| `disabled_hooks` | string[] | `[]` | Plugins to exclude |

Everything else goes in a plugin namespace.

### License configuration

The root `license` key lets you declare how visitors may reuse your content. It accepts either a string key or the literal `false` value:

- **String keys** register a license that is rendered in the footer and made available to templates via `config.license`. Use one of the supported values:
  - `all-rights-reserved` – All rights reserved (no reuse allowed).
  - `cc-by-4.0` (recommended) – Creative Commons Attribution 4.0 International (`https://creativecommons.org/licenses/by/4.0/`).
  - `cc-by-sa-4.0` – Creative Commons Attribution-ShareAlike 4.0 International (`https://creativecommons.org/licenses/by-sa/4.0/`).
  - `cc-by-nc-4.0` – Creative Commons Attribution-NonCommercial 4.0 International (`https://creativecommons.org/licenses/by-nc/4.0/`).
  - `cc-by-nd-4.0` – Creative Commons Attribution-NoDerivatives 4.0 International (`https://creativecommons.org/licenses/by-nd/4.0/`).
  - `cc-by-nc-sa-4.0` – Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International (`https://creativecommons.org/licenses/by-nc-sa/4.0/`).
  - `mit` – MIT License (`https://opensource.org/licenses/MIT`).

- **Boolean `false`** suppresses the license footer and prevents the validation warning (useful for sites that intentionally publish without an explicit license).
- **Omitted key** (default) triggers a validation warning and the serve banner/toast reminder until a license string is configured or `false` is set.

The default scaffolding details from `markata-go config init` include `license = "cc-by-4.0"`, so new sites ship with the recommended Creative Commons attribution out of the box.

---

## Plugin Configuration Declaration

Plugins declare their configuration schema, defaults, and descriptions. This enables:
- Config validation
- Auto-generated documentation
- Editor completions
- CLI config helpers

### Declaration Format

Each plugin exports a config schema:

**Python (Pydantic):**
```python
from pydantic import BaseModel, Field

class GlobConfig(BaseModel):
    """Configuration for the glob plugin."""

    patterns: list[str] = Field(
        default=["**/*.md"],
        description="Glob patterns to find content files"
    )
    use_gitignore: bool = Field(
        default=True,
        description="Respect .gitignore when finding files"
    )
    exclude: list[str] = Field(
        default=[],
        description="Patterns to exclude from results"
    )

class Config(BaseModel):
    """Adds glob section to config."""
    glob: GlobConfig = Field(
        default_factory=GlobConfig,
        description="File discovery settings"
    )

@hook_impl
def config_model(core):
    core.register_config(Config, namespace="glob")
```

**TypeScript (Zod):**
```typescript
import { z } from 'zod';

export const GlobConfig = z.object({
  patterns: z.array(z.string())
    .default(["**/*.md"])
    .describe("Glob patterns to find content files"),

  use_gitignore: z.boolean()
    .default(true)
    .describe("Respect .gitignore when finding files"),

  exclude: z.array(z.string())
    .default([])
    .describe("Patterns to exclude from results"),
});

export function configModel(core: Core) {
  core.registerConfig(GlobConfig, { namespace: "glob" });
}
```

**Go:**
```go
type GlobConfig struct {
    // Glob patterns to find content files
    Patterns []string `toml:"patterns" default:"[\"**/*.md\"]" description:"Glob patterns to find content files"`

    // Respect .gitignore when finding files
    UseGitignore bool `toml:"use_gitignore" default:"true" description:"Respect .gitignore when finding files"`

    // Patterns to exclude from results
    Exclude []string `toml:"exclude" description:"Patterns to exclude from results"`
}

func (p *GlobPlugin) ConfigModel(core *Core) {
    core.RegisterConfig("glob", GlobConfig{}, ConfigOptions{
        Description: "File discovery settings",
    })
}
```

**Rust:**
```rust
use serde::{Deserialize, Serialize};

/// Configuration for the glob plugin
#[derive(Debug, Deserialize, Serialize)]
#[serde(default)]
pub struct GlobConfig {
    /// Glob patterns to find content files
    #[serde(default = "default_patterns")]
    pub patterns: Vec<String>,

    /// Respect .gitignore when finding files
    #[serde(default = "default_true")]
    pub use_gitignore: bool,

    /// Patterns to exclude from results
    #[serde(default)]
    pub exclude: Vec<String>,
}

fn default_patterns() -> Vec<String> {
    vec!["**/*.md".to_string()]
}

impl Plugin for Glob {
    fn config_model(&self, core: &mut Core) {
        core.register_config::<GlobConfig>("glob");
    }
}
```

### Config Metadata

Each config field should have:

| Metadata | Purpose |
|----------|---------|
| `description` | Human-readable explanation |
| `default` | Default value if not specified |
| `type` | Data type for validation |
| `required` | Whether field must be provided |
| `deprecated` | Mark old config options |
| `env_var` | Environment variable override |
| `examples` | Example values |
| `see_also` | Related config fields |

### Example with Full Metadata

```python
class MarkdownConfig(BaseModel):
    """Markdown rendering configuration."""

    backend: str = Field(
        default="auto",
        description="Markdown parser backend",
        examples=["markdown-it", "commonmark", "mistune"],
    )

    extensions: list[str] = Field(
        default=["tables", "admonitions", "footnotes"],
        description="Markdown extensions to enable",
    )

    highlight_theme: str = Field(
        default="github-dark",
        description="Syntax highlighting theme",
        deprecated="Use markdown.highlight.theme instead",
        see_also=["markdown.highlight"],
    )

    class Highlight(BaseModel):
        """Syntax highlighting settings."""
        enabled: bool = Field(default=True, description="Enable syntax highlighting")
        theme: str = Field(default="github-dark", description="Color theme")
        line_numbers: bool = Field(default=False, description="Show line numbers")

    highlight: Highlight = Field(
        default_factory=Highlight,
        description="Syntax highlighting configuration",
    )
```

---

## Configuration Resolution

### Merge Order (lowest to highest precedence)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CONFIGURATION RESOLUTION                          │
├─────────────────────────────────────────────────────────────────────┤
│  1. Built-in defaults (from plugin declarations)                     │
│  2. Global config file (~/.config/my-ssg/config.toml)               │
│  3. Local config file (./my-ssg.toml)                               │
│  4. Environment variables (MY_SSG_SECTION_KEY)                      │
│  5. CLI arguments (--output-dir public)                             │
│                                                                      │
│  Later sources OVERRIDE earlier sources                              │
│  Nested objects are MERGED, not replaced                            │
└─────────────────────────────────────────────────────────────────────┘
```

### Environment Variables

Environment variables follow the pattern: `{NAME}_{SECTION}_{KEY}`

```bash
# Set output directory
MY_SSG_OUTPUT_DIR=public

# Set nested config
MY_SSG_FEEDS_DEFAULTS_ITEMS_PER_PAGE=20
MY_SSG_MARKDOWN_HIGHLIGHT_THEME=monokai

# Boolean values
MY_SSG_GLOB_USE_GITIGNORE=true
MY_SSG_GLOB_USE_GITIGNORE=1
MY_SSG_GLOB_USE_GITIGNORE=yes

# List values (comma-separated)
MY_SSG_GLOB_PATTERNS="posts/**/*.md,pages/*.md"
```

### CLI Arguments

Common config options have CLI flags:

```bash
my-ssg build --output-dir public --url https://example.com
my-ssg serve --port 8080 --host 0.0.0.0
```

Arbitrary config can be set with `--config` or `-c`:

```bash
my-ssg build -c feeds.defaults.items_per_page=20
my-ssg build -c markdown.highlight.theme=monokai
```

### Merge Behavior

**Scalar values:** Later wins
```toml
# Global: output_dir = "dist"
# Local:  output_dir = "public"
# Result: output_dir = "public"
```

**Objects:** Deep merge
```toml
# Global:
[my-ssg.feeds.defaults.formats]
html = true
rss = true

# Local:
[my-ssg.feeds.defaults.formats]
atom = true

# Result:
html = true   # from global
rss = true    # from global
atom = true   # from local
```

**Lists:** Replace (not append)
```toml
# Global: patterns = ["**/*.md"]
# Local:  patterns = ["posts/*.md", "pages/*.md"]
# Result: patterns = ["posts/*.md", "pages/*.md"]
```

**List append syntax (optional):**
```toml
# To append instead of replace
patterns = ["posts/*.md"]
patterns_append = ["pages/*.md"]
# Result: ["posts/*.md", "pages/*.md"]
```

---

## Configuration CLI

### `config show`

Display resolved configuration:

- bare `my-ssg config` MUST behave like `my-ssg config show`
- `config show` MUST honor the same config resolution path as build commands, including `--config` and `--merge-config`
- conflicting output flags such as `--json` with `--toml` MUST fail with a usage error (exit code `2`)

```bash
$ my-ssg config show
output_dir = "public"
url = "https://example.com"

[glob]
patterns = ["**/*.md"]
use_gitignore = true

[feeds.defaults]
items_per_page = 10
...
```

With source information:

```bash
$ my-ssg config show --sources
output_dir = "public"          # ./my-ssg.toml
url = "https://example.com"    # MY_SSG_URL env var

[glob]
patterns = ["**/*.md"]         # (default)
use_gitignore = true           # ~/.config/my-ssg/config.toml
```

### `config list`

List all available configuration options:

```bash
$ my-ssg config list

[my-ssg] Core configuration
  output_dir     Path     "output"    Build output directory
  url            URL?     null        Site base URL
  title          string?  null        Site title
  hooks          string[] ["default"] Plugins to load

[my-ssg.glob] File discovery settings
  patterns       string[] ["**/*.md"] Glob patterns to find content files
  use_gitignore  bool     true        Respect .gitignore when finding files
  exclude        string[] []          Patterns to exclude from results

[my-ssg.markdown] Markdown rendering configuration
  backend        string   "auto"      Markdown parser backend
  extensions     string[] [...]       Markdown extensions to enable
  ...
```

### `config get`

Get a specific value:

```bash
$ my-ssg config get feeds.defaults.items_per_page
10

$ my-ssg config get glob.patterns
["**/*.md"]

$ my-ssg config get glob.patterns --json

Behavior:

- Reads values directly from the config file to preserve source-of-truth behavior.
- Supports TOML, YAML, and JSON.
- Uses tree-sitter parsing for TOML/YAML in CGO-enabled builds to locate byte ranges without reformatting.
- CGO-disabled builds parse TOML/YAML via full decode.
- JSON output may be re-emitted for structured values.
["**/*.md"]
```

### `config set`

Set a value (writes to local config file):

```bash
$ my-ssg config set output_dir public
$ my-ssg config set feeds.defaults.items_per_page 20
$ my-ssg config set glob.patterns '["posts/*.md", "pages/*.md"]'

Behavior:

- TOML/YAML are updated with byte-range edits to preserve formatting and comments in CGO-enabled builds.
- CGO-disabled builds re-emit TOML/YAML with standard encoders.
- JSON is re-emitted with stable indentation.
- File permissions are preserved.
```

### `config init`

Generate a starter config file:

```bash
$ my-ssg config init
Created my-ssg.toml with default configuration

$ my-ssg config init --format yaml
Created my-ssg.yaml with default configuration

$ my-ssg config init --full
Created my-ssg.toml with all options documented
```

### `config validate`

Validate configuration:

- `config validate` MUST honor the same config resolution path as build commands, including `--config` and `--merge-config`

```bash
$ my-ssg config validate
✓ Configuration is valid

$ my-ssg config validate
✗ Configuration errors:
  - feeds.defaults.items_per_page: must be >= 0, got -5
  - glob.patterns: must be non-empty array
  - unknown field: my-ssg.typo_field
```

### `config docs`

Generate configuration documentation:

```bash
$ my-ssg config docs
# my-ssg Configuration

## Core Settings

### output_dir
- Type: Path
- Default: "output"
- Environment: MY_SSG_OUTPUT_DIR

Build output directory. All generated files will be written here.

### url
- Type: URL (optional)
- Default: null
- Environment: MY_SSG_URL

Site base URL. Required for generating absolute URLs in feeds and sitemaps.
...
```

---

## Complete Configuration Reference

### Core (`[my-ssg]`)

```toml
[my-ssg]
# Build output directory
output_dir = "output"

# Site metadata
url = "https://example.com"      # Base URL for absolute links
title = "My Site"                 # Site title
description = "A great site"      # Site description
author = "Jane Doe"               # Default author
language = "en"                   # Site language for feeds and metadata
author_url = "https://example.com/about/"
managing_editor = "editor@example.com (Jane Doe)"
webmaster = "webmaster@example.com (Jane Doe)"
copyright = "Copyright 2026 Jane Doe"

# Plugin loading
hooks = ["default"]               # Plugins to load
disabled_hooks = []               # Plugins to exclude

# Build settings
concurrency = 0                   # Worker threads (0 = auto)
```

### Glob (`[my-ssg.glob]`)

```toml
[my-ssg.glob]
patterns = ["pages/**/*.md", "posts/**/*.md"] # File patterns to match
use_gitignore = true              # Respect .gitignore
exclude = ["node_modules/**"]     # Patterns to exclude
```

### Markdown (`[my-ssg.markdown]`)

```toml
[my-ssg.markdown]
backend = "auto"                  # Parser backend
extensions = ["tables", "admonitions", "footnotes"]

[my-ssg.markdown.highlight]
enabled = true
theme = "github-dark"
line_numbers = false
```

### Feeds (`[my-ssg.feeds]`)

```toml
[my-ssg.feeds.defaults]
items_per_page = 10
orphan_threshold = 3

[my-ssg.feeds.defaults.formats]
html = true
simple_html = true
rss = true
atom = true
json = true
sitemap = true

[my-ssg.feeds.syndication]
max_items = 20
include_content = false
site_archive_disabled = false
feed_archives_disabled = false

[my-ssg.feeds_page]
enabled = true
title = "Feeds"
description = "Browse the public feeds available on this site."
template = "feeds.html"
slug_prefix = "feeds"

[[my-ssg.feeds]]
slug = "archive"
title = "Archive"
description = "All posts"
filter = "published == True"
sort = "date"
reverse = true
limit = 0
offset = 0

[[my-ssg.feeds]]
slug = "blog"
archive_disabled = true
```

### Serve (`[my-ssg.serve]`)

```toml
[my-ssg.serve]
port = 3000
host = "localhost"
livereload = true
open_browser = false
debounce_ms = 100
```

Serve mode starts the HTTP server immediately while the initial build runs in the background.
During builds, a status banner is injected into HTML responses, and a minimal 404 page is
served until the generated 404.html is available.

### Assets (`[my-ssg.assets]`)

```toml
[my-ssg.assets]
dir = "static"
output_subdir = ""

[my-ssg.assets.fingerprint]
enabled = false
algorithm = "sha256"
length = 8
exclude = ["robots.txt", "favicon.ico"]
```

markata-go also supports self-hosting third-party CDN assets (HTMX, GLightbox, Mermaid, Chart.js, Cal-Heatmap, D3, Lite YouTube, Reveal.js) through the assets config. When enabled, assets are downloaded into a cache directory and copied into the output under the vendor directory. Templates can use the `asset_urls` mapping to reference the local paths.

```toml
[markata-go.assets]
mode = "self-hosted"           # default: "self-hosted"
cache_dir = ".markata/assets-cache"
output_dir = "assets/vendor"
verify_integrity = true
```

### CSS Purge (`[my-ssg.css_purge]`)

```toml
[my-ssg.css_purge]
enabled = false
verbose = false
preserve = ["js-*", "htmx-*", "theme-*", "palette-*"]
preserve_attributes = ["data-theme", "data-palette"]
skip_files = ["vendor/*", "normalize.css"]
warning_threshold = 0
```

CSS purge removes unused rules by scanning generated HTML and keeping only selectors
that are actually present. The purge logic always preserves key @-rules and keeps
pseudo-only selectors like `:root` or `::selection` to avoid dropping base/theme styles.

### Tailwind (`[my-ssg.tailwind]`)

```toml
[my-ssg.tailwind]
include = "css"                # "css", "js", or false (default: "css")
preflight = false               # Enable Tailwind Preflight reset styles (default: false)
input = "tailwind.css"          # Input CSS (relative to assets_dir)
output = "markata-tailwind.css" # Output CSS (relative to assets_dir)
config_file = ""                # Optional tailwind.config.js path
build = true                     # Run Tailwind CLI during build
minify = true                    # Pass --minify to Tailwind CLI
auto_install = true              # Auto-download Tailwind CLI (default: true)
version = "v3.4.19"             # Managed Tailwind CLI version tag
cache_dir = ""                  # Cache dir for Tailwind CLI
binary = ""                     # Optional path to tailwindcss binary
extra_args = []                  # Optional extra CLI arguments
verbose = false                  # Verbose installer/build logs
```

Tailwind automation runs a markata-managed standalone Tailwind CLI and writes the
compiled CSS into your assets directory so it is copied and fingerprinted like any
other static file. The zero-setup default is: enable Tailwind in config and build.
If `tailwind.css` is missing, markata-go generates a default input containing
`@tailwind base`, `@tailwind components`, and `@tailwind utilities`.

Behavior:

- `build = true` injects the Tailwind output during configure and performs the
  actual Tailwind rebuild in cleanup when needed.
- `preflight = false` is the default for markata-go-managed Tailwind configs so
  Tailwind utilities work without resetting the built-in theme's typography and
  spacing. Set `preflight = true` for Tailwind-first sites that want the reset.
- `input`/`output` resolve relative to `assets_dir` (absolute paths are respected).
- If `extra_args` is empty and `config_file` is unset, markata-go generates a
  temporary Tailwind config that scans a generated token manifest derived from
  rendered page HTML plus local JS/template sources. The generated config also
  sets `corePlugins.preflight` from `tailwind.preflight`. The manifest is hashed
  and cached so Tailwind is skipped when the effective utility set is unchanged.
- `include = "css"` ensures the output CSS is included in templates. If
  `theme.custom_css` is unset, it is set to the output path. The plugin does not
  override explicit `theme.custom_css` values.
- `include = "js"` injects `<script src="https://cdn.tailwindcss.com"></script>`
  into the document head. If `[my-ssg.assets].mode` is self-hosted, the JS is
  pulled from the vendor asset registry and served locally.
- `include = false` disables automatic inclusion; build can still run.
- If `include = "css"` and CSS purge is disabled, a validation warning is emitted.
- `auto_install = true` downloads and uses the managed Tailwind CLI (versioned,
  checksum verified) into the cache directory when needed. This is preferred over
  `PATH` for consistent builds. If disabled, `binary` or `PATH` is used.
- Fast mode skips Tailwind rebuilds when the compiled CSS asset already exists,
  keeping development builds fast without requiring a separate output directory.

### Theme (`[my-ssg.theme]`)

```toml
[my-ssg.theme]
name = "default"              # Theme name (built-in or installed)
custom_css = ""               # Path to custom CSS file (loaded after theme CSS)
fallback_mode = "dark"        # Fallback when system preference is unavailable: "dark" or "light"

# Theme-specific options (defined by theme)
[my-ssg.theme.options]
primary_color = "#3b82f6"     # Varies by theme
font_family = "system-ui"
show_toc = true

# CSS custom property overrides
[my-ssg.theme.variables]
"--color-primary" = "#8b5cf6"
"--color-primary-dark" = "#7c3aed"
"--font-body" = "Inter, system-ui"
"--content-width" = "70ch"
```

See [THEMES.md](./THEMES.md) for complete theming documentation.

### Post Formats (`[my-ssg.post_formats]`)

```toml
[my-ssg.post_formats]
html = true       # /slug/index.html (default: true)
markdown = true   # /slug.md - raw source with frontmatter (default: true)
text = true       # /slug.txt - plain terminal-friendly content (default: true)
ansi = true       # /slug.ansi - ANSI-styled terminal output (default: false)
og = true         # /slug/og/index.html - social card for screenshots
```

This section controls what output formats are generated for each post:

| Format | Default | Output Path | Description |
|--------|---------|-------------|-------------|
| `html` | `true` | `/slug/index.html` | Standard rendered HTML page |
| `markdown` | `true` | `/slug.md` | Raw markdown with reconstructed frontmatter |
| `text` | `true` | `/slug.txt` | Plain terminal-friendly content with no ANSI escapes |
| `ansi` | `false` | `/slug.ansi` | ANSI-styled terminal page output |
| `og` | `true` | `/slug/og/index.html` | OpenGraph card HTML (1200x630) for social screenshots |

Posts MAY override these site defaults in frontmatter with a `post_formats` mapping. Per-post overrides merge with `[my-ssg.post_formats]` key-by-key; omitted keys inherit the site setting.

```yaml
---
title: "Terminal-first post"
post_formats:
  ansi: true
  og: false
---
```

In this example, the post inherits the site defaults for `html`, `markdown`, and `text`, enables `.ansi` for this post only, and suppresses OG output for this post only.

`text` and `ansi` are separate explicit variants:

- `text` MUST emit readable plain text with no ANSI escape sequences.
- `ansi` MUST emit the same terminal-oriented structure with ANSI styling for capable clients.
- ANSI output is opt-in via the `.ansi` path; markata-go MUST NOT inject ANSI escape sequences into `.txt` output.

Terminal rendering for both `text` and `ansi` variants MUST:

- derive structure from rendered page content when rendered HTML is available
- preserve headings, emphasis, links, blockquotes, lists, horizontal rules, tables, admonitions, and code fences in terminal-safe form
- preserve image and video references as readable labeled URLs when media appears inline or in frontmatter
- degrade cleanly to plain text when ANSI styling is disabled
- keep canonical `.txt` endpoints readable in clients that do not support ANSI

Theme-aware ANSI rendering MUST derive colors from the active site palette when possible. Palette resolution for ANSI output follows:

1. `theme.palette`
2. `theme.palette_dark`
3. `theme.palette_light`
4. built-in dark fallback palette

**Directory-based Redirects for txt/md/ansi:**

For `.txt`, `.md`, and `.ansi` formats, content is placed at the canonical short URL (`/slug.txt`, `/slug.md`, `/slug.ansi`). Redirects are provided at `/slug.<ext>/index.html` (for hosts that serve `index.html` in a directory) and `/slug/index.<ext>/index.html` (for backwards compatibility).

Rendered feed/sidebar variant links MUST use the same canonical short URLs rather than nested `/slug/index.<ext>` paths.
If a post format is disabled in the resolved config, the corresponding sidebar link MUST be omitted.

**Special Files (robots, llms, humans, security, ads):**

Special web files have an inverted structure to serve content at their expected root-level locations:
- Content at `/slug.txt` (e.g., `/robots.txt`)
- HTML redirect at `/slug/index.txt/index.html` pointing to `/slug.txt`

This enables standard web txt files to be served at their expected locations:
- `/robots.txt` - Robot exclusion standard
- `/llms.txt` - AI/LLM guidance file  
- `/humans.txt` - Human-readable site info
- `/security.txt` - Security contact information
- `/ads.txt` - Authorized digital sellers

**Use cases:**
- **markdown**: API consumers, "view source" links, copy-paste code
- **text**: Standard web txt files, plain text readers, CLI tools
- **ansi**: `curl`, pagers, and intentional terminal reading experiences
- **og**: Automated social image generation with puppeteer/playwright

**Example:**
```toml
[markata-go.post_formats]
html = true
markdown = true  # Enable raw markdown output at /slug.md
text = true      # Enable plain terminal-friendly output at /slug.txt
ansi = true      # Enable ANSI terminal output at /slug.ansi
og = true        # Enable social card HTML for screenshot tools
```

---

### Glob Settings (`[my-ssg.glob]`)

```toml
[my-ssg.glob]
patterns = ["posts/**/*.md", "pages/**/*.md"]
use_gitignore = true
slug_mode = "flat"    # "flat" (default) or "path"
```

The glob section controls both file discovery and the default slug derivation strategy.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `patterns` | string[] | `["pages/**/*.md", "posts/**/*.md"]` | Glob patterns to find content files |
| `use_gitignore` | bool | `true` | Respect `.gitignore` when discovering content |
| `slug_mode` | string | `"flat"` | How to derive slugs when frontmatter does not set `slug` |

`slug_mode` values:

- `flat` - Current markata behavior. Slugs come from the filename only. Examples: `posts/2026/hello.md -> hello`, `docs/index.md -> docs`.
- `path` - Derive slugs from the relative content path. Leading `posts/` and `pages/` path segments are removed. `index.md`, `README.md`, and `readme.md` become the root of their containing directory. Examples: `posts/notes/today.md -> notes/today`, `pages/docs/README.md -> docs`.

Optional path-specific overrides can be declared with `slug_rules`:

```toml
[my-ssg.glob]
slug_mode = "flat"

[[my-ssg.glob.slug_rules]]
prefix = "posts/blog"
mode = "flat"

[[my-ssg.glob.slug_rules]]
prefix = "posts/notes"
mode = "path"
```

Rules use the longest matching prefix, so more specific directories win.

Explicit frontmatter `slug` always wins over `slug_mode`.

---

### Well-Known Files (`[my-ssg.well_known]`)

```toml
[my-ssg.well_known]
enabled = true
auto_generate = ["host-meta", "host-meta.json", "webfinger", "nodeinfo", "time", "links"]

# Optional entries requiring config
ssh_fingerprint = "SHA256:abcdef..."
keybase_username = "username"
```

This section controls auto-generated `.well-known` endpoints derived from site metadata:

| Entry | Output Path | Description |
|-------|-------------|-------------|
| `host-meta` | `/.well-known/host-meta` | XRD host metadata for discovery |
| `host-meta.json` | `/.well-known/host-meta.json` | JSON host metadata (JRD) |
| `webfinger` | `/.well-known/webfinger` | WebFinger endpoint response |
| `nodeinfo` | `/.well-known/nodeinfo` + `/nodeinfo/2.0` | NodeInfo discovery + instance metadata |
| `time` | `/.well-known/time` | Build timestamp (RFC3339) |
| `links` | `/.well-known/links` + `/.well-known/internal-links` + `/external-links/` + `/internal-links/` | Outbound links grouped by target domain and internal links grouped by target URL, each with HTML view |
| `sshfp` | `/.well-known/sshfp` | SSH fingerprint text (requires `ssh_fingerprint`) |
| `keybase` | `/.well-known/keybase.txt` | Keybase verification (requires `keybase_username`) |

**Defaults:**
- `enabled` defaults to `true`
- `auto_generate` defaults to `host-meta`, `host-meta.json`, `webfinger`, `nodeinfo`, `time`, and `links`

**Notes:**
- If `auto_generate` is empty, only optional entries with explicit config are generated.
- `nodeinfo` generates both the discovery document and a minimal `/nodeinfo/2.0` payload.
- `time` is rebuilt on each build and is always UTC RFC3339.
- `links` follows the Jim Nielsen-style grouped JSON format for external links, and additionally writes internal links plus `/external-links/` and `/internal-links/`.

---

### WebSub (`[my-ssg.websub]`)

```toml
[my-ssg.websub]
enabled = true
hubs = ["https://hub.example.com/"]
```

When enabled, markata-go emits WebSub discovery links:

- HTML pages include `<link rel="hub" href="...">` for each hub
- RSS/Atom feeds include `rel="hub"` and `rel="self"` links

**Defaults:**
- `enabled` defaults to `false`
- `hubs` defaults to an empty list

---

### Templates (`[my-ssg.templates]`)

```toml
[my-ssg.templates.media]
trusted_domains = [
  "dropper.wayl.one",
  "dropper.waylonwalker.com",
  "dropper-dev.wayl.one",
]
```

- `media.trusted_domains` controls which hosts the built-in template helpers will decorate with `w`/`h` sizing parameters and derived posters. Relative URLs are always treated as trusted.
- The default values match the dropper CDN. Override this list when you serve media through a different host so that video posters and cached previews stay consistent.

## See Also

- [SPEC.md](./SPEC.md) - Core specification
- [THEMES.md](./THEMES.md) - Theming system
- [PLUGINS.md](./PLUGINS.md) - Plugin development
