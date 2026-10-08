---
title: "Adaptive Client Loading Design"
description: "Design for connection-aware media loading with a stable zero-JavaScript baseline."
date: 2026-10-07
published: false
tags:
  - performance
  - design
---

# Adaptive Client Loading

Issue: #1524

## Context

Markata-Go already emits native image and video elements, supports image
variants, and conditionally loads theme scripts. The generated sites also use
resource hints and video previews. The feature should improve mobile behavior
without turning every page into a JavaScript application or mutating content
that a reader has already seen.

Three approaches were considered:

1. **Static-only defaults.** This costs no runtime bytes, but cannot provide
   user-selected modes or learn from actual connection behavior.
2. **A small global controller.** This supports mode choice, conservative
   session learning, and future-resource decisions. It adds one small script
   request to each page.
3. **Separate controller per media type.** This keeps code conditional, but
   duplicates policy and makes settings inconsistent across components.

Use the second approach. One controller owns preference, confidence,
hysteresis, and the effective policy. Existing conditional loaders remain
responsible for feature-specific code.

## Behavior

The public modes are Auto, Save Data, and Full Quality. Auto is the default.
Explicit user choice always determines the effective policy and is stored in
local storage. A small session record may retain recent Auto confidence across
page navigations, with a short lifetime. Network Information API values are
optional hints; navigation/resource timing and online state work without them.

The controller requires repeated observations to change Auto state. Poor
signals enter constrained mode sooner than good signals restore full quality.
Manual/Auto mismatch prompts need stronger confidence than internal policy
changes. A dismissal remains quiet for the session. Manual Full Quality on a
confidently constrained connection may show one restrained prompt sooner.

The controller sets a stable data attribute and exposes the effective mode to
optional components. It does not poll or scan the document repeatedly. It
collects a small bounded sample of meaningful resource timings, then settles.
Offline state is immediate; online recovery returns through normal hysteresis.

## Media and stability

Static HTML remains useful without JavaScript. Images keep native lazy loading,
intrinsic dimensions, and responsive sources. Video elements keep a poster,
controls where appropriate, and native sources, but previews do not autoplay or
preload bytes. Article video uses `preload="none"` by default so browser-native
playback still works on interaction. Existing author configuration can request
autoplay explicitly.

Cards and embeds reserve their existing geometry. The controller never replaces
an image source, changes dimensions, restarts playback, or inserts a large
placeholder after initial render. Optional external players load only after
reader interaction where practical. Static resource hints remain limited to
critical assets; speculative third-party connection work must not be triggered
by mere mention of a URL.

## UI and implementation

Add a compact loading preference control to the existing header controls, with
a native disclosure-style panel and a polite mismatch status. It must work on
small screens, with keyboard input, and without relying on color alone. The
controller is a standalone theme asset, measured after minification and gzip.
The desired budget is 2 KB compressed; 3 KB is the hard ceiling.

Add pure policy tests for signal classification, hysteresis, overrides,
persistence decisions, and offline/online transitions. Add template tests for
native media defaults and verify both active template trees. Use the
waylonwalker.com site as the real corpus and compare desktop plus constrained
mobile loads in Chromium. Report browser automation limits for Firefox and
Brave when unavailable locally.
