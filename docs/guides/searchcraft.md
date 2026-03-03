---
title: "Searchcraft Integration"
description: "Connect markata-go to Searchcraft Core for fast semantic search across your site."
date: 2026-03-02
published: true
tags:
  - documentation
  - search
  - searchcraft
---

# Searchcraft Integration

This guide explains how to hook markata-go into a self-hosted [Searchcraft Core](https://www.searchcraft.io/) instance, keep documents synchronized on every build, and surface results through a `/search` page.

## Run Searchcraft via Podman

1. Pull the official image:

    ```bash
    podman pull searchcraftinc/searchcraft-core:latest
    ```

2. Start the service with persistent storage and port 8000 exposed:

    ```bash
    podman run -d --name searchcraft -p 8000:8000 \
      -v searchcraft-data:/data \
      searchcraftinc/searchcraft-core:latest
    ```

3. Verify health:

    ```bash
    curl http://localhost:8000/health
    ```

4. Create access keys (ingest + read) per [Searchcraft API docs](https://docs.searchcraft.io/api/reference/keys/).

## Configure markata-go

Add a `[searchcraft]` table to `markata-go.toml` (or the equivalent config file) with your endpoint and keys.

```toml
[searchcraft]
enabled = true
endpoint = "http://localhost:8000"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_prefix = "waylonwalker"
index_per_site = true
delete_missing = true
batch_size = 100
skip_on_fast_mode = true
```

- `endpoint` must be a reachable HTTP(S) URL.
- `ingest_key` performs the write operations (`POST`/`DELETE`). The plugin refuses to run without it.
- `read_key` is used by the `/search` page to query Searchcraft (`POST /index/{index}/search`).
- `index_prefix` and `index_per_site` control deterministic index naming (`{index_prefix}_{normalized_site_name}`). Use `index_name` to override the computed name.
- `batch_size` controls how many documents are sent per request (defaults to 100).
- `delete_missing` removes documents that disappear from the build.
- `skip_on_fast_mode` prevents remote synchronization during fast/local builds.

The configuration is validated at build time. Invalid endpoints or missing keys produce errors.

## Build-time synchronization

- The Searchcraft cleanup plugin runs after HTML is written.
- It filters posts by the `published`, `draft`, and `private` flags according to `include_drafts`/`include_private`.
- Documents include rich metadata (`id`, `title`, `summary`, `body`, `content`, `tags`, `authors`, `url`, `path`, `site`, `feed`, `template`, `published_at`, `modified_at`, `published`, `draft`, `private`).
- Each document is hashed (SHA256) and compared against `.markata/searchcraft-cache.json`. Unchanged posts are skipped.
- Changed posts are upserted via `POST /index/{index}/documents` and deletions are issued per slug via `DELETE /index/{index}/documents/query` when `delete_missing` is `true`.
- The cache file mirrors the build cache directory so future builds know what changed.

## Index naming

- By default, indexes are named `{index_prefix}_{normalized_site}` where normalization keeps only `[a-z0-9_-]` characters.
- Set `index_name` to target a shared index (handy for multi-site federation).
- The plugin honors `site_name` from the config or falls back to the site title when constructing the suffix.

## Search page (example usage)

Expose Searchcraft queries through a `/search` page like the one in `waylonwalker.com-go`. Within that page access `config.searchcraft` to read the `read_key`, `endpoint`, and `resolved_index` values generated during build. Example JavaScript:

```html
<script>
const indexName = "{{ config.searchcraft.resolved_index }}";
const endpoint = "{{ config.searchcraft.endpoint }}".replace(/\/$/, "");
const readKey = "{{ config.searchcraft.read_key }}";

async function search(query) {
  if (!query) {
    return [];
  }
  const response = await fetch(`${endpoint}/index/${encodeURIComponent(indexName)}/search`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: readKey,
    },
    body: JSON.stringify({ query: { fuzzy: { ctx: query } }, limit: 10 }),
  });
  const payload = await response.json();
  return payload.data?.hits ?? [];
}
</script>
```

The `/search` page can render results with the `searchcraft` payload fields for titles, excerpts, tags, and URLs. Pass `read_key` through environment variables or a secure secrets store so you can rotate it independently of builds.

## Next steps

1. Run `markata-go build` after configuring Searchcraft to populate the index.
2. Open `/search` to exercise results and confirm that semantic ranking is visible.
3. Rotate keys by updating the config and restarting Searchcraft; the plugin will reindex everything during the next build when caches are invalidated.
