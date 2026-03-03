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

## Security model (recommended)

Use a two-key setup:

- `SEARCHCRAFT_INGEST_KEY`: write key used only by markata-go builds (upsert/delete/index management)
- `SEARCHCRAFT_READ_KEY`: read-only key exposed to browser search UI (`POST /index/{index}/search`)

Security goals:

1. Users can query search
2. Users cannot ingest/update/delete documents
3. Only your build system can write

You enforce this with both key scope and network policy:

- key scope: read key cannot call write endpoints
- network policy: write endpoints only reachable from build host/CI network

## Deployment topologies

### Same host (site + Searchcraft on one machine)

- Bind Searchcraft to localhost only
- Put a reverse proxy in front
- Expose only search endpoint publicly
- Keep ingest routes private or IP-allowlisted

### Separate host (build host != Searchcraft host)

- Expose Searchcraft behind TLS (`https`)
- Allow ingest endpoints only from CI/build IPs or VPN network
- Expose search endpoint publicly with read key

## Podman (secure baseline)

1. Pull the official image:

    ```bash
    podman pull docker.io/searchcraftinc/searchcraft-core:latest
    ```

2. Create persistent storage:

    ```bash
    podman volume create searchcraft-data
    ```

3. Start Searchcraft bound to localhost (same-host secure default):

    ```bash
    podman run -d --name searchcraft -p 127.0.0.1:18000:18000 \
      -v searchcraft-data:/data \
      docker.io/searchcraftinc/searchcraft-core:latest --port 18000
    ```

4. Verify health:

    ```bash
    curl http://localhost:18000/healthcheck
    ```

5. Create separate ingest/read keys per [Searchcraft API docs](https://docs.searchcraft.io/api/reference/keys/).

## Docker

```bash
docker volume create searchcraft-data

docker run -d --name searchcraft \
  -p 127.0.0.1:18000:18000 \
  -v searchcraft-data:/data \
  searchcraftinc/searchcraft-core:latest --port 18000

curl http://localhost:18000/healthcheck
```

Use firewall rules (or cloud security groups) so only trusted build systems can reach write endpoints if you expose Searchcraft beyond localhost.

## Docker Compose (same host)

Use `.env.searchcraft` (never commit real keys):

```bash
SEARCHCRAFT_INGEST_KEY=replace-me-ingest
SEARCHCRAFT_READ_KEY=replace-me-read
```

`compose.searchcraft.yml`:

```yaml
services:
  searchcraft:
    image: searchcraftinc/searchcraft-core:latest
    command: ["--port", "18000"]
    ports:
      - "127.0.0.1:18000:18000"
    volumes:
      - searchcraft-data:/data
    restart: unless-stopped

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
    depends_on:
      - searchcraft

volumes:
  searchcraft-data:
```

Example `Caddyfile` that only exposes search publicly:

```caddyfile
search.example.com {
  encode gzip

  @search path_regexp search ^/index/[^/]+/search$
  reverse_proxy @search searchcraft:18000

  @health path /health /healthcheck
  reverse_proxy @health searchcraft:18000

  respond "forbidden" 403
}
```

For separate-host builds, add an internal ingress hostname (or VPN-only route) that permits write operations.

## Kubernetes (production pattern)

See [[searchcraft-kubernetes|Searchcraft on Kubernetes]] for a full, copy-paste deployment with PVC, ingress split (read vs write), and NetworkPolicy examples.

Core pieces:

- `Deployment` for Searchcraft container
- `PersistentVolumeClaim` mounted at `/data`
- `Service` for in-cluster access
- `Ingress` (or Gateway) with TLS
- `NetworkPolicy` allowing:
  - read traffic from ingress
  - write traffic only from CI/build namespace or CIDR
- `Secret` storing ingest/read keys for automation jobs

Minimal example (trimmed):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: searchcraft
spec:
  replicas: 1
  selector:
    matchLabels:
      app: searchcraft
  template:
    metadata:
      labels:
        app: searchcraft
    spec:
      containers:
        - name: searchcraft
          image: searchcraftinc/searchcraft-core:latest
          args: ["--port", "18000"]
          ports:
            - containerPort: 18000
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: searchcraft-data
---
apiVersion: v1
kind: Service
metadata:
  name: searchcraft
spec:
  selector:
    app: searchcraft
  ports:
    - port: 18000
      targetPort: 18000
```

Add ingress rules that expose only `POST /index/*/search` publicly, and keep write routes on an internal ingress class.

## Configure markata-go

Add a `[markata-go.searchcraft]` table to `markata-go.toml` (or the equivalent config file) with your endpoint and keys.

```toml
[markata-go.searchcraft]
enabled = true
endpoint = "https://search.example.com"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_prefix = "waylonwalker"
index_per_site = true
delete_missing = true
batch_size = 100
skip_on_fast_mode = true
```

- `endpoint` must be a reachable HTTP(S) URL.
- `ingest_key` performs write operations (`POST`/`DELETE`) and must never be exposed in public JS.
- `read_key` is used by `/search` page queries (`POST /index/{index}/search`).
- `index_prefix` and `index_per_site` control deterministic index naming (`{index_prefix}_{normalized_site_name}`). Use `index_name` to override the computed name.
- `batch_size` controls how many documents are sent per request (defaults to 100).
- `delete_missing` removes documents that disappear from the build.
- `skip_on_fast_mode` prevents remote synchronization during fast/local builds.

The configuration is validated at build time. Invalid endpoints or missing keys produce errors.

## Same-host vs separate-host markata examples

### Same host

```toml
[markata-go.searchcraft]
enabled = true
endpoint = "http://127.0.0.1:18000"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_per_site = true
```

### Separate host

```toml
[markata-go.searchcraft]
enabled = true
endpoint = "https://search.example.com"
ingest_key = "${SEARCHCRAFT_INGEST_KEY}"
read_key = "${SEARCHCRAFT_READ_KEY}"
index_per_site = true
delete_missing = true
```

For separate host deployments, require TLS and allow write paths only from trusted build-network sources.

## Build-time synchronization

- The Searchcraft cleanup plugin runs after HTML is written.
- It filters posts by the `published`, `draft`, and `private` flags according to `include_drafts`/`include_private`.
- Documents include rich metadata (`id`, `title`, `summary`, `body`, `content`, `card_html`, `tags`, `authors`, `url`, `path`, `site`, `feed`, `template`, `published_at`, `modified_at`, `published`, `draft`, `private`).
- `card_html` is rendered from `partials/cards/card-router.html`, so `/search` can display the exact same card markup used in feeds.
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

The `/search` page can render results with `hit.doc.card_html` for exact feed/embed card parity. If `card_html` is missing, fall back to title/summary rendering. Pass `read_key` through environment variables or a secure secrets store so you can rotate it independently of builds.

## Secure operations checklist

- Keep ingest and read keys separate
- Never expose `SEARCHCRAFT_INGEST_KEY` in templates or frontend bundles
- Use HTTPS between markata-go and Searchcraft when remote
- Restrict write endpoints by IP/network policy
- Rotate keys periodically and on incident
- Back up Searchcraft `/data` volume

## Next steps

1. Run `markata-go build` after configuring Searchcraft to populate the index.
2. Open `/search` to exercise results and confirm that semantic ranking is visible.
3. Rotate keys by updating the config and restarting Searchcraft; the plugin will reindex everything during the next build when caches are invalidated.
