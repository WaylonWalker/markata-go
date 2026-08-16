---
title: "Content Index"
description: "Use Markata's stable metadata index for search and tooling."
date: 2026-08-15
published: true
tags:
  - documentation
  - content-index
---

# Content Index

Markata can write one compact JSON file with the metadata it resolved during a
build. It does not include article bodies.

```toml
[markata-go.content_index]
enabled = true
output = "content-index.json"
```

The file is written below `output_dir` unless `output` is absolute. Its
identity is `schema = "markata.content-index"` and its current generation is
`schema_version = 1`. V1 emits `scope = "public"`.

## Reading the file

Read `schema` and `schema_version` before decoding. Ignore fields you do not
know. If the generation is newer than the reader supports, stop with a clear
upgrade message. Do not guess how a future generation works.

Use each document's repository-relative `path` to find its source file. The
`feeds` array already contains resolved Markata feed membership. Consumers
should not implement Markata's feed filter language again.

`source.commit` identifies the Git `HEAD` observed for the build.
`source.dirty` is false when the working tree was observed clean at the Content
Index source-state boundaries. It is true when Git reports tracked, staged,
deleted, untracked, or ignored Markdown source files. Compare both values with
the source repository revision before using metadata as a bootstrap cache.
Both fields are absent when Git cannot provide source state.
Markata captures this state before content discovery and rechecks it before
writing. A changing source state fails an enabled index write instead of
publishing a mismatched identity.

The pair `source.commit` plus `dirty: false` is build provenance. It does not
prove that every document came byte-for-byte from a blob in that Git revision.
A source file can change and be restored between the two observations. Use Git
commit/blob identity, or another exact source identity mechanism, when exact
per-file synchronization is required.

The public scope excludes private, draft, and skipped documents. A
`published: false` direct page may still appear. A feed named `draft` is feed
membership only and does not mean that the document has `draft: true`.

Markata retains readers for every released generation and normalizes them into
one current internal model. The index is not a source archive, rendered-body
cache, access-control system, or guarantee that source files remain available.
Private and draft documents are excluded from the public artifact.
If you disable the output, use a clean output directory or remove the old
artifact before deployment.

The v1 schema and immutable interoperability fixtures are in
`pkg/contentindex/v1_schema.json` and `pkg/contentindex/fixtures/`.
