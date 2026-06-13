# Markata Notes Helm Chart

This chart deploys a reusable markata-go notes workload that:

- pulls a source archive from object storage
- optionally decrypts it with `MARKATA_GO_ENCRYPTION_KEY_DEFAULT`
- renders the site in-cluster on a schedule or manual trigger
- serves the rendered site with nginx
- serves bleve search from the same host under `/api/search`

## Notes

- `work_notes/justfile` now uploads `source.tar.gz` from `git archive HEAD`.
- Set `MARKATA_GO_SOURCE_ARCHIVE_ENCRYPT=true` when publishing if you want the uploaded archive encrypted with `MARKATA_GO_ENCRYPTION_KEY_DEFAULT`.
- The search pod runs `markata-go search-server --mode watch-content --host 0.0.0.0` so bleve stays in sync when the source PVC changes.
- The search pod now waits for the source PVC to be populated before starting, which avoids booting against an empty archive mount and indexing `0 posts`.
- Runtime pods use a dedicated ServiceAccount with `automountServiceAccountToken: false`, disable service links, and apply `RuntimeDefault` seccomp with stricter container security settings where they are low risk.
- A NetworkPolicy now limits runtime pod egress to cluster DNS, so the site and search pods cannot freely call other cluster services by default.
- The build CronJob uses a source PVC lock by default so overlapping manual jobs cannot update the shared source and site PVCs at the same time.
- The build CronJob now overrides `[markata-go.mermaid].mode` to `client` by default, which avoids Chromium hangs seen in-cluster; set `build.mermaid.mode` back to `chromium` or `""` if you want to rely on the source repo config instead.
- If your new namespace does not already have `aws-default`, enable `aws.sealedSecret` and provide sealed credentials in values.
- By default the chart reuses a shared `markata-go-encryption` Secret name for `MARKATA_GO_ENCRYPTION_KEY_DEFAULT`; override `markataEncryption.secretName` only if your established secret uses a different name.
- If your notes contain private content, provide that Secret externally in the namespace or enable `markataEncryption.sealedSecret`.

## Setting up the encryption secret

1. Reuse the same plaintext value you already use locally for `MARKATA_GO_ENCRYPTION_KEY_DEFAULT`.
2. Seal it for the target namespace:

```bash
printf '%s' "$MARKATA_GO_ENCRYPTION_KEY_DEFAULT" | kubeseal \
  --controller-name sealed-secrets \
  --namespace <project>-notes \
  --name markata-go-encryption \
  --raw
```

3. Put the returned encrypted blob into your ArgoCD values:

```yaml
markataEncryption:
  secretName: "markata-go-encryption"
  sealedSecret:
    enabled: true
    encryptedData:
      MARKATA_GO_ENCRYPTION_KEY_DEFAULT: "Ag..."
```

4. If you also want the uploaded `source.tar.gz` encrypted, set:

```yaml
sourceArchive:
  encryption:
    enabled: true
```

Then publish with:

```bash
export MARKATA_GO_SOURCE_ARCHIVE_ENCRYPT=true
just push
```

## Generic defaults

- Set `project_identifier`, `project_name`, `sourceArchive.bucket`, and `sourceArchive.location` for your site before installing.
- Set `ingress.host` explicitly if you do not want the default `<project>.example.com` hostname.
- Ingress auth is disabled by default. If you enable it, set `ingress.auth.url` and optionally `ingress.auth.internalUrl` for your auth provider.

## Offline builds

The official `ghcr.io/waylonwalker/markata-go-builder` image ships with:

- preloaded bundled CDN asset cache
- bundled Mermaid JS source
- `aws` and `openssl` for the source fetch/decrypt step

That means the chart can run without internet egress as long as your workload can still reach the configured source archive location and your site only needs assets that are already bundled or already cached.

Simple Helm values:

```yaml
offline:
  enabled: true
```

When `offline.enabled` is true, the build and search workloads run with:

- `MARKATA_GO_OFFLINE=true`
- `MARKATA_GO_BUNDLED_ASSETS_CACHE_DIR=/usr/local/share/markata-go/assets-cache`
- `MARKATA_GO_BUNDLED_MERMAID_DIR=/usr/local/share/markata-go/mermaid`

If your site depends on extra self-hosted CDN assets beyond the bundled cache, pre-populate `.markata/assets-cache` in the source archive before publishing it:

```bash
markata-go assets download
just push
```

Or override `offline.bundledAssetsCacheDir` / `offline.bundledMermaidDir` to point at a custom preloaded location in your image.

## Manual rebuild

```bash
kubectl create job --from=cronjob/<project>-notes-build <project>-notes-build-manual-$(date +%s) -n <project>-notes
```

If another build is already running, the manual job waits on the build lock until the active build finishes or the lock becomes stale.
