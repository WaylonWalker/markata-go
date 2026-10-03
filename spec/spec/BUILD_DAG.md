# Build DAG Core Contract

Tracking: #1451 (task ownership) and #1454 (graph/resource diagnostics).
User guidance: `docs/architecture/builddag-plugin-ownership.md`.

## Declarations and compilation

`Builder` collects task declarations and explicit external artifacts. `Compile`
MUST snapshot each task's `Requires`, `Provides`, and `Resources` backing arrays,
as well as the external artifact set. Later mutation of declarations, builder
inputs, or metadata returned by `Graph.Task` MUST NOT change the compiled graph's
order, serialization, digest, reports, resource claims, or execution.

`Graph.Task` MUST return independently owned metadata slices, preserving nil
versus non-nil empty slices. Scalar metadata and the function value are copied.
This contract does not freeze mutable state captured by task functions and does
not permit concurrent mutation of a builder during compilation. Internal readers,
including the executor, SHOULD read compiled metadata directly without allocating
defensive copies per task.

Compilation MUST retain existing validation: nonempty unique task IDs, nonempty
artifact kind/key, unique providers, declared external inputs for missing
providers, valid resource kinds/access, no duplicate resource identity within a
task, and no exclusive/parallel-safe conflict. Parallel-safe tasks MUST declare
resources. An internal provider takes precedence over an external declaration.
Scope is descriptive metadata, not an inferred dependency or ownership claim.
Unknown ownership and legacy tasks remain conservative; empty claims are valid
unless the task declares itself parallel-safe.

## Deterministic order and identity

At each topological step, the lexicographically smallest currently ready task ID
MUST be chosen. Declaration order and dependency-map iteration MUST NOT affect
the result. Compilation SHOULD use reverse adjacency and a lexicographic min-heap,
avoiding scans of every task after each completion and repeated full ready sorts.
For V tasks and E unique dependency edges, topological ordering SHOULD take
O(E + V log V) time and O(V + E) space.

Cycles MUST fail compilation with stable diagnostics: visit task IDs and each
task's dependencies lexicographically, returning the first encountered cycle
with its starting task repeated at the end. Missing providers and validation
errors MUST remain errors, not suppressed or replaced with partial graphs.

Artifact identity is the structured tuple `(Kind, Key)`. Canonical artifact
ordering MUST compare Kind, then Key, lexicographically, including external
artifacts. Human-readable `String()` forms are not identity or sorting keys:
`{Kind: "post", Key: "rendered:example"}` and
`{Kind: "post:rendered", Key: "example"}` are distinct accepted artifacts.

Resource claims MUST be ordered lexicographically by the structured tuple
`(Resource.Kind, Resource.Key, Access)`. Resource validation remains unchanged;
opaque keys may contain delimiters. Serialization and reports MUST share these
comparators, independently of input slice and map order.

Serialization excludes executable functions and omits empty optional metadata
as before. Reports explicitly emit empty artifact/resource lists as `[]`, retain
their existing schema/version and bounds, and sort full values before truncation.
Digest is SHA-256 of canonical serialized declarations, not a content cache key;
it MUST NOT be used to skip execution or infer unchanged task inputs.

## Execution and verification

The executor MUST remain serial (`MaxParallel=1`, with constructor zero defaulting
to one). It MUST check cancellation, artifact availability, and task functions,
and return task errors with context. No concurrency or scheduler-default change
is implied by graph hardening.

For scheduler-owned Load, Transform, Render, Collect, Write, and Cleanup stages,
the DAG build MUST snapshot the ordered runnable plugin list once at stage start.
It MUST execute a stage-start graph, then compile and execute each plugin graph
only after its predecessor finishes, then execute a stage-complete graph. Each
plugin graph MUST declare the preceding completion artifact as an external
input. A plugin registered during a stage MUST NOT join that stage's snapshot;
one registered before a later stage starts MAY join the later snapshot. Plugin
expansion MUST preserve the compatibility task's required and provided boundary
artifacts. The stage observer, template-cache clearing, and stage timing MUST
remain at lifecycle-stage boundaries. The legacy executor remains the default,
and each compatibility plugin graph remains serial and exclusive.

Regression coverage MUST mutate original and accessor backing arrays, mutate
builder external inputs after compilation, and verify unchanged graph metadata,
diagnostics, identity, and execution. Tests MUST cover ambiguous display strings,
reversed declarations, repeated map-backed serialization, fan-out/fan-in and
disconnected ordering, and stable cycle diagnostics. Compiler benchmarks MUST
include chain and wide graphs of at least 1,000, 4,000, and 16,000 tasks, report
allocations, and keep declaration construction outside the measured compile loop.
