# Markata Image Index and Library Specification

The image library is a build-time inventory of images and referenced videos
that a public Markata site can use. It provides an authoring page at `/images/`,
a stable JSON artifact at `/images/index.json`, and a root-level copy at
`/images.json` for tools that discover site data without knowing the page path.
Markdown and frontmatter remain the source of truth; the index contains derived
metadata only.

## Scope

The image library MUST:

- discover image and video files in the configured static asset directory;
- discover images in supported Markdown image syntax through the Goldmark AST;
- discover images emitted by supported raw HTML image elements through an HTML
  parser, not Markdown regular expressions;
- discover referenced HTML video elements and video sources without treating
  them as image elements;
- discover media-valued post frontmatter, including `image`, `cover`,
  `cover_image`, `video`, `og_image`, `social_image`, `thumbnail`,
  `featured_image`, `hero_image`, `avatar`, and `author_image`;
- exclude skipped, draft, private, and unpublished posts from public usage
  relationships;
- omit a local static media file when its only discovered post references are from
  private posts;
- deduplicate an image within a post while retaining whether any usage is a
  cover usage;
- preserve the canonical source URL in the index and in copy actions;
- write the same canonical JSON bytes to `path/index.json` and
  `output_dir/images.json` when JSON export is enabled;
- identify video sources by their MIME type or supported video extension and
  retain a poster source when one is available;
