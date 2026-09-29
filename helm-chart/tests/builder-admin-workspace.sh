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

common=(
  --show-only templates/builder-admin.yaml
  --set builderAdmin.enabled=true
  --set builderAdmin.ingress.enabled=true
  --set builderAdmin.ingress.auth.enabled=true
  --set-string builderAdmin.ingress.auth.internalUrl=http://hlab-auth.default.svc.cluster.local:8000
  --set-string builderAdmin.ingress.host=builder.example.com
  --set-string builderAdmin.ingress.ingressClassName=traefik
  --set builderAdmin.ingress.tls.enabled=true
  --set-string builderAdmin.ingress.tls.secretName=builder-tls
  --set builderAdmin.networkPolicy.enabled=true
  --set-string 'builderAdmin.auth.trustedProxyCIDRs[0]=10.42.0.0/16'
)

helm template workspace-test "$chart" "${common[@]}" >"$tmp_dir/default.yaml"
if grep -q -- '--work-dir' "$tmp_dir/default.yaml"; then
  fail "disabled workspace emitted --work-dir"
fi
if grep -q 'name: workspace' "$tmp_dir/default.yaml"; then
  fail "disabled workspace emitted a workspace mount or volume"
fi

helm template workspace-test "$chart" "${common[@]}" \
  --set builderAdmin.workspace.enabled=true \
  >"$tmp_dir/emptydir.yaml"

grep -q -- '--work-dir' "$tmp_dir/emptydir.yaml" || fail "emptyDir workspace omitted --work-dir"
grep -q '/data/work/build' "$tmp_dir/emptydir.yaml" || fail "emptyDir workspace omitted configured workDir"
grep -q 'mountPath: /data/work' "$tmp_dir/emptydir.yaml" || grep -q 'mountPath: "/data/work"' "$tmp_dir/emptydir.yaml" || fail "emptyDir workspace omitted mountPath"
grep -q 'name: workspace' "$tmp_dir/emptydir.yaml" || fail "emptyDir workspace volume was not rendered"
grep -q 'emptyDir:' "$tmp_dir/emptydir.yaml" || fail "emptyDir workspace did not render an emptyDir volume"

grep -q 'name: site' "$tmp_dir/emptydir.yaml" || fail "workspace render lost the durable site volume"
grep -q 'mountPath: /data/site' "$tmp_dir/emptydir.yaml" || grep -q 'mountPath: "/data/site"' "$tmp_dir/emptydir.yaml" || fail "workspace render lost the durable site mount"

helm template workspace-test "$chart" "${common[@]}" \
  --set builderAdmin.workspace.enabled=true \
  --set builderAdmin.workspace.mode=hostPath \
  --set-string builderAdmin.workspace.hostPath.path=/var/lib/markata-work \
  >"$tmp_dir/hostpath.yaml"

grep -q '/var/lib/markata-work' "$tmp_dir/hostpath.yaml" || fail "hostPath workspace omitted configured path"
grep -q 'hostPath:' "$tmp_dir/hostpath.yaml" || fail "hostPath workspace did not render a hostPath volume"

if helm template workspace-test "$chart" "${common[@]}" \
  --set builderAdmin.workspace.enabled=true \
  --set builderAdmin.workspace.mode=pvc \
  >/dev/null 2>&1; then
  fail "unsupported workspace mode rendered successfully"
fi

if helm template workspace-test "$chart" "${common[@]}" \
  --set builderAdmin.workspace.enabled=true \
  --set builderAdmin.workspace.mode=hostPath \
  >/dev/null 2>&1; then
  fail "hostPath workspace without a path rendered successfully"
fi

printf 'Builder Admin workspace Helm tests passed.\n'
bash "$repo_root/helm-chart/tests/portable-defaults.sh"
