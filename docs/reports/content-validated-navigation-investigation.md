---
title: "Content-Validated Navigation: Stop After Measurement"
description: "Canonical projection byte probes show modest savings before lifecycle and publication integration."
date: 2026-10-08
published: true
tags:
  - performance
  - navigation
  - benchmark
---

# Content-Validated Navigation Investigation

Issue [#1528](https://github.com/WaylonWalker/markata-go/issues/1528), separate
`feat/content-validated-navigation-prototype` branch from `73c4f606`.
**NOT JUSTIFIED for the broad projection tested.** Stop here under the requested
weak-savings gate. No production navigator or publication path was added.

## Benefit

Five real waylonwalker.com routes were examined using the isolated adaptive
benchmark output at `aa86d8ec`. The source site's local edits were untouched.
The probe extracts the exact existing `#view-transition-page` byte range from
the canonical full document, retains its canonical head/body attributes, and
adds content-derived route/asset metadata. It never independently renders a
partial template. Every fixture passed byte-for-byte region, title, and body
attribute assertions. Local script/style requirements include content hashes.
This is an incomplete lifecycle envelope, not proof of full route-state equivalence.

### HTML transport and inert parse

The following are medians of three Chromium runs per representation, route,
and profile: 90 total runs. Mobile viewport 390×844; desktop 1365×900.
Constrained mobile uses 150 ms latency and 192 KiB/s downstream, with cache
disabled. All requests use Node gzip level 9. The table uses actual browser
`encodedBodySize`, not Python compressor estimates; zlib implementations produce
slightly different sizes. Profiles called normal are unthrottled local HTTP;
mobile CPU was not throttled.

| Destination | HTML gzip bytes full → projection | Avoided | Desktop fetch+parse ms | Mobile normal ms | Mobile constrained ms |
|---|---:|---:|---:|---:|---:|
| /vimgrep-open-buffers/ | 83,530 → 75,602 | 7,928 (9.5%) | 13.8 → 13.3 | 14.3 → 11.7 | 593.8 → 551.2 |
| /til/ | 48,621 → 40,627 | 7,994 (16.4%) | 12.8 → 11.0 | 13.2 → 11.7 | 418.0 → 376.0 |
| /2025-nas/ | 317,022 → 308,843 | 8,179 (2.6%) | 45.6 → 43.2 | 45.9 → 42.1 | 1807.7 → 1786.3 |
| /tmux-pop-size/ | 314,759 → 306,793 | 7,966 (2.5%) | 44.4 → 42.7 | 44.7 → 42.9 | 1808.3 → 1768.9 |
| /bloatware-is-dying/ | 166,823 → 158,806 | 8,017 (4.8%) | 41.2 → 37.2 | 41.6 → 36.9 | 1062.9 → 1018.7 |

This is fetch completion plus inert DOMParser extraction, **not click-to-visible
content**. No scripts run, destination assets load, page swap occurs, or history
changes. Browser HTTP request count is one for each representation; total page
transfer, LCP, CLS, CPU, back/forward, fallback frequency, and real transition
latency were not measured. The probe cannot claim improvements in those metrics.
The HTML savings apply to the destination response for article→article/feed,
feed→article, and article→image/video/embed, but those transitions were not
implemented or exercised as partial navigation. Ordinary back/forward may already
benefit from browser cache or BFCache; this experiment does not compare that.

The savings are approximately 8 KB per destination and 21–44 ms median constrained
fetch+parse time. Large image/video routes retain roughly 98% of compressed HTML.
The broad wrapper contains the large feed sidebar DOM and JSON data. Keeping
one atomic region avoids template-specific browser logic but keeps that weight.
Reducing this further would require a separate sidebar/data design experiment,
not an assumption that arbitrary fragment composition is warranted.

### Runtime alternatives and amortization

| Alternative | Destination HTML | Additional runtime evidence |
|---|---|---|
| A. Ordinary MPA load | Full document above | No new navigator |
| B. HTMX hx-boost with static full document | Same full document response | HTMX library cost below; no payload reduction by itself |
| C. HTMX with real projection | Projection response above, before lifecycle bridge | HTMX cost plus unimplemented integration |
| D. Native projection navigator | Same projection response | Not implemented after weak benefit gate; shipped cost unmeasured |

[Official HTMX documentation](https://htmx.org/docs/) links version 2.0.11.
Its downloaded distribution is 171,362 source bytes / 52,182 minified bytes /
16,897 gzip bytes / 15,259 Brotli bytes. The
[standard hx-boost behavior](https://htmx.org/attributes/hx-boost/) still requests
the original URL; a basic static server returns the full document. Real
projection selection and lifecycle integration require additional work. At
about 8 KB saved per destination, the library alone takes roughly three
uncached subsequent transitions to amortize. It provides no other demonstrated
benefit in this experiment. With cached runtime, amortization differs.

Markata already ships a native full-document View Transition navigator. Its
actual generated checkpoint file is 23,339 bytes minified / 6,820 gzip /
6,090 Brotli. It already preserves the header and swaps the broad wrapper.
A projection experiment should adapt that existing contract if justified,
rather than ship a second competing navigator. Neither those existing bytes
nor the transport harness are a measured new tiny navigator implementation.
A tiny native implementation's source/minified/compressed budget remains
unmeasured because the experiment stopped before building one.

## Tradeoffs

The runtime audit found head/body attribute synchronization, conditional
scripts/styles, inline bootstraps, search initialization, feed cycling,
scroll-spy, tooltips, mention cards, pagination, keyboard navigation, galleries,
media, progress widgets, and sidebar scroll/pin state. Some route widgets sit
outside `#view-transition-page`. The footer differs by route and cannot simply
persist unchanged. Global header markup is identical on the five sampled routes.
A production shell compatibility hash must cover the schema, persistent
structure and incompatible runtime lifecycle changes; the probe hashes only an
experimental schema plus that header. It must not be used as a production
compatibility guarantee. A content edit changes its route revision while leaving
that experimental shell hash stable. No global build identifier is stamped.

The existing navigator stores prefetched full documents in memory. That cache
needs a deliberate HTTP validation contract before it can promise latest content
published during an open session. Unknown wrappers, assets/initializers, or
failed initialization must fall back to the original ordinary link, preserving
modifier/download/external/target behavior. No new interception or fallback
implementation was shipped, so fallback frequency is unknown. View Transitions
remain optional; reduced-motion and unsupported browsers must work independently.
History, focus, scroll restoration, cleanup, and idempotent route initialization
remain production correctness work, not benefits proved by this probe.

The audit uncovered a separate adaptive runtime issue: the existing navigator
could reset adaptive root attributes. That was fixed and tested on the adaptive
branch in `adccb481`, preserving policy and giving the new route its own bounded
timing window. It is a prerequisite for preserving adaptive session state; no
adaptive changes were mixed into this main-derived investigation branch.

### HTTP validation experiment

Five Python tests cover exact canonical extraction/metadata, deterministic route
revision/locality, changed local asset revision, missing-region rejection, and
actual HTTP conditional validation. A small in-test hosting adapter serves stable
`/a/_nav.html` with `ETag` and `Cache-Control: no-cache`. Unchanged `If-None-Match`
returns 304; changed content, local asset bytes, and shell structure return 200
with a changed revision. Content/asset changes leave the experimental shell hash
unchanged; structural header changes alter it. These are controlled fixtures,
not deployed CDN or navigation history tests. The test does not implement an
asset initializer or a shell-mismatch browser fallback.

The 90-run transport server supplied no ETags and served both representations
successfully as ordinary static 200 responses. Generated static files cannot set
HTTP response headers on their own. A basic
static server can serve the artifact with a 200 response; accurate ETag/304
behavior depends on server/CDN support and cache configuration. Embedding a
revision is not HTTP validation. The route URL remains stable; no revision query
parameter or build-session cachebuster is introduced.

### Incremental publication

Production write counts were **not measured** after the weak-benefit stop.
The offline script writes transport fixtures outside the site's output/cache;
it is not a publisher and must not be integrated as a second invalidation system.
No production build writes `_nav.html`, so existing build locality is unchanged.
If revived, production emission must consume the same canonical `post.HTML`/feed
HTML in existing publication paths, obey their cache/DAG decisions, compare bytes
before writing, and participate in expected-output repair. All post/feed/generated
publishers need coverage. No-change, one-article, feed dependency and shell-change
written-file counts are mandatory before any implementation readiness claim.

## Readiness

**Partial navigation: NOT JUSTIFIED for this broad projection.** Measured savings
are small enough to stop before building a lifecycle/history/asset runtime or
altering publication. Canonical extraction and hosting validation are promising
mechanisms but do not establish a production-ready feature. No full browser
accuracy matrix, deployed cache experiment, production DAG/write-count test, or
cross-browser partial navigation claim is made.

**Adaptive loading: READY FOR PR**, independently tracked in
[PR #1527](https://github.com/WaylonWalker/markata-go/pull/1527). Its browser and
transfer evidence is in [the hardening report](https://github.com/WaylonWalker/markata-go/blob/feat/adaptive-client-loading/docs/reports/adaptive-client-loading-hardening.md).
That report belongs to the adaptive branch, not this separate main-derived one.

The bundled site-agent skill was reviewed. No update is needed here: this
investigation adds no supported build option, site-author workflow, or runtime.
It would be misleading to teach agents to enable this offline projection probe.

## Next steps

Review adaptive PR #1527 without merging automatically. Retain this separate
investigation as evidence and stop projection implementation. If navigation HTML
weight becomes a priority, first measure how much of the sidebar/list payload can
be removed through existing content/template configuration while preserving normal
MPA output. Revisit projection integration only after a new byte measurement shows
material gains, then require lifecycle, history, assets and incremental write tests.

## Reproduce the bounded probe

Use an existing generated site in read-only mode and an isolated fixture directory:

```bash
python3 scripts/navigation-projection/probe.py /path/to/site-output /tmp/nav-fixtures > /tmp/nav-projections.json
node scripts/navigation-projection/transport.cjs /tmp/nav-fixtures > /tmp/nav-transport.json
python3 -m unittest discover -s scripts/navigation-projection -p 'test_*.py' -v
```

The transport command uses existing `/usr/bin/chromium` and agent-browser's
Playwright installation; `CHROMIUM_EXECUTABLE` and `PLAYWRIGHT_MODULE` can override
those paths. The scripts have help/version output. No browser/framework package
was added. Raw artifact and per-run transport measurements are in
`docs/reports/navigation-projection-probe-results.json`. The design/benchmark plan
and experiment spec were committed before the probes.
