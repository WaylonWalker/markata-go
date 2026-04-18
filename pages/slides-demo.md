---
title: "Reveal Slides Demo"
slug: "slides-demo"
date: 2026-04-07
published: true
template: "slides.html"
description: "Demo deck for the reveal.js slides template in markata-go"
---

Intro text before the first slide heading becomes its own slide.

## markata-go Slides

Write slide decks as normal Markdown pages.

- `##` starts a horizontal slide
- `###` starts a vertical slide
- `---` forces a new horizontal slide

### Vertical Detail

This is a child slide nested under the current horizontal section.

---

## Asset Modes

Reveal.js follows the shared asset pipeline.

```toml
[markata-go.assets]
mode = "self-hosted"
```

Run `markata-go assets download` to prefetch vendor files, or just build and let markata-go download them automatically.

### Image Zoom Too

GLightbox now follows the same shared asset flow.

---

## Lint-Friendly Authoring

The deck uses `##` and `###`, so normal page H1 rules stay intact.

### End

Use arrow keys, space, or swipe gestures to move through the deck.
