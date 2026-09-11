# Content Diagnostics and Disposition Specification

This specification defines the canonical ledger for explaining what happens to
content during a Markata Go build.

## Goals

The ledger MUST make the path from discovery to output observable without
requiring raw build logs. It MUST:

- count discovered files and content candidates;
- record load, frontmatter, post, eligibility, render, feed, and output state;
- assign stable reason codes to content that is not eligible or is not emitted;
- retain one deterministic final disposition for every discovered content
  candidate; and
- be safe to consume from human and machine-readable reporting surfaces; and
- publish a safe, versioned diagnostics artifact after a successful full build.

The ledger is the in-memory build result. The normal full build also persists a
sanitized production artifact at `.markata/diagnostics.json` below the resolved
output directory.

## Ownership and Data Flow

`lifecycle.Manager` owns one `diagnostics.ContentLedger` for each build. The
ledger is reset with the manager and is populated at these boundaries:

```text
glob       -> discovered files and content candidates
load       -> bytes loaded, frontmatter state, diagnostics, posts
collect    -> feed selection and filter reasons
render     -> rendered posts and render errors
write      -> emitted posts and output errors
cleanup    -> successful-build diagnostics artifact
```

Plugins MUST update this ledger rather than create another content inventory.
Consumers MUST use the immutable, sorted snapshot returned by the ledger.

Synthetic posts used for feed and page lookup have no source path and MUST NOT
be counted as content files or source posts.

## Candidate and Count Definitions

The snapshot contains one entry for every file returned by glob. Entries whose
extension is not a supported content extension are marked `not_candidate` and
are excluded from content-stage counts. Markdown and supported text content
extensions are candidates.

For candidate entries, the summary fields mean:

| Field | Definition |
| --- | --- |
| `discovered` | Every file returned by glob, including non-candidates. |
| `candidates` | Discovered files with a supported content extension. |
| `loaded` | Candidates whose source bytes were read successfully, including cache-backed posts whose source was inspected. |
| `frontmatter_valid` | Candidates with a structurally valid frontmatter block whose YAML parsed successfully. Empty frontmatter counts as valid. No frontmatter does not count. |
| `posts` | Candidates converted into a source-backed `Post`. |
| `eligible` | Posts with `published: true` that are not drafts or skipped. This is public content eligibility; private feed opt-in is recorded separately. |
| `rendered` | Posts for which Markdown HTML was produced or restored. |
| `emitted` | Posts for which at least one post or feed output was produced or confirmed by the write stage. |
| `excluded` | Candidates that are not eligible or have no emitted output. An unpublished shadow page can therefore be both emitted and excluded from public eligibility. |
| `warnings` | Warning-severity content diagnostics. |
| `errors` | Error-severity content diagnostics. Recoverable authoring errors do not hide valid sibling files; operational source errors fail the build. |

Counts are derived from ledger entries. They MUST NOT be assembled by adding
plugin-local counters.

## Per-File Entry

Each entry records these booleans:

- `candidate`
- `loaded`
- `frontmatter_present`
- `frontmatter_valid`
- `post_created`
- `eligible`
- `rendered`
- `emitted`
- `excluded`

It also records:

- `path`, relative to the content root when discovered by the standard glob;
- `disposition`, one of `emitted`, `shadow`, `excluded`, or `not_candidate`;
- sorted, unique `reasons`;
- frontmatter and lifecycle diagnostics; and
- feed-level dispositions with the feed name, inclusion state, and reasons.

Disposition rules are deterministic:

1. non-candidates are `not_candidate`;
2. emitted eligible posts are `emitted`;
3. emitted ineligible posts are `shadow`; and
4. all other candidates are `excluded`.

Entries, feed dispositions, reasons, and diagnostics MUST be sorted before a
snapshot is exposed. Concurrent plugin execution MUST NOT change snapshot
ordering or duplicate a diagnostic.

## Reason Codes

Reason codes are stable identifiers. Human-readable messages may change without
changing the code.

### Frontmatter

| Code | Meaning |
| --- | --- |
| `frontmatter.suspicious_delimiter` | A frontmatter-like block starts with a delimiter such as `----` or `-----` instead of `---`. |
| `frontmatter.leading_whitespace` | A frontmatter-like opening delimiter is preceded by whitespace. |
| `frontmatter.malformed_closing_delimiter` | A frontmatter block ends with a delimiter other than `---`. |
| `frontmatter.missing_closing_delimiter` | An exact `---` opening has no closing delimiter. |
| `frontmatter.parse_error` | YAML or frontmatter metadata could not be parsed. |
| `frontmatter.duplicate_key` | YAML contains a duplicate key. |
| `frontmatter.invalid_type` | A common metadata field has an unsupported scalar or collection type. |
| `frontmatter.invalid_date` | A date field cannot be parsed using Markata's supported date formats. |

### Content and output

| Code | Meaning |
| --- | --- |
| `content.published_false` | The post is not eligible for public published feeds. It may still produce a shadow page. |
| `content.draft` | The post is a draft and is not emitted. |
| `content.skip` | The post is explicitly skipped. |
| `content.private` | The post is outside a public feed unless that feed opts into private content. |
| `content.filtered` | A configured feed filter did not select the post. |
| `content.duplicate_slug` | A slug conflict prevents a unique output destination. |
| `content.no_output` | No post or feed output was available for the candidate. |
| `content.load_error` | Source bytes or source-backed post state could not be loaded. |
| `content.render_error` | Markdown rendering failed for the post. |
| `content.write_error` | An output write failed. |
| `feed.offset` | The post matched the feed's privacy and filter selection, but was removed by the configured offset. |
| `feed.limit` | The post matched the feed's privacy and filter selection and offset window, but was removed by the configured limit. |

