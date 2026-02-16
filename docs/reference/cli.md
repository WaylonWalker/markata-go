---
title: "CLI Reference"
description: "Complete reference for all markata-go commands, flags, and options"
date: 2024-01-15
published: true
slug: /docs/reference/cli/
tags:
  - documentation
  - reference
  - cli
---

# CLI Reference

markata-go provides a command-line interface for building, serving, and managing your static site.

## Overview

The markata-go CLI follows a subcommand pattern similar to tools like `git` and `docker`. Each command has its own set of flags, with some global flags available across all commands.

```bash
markata-go [global flags] <command> [command flags] [arguments]
```

## Global Flags

These flags are available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to configuration file | Auto-discovered |
| `--merge-config` | `-m` | Additional config file(s) to merge (can be used multiple times) | None |
| `--output` | `-o` | Output directory (overrides config) | `public` |
| `--verbose` | `-v` | Enable verbose output | `false` |

### Config File Discovery

When `--config` is not specified, markata-go searches for configuration files in the following order:

1. `markata-go.toml`
2. `markata-go.yaml` / `markata-go.yml`
3. `markata-go.json`
4. `.markata-go.toml`
5. `.markata-go.yaml` / `.markata-go.yml`
6. `.markata-go.json`

See [[configuration-guide|Configuration]] for details on configuration options.

---

## Commands

### build

Build the static site by processing all content files through the plugin lifecycle.

#### Usage

```bash
markata-go build [flags]
```

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--clean` | | Remove output directory before building | `false` |
| `--dry-run` | | Show what would be built without writing files | `false` |
| `--verbose` | `-v` | Enable verbose logging | `false` |
| `--output` | `-o` | Override output directory | from config |

#### Examples

```bash
# Standard build
markata-go build

# Clean build (removes output directory first)
markata-go build --clean

# Preview what would be built
markata-go build --dry-run

# Build with verbose output
markata-go build -v

# Build to a custom output directory
markata-go build -o dist

# Build with a specific config file
markata-go build -c production.toml

# Combine flags
markata-go build --clean -v -o dist
```

#### Exit Codes

| Code | Description |
|------|-------------|
| `0` | Build completed successfully |
| `1` | Build failed (configuration error, plugin error, etc.) |
| `2` | No content files found |

#### Build Process

The build command executes the full 9-stage lifecycle:

1. **Configure** - Load and merge configuration
2. **Validate** - Validate configuration settings
3. **Glob** - Discover content files
4. **Load** - Parse markdown files and frontmatter
5. **Transform** - Pre-render processing (wikilinks, jinja-md, etc.)
6. **Render** - Convert markdown to HTML
7. **Collect** - Build feeds and collections
8. **Write** - Output files to disk
9. **Cleanup** - Release resources

When `--verbose` is enabled, build output includes per-stage timing to highlight slow stages.

---

### serve

Start a development server with live reload support.

#### Usage

```bash
markata-go serve [flags]
```

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--port` | `-p` | Port to listen on | `8000` |
| `--host` | | Host address to bind to | `localhost` |
| `--watch` | | Enable file watching and auto-rebuild | `true` |
| `--no-watch` | | Disable file watching (legacy, overrides --watch) | `false` |
| `--verbose` | `-v` | Enable verbose logging | `false` |

#### Examples

```bash
# Serve on default port (localhost:8000)
markata-go serve

# Serve on a custom port
markata-go serve -p 3000
markata-go serve --port 3000

# Bind to all network interfaces (accessible from other devices)
markata-go serve --host 0.0.0.0

# Explicitly enable file watching (default behavior)
markata-go serve --watch

# Serve without file watching
markata-go serve --watch=false
markata-go serve --no-watch

# Serve with verbose logging
markata-go serve -v

# Combine options
markata-go serve -p 8080 --host 0.0.0.0 -v
```

#### Features

- **Automatic rebuild**: File changes trigger a rebuild automatically
- **Live reload**: Connected browsers refresh when content changes
- **Static file serving**: Serves the output directory
- **MIME type detection**: Correct content types for all file types
- **Immediate serve**: Server starts before the initial build completes
- **Build status banner**: Shows build progress and errors during serve
- **Early 404**: Minimal 404 is served until the generated 404.html exists

