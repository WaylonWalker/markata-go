# CLI UX Specification

## Overview

The `markata-go` CLI follows a human-first, script-friendly interaction model.
This specification defines shared user experience rules for command output,
color, prompts, help text, and error handling.

The design reference is [clig.dev](https://clig.dev).

## Shared Principles

- Commands MUST be pleasant for humans to run directly.
- Commands MUST remain composable in shell pipelines.
- Defaults SHOULD favor interactive human use.
- Machine-readable modes SHOULD be explicit.
- Help and errors SHOULD teach the next step.

## Output Streams

### `stdout`

Commands MUST send primary results to `stdout`.

Examples:
- `config show`
- `config get`
- `list posts --format json`
- `version`
- `explain`
- generated passwords or other command results intended for piping

### `stderr`

Commands MUST send operational messages to `stderr`.

Examples:
- progress updates
- verbose logs
- warnings
- prompt text
- validation diagnostics that are not part of the command's primary result
- user-facing error messages emitted by the command runner

### Writer Discipline

Command handlers SHOULD use Cobra command writers instead of hard-coded
`os.Stdout` and `os.Stderr`.

- Use `cmd.OutOrStdout()` for result output.
- Use `cmd.ErrOrStderr()` for progress, warnings, and errors.

This makes commands testable, consistent under redirection, and safe to compose.

## Color and Terminal Detection

Color MAY be used to increase scannability, but MUST remain optional.

Color output MUST be disabled when any of the following are true for the target
stream:

- the stream is not a TTY
- `NO_COLOR` is set and non-empty
- `TERM=dumb`
- the user passes `--no-color`

Commands MAY also offer explicit formatting controls for operational logs.
When they do, the supported modes SHOULD include:

- `plain` - unstyled, stable text for copy/paste and scripting
- `rich` - colored, more scannable human-oriented logs
- `auto` - choose `rich` for interactive terminals and `plain` otherwise

If both force-enable and force-disable color flags exist, commands MUST reject
conflicting combinations such as `--color` with `--no-color`.

Rich log format MAY use structured logging metadata to drive presentation.
When phase metadata is available, lifecycle-oriented logs SHOULD colorize by
phase rather than by component name alone. If a site palette is configured,
rich logs SHOULD prefer palette-derived colors when they remain readable.

Commands SHOULD make color decisions per stream. For example, `stderr` MAY stay
colored when `stdout` is piped.

When color is enabled for an interactive stream, human-facing summaries SHOULD
use color by default to improve scannability.

Commands MUST preserve readable plain output when color is disabled.

## Output Modes

### Human-Friendly Default

Commands SHOULD default to concise, readable output for humans.

### `--json`

Commands that expose structured records SHOULD offer `--json` or an equivalent
structured format flag.

### `--plain`

Commands SHOULD offer `--plain` when their default human output adds styling,
wrapping, or layout choices that make scripting less reliable.

### `--quiet`

Commands with non-essential progress or status output SHOULD support
`-q, --quiet`.

- Quiet mode MUST suppress non-essential chatter.
- Quiet mode MUST NOT suppress primary command results.
- Quiet mode MUST NOT suppress fatal errors.

## Interactive Input

Commands MUST prompt only when `stdin` is a TTY.

Commands that may prompt SHOULD support `--no-input`.

- If `--no-input` is passed, the command MUST not prompt.
- If required values are missing, the command MUST return a clear error telling
  the user which argument or flag to pass.

Commands MAY offer richer TUI flows when `stdin` and `stdout` are terminals.
Those flows MUST degrade to plain text or fail clearly in non-interactive use.

## Help Text

Top-level help MUST include:

- a short description
- common example commands
- the most important global flags
- how to get command-specific help
- a documentation path
- an issue-reporting path

Subcommand help SHOULD lead with the common workflow and examples before edge
cases.

Running `markata-go` with no arguments MUST display help.

## Error Handling

Commands SHOULD return errors instead of calling `os.Exit()` directly.

Command invocation and usage errors MUST return exit code `2`.

Examples:

- unknown commands
- unknown flags
- conflicting flags
- missing required positional arguments

Error messages SHOULD:

- explain what went wrong
- name the relevant flag, argument, path, or topic when possible
- suggest the next command or action when there is a clear fix

Unexpected diagnostic detail belongs in verbose/debug modes, not normal output.

## Core Command Expectations

- `build` and `serve` print progress and warnings to `stderr`, with summaries and
  explicit results remaining readable in both terminal and redirected use.
- `build` SHOULD include a concise benchmark summary in its final result output,
  including estimated wall-time spent on CPU work, network wait, disk read wait,
  disk write wait, and idle time, plus the slowest lifecycle hotspots.
- When outbound HTTP requests occur during a build, the benchmark summary SHOULD
  also include the slowest requests with enough context to identify the owning
  stage/plugin and remote endpoint without exposing query-string secrets.
- `build` SHOULD offer an explicit machine-readable benchmark mode such as
  `--benchmark-json` for scripting and regression analysis.
- Detailed per-stage benchmark summaries SHOULD be opt-in via verbose mode or a
  dedicated benchmark detail flag, rather than always-on in normal output.
- `config`, `list`, `version`, and `explain` keep their primary output on
  `stdout`.
- `new` and `init` support interactive prompts by default and honor
  `--no-input`.
- `lint` MAY style severities, but the output MUST remain readable without
  color.
