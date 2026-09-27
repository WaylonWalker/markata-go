# Serve Control Center

## Scope

`markata-go serve` owns one session runtime. Terminal, plain output, and the local
web control center read snapshots from that runtime. A successful rebuild updates
the current site and page state while completed jobs and their diagnostics remain
available for the session. The existing Builder Admin retains its production
queue, releases, persistence, proxy authentication, and webhooks. Its local control
view uses the same runtime representation as the terminal.

## Runtime records

The runtime records server status, jobs, steps, pages, feeds, diagnostics, and
structured logs. Jobs have stable session IDs, source/trigger, queued/start/end
times, state, affected pages, steps, and related logs and diagnostics. Steps use the build
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

`serve` mounts the local control center under `/_markata/` on the same listener
when the bind address is loopback (`localhost`, `127.0.0.1`, `::1`, or another
parsed loopback IP). `--no-admin` disables it. `--admin` remains an explicit
opt-in but is rejected on a non-loopback listener; combining `--admin` and
`--no-admin` is a usage error. Non-loopback listeners do not mount local admin
by default. `markata-go admin` starts the loopback-only local web experience
with serve's build and watch runtime. The session server state contains the
admin URL only when the route is mounted. Plain output advertises that URL,
and the TUI offers an Admin shortcut that opens the same loopback URL.
Production Builder Admin authentication remains unchanged.

The runtime feed inventory projects lifecycle feed names, titles, output paths,
and post entries. Each entry retains its source path, title, date, and preview
URL. Clients use this metadata rather than parsing generated HTML. The TUI
supports feed search, feed detail, entry search, and navigation to existing page
detail. Nested resource traversal keeps a navigation stack so `Esc` returns one
level at a time. Source actions resolve relative page paths from the configured
content directory. The local web dashboard uses system fonts and does not load
external assets. Builder Admin serves its control projection from memory during
state polling and refreshes it when its state changes.

## Clients

The TUI shows server/site state, jobs and steps, recent and job-scoped logs,
warning and error inboxes, pages, and feeds. Navigation, filtering, resize, and
scroll position are client-only state. Scrolling away from the log tail freezes
position while new lines accumulate. Narrow terminals use a single-column view.
The web client presents the same runtime objects and contextual actions through
the session API. A warning or error never exists only in transient scrollback.
During active work, the TUI may show a fixed-width, palette-aware pulsing dot.
Animation is limited to interactive terminals and does not appear in plain
output. `NO_COLOR` disables TUI color.

Safe source fixes are planned against a source digest. Clients preview selected
changes before applying them. Apply rechecks the digest and edit spans; stale or
ambiguous changes are rejected without modifying the file.

## Failure and shutdown

Bind failures must be reported as server failures and prevent a false ready
state. A failed build stays visible until a later successful build supersedes
the current site state, while its job remains in history. Cancellation closes
subscriptions and stops HTTP and file watching. A fatal TUI error is printed
after alternate-screen teardown.
