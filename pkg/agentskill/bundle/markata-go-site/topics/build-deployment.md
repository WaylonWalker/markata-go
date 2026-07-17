# Build And Deployment

Use this topic when the task involves local preview, CI, publishing, or hosting strategy.

## Local Iteration

- `markata-go serve --fast` for active editing
- `markata-go build` for full output validation
- `markata-go build --clean` only when you need to rule out stale output
- `markata-go build -o dist` when CI or previews need an isolated artifact directory

## Production Build Basics

- set the correct `url` before production builds
- treat `output_dir` as the deploy artifact root
- prefer clean builds for deployment validation
- validate config before deploy if the workflow can afford it
- if the deploy target runs with limited or no internet egress, prefetch self-hosted CDN assets with `markata-go assets download` before shipping the repo or build input

Examples:

```bash
markata-go config validate
markata-go build --clean
MARKATA_GO_URL=https://example.com markata-go build
```

## Recommended CI Shape

For most hosts, the safe build flow is:

1. checkout repo
2. install Go
3. install or build `markata-go`
4. run `markata-go config validate`
5. run `markata-go build --clean`
6. publish the build artifact from `public/` or the chosen output dir

Minimal GitHub Actions shape:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
    with:
      go-version: '1.22'
  - run: go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
  - run: markata-go config validate
  - run: markata-go build --clean
    env:
      MARKATA_GO_URL: https://example.com
```

## Hosting Targets That Fit Well

- GitHub Pages
- Netlify
- Vercel
- Cloudflare Pages
- AWS S3 or another static bucket
- self-hosted nginx, Caddy, or Docker-based static hosting

## Provider Patterns

### GitHub Pages

- build in GitHub Actions
- upload `public/` as the Pages artifact
- set `MARKATA_GO_URL` to the final GitHub Pages or custom domain URL

### Netlify

- build command usually installs markata-go then runs `markata-go build --clean`
- publish directory is `public`
- set production and preview `MARKATA_GO_URL` separately if needed

### Vercel

- use a custom build command because this is a static Go-built site, not a framework preset
- set `outputDirectory` to `public`
- verify preview and production URLs separately

### Cloudflare Pages

- framework preset is usually none
- build output directory is `public`
- `_headers` and `_redirects` can live under `static/` so they copy into output

### Self-hosted

- deploy the built `public/` directory behind nginx, Caddy, S3, or another static file server
- ensure the server preserves nested `index.html` routing and static asset paths

## Guidance

- Prefer simple static hosting first.
- Keep deployment changes focused on reproducible builds and correct output paths.
- If the repo already has CI, extend the existing workflow instead of replacing it.
- If a deploy bug is path-related, inspect `output_dir`, `url`, asset paths, and feed URLs before changing templates.
- If previews and production use different domains, inject `MARKATA_GO_URL` per environment instead of hardcoding one value.
- if the runtime build environment is offline, make sure `.markata/assets-cache` or another configured asset cache is already populated before relying on self-hosted CDN assets.
- for Helm or ArgoCD source-archive deployments, prefer environment-specific `MARKATA_GO_*` overrides such as `MARKATA_GO_URL` instead of editing the repo just to change hostnames
- for Helm or ArgoCD feedback loops, inspect the chart's lock and debounce knobs (`build.lock.pollIntervalSeconds`, `search.waitForSource.pollIntervalSeconds`, `search.watchDebounce`) before chasing deeper build bugs; shorter values make manual rebuilds and search restarts feel noticeably snappier
- for Kubernetes hostPath authoring deployments, prefer the long-lived `builder-admin` service over one-shot build Jobs when the goal is fast interactive rebuilds, release history, rollback, and scheduled remote refreshes
- for Kubernetes hostPath deployments, confirm the mounted source path and served site root are the real node paths, and remember the served site root may contain release directories plus a `current` symlink rather than a flat output tree
- builder-admin is an operator surface: expose it only through its dedicated protected Traefik/hlab-auth ingress, configure `builderAdmin.auth.trustedProxyCIDRs` for the actual Traefik sources and required builder-admin peers, and do not use Service access or `kubectl port-forward` as an authentication bypass
- when enabling builder-admin, configure its TLS host, HTTPS ForwardAuth URL, and explicit ingress NetworkPolicy selectors for the live Traefik pods; a shared Pod CIDR is acceptable only with those selectors for peer forwarding, while universal, loopback, and link-local CIDRs are rejected; the chart derives the exact `https://<host>` CSRF origin, so do not derive an origin from forwarded request headers or change hlab-auth's primary RP/origin
- Validate that feed URLs, social URLs, and asset URLs use the expected domain after build.

## Markata-Go-Specific Checks Before Shipping

- `url` matches the actual deployment domain
- `output_dir` matches the host publish directory
- static assets copied from `static/` are present in output
- feeds like `rss.xml` or `atom.xml` contain the correct absolute URLs
- homepage and archive slugs resolve to the intended directories
- any `_headers`, `_redirects`, or `CNAME` files under `static/` are present in output when the host expects them

## Preview Deployments

Good preview strategy:

- keep the same build command
- inject a preview URL only when absolute links must be correct
- if the provider gives automatic preview URLs, prefer provider env configuration over editing config files

## Inspect

- workflow files under `.github/workflows/` or other CI directories
- deployment docs in the repo
- output path settings in config
- asset or CDN self-hosting settings when debugging missing static files
- any `static/CNAME`, `static/_headers`, or `static/_redirects` files
