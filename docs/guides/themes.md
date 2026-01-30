---
title: "Themes and Styling"
description: "Complete guide to customizing your site's appearance with themes, color palettes, and CSS"
date: 2024-01-15
published: true
slug: /docs/guides/themes/
tags:
  - documentation
  - themes
  - styling
  - customization
---

# Themes and Styling

markata-go makes it easy to customize your site's appearance. You can go from zero configuration to a beautiful site, then progressively customize as needed.

> **Prerequisites:** This guide assumes you've completed the [Getting Started](/docs/getting-started/) guide and have a working markata-go site.

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

## Multi-Palette Theme Switcher

Allow visitors to choose any available color palette at runtime through an interactive UI in the header. The switcher provides:

- **Dark/Light toggle**: A sun/moon button to instantly switch between dark and light mode
- **Palette family selector**: A dropdown to choose from palette families (Catppuccin, Gruvbox, Rose Pine, etc.)
- **Smart variant selection**: Automatically picks the appropriate light/dark variant based on current mode
- **Keyboard shortcuts**: `]` next family, `[` previous family, `\` toggle dark/light mode
- **Toast notifications**: Visual feedback when cycling through palettes

### Enabling the Switcher

```toml
[markata-go.theme]
palette = "rose-pine"  # Default palette

[markata-go.theme.switcher]
enabled = true
include_all = true  # Include all 70+ built-in palettes
```

When enabled, the switcher UI appears in the site header. Visitor selections are persisted in localStorage and restored on return visits.

### Keyboard Shortcuts

The palette switcher includes convenient keyboard shortcuts:

| Key | Action |
|-----|--------|
| `]` | Switch to next palette family |
| `[` | Switch to previous palette family |
| `\` | Toggle dark/light mode |

These shortcuts work anywhere on the page and show a toast notification with the new palette name.

### Filtering Palettes

By default, all discovered palettes are included. You can control which palettes appear:

**Exclude specific palettes:**

```toml
[markata-go.theme.switcher]
enabled = true
include_all = true  # Default
exclude = ["default-light", "default-dark"]  # Hide these palettes
```

**Include only specific palettes:**

```toml
[markata-go.theme.switcher]
enabled = true
include_all = false
include = ["catppuccin-mocha", "catppuccin-latte", "nord-dark", "nord-light"]
```

### Switcher Configuration Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Show the palette switcher UI |
| `include_all` | boolean | `true` | Include all discovered palettes |
| `include` | array | `[]` | Palettes to include (when `include_all` is false) |
| `exclude` | array | `[]` | Palettes to exclude (when `include_all` is true) |
| `position` | string | `"header"` | Where to place the switcher |

### How It Works

1. **Palette Manifest**: When the switcher is enabled, markata-go generates a JSON manifest of all available palettes embedded in `variables.css` as a CSS custom property (`--palette-manifest`).

2. **CSS Variables**: Each palette's colors are generated as CSS custom properties using `[data-palette="palette-name"]` selectors. When a user selects a palette, a data attribute is set on the `<html>` element.

3. **JavaScript UI**: The `palette-switcher.js` script (loaded conditionally when enabled):
   - Reads the palette manifest from CSS
   - Groups palettes into "families" (e.g., all Catppuccin variants)
   - Creates the sun/moon toggle and family dropdown
   - Handles keyboard shortcuts and persistence

4. **Persistence**: Selected palette family and dark/light preference are saved to `localStorage` and restored on page load.

5. **Smart Variants**: When you select a family like "Catppuccin", the switcher automatically chooses `catppuccin-latte` in light mode and `catppuccin-mocha` in dark mode.

### Styling the Switcher

The switcher UI uses CSS custom properties for easy customization:

```css
/* In your custom CSS */
:root {
  /* Toast notifications */
  --toast-bg: var(--color-surface);
  --toast-text: var(--color-text);
  --toast-border: var(--color-border);

  /* Mode toggle button */
  --toggle-size: 2rem;
  --toggle-bg: var(--color-surface);
  --toggle-hover-bg: var(--color-surface-hover);

  /* Family dropdown */
  --dropdown-bg: var(--color-surface);
  --dropdown-border: var(--color-border);
}
```

The switcher container can also be styled:

```css
.palette-switcher {
  /* Container styles */
  gap: 0.5rem;
}

.palette-family-select {
  /* Dropdown styles */
  min-width: 120px;
}

.mode-toggle {
  /* Sun/moon button styles */
  border-radius: 50%;
}
```

### Hide on Mobile

To hide the family dropdown on mobile (keeping only the dark/light toggle):

```css
@media (max-width: 768px) {
  .palette-family-select {
    display: none;
  }
}
```

### JavaScript API

The switcher exposes a JavaScript API for programmatic control:

```javascript
// Get the current palette info
const current = PaletteSwitcher.getCurrentPalette();
// { family: "catppuccin", variant: "mocha", isDark: true }

// Set a specific palette
PaletteSwitcher.setPalette("rose-pine-moon");

// Toggle dark/light mode
PaletteSwitcher.toggleDarkMode();

// Get current dark mode state
const isDark = PaletteSwitcher.isDarkMode();

// Get all palette families
const families = PaletteSwitcher.getFamilies();
// ["catppuccin", "gruvbox", "rose-pine", ...]
```

### Event Handling

Listen for palette and mode changes:

```javascript
// Palette family changed
window.addEventListener('palette-family-change', (e) => {
  console.log('Family:', e.detail.family);
  console.log('Palette:', e.detail.palette);
});

// Dark/light mode toggled
window.addEventListener('dark-mode-change', (e) => {
  console.log('Dark mode:', e.detail.isDark);
});
```

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

## CSS Optimization

markata-go automatically optimizes CSS loading by only including stylesheets that are actually needed for each page. This reduces page size and improves load times.

### How It Works

When rendering a page, markata-go scans the HTML content and detects which CSS features are used:

| CSS File | Loaded When |
|----------|-------------|
| `variables.css` | Always (core theme variables) |
| `main.css` | Always (core layout styles) |
| `components.css` | Always (navigation, footer, etc.) |
| `cards.css` | Feed/index pages with post cards |
| `admonitions.css` | Posts containing admonition blocks |
| `code.css` | Posts containing code blocks |
| `chroma.css` | Posts with syntax-highlighted code |
| `webmentions.css` | When webmentions are enabled |
| `palette-switcher.css` | When palette switcher is enabled |
| `search.css` | When search is enabled |

### Content Detection

The CSS detection works by analyzing the rendered HTML:

- **Admonitions**: Detected when `class="admonition` is present
- **Code blocks**: Detected when syntax highlighting classes (`class="chroma"`, `class="highlight"`) or code elements (`<pre><code`, `<code class="language-`) are present
- **Cards**: Included on feed pages (where `feed` variable exists in template context)

### Benefits

- **Smaller page sizes**: Simple pages without code blocks or admonitions skip those CSS files
- **Faster load times**: Less CSS to download and parse
- **Better caching**: Core CSS files are shared across all pages

### Custom CSS

Your custom CSS (via `theme.custom_css`) is always loaded when configured. If you need conditional loading for custom styles, consider using CSS custom properties or JavaScript-based loading.

---

## Media Borders and Gradient Effects

markata-go provides beautiful, configurable borders for images and videos. From subtle solid borders to animated gradients, you have full control over how your media looks.

### Default Media Styling

By default, images and videos in your content get:

- Rounded corners (`--media-border-radius`)
- A subtle border (`--media-border-width`, `--media-border-color`)
- Proper spacing and centering

### Configuring Media Borders

Customize the default borders via CSS variables:

```css
/* In your custom CSS or via theme.variables */
:root {
  --media-border-width: 3px;      /* Border thickness */
  --media-border-color: #e5e7eb;  /* Border color */
  --media-border-radius: 0.5rem;  /* Corner rounding */
}
```

Or via config:

```toml
[markata-go.theme.variables]
"--media-border-width" = "4px"
"--media-border-color" = "#8b5cf6"
"--media-border-radius" = "1rem"
```

### Gradient Borders

Enable colorful gradient borders for a modern, eye-catching look. Add a class to your post content to enable gradients for all media:

```yaml
---
title: "My Post with Gradient Borders"
css_class: gradient-borders
---
```

This applies the default accent gradient to all images and videos in that post.

### Available Gradient Presets

markata-go includes several beautiful gradient presets:

| Class | Colors | Best For |
|-------|--------|----------|
| `gradient-borders` | Primary to primary-dark | Brand-consistent |
| `gradient-vibrant` | Purple to pink | Creative, artistic |
| `gradient-warm` | Pink to orange | Energetic, warm |
| `gradient-cool` | Blue to cyan | Professional, tech |
| `gradient-sunset` | Pink to yellow | Warm, inviting |
| `gradient-ocean` | Teal to light blue | Calm, refreshing |

Use them in frontmatter:

```yaml
---
title: "Ocean-Themed Post"
css_class: gradient-ocean
---
```

### Animated Gradient Borders

For extra visual impact, use animated gradients that slowly shift colors:

```yaml
---
title: "Attention-Grabbing Post"
css_class: gradient-animated
---
```

The animation cycles through purple, pink, and blue over 6 seconds.

### Glow Effects

Add a subtle glow behind your media:

```yaml
---
title: "Glowing Media"
css_class: glow
---
```

Combine glow with gradients:

```yaml
---
title: "Maximum Impact"
css_class: gradient-vibrant glow
---
```

### Per-Image Styling

For fine-grained control, add classes directly to images in your Markdown using HTML:

```html
<img src="/images/hero.jpg" alt="Hero" class="gradient-vibrant glow">
```

Or use a wrapper div:

```html
<div class="media-frame gradient-sunset glow">
  <img src="/images/featured.jpg" alt="Featured">
</div>
```

### CSS Variable Reference for Media

| Variable | Description | Default |
|----------|-------------|---------|
| `--media-border-width` | Border thickness | `3px` |
| `--media-border-style` | Border style | `solid` |
| `--media-border-color` | Border color | `var(--color-border)` |
| `--media-border-radius` | Corner rounding | `0.5rem` |
| `--gradient-accent` | Default gradient | Primary colors |
| `--gradient-vibrant` | Purple-pink gradient | `#667eea` to `#f093fb` |
| `--gradient-warm` | Pink-orange gradient | `#f093fb` to `#f8b500` |
| `--gradient-cool` | Blue-cyan gradient | `#4facfe` to `#00f2fe` |
| `--gradient-sunset` | Pink-yellow gradient | `#fa709a` to `#fee140` |
| `--gradient-ocean` | Teal-blue gradient | `#2193b0` to `#6dd5ed` |

### Palette-Matching Gradients

If you're using a specific color palette, use the matching gradient for visual consistency:

| Class | Palette | Colors |
|-------|---------|--------|
| `gradient-catppuccin` | Catppuccin | Mauve → Pink → Red |
| `gradient-nord` | Nord | Frost colors (cyan → blue) |
| `gradient-dracula` | Dracula | Purple → Pink → Cyan |
| `gradient-gruvbox` | Gruvbox | Yellow → Orange → Red |
| `gradient-rose-pine` | Rosé Pine | Iris → Rose → Gold |
| `gradient-solarized` | Solarized | Blue → Cyan → Green |
| `gradient-tokyo-night` | Tokyo Night | Blue → Purple → Pink |

Example: If your site uses `catppuccin-mocha` palette, use `gradient-catppuccin` for borders:

```toml
# markata-go.toml
[markata-go.theme]
palette = "catppuccin-mocha"
```

```yaml
# In your post frontmatter
---
title: "Catppuccin-Styled Gallery"
css_class: gradient-catppuccin
---
```

### Custom Gradients

Create your own gradient by overriding the variables:

```css
/* In static/custom.css */
:root {
  --gradient-accent: linear-gradient(135deg, #ff6b6b, #feca57, #48dbfb);
}
```

Or define a completely new one:

```css
.post-content.gradient-custom img,
.post-content.gradient-custom video {
  border: none;
  padding: 3px;
  background: linear-gradient(45deg, #12c2e9, #c471ed, #f64f59);
  background-origin: border-box;
}
```

Then use in frontmatter:

```yaml
---
css_class: gradient-custom
---
```

### Dark Mode Considerations

Gradient borders adapt to dark mode:
- Glow effects become more prominent
- Border colors adjust automatically
- Gradients remain vibrant on dark backgrounds

Test your gradient choices in both light and dark mode.

---

## Background Decorations

Add multi-layered background decorations to your site for visual effects like snow, particles, stars, or animated elements.

### Basic Configuration

```toml
[markata-go.theme.background]
enabled = true

backgrounds = [
  { html = '<snow-fall count="200"></snow-fall>' },
]

scripts = ["/static/js/snow-fall.js"]
```

This adds a snow effect using a custom web component with its supporting JavaScript.

### Multiple Layers

Stack multiple background layers with different z-index values:

```toml
[markata-go.theme.background]
enabled = true

backgrounds = [
  { html = '<div class="stars"></div>', z_index = -20 },
  { html = '<div class="clouds"></div>', z_index = -10 },
  { html = '<snow-fall count="100"></snow-fall>', z_index = -5 },
]

scripts = ["/static/js/background-effects.js"]

css = '''
.stars {
  position: absolute;
  inset: 0;
  background: url("/images/stars.png") repeat;
  opacity: 0.3;
}

.clouds {
  position: absolute;
  inset: 0;
  background: url("/images/clouds.png") repeat-x;
  animation: drift 60s linear infinite;
}

@keyframes drift {
  from { background-position: 0 0; }
  to { background-position: 100% 0; }
}
'''
```

### Configuration Reference

| Option | Type | Description |
|--------|------|-------------|
| `enabled` | boolean | Enable/disable background decorations (default: false) |
| `backgrounds` | array | List of background elements |
| `backgrounds[].html` | string | HTML content for this layer |
| `backgrounds[].z_index` | integer | Stacking order (-1 is default, behind content) |
| `scripts` | array | Script URLs to load for background functionality |
| `css` | string | Custom CSS for styling background elements |

### Tips for Background Decorations

1. **Performance**: Complex animations can impact performance. Test on lower-powered devices.

2. **Accessibility**: Ensure backgrounds don't interfere with content readability. Use `pointer-events: none` (applied automatically).

3. **Z-Index**: Use negative values to place backgrounds behind content. Positive values overlay content.

4. **Web Components**: Custom elements like `<snow-fall>` provide encapsulated, reusable effects.

5. **Reduced Motion**: Consider respecting `prefers-reduced-motion` in your CSS:

```css
@media (prefers-reduced-motion: reduce) {
  .background-layer * {
    animation: none !important;
  }
}
```

### Example: Particle Background

Using [particles.js](https://vincentgarreau.com/particles.js/):

```toml
[markata-go.theme.background]
enabled = true

backgrounds = [
  { html = '<div id="particles-js"></div>' },
]

scripts = [
  "https://cdn.jsdelivr.net/particles.js/2.0.0/particles.min.js",
  "/static/js/particles-config.js",
]

css = '''
#particles-js {
  position: absolute;
  inset: 0;
}
'''
```

---

## Font Configuration

markata-go provides flexible font configuration to customize your site's typography without writing CSS.

### Quick Start

Add custom fonts via Google Fonts:

```toml
[markata-go.theme.font]
google_fonts = ["Inter", "Fira Code"]
family = "'Inter', sans-serif"
code_family = "'Fira Code', monospace"
```

### Font Options

| Option | Description | Default |
|--------|-------------|---------|
| `family` | Body text font | System fonts |
| `heading_family` | Heading font (inherits from `family` if not set) | Same as `family` |
| `code_family` | Code/monospace font | System monospace |
| `size` | Base font size | `16px` |
| `line_height` | Base line height | `1.6` |
| `google_fonts` | Array of Google Fonts to load | `[]` |
| `custom_urls` | Array of custom font CSS URLs | `[]` |

### Using Google Fonts

Specify fonts to load from Google Fonts:

```toml
[markata-go.theme.font]
# Load these fonts from Google Fonts
google_fonts = ["Inter", "Playfair Display", "JetBrains Mono"]

# Use them in your font families
family = "'Inter', sans-serif"
heading_family = "'Playfair Display', serif"
code_family = "'JetBrains Mono', monospace"
```

The `google_fonts` array automatically generates the Google Fonts CSS URL with weights 400, 500, 600, and 700.

### Using Custom Fonts

Load fonts from any URL:

```toml
[markata-go.theme.font]
custom_urls = [
  "https://fonts.example.com/my-font.css",
  "/fonts/local-font.css"
]
family = "'My Custom Font', sans-serif"
```

### Typography Variables

Font configuration generates CSS custom properties that you can use in your custom CSS:

| Variable | Description |
|----------|-------------|
| `--font-family` | Body text font stack |
| `--font-heading` | Heading font stack |
| `--font-code` | Code/monospace font stack |
| `--font-size` | Base font size |
| `--line-height` | Base line height |

### Complete Example

```toml
[markata-go.theme]
palette = "catppuccin-mocha"

[markata-go.theme.font]
# Google Fonts to load
google_fonts = ["Source Sans Pro", "Source Serif Pro", "Source Code Pro"]

# Font assignments
family = "'Source Sans Pro', sans-serif"
heading_family = "'Source Serif Pro', serif"
code_family = "'Source Code Pro', monospace"

# Typography settings
size = "18px"
line_height = "1.7"
```

### Using Self-Hosted Fonts

For better performance and privacy, you can self-host fonts:

1. Download font files to `static/fonts/`
2. Create a CSS file defining `@font-face` rules
3. Reference it in `custom_urls`

```css
/* static/fonts/fonts.css */
@font-face {
  font-family: 'MyFont';
  src: url('/fonts/MyFont-Regular.woff2') format('woff2');
  font-weight: 400;
}

@font-face {
  font-family: 'MyFont';
  src: url('/fonts/MyFont-Bold.woff2') format('woff2');
  font-weight: 700;
}
```

```toml
[markata-go.theme.font]
custom_urls = ["/fonts/fonts.css"]
family = "'MyFont', sans-serif"
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

---

## Next Steps

Now that you've styled your site, here are recommended next steps:

**Customize your templates:**
- [Templates Guide](/docs/guides/templates/) - Modify HTML structure, add custom partials, and use template inheritance

**Organize your content:**
- [Feeds Guide](/docs/guides/feeds/) - Create filtered collections, archives, and tag pages

**Deploy your site:**
- [Deployment Guide](/docs/guides/deployment/) - Deploy to GitHub Pages, Netlify, Vercel, or self-host

---

## See Also

- [Configuration Guide](/docs/guides/configuration/) - All configuration options
- [Templates Guide](/docs/guides/templates/) - Template syntax and customization
- [Frontmatter Guide](/docs/guides/frontmatter/) - Post-level configuration
- [Quick Reference](/docs/guides/quick-reference/) - Theme snippets and CLI commands
