---
title: "Serve Control Center"
description: "Inspect live builds, warnings, pages, and logs in the terminal or local web dashboard"
date: 2026-09-26
published: true
tags:
  - documentation
  - development
  - cli
---

# Serve Control Center

Start a local development session:

```bash
markata-go serve
```

In a terminal, `serve` opens a dashboard showing server status, recent jobs,
warnings, errors, pages, and logs. The site remains at
`http://localhost:8000`. File changes create rebuild jobs. Each job keeps its
steps and diagnostics, so a failed rebuild remains inspectable after the next
successful one.

For line-oriented output, use `markata-go serve --no-tui`. Markata also chooses
line-oriented output when input or output is redirected, when `--no-input` is
set, or when the terminal is unavailable. This mode works in scripts and CI.
While the interactive dashboard is open, build and plugin messages appear in
its Logs and Problems views. Serve keeps terminal ownership so rebuild output
does not overwrite the dashboard, including while the terminal is resized.

## Local web dashboard

```bash
markata-go serve --admin
# Site:  http://localhost:8000
# Admin: http://localhost:8000/_markata/
```

`markata-go admin` starts the same local serve session with the web dashboard
enabled and line-oriented terminal output. Both commands show the same jobs,
warnings, errors, and pages that the terminal dashboard sees. The local admin
dashboard only binds to loopback. For remote, release-oriented operations use
the protected [Builder Admin deployment guide](/docs/guides/deployment/builder-admin/).

The web dashboard supports manual builds, job reruns, and rebuilding a selected
page. Jobs keep logs and lifecycle steps. Page details show source diagnostics
and suggested fixes when Markata has enough context. You can open a generated
page from its page entry when it has a URL. The Feeds section uses the same
feed and post metadata as the build, and its post links open the matching page
details.

## Terminal controls

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Move selection or scroll |
| `Enter` | Inspect selected job, page, or diagnostic |
| `Esc` | Return to the previous view |
| `t` | Trigger a build |
| `r` | Rerun the selected job |
| `w` / `e` | Open warnings or errors |
| `l` | Open logs |
| `p` / `f` | Open pages or feeds |
| `o` / `v` | Open selected source or local preview |
| `/` | Search or filter the current view |
| `g` / `G` | Jump to first or last item |
| `PgUp` / `PgDn` | Scroll by a page |
| `?` | Show help |
| `q` | Quit |

When you scroll away from the live log tail, new lines do not move your view.
The dashboard shows a count of new lines until you return to the bottom.
Resize the terminal at any time; narrow windows use a single-column layout.

## Reading diagnostics

A diagnostic has a stable code, source path and line when known, the job that
found it, and a suggested fix. The warnings and errors inboxes retain entries
for the session. Selecting a page shows its current status and diagnostics;
selecting a job shows the diagnostics from that run. Some plugin messages do
not include a source location. Those remain in the job and session inbox so
they cannot disappear into log scrollback.
