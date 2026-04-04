---
title: "Admin CMS"
description: "Use the local admin UI to edit posts, preview rebuilds, and update site settings."
date: 2026-03-26
published: true
tags:
  - documentation
  - admin
  - cms
---

# Admin CMS

The admin UI is a local editing surface for `markata-go serve`. It lets you edit markdown posts, save source files, review the real built preview, and update the active config file.

## What It Does

- Edit frontmatter and markdown body for existing posts
- Create new posts from the admin editor
- Save source files directly to disk
- Preview the real built site after save and rebuild
- Edit the active config file from a settings page

## Start The Admin UI

Run the dev server:

```bash
go run ./cmd/markata-go serve
```

Then open:

```text
http://localhost:8000/__admin/
```

On first run, create an admin username and password. Credentials are stored in `.markata-secrets/` by default.

## Editing Posts

The editor has three parts:

- `Path` for the markdown file location
- `Frontmatter` for structured metadata
- `Body` for markdown content

The preview panel now supports two modes:

- `Live preview` renders your current draft as you type
- `Built preview` shows the saved page from the real site output

When you save:

1. The markdown file is written to disk
2. The dev server rebuilds the site
3. The preview iframe refreshes to the built page

This is a real-build preview, not a draft-only preview. That keeps the source file and the rendered site in sync.

The live preview is faster, but the built preview is the final check because it uses the actual site build.

## Editing Settings

Open `/__admin/settings` to edit the active config file.

The current POC uses a raw config editor. Save writes the file to disk, then the dev server rebuilds using the normal config reload path.

## Preview Model

The current preview model is intentionally simple:

- Preview updates after save
- Preview shows the real built output
- Unsaved draft preview is out of scope for now

This keeps one rendering path while the editing flow is stabilized.

## Current Limits

- Settings are edited as raw config text
- Preview reflects saved content only
- Remote admin hardening is not finished; use this locally during development
