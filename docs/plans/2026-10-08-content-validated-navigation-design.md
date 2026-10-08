---
title: "Content-Validated Navigation Investigation"
description: "A bounded prototype and benchmark plan for static route navigation projections."
date: 2026-10-08
published: true
tags:
  - performance
  - navigation
  - design
---

# Content-Validated Navigation Investigation

Issue [#1528](https://github.com/WaylonWalker/markata-go/issues/1528). Separate
branch from main; adaptive loading remains in PR #1527. This is an experiment,
not a production router. Stop when broad projections show weak savings or
correctness requires an unjustified runtime.

## Proposed contract and measurement order

1. **Artifact.** One stable `/route/_nav.html`, using the existing broad
   `#view-transition-page` wrapper (main and route sidebars). A small HTML
   envelope carries schema, route revision, shell revision, canonical head state,
   body attributes, and route asset declarations. Start broad; measure before
   cutting regions or refactoring every template.
2. **Canonical equivalence.** Extract exact byte ranges from the same canonical
   document already passed to the publisher. Tokenize only to locate ranges;
   never re-render the region. Require byte-for-byte wrapper equivalence across
   article, feed, image-heavy, video, and embed routes. Metadata/asset requirements
   also come from that document. Unknown/missing wrapper means no projection.
3. **Route revision.** SHA-256 of schema, exact route region, metadata, and asset
   requirements. Local hashed asset URLs bind their requirements to content.
   Mutable/unidentified assets are an uncertainty and require full navigation.
   The revision is metadata/ETag, never a cache-busting URL.
4. **Shell revision.** Schema/lifecycle contract version plus persistent shell
   structure. Ordinary article edits cannot change it. Measure whether current
   headers/footer contain route state that must instead be metadata. Compatible
   freshly published content stays usable by an older open shell. Incompatible
   global runtime/structure changes intentionally fall back.
5. **Incremental integration.** Emit projections inside existing post/feed
   publication paths, only when those paths run under their existing cache/DAG
   decisions. Compare candidate bytes before writing. Include missing projections
   in expected-output repair. No new build ID, route cache, or invalidation graph.
   Prototype limitations for other generated route publishers must be explicit.
6. **Metadata.** Begin with title/canonical and actual route head/body state;
   inventory classes, selected nav, critical styles, script/style requirements.
   Measure how much head data is truly persistent before removing it.
7. **Assets.** Inventory current conditional/global scripts and styles. Prototype
   compatible already-loaded assets first. Load declared supported new assets
   before commit; unknown initializers or failures use the original link. Do not
   silently claim template equivalence with missing route assets.
8. **Lifecycle.** Audit existing full-document View Transition navigation,
   cleanup/idempotence, retained DOM, search, tooltips, gallery, video, timers,
   and adaptive loading. Evaluate `markata:navigation-before/swap/after` as a tiny
   generic contract. Preserve header/theme/adaptive objects; route setup must be
   explicit and repeatable. Adaptive confidence survives but each new route
   needs a fresh bounded observation budget and authored-media entry hook.
9. **HTTP cache.** A local server serves content-derived ETags and no-cache
   revalidation. Test unchanged 304, changed content, changed route asset, and
   shell mismatch. Also serve artifacts on a basic static server without ETags;
   correctness may use a 200 response, never an assumed global deployment ID.
10. **Fallback.** Safe same-origin ordinary links only. Missing/malformed
    projection, schema/shell mismatch, missing region, unsafe assets, navigator
    exception, external/download/target/modifier click all retain normal browser
    navigation. Latest compatible projection wins; no stale application cache.
11. **Benchmarks.** Reuse isolated real-site inputs/output; leave the source site's
    local edits untouched. Measure article→article/feed/image-heavy/video/embed,
    feed→article, back/forward on desktop, mobile, and constrained mobile
    (390×844, 150 ms, 192 KiB/s). Use three runs/medians. Record HTML and total
    bytes, requests, click-to-useful-content, CLS, assets, and runtime cost. Count
    actual output writes for no change, article, feed dependency, and shell change.
12. **Alternatives.** Compare ordinary loads, HTMX hx-boost full documents, HTMX
    with true projections, and a tiny native navigator. Measure actual HTMX
    2.0.11 distribution sizes from its official documentation; account for any
    existing HTMX/runtime shipping cost. A full-document fetch does not reduce
    HTML bytes. Keep View Transitions optional and reduced-motion-safe.

## Stop conditions

First quantify a correctness-preserving broad projection, including metadata
and required assets. If it saves only a few KB while retaining the large route
sidebar, stop rather than building a second router. A prototype can remain useful
as reproducible evidence even when the recommendation is NOT JUSTIFIED. Strong
savings justify only the smallest follow-up; unresolved asset/lifecycle/history
contracts keep production status NEEDS MORE EVIDENCE.
