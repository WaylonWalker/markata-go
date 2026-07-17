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

Kubernetes rollouts MUST support clean cutover without dropping the currently active builder-admin pod before the replacement pod is ready.

To support that requirement, the service MUST tolerate an active/standby deployment shape where multiple pods may be live briefly but only one pod is allowed to perform mutating work.

Required trigger sources:

- manual UI action
- manual HTTP API call
- file watch
- scheduled refresh completion when configured to enqueue a build
- rollback action

## Leadership And Handoff

When more than one builder-admin pod is running for the same site, exactly one pod MUST hold leadership for mutating work.

The leader is responsible for:

- draining the serialized work queue
- running file watching
- running scheduled refresh tasks
- executing builds and rollbacks
- persisting queue/running/history state

Standby pods MUST:

- serve the read-only HTTP UI and API state
- remain ready so rolling updates can keep the old leader serving while the new pod starts
- refuse or forward mutating requests unless they become leader

If a standby pod receives a mutating HTTP request while another pod is leader, it SHOULD forward that request to the active leader so operator actions do not fail during rollout handoff.

On leadership acquisition after a restart or rollout, persisted queued work MUST be replayed. A previously running in-flight operation MAY be marked interrupted instead of resumed.

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

The workspace view MUST display one primary panel at a time when switching between builds, refresh runs, and releases. Tabs MUST NOT leave multiple primary panels visually stacked on top of each other.

Build, refresh, running, and release state labels SHOULD use distinct visual status treatment so success, failure, queued, running, and live states are scannable without reading raw text.

Build and release timestamps shown in the UI SHOULD pair an absolute timestamp with a relative age label such as `(5m ago)`.

The UI MUST expose an at-a-glance browser-tab indicator by updating the favicon to reflect the current admin state:

- idle when nothing is running and the queue is empty
- queued when work is waiting but not yet running
- build when a build or rollback is running
- refresh when a refresh task is running
- error when live polling fails

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

### Operator Authentication

Every builder-admin UI, API, and log route except the Kubernetes `/health` probe MUST require a
trusted `hlab-auth` identity. The supported v1 integration is a Traefik `ForwardAuth` chain using
the hlab-auth `/api/v1/forward-auth` endpoint. The chain MUST first remove client-supplied stable
`X-Hlab-*` headers, forward the authentication decision and stable identity assertions, and copy
the `__Host-hlab-app-session` response cookie with `addAuthCookiesToResponse`. The cookie copy is
required for the cross-origin login handoff. The service MUST reject a request unless its remote
address belongs to an explicitly configured trusted-proxy CIDR and it includes exactly one
non-empty `X-Hlab-User-Id` header. Universal, loopback, and link-local trusted-proxy CIDRs MUST
be rejected. The leader-forwarding marker MUST never substitute for source provenance: it is valid
only from a configured trusted CIDR.

The operator UI MUST use `X-Hlab-User-Id` only as the durable identity key. Username, display
name, email, groups, roles, and scopes are display-only assertions; they MUST NOT grant or widen
privileges. The UI MAY show only the stable fields present in the header contract. It MAY show the
operator's own profile picture only when an explicit HTTPS public auth-origin configuration is
set; it MUST construct the escaped URL from that origin and the stable user ID as
`/users/{user_id}/picture`, with a useful no-image fallback. It MUST NOT use an identity-picture
header, forward credentials server-side, or change hlab-auth's public origin, cookie, session, or
WebAuthn configuration.

Direct Service access and `kubectl port-forward` cannot establish proxy provenance and MUST fail
closed. They are not supported operator-access paths for authenticated deployments.

### Browser Mutation Protection

An authenticated `GET /` MUST mint a cryptographically random double-submit CSRF token. It MUST
set the token in a host-only `__Host-` cookie with `Secure`, `HttpOnly`, `SameSite=Strict`, and
`Path=/`, and embed the same token in every browser mutation form. Every mutation MUST execute
only after the active leader validates a cookie token against either the submitted form token or
`X-CSRF-Token` using a constant-time comparison. It MUST also require an exact `Origin` match to
the configured HTTPS public origin and reject a present `Sec-Fetch-Site` other than `same-origin`.
Standbys MUST forward mutations before CSRF validation so that the active leader performs the
validation and active/standby handoff remains correct. The forwarding marker MUST be stripped at
the public ingress and accepted only from a configured trusted CIDR for builder-admin peer traffic
protected by the required ingress NetworkPolicy.

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

The browser UI SHOULD derive favicon state from the same `/api/state` polling payload used to update the dashboard so the tab indicator stays consistent with the visible live state.

## Helm Integration

The Helm chart MUST support enabling the builder admin service independently of the scheduled build CronJob.

Required chart configuration includes:

- service enable/disable
- host/port
- file-watch enable/disable and debounce
- release retention
- build history retention
- refresh task definitions
- rollout strategy settings for clean active/standby cutover
- protected builder-admin ingress using `/api/v1/forward-auth`, its cross-origin handoff-cookie
  forwarding, optional public auth origin for self profile pictures, and trusted-proxy CIDRs
- an exact public origin derived as `https://<builder-admin ingress host>` and passed to the
  service for CSRF origin validation, never inferred from request headers
- an enabled ingress NetworkPolicy that allows the configured builder-admin port only from the
  configured Traefik namespace/pod selectors and builder-admin peers

The first protected deployment target MUST use the dedicated authenticated ingress; `kubectl
port-forward` is intentionally not an operator-access path because it bypasses proxy provenance.

The default rendered-release retention MUST keep at least 25 releases, including the current live release, so operators have more than ten rollback targets by default.
