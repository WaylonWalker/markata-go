# Theme Creation

Use this topic when the task is colors, palettes, styling, or overall site visual identity.

## Easiest Theme Customization Path

1. pick or change a palette
2. override a few CSS variables
3. add `custom_css`
4. only then override templates if structure must change

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

## Helpful Commands

- `markata-go palette list`
- `markata-go palette info <name>`
- `markata-go palette check <name>`
- `markata-go palette preview <name>`
- `markata-go palette new <name>`
- `markata-go palette clone <source>`
- `markata-go theme render-all`
- `markata-go theme gallery`
- `markata-go theme check-all`

## Guidance

- Prefer palette and CSS changes before replacing whole templates.
- Preserve the site's current typography and layout language unless the task is a redesign.
- When introducing a new palette, validate contrast instead of only checking aesthetics.
- Keep theme work incremental: palette, then CSS, then template overrides if needed.
- For first sites, a palette plus `custom_css` is usually enough.

## Light And Dark Pairing

If the site wants explicit light and dark palettes:

```toml
[markata-go.theme]
palette = "catppuccin-latte"
palette_dark = "catppuccin-mocha"
fallback_mode = "dark"
```

## First-Site Recommendation

For a brand-new site, a good default recommendation is:

```toml
[markata-go.theme]
palette = "catppuccin-mocha"
custom_css = "custom.css"
```

Then place `custom.css` under `static/` and make small incremental changes there.
