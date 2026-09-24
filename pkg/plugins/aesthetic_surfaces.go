package plugins

// aestheticSurfaceCSS makes each aesthetic visibly distinct. The default theme
// components only read radius tokens, so the shape, border, and depth of the
// shared reading surfaces are set here per aesthetic. Tokens are keyed on any
// [data-aesthetic] element so theme picker previews can scope them to a card.
// Minimal is the theme's native look: it defines tokens for previews but its
// surfaces are not restyled. Surface rules sit in the utilities layer so they
// beat component rules while unlayered site CSS still wins.
const aestheticSurfaceCSS = `
@layer tokens {
  [data-aesthetic="minimal"] { --radius: 4px;
    --surface-border: 1px solid var(--color-border); --surface-shadow: none; }
  [data-aesthetic="balanced"] { --radius: 8px; --radius-sm: 6px; --radius-md: 8px; --radius-lg: 12px; --radius-xl: 16px;
    --surface-border: 1px solid var(--color-border); --surface-shadow: 0 1px 2px rgb(0 0 0 / .10), 0 6px 16px -8px rgb(0 0 0 / .28); }
  [data-aesthetic="elevated"] { --radius: 12px; --radius-sm: 8px; --radius-md: 12px; --radius-lg: 20px; --radius-xl: 28px;
    --surface-border: 1px solid transparent; --surface-shadow: 0 2px 6px rgb(0 0 0 / .14), 0 18px 40px -16px rgb(0 0 0 / .45); }
  [data-aesthetic="precision"] { --radius: 2px; --radius-sm: 1px; --radius-md: 2px; --radius-lg: 3px; --radius-xl: 4px;
    --surface-border: 1px solid color-mix(in srgb, var(--color-text) 30%, var(--color-background)); --surface-shadow: none; }
  [data-aesthetic="brutal"] { --radius: 0px; --radius-sm: 0px; --radius-md: 0px; --radius-lg: 0px; --radius-xl: 0px;
    --surface-border: 2px solid var(--color-text); --surface-shadow: 4px 4px 0 var(--color-text); }
}
@layer utilities {
  html[data-aesthetic]:is([data-aesthetic="balanced"], [data-aesthetic="elevated"], [data-aesthetic="precision"], [data-aesthetic="brutal"]) :is(pre, .card, .admonition, .embed-card, .blogroll-card, .post-content table, .post-content img) {
    border: var(--surface-border);
    box-shadow: var(--surface-shadow);
  }
  html[data-aesthetic="elevated"] :is(pre, .card, .admonition) { background: color-mix(in srgb, var(--color-surface) 85%, var(--color-text) 4%); }
  html[data-aesthetic="brutal"] blockquote { border-left-width: 6px; border-left-color: var(--color-text); border-radius: 0; }
  html[data-aesthetic="brutal"] :is(button, input, select):not(.theme-card) { border-width: 2px; border-color: var(--color-text); border-radius: 0; }
  html[data-aesthetic="brutal"] .post-content :is(h1, h2, h3)::after { height: 4px; }
  html[data-aesthetic="brutal"] code:not(pre code) { border: 1px solid var(--color-text); }
  html[data-aesthetic="precision"] blockquote { border-radius: 0; }
}
`