#### Development Workflow

```bash
# Terminal 1: Start the server
markata-go serve -v

# The server will:
# - Start serving at http://localhost:8000 immediately
# - Build your site in the background
# - Watch for file changes
# - Rebuild and reload on changes
```

---

### tui

Interactive terminal UI for browsing and managing your markata site. Inspired by [k9s](https://k9scli.io/).

#### Usage

```bash
markata-go tui [flags]
```

#### Description

The TUI provides a keyboard-driven interface for exploring your site's posts, tags, and feeds. It features a k9s-style table layout with filtering, sorting, and quick actions.

#### Views

The TUI has three main views, accessible via number keys:

| Key | View | Description |
|-----|------|-------------|
| `1` | Posts | Browse all posts in a table |
| `2` | Tags | View and filter by tags |
| `3` | Feeds | View configured feeds |

#### Global Keybindings

These keys work in all views:

| Key | Action |
|-----|--------|
| `1`, `2`, `3` | Switch between views |
| `/` | Enter filter mode |
| `Esc` | Clear filter / exit current mode |
| `?` | Show help |
| `q`, `Ctrl+C` | Quit |

#### Posts View

The posts view displays all posts in a table with columns for title, date, word count, tags, and path.

##### Posts Keybindings

| Key | Action |
|-----|--------|
| `↑`/`↓`, `j`/`k` | Navigate up/down |
| `Enter` | View post details |
| `e` | Edit post in $EDITOR |
| `s` | Open sort menu |
| `/` | Filter posts |

##### Post Detail View

Press `Enter` on any post to see its full details including:
- Title, path, and date
- Published status and tags
- Word count and description
- Content preview

Press `Esc` to return to the post list, or `e` to edit the post.

#### Sorting

Press `s` in the posts view to open the sort menu:

| Sort Option | Description |
|-------------|-------------|
| Date | Sort by post date (default) |
| Title | Sort alphabetically by title |
| Word Count | Sort by word count |
| Path | Sort by file path |

Use `a` for ascending order, `d` for descending order.

#### Filtering

Press `/` to enter filter mode. Type a filter expression and press `Enter` to apply.

##### Filter Expression Examples

```
published==true           # Only published posts
tags contains "go"        # Posts tagged with "go"  
date > "2024-01-01"       # Posts after a specific date
title startswith "How"    # Titles starting with "How"
```

See [Filter Expressions](/docs/guides/filters) for the complete filter syntax.

#### Feeds View

The feeds view displays all configured feeds with:
- Feed name
- Number of posts in the feed
- Filter expression used
- Output path

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `EDITOR` | Primary editor for `e` key (falls back to `vim`) |
| `VISUAL` | Secondary editor if `EDITOR` is not set |

#### Examples

```bash
# Launch the TUI
markata-go tui

# The TUI will display your posts in a table format
# Use j/k to navigate, Enter to view details, e to edit
```

---

### list

List posts, tags, or feeds in table, JSON, CSV, or path-only formats.

#### Usage

```bash
markata-go list posts [flags]
markata-go list tags [flags]
markata-go list feeds [flags]
markata-go list feeds posts <feed>
```

#### Shared Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--format` | Output format: `table`, `json`, `csv`, `path` | `table` |
| `--sort` | Sort field (varies by subcommand) | posts: `date`, tags: `count`, feeds: `name` |
| `--order` | Sort order: `asc` or `desc` | posts/tags: `desc`, feeds: `asc` |

#### Posts Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--filter` | Filter expression for posts | none |
| `--feed` | Limit posts to a feed by name | none |

#### Examples

```bash
# List posts as a table
markata-go list posts

# Path-only output for piping
markata-go list posts --format path

# List tags as JSON
markata-go list tags --format json

# List feeds as CSV
markata-go list feeds --format csv

# List posts for a feed
markata-go list posts --feed blog

# List posts for a feed
markata-go list feeds posts blog

# List post paths for a feed
markata-go list feeds posts blog --format path
```

#### Cache

`list` and `tui` use a persistent cache at `.markata/cache/list.json`. Delete this file to force a full refresh.

---

### init

Initialize a new markata-go project with interactive setup.

#### Usage

```bash
markata-go init [flags]
```

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Overwrite existing files | `false` |
| `--plain` | Use plain text prompts instead of the TUI wizard | `false` (auto-detected) |

#### Examples

```bash
# Start interactive project setup (uses TUI wizard when available)
markata-go init

# Overwrite existing configuration
markata-go init --force

# Use plain text prompts (for scripts or non-TTY environments)
markata-go init --plain
```

#### Interactive Flow

The `init` command provides a rich TUI (Text User Interface) wizard powered by [charmbracelet/huh](https://github.com/charmbracelet/huh) for an enhanced user experience. The wizard automatically falls back to plain text prompts when:

- The `--plain` flag is specified
- stdin is not a TTY (e.g., when piping input or running in CI)

**TUI Mode Features:**
- Searchable palette selection
- Multi-select for features
- Form validation
- Styled using your site's configured palette (for existing projects)

**Plain mode example:**

```
$ markata-go init --plain

Welcome to markata-go!

Site title [My Site]: My Awesome Blog
Description [A site built with markata-go]: A blog about things
Author []: Your Name
URL [https://example.com]: https://myblog.com

Creating project structure...
  ✓ Created posts/
  ✓ Created static/
  ✓ Created markata-go.toml

Create your first post? (Y/n): y
Post title [Hello World]: My First Post

  ✓ Created posts/my-first-post.md

Done! Run 'markata-go serve' to start.
```

#### What Gets Created

The command creates:

1. **markata-go.toml** - Configuration file with your site settings
2. **posts/** - Directory for your blog posts
3. **static/** - Directory for static assets (images, CSS, etc.)
4. **(Optional) First post** - A starter markdown file

---

### new

Create a new content file with frontmatter template.

#### Usage

```bash
markata-go new [title] [flags]
```

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `title` | The title of the new content | No (prompted if not provided) |

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--template` | `-t` | Content template to use | `post` |
| `--list` | `-l` | List available templates | `false` |
| `--dir` | | Directory to create the content in (overrides template) | Template's default |
| `--draft` | | Create as a draft | `false` |
| `--tags` | | Comma-separated list of tags | `""` |
| `--plain` | | Use plain text prompts instead of TUI wizard | `false` |

#### Built-in Templates

| Template | Directory | Description |
|----------|-----------|-------------|
| `post` | `posts/` | Blog posts with standard frontmatter |
| `page` | `pages/` | Static pages |
| `docs` | `docs/` | Documentation pages |

#### Examples

```bash
# Create a new post (creates posts/my-first-post.md)
markata-go new "My First Post"

# Use a different template
markata-go new "About" --template page
# Creates: pages/about.md

markata-go new "Getting Started" -t docs
# Creates: docs/getting-started.md

# List available templates
markata-go new --list

# Override the template's directory
markata-go new "Hello World" --template post --dir blog
# Creates: blog/hello-world.md

# Create as a draft (opt-in, default is published)
markata-go new "Work in Progress" --draft
# Creates: posts/work-in-progress.md with draft: true

# Create as published (default behavior)
markata-go new "Ready to Publish"
# Creates: posts/ready-to-publish.md with draft: false, published: true

# Create with tags
markata-go new "Go Tutorial" --tags "go,tutorial,programming"
# Creates: posts/go-tutorial.md with tags: ["go", "tutorial", "programming"]

# Interactive mode (no arguments) - launches TUI wizard
markata-go new
# TUI wizard prompts for: template, title, directory, tags, privacy, authors

# Interactive mode with plain text prompts (no TUI)
markata-go new --plain
```

#### Template System

Templates control the default frontmatter and output directory for new content.

**Template Discovery:**

1. **Built-in templates** - `post`, `page`, `docs` are always available
2. **Config templates** - Defined in `markata-go.toml` under `[content_templates]`
3. **File templates** - Markdown files in `content-templates/` directory

**Configuration Example:**

```toml
[content_templates]
directory = "content-templates"

[content_templates.placement]
post = "blog"       # Override: posts go to blog/
page = "pages"
docs = "documentation"

[[content_templates.templates]]
name = "tutorial"
directory = "tutorials"
body = "## Prerequisites\n\n## Steps\n\n## Summary"

[content_templates.templates.frontmatter]
templateKey = "tutorial"
series = ""
```

**File Template Example:**

Create `content-templates/recipe.md`:

```markdown
---
templateKey: recipe
_directory: recipes
prep_time: ""
cook_time: ""
servings: 4
---

## Ingredients

-

## Instructions

1.
```

The `_directory` field in frontmatter sets the output directory (removed from generated content).

#### Interactive Mode

When called without a title argument (or no arguments at all), the command launches an
interactive TUI wizard powered by [charmbracelet/huh](https://github.com/charmbracelet/huh).

The wizard guides you through:

1. **Template selection** - Pick from built-in, config, and file-based templates
2. **Title** - Enter the post title
3. **Directory** - Choose from existing directories or enter a custom one, with config-driven defaults per template type
4. **Tags** - Select from existing site tags and/or type custom tags
5. **Privacy** - Mark the post as private
6. **Authors** - If multiple authors are configured, choose whether to use the default or select specific authors
7. **Summary** - Review all choices before creating the file

```
$ markata-go new

? Select a template
  > post - Blog post (posts/)
    page - Static page (pages/)
    docs - Documentation (docs/)

? Title: My New Post
? Select a directory: posts (default)
? Select tags: [go, tutorial]
? Make this post private? No
? Create my-new-post.md in posts/? Yes

Created: posts/my-new-post.md
```

**Plain text mode:** Use `--plain` or pipe input to get simple text prompts instead of the TUI.
Falls back automatically when stdin is not a TTY.

#### Generated File

The command creates a markdown file with template-specific frontmatter:

```markdown
---
title: My First Post
slug: my-first-post
date: "2024-01-15"
published: true
draft: false
templateKey: post
tags:
  - go
  - tutorial
description: ""
---

# My First Post

Write your content here...
```

The slug is automatically generated from the title by:
- Converting to lowercase
- Replacing spaces with hyphens
- Removing special characters

---

### lint

Lint markdown files for common issues that can cause build failures.

#### Usage

```bash
markata-go lint [files...] [flags]
```

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `files` | Glob patterns or file paths to lint | Yes |

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--fix` | Automatically fix detected issues | `false` |

#### Detected Issues

The linter checks for these common problems:

| Issue | Severity | Auto-fixable | Description |
|-------|----------|--------------|-------------|
| `duplicate-key` | Error | Yes | Duplicate YAML keys in frontmatter |
| `invalid-date` | Warning | Yes | Non-ISO 8601 date formats |
| `missing-alt-text` | Warning | Yes | Image links without alt text `![]()` |
| `protocol-less-url` | Warning | Yes | URLs starting with `//` instead of `https://` |

#### Examples

```bash
# Lint all markdown files in posts directory
markata-go lint posts/*.md

# Lint with glob pattern (recursive)
markata-go lint 'posts/**/*.md'

# Lint multiple patterns
markata-go lint posts/*.md pages/*.md docs/**/*.md

# Lint and auto-fix issues
markata-go lint posts/*.md --fix

# Lint a specific file
markata-go lint posts/hello-world.md

# Lint with verbose output
markata-go lint posts/*.md -v
```

#### Output Format

```
posts/example.md:
  error [line 5]: duplicate key 'title' (first occurrence at line 2)
  warning [line 3]: invalid date format for 'date': 2020-1-15 (single-digit month/day)
  warning [line 12, col 1]: image link missing alt text

✗ 5 file(s) linted, 3 issue(s) in 1 file(s)
```

#### Auto-fix Behavior

When `--fix` is enabled:

| Issue | Fix Applied |
|-------|-------------|
| `duplicate-key` | Keeps last occurrence, removes earlier duplicates |
| `invalid-date` | Pads single-digit months/days (e.g., `2020-1-5` → `2020-01-05`) |
| `missing-alt-text` | Adds placeholder alt text: `![]()` → `![image]()` |
| `protocol-less-url` | Adds HTTPS protocol: `//example.com` → `https://example.com` |

#### Exit Codes

| Code | Description |
|------|-------------|
| `0` | No errors found (warnings are allowed) |
| `1` | Errors found (or file read errors) |

#### Use Cases

**Migration from Python markata:**
```bash
# Find and fix common issues when migrating content
markata-go lint content/**/*.md --fix
```

**CI/CD integration:**
```bash
# Fail CI if markdown has errors
markata-go lint posts/**/*.md || exit 1
```

**Pre-commit hook:**
```bash
#!/bin/bash
markata-go lint $(git diff --cached --name-only -- '*.md')
```

---

### config

Configuration management commands for viewing, validating, and initializing configuration.

#### Usage

```bash
markata-go config <subcommand> [flags]
```

#### Subcommands

##### show

Display the resolved configuration after merging defaults, config file, and environment variables.

```bash
markata-go config show [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output as JSON | `false` |
| `--toml` | Output as TOML | `false` |
| (none) | Output as YAML | default |

**Examples:**

```bash
# Show config as YAML (default)
markata-go config show

# Show config as JSON
markata-go config show --json

# Show config as TOML
markata-go config show --toml

# Show config from specific file
markata-go config show -c production.toml
```

##### get

Get a specific configuration value using dot notation for nested keys.

```bash
markata-go config get <key>
```

**Examples:**

```bash
# Get top-level value
markata-go config get output_dir
# Output: public

# Get nested value
markata-go config get glob.patterns
# Output: ["posts/**/*.md", "pages/*.md"]

# Get deeply nested value
markata-go config get feeds.defaults.items_per_page
# Output: 10

# Get from specific config file
markata-go config get url -c production.toml
# Output: https://example.com
```

##### validate

Validate the configuration file and report any errors or warnings.

```bash
markata-go config validate [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Config file to validate |

**Examples:**

```bash
# Validate default config
markata-go config validate

# Validate specific config file
markata-go config validate -c production.toml
markata-go config validate -c custom.yaml
```

**Output:**

```
Validating configuration...
OK: Configuration is valid

# Or with errors:
ERROR: Invalid configuration
  - output_dir: directory does not exist
  - glob.patterns: at least one pattern required
  - feeds[0].filter: invalid filter expression
```

##### init

Create a new configuration file with sensible defaults.

```bash
markata-go config init [filename] [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Overwrite existing config file | `false` |

**Examples:**

```bash
# Create markata-go.toml with defaults
markata-go config init

# Create YAML config
markata-go config init markata-go.yaml

# Create with specific name
markata-go config init site.toml

# Overwrite existing config
markata-go config init --force
```

---

### aesthetic

Manage and inspect aesthetic presets for your site's visual styling.

#### Usage

```bash
markata-go aesthetic <subcommand>
```

#### Subcommands

##### list

List all available aesthetic presets.

```bash
markata-go aesthetic list
```

**Example Output:**

```
Available aesthetics:

  balanced    Default. Comfortable rounding, subtle shadows, normal spacing
  brutal      Sharp corners, thick borders, tight spacing, no shadows
  elevated    Generous rounding, layered shadows, generous spacing
  minimal     No rounding, maximum whitespace, no shadows, hairline borders
  precision   Subtle corners, compact spacing, hairline borders, minimal shadows
```

##### show

Display the details of a specific aesthetic preset.

```bash
markata-go aesthetic show <name>
```

**Arguments:**

| Argument | Description | Required |
|----------|-------------|----------|
| `name` | Name of the aesthetic to show | Yes |

**Examples:**

```bash
# Show details of the elevated aesthetic
markata-go aesthetic show elevated

# Show details of the brutal aesthetic
markata-go aesthetic show brutal
```

**Example Output:**

```
Aesthetic: elevated

Description: Generous rounding, layered shadows, generous spacing

Values:
  border_radius:    16px
  spacing_scale:    1.5
  border_width:     0px
  border_style:     none
  shadow_intensity: 0.6
  shadow_size:      lg
```

#### Available Aesthetics

| Aesthetic | Description |
|-----------|-------------|
| `balanced` | Default. Comfortable rounding, subtle shadows, normal spacing |
| `brutal` | Sharp corners, thick borders, tight spacing, no shadows |
| `elevated` | Generous rounding, layered shadows, generous spacing |
| `minimal` | No rounding, maximum whitespace, no shadows, hairline borders |
| `precision` | Subtle corners, compact spacing, hairline borders, minimal shadows |

See [[themes-and-styling|Themes Guide]] for detailed aesthetic customization.

---

### palette

Manage, browse, and select color palettes for your site's theme.

#### Usage

```bash
markata-go palette <subcommand> [flags]
```

#### Subcommands

##### list

List all available palettes from built-in, user, and project sources.

```bash
markata-go palette list
markata-go palette list --variant dark
markata-go palette list --json
```

##### info

Show detailed information about a specific palette including colors, semantic mappings, and component colors.

```bash
markata-go palette info catppuccin-mocha
markata-go palette info catppuccin-mocha --json
```

##### pick

Open an interactive full-screen TUI to browse and select a palette with live color previews. Sets the chosen palette in your config by default.

```bash
# Pick and set in your config (default)
markata-go palette pick

# Only print the name without updating config
markata-go palette pick --no-set
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--no-set` | Only print the palette name without updating the config file |

The picker shows a two-panel layout with a fuzzy-filterable palette list on the left and a live color swatch preview on the right. Type to filter, arrow keys to navigate, Enter to select, Esc to cancel.

**Composability:**

```bash
# View info for the palette you pick (without setting it)
markata-go palette info "$(markata-go palette pick --no-set)"
```

##### check

Check a palette for accessibility (WCAG contrast ratios).

```bash
markata-go palette check catppuccin-mocha
```

##### preview

Preview a palette's colors in the terminal.

```bash
markata-go palette preview catppuccin-mocha
```

##### new

Create a new custom palette interactively.

```bash
markata-go palette new my-brand
```

##### clone

Clone an existing palette as a starting point for customization.

```bash
markata-go palette clone catppuccin-mocha my-custom
```

##### export

Export a palette to different formats (CSS, SCSS, JSON).

```bash
markata-go palette export catppuccin-mocha --format css
```

##### fetch

Import a palette from [Lospec.com](https://lospec.com/palette-list).

```bash
markata-go palette fetch https://lospec.com/palette-list/sweetie-16.txt
```

See [[themes-and-styling|Themes Guide]] for detailed palette customization.

---

## Environment Variables

markata-go configuration can be overridden using environment variables with the `MARKATA_GO_` prefix.

### Variable Naming Convention

Environment variables follow this pattern:
- Prefix: `MARKATA_GO_`
- Key names: UPPERCASE with underscores
- Nested keys: Use single underscore separation

### Common Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MARKATA_GO_OUTPUT_DIR` | Output directory | `dist` |
| `MARKATA_GO_URL` | Site base URL | `https://staging.example.com` |
| `MARKATA_GO_TITLE` | Site title | `My Staging Site` |
| `MARKATA_GO_CONCURRENCY` | Worker count (0=auto) | `4` |
| `MARKATA_GO_TEMPLATES_DIR` | Templates directory | `themes/custom/templates` |
| `MARKATA_GO_ASSETS_DIR` | Static assets directory | `assets` |

### Examples

```bash
# Build to a different directory
MARKATA_GO_OUTPUT_DIR=dist markata-go build

# Build for staging environment
MARKATA_GO_URL=https://staging.example.com markata-go build

# Limit concurrency for debugging
MARKATA_GO_CONCURRENCY=1 markata-go build -v

# Multiple overrides
MARKATA_GO_OUTPUT_DIR=dist \
MARKATA_GO_URL=https://prod.example.com \
markata-go build --clean

# In CI/CD pipelines
export MARKATA_GO_URL="${DEPLOY_URL}"
export MARKATA_GO_OUTPUT_DIR="build"
markata-go build --clean
```

### Override Precedence

Configuration values are resolved in this order (later sources override earlier):

1. Built-in defaults
2. Configuration file
3. Environment variables
4. Command-line flags

---

## Exit Codes

All markata-go commands use standard exit codes:

| Code | Name | Description |
|------|------|-------------|
| `0` | Success | Command completed successfully |
| `1` | Error | General error (configuration, plugin, I/O, etc.) |
| `2` | No Content | No content files found matching glob patterns |
| `3` | Validation Error | Configuration validation failed |
| `130` | Interrupted | Command interrupted by user (Ctrl+C) |

### Using Exit Codes in Scripts

```bash
#!/bin/bash

# Build and check for success
if markata-go build --clean; then
    echo "Build successful!"
    rsync -av public/ user@server:/var/www/
else
    echo "Build failed with exit code $?"
    exit 1
fi
```

```bash
# Validate before deploying
markata-go config validate
if [ $? -eq 0 ]; then
    markata-go build --clean
fi
```

---

## Common Workflows

### Development

```bash
# Start development server
markata-go serve -v

# In another terminal, create new content
markata-go new "My New Post"
```

### Production Build

```bash
# Clean build for production
markata-go build --clean -v

# Or with environment-specific URL
MARKATA_GO_URL=https://example.com markata-go build --clean
```

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Validate configuration
markata-go config validate

# Build with production settings
MARKATA_GO_URL="${DEPLOY_URL}" markata-go build --clean

# Deploy (example)
aws s3 sync public/ s3://my-bucket/ --delete
```

See [[deployment-guide|Deployment]] for detailed deployment guides.

---

### migrate

Migrate from Python markata to markata-go. Analyzes configuration files, filter expressions, and templates for compatibility.

#### Usage

```bash
markata-go migrate [flags]
markata-go migrate config [flags]
markata-go migrate filter [expression]
markata-go migrate templates [path]
```

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--input` | `-i` | Input config file path | Auto-detect |
| `--output` | `-o` | Output config file path | None |
| `--dry-run` | `-n` | Show changes without writing | `false` |
| `--format` | `-f` | Output format (toml, yaml) | `toml` |
| `--json` | | Output results as JSON | `false` |

#### Subcommands

##### config

Migrate configuration file only.

```bash
markata-go migrate config -i markata.toml -o markata-go.toml
```

##### filter

Check and migrate filter expressions.

```bash
# Check a specific expression
markata-go migrate filter "published == 'True'"

# Check all filters in config
markata-go migrate filter
```

##### templates

Check template compatibility with pongo2.

```bash
markata-go migrate templates
markata-go migrate templates ./my-templates
```

#### Examples

```bash
# Full migration analysis
markata-go migrate

# Dry run to see what would change
markata-go migrate --dry-run

# Migrate and write output
markata-go migrate -o markata-go.toml

# Migrate from pyproject.toml
markata-go migrate -i pyproject.toml -o markata-go.toml

# Check a filter expression
markata-go migrate filter "templateKey in ['blog', 'til']"

# Check templates for compatibility
markata-go migrate templates
```

#### Exit Codes

| Code | Description |
|------|-------------|
| `0` | Migration successful, no issues |
| `1` | Migration completed with warnings |
| `2` | Migration has incompatibilities |
| `3` | Migration failed (invalid input) |

See [[migration-guide|Migration Guide]] for detailed migration instructions.

---

### explain

Show detailed information about markata-go for AI agents and developers.

#### Usage

```bash
markata-go explain [topic]
```

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `topic` | The topic to explain | No (shows overview if omitted) |

#### Available Topics

| Topic | Description |
|-------|-------------|
| (none) | General overview of markata-go |
| `build` | The build command and build process |
| `serve` | The development server |
| `new` | Creating new content |
| `init` | Initializing projects |
| `config` | Configuration system |
| `plugins` | Plugin system and development |
| `lifecycle` | Build lifecycle stages |
| `templates` | Template system |
| `feeds` | Feed generation system |

#### Examples

```bash
# Show general overview
markata-go explain

# Get detailed info about the build process
markata-go explain build

# Learn about the plugin system
markata-go explain plugins

# Understand the build lifecycle
markata-go explain lifecycle

# Learn about feeds and filtering
markata-go explain feeds
```

#### Use Cases

The `explain` command is particularly useful for:

- **AI coding agents**: Provides comprehensive context about markata-go's architecture and commands
- **New developers**: Quick reference for understanding how different parts work together
- **Debugging**: Understanding the build process and configuration options
- **Writing plugins**: Learning the plugin interfaces and lifecycle stages

---

## See Also

- [[getting-started|Getting Started]] - Quick introduction to markata-go
- [[configuration-guide|Configuration]] - Detailed configuration reference
- [[deployment-guide|Deployment]] - Deployment guides for various platforms
