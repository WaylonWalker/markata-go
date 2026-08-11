# Theme Creation

Use this topic for palettes, colors, typography, texture, motifs, and CSS.

## Process

Follow these steps for theme work:

1. Discover the active rendering contract and its version.
2. Choose a canonical palette and, when needed, a palette family.
3. Make the smallest semantic configuration change.
4. Use CSS variables or `custom_css` as escape hatches.
5. Add palettes and presentation dials to the contract before adding projections.
6. Regenerate projections and run conformance checks.

The contract source is `spec/rendering-contract/contract-v1.json`. Read
`spec/rendering-contract/README.md` and `MIGRATIONS.md` before changing shared
IDs, defaults, bounds, or aliases. Do not create a second registry in a site
or consumer.

## Choose the Smallest Layer

- Choose a palette for a site-wide color system.
- Use `[markata-go.theme.variables]` for a few generated CSS variable overrides.
- Use `[markata-go.theme]` `custom_css` for selectors or components.
- Override templates only when the markup or layout must change.

Inspect the active config, `palettes/`, `templates/`, and site CSS before editing.
Preserve the site's existing layout and typography unless the task changes them.

## Canonical Theme Configuration

Keep theme-level keys in `[markata-go.theme]`. Put each nested group under its
own table. In TOML, a key remains in the current table until another table
header appears. Therefore, keep `custom_css` in the theme table, not in
`[markata-go.theme.motif]` or another nested table.

```toml
[markata-go.theme]
contract_version = 1
palette = "ayu-dark"
aesthetic = "minimal"
fontpack = "brush-poster"
custom_css = "custom.css"

[markata-go.theme.texture]
kind = "screenprint"
color_mix = 0.35
scale = 1.0
scope = "all"

[markata-go.theme.heading_texture]
kind = "inherit"
color_mix = 0.45
scale = 1.0

[markata-go.theme.motif]
kind = "block-w"
glyph = "W"
size = "78px"
gap = "10px"
row_offset = 0.24
wobble = 0.18
scatter = 0.0
layer = "sandwich"
color = "ink"
color_mix = 0.01
url = "https://waylonwalker.com/w.svg"

[markata-go.theme.variables]
"--content-width" = "68ch"
"--color-primary" = "#8b5cf6"
```

The contract uses normalized numeric dials. `color_mix`, `row_offset`,
`wobble`, and `scatter` use `0..1`. Texture scales use `0.25..3`. Use enum
values from the contract for palettes, aesthetics, textures, heading textures,
motifs, motif colors, motif layers, and scopes.

`heading_texture.kind = "inherit"` uses the surface texture kind while keeping
the heading texture's own `color_mix` and `scale`.

## CSS Escape Hatches

Put direct CSS variable overrides in `[markata-go.theme.variables]` when the
change is a token change. Put selector-level rules in the file named by
`[markata-go.theme]` `custom_css`:

```toml
[markata-go.theme]
custom_css = "custom.css"
```

Place that file at `static/custom.css`. Use templates only when CSS cannot
change the required structure.

## Palettes and Semantic Roles

Use `markata-go palette list` to discover available palette IDs. Use
`markata-go palette info <name>` to inspect a palette and
`markata-go palette check <name>` to check its contrast.

Use semantic palette roles such as `accent`, `background`, `ink`, and `surface`
instead of styling individual selectors. For a site-local palette, read
`../reference/palette-reference.md` for the `palettes/<name>.toml` shape.

When adding a shared palette, add its family, ID, variant, and roles to
`contract-v1.json` first. Add the light and dark members in the same change
when the family supports both variants. Regenerate all projections before
using the new ID.

```bash
go run ./scripts/rendering-contract
go run ./scripts/rendering-contract --check
```

The generator writes projections for the repository and its supported consumer
repositories. Run it from the markata-go repository root.

## Fonts

Select a `fontpack` from the contract. A fontpack assigns named roles instead
of requiring each selector to name a font family. The contract roles are
`body`, `heading`, and `mono`; use the roles supplied by the selected pack. The
`mono` role controls code and diagrams. Do not invent a fontpack ID or a role
that the catalog does not provide.

## Migration Rules

Canonical nested values take precedence over legacy flat values. The loader can
accept legacy values as migration inputs, but canonical serialization emits
only nested values. If both forms conflict, read the migration warning and
replace the legacy value.

The legacy `texture_scope = "headings"` setting needs special care. It means
headings-only behavior; it does not mean that `scope = "headings"` is a valid
canonical value. The migration moves the texture kind to
`theme.heading_texture.kind`, sets the surface texture to `none`, and uses the
quiet surface scope. Preserve this meaning when converting old configuration.

## Conformance Checks

Run the smallest relevant checks after editing:

```bash
markata-go palette check my-brand
markata-go palette check my-brand --strict
go run ./scripts/rendering-contract --check
go test ./pkg/config ./pkg/renderingcontract
```

Use `markata-go theme check-all` to check all discovered palettes. Add
`--colorblindness` for color-vision warnings. The command returns an error when
any palette fails its selected contrast checks.
