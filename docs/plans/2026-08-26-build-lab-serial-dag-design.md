---
title: "Build Lab and Serial DAG Design"
description: "Design for reproducible Markata-Go build experiments and the first serial task graph."
date: 2026-08-26
published: true
tags:
  - architecture
  - build-lab
  - dag
---

# Build Lab and Serial DAG

## Goal

Markata-Go needs a correctness oracle before it needs a faster scheduler.  The
Build Lab therefore runs real binaries in isolated workspaces, records complete
output manifests, and compares clean and incremental builds.  The first graph
executor remains serial.  It makes dependencies and artifacts explicit without
changing the current lifecycle contract.

## Chosen approach

The change uses two small packages:

* `pkg/buildlab` owns isolated subprocesses, declarative mutations, generated
  fixtures, canonical output manifests, and machine-readable results.
* `pkg/builddag` owns task and artifact identifiers, graph compilation,
  validation, invalidation, legacy hook adapters, and serial execution.

The packages do not share mutable runtime state.  A Build Lab workspace is a
copy of the fixture and receives its own output, cache, home, XDG cache, and
temporary directories.  The runner sets `cwd` and the environment on the
subprocess instead of changing the parent process directory.  A process group
is killed on timeout so child tools cannot outlive a run.

The manifest contains every regular output and symlink, with normalized
project-relative paths, streamed SHA-256 hashes, size, mode, type, and an
explicit output class.  Unknown paths are deterministic.  Missing and extra
paths always fail.  Deterministic and semantic outputs compare hashes unless a
registered semantic comparator says otherwise.  Secure-nondeterministic and
volatile outputs still require the same path/type contract, but do not require
equal bytes.  The policy is encoded in the manifest and result, not in an
ad-hoc ignore list.

## Graph model

`TaskSpec` declares an ID, metadata group, scope, required artifacts, and
provided artifacts.  The compiler rejects duplicate task IDs, duplicate
providers, missing providers, and cycles.  External input artifacts must be
declared explicitly.  It emits a stable sorted representation and stable
topological order.  `TaskResult` returns artifacts and dynamic dependency IDs;
the executor persists those dependencies in `ExecutionState` so restoring a
cached result also restores its invalidation edges.

The executor accepts a ready-task ordering seed for deterministic serial
schedule testing.  It always executes with `MaxParallel=1` in this milestone.
Legacy lifecycle hooks are represented as exclusive tasks and are ordered by
their existing stage priority and registration order.  Native tasks can be
added beside them, but no native task is allowed to assume concurrent mutation
of `*models.Post`.

## Migration boundary

The initial native slice is exposed as an opt-in build mode.  It executes the
existing configure/validate lifecycle, then expresses glob, load, title
derivation, index construction, wikilinks, and Markdown rendering as named
serial tasks.  The remaining output lifecycle continues through the legacy
adapter.  This keeps the default build behavior unchanged while allowing the
Build Lab to compare legacy and candidate binaries and to prove incremental
equivalence before a default switch.

## Verification

Unit tests cover manifest canonicalization and differences, workspace isolation,
scenario preconditions, seeded generation, graph validation, stable ordering,
dynamic dependency persistence, invalidation closure, and legacy ordering.
Integration tests use the checked-in CLI fixture at
`cmd/markata-go/cmd/testdata/dag-site/`.  The real site at `waylonwalker.com`
is a manual smoke/benchmark fixture, not a checked-in test dependency.  Build
Lab reproducibility timestamps must not change current-time date plausibility
rules.  No broad DAG concurrency is enabled until clean output, incremental
output, and multiple legal serial schedules agree.
