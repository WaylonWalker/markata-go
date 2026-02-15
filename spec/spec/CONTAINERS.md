# Container Images

markata-go provides official container images for two primary use cases: minimal runtime execution and build/publish workflows.

## Image Names

- Minimal image: `ghcr.io/waylonwalker/markata-go:<version>`
- Builder image: `ghcr.io/waylonwalker/markata-go-builder:<version>`

Both images MUST publish a matching `:latest` tag for the most recent stable release.

## Minimal Image

The minimal image is intended for running a single `markata-go` command with the smallest footprint.

### Requirements

- Base image MUST be `scratch` or a distroless equivalent.
- Include the `markata-go` binary at `/usr/local/bin/markata-go`.
- Include TLS roots at `/etc/ssl/certs/ca-certificates.crt`.
- Default entrypoint MUST be `markata-go`.
- Default command SHOULD be `--help`.
- Working directory MUST be `/site`.
- Default user SHOULD be a non-root UID/GID (e.g., `65532:65532`).

## Builder Image

The builder image is intended for CI pipelines, sidecar builders, and scripts that need shell tooling.

### Required Tools

The builder image MUST include:

- `markata-go` binary
- POSIX shell at `/bin/sh`
- Filesystem utilities: `find`, `sort`, `xargs`, `sha256sum`, `awk`
- Core utilities: `date`, `ln`, `rm`, `mkdir`, `cp`, `mv`, `sleep`
- `rsync` for publish/sync workflows
- `ca-certificates` for HTTPS access

### Optional Tools

The builder image MAY include:

- `git`
- `libavif-apps` (provides `avifenc`)
- `libwebp-tools` (provides `cwebp`)
- `openssh-client` (for rsync over SSH)
- `tzdata`

### Behavior

- The builder image MUST allow running shell scripts directly.
- `docker run --rm <builder-image> sh -c 'markata-go --help'` MUST succeed.
- `docker run --rm <builder-image> sh -c 'rsync --version'` MUST succeed.

## Tagging and Versioning

- All container tags MUST follow semantic versioning.
- The builder and minimal images MUST use the same tag for a given release.
- `:latest` MUST point to the most recent stable release.
