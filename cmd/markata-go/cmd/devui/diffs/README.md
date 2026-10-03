# Serve diff review bundle

This directory owns the browser-only source used to build Markata's local source-fix diff renderer.

The Control Center must remain usable offline and must not load JavaScript, CSS, fonts, grammars, or workers from a CDN at runtime. `@pierre/diffs` is therefore pinned here and built into local assets before they are embedded by the Go binary.

## Build

```sh
cd cmd/markata-go/cmd/devui/diffs
bun install --frozen-lockfile
bun run build
```

The build writes the embedded assets to `pkg/servecontrol/pierre-diffs/`.
It includes the entry point, Markdown grammar, Pierre light and dark themes,
Shiki runtime, and their static imports. The resulting set is about 1.2 MB
across eight JavaScript files, plus license notices. A full Shiki bundle would add roughly 10 MB of unused
languages and themes.

Markata's fix surface edits Markdown source, so the build script selects only
that dependency closure. Commit the generated files when changing the adapter
or dependency. Release builds do not require Bun, npm access, or runtime
internet access.

`pierre-review.js` deliberately accepts only the `path`, `before`, and `after` values returned by the existing fix-preview API. Pierre Diffs is a presentation layer; it does not select edits, authorize writes, or bypass Markata's preview/apply safety checks.

The fix review dialog creates one host per previewed file and calls
`renderPierreDiff` with that file's frozen before/after snapshot. Cleanup runs
when the dialog closes or is re-rendered. The grouped apply request is
unchanged.
