# Agent Integrations Specification

This document specifies how markata-go distributes agent-facing project guidance for site repositories.

## Goals

- Provide a bundled, installable skill for agents working on markata-go sites.
- Keep the skill portable across agent tools.
- Split guidance into focused topics instead of a single monolithic prompt.
- Leave room in the CLI for future tool-specific exports and MCP-backed integrations.

## Bundled Skill

Markata-go ships a bundled skill named `markata-go-site`.

The bundled skill MUST include:

- `SKILL.md` as the portable entrypoint
- a `topics/` directory with focused guidance files

The initial required topic files are:

- `configuration.md`
- `writing-frontmatter.md`
- `cli-usage.md`
- `build-deployment.md`
- `faster-builds.md`
- `theme-creation.md`
- `template-management.md`
- `plugin-creation.md`

The entrypoint SHOULD tell agents to read only the topic files relevant to the current task.

## Skill Content Requirements

The bundled skill MUST guide agents toward:

- inspecting the active config before editing behavior
- preferring content, frontmatter, config, templates, and CSS before custom plugin work
- using built-in CLI inspection commands where possible
- preserving the site's existing layout and conventions unless the task explicitly changes them
- using warm-build comparisons for performance work

## Installation Targets

The CLI MUST support installing the bundled skill into target-specific directory layouts.

Initial targets:

1. `agents`
Path:
`.agents/skills/<skill-name>/`

2. `claude`
Path:
`.claude/skills/<skill-name>/`

The installed file contents MUST remain the same across targets unless a future target explicitly requires generated wrappers.

## CLI

Markata-go MUST expose an `agent` command group.

Initial command:

```bash
markata-go agent install [site-path]
```

Required flags:

- `--target`
- `--name`
- `--force`
- `--dry-run`

### `install` behavior

- `site-path` defaults to the current directory.
- The command MUST install bundled files into the selected target layout.
- The command MUST fail clearly when destination files already exist and `--force` is not set.
- `--dry-run` MUST report what would be written without modifying the filesystem.
- Primary results MUST be written to `stdout`.
- Errors MUST suggest `--force` when overwrite conflicts are the only blocker.

## Future Extensibility

The command group is reserved for future additions such as:

- `agent export`
- `agent inspect`
- `agent mcp ...`

Future subcommands MUST be able to reuse the same bundled skill source without requiring a rewrite of the portable `SKILL.md` and `topics/` content.
