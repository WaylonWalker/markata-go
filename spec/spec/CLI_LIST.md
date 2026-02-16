# CLI List Specification

## Overview

The `list` command provides a fast, scriptable way to inspect posts, tags, and feeds without launching the TUI. It supports table, JSON, CSV, and path-only output for shell pipelines.

## Command Structure

```
[name] list posts [flags]
[name] list tags [flags]
[name] list feeds [flags]
[name] list feeds posts <feed-name>
```

## Shared Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `table` | Output format: `table`, `json`, `csv`, `path` |
| `--sort` | string | varies | Sort field (see per-command options) |
| `--order` | string | varies | Sort order: `asc` or `desc` |

## `list posts`

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filter` | string | empty | Filter expression applied to posts |
| `--feed` | string | empty | Limit posts to a feed by name |

### Sort Fields (default: `date`, order `desc`)

`date`, `title`, `words`, `path`, `reading_time`, `tags`

### Output Columns

| Format | Columns |
|--------|---------|
| `table` | Title, Date, Words, Read, Tags, Path |
| `json` | `title`, `date`, `words`, `reading_time`, `tags`, `path` |
| `csv` | Same order as table |
| `path` | `path` only, one per line |

You can also list posts for a feed:

```bash
[name] list posts --feed <feed-name>
```

## `list tags`

### Sort Fields (default: `count`, order `desc`)

`name`, `count`, `words`, `reading_time`

### Output Columns

| Format | Columns |
|--------|---------|
| `table` | Tag, Count, Words, Read, Slug |
| `json` | `name`, `count`, `words`, `reading_time`, `slug` |
| `csv` | Same order as table |
| `path` | `name` only, one per line |

## `list feeds`

### Sort Fields (default: `name`, order `asc`)

`name`, `posts`, `words`, `reading_time`, `avg_reading_time`

### Output Columns

| Format | Columns |
|--------|---------|
| `table` | Name, Posts, Words, Total Read, Avg Read, Output |
| `json` | `name`, `posts`, `words`, `reading_time`, `avg_reading_time`, `output` |
| `csv` | Same order as table |
| `path` | `path` (fallback to name), one per line |

### Posts Output

Use `list feeds posts` with a feed name to print the posts in that feed:

```bash
[name] list feeds posts <feed-name>
```

## Output Rules

- `table` output is plain ASCII, no color.
- `json` values use raw types (ints for counts, minutes for reading time).
- `csv` uses a header row with column names.
- `path` outputs one value per line with no header.

## Cache

`list` and `tui` use a persistent cache at `.markata/cache/list.json`.

- Cache is reused when the config hash matches and file metadata is unchanged.
- Partial refresh re-parses only changed files and rebuilds feed/tag aggregates from cached posts.
- Delete the cache file to force a full refresh.

## Errors

- Invalid format or sort field returns exit code 1 and prints a clear error.
- Invalid filter expression returns exit code 1 and prints the filter error.

## Data Source

The command loads posts and feeds via the standard lifecycle, then queries data through service interfaces. This ensures posts include computed fields like word count and reading time.