- mark images emitted inside external embed cards with `embed: true`;
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
      "added_at": "2024-03-10T00:00:00Z",
      "last_used_at": "2026-01-15T12:00:00Z",
      "cover": true,
      "uses": [
        {
          "post": "posts/build.md",
          "href": "/build/",
          "title": "Build notes",
          "caption": "The finished build",
          "cover": true
        }
      ]
    }
  ]
}
```

`src`, `width`, `height`, `alt`, `mime_type`, `cover`, and `uses` are stable
fields. A `uses[].caption` value contains the normalized visible text of the
`<figcaption>` associated with that media use, when one exists. Captions belong
to uses rather than the canonical media record because the same source may have
different captions in different posts. `poster_src` is the canonical poster
source for a video, when one is available. `last_used_at` is the latest
publication date of a public post that uses the source. It is omitted when no
public use has a publication date.
`embed` identifies an image emitted by an external embed card;
`uses[].embed` identifies the corresponding post relationship. Unknown fields
MUST be ignored by readers. Optional media and relationship fields
(`poster_src`, `last_used_at`, `embed`, `uses[].embed`, and `uses[].caption`) are
omitted when empty or false. A dimension of `0` means that the source dimensions
are not available locally. `added_at` is the earliest valid publication date of
any public post that uses or references the source, including remote sources. It
is omitted when no public use has a valid publication date. Filesystem
modification times are used only as an incremental-cache invalidation signal;
they MUST NOT be emitted as public metadata.

Images are sorted by `src`. Uses are sorted by post path, then href. Consumers
that need recency MUST sort by `last_used_at` descending, place missing values
last, and use `src` ascending as the deterministic tie-breaker. JSON is encoded
with fixed struct field order and no insignificant whitespace.

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

## Local dimensions

The collector MUST inspect dimensions without fetching remote media. It MUST
provide dimensions for PNG, JPEG, GIF, WebP, BMP, and TIFF when the local file
contains valid decoder metadata. SVG dimensions MAY be read from positive
`width` and `height` values or from a positive `viewBox`; CSS percentages and
malformed values are treated as unknown. ICO dimensions MAY use the largest
directory entry, with a zero directory byte meaning 256 pixels. AVIF and
HEIC/HEIF dimensions are intentionally unknown unless a future implementation
adds a vetted decoder; their MIME types and inventory records remain valid.
Unsupported or malformed local media MUST retain zero dimensions rather than
causing the inventory build to fail.

## Frontmatter and Markdown

The effective cover convention is `cover` first, then `cover_image`, then
`image`, then `video`. If multiple fields are present, all values may be
inventoried, but only the first non-empty cover field is marked as the cover.
This makes the common `image` frontmatter field a cover fallback. Other
supported media frontmatter is inventoried as a non-cover relationship.

Body images MUST be extracted from a Goldmark AST using the site's supported
image syntax. Inline attributes and figure captions MUST not prevent image
discovery. When a Markdown image is inside a figure with a caption, the
normalized caption text MUST be retained on its usage relationship. A shared
figure caption MUST be retained for every image in that figure. Raw HTML
`<img>` elements MAY be used as a fallback for content that Goldmark represents
as raw HTML. Referenced `<video>` and `<source>` elements MUST be inventoried as
video media, not image media. A media element inside a `<figure>` MUST retain
the normalized visible text of its associated `<figcaption>`, including for
video sources. Obsidian attachment embeds are discovered after `EmbedsPlugin`
has transformed them into standard Markdown.

External embed cards MUST expose `data-markata-embed="true"` on their generated
HTML so the library can label their OG images as embeds without fetching remote
pages. An image used by both an embed and ordinary content remains one
canonical record and is marked as an embed if any public use is an embed.

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
is written to `path/index.html`; when `export_json` is enabled, identical JSON
artifacts are written to `path/index.json` and `output_dir/images.json`.
`template` defaults to `images.html`. Relative paths MUST stay inside
`output_dir`.

## Authoring page

The default page MUST:

- show a dense, image-first responsive grid;
- use at least two columns on ordinary mobile widths when the viewport allows;
- reserve media aspect-ratio space before images load;
- use lazy loading and `decoding="async"` for non-critical thumbnails;
- provide a sticky, mobile-friendly toolbar;
- provide client-side search over filename, path, alt text, figure captions,
  and post titles;
- provide `All`, `Used`, `Unused`, `Cover`, and `Recently added` views;
- provide deterministic sorting controls, including `Latest used`;
- show up to three usage links inline and a compact `+N more` disclosure;
- show available figure captions with their usage links;
- render video sources with a `<video>` element, a `<source>` element, and a
  poster when available;
- defer video source URLs until the card enters the viewport while keeping an
  available poster visible, so an inventory page does not fetch every video
  during initial load;
- label external embed images and video sources with an `Embed` badge;
- provide Copy Markdown and Copy URL actions using the canonical `src`;
- provide visible success and failure feedback for copy actions;
- expose labels, focus states, and accessible names for all controls; and
- show a useful empty state when no images are available.

The `Recently added` view means a source with a positive `added_at` Unix
timestamp no older than 30 days from the browser's current time. A missing
timestamp is not recent. This client-side window is intentionally separate
from the canonical inventory format.

The page MUST remain useful if JavaScript fails: images, titles, and usage
links still render in the server-generated HTML. JavaScript only enhances
filtering, sorting, usage expansion, copy feedback, and video loading. The
bundled video cards MUST keep their media source URL in a data attribute until
an `IntersectionObserver` reports that the card is visible. An available
poster MAY remain in the `poster` attribute so it can serve as the card's
thumbnail. When that API is unavailable, the page MUST load a video only after
pointer, focus, or touch interaction. The card link MUST still expose the
canonical media URL without JavaScript.

## Incremental builds

The image index writer MUST compute an aggregate input hash from image-related
configuration, eligible post input hashes and publication dates, and the
configured asset image and video files. When the hash matches the cached value
and all enabled generated outputs exist, it MUST skip reparsing posts and
rewriting the artifacts. A changed, added, removed, or renamed media file, or
a changed public post publication date, MUST invalidate the aggregate output.

Media hashing MUST use one filesystem traversal when the content and asset
roots overlap. The cache MUST retain each media file's normalized path, size,
and modification time alongside its content fingerprint. It SHOULD retain an
operating-system change-time signal when the filesystem exposes one. An
unchanged cheap state MUST reuse the cached content fingerprint without opening
the media file; when the state changes, the file MUST be read and its fingerprint
refreshed. On filesystems without a change-time signal, the cache assumes that
size and modification time identify unchanged content. The cache state is an
optimization only and MUST NOT affect generated metadata.

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
Platform-managed symlink ancestors above the output root MAY be present; the
writer MUST still reject a symlink at the output root or below it.
