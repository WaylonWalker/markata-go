# Font packs

Markata uses `fontpack` to select typography. The default is `system`, a
zero-download pack that emits semantic `--font-*` variables but no font
binary or `@font-face` rule. Built-in catalog data, manifests, licenses, and
font files are embedded in the Markata executable; they never require a
source checkout or a catalog in the site's working directory.

Bundled packs use prebuilt WOFF2 files and stable catalog tiers. Builds copy
only referenced tiers into `output/assets/fonts` and write one shared
`output/css/fonts.css`; page content never creates a new subset. The optional
FontTools workflow is intentionally separate from ordinary builds.

Custom catalogs may be selected with `fontpacks_file`. Relative paths in a
custom catalog resolve relative to the catalog file. Markata records generated
font filenames in `output/assets/fonts/.markata-fonts.json` and removes only
stale files listed by that manifest on later builds.

`fonts verify` checks manifest and lockfile provenance, license metadata, full
64-character SHA-256 hashes, and WOFF2 assets. Every bundled manifest must
record a non-empty `source.files` map, and its keys and hashes must exactly
match the lockfile. Short hashes are display-only. With no pack argument it
verifies every unique source used by every built-in bundled pack;
`fonts verify <pack>` limits the check to that pack and uses the configured
catalog when one is present. The built-in
`painted-sign` pack uses Finger Paint for display lettering, Rock Salt for
handwritten accents, Source Sans 3 for prose, and DM Mono for code.

The manifest generator reads every source from the catalog and its locked
metadata; it is not permitted to maintain a separate family list. Generated
tiers with different Unicode ranges must not have identical content. A family
that has no additional glyphs for a tier omits that tier rather than shipping a
duplicate asset.
