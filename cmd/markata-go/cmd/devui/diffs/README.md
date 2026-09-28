# Serve diff review bundle

This directory owns the browser-only source used to build Markata's local source-fix diff renderer.

The Control Center must remain usable offline and must not load JavaScript, CSS, fonts, grammars, or workers from a CDN at runtime. `@pierre/diffs` is therefore pinned here and built into local assets before they are embedded by the Go binary.

## Build

```sh
cd cmd/markata-go/cmd/devui/diffs
bun install --frozen-lockfile
bun run build
```

The build writes split browser assets under `../pierre-diffs/`. Profiling showed that a single minified bundle is about 10.75 MB because Shiki's complete language/theme graph is bundled. Code splitting reduces the initial `pierre-review.js` entry to about 491 KB, but the full lazy asset graph is still about 10.78 MB.

Markata's fix surface edits Markdown source, so the integration should commit and embed only the reproducible dependency closure needed by the entry point, Markdown highlighting, and the Pierre light/dark themes. It must not ship every Shiki grammar merely because the upstream package can resolve them. Release builds should not require Bun, npm access, or runtime internet access.

`pierre-review.js` deliberately accepts only the `path`, `before`, and `after` values returned by the existing fix-preview API. Pierre Diffs is a presentation layer; it does not select edits, authorize writes, or bypass Markata's preview/apply safety checks.

## Integration target

The fix review dialog should create one host per previewed file and call `renderPierreDiff` with that file's frozen before/after snapshot. Cleanup runs when the dialog closes or is re-rendered. The existing grouped apply request remains unchanged.
