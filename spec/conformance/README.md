# Spec conformance

This directory is the machine-readable bridge between the normative specification, the Go implementation, tests, and user documentation.

Tracked by #1493.

## Requirements manifest

`requirements.yaml` assigns stable IDs to normative requirements and records their current conformance state.

Each entry includes:

- `id` — stable `DOMAIN-NNN` identifier such as `CORE-001`
- `level` — `MUST`, `SHOULD`, or `MAY`
- `scope` — `portable` or `markata-go`
- `summary` — concise behavioral requirement
- `spec` — authoritative specification file
- `implementation` — concrete source files implementing the behavior
- `tests` — concrete test files exercising the behavior
- `status` — `implemented`, `partial`, `planned`, or `not-applicable`
- `notes` — optional explanation for intentional deviations or incomplete work

The initial manifest is intentionally small. It seeds the core architecture and known plugin-model drift so later PRs can expand the matrix without first inventing a tracking format.

## CI enforcement

`pkg/specconformance/conformance_test.go` is part of the normal Go test suite. It rejects:

- malformed or duplicate requirement IDs
- invalid levels, scopes, or statuses
- missing spec/implementation/test file references
- `implemented` requirements without both implementation and test references

This keeps the first version dependency-free from a CI configuration perspective: existing `go test ./...` coverage enforces the manifest automatically.

## Rollout

The intended order is:

1. establish stable IDs and linting
2. enumerate existing MUST/SHOULD requirements
3. separate portable contracts from markata-go-specific product contracts
4. reconcile spec/API drift
5. make `spec/spec/tests.yaml` directly executable
6. generate a human-readable conformance report from the same metadata

Until enumeration is complete, absence from `requirements.yaml` does **not** imply that a requirement is intentionally unsupported.
