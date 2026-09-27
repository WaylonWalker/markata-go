# Serve Control Center

## Scope

`markata-go serve` owns one session runtime. Terminal, plain output, and the local
web control center read snapshots from that runtime. A successful rebuild updates
the current site and page state while completed jobs and their diagnostics remain
available for the session. The existing Builder Admin retains its production
queue, releases, persistence, proxy authentication, and webhooks. Its local control
view uses the same runtime representation as the terminal.

## Runtime records

The runtime records server status, jobs, steps, pages, diagnostics, and structured
logs. Jobs have stable session IDs, source/trigger, queued/start/end times, state,
affected pages, steps, and related logs and diagnostics. Steps use the build
lifecycle vocabulary. Diagnostics have severity, stable code, message, optional
explanation, source location, page, job and step identity, and an optional fix.
Missing optional fields remain empty. The runtime makes defensive snapshots for
clients and never asks a client to infer state from a formatted log line.

Content diagnostics from the existing `pkg/diagnostics` ledger retain their
existing stable codes. Session-level errors use a separate `serve.*` code. A
diagnostic belongs to the job that found it; the current page view reflects the
latest build that touched that page. Historical job diagnostics remain browsable
after later builds. Logs are bounded; diagnostics and useful job history are
retained for the session. Subscribers receive coalesced snapshots without holding
up the build path.

## Actions

The runtime accepts explicit build and rerun requests. An action is validated
against current state before invoking a handler. The terminal and web clients
submit the same action request. Page source/open and preview links are exposed
only when a source file or output URL exists.

## Serve integration

The initial build and every rebuild create a job. Build stages create steps.
Build results supply page dispositions and diagnostics; build errors create a
durable diagnostic even if the build stopped before producing a ledger snapshot.
The server status changes only after the listener has bound. `serve --no-tui`
and non-TTY output use line-oriented text. A real interactive terminal defaults
to the TUI. No build output may write beneath the alternate screen; producers
send structured events to the runtime or a captured log sink.

`serve --admin` mounts the local control center under `/_markata/` on the same
listener. `markata-go admin` starts the local web experience with serve's build
and watch runtime. Local admin access binds to loopback by default. Production
Builder Admin authentication remains unchanged.

## Clients

The TUI shows server/site state, jobs and steps, recent and job-scoped logs,
warning and error inboxes, and page details. Navigation, filtering, resize, and
scroll position are client-only state. Scrolling away from the log tail freezes
position while new lines accumulate. Narrow terminals use a single-column view.
The web client presents the same runtime objects and contextual actions through
the session API. A warning or error never exists only in transient scrollback.

## Failure and shutdown

Bind failures must be reported as server failures and prevent a false ready
state. A failed build stays visible until a later successful build supersedes
the current site state, while its job remains in history. Cancellation closes
subscriptions and stops HTTP and file watching. A fatal TUI error is printed
after alternate-screen teardown.
