---
title: "Build Lab"
description: "Compare clean and incremental Markata-Go builds in isolated workspaces"
date: 2026-08-26
published: true
tags:
  - documentation
  - build-lab
  - reproducibility
---

# Build Lab

Build Lab compares a baseline build with the experimental serial DAG build. It
runs each build in a separate temporary workspace and writes a structured
result with output manifests and correctness checks. It redirects built-in
Markata-Go cache settings into each workspace, including cache paths supplied
by fixture merge files. It does not load a fixture's `.env` file; pass safe,
non-secret settings with `--env`. The child is bound to the copied workspace;
ambient `MARKATA_GO_SITE_DIR` settings are ignored. Custom plugins and arbitrary
binaries are not sandboxed. Known generated trees such as `output`, `.markata`,
and cache directories are not copied into clean comparison workspaces.

## Run a smoke comparison

Build the binary first. Run the command from the site repository or provide an
explicit fixture path:

```bash
go build -o /tmp/markata-go ./cmd/markata-go
markata-go buildlab run \
  --fixture /path/to/site \
  --baseline /tmp/markata-go \
  --candidate /tmp/markata-go \
  --result buildlab-result.json
```

The candidate receives `--dag` by default. Builds run without `--fast` by
default, so the comparison includes production output steps. The command checks:

- baseline clean output versus candidate clean output;
- candidate incremental output versus candidate clean output; and
- two candidate clean builds for deterministic output.

The default scenario clears cache and output, then performs a priming build and
a no-op incremental build. The command exits unsuccessfully when a build fails
or a correctness check fails.
The JSON result is printed to standard output and is also written to
`--result`, when provided.

Use `--fast` for a quicker development smoke check. Fast mode skips production
output steps and is not a substitute for a full Build Lab comparison.

When the candidate also contains intentional output changes, use the same
candidate binary for `--baseline` and `--candidate` to isolate legacy versus
serial-DAG behavior. This is a DAG compatibility check, not a replacement for
reviewing the candidate's output against a historical baseline.

Use the global `--merge-config` flag to apply a fixture-local profile such as
`fast.toml` to both binaries:

```bash
markata-go buildlab run --fixture /path/to/site --merge-config fast.toml
```

## Use a scenario file

Pass a versioned JSON scenario to exercise mutations:

```json
{
  "id": "title-change",
  "version": "1",
  "seed": 42,
  "operations": [
    {"type": "build"},
    {
      "type": "replace-exact",
      "path": "content/foo.md",
      "old": "title: Old",
      "new": "title: New"
    },
    {"type": "build"}
  ]
}
```

Run it with:

```bash
markata-go buildlab run \
  --fixture /path/to/site \
  --baseline /tmp/markata-go \
  --candidate /tmp/markata-go \
  --scenario title-change.json \
  --seed 42 \
  --result buildlab-title-change.json
```

`replace-exact` must match exactly one occurrence. Paths must remain inside the
fixture. Build Lab supports `build`, `write-file`, `replace-exact`,
`delete-file`, `rename-file`, `copy-file`, `set-config`, `touch-file`,
`clear-cache`, and `clear-output` operations. `clear-output` uses the effective
candidate output directory, including a configured non-default path.

## Output classes

Manifest records are deterministic by default. Known minification and fontpack
metadata files are excluded because they are build state, not publishable
output. Use `--volatile` for known changing publishable paths, such as the
default `.well-known/time` file:

```bash
markata-go buildlab run --fixture /path/to/site --volatile .well-known/time
```

Use `--check-determinism=false` only when a fixture cannot support the clean
rebuild check. Keep that exception visible in CI configuration.

## Candidate scheduling

The serial DAG always uses one worker. When the graph has independent ready
tasks, you can test different legal orders while retaining reproducibility by
enabling seeded randomization. The current compatibility spine is mostly a
single lifecycle chain, so this flag is a no-op where no task choices exist:

```bash
markata-go buildlab run \
  --fixture /path/to/site \
  --candidate-dag-random-ready \
  --seed 42
```

Use `--timeout` and `--gomaxprocs` to control isolated process limits. Build Lab
pins subprocesses to UTC, the C.UTF-8 locale, and `SOURCE_DATE_EPOCH=0`.
Record external tool versions with repeated comma-separated entries:

```bash
markata-go buildlab run --fixture /path/to/site --tool-version pagefind=1.5.2,tailwind=4.1.0
```

Pass non-secret build settings with `--env`. For example, disable encryption
when a benchmark fixture has private source files but no benchmark key:

```bash
markata-go buildlab run --fixture /path/to/site --env MARKATA_GO_ENCRYPTION_ENABLED=false
```

Build Lab does not pass secret-looking environment names to child processes.
