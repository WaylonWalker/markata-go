# Theme Creation

Use this topic when the task is colors, palettes, styling, or overall site visual identity.

## Easiest Theme Customization Path

1. pick or change a palette
2. override a few CSS variables
3. add `custom_css`
4. only then override templates if structure must change

## Decision Guide

Use the smallest layer that can solve the task:

- change the overall color system across the site: create or edit a palette
- tweak a handful of design tokens: use `[markata-go.theme.variables]`
- restyle specific selectors or components: use `custom_css`
- change structure, markup, or layout composition: override templates

If the task is "make links more vivid" or "soften surfaces site-wide", a palette or theme variables change is usually the right answer.

If the task is "change card spacing" or "move the header layout", CSS or templates are usually better.

## Core Config Pattern

```toml
[markata-go.theme]
palette = "catppuccin-mocha"
custom_css = "custom.css"

[markata-go.theme.variables]
"--color-primary" = "#8b5cf6"
"--content-width" = "800px"
```

## Progressive Customization Order

Follow this order unless the task explicitly requires a deeper redesign:

1. choose a palette
2. override a few theme variables
3. add `custom_css`
4. adjust layout config
5. override specific templates
6. create a full custom theme only if needed

## Inspect First

- `palettes/`
- `templates/`
- site CSS under `static/` or theme asset directories
- theme settings in config

For exact palette file structure, read `../reference/palette-reference.md`.

## Helpful Commands

- `markata-go palette list`
- `markata-go palette info <name>`
- `markata-go palette check <name>` (WCAG AA contrast validation)
- `markata-go palette check <name> --strict` (WCAG AAA)
- `markata-go palette check --all` (check all palettes at once)
- `markata-go palette check --all --json` (machine-readable output)
- `markata-go palette preview <name>`
- `markata-go palette new <name>`
- `markata-go palette clone <source>`
- `markata-go theme render-all`
- `markata-go theme gallery`
- `markata-go theme check-all` (checks 16 contrast combos per palette)
- `markata-go theme check-all --colorblindness` (simulate color vision deficiencies)
- `markata-go aesthetic list`
- `markata-go aesthetic show <name>`

## Palette Validation

Always run `palette check` after creating or modifying a palette:

```bash
markata-go palette check my-brand
markata-go palette check my-brand --strict   # AAA level
```

`palette check` validates WCAG contrast ratios between key color pairs (text on background, links on background, etc.). Without `--strict` it checks WCAG AA (4.5:1 for normal text). With `--strict` it checks AAA (7:1).

`theme check-all` tests 16 contrast combinations across all palettes and can simulate color vision deficiencies (`--colorblindness`) to catch accessibility issues beyond normal contrast.

## Font Configuration

Fonts are configured under the theme namespace:

```toml
[markata-go.theme.fonts]
heading = "Inter"
body = "Source Sans Pro"
code = "JetBrains Mono"
```

In templates, use the `google_fonts_url` filter to generate a Google Fonts import URL from the configured font names.

## Aesthetics

Aesthetics control structural design tokens like border radius, spacing, border styles, and shadow effects. They are separate from color palettes.

Built-in aesthetics: `brutal`, `precision`, `balanced`, `elevated`, `minimal`.

```bash
markata-go aesthetic list
markata-go aesthetic show balanced
```

Configure in TOML:

```toml
[markata-go.aesthetics]
style = "balanced"
```

## Guidance

- Prefer palette and CSS changes before replacing whole templates.
- Preserve the site's current typography and layout language unless the task is a redesign.
- When introducing a new palette, validate contrast with `palette check` instead of only eyeballing aesthetics.
- Keep theme work incremental: palette, then CSS, then template overrides if needed.
- For first sites, a palette plus `custom_css` is usually enough.
- Run `palette check --strict` for sites that need AAA accessibility compliance.
- When tuning view transitions, avoid broad `view-transition-name` rules on whole articles, sidebars, or global chrome. Prefer small shared elements and use `view_transitions.debug = true` to inspect fetch, parse, swap, and reinit timing.

## Creating A Site-Local Palette

Project-local palettes live in `palettes/` and can be selected by name:

```toml
[markata-go.theme]
palette = "my-brand"
```

A starter file is included at `../examples/palettes/my-brand.toml`.

## Theme Variables Vs Palette Files

Use a palette file when you want semantic control such as:

- `accent`
- `text-primary`
- `bg-primary`
- `link`
- `success`
- `error`

Use `theme.variables` when you want to override generated CSS variables directly for one site without maintaining a custom palette file.

## Light And Dark Pairing

If the site wants explicit light and dark palettes:

```toml
[markata-go.theme]
palette = "catppuccin-latte"
palette_dark = "catppuccin-mocha"
fallback_mode = "dark"
```

`fallback_mode` controls what happens when the user's system preference is unknown or the browser does not support `prefers-color-scheme`:

- `"dark"`: use the dark palette as fallback
- `"light"`: use the light palette as fallback
- `"system"`: respect the OS/browser preference (no explicit fallback)

## First-Site Recommendation

For a brand-new site, a good default recommendation is:

```toml
[markata-go.theme]
palette = "catppuccin-mocha"
custom_css = "custom.css"
```

Then place `custom.css` under `static/` and make small incremental changes there.