The implementation MAY add codes in a namespace, but existing codes MUST retain
their meaning.

## Frontmatter Detection

The parser recognizes frontmatter only when the first line, after an optional
UTF-8 BOM, is exactly `---`. The closing delimiter is also a line containing
exactly `---`. Line endings are normalized for parsing.

The following rules apply:

1. A BOM is accepted and does not produce a diagnostic.
2. A first line of `----`, `-----`, or another longer hyphen run produces a
   warning only when a following frontmatter-like block has an exact closing
   delimiter and YAML-shaped fields. Ordinary Markdown horizontal rules do not
   produce a warning.
3. Leading spaces or tabs before an otherwise valid opening delimiter produce a
   warning only when the surrounding block is frontmatter-like.
4. A malformed or missing closing delimiter produces a diagnostic with the
   source line and expected delimiter.
5. Invalid YAML, duplicate keys, invalid dates, and common metadata type errors
   are recorded with source positions when available.
6. Markdown that has no frontmatter remains valid and produces no frontmatter
   warning merely because YAML-shaped text appears later in the body.

Malformed frontmatter, invalid YAML, and invalid frontmatter metadata are
source-local authoring errors. They do not cause a valid sibling file to
disappear. The loader records the affected candidate and continues the content
lifecycle. Direct parser APIs continue to return parsing errors to their
callers.

Operational source errors are different. Stat failures, read failures,
permission errors, unexpected missing sources, and other filesystem or source
I/O errors are recorded as `content.load_error` and propagated from the load
stage. They MUST fail the build instead of producing a successful partial
build.

## Feed and Output Semantics

Feed selection is recorded per feed. A post can be emitted as a direct shadow
page while being excluded from a public feed because `published` is false,
because it is private, because its feed filter does not match, or because it
falls outside the feed's offset or limit window. `content.filtered` is used
only for filter rejection; `feed.offset` and `feed.limit` identify posts that
matched the feed before windowing. Feed-level reasons do not change the
source's single global final disposition.

The absence of an automatically generated tag feed is explained by the ledger
through the metadata and feed state that led to it. A tag that was lost because
frontmatter was not recognized does not create a tag feed; the source entry
still contains the frontmatter diagnostic and the resulting publication
eligibility reason.

## Console Reporting

The normal `build` summary MUST include a `Content:` section with the summary
counts. Content warnings identify the relative source path, line when known,
reason code, and concise message.

The existing global `-v, --verbose` flag is the explain mode for builds. It MUST
show every excluded or shadow entry and its sorted reason codes. It MAY also
show feed-level exclusions. The mode MUST NOT print raw configuration,
environment values, secrets, absolute source paths, or raw build logs.

The existing `--benchmark-json` machine-readable build output includes the
sanitized content snapshot. No separate diagnostics command or public HTML page
is added.

## Production Diagnostics Artifact

The normal full build MUST write the diagnostics artifact to:

```text
<output_dir>/.markata/diagnostics.json
```

The artifact is enabled by default and does not require a configuration entry.
It is part of the generated output, so Builder Admin carries it into each
successful release automatically.

### Versioned wire format

The artifact uses this top-level shape:

```json
{
  "$schema": "markata://schemas/content-diagnostics/v1",
  "schema": "markata.content-diagnostics",
  "schema_version": 1,
  "generator": {
    "name": "markata-go",
    "version": "0.0.0",
    "commit": "..."
  },
  "source": {
    "commit": "..."
  },
  "built_at": "2026-09-10T12:00:00Z",
  "summary": {},
  "entries": []
}
```

`schema`, `schema_version`, `generator`, `built_at`, `summary`, and `entries`
are required. `generator.commit` is omitted when the binary does not provide a
reliable build commit. `source` is omitted when the source directory is not a
Git checkout or its `HEAD` cannot be read. `built_at` is an RFC 3339 timestamp
and is recorded in UTC.

`summary` and `entries` are the manager's `ContentLedgerSnapshot` copied
without recomputing counts or making a second content inventory. Entries,
reasons, feed names, and diagnostics retain the deterministic ordering defined
above. Consumers MUST dispatch on `schema` and `schema_version`; future
versions MUST NOT be interpreted as version 1.

### Privacy and failure behavior

The artifact MUST contain only the sanitized ledger fields. It MUST NOT contain
raw configuration, environment values, secrets, absolute paths, raw logs,
frontmatter, Markdown bodies, rendered HTML, decrypted private content, or
encryption key names. Persisted diagnostic messages MUST be concise canonical
messages for approved diagnostic codes; arbitrary producer-provided message
text MUST NOT be copied into the artifact.

The artifact is written after all normal write and cleanup hooks complete. The
writer MUST use a temporary file in the artifact directory and replace the
destination only after the complete JSON document has been written. A failed
build MUST leave an existing artifact unchanged; a first failed build MUST NOT
leave a partial artifact. A run that completes all stages with non-critical
lifecycle warnings may publish the artifact; an artifact write failure is a
build failure.

Dry runs, fast or incremental builds, partial lifecycle calls that stop before
cleanup, and development server requests that do not complete a full build MUST
NOT publish a new artifact.
