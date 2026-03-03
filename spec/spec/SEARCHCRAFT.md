# Searchcraft Integration

Describes the metadata synchronization between markata-go and a self-hosted Searchcraft Core instance.

## Overview

- Adds a cleanup-stage plugin that upserts build output into Searchcraft via the `/index/:index/documents` and `/index/:index/documents/query` endpoints.
- Supports per-site index naming so a single Searchcraft instance can host multiple markata-go sites (prefix + normalized site name, or a configured `index_name`).
- Skips unchanged posts by hashing the semantic payload and caching the hashes in `.markata/searchcraft-cache.json`.
- Handles deletions by issuing delete-by-query calls when cached IDs are absent from the current build.
- Provides configuration to control batching, draft/private inclusion, fast-mode behavior, and cache cleanup.

## Lifecycle Contract

- Runs in the **cleanup** stage with `PriorityLast` so it sees the final set of rendered posts.
- If searchcraft syncing is disabled or the fast-mode flag is set (and `skip_on_fast_mode` is true) the plugin is skipped.

## Configuration Schema

Searchcraft is configured through a `[searchcraft]` table in `markata-go.toml`. Fields:

| Key | Type | Default | Description |
| --- | --- | ------- | ----------- |
| `enabled` | bool | `false` | Enable remote syncing. |
| `endpoint` | string | `http://localhost:8000` | Base URL of Searchcraft Core. Required when enabled. |
| `ingest_key` | string | `""` | Ingestion key used for POST/DELETE calls. Required when enabled. |
| `read_key` | string | `""` | Read-only key exposed to client search pages. Optional but recommended. |
| `site_name` | string | `config.title` | Friendly site identifier used when `index_per_site` is true. |
| `index_name` | string | `""` | Override the final index name entirely. |
| `index_prefix` | string | `markata` | Namespace prefix for generated indexes. |
| `index_separator` | string | `_` | Separator between prefix and site segment. |
| `index_per_site` | bool | `true` | Build a unique index per site name. |
| `batch_size` | int | `100` | Number of documents per ingest request. |
| `delete_missing` | bool | `true` | Remove docs whose source files disappeared. |
| `include_drafts` | bool | `false` | Index draft posts. |
| `include_private` | bool | `false` | Index private posts. |
| `skip_on_fast_mode` | bool | `true` | Skip remote syncing when fast-mode is flagged. |

Validation: when enabled, `endpoint` must be a valid http/https URL and `ingest_key` must be set. `batch_size` must be ≥ 0.

## Document Payload

Each Searchcraft document includes the following fields:

| Field | Description |
| --- | --- |
| `id` | Post slug or path (unique). |
| `title` | Post title (frontmatter or generated). |
| `summary` | Description/excerpt. |
| `body` | Rendered HTML article body. |
| `content` | Raw markdown content. |
| `card_html` | Pre-rendered card markup from `partials/cards/card-router.html` (exact feed card layout per post type). |
| `tags` | Sorted list of tags. |
| `authors` | List of author IDs. |
| `url` | Absolute URL (`config.url` + `post.href`). |
| `path` | Relative href. |
| `site` | Site title for index partitioning/filters. |
| `feed` | `prevnext_feed` (series slug) if set. |
| `template` | Template name. |
| `published_at` | RFC3339 `date`. |
| `modified_at` | RFC3339 `modified`. |
| `published` | `post.published`. |
| `draft` | `post.draft`. |
| `private` | `post.private`. |

Document hashes are derived from the concatenation of these fields plus boolean flags so identical payloads skip building.

## Index Naming

- If `index_name` is provided that value (normalized to `[a-z0-9-_]`) is used directly.
- Otherwise the index name is `index_prefix + index_separator + normalized(site_name)` when `index_per_site` is true.
- `site_name` defaults to the config title but can also be set explicitly in `[searchcraft]`.
- Normalization collapses invalid characters to hyphens and trims extra separators.

## Sync Strategy

1. Build loads `.markata/searchcraft-cache.json` (JSON map of `document_id → document_hash`).
2. Iterates posts, filters out `Skip`, private/draft (unless configured), and constructs payloads.
3. Hashes each payload; if the hash matches the cached value the document is skipped.
4. Changed/new payloads are sent in batches via `POST /index/{index}/documents` with the ingestion key in the `Authorization` header.
5. Documents missing from the current build are deleted via `DELETE /index/{index}/documents/query` using the same ingestion key.
6. After syncing the cache file is rewritten with the latest hashes to enable efficient future runs.

## Cache File

- Stored at `{output_dir}/../.markata/searchcraft-cache.json` (mirrors the build cache directory).
- Format:
  ```json
  {
    "entries": {
      "path/to/post": {
        "document_hash": "abc",
        "updated_at": "2026-01-01T00:00:00Z"
      }
    }
  }
  ```

## Frontend Considerations

- Supply `read_key` to the search page so it can call `POST /index/{index}/search`.
- Render `doc.card_html` directly on `/search` to reuse the same card templates used by feeds and embeds.
- The search payload is served by static pages (e.g., `/search`) and uses the resolved index name for the current site.
