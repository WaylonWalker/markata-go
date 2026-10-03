---
title: "Build Span Presentation Model"
description: "Developer contract for the Builder Admin span view foundation"
date: 2026-10-02
published: true
tags:
  - documentation
  - architecture
  - builder-admin
---

Builder Admin has an internal span presentation model for future trace views.
It converts completed build spans into rows ordered by start time, with nesting
depth, integer millisecond timings, attribution, and copied attributes.

The highlighted completion chain follows the latest-finishing leaf and its
ancestors. This is a timing heuristic; it does not calculate a dependency-aware
critical path. Missing parents and cyclic parent links terminate traversal.
Negative timings display as zero, and sub-millisecond timings round down.

The model does not yet expose a waterfall or trace API. Benchmark retention and
operator UI integration remain follow-up work. For recorded span data today,
see [the performance guide](/guides/performance/).
