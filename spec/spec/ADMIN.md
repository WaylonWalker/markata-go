# Admin / CMS Specification

This document specifies the admin CMS functionality integrated into `markata-go serve`.

## Overview

The admin system provides a browser-based editor for markdown posts with frontmatter. It is accessed via `/__admin` routes on the serve server.

## Key Design Decisions

- **Source of truth**: Markdown files with YAML frontmatter remain the only source of truth
- **Frontmatter handling**: Parse as YAML for validation, re-serialize with consistent clean formatting; no mixed JSON/YAML, sorted keys, standard YAML practices
- **Auth model**: File-backed secrets in mounted directory; first-run setup creates admin user if no secrets exist
- **Git integration**: Explicit actions after save (status, diff, stage, commit, push); no auto-commit
- **Preview**: Uses real build output after save + rebuild

## Route Structure

| Route | Method | Description |
|-------|--------|-------------|
| `/__admin` | GET | Redirect to `/__admin/` |
| `/__admin/` | GET | Admin shell (login, setup, or editor) |
| `/__admin/dashboard` | GET | Dashboard with editable posts |
| `/__admin/editor` | GET | Post editor shell |
| `/__admin/settings` | GET | Settings editor shell |
| `/__admin/api/posts` | GET | List editable posts |
| `/__admin/api/post` | GET | Get single post (frontmatter + body) |
| `/__admin/api/post` | PUT | Save post |
| `/__admin/api/post` | POST | Create new post |
| `/__admin/api/post` | DELETE | Delete post |
| `/__admin/api/settings` | GET | Get active config file for editing |
| `/__admin/api/settings` | PUT | Save active config file |
| `/__admin/api/build-trigger` | POST | Trigger rebuild |
| `/__admin/api/build-status` | GET | Get build status |
| `/__admin/api/git/status` | GET | Git status for repo |
| `/__admin/api/git/diff` | GET | Git diff for file |
| `/__admin/api/git/stage` | POST | Git stage file |
| `/__admin/api/git/commit` | POST | Git commit |
| `/__admin/api/git/push` | POST | Git push |
| `/__admin/api/setup` | POST | First-run admin setup |

## Authentication

### Secret Discovery

Secrets are stored in a mounted directory (default: `.markata-secrets/`).

Required files:
- `admin_username` - admin username
- `admin_password_hash` - bcrypt/Argon2 hash of password
- `session_hmac_key` - HMAC key for session tokens

### First-Run Setup

When `/__admin` is accessed and no valid secrets exist:
1. Redirect to setup page
2. User enters username and password
3. Server creates secrets directory and writes credential files
4. User is redirected to login

### Session Management

- Cookie-based sessions with `HttpOnly`, `SameSite=Lax`, `Secure` (when not localhost)
- Session contains: user ID, expiration, HMAC signature
- CSRF tokens required for all mutating requests (POST, PUT, DELETE)

### Security

- Rate limiting on login and setup endpoints
- Default refusal for remote admin over plain HTTP
- Path validation: only allow edits under content directories

## Post Editing API

### List Posts

```
GET /__admin/api/posts
```

Response:
```json
{
  "posts": [
    {
      "path": "pages/post/hello-world.md",
      "title": "Hello World",
      "slug": "hello-world",
      "date": "2024-01-15",
      "published": true,
      "modified": "2024-01-16T10:30:00Z"
    }
  ]
}
```

### Get Post

```
GET /__admin/api/post?path=pages/post/hello-world.md
```

Response:
```json
{
  "path": "pages/post/hello-world.md",
  "frontmatter": "title: Hello World\ndate: 2024-01-15\npublished: true\ntags:\n  - hello\n  - world\n",
  "body": "# Hello World\n\nThis is the content...",
  "preview_url": "/hello-world/",
  "slug": "hello-world",
  "git_status": "modified",
  "base_hash": "abc123",
  "exists": true
}
```

### Save Post

```
PUT /__admin/api/post
```

Request:
```json
{
  "path": "pages/post/hello-world.md",
  "frontmatter": "title: Hello World\ndate: 2024-01-15\npublished: true\ntags:\n  - hello\n  - world\n",
  "body": "# Hello World\n\nThis is the content...",
  "base_hash": "abc123"
}
```

Response:
```json
{
  "success": true,
  "new_hash": "def456",
  "path": "pages/post/hello-world.md",
  "preview_url": "/hello-world/"
}
```

Error response (conflict):
```json
{
  "success": false,
  "error": "conflict",
  "message": "File was modified externally"
}
```

## Frontmatter Handling

### Parsing

1. Extract frontmatter block between `---` delimiters
2. Parse as YAML for validation
3. Report YAML syntax errors to user

### Formatting

On save, frontmatter is re-serialized with consistent formatting:

- **Key ordering**: Alphabetically sorted for consistency
- **List style**: Use block style (`- item`) for lists > 2 items
- **String quoting**: Quote only when necessary (special chars, colons)
- **No JSON**: Never dump JSON inside YAML structures
- **Indentation**: 2 spaces
- **Line endings**: Preserve original (LF or CRLF) if possible, default to LF

Example input (messy):
```yaml
title: Hello
tags: [one, two]
published: true
date: 2024-01-15
```

Example output (formatted):
```yaml
date: 2024-01-15
published: true
tags:
  - one
  - two
title: Hello
```

