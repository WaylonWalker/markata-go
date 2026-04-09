# Analytics And Storytelling

Use this topic when the user wants an `analytics.md` page, a metrics dashboard, chart-driven posts, contribution graphs, or template badges for reading time and content stats.

## What Agents Should Do First

1. Inspect the active config with `markata-go config show`.
2. Confirm whether the site already enables `stats`, `chartjs`, and `contribution_graph`.
3. Inspect existing analytics pages, feed pages, and template partials before adding a new pattern.
4. Decide whether the task belongs in:
   - a markdown page with `jinja: true`
   - an existing HTML template or partial
   - config only
   - a plugin change

Prefer content, templates, and built-in stats over custom plugin work.

## Built-In Data Surfaces

### Per-post stats

Available on posts after the `stats` plugin runs:

- `post.Extra.word_count`
- `post.Extra.char_count`
- `post.Extra.reading_time`
- `post.Extra.reading_time_text`
- `post.Extra.code_lines`
- `post.Extra.code_blocks`
- `post.Extra.stats`

These are good for post headers, feed cards, article footers, and "long read" badges.

### Site-wide stats

Available in templates and Jinja-enabled markdown via `config.Extra.site_stats`:

- `total_posts`
- `total_words`
- `total_chars`
- `total_reading_time`
- `total_reading_time_text`
- `average_words`
- `average_reading_time`
- `average_reading_time_text`
- `total_code_lines`
- `total_code_blocks`
- `posts_by_year`
- `words_by_year`
- `posts_by_tag`

Use these for analytics pages, about pages, and site summary components.

### Advanced helper

`config.Extra.stats` exposes a helper object for site and feed KPIs. Prefer `config.Extra.site_stats` for straightforward template access. Reach for the helper only when the site already uses it or the task specifically needs helper-driven feed metrics.

## Chart And Graph Options

- Use `chartjs` fenced blocks for bar, line, pie, doughnut, radar, polar area, bubble, and scatter charts.
- Use `contribution-graph` fenced blocks for calendar-style publishing/activity views.
- When the repo already has a chart palette or CSS treatment, reuse it instead of inventing a new visual language.

## Recommended `analytics.md` Pattern

Use a normal markdown page with frontmatter like:

```yaml
---
title: "Site Analytics"
description: "Publishing metrics, activity charts, and yearly trends for this site."
date: 2026-04-09
published: true
slug: /analytics/
jinja: true
tags:
  - analytics
  - meta
  - writing
---
```

Then structure the page in three layers:

1. KPI summary
2. one or two charts
3. author-written interpretation

Agents should compute or scaffold the first two layers and leave space for the third.

## Good Analytics Page Structure

Use sections like:

- `## At a glance`
- `## Publishing activity`
- `## Topic mix`
- `## What changed this year`
- `## Notes from the author`

The first sections can be data-driven. The last sections should invite the author to explain timeline changes, experiments, and tradeoffs in their own voice.

## Safe Authoring Pattern

When asked to create an analytics post, prefer this workflow:

1. Gather site metrics from built-in stats, list commands, and nearby content.
2. Draft an `analytics.md` page with working charts and KPI callouts.
3. Add short prompts or placeholders where the human should add interpretation.
4. Avoid inventing personal history, timelines, or motivations unless the user explicitly asks for prose generation.

Example placeholder style:

```markdown
## What changed this year

Write about the periods where output increased or dipped.
Possible prompts:
- Which projects or life events changed your publishing pace?
- Which topics became more important?
- Were there intentional breaks, experiments, or format changes?
```

## Template Snippets

### Post badge

```django
<div class="post-meta">
  <span>{{ post.Extra.reading_time_text }}</span>
  <span>{{ post.Extra.word_count }} words</span>
</div>
```

### Site KPI block

```django
<section class="site-kpis">
  <p>{{ config.Extra.site_stats.total_posts }} posts</p>
  <p>{{ config.Extra.site_stats.total_words }} words</p>
  <p>{{ config.Extra.site_stats.total_reading_time_text }} total reading time</p>
</section>
```

## Markdown Snippets

### KPI summary in `analytics.md`

```markdown
## At a glance

- {{ config.Extra.site_stats.total_posts }} published posts
- {{ config.Extra.site_stats.total_words }} total words
- {{ config.Extra.site_stats.total_reading_time_text }} total reading time
- {{ config.Extra.site_stats.average_words }} average words per post
```

### Topic mix chart

Agents may either:

- generate a static `chartjs` block from inspected repo data
- or wire the page to existing site stats if the repo already has a stable pattern for dynamic Jinja-generated chart data

Keep the chart JSON valid and prefer a small number of clearly labeled datasets.

## Storytelling Guidance

Agents should help authors create stories from their data by:

- identifying notable changes in publishing volume, topic mix, or reading-time distribution
- suggesting section headings and narrative prompts grounded in the metrics
- calling out where the data is strong and where it is only directional
- keeping claims tied to visible charts or explicit metrics

Agents should not overstate causation from site metrics alone.
