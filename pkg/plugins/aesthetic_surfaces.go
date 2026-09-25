package plugins

// aestheticSurfaceCSS holds the surface rules shared by every aesthetic
// (borders and depth on code, cards, admonitions, embeds, tables and media)
// plus brutal's form, search and inline-code treatment. Radius and
// --surface-border/--surface-shadow tokens live in the theme's aesthetic.css,
// which is appended after this block; only brutal's ink/accent tokens are
// defined here. Rules sit in the utilities layer so they beat component rules
// while unlayered site CSS still wins.
const aestheticSurfaceCSS = `
@layer tokens {
  [data-aesthetic="brutal"] {
    --brutal-ink: color-mix(in srgb, var(--color-text) 82%, var(--color-background));
    --brutal-accent: var(--color-primary, var(--color-text));
  }
}
@layer utilities {
  html[data-aesthetic]:is([data-aesthetic="balanced"], [data-aesthetic="elevated"], [data-aesthetic="precision"], [data-aesthetic="brutal"]) :is(pre, .card, .admonition, .embed-card, .blogroll-card, .post-content table, .post-content img) {
    border: var(--surface-border);
    box-shadow: var(--surface-shadow);
  }
  html[data-aesthetic="brutal"] :is(.card, .embed-card, .blogroll-card):hover { box-shadow: 8px 8px 0 var(--brutal-accent); }
  html[data-aesthetic="brutal"] blockquote { border-left: 6px solid var(--brutal-accent); border-radius: 0; }
  html[data-aesthetic="brutal"] :is(button, input, select, textarea, .share-button, .post-copy__summary):not(.theme-card, .theme-picker-panel *) { border: 2px solid var(--brutal-ink); border-radius: 0; }
  html[data-aesthetic="brutal"] :is(button, .share-button, .post-copy__summary):not(.theme-card, .theme-picker-panel *):hover { box-shadow: 3px 3px 0 var(--brutal-accent); }
  html[data-aesthetic="brutal"], html[data-aesthetic="brutal"] .pagefind-ui { --pagefind-ui-border: var(--brutal-ink); --pagefind-ui-border-width: 2px; --pagefind-ui-border-radius: 0; }
  html[data-aesthetic="brutal"] .pagefind-ui__search-input { border: 2px solid var(--brutal-ink) !important; border-radius: 0 !important; }
  html[data-aesthetic="brutal"] .post-content :is(h1, h2, h3)::after { height: 4px; }
  html[data-aesthetic="brutal"] code:not(pre code) { border: 1px solid var(--brutal-ink); border-radius: 0; }
  html[data-aesthetic="precision"] blockquote { border-radius: 0; }
}
`
