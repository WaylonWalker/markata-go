---
title: "Markdown Containers"
description: "Use ::: fenced containers safely, including predictable nested containers"
date: 2026-09-27
published: true
slug: /docs/guides/markdown-containers/
tags:
  - documentation
  - markdown
  - containers
---

# Markdown Containers

markata-go supports fenced Markdown containers for grouping content in a `<div>` with classes, an optional ID, and custom attributes.

## Basic container

Use three or more colons followed by a class name or attributes to open a container. Close it with a line containing only the same number of colons.

````markdown
::: card {#intro .featured data-kind="note"}
This Markdown is inside the card.
:::
````

The closing line must be **colon-only**. A line such as `::: card` or `::: {.card}` is an opening line, not a closing line.

## Nested containers

Use a longer fence for each nested level. Matching fence lengths make it clear which container closes at each point.

````markdown
::: cards
Outer content before the card.

:::: card
Inner card content.
::::

Outer content after the card.
:::
````

The example above renders the `card` inside `cards`, then continues rendering the final paragraph inside the outer `cards` container.

A useful rule is:

| Purpose | Example |
|---|---|
| Open outer container | `::: cards` |
| Open nested container | `:::: card` |
| Close nested container | `::::` |
| Close outer container | `:::` |

## Why shorter closing fences can look surprising

Container depth is determined by the number of opening colons. When a shorter colon-only closing fence is encountered while a deeper container is open, the deeper container closes without consuming that line so the parent container can process it. This lets an outer fence close the remaining nested levels, but it can look like multiple containers closed at once.

For predictable one-level-at-a-time nesting, give inner containers longer fences and close each one with its matching fence before closing the parent.

## Classes and attributes

The opening line can include multiple classes and an attribute block:

````markdown
:::: panel wide {#example .accent data-state="ready"}
Content
::::
````

This produces a container with the classes `panel`, `wide`, and `accent`, the ID `example`, and the custom `data-state` attribute.

## Web Awesome example

The same nesting rule is useful for components that transform Markdown containers after rendering:

````markdown
::: wa-tabs

:::: wa-tab {label="macOS"}
```bash
brew install markata-go
```
::::

:::: wa-tab {label="Linux"}
```bash
curl -fsSL https://example.com/install.sh | sh
```
::::

:::
````

The outer `wa-tabs` container uses three colons and each nested `wa-tab` uses four, so the structure is unambiguous.
