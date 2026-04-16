# CLI Usage

Use this topic for everyday site work and safe project inspection.

## High-Value Commands

### Build And Serve

- `markata-go build`
- `markata-go build --clean` (remove output + build cache)
- `markata-go build --clean-all` (also remove external caches: blogroll, embeds, mentions)
- `markata-go build --fast` (skip minification, CSS purging, Pagefind indexing)
- `markata-go build --dry-run` (run through Collect, show counts, skip Write)
- `markata-go build --benchmark-json benchmark.json`
- `markata-go build -v --benchmark-detailed`
- `markata-go serve` (dev server with live reload)
- `markata-go serve --fast`

### Config And Inspection

- `markata-go config show`
- `markata-go config show --annotate`
- `markata-go config show --diff`
- `markata-go config get <key>`
- `markata-go config validate`

Bare `markata-go config` behaves like `markata-go config show`.
- `markata-go list posts`
- `markata-go list feeds`
- `markata-go list tags`

### Search

- `markata-go search <query>` (BM25-ranked full-text search)
- `markata-go search <query> --format json` (machine-readable output)
- `markata-go search <query> --filter "published == True"` (combine with filter)
- `markata-go search <query> --fields title,tags` (restrict fields)
- `markata-go search <query> --fuzzy` (typo-tolerant matching)
- `markata-go search <query> --limit 10` (cap results)
- `markata-go search <query> --format path` (file paths only, for piping)
- `markata-go search <query> --sort date` (sort by date instead of relevance)

### Content Creation

- `markata-go new` (create content from built-in templates)
- `markata-go new --list` (list available content templates)
- `markata-go init` (initialize a new project with TUI wizard)
- `markata-go init --plain` (plain text prompts for non-TTY environments)

### Content Quality

- `markata-go lint` (lint markdown files for common issues)
- `markata-go lint --fix` (auto-fix fixable issues)
- `markata-go lint --dry-run` (show files without linting)

### Theme And Palette

- `markata-go palette list`
- `markata-go palette info <name>`
- `markata-go palette check <name>` (WCAG contrast validation)
- `markata-go palette check <name> --strict` (WCAG AAA instead of AA)
- `markata-go palette check --all` (check all palettes)
- `markata-go palette preview <name>`
- `markata-go palette new <name>`
- `markata-go palette clone <source>`
- `markata-go theme render-all`
- `markata-go theme gallery`
- `markata-go theme check-all` (check 16 contrast combos per palette)
- `markata-go theme check-all --colorblindness` (simulate color vision deficiencies)
- `markata-go aesthetic list`
- `markata-go aesthetic show <name>`

### Explain

- `markata-go explain` (list topics)
- `markata-go explain config`
- `markata-go explain templates`
- `markata-go explain plugins`
- `markata-go explain agents`
- `markata-go explain feeds`
- `markata-go explain lifecycle`

### Migration And Import

- `markata-go migrate config` (convert Python markata config)
- `markata-go migrate filter [expression]` (check filter expression compatibility)
- `markata-go migrate templates [path]` (validate template compatibility)
- `markata-go migrate compare --old <dir> --new <dir>` (compare site outputs)
- `markata-go import rss <url>` (import from RSS/Atom feed)
- `markata-go import jsonfeed <url>` (import from JSON Feed)
- Shared import flags: `--output`, `--since`, `--dry-run`, `--tags`

### Maintenance

- `markata-go update` (self-update from GitHub releases)
- `markata-go update --check` (check for updates without installing)
- `markata-go benchmark --scenario small|medium|large` (performance benchmarks)
- `markata-go agent list-agents` (list supported agent ids and their install paths)
- `markata-go agent install` (install bundled agent skill into the detected project agent or the universal layout)
- `markata-go agent install --agent <name> [-g]` (choose a specific agent and optional global install scope)
- `markata-go agent doctor` (check for drift after binary upgrades)
- `markata-go version`

## Lint Checks

`markata-go lint` detects:

- duplicate YAML keys in frontmatter
- invalid date formats (non-ISO 8601)
- malformed image links (missing alt text)
- protocol-less URLs (should use `https://`)
- encryption policy issues (when encryption is configured)

Use `--fix` to auto-fix fixable issues. Only error-severity issues cause a non-zero exit code; warnings alone pass.

## Global Flags Agents Should Know

- `-c`, `--config`: use a specific config file
- `-m`, `--merge-config`: merge override configs such as `fast.toml`
- `-o`, `--output`: override the output directory without editing config
- `-v`, `--verbose`: show detailed logs and plugin-stage hints
- `-q`, `--quiet`: suppress non-essential progress output
- `--no-input`: disable prompts for scripted or non-interactive runs

Examples:

```bash
markata-go build -c markata-go.toml
markata-go serve -m fast.toml
markata-go build -o dist
markata-go new "My Post" --no-input
markata-go build -v
markata-go lint --fix
```

## What To Use When

- inspect content inventory: `markata-go list posts`
- search for content by keyword: `markata-go search <query>`
- inspect feed definitions and sizes: `markata-go list feeds`
- inspect resolved configuration: `markata-go config show`
- create new content: `markata-go new`
- lint content before committing: `markata-go lint`
- validate config before deploy: `markata-go config validate`
- validate palette contrast: `markata-go palette check <name>`
- interactive local editing: `markata-go serve --fast`
- final verification: `markata-go build`
- bootstrap a new project: `markata-go init`
- migrate from Python markata: `markata-go migrate config`
- import content from external feeds: `markata-go import rss <url>`

## Operator Patterns

- use `-m fast.toml` when you want a lighter dev build without rewriting main config
- use the same `-m` overrides with `config show` and `config validate` when you need to inspect or verify the exact config that `build` or `serve` will use
- use `-c` when a repo has multiple configs or examples and you need the exact active one
- use `-o dist` in CI or preview contexts when you want a temporary artifact path
- use `--no-input` for automation or when the agent must avoid prompts
- use `-v` when debugging plugin order, missing outputs, or config resolution issues
- use `--dry-run` on build or lint to preview behavior without side effects

## Guidance

- Prefer `list` commands when you need structured inspection.
- Prefer `search` when you need to find posts by content or keyword.
- Prefer `explain` when you need command-specific or subsystem context.
- Prefer `serve` for interactive local work and `build` for validation or CI-like runs.
- Run `lint` before committing content changes.
- Run `palette check` after creating or modifying palettes.
- Use `--verbose` only when normal output is not enough.
- Keep primary results script-friendly by using built-in machine-readable output when available.
- Prefer merged config overrides over editing the main config for temporary local or CI changes.