### Validation

- Validate YAML parses successfully
- Validate known fields have correct types
- Pass through unknown fields (stored in `Extra`)
- Report validation errors with line numbers

## Preview Pipeline

1. User saves post
2. Server writes Markdown file
3. Dev server notices the changed file and starts a rebuild
4. If file watching is disabled, admin explicitly triggers the rebuild
5. Server returns new hash immediately after the source file is saved
6. Client polls build status
7. On success, preview iframe shows `/slug/` from output

Rebuild uses the existing lifecycle. In fast mode, only rebuild changed posts.

### Preview vs Live Build

- **POC behavior**: Preview uses the real built site after save. There is no unsaved draft preview in v1.
- **Reason**: This keeps one rendering path and one source of truth while editing is being stabilized.
- **Post saves**: Rebuild should be incremental in fast/watch mode when only content changes.
- **Settings saves**: Rebuild may become full because config changes can affect templates, assets, feeds, and global state.
- **Future work**: A draft-only preview namespace may be added later, but it MUST not replace the real-build preview flow.

### Live Preview

- The editor MAY also provide a draft-only live preview while the user types
- Draft preview does not write files or run the full build lifecycle
- Draft preview is advisory only; the built preview remains the source of truth for final output
- The UI SHOULD make the difference between live draft preview and built preview obvious

## Settings Editor API

### Get Settings

```
GET /__admin/api/settings
```

Response:
```json
{
  "path": "markata-go.toml",
  "content": "[markata-go]\ntitle = \"My Site\"\n",
  "base_hash": "abc123",
  "exists": true
}
```

### Save Settings

```
PUT /__admin/api/settings
```

Request:
```json
{
  "content": "[markata-go]\ntitle = \"My Site\"\n",
  "base_hash": "abc123"
}
```

Response:
```json
{
  "success": true,
  "new_hash": "def456"
}
```

### Settings Editor Scope

- V1 edits the active config file as raw text
- V1 does not provide structured field-by-field controls yet
- Save uses conflict detection based on `base_hash`
- Save triggers the same dev rebuild flow used for watched file changes
- Validation errors come from the normal config reload/build path

## Git Integration

### Status

```
GET /__admin/api/git/status?path=pages/post/hello-world.md
```

Response:
```json
{
  "status": "modified",
  "staged": false,
  "tracked": true
}
```

### Diff

```
GET /__admin/api/git/diff?path=pages/post/hello-world.md
```

Response:
```json
{
  "diff": "--- a/pages/post/hello-world.md\n+++ b/pages/post/hello-world.md\n@@ -1,4 +1,4 @@\n..."
}
```

### Stage

```
POST /__admin/api/git/stage
{
  "path": "pages/post/hello-world.md"
}
```

### Commit

```
POST /__admin/api/git/commit
{
  "message": "Update hello-world post",
  "files": ["pages/post/hello-world.md"]
}
```

### Push (optional v1)

```
POST /__admin/api/git/push
{
  "remote": "origin",
  "branch": "main"
}
```

## Admin UI

### Pages

1. **Login**: Username/password form
2. **Setup**: First-run username/password creation
3. **Dashboard**: List of posts with search/filter
4. **Editor**: Edit post frontmatter + body, preview pane, save button
5. **Settings**: Edit the active config file and rebuild the site
6. **Git Panel**: Status, diff, stage, commit, push controls

### Editor Layout

```
+------------------------------------------+
|  [Save] [Preview] [Git]    [Logout]      |
+------------------------------------------+
|  Frontmatter  |  Body                   |
|  [textarea]   |  [textarea]             |
|               |                          |
|               |                          |
+---------------+--------------------------+
|  Preview                              |
|  [iframe /slug/]                       |
+------------------------------------------+
```

### Keybindings

- `Ctrl+S` / `Cmd+S`: Save
- `Ctrl+P` / `Cmd+P`: Toggle preview
- `Escape`: Close modals

## Theme Integration

- Admin UI SHOULD derive its colors from the active site palette when possible
- Admin UI SHOULD honor the site's light/dark palette pairing and fallback mode
- Admin UI MAY use its own layout, but it MUST feel like part of the same site rather than a separate product
- Admin UI SHOULD prefer local assets and generated CSS variables over unrelated third-party defaults

## Configuration

### Config Options

```toml
[admin]
# Enable admin (default: true when auth secrets present)
enabled = true

# Secret directory (default: ".markata-secrets")
secrets_dir = ".markata-secrets"

# Allow remote admin over plain HTTP (default: false)
allow_insecure_remote = false

# Session expiry (default: 24h)
session_expiry = "24h"
```

### Environment Variables

- `MARKATA_ADMIN_SECRETS_DIR` - override secret directory
- `MARKATA_ADMIN_ALLOW_INSECURE` - allow plain HTTP

## Security Considerations

- All `/__admin` routes require authentication
- CSRF tokens on all mutating requests
- Path traversal prevention
- Rate limiting on auth endpoints
- Secure cookie flags in production
- Content Security Policy for admin pages

## Implementation Notes

- Refactor `serve` to use `http.ServeMux` instead of single handler
- Create `pkg/serveadmin` package for admin logic
- Create `pkg/contentedit` for file editing with YAML formatting
- Reuse existing build services for preview
- Use HTMX for progressive enhancement
