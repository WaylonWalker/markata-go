#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
chart="$repo_root/helm-chart"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

helm template portable-defaults "$chart" >"$tmp_dir/default.yaml"

if grep -q 'storageClassName:' "$tmp_dir/default.yaml"; then
  fail "default render emitted storageClassName instead of using the cluster default"
fi
if grep -q 'global-regcred' "$tmp_dir/default.yaml"; then
  fail "default render still references the homelab registry secret"
fi

helm template portable-defaults "$chart" \
  --set-string storage.source.storageClassName=longhorn \
  --set-string storage.site.storageClassName=longhorn \
  --set-string storage.search.storageClassName=longhorn \
  --set storage.cache.enabled=true \
  --set-string storage.cache.storageClassName=longhorn \
  --set-string 'imagePullSecrets[0].name=custom-regcred' \
  >"$tmp_dir/explicit.yaml"

storage_classes="$(grep -c 'storageClassName: "longhorn"' "$tmp_dir/explicit.yaml" || true)"
[[ "$storage_classes" == "4" ]] ||
  fail "explicit storage classes were not rendered for all four PVCs (got $storage_classes)"
grep -q 'name: custom-regcred' "$tmp_dir/explicit.yaml" ||
  fail "explicit imagePullSecrets value was not rendered"

printf 'Portable Helm defaults tests passed.\n'
