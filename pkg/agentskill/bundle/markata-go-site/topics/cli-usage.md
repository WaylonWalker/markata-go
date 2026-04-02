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

## Global Flags Agents Should Know

- `-c`, `--config`: use a specific config file
- `-m`, `--merge-config`: merge override configs such as `fast.toml`
- `-o`, `--output`: override the output directory without editing config
- `-v`, `--verbose`: show detailed logs and plugin-stage hints
- `-q`, `--quiet`: suppress non-essential progress output
- `--no-input`: disable prompts for scripted or non-interactive runs

Examples:

```bash
markata-go build -c markata-go.toml
markata-go serve -m fast.toml
markata-go build -o dist
markata-go new "My Post" --no-input
markata-go build -v
```

## What To Use When

- inspect content inventory: `markata-go list posts`
- inspect feed definitions and sizes: `markata-go list feeds`
- inspect resolved configuration: `markata-go config show`
- create new content: `markata-go new`
- interactive local editing: `markata-go serve --fast`
- final verification: `markata-go build`

## Operator Patterns

- use `-m fast.toml` when you want a lighter dev build without rewriting main config
- use `-c` when a repo has multiple configs or examples and you need the exact active one
- use `-o dist` in CI or preview contexts when you want a temporary artifact path
- use `--no-input` for automation or when the agent must avoid prompts
- use `-v` when debugging plugin order, missing outputs, or config resolution issues

## Guidance

- Prefer `list` commands when you need structured inspection.
- Prefer `explain` when you need command-specific or subsystem context.
- Prefer `serve` for interactive local work and `build` for validation or CI-like runs.
- Use `--verbose` only when normal output is not enough.
- Keep primary results script-friendly by using built-in machine-readable output when available.
- Prefer merged config overrides over editing the main config for temporary local or CI changes.
