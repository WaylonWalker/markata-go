# Aesthetics Specification

This document specifies **aesthetics**: non-color design token presets that control the site's "feel" (shape, rhythm, depth, and visual effects) without requiring custom CSS.

An aesthetic is selected independently from the color palette.

## Goals

- Provide bold, high-level levers (shadows, borders, glow, gradient frames, gradient headings, glass/noise overlays)
- Keep the system simple: token preset + optional overrides
- Avoid coupling to a specific theme implementation; tokens are emitted as CSS variables

---

## Token Categories

Aesthetic files are TOML documents with this structure:

```toml
name = "Neon Arcade"
description = "Electric glow, gradient frames, loud headings"

[tokens.radius]
sm = "6px"
md = "12px"
lg = "18px"
xl = "28px"

[tokens.spacing]
scale = 1.05

[tokens.border]
width_thin = "1px"
width_normal = "2px"
width_thick = "4px"
style = "solid"

[tokens.shadow]
sm = "0 1px 3px rgba(0,0,0,0.08)"
md = "0 10px 30px rgba(0,0,0,0.16)"

[tokens.typography]
font_primary = "var(--font-sans)"
leading_scale = 1.05

[tokens.effects]
glow_shadow = "0 0 22px color-mix(in srgb, var(--color-primary) 65%, transparent)"
frame_border_width = "2px"
frame_gradient = "linear-gradient(135deg, var(--color-primary), var(--color-info), var(--color-primary-dark))"
heading_gradient = "linear-gradient(90deg, var(--color-primary), var(--color-info), var(--color-text))"
heading_text_fill = "transparent"
noise_opacity = "0.06"
noise_image = "repeating-radial-gradient(circle at 10% 10%, rgba(255,255,255,0.08) 0 1px, transparent 1px 3px)"
surface_mix = "86%"
surface_blur = "14px"
```

### `tokens.effects` (Normative)

Effects tokens are emitted as CSS variables with the `--fx-` prefix.

Required naming:
- TOML keys use `snake_case`
- CSS variables use `kebab-case`

Example: `heading_text_fill` -> `--fx-heading-text-fill`.

Effects values MUST be valid CSS values for their intended property (e.g. gradients, lengths, opacities).

Recommended core effects tokens:

| Token | CSS Var | Intended Use |
|------|---------|--------------|
| `glow_shadow` | `--fx-glow-shadow` | Additional `box-shadow` layer to create neon glow |
| `frame_border_width` | `--fx-frame-border-width` | Gradient frame border width for major surfaces |
| `frame_gradient` | `--fx-frame-gradient` | Gradient used for the frame/border |
| `heading_gradient` | `--fx-heading-gradient` | Background gradient for heading text |
| `heading_text_fill` | `--fx-heading-text-fill` | `transparent` to reveal gradient via background-clip |
| `noise_image` | `--fx-noise-image` | Background-image string for a noise/texture overlay |
| `noise_opacity` | `--fx-noise-opacity` | Overlay opacity (0..1) |
| `surface_mix` | `--fx-surface-mix` | Percentage for `color-mix` to make surfaces translucent |
| `surface_blur` | `--fx-surface-blur` | Backdrop blur for glassy surfaces |

Themes MAY add additional effects tokens.

---

## CSS Output

The build MUST output aesthetic tokens to `css/aesthetic.css`:

- `:root` contains the configured aesthetic values
- Optional `[data-aesthetic="..."]` blocks are included for runtime switching

---

## Configuration

Users MUST be able to select an aesthetic in config:

```toml
[markata-go]
aesthetic = "balanced"
```

Users SHOULD be able to override tokens without editing CSS.

Overrides are specified as a free-form map and merged on top of the loaded aesthetic:

```toml
[markata-go]
aesthetic = "neon-arcade"

[markata-go.aesthetic_overrides]
shadow_size = "lg"
shadow_intensity = 1.4

[markata-go.aesthetic_overrides.effects]
frame_border_width = "3px"
heading_text_fill = "transparent"
```

Implementations MUST preserve backward compatibility for simple overrides and SHOULD allow nested category overrides.
