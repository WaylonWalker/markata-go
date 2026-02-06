# Random Post Specification

This document specifies the built-in `random_post` plugin.

## Goal

Provide a static `/random/` endpoint that redirects in the browser to a random eligible post, enabling serendipitous discovery without requiring a server.

## Lifecycle

- **Stage:** `write`
- **Determinism:** build output is deterministic; randomness happens at runtime in the browser.

## Configuration

Configuration is namespaced under the top-level `markata-go` section.

```toml
[markata-go.random_post]
enabled = true

# Directory path under output/ (default: "random")
path = "random"

# Optional JSON file listing eligible hrefs (default: false)
emit_posts_json = false

# Optional denylist of tags; posts containing any of these tags are excluded
exclude_tags = ["private", "draft"]
```

### Fields

| Field | Type | Default | Description |
|------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable generation |
| `path` | string | `"random"` | Output path segment for the endpoint |
| `emit_posts_json` | bool | `false` | Also write `{path}/posts.json` with eligible hrefs |
| `exclude_tags` | []string | `[]` | Case-insensitive tag denylist |

## Eligibility

A post is eligible if all of the following are true:

- `published == true`
- `draft == false`
- `private == false`
- `skip == false`
- `href` is non-empty
- does not contain any `exclude_tags` (if configured)

## Generated Output

When enabled, the plugin writes:

- `{output_dir}/{path}/index.html`
- `{output_dir}/{path}/posts.json` (optional, when `emit_posts_json = true`)

### `index.html` behavior

The page must:

- choose a random href from the eligible list
- redirect using `window.location.replace(...)`
- include a `<noscript>` fallback that links to the home page and/or lists eligible posts
- set `robots` to `noindex` to avoid indexing the redirect endpoint

If the eligible list is empty, the page must not error; it should present a helpful fallback message/link.

### `posts.json` format

If emitted, `posts.json` contains a JSON array of href strings:

```json
["/post-a/","/post-b/"]
```

Ordering is stable (deterministic) to avoid unnecessary diffs.

## Error Handling

- If the plugin is disabled (default), it does nothing.
- If output directories cannot be created or files cannot be written, the plugin returns an error.
