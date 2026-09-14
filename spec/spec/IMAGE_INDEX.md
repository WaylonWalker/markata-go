# Markata Image Index and Library Specification

The image library is a build-time inventory of images that a public Markata
site can use. It provides an authoring page at `/images/` and a stable JSON
artifact at `/images/index.json`. Markdown and frontmatter remain the source
of truth; the index contains derived metadata only.

## Scope

The image library MUST:

- discover image files in the configured static asset directory;
- discover images in supported Markdown image syntax through the Goldmark AST;
- discover images emitted by supported raw HTML image elements through an HTML
  parser, not Markdown regular expressions;
- discover image-valued post frontmatter, including `image`, `cover`,
  `cover_image`, `og_image`, `social_image`, `thumbnail`, `featured_image`,
  `hero_image`, `avatar`, and `author_image`;
- exclude skipped, draft, private, and unpublished posts from public usage
  relationships;
- omit a local static image when its only discovered post references are from
  private posts;
- deduplicate an image within a post while retaining whether any usage is a
  cover usage;
- preserve the canonical source URL in the index and in copy actions;
- use trusted Dropper derivatives only for presentation URLs;
- avoid network requests while building the index; and
- produce deterministic output for the same source tree.

The library MUST NOT expose private post metadata or private media references.
An asset shared by a public post remains eligible for the public inventory;
an asset referenced only by private posts is omitted even when
`include_unreferenced` is enabled.
It MUST NOT add Plaindown-specific behavior.

## Canonical data model

The JSON artifact uses the following top-level shape:

```json
{
  "$schema": "markata://schemas/image-index/v1",
  "schema": "markata.image-index",
  "schema_version": 1,
  "generator": {"name": "markata-go", "version": "..."},
  "image_count": 1,
  "images": [
    {
      "src": "/images/keyboard.webp",
      "width": 1600,
      "height": 1000,
      "alt": "keyboard",
      "mime_type": "image/webp",
      "added_at": "2026-01-15T12:00:00Z",
      "cover": true,
      "uses": [
        {
          "post": "posts/build.md",
          "href": "/build/",
          "title": "Build notes",
          "cover": true
        }
      ]
    }
  ]
}
```

`src`, `width`, `height`, `alt`, `mime_type`, `cover`, and `uses` are stable
fields. Unknown fields MUST be ignored by readers. A dimension of `0` means
that the source dimensions are not available locally. `added_at` is emitted
for local files from their filesystem modification time and is omitted for
remote-only references.

Images are sorted by `src`. Uses are sorted by post path, then href. JSON is
encoded with fixed struct field order and no insignificant whitespace.

The `cover` field on an image is true when at least one use marks it as a
cover. A repeated body reference does not create a repeated use. If a post
uses the same image as both body content and a cover, its single use has
`cover: true`.

## Source and URL rules

Static files are mapped from the asset directory to site-root URLs:

```text
static/images/logo.png -> /images/logo.png
```

The library MUST use the same asset-root convention as the static asset
writer. A local reference that resolves to a scanned asset MUST share that
asset's canonical record, even when the authored URL uses a relative path or
the legacy `/static/` attachment prefix.

Remote URLs MUST remain untouched in `src`. Data URLs, blob URLs, and empty
sources are ignored. Trusted Dropper hosts are the exact hosts already listed
by `models.DefaultTrustedMediaDomains`:

- `dropper.wayl.one`
- `dropper.waylonwalker.com`
- `dropper-dev.wayl.one`

The existing `templates.WithSize` helper is the only approved presentation
derivative mechanism. It adds `w` and optional `h` query parameters, preserves
other query parameters, and normalizes trusted HTTP URLs to HTTPS. The
canonical `src` MUST NOT be replaced by a derivative URL.

## Frontmatter and Markdown

The effective cover convention is `cover` first, then `cover_image`. If both
are present, both images may be inventoried, but only the first non-empty value
is marked as the cover. Other supported image frontmatter is inventoried as a
non-cover relationship.

Body images MUST be extracted from a Goldmark AST using the site's supported
image syntax. Inline attributes and figure captions MUST not prevent image
discovery. Raw HTML `<img>` elements MAY be used as a fallback for content
that Goldmark represents as raw HTML. Obsidian attachment embeds are discovered
after `EmbedsPlugin` has transformed them into standard Markdown.

Alt text uses authored Markdown or frontmatter alt metadata when available.
When no authored alt text exists, the library derives a readable fallback from
the source filename.

## Configuration

The `images` plugin is enabled by default and can be configured under
`[markata-go.images]`:

```toml
[markata-go.images]
enabled = true
path = "images"
template = "images.html"
export_json = true
include_unreferenced = true
```

`path` is a relative output directory and defaults to `images`. The HTML page
is written to `path/index.html`; the JSON artifact is written to
`path/index.json`. `template` defaults to `images.html`. Relative paths MUST
stay inside `output_dir`.

## Authoring page

The default page MUST:

- show a dense, image-first responsive grid;
- use at least two columns on ordinary mobile widths when the viewport allows;
- reserve media aspect-ratio space before images load;
- use lazy loading and `decoding="async"` for non-critical thumbnails;
- provide a sticky, mobile-friendly toolbar;
- provide client-side search over filename, path, alt text, and post titles;
- provide `All`, `Used`, `Unused`, `Cover`, and `Recently added` views;
- provide deterministic sorting controls;
- show up to three usage links inline and a compact `+N more` disclosure;
- provide Copy Markdown and Copy URL actions using the canonical `src`;
- provide visible success and failure feedback for copy actions;
- expose labels, focus states, and accessible names for all controls; and
- show a useful empty state when no images are available.

The page MUST remain useful if JavaScript fails: images, titles, and usage
links still render in the server-generated HTML. JavaScript only enhances
filtering, sorting, usage expansion, and copy feedback.

## Incremental builds

The image index writer MUST compute an aggregate input hash from image-related
configuration, eligible post input hashes, and the configured asset image
files. When the hash matches the cached value and both generated outputs
exist, it MUST skip reparsing posts and rewriting the artifacts. A changed,
added, removed, or renamed image MUST invalidate the aggregate output.

The cache field is an optimization only. A missing output file MUST force a
rewrite even when the cached hash matches.

The writer MUST detect collisions with post, feed, and static-file outputs
before writing. It MUST NOT overwrite a file unless the cache records that
exact path as an image-library output. Cleanup after disabling the feature or
changing its path MUST remove only those recorded paths whose contents still
match the recorded output hash. A recorded path with modified contents, or an
existing unowned path, MUST be preserved and reported as a collision rather
than silently left behind. The output root and every existing component below
it MUST be checked for symlinks before the writer creates or removes files.
