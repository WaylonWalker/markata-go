# Font packs

Markata uses `fontpack` to select typography. The default is `system`, a
zero-download pack that emits semantic `--font-*` variables but no font
binary or `@font-face` rule. System stacks are read from
`markata-fontpacks.yaml`.

Bundled packs use prebuilt WOFF2 files and stable catalog tiers. Builds copy
only referenced tiers into `output/assets/fonts` and write one shared
`output/css/fonts.css`; page content never creates a new subset. The optional
FontTools workflow is intentionally separate from ordinary builds.
