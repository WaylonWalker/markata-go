---
title: "Themes and Styling"
description: "Complete guide to customizing your site's appearance with themes, color palettes, and CSS"
date: 2024-01-15
published: true
tags:
  - documentation
  - themes
  - styling
  - customization
---

# Themes and Styling

markata-go makes it easy to customize your site's appearance. You can go from zero configuration to a beautiful site, then progressively customize as needed.

## Quick Start

The fastest way to change your site's look is to pick a color palette:

```toml
[markata-go.theme]
palette = "catppuccin-mocha"
```

That's it! Your entire site now uses the Catppuccin Mocha color scheme.

---

## Available Palettes

markata-go includes 10 built-in color palettes. Use `markata-go palette list` to see them all:

| Palette | Variant | Description |
|---------|---------|-------------|
| `default-light` | light | Clean, minimal light theme |
| `default-dark` | dark | Clean, minimal dark theme |
| `catppuccin-mocha` | dark | Soothing pastel colors, high contrast |
| `catppuccin-latte` | light | Soothing pastel colors, light variant |
| `nord-dark` | dark | Arctic, north-bluish colors |
| `gruvbox-dark` | dark | Retro groove, warm colors |
| `dracula` | dark | Vibrant purple on dark |
| `rose-pine` | dark | Soho vibes, natural colors |
| `solarized-dark` | dark | Scientifically designed colors |
| `tokyo-night` | dark | Tokyo city lights inspired |

### Previewing Palettes

Preview a palette before using it:

```bash
# Show palette colors and contrast info
markata-go palette info catppuccin-mocha

# Check WCAG accessibility compliance
markata-go palette check catppuccin-mocha
```

---

## Configuration Levels

markata-go supports progressive customization - start simple and add complexity only when needed.

### Level 1: Just Pick a Palette

```toml
[markata-go.theme]
palette = "dracula"
```

### Level 2: Override Specific Colors

Keep your palette but tweak a few colors:

```toml
[markata-go.theme]
palette = "nord-dark"

[markata-go.theme.variables]
"--color-primary" = "#88c0d0"
"--color-link" = "#8fbcbb"
```

### Level 3: Add Custom CSS

Add a custom CSS file that loads after the theme:

```toml
[markata-go.theme]
palette = "catppuccin-mocha"
custom_css = "my-styles.css"
```

Then create `static/my-styles.css`:

```css
/* Override any theme styles */
.post-title {
  font-family: 'Georgia', serif;
}

.site-header {
  border-bottom: 2px solid var(--color-primary);
}
```

### Level 4: Override Templates

Override specific template files by creating them in your `templates/` directory:

```
my-site/
├── templates/
│   └── partials/
│       └── footer.html    # Your custom footer
└── markata-go.toml
```

Your custom templates take precedence over theme templates.

---

## Theme Configuration Reference

Full configuration options:

```toml
[markata-go.theme]
# Theme name (currently only "default" is available)
name = "default"

# Color palette to use
palette = "catppuccin-mocha"

# CSS variable overrides
[markata-go.theme.variables]
"--color-primary" = "#8b5cf6"
"--color-background" = "#1a1a2e"
"--color-text" = "#eaeaea"
"--color-link" = "#06b6d4"
"--color-link-hover" = "#22d3ee"
"--content-width" = "800px"
"--font-family" = "'Inter', sans-serif"

# Custom CSS file (relative to static/ directory)
custom_css = "custom.css"
```

### Available CSS Variables

These CSS custom properties can be overridden:

| Variable | Description | Default |
|----------|-------------|---------|
| `--color-background` | Page background | Depends on palette |
| `--color-text` | Body text | Depends on palette |
| `--color-primary` | Primary accent color | Depends on palette |
| `--color-link` | Link color | Depends on palette |
| `--color-link-hover` | Link hover color | Depends on palette |
| `--color-border` | Border color | Depends on palette |
| `--color-code-bg` | Code block background | Depends on palette |
| `--content-width` | Max content width | `720px` |
| `--font-family` | Body font | System fonts |
| `--font-family-mono` | Code font | Monospace fonts |

---

## Dark Mode Support

markata-go supports automatic dark/light mode switching based on user's system preference.

### Using Different Palettes for Light/Dark

```toml
[markata-go.theme]
palette = "catppuccin-latte"        # Light mode
palette_dark = "catppuccin-mocha"   # Dark mode (prefers-color-scheme: dark)
```

The site automatically switches based on the visitor's system settings.

---

## Palette CLI Commands

### List All Palettes

```bash
markata-go palette list
```

Output:
```
NAME                      VARIANT  SOURCE     DESCRIPTION
----------------------------------------------------------------------
Catppuccin Mocha          dark     builtin    Soothing pastel theme
Catppuccin Latte          light    builtin    Soothing pastel theme (light)
Nord Dark                 dark     builtin    Arctic, north-bluish
...
```

### Get Palette Info

```bash
markata-go palette info catppuccin-mocha
```

Shows all colors in the palette with their hex values.

