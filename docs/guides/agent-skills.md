---
title: "Agent Skills"
description: "Install the bundled markata-go site skill so coding agents can work with your site using focused topic guidance"
date: 2026-04-01
published: true
tags:
  - documentation
  - agents
  - cli
---

# Agent Skills

markata-go can install a bundled skill into your site repository so coding agents have project-specific guidance for common markata-go tasks.

## Install

Install the portable skill layout into the current site:

```bash
markata-go agent install
```

This creates:

```text
.agents/skills/markata-go-site/
```

Install into Claude Code's skill layout instead:

```bash
markata-go agent install --target claude
```

This creates:

```text
.claude/skills/markata-go-site/
```

## Preview Without Writing

```bash
markata-go agent install --dry-run
```

## Overwrite An Existing Installed Skill

```bash
markata-go agent install --force
```

## Installed Layout

The skill is split into an entrypoint plus focused topic files:

```text
SKILL.md
topics/
  configuration.md
  writing-frontmatter.md
  cli-usage.md
  build-deployment.md
  faster-builds.md
  theme-creation.md
  template-management.md
  plugin-creation.md
```

This keeps the main skill small while giving agents a place to read deeper guidance only when the task needs it.

## What The Skill Covers

- configuration and config inspection
- writing content and frontmatter
- everyday CLI usage
- build and deployment strategy
- faster local build loops
- theme and palette work
- template overrides and layout changes
- deciding when plugin work is actually necessary

## Recommended Workflow For Agents

1. Inspect active config with `markata-go config show`.
2. Inspect site content with `markata-go list posts`, `markata-go list feeds`, or `markata-go list tags`.
3. Use `markata-go explain <topic>` for built-in command context.
4. Iterate with `markata-go build --fast` or `markata-go serve --fast`.
5. Reach for plugin work only when config, frontmatter, templates, and CSS are not enough.

## Why The Command Is Named `agent`

The `agent` command group is intentionally generic.

Today it installs the bundled skill. Later it can grow additional subcommands for export workflows or MCP-oriented integrations without changing the skill format that site repositories already use.
