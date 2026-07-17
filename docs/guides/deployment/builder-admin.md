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

## Secure Access with hlab-auth

Builder admin is an operator control plane. Expose it only through its protected Traefik ingress;
direct Service access and `kubectl port-forward` fail closed because they cannot prove that the
request came through the trusted proxy.

```yaml
builderAdmin:
  enabled: true
  auth:
    # CIDR used by the Traefik instances and builder-admin peers that forward this
    # Ingress. 0.0.0.0/0 and ::/0 are rejected.
    trustedProxyCIDRs:
      - 10.42.0.0/24
  ingress:
    enabled: true
    host: builder.example.com
    ingressClassName: traefik
    tls:
      enabled: true
      secretName: builder-example-com-tls
    auth:
      enabled: true
      internalUrl: https://hlab-auth.example.svc.cluster.local
      # Optional: browser-reachable auth origin for the operator's own picture.
      # This does not alter hlab-auth login, session, or WebAuthn configuration.
      publicAuthOrigin: https://auth.wayl.one
  networkPolicy:
    enabled: true
    # Verify these against the live Traefik installation before applying.
    traefikNamespace: kube-system
    traefikNamespaceSelector:
      kubernetes.io/metadata.name: kube-system
    traefikPodSelector:
      app.kubernetes.io/name: traefik
      app.kubernetes.io/instance: traefik-kube-system
```

The chart strips all client-provided stable `X-Hlab-*` headers before calling hlab-auth. On a
successful decision, Traefik forwards only the stable hlab-auth headers to builder admin. The
ForwardAuth `internalUrl` must be an `https://` cluster-reachable URL; Helm rejects HTTP URLs.
Set `trustedProxyCIDRs` to only the actual Traefik source CIDRs seen by the pod. A shared Pod CIDR
is permitted only when it is needed for builder-admin peer forwarding and the required selector
NetworkPolicy restricts that CIDR to the configured Traefik pods and builder-admin peers. Do not
use universal, loopback, or link-local CIDRs; the service rejects them.

The chart calls hlab-auth's supported `/api/v1/forward-auth` endpoint and configures Traefik to
copy `__Host-hlab-app-session` back to the browser. This is required for cross-origin auth
handoff. Before installing, configure an hlab-auth route for this hostname/path and grant the
selected users/groups the route's required scopes in hlab-auth. Builder admin makes no authorization decision from a
username, display name, group, role, or scope; hlab-auth is the access decision point.

The operator panel shows the trusted durable user ID and any supplied username, display name,
email, groups, roles, and scopes. These are display-only. Set `publicAuthOrigin` only when the
browser can reach hlab-auth at that HTTPS origin and you want the panel to request the signed-in
operator's own picture. The URL is derived from the authenticated stable user ID; a `No image`
fallback remains visible when it is unset, unavailable, or cannot load. Do not use this setting to
change hlab-auth's primary public auth origin, cookies/sessions, or primary WebAuthn RP.

When `builderAdmin.enabled` is true, Helm fails unless the protected ingress, TLS secret, ingress
host/class, ForwardAuth URL, trusted proxy CIDRs, and builder-admin ingress NetworkPolicy are all
configured. The service receives its CSRF public origin as exactly `https://<ingress host>` from
the chart. Do not supply that value from request headers or alter hlab-auth's primary RP/origin.
The NetworkPolicy permits the configured `builderAdmin.port` only from the configured Traefik
namespace/pod selectors and other builder-admin pods; confirm the default labels match the
installed controller. The peer-forwarding marker is accepted only from a configured trusted CIDR;
it cannot authenticate a direct request by itself.

Authenticated `GET /` responses set a host-only, secure, HttpOnly, strict-SameSite CSRF cookie
and include its token in each mutation form. Browser mutations must present the matching token and
the exact configured Origin; programmatic clients may use `X-CSRF-Token` instead of a form field.

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
