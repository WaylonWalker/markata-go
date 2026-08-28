# Markata Content Index Specification

The Markata Content Index is an optional, compact, static JSON artifact of
derived metadata. It answers what Markata knew about content at build time.
Markdown files and their repository remain the source of truth.

## Scope

Markata MUST default to the newest supported generation when enabled. A
configuration MAY explicitly select an older supported generation for
compatibility. The artifact
MUST NOT contain complete article bodies, rendered HTML, source backups, or
Plaindown-specific data. Consumers MAY use it for metadata, search bootstrap,
dashboards, and integrations.

The canonical identity is `markata.content-index`. `schema_version` is an
integer wire-format generation. `generator.version` identifies the producer,
and `source.commit` identifies the source Git tree. These values MUST NOT be
used interchangeably. `$schema` is the internal canonical identifier for the
selected generation (`markata://schemas/content-index/v1` or
`markata://schemas/content-index/v2`); it is not a promise that a public URL is
currently hosted.

## Compatibility

Released generations are immutable. Additive optional fields MAY be added
within a generation. Renaming, removing, changing types or semantics, or
changing required relationships requires a new integer generation.

Markata MUST retain readers for every released generation forever. Readers
MUST dispatch by generation at the parser boundary and normalize into the
current internal model. Unknown fields MUST be ignored. Unsupported future
generations MUST produce a clear error and MUST NOT be interpreted as the
latest known generation.

## v1 Artifact

The required top-level fields are `$schema`, `schema`, `schema_version`,
`scope`, `generator`, `source`, `document_count`, and `documents`. `documents`
is a single array in v1; chunks and pagination are not implemented.

V1 emits `scope: "public"` and contains only non-skipped, non-draft,
non-private documents. Private documents MUST NOT be present in v1. Scope is an
open, non-empty string so a producer can define a workspace scope without
changing the wire type. Consumers MUST preserve unknown scope values at their
normalization boundary and MUST NOT treat them as public unless they understand
their definition.

Each document is identified by its repository-relative `path`. A consumer
MUST use `path` to map a record back to its source file. `slug` and `href` are
the resolved Markata URL identifiers. `title` is the authored/display value;
`title_text` is the derived semantic plain-text title when available. Dates
are RFC 3339 date-time strings and absent dates are omitted. `description` is
optional. Tags, aliases, and feeds are arrays of strings.

The following optional document fields MAY be emitted when present in resolved
frontmatter or author metadata: `image`, `video`, `avatar`, `thumbnail`,
`cover`, `og_image`, `author`, `authors`, `bio`, `category`, and `categories`.
Media fields are source references, not downloaded assets. `authors` and
`categories` are arrays of strings; the other fields are strings. Author order
is preserved because the first author can be semantically significant.
Consumers
MUST continue to ignore these fields when they do not need them.

Documents are sorted by normalized source path. Tags, aliases, and feed names
are sorted lexicographically. JSON object keys are emitted in a fixed Go
struct order. These rules make equivalent builds byte-deterministic.

Feed membership is stored on each document as `feeds`. It is the resolved
membership after Markata evaluates filters, sorting, limits, and offsets; a
consumer MUST NOT reconstruct Markata's feed language when this field is
present. Feed names are sorted. There is no duplicated top-level feed map.

`draft`, `published`, `private`, and `skip` are separate Markata concepts. A
document with `draft: true` is source/workspace content and is excluded from
the artifact. `published: false` does not alone make a document private: a
non-draft, non-private direct page MAY be included with no feed membership. A
feed named `draft` is only a feed name and is not equivalent to
`document.draft`; feed membership always reports Markata's resolved result.

## v2 Artifact

V2 retains the v1 document shape and adds the `public-metadata` scope. The
`public-metadata` scope permits private documents, but only within the safe
metadata boundary below. V2 is the default writer generation. V2 MAY use the
`public` scope only when no private document is present.

## Privacy and revision

The v2 public-metadata artifact includes every non-skipped, non-draft document,
including private documents. A non-private unpublished shadow page MAY be
present because Markata renders it as a direct page; its `published` value
remains false and it is normally in no feeds. `robots` is not an access-control
mechanism. The v1 public artifact retains its released exclusion of private
documents.

Private documents MUST contain no article body, rendered article HTML, source
backup, encryption key name, or other secret. Their identity and routing fields
(`path`, `slug`, `href`, `published`, `draft`, `private`, dates, and `template`)
may be emitted. Tags, aliases, author identifiers, categories, and a
frontmatter-provided avatar are metadata that may be emitted. `title` and
`title_text` may be emitted only when the title was explicitly provided in
frontmatter. `description` may be emitted only when it was explicitly provided
in frontmatter. Content-derived title and description values MUST be omitted.
Private media references (`image`, `video`, `thumbnail`, `cover`, and
`og_image`) and resolved author biographies MUST be omitted. The writer MUST
not copy arbitrary frontmatter or runtime `Extra` values into a private
document. The v2 marshal API MUST reject a private document that supplies any
of those forbidden fields rather than silently discarding them. The v2 schema
and parser apply the same presence-based rule, including when a forbidden field
has a `null` value. Optional text fields (`title`, `title_text`, and
`description`) MAY be omitted or `null`; both forms normalize to an absent
value.

Feed membership follows the resolved feed result. A private document MAY list a
feed only when that feed explicitly opts into private posts with
`include_private = true` (or its `private = true` compatibility alias). A
private document with no opted-in feed has no feed membership in the index.

`source.commit` identifies the Git `HEAD` observed for the build when it can be
read from the content directory. `source.dirty` is `false` when the working
tree was observed clean at the Content Index source-state boundaries used by
Markata. It is `true` when Git reports tracked modifications, staged changes,
deleted tracked files, untracked files, or ignored Markdown source files.
Untracked and ignored source files count because they can be visible to
Markata's content glob when Git-ignore filtering is disabled. If Git/source
state is unavailable, both fields are omitted; Markata MUST NOT fabricate
`dirty: false`.

When `dirty` is `true`, `commit` is only the base HEAD and the index includes
working-tree-derived metadata. When either value is unavailable, revision
equality is not a freshness proof. Markata captures source state before content
discovery and rechecks it before writing; if the snapshots differ, the enabled
index build fails rather than publishing a mismatched identity.

`commit` plus `dirty: false` is build provenance, not byte-level proof that every
derived document came from a blob in that Git revision. A source file can change
and be restored between the two state observations. Consumers that require
exact per-file synchronization SHOULD use Git commit/blob identity or another
exact source identity mechanism.

## Configuration and discovery

The output is disabled by default. Enable it with:

```toml
[markata-go.content_index]
enabled = true
output = "content-index.json" # relative to output_dir
```

An absolute `output` is allowed. `schema_version = 1` writes the compatibility
v1 public-only artifact; `schema_version = 2` writes the v2 public-metadata
artifact. The default output is v2 minified JSON at
`output_dir/content-index.json`. The configured destination is treated as owned
by this output. Deployments that disable the index SHOULD use a clean output
directory or explicitly remove the old artifact so stale metadata is not
published.

## Consumer behavior

Consumers SHOULD inspect `schema` and `schema_version` before decoding, ignore
unknown optional fields, and normalize each supported generation at their
boundary. They SHOULD fail clearly for an unsupported future generation and
offer migration rather than guessing. The index does not guarantee source
content, rendered output bytes, build success for every record, or access to
private content. Canonical fixtures in `pkg/contentindex/fixtures/` are
interoperability fixtures and MUST be treated as immutable after their
generation is released.
