# Build Lab Specification

## Purpose

Build Lab is a reproducibility and incremental-correctness harness for
Markata-Go builds. It compares real command-line builds in isolated workspaces.
It is an independent diagnostic tool. It MUST NOT depend on a task graph or
change the behavior of the ordinary `markata-go build` command.

## Command

The CLI MUST expose:

```text
markata-go buildlab run --fixture PATH
```

The command MAY accept explicit baseline and candidate binaries. If either is
omitted, it MUST use the current executable. Baseline and candidate MUST
receive equivalent ordinary build arguments. Build Lab MUST NOT add `--dag`,
`--dag-seed`, or another experimental scheduler flag.

The command MUST emit one versioned JSON result to standard output. `--result`
MAY write the same result to a file. A failed process or failed required
comparison MUST produce a non-zero exit status.

Setup failures after the command is accepted, including invalid fixture,
binary, scenario, environment, or output configuration, MUST emit a versioned
result with `failure_class: "harness"` and a diagnostic before returning a
non-zero exit status.

## Isolation

For every comparison, Build Lab MUST:

1. copy the fixture into a temporary workspace without changing the fixture;
2. run the child with the copied site as its working directory;
3. redirect HOME, temporary directories, cache directories, locale, timezone,
   and `SOURCE_DATE_EPOCH` into deterministic workspace-local values;
4. disable automatic fixture `.env` loading;
5. reject fixture paths and output paths that escape the workspace; and
6. remove the workspace after the result is captured.

Secret-looking environment variables MUST NOT be inherited or accepted through
the non-secret `--env` option. The `--env` option MUST accept only `PATH`,
`MARKATA_GO_ENCRYPTION_ENABLED`, and `MARKATA_GO_OFFLINE`. Build Lab provides
workspace separation and process-lifecycle cleanup, not an operating-system
sandbox. The command, custom plugins, external binaries, and fixture content
MUST be trusted. Process groups and Windows Job Objects MUST be documented as
best-effort descendant cleanup; they do not restrict filesystem, network,
credentials, or host-process access.

## Scenarios

A scenario is versioned JSON containing an identifier, seed, and ordered
operations. Supported operations include `clean-cache`, `clear-output`,
`build`, `write-file`, `replace-exact`, `delete-file`, `rename-file`,
`copy-file`, `set-config`, and `touch-file`. Mutation paths MUST remain inside
the fixture. `replace-exact` MUST match exactly one occurrence. A semantic
mutation MUST be followed by a build.

The foundation MUST support clean builds, no-op incremental builds, edits,
source additions, source deletions, source renames, linked-target mutations,
and configuration mutations. A fixture MAY provide additional scenario files
for templates, assets, feeds, or other product-specific behavior.

Each build checkpoint MUST capture:

- a clean baseline result;
- a clean candidate result;
- a candidate incremental result when a prior build exists; and
- a second clean candidate result when determinism checking is enabled.

## Output comparison

The harness MUST collect sorted, path-relative manifests without following
symlinks. Records include path, type, mode, size, bytes digest, and output
class. Deterministic output compares all record fields. Semantic output MAY
use a registered comparator. Secure-nondeterministic and explicitly volatile
output compare presence and type but not bytes.

Known build metadata and fixture cache/output directories MUST be excluded from
fixture copies and fixture identity digests.

## Failure reporting

The result MUST distinguish:

- `product`: a child build exits non-zero or a required output comparison
  differs; and
- `harness`: Build Lab cannot establish a trustworthy comparison, including
  path violations, process timeouts, stream truncation, unsupported process
  containment, or manifest failures.

Each failure MUST include a machine-readable observation or diagnostic. The
CLI SHOULD also print concise diagnostic lines to standard error while keeping
standard output valid JSON. A product failure is evidence from the instrument;
it MUST NOT cause the foundation change to silently modify ordinary build
semantics.

## Process limits

Child stdout and stderr MUST be bounded. Truncation MUST be visible in the
structured observation and MUST make the run fail. Timeouts MUST terminate the
child and descendants that remain in the managed process group or Job Object
where the host platform supports that mechanism. Unsupported platforms MUST
fail clearly rather than silently leaking descendants.

## Reproducibility inputs

The result MUST record binary identity, fixture digest, scenario, timezone,
locale, source-date epoch, GOMAXPROCS, Go version, and declared external tool
versions. Build Lab supplies `SOURCE_DATE_EPOCH` as the deterministic build-time
input. Time-dependent production output that does not consume the build clock
is a product failure to fix in a follow-up change, not a reason to weaken the
comparison.