### Check Accessibility

```bash
markata-go palette check catppuccin-mocha
```

Checks WCAG 2.1 AA contrast requirements for text readability.

### Export Palette

Export a palette to different formats:

```bash
# Export as CSS custom properties
markata-go palette export catppuccin-mocha --format css

# Export as SCSS variables
markata-go palette export catppuccin-mocha --format scss

# Export as JSON
markata-go palette export catppuccin-mocha --format json

# Export as Tailwind config
markata-go palette export catppuccin-mocha --format tailwind
```

### Create New Palette

Generate a starter palette file:

```bash
markata-go palette new my-palette
```

Creates `palettes/my-palette.toml` that you can customize.

---

## Creating Custom Palettes

Create a custom palette by adding a TOML file to `palettes/` in your project:

```toml
# palettes/my-brand.toml
[palette]
name = "My Brand"
variant = "light"  # or "dark"
author = "Your Name"

# Raw colors
[palette.colors]
brand-primary = "#3b82f6"
brand-secondary = "#8b5cf6"
brand-accent = "#06b6d4"
white = "#ffffff"
gray-50 = "#f9fafb"
gray-100 = "#f3f4f6"
gray-700 = "#374151"
gray-800 = "#1f2937"
gray-900 = "#111827"

# Semantic mapping
[palette.semantic]
text-primary = "gray-800"
text-secondary = "gray-700"
bg-primary = "white"
bg-secondary = "gray-50"
accent = "brand-primary"
link = "brand-primary"
link-hover = "brand-secondary"
```

Then use it:

```toml
[markata-go.theme]
palette = "my-brand"
```

---

## Template Overrides

Override any template by placing it in your `templates/` directory.

### Template Search Order

1. `templates/` - Your project templates (highest priority)
2. `themes/{theme}/templates/` - Theme templates
3. Embedded default templates (fallback)

### Available Templates

| Template | Purpose |
|----------|---------|
| `base.html` | HTML skeleton, head, header, footer |
| `post.html` | Single post/article layout |
| `feed.html` | List of posts (index, archive, tags) |
| `card.html` | Post preview card in feeds |
| `partials/header.html` | Site header/navigation |
| `partials/footer.html` | Site footer |
| `partials/head.html` | Additional head content |

### Example: Custom Footer

Create `templates/partials/footer.html`:

```html
<footer class="site-footer">
  <div class="container">
    <p>&copy; {{ now().year }} {{ config.title }}. Built with markata-go.</p>
    <nav>
      <a href="/about/">About</a>
      <a href="/contact/">Contact</a>
      <a href="https://github.com/yourusername">GitHub</a>
    </nav>
  </div>
</footer>
```

---

## Static Assets

Add custom CSS, JavaScript, images, and fonts to the `static/` directory:

```
my-site/
├── static/
│   ├── css/
│   │   └── custom.css
│   ├── js/
│   │   └── analytics.js
│   ├── images/
│   │   └── logo.png
│   └── fonts/
│       └── MyFont.woff2
└── markata-go.toml
```

Files in `static/` are copied directly to the output directory.

Reference them in templates:

```html
<link rel="stylesheet" href="/css/custom.css">
<script src="/js/analytics.js"></script>
<img src="/images/logo.png" alt="Logo">
```

---

## Per-Post Styling

Override styles for specific posts using frontmatter:

```yaml
---
title: "Special Post"
template: landing.html  # Use a different template
---
```

Or add custom CSS classes:

```yaml
---
title: "Featured Article"
css_class: featured-post
---
```

Then style it:

```css
.featured-post {
  background: linear-gradient(to right, var(--color-primary), var(--color-accent));
}
```

---

## Best Practices

### 1. Start with a Palette

Don't write CSS from scratch. Pick the closest palette and customize from there.

### 2. Use CSS Variables

Override `--color-*` variables instead of hardcoding colors. This ensures consistency and makes future changes easier.

### 3. Keep Customizations Minimal

The less you customize, the easier upgrades will be. Only override what you need.

### 4. Test Dark Mode

If you customize colors, test both light and dark mode to ensure readability.

### 5. Check Accessibility

Use `markata-go palette check` to verify your color choices meet WCAG guidelines.

---

## Troubleshooting

### Styles Not Loading

1. Check that CSS files exist in `public/css/` after building
2. Verify your browser's network tab shows CSS loading
3. Clear browser cache and rebuild: `markata-go build`

### Custom CSS Not Applying

1. Ensure `custom_css` path is relative to `static/`
2. Check for CSS specificity issues (theme styles may override yours)
3. Use browser dev tools to inspect applied styles

### Template Not Found

1. Verify the template file exists in `templates/`
2. Check the filename matches exactly (case-sensitive)
3. Ensure frontmatter `template:` value includes `.html` extension

---

## See Also

- [Configuration Guide](./configuration.md) - All configuration options
- [Templates Guide](./templates.md) - Template syntax and customization
- [Frontmatter Guide](./frontmatter.md) - Post-level configuration
