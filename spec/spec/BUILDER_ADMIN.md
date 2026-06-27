# Builder Admin Specification

This document specifies the long-lived builder admin service for markata-go Kubernetes deployments.

## Goals

- Provide a warm, long-lived build worker so authoring builds avoid per-Job startup cost.
- Expose an operator-facing HTTP UI and API for builds, releases, logs, refresh tasks, and rollback.
- Preserve the existing release model based on `releases/<id>/` plus a `current` symlink.
- Keep remote-content refresh work out of normal content builds unless explicitly configured otherwise.

## Scope

The builder admin service is intended for self-hosted and Kubernetes workflows, especially hostPath-backed authoring deployments.

The first required capabilities are:

- serialized build queue
- manual HTTP-triggered builds
- file-watch triggered builds that enqueue through the same queue
- build history with full raw logs
- release history with current/live indicator
- promote-previous-release rollback
- scheduled refresh tasks for reader/blogroll/other external data commands
- operator UI that shows running, queued, successful, and failed work

## Runtime Model

The builder admin service MUST run as a long-lived HTTP process.

It MUST mount the same site-authoring paths as the existing build workflow:

- source tree
- rendered site root
- optional dedicated cache volume

The service MUST process queued work one item at a time for a given site.

Triggers MUST enqueue work rather than executing builds directly.

Required trigger sources:

- manual UI action
- manual HTTP API call
- file watch
- scheduled refresh completion when configured to enqueue a build
- rollback action

## Build Workflow

Successful builds MUST preserve the existing atomic release publication model:

1. prepare cache symlinks when a dedicated cache mount is configured
2. seed a stable work directory from the current release when one exists
3. run `markata-go build` into the work directory
4. move the finished output into `releases/<release-id>/`
5. atomically repoint `current` to the new release
6. prune old releases according to retention policy

The service MUST record phase timings for at least:

- queue wait
- prepare
- build
- promote
- prune
- total

The service MUST store the full raw build log and a parsed performance summary that includes any `Duration:` and `Hotspots:` lines emitted by markata-go.

## File Watching

When file watching is enabled, the service MUST watch the configured source roots recursively.

Watch events MUST be debounced and coalesced into a single queued build request.

The recorded build trigger MUST include:

- trigger type `file-watch`
- the set of changed paths captured during the debounce window

The watcher SHOULD ignore internal cache and admin-state paths.

## Build History

Each build record MUST include:

- unique build id
- operation kind: `build`, `refresh`, or `rollback`
- status: `queued`, `running`, `success`, `failed`, `cancelled`
- trigger type
- trigger detail text
- changed paths when available
- enqueue, start, and finish timestamps
- per-phase timings
- total duration
- raw log path
- parsed performance summary
- produced release id, when applicable
- whether the result became live

The UI MUST show current queue state, running build state, and the current live release.

## Releases And Rollback

The service MUST discover releases from the site root `releases/` directory and the `current` symlink.

Rollback in the first version is defined as:

- selecting a previously successful rendered release directory
- atomically repointing `current` to that release
- recording a rollback operation in history

The UI MUST clearly indicate that rollback promotes a prior rendered release rather than restoring the historical source tree.

## Refresh Tasks

The builder admin service MUST support configured scheduled refresh commands.

Each refresh task MUST define:

- stable task name
- command argv
- interval duration
- whether a successful run enqueues a build

The first version MAY use fixed interval durations instead of cron expressions.

Refresh runs MUST have their own history with:

- task name
- status
- duration
- raw log path
- optional follow-up build id when a build was enqueued

## Persistence

The service MUST persist operator state on disk so restarts do not lose build history.

Required persisted data:

- build records
- refresh records
- release metadata derived from disk and linked build ids when known
- full raw logs

The first version MAY use a JSON state file plus log files instead of a relational database.

## HTTP UI And API

The service MUST expose an HTTP admin interface.

Required UI views:

- dashboard summary
- build history list
- build detail/log view
- release list with current/live indicator
- refresh task list and refresh history

Required actions:

- enqueue manual build
- trigger refresh task immediately
- promote a prior release to live

The service SHOULD also expose JSON endpoints for the same core operations.

## Helm Integration

The Helm chart MUST support enabling the builder admin service independently of the scheduled build CronJob.

Required chart configuration includes:

- service enable/disable
- host/port
- file-watch enable/disable and debounce
- release retention
- build history retention
- refresh task definitions
- optional ingress/auth settings for future protected access

The first deployment target SHOULD work via `kubectl port-forward` even when no ingress is enabled.
