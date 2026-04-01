# CLI Usage

Use this topic for everyday site work and safe project inspection.

## High-Value Commands

- `markata-go build`
- `markata-go build --clean`
- `markata-go build --fast`
- `markata-go build --benchmark-json benchmark.json`
- `markata-go serve`
- `markata-go serve --fast`
- `markata-go config show`
- `markata-go config get <key>`
- `markata-go list posts`
- `markata-go list feeds`
- `markata-go list tags`
- `markata-go new`
- `markata-go explain`
- `markata-go explain config`
- `markata-go explain templates`
- `markata-go explain plugins`
- `markata-go explain agents`

## What To Use When

- inspect content inventory: `markata-go list posts`
- inspect feed definitions and sizes: `markata-go list feeds`
- inspect resolved configuration: `markata-go config show`
- create new content: `markata-go new`
- interactive local editing: `markata-go serve --fast`
- final verification: `markata-go build`

## Guidance

- Prefer `list` commands when you need structured inspection.
- Prefer `explain` when you need command-specific or subsystem context.
- Prefer `serve` for interactive local work and `build` for validation or CI-like runs.
- Use `--verbose` only when normal output is not enough.
- Keep primary results script-friendly by using built-in machine-readable output when available.
