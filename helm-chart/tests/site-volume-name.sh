#!/usr/bin/env bash
set -euo pipefail

chart="$(dirname "$0")/.."
default_render="$(helm template site-volume-test "$chart" --show-only templates/pvcs.yaml)"
bound_render="$(helm template site-volume-test "$chart" --show-only templates/pvcs.yaml --set storage.site.volumeName=pv-fast-site)"

if grep -q 'volumeName:' <<<"$default_render"; then
  echo "Default PVC unexpectedly pins a volume" >&2
  exit 1
fi
grep -q 'volumeName: "pv-fast-site"' <<<"$bound_render"
