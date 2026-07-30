# CLI Site Directory Specification

## Overview

`markata-go` can operate on a site other than the caller's current working
directory. This supports stable shell aliases and agent automation that manage
multiple sites from any directory.

```bash
markata-go --site-dir ~/sites/blog search "release process"
MARKATA_GO_SITE_DIR=~/notes/work markata-go new "Meeting notes" --no-input
```

## Site Directory Selection

The global `--site-dir <path>` flag selects the active site directory. The
`MARKATA_GO_SITE_DIR` environment variable provides the same capability for
shell aliases and wrappers.

Selection precedence is:

1. `--site-dir`
2. `MARKATA_GO_SITE_DIR`
3. the process current working directory

An explicitly selected path MUST exist and be a directory. Relative paths are
resolved from the caller's working directory before markata-go starts site
operations. Once selected, the directory is the operational working directory
for the command.

## Path Resolution

With an explicit site directory, markata-go MUST resolve all caller-relative
site operations from that directory, including:

- configuration discovery and `.env` loading
- relative `--config` and `--merge-config` paths
- content template discovery and content creation
- content globbing and linting
- `.markata` build, list, and search caches

Absolute paths remain unchanged. Existing commands without an explicit site
directory retain their current working-directory behavior.

`builder-admin` reserves `--release-dir` for its release-serving directory so
the global `--site-dir` flag retains one meaning across commands.

## Content Path Output

When `--site-dir` or `MARKATA_GO_SITE_DIR` selects a site, `list` and `search`
MUST report Markdown source paths as absolute paths in every output format.
This includes table, JSON, CSV, and `--format path` output. The absolute paths
allow agents to open or edit a result without separately resolving the site
root.

When no site directory is explicitly selected, these commands retain their
existing site-relative path output.

`new` MUST print the absolute path of the created content file when a site
directory is explicitly selected.

## Agent Content Workflow

The `explain content` topic MUST document how agents inspect, search, create,
and edit content:

1. inspect configuration with `config show`
2. list or search posts with machine-readable or path output
3. edit the returned Markdown source path directly
4. create new content with `new --no-input`
5. validate with `lint` and `build --fast`

The bundled site skill MUST point agents to this topic and explain that global
skill installation is appropriate when agents begin outside a site repository.
