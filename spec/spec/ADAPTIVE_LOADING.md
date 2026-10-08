# Adaptive Client Loading

## Purpose

Generated pages SHOULD keep text useful on first render and avoid fetching
expensive media or speculative resources before readers need them. A site MUST
remain useful when JavaScript is disabled or fails.

## User modes

The public modes are `auto`, `save-data`, and `full-quality`. `auto` is the
default. A persisted manual mode MUST always determine the effective policy,
even when the automatic recommendation disagrees. The preference is local to
the browser and MUST NOT change the generated document or server state.

The document root MUST expose stable `data-loading-mode` and
`data-loading-policy` values. Optional components MAY use the effective policy
to decide whether to begin work that has not started. They MUST NOT replace an
already requested image, alter media geometry, restart playback, or remove
content after it has appeared.

## Automatic recommendation

The controller MAY use `navigator.connection.saveData` and
`navigator.connection.effectiveType` as hints. The API MUST be optional. The
controller MUST also use portable signals, including online/offline events and
a bounded sample of navigation/resource timing entries when available.

One ordinary request MUST NOT change the recommendation. Constrained state
requires either a strong browser save-data/slow-2g hint or multiple meaningful
poor resource observations. Recovery to full-quality SHOULD require more
independent good observations than entry into constrained state. Unknown or
incomplete evidence MUST choose conservative behavior for optional/speculative
work while leaving critical HTML, CSS, and native content available.

Observation MUST be bounded by sample count. Implementations MUST NOT poll or
continuously scan the document. Session state MAY persist briefly across page
navigations and MUST expire so old connection conditions do not become
permanent truth. Offline state MUST take effect immediately; online recovery
MUST follow normal hysteresis.

## Manual mismatch

When Auto confidently recommends a different policy from the saved manual
choice, the page SHOULD expose a small, accessible status affordance with
actions to use Auto or keep the manual choice. It MUST NOT block content or
remain as a permanent banner. Dismissal or keep-choice actions MUST suppress
repeat prompts for the session. The controller SHOULD surface manual
Full Quality on a confidently constrained connection sooner than
Save Data on a newly fast connection.

## Static media behavior

Images SHOULD use native responsive sources, lazy loading below the fold,
asynchronous decoding, and intrinsic dimensions. Video MUST retain a poster and
native source/control path. Video previews MUST NOT autoplay or preload media
bytes unless a site author explicitly configures autoplay. Raw HTML and
Markdown video elements SHOULD follow the same site-level autoplay and preload
defaults. Article video SHOULD default to `preload="none"`. Optional player code MUST load only on pages that
contain that player, and expensive external embeds SHOULD wait for reader
interaction.

Speculative `preconnect`, `preload`, and `prefetch` hints MUST be limited to
critical resources or gated by an effective full-quality policy. A static
critical hint MUST NOT depend on JavaScript to be removed later.

## Runtime budget and compatibility

The adaptive controller SHOULD compress to at most 2 KB and MUST NOT exceed
3 KB gzip without a documented justification. It MUST avoid frameworks,
polling, broad repeated DOM queries, and layout-dependent behavior. It MUST
work in Chromium, Brave, and Firefox without Network Information API support,
and SHOULD use APIs that degrade cleanly in Safari/iOS.

Policy tests MUST cover missing, strong, and weak connection hints; slow and
fast observations; all three user modes; mismatch confidence; hysteresis;
session expiry; and offline/online transitions. Browser checks SHOULD cover
mobile viewport, constrained networking, no premature media requests, no
visible-content mutation, and layout stability.
