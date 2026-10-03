---
title: "Annual calendar archive design"
description: "Readable months and bounded calendar rendering."
date: 2026-09-30
published: false
tags: [design, feeds]
---

The archive reads as a publishing almanac: a prominent year, publishing totals, and spacious month panels using the site's heading font and palette. Columns respond to their container with a 17rem minimum where space permits. Narrow screens show one month per row.

Annual pagination bounds the active grid and gives visitors a stable URL. Show the newest populated year, with older/newer controls skipping empty years and a native year selector. Restore view and year through browser history; invalid years fall back to the newest populated year.

List-first feeds defer month construction until Calendar opens. Each year selection replaces the grid with at most twelve months. Retain lazy rich previews, thumbnails, and the no-JavaScript list fallback. Full archive metadata stays in the HTML; this reduces DOM work without reducing transfer size.

Browser coverage verifies deferred rendering, month bounds, empty-year skipping, URL preservation, and Back/Forward. A clean real-site build verifies desktop/mobile sizing and DOM reduction.
