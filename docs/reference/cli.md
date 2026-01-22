---
title: "CLI Reference"
description: "Complete reference for all markata-go commands, flags, and options"
date: 2024-01-15
published: true
template: doc.html
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

See [Configuration](../guides/configuration.md) for details on configuration options.

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
| `--no-watch` | | Disable file watching and auto-rebuild | `false` |
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

# Serve without file watching
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

#### Development Workflow

```bash
# Terminal 1: Start the server
markata-go serve -v

# The server will:
# - Build your site
# - Start serving at http://localhost:8000
# - Watch for file changes
# - Rebuild and reload on changes
```

---

### new

Create a new content file with frontmatter template.

#### Usage

```bash
markata-go new <title> [flags]
```

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `title` | The title of the new post | Yes |

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--dir` | Directory to create the post in | `posts` |
| `--draft` | Create as a draft | `true` |

#### Examples

```bash
# Create a new post (creates posts/my-first-post.md)
markata-go new "My First Post"

# Create in a specific directory
markata-go new "Hello World" --dir blog
# Creates: blog/hello-world.md

# Create as a draft (default behavior)
markata-go new "Work in Progress" --draft
# Creates: posts/work-in-progress.md with draft: true

# Create as published
markata-go new "Ready to Publish" --draft=false
# Creates: posts/ready-to-publish.md with draft: false, published: true
```

#### Generated File

The command creates a markdown file with this template:

```markdown
---
title: "My First Post"
slug: "my-first-post"
date: 2024-01-15
draft: true
published: false
tags: []
---

# My First Post

```

The slug is automatically generated from the title by:
- Converting to lowercase
- Replacing spaces with hyphens
- Removing special characters

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

**Generated Config:**

```toml
[markata-go]
title = "My Site"
description = "A site built with markata-go"
url = "https://example.com"
author = "Your Name"

output_dir = "public"
templates_dir = "templates"
assets_dir = "static"

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]

[markata-go.markdown]
extensions = ["tables", "strikethrough", "autolinks", "tasklist"]
```

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

See [Deployment](../guides/deployment.md) for detailed deployment guides.

---

## See Also

- [Getting Started](../getting-started.md) - Quick introduction to markata-go
- [Configuration](../guides/configuration.md) - Detailed configuration reference
- [Deployment](../guides/deployment.md) - Deployment guides for various platforms
