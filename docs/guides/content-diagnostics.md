---
title: "Content Diagnostics"
description: "Inspect the content diagnostics artifact produced by a markata-go build"
date: 2026-09-10
published: true
tags:
  - documentation
  - content
  - diagnostics
---

# Content Diagnostics

After a successful full build completes, markata-go writes a
sanitized diagnostics artifact below the configured output directory:

```text
<output_dir>/.markata/diagnostics.json
```

For example, a build with `-o dist` writes `dist/.markata/diagnostics.json`.
The artifact is enabled by default. It is part of the generated output, so a
Builder Admin release from such a build carries it with the rest of the site.

## Inspect the artifact

Use a JSON viewer such as `jq`:

```bash
jq . public/.markata/diagnostics.json
jq '.summary' public/.markata/diagnostics.json
jq '.entries[] | select(.disposition == "excluded")' public/.markata/diagnostics.json
```

Replace `public` with the output directory used by the build. There is no
separate diagnostics command or generated public HTML page.

## Stable fields and reason codes

The top-level `schema`, `schema_version`, `generator`, `built_at`, `summary`,
and `entries` fields are stable parts of the versioned format. Consumers must
check both `schema` and `schema_version` before interpreting the document.

`summary` contains counts for discovered files, candidates, loaded sources,
valid frontmatter, posts, eligible posts, rendered posts, emitted outputs,
excluded candidates, warnings, and errors. Each entry includes a relative
source `path`, lifecycle state booleans, a final `disposition`, sorted `reasons`,
diagnostics, and feed-level dispositions.

Reason codes are stable identifiers. Messages can change without changing a
code. The current codes cover:

- frontmatter: `frontmatter.suspicious_delimiter`,
  `frontmatter.leading_whitespace`, `frontmatter.malformed_closing_delimiter`,
  `frontmatter.missing_closing_delimiter`, `frontmatter.parse_error`,
  `frontmatter.duplicate_key`, `frontmatter.invalid_type`, and
  `frontmatter.invalid_date`;
- content and output: `content.published_false`, `content.draft`,
  `content.skip`, `content.private`, `content.filtered`,
  `content.duplicate_slug`, `content.no_output`, `content.load_error`,
  `content.render_error`, and `content.write_error`;
- feed windowing: `feed.offset` and `feed.limit`.

The artifact uses schema `markata.content-diagnostics` and schema version `1`.

## Privacy and publication behavior

The artifact contains sanitized ledger data only. It does not contain raw
configuration, environment values, secrets, absolute paths, raw logs,
frontmatter, Markdown bodies, rendered HTML, decrypted private content, or
encryption key names.

The file is published only after the normal full lifecycle completes. A failed
build leaves an existing artifact unchanged and does not leave a partial new
file. `markata-go build --dry-run`,
`markata-go build --fast`, partial lifecycle calls, and fast or incremental
development server requests do not publish a new artifact. A normal `serve`
build may publish one when its full build completes successfully.
