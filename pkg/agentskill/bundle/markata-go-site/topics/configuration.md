# Configuration

Use this topic when the task involves `markata-go.toml`, environment overrides, feed setup, theme settings, or figuring out which config value is active.

## Preferred File

For standalone sites, prefer `markata-go.toml` unless the repo is already using YAML or JSON.

When a site has many feeds or environment-specific settings, keep `markata-go.toml` as the entrypoint and split the rest with `[markata-go].include`.

For blogroll work, remember that explicit feed config overrides such as `site_url`, `image_url`, `title`, `handle`, aliases, category, and tags should take effect even when cached feed data is reused. Do not assume a stale cache must be deleted before config fixes apply.

## Config Discovery

When `--config` is not passed, markata-go looks for config in this order:

1. `markata-go.toml`
2. `markata-go.yaml`
3. `markata-go.yml`
4. `markata-go.json`
5. `~/.config/markata-go/config.toml` (user-level fallback)

## High-Value Commands

- `markata-go config show`
- `markata-go config show --annotate`
- `markata-go config show --diff`
- `markata-go config get <key>`
- `markata-go config validate`

Bare `markata-go config` behaves like `markata-go config show`.

## Core Namespace

Most settings live under `[markata-go]` and nested namespaces like `[markata-go.glob]` and `[markata-go.theme]`.

Example:

```toml
[markata-go]
title = "My Site"
url = "https://example.com"
output_dir = "public"
assets_dir = "static"
templates_dir = "templates"

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
use_gitignore = true
```

## Safe Defaults For New Sites

```toml
[markata-go]
title = "My Site"
url = "https://example.com"
output_dir = "public"
assets_dir = "static"
templates_dir = "templates"

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
```

## Common Keys To Check

- `output_dir`
- `url`
- `templates_dir`
- `assets_dir`
- `concurrency`
- `hooks`
- `disabled_hooks`
- `glob.patterns`
- `theme.palette`
- `layout.name`
- feed definitions under `[[markata-go.feeds]]`

## Useful Site Patterns

### Blog-like Site

```toml
[markata-go]
title = "My Blog"
url = "https://example.com"

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true
```

### Docs-like Site

```toml
[markata-go]
title = "Project Docs"
url = "https://docs.example.com"

[markata-go.glob]
patterns = ["docs/**/*.md"]

[markata-go.layout]
name = "docs"
```

For feed-specific patterns, read `../reference/feed-patterns.md`.

## Layout Config Is Often The Right Answer

If the task sounds like page structure rather than full template replacement, inspect layout config first.

Example:

```toml
[markata-go.layout]
name = "blog"

[markata-go.layout.paths]
"/docs/" = "docs"
"/about/" = "landing"

[markata-go.layout.blog]
show_toc = true
show_prev_next = true

[markata-go.layout.docs]
sidebar_position = "left"
toc_position = "right"
```

## Hooks And Plugin Toggles

If a feature seems absent, check whether it was disabled in config before editing templates or content.

```toml
[markata-go]
hooks = ["default"]
disabled_hooks = []
```

Use `disabled_hooks` to isolate plugin-related issues. Use `markata-go explain plugins` to see available hook names.

Optional source-backed docs can also be enabled in config. For Python API reference pages:

```toml
[markata-go]
hooks = ["default", "python_docs"]

[markata-go.python_docs]
enabled = true
patterns = ["src/**/*.py"]
slug_prefix = "api"
template = "docs"
```

This generates posts from Python modules without replacing normal markdown content loading, but it only works when `python_docs` is explicitly listed in `hooks`.

## Authors Config

If the site has multiple authors, configure them under `[markata-go.authors]`:

```toml
[markata-go.authors]
generate_pages = true
url_pattern = "/authors/{author}/"
feeds_enabled = true

[markata-go.authors.authors.waylon]
name = "Waylon Walker"
bio = "Software engineer"
avatar = "/images/waylon.jpg"
url = "https://waylonwalker.com"
default = true
active = true

[markata-go.authors.authors.waylon.social]
github = "https://github.com/waylonwalker"
```

Key fields per author: `name`, `bio`, `email`, `avatar`, `url`, `social` (map), `guest`, `active`, `default`, `role`, `contributions` (CReDiT roles).

When `generate_pages = true`, author profile pages are auto-generated. When `feeds_enabled = true`, per-author feeds are created.

## Environment Overrides

Environment variables use the `MARKATA_GO_` prefix.

Examples:

```bash
MARKATA_GO_URL=https://staging.example.com markata-go build
MARKATA_GO_OUTPUT_DIR=dist markata-go build
```

## Config Merging

Local override files like `markata-go.local.toml` are NOT auto-discovered. They must be passed explicitly via the `--merge-config` / `-m` flag:

```bash
markata-go build -m markata-go.local.toml
markata-go serve -m markata-go.local.toml
markata-go config show -m markata-go.local.toml
markata-go config validate -m markata-go.local.toml
```

Multiple merge files can be specified and are applied in order on top of the base config.

## Config Composition

The main config can include other config files:

```toml
[markata-go]
include = ["config/common/*.toml", "config/feeds/*.toml"]
```

Key rules:

- include paths are relative to the file that declared them
- glob matches load in lexicographic order
- repeated includes are loaded once
- feed entries merge by `slug`
- environment variables still win last

## Guidance

- Prefer editing the site's real config file over hardcoding behavior in templates.
- Confirm whether a value is user-defined or default before changing it.
- Keep related settings together under existing namespaces.
- For feed work, inspect both feed definitions and feed defaults.
- When debugging paths, remember CLI flags override config.
- If a change is site-wide, config is usually the right place before template logic or plugin logic.
