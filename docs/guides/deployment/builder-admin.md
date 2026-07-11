---
title: "Builder Admin"
description: "Run a long-lived builder admin service for fast Kubernetes authoring loops"
date: 2026-06-27
published: true
tags:
  - documentation
  - deployment
  - kubernetes
  - performance
---

# Builder Admin

The builder admin service keeps a warm markata-go build worker running inside your cluster.

Instead of starting a new Kubernetes Job for every authoring build, it keeps one HTTP service alive with:

- a serialized build queue
- file watching that enqueues builds
- build history and full raw logs
- release history and current live release
- rollback by promoting an older rendered release
- scheduled refresh tasks for reader, blogroll, or other remote-content commands

## What It Is Good For

Use builder admin when:

- your site content already lives on a mounted filesystem such as a hostPath
- you care about fast authoring loops more than one-shot batch builds
- you want an operator UI for builds and releases
- you want remote-content refreshes to stay decoupled from normal content builds

## Basic Helm Values

```yaml
builderAdmin:
  enabled: true
  port: 8080
  fast: false
  watch:
    enabled: true
    debounce: 2s
  releases:
    keep: 10
  history:
    successfulBuilds: 50
    failedBuilds: 100
    refreshRuns: 100
  refreshTasks:
    - name: reader-update
      every: 30m
      enqueueBuildOnSuccess: true
      args:
        - markata-go
        - --config
        - /data/source/markata-go.toml
        - reader
        - update
```

Keep `builderAdmin.fast` at `false` when queued builds publish the live site. In this repo,
`--fast` is an authoring optimization, not a production-equivalent build mode: it skips
blogroll, mentions, and other expensive work that can affect user-facing output. Enable it
only when the admin service is being used as a preview loop and a separate full build path
still exists for public releases.

Current chart defaults also prefer clean rolling cutover instead of stop-then-start replacement.
Builder-admin keeps one active leader for queue draining, file watching, refresh scheduling, and
 release promotion while standby pods stay ready during rollout handoff.

## Accessing The UI

The first version is designed to work well with `kubectl port-forward`.

```bash
kubectl port-forward svc/go-waylonwalker-com-notes-builder-admin 8080:8080 -n go-waylonwalker-com-notes
```

Then open:

```text
http://localhost:8080
```

## What You Can See

The UI shows:

- queued and running builds
- recent successful and failed builds
- the trigger source for each build
- per-build duration and phase timings
- full raw logs
- parsed markata-go performance summary lines
- current live release
- old releases that can be promoted back to live
- refresh task history

The browser tab favicon also reflects live admin state so you can spot activity without keeping the tab focused:

- idle when nothing is running
- queued when work is waiting
- build when a build or rollback is running
- refresh when a refresh task is running
- error when UI polling fails

The workspace tabs show one primary view at a time so build history, refresh runs, and releases do
not visually stack on top of each other during tab switches. Build and release timestamps also pair
the absolute RFC3339 value with a relative age label such as `(5m ago)`.

## Build Triggers

Builder admin can enqueue builds from:

- the UI
- HTTP API calls
- debounced file-watch events
- successful refresh task runs when configured to enqueue a build

## Rollback

Rollback in the first version promotes a previous rendered release by repointing the `current` symlink.

This is fast and keeps the release model simple, but it does not restore the historical source tree.

## Scheduled Refresh Tasks

Refresh tasks are external commands that run on an interval.

Examples:

- `markata-go reader update`
- `markata-go blogroll update --force`
- a custom remote-asset fetch command

Use them to keep reader/blogroll data or other remote caches fresh without slowing down every normal content build.
