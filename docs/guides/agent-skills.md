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

markata-go can install a bundled skill into your site repository or an agent-specific global skill directory so coding agents have project-specific guidance for common markata-go tasks.

## Install

See the supported agent ids and install paths directly in the CLI:

```bash
markata-go agent list-agents
```

Install into the current agent's project layout:

```bash
markata-go agent install
```

When markata-go can detect the current agent from the environment, it uses that agent's project path. Otherwise it falls back to the portable `universal` layout:

```text
.agents/skills/markata-go-site/
```

Install into Claude Code's project layout explicitly:

```bash
markata-go agent install --agent claude-code
```

This creates:

```text
.claude/skills/markata-go-site/
```

Install into OpenCode's global skill directory instead of the current repository:

```bash
markata-go agent install --agent opencode -g
```

This creates:

```text
~/.config/opencode/skills/markata-go-site/
```

`-g` / `--global` always requires an explicit `--agent` so markata-go can choose the right user directory.

## Preview Without Writing

```bash
markata-go agent install --dry-run
```

## Overwrite An Existing Installed Skill

```bash
markata-go agent install --force
```

## Update The Installed Skill

Use `update` as the friendlier wrapper around reinstalling with overwrite:

```bash
markata-go agent update
```

Preview the update without writing files:

```bash
markata-go agent update --dry-run
```

## Check For Drift

After upgrading the markata-go binary, the bundled skill may have new or updated files. Use `doctor` to check:

```bash
markata-go agent doctor
```

Example output when the skill is current:

```text
Skill:     markata-go-site
Agent:     universal
Scope:     project
Location:  .agents/skills/markata-go-site/
Installed: 0.5.0
Current:   0.5.0

  ok        SKILL.md
  ok        topics/configuration.md
  ...

Skill is up to date.
```

Example output when drift is detected:

```text
Skill:     markata-go-site
Agent:     universal
Scope:     project
Location:  .agents/skills/markata-go-site/
Installed: 0.4.0
Current:   0.5.0

  ok        SKILL.md
  modified  topics/configuration.md
  new       reference/new-reference.md

Skill has 2 issue(s). Run 'markata-go agent install --force' to update.
```

File statuses:

| Status | Meaning |
|--------|---------|
| `ok` | File matches the bundled version |
| `modified` | File content differs from the bundled version |
| `new` | File was added to the bundle since last install |
| `missing` | File was in the bundle at install time but no longer is |

If the skill was installed before manifest support was added, `doctor` will recommend re-installing with `--force`.

## Remove The Installed Skill

Remove the installed skill directory:

```bash
markata-go agent remove
```

`uninstall` is an alias:

```bash
markata-go agent uninstall
```

You can also target Claude Code's layout explicitly:

```bash
markata-go agent remove --agent claude-code
```

Remove from a global agent directory:

```bash
markata-go agent remove --agent opencode -g
```

## Installed Layout

The skill is split into an entrypoint, focused topic files, reference material, and starter examples:

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
reference/
  template-context.md
  feed-patterns.md
  palette-reference.md
examples/
  fast.toml
  markata-go.local.toml
  palettes/
    my-brand.toml
  templates/
    base.html
    post.html
    feed.html
evals/
  evals.json
```

Topic files provide narrative guidance for common tasks. Reference files give quick-lookup shapes (template variables, feed config patterns). Example files provide starter configs and templates for new sites or agents that need a concrete starting point. `evals/evals.json` provides a starter regression prompt set for reviewing changes to the bundled skill itself.

## What The Skill Covers

- configuration and config inspection
- writing content and frontmatter
- everyday CLI usage
- build and deployment strategy
- faster local build loops
- theme and palette work
- template overrides and layout changes
- deciding when plugin work is actually necessary
- template variable reference for post, feed, and base templates
- feed config patterns and common feed recipes
- starter config and template files for new sites

## Supported Agents

`markata-go agent` mirrors the same agent identifiers as `vercel-labs/skills`, including `opencode`, `claude-code`, `codex`, `cursor`, `gemini-cli`, `qwen-code`, `warp`, `windsurf`, `github-copilot`, and the rest of that compatibility matrix.

Use `markata-go agent list-agents` to print the full supported list with project and global install paths.

For project installs, omitting `--agent` uses the current agent when markata-go can detect it from the environment. If no agent is detected, markata-go falls back to `universal`, which installs to `.agents/skills/markata-go-site/`.

## Recommended Workflow For Agents

1. Inspect active config with `markata-go config show`.
2. Inspect site content with `markata-go list posts`, `markata-go list feeds`, or `markata-go list tags`.
3. Use `markata-go explain <topic>` for built-in command context.
4. Iterate with `markata-go build --fast` or `markata-go serve --fast`.
5. Reach for plugin work only when config, frontmatter, templates, and CSS are not enough.

## Why The Command Is Named `agent`

The `agent` command group is intentionally generic.

Today it installs the bundled skill. Later it can grow additional subcommands for export workflows or MCP-oriented integrations without changing the skill format that site repositories already use.
