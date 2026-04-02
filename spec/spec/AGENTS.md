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
- a `reference/` directory with quick-lookup reference material
- an `examples/` directory with starter config and template files

The initial required topic files are:

- `configuration.md`
- `writing-frontmatter.md`
- `cli-usage.md`
- `build-deployment.md`
- `faster-builds.md`
- `theme-creation.md`
- `template-management.md`
- `plugin-creation.md`

The initial reference files are:

- `template-context.md`
- `feed-patterns.md`

The initial example files are:

- `fast.toml`
- `markata-go.local.toml`
- `templates/base.html`
- `templates/post.html`
- `templates/feed.html`

The entrypoint SHOULD tell agents to read only the topic files relevant to the current task. Reference and example files SHOULD be used when agents need exact shapes or starter material rather than narrative guidance.

## Skill Content Requirements

The bundled skill MUST guide agents toward:

- inspecting the active config before editing behavior
- preferring content, frontmatter, config, templates, and CSS before custom plugin work
- using built-in CLI inspection commands where possible
- preserving the site's existing layout and conventions unless the task explicitly changes them
- using warm-build comparisons for performance work

## Skill Maintenance Requirement

The bundled site skill is part of markata-go's user-facing product surface for coding agents.

Any change that affects how an agent should work in a markata-go site repository MUST review the bundled skill and update it when needed.

Common maintenance triggers include changes to:

- configuration behavior or recommended config patterns
- frontmatter semantics or content creation workflows
- CLI usage, flags, or command output that agents rely on
- build and deployment workflows
- performance guidance such as `--fast`, benchmarking, or cache behavior
- theme, palette, and aesthetic workflows
- template, layout, and template-context behavior
- plugin authoring constraints or extension workflows

When such a change does not require modifying the bundled skill, the implementation or PR SHOULD explicitly state why no skill change was needed.

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

---

```bash
markata-go agent update [site-path]
```

Required flags:

- `--target`
- `--name`
- `--dry-run`

### `update` behavior

- `site-path` defaults to the current directory.
- The command MUST update the installed skill in place using the currently bundled files.
- `update` MUST behave like `install --force` with the same target and name resolution.
- `--dry-run` MUST report what would be updated without modifying the filesystem.
- Primary results MUST be written to `stdout`.

---

```bash
markata-go agent remove [site-path]
markata-go agent uninstall [site-path]
```

Required flags:

- `--target`
- `--name`

### `remove` behavior

- `site-path` defaults to the current directory.
- The command MUST remove the installed skill directory for the selected target and name.
- `uninstall` MUST be a direct alias for `remove`.
- The command MUST fail clearly when the skill is not installed at the resolved location.
- Primary results MUST be written to `stdout`.

### Install Manifest

After writing skill files, `install` MUST write a manifest file named `.manifest.json` inside the installed skill directory.

The manifest MUST contain:

- `version` — the markata-go binary version string (from `cmd.Version`)
- `installed_at` — RFC 3339 timestamp of installation
- `target` — the install target used (`agents` or `claude`)
- `files` — a map of relative file paths to their SHA-256 hex digests

The manifest MUST NOT be included in the overwrite-protection check. It is always overwritten on install.

The manifest MUST NOT appear in the `--dry-run` output file list or the installed file count reported to the user. It is an internal bookkeeping file.

Example manifest:

```json
{
  "version": "0.5.0",
  "installed_at": "2026-04-01T12:00:00Z",
  "target": "agents",
  "files": {
    "SKILL.md": "a1b2c3...",
    "topics/configuration.md": "d4e5f6..."
  }
}
```

---

```bash
markata-go agent doctor [site-path]
```

Required flags:

- `--target`
- `--name`

### `doctor` behavior

The `doctor` command detects drift between installed skill files and the bundled versions in the current binary.

- `site-path` defaults to the current directory.
- The command MUST locate the installed skill directory using the same target/name resolution as `install`.
- If no manifest file exists, the command MUST report that the skill was installed without a manifest and recommend re-installing with the current binary.
- If the manifest exists, the command MUST compare:
  1. The `version` field against the current binary version.
  2. Each file in the manifest `files` map against the SHA-256 digest of the corresponding bundled file.
  3. Whether new files exist in the bundle that are not present in the manifest.

### `doctor` output

The command MUST report a per-file status using these categories:

| Category | Meaning |
|----------|---------|
| `ok` | File hash matches the bundled version |
| `modified` | File exists on disk but hash differs from the bundled version |
| `new` | File exists in the current bundle but was not present when the skill was installed |
| `missing` | File is listed in the manifest but does not exist on disk |

The command MUST print a summary line:

- If all files are `ok` and no `new` files exist: `Skill is up to date.`
- Otherwise: `Skill has N issue(s). Run 'markata-go agent install --force' to update.`

### `doctor` exit codes

| Code | Meaning |
|------|---------|
| `0` | Skill is up to date (all files `ok`, no `new` files) |
| `1` | Drift detected (any `modified`, `new`, or `missing` files) |
| `2` | Error (skill not installed, manifest unreadable, etc.) |

### `doctor` dry-run and machine-readable output

- `doctor` does not modify the filesystem. It is read-only.
- Future iterations MAY add `--json` for machine-readable output.

## Future Extensibility

The command group is reserved for future additions such as:

- `agent export`
- `agent inspect`
- `agent mcp ...`

Future subcommands MUST be able to reuse the same bundled skill source without requiring a rewrite of the portable `SKILL.md`, `topics/`, `reference/`, or `examples/` content.
