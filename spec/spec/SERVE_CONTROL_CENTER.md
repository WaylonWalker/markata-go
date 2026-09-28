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

Snapshots expose both historical `diagnostics` and `current_diagnostics`.
Current diagnostics combine the latest page inventory's source findings with
non-source findings from the newest completed job and any current running job.
A queued job does not clear existing findings. A successful rebuild replaces that
projection, so resolved page findings and their old fix plans leave the current
inbox while remaining attached to the historical job. A failed build with no
page inventory retains the last successful page state.

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
external assets. The local browser capabilities include build, rerun, page rebuild,
preview, and local source-fix actions. Production Builder Admin can reuse the
semantic theme and shell assets, but its capability set remains limited to its
protected queue, refresh, release, rollback, and diagnostic APIs. The production
view does not inherit local source mutation actions. Builder Admin serves its
control projection from memory during state polling and refreshes it when its
state changes.

## Clients

The TUI shows server/site state, jobs and steps, recent and job-scoped logs,
warning and error inboxes, pages, and feeds. Navigation, filtering, resize, and
scroll position are client-only state. Scrolling away from the log tail freezes
position while new lines accumulate. Narrow terminals use a single-column view.
The web client presents the same runtime objects and contextual actions through
the session API. Its semantic browser theme uses the shared token stylesheet and
Builder Admin palette roles: background, panel, surface, elevated, primary and
secondary text, accent, link, border, focus, success, warning, error, info, code,
and button colors. Both browser surfaces resolve these roles from the configured
Markata palette, with complete dark and light fallback values when a role is
unavailable. Palette CSS values are validated hex colors; invalid values use the
fallback for the active mode.

This contract shares theme tokens and a small focus primitive. Local and Builder
Admin page markup and navigation remain separately implemented. Their browser
shell convergence is tracked in issue #1300; local source-fix capabilities remain
unavailable in production Builder Admin.

The local browser URL stores the current section and selected resource in its
fragment. Loading a copied URL restores that view, and browser Back/Forward
restores earlier selections. Keyboard navigation uses `j`/`k` or arrow keys to
move through the current list, Enter to inspect, Escape to clear selection,
`/` to focus the filter, and `?` to show help. Section shortcuts include `w`
(warnings), `e` (errors), `p` (pages), `f` (feeds), and `l` (logs). Shortcuts are
inactive while focus is in a text input, textarea, select, or editable element.
The master/detail view collapses to a single column at narrow widths and keeps
visible focus indicators.

A warning or error never exists only in transient scrollback. During active
work, the TUI may show a fixed-width, palette-aware pulsing dot. Animation is
limited to interactive terminals and does not appear in plain output. `NO_COLOR`
disables TUI color.

Safe source fixes are planned against a source digest. Clients preview selected
changes before applying them. Apply rechecks the digest and edit spans; stale or
ambiguous changes are rejected without modifying the file.

### Problems and fix plans

Local Serve diagnostics may include zero or more structured fix plans. A plan
has a stable ID, category, safety class, root-relative source path, source range,
before/after text, explanation, and source digest. `SAFE` plans are deterministic
mechanical edits eligible for grouped approval; every edit still requires
preview and explicit apply. `REVIEW` plans require individual review. `MANUAL`
diagnostics provide source navigation and an explanation without a mutation
plan. A diagnostic's legacy suggested-fix text may remain for clients that do
not yet render plans.

The local browser can preview one plan, an arbitrary selection, one or more
categories, or all eligible `SAFE` plans. A batch preview freezes the selected
IDs and one digest per source file. Applying a batch checks each file against
its preview digest before any file is changed. Stale or conflicting files are
skipped and reported; valid files may still be applied. File replacements use
the existing atomic replacement path. A batch does not promise rollback across
multiple files if a later filesystem replacement fails; each outcome is
reported. The server requests one rebuild when at least one file changed, then
replaces the current diagnostic projection when that build completes. The
browser can compare the updated projection with the previewed Problems list.
The apply response reports that the rebuild was queued, not that it completed.
An apply request never silently selects or applies fixes.

Log-derived warning/error inbox entries are deduplicated only when a structured
diagnostic for the same job, severity, and normalized message exists. The
original log remains visible, and a log-only warning/error remains in the
inbox. Structured diagnostics retain their distinct source locations and codes.

Source mutation is a local Serve capability. The production Builder Admin
projection does not expose these fix endpoints or source edits.

## Failure and shutdown

Bind failures must be reported as server failures and prevent a false ready
state. A failed build stays visible until a later successful build supersedes
the current site state, while its job remains in history. Cancellation closes
subscriptions and stops HTTP and file watching. A fatal TUI error is printed
after alternate-screen teardown.
