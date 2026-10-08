---
title: "Adaptive Loading"
description: "How Markata-Go keeps sites useful and calm on slow or unstable connections."
date: 2026-10-07
published: true
tags:
  - performance
  - themes
---

# Adaptive Loading

Markata-Go sites keep article text available immediately and let the browser
load images and video with native HTML behavior. Visitors can choose how much
optional media work the site should do.

## Choose a loading mode

Open **Loading** in the site header and choose one of these modes:

- **Auto** uses recent browser connection hints and a few real resource timings.
- **Save Data** limits optional media work and speculative requests.
- **Full Quality** allows optional work to begin sooner.

Auto is the default. A saved Save Data or Full Quality choice always takes
precedence over Auto. The preference is stored in the current browser only.

## What Auto does

Auto uses `navigator.connection` when the browser exposes it, but the site
works without that API. It also learns from a small number of recent navigation
and resource timings. One slow image or request does not change the mode. Auto
requires repeated evidence and uses stronger evidence before recommending a
manual setting change.

If Auto becomes confident that your manual choice no longer fits the
connection, a small status prompt offers **Use Auto** or **Keep [mode]**.
Keeping or dismissing the choice quiets the prompt for the current session.

## Media and JavaScript

Images use browser-native loading and retain their dimensions while loading.
Video previews show a poster and do not fetch the video until playback is
requested. Article video keeps native controls and remains usable if JavaScript
does not run. The adaptive script does not replace an image that is already
visible or restart a video when the connection changes.

Markdown and raw HTML videos default to no autoplay and `preload = "none"`.
To opt into GIF-like playback for site videos, set:

```toml
[markata-go.md_video]
autoplay = true
loop = true
muted = true
preload = "metadata"
```

The controller is a small theme asset. Video and embed helpers remain
conditional, so a text-only page does not load a player just in case.
