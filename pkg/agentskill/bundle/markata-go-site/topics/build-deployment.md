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
- when using nginx, markata-go can generate `redirects.conf` from `static/_redirects`; include `public/redirects.conf` from the nginx `server` block for native redirects while leaving HTML fallback redirects enabled unless you intentionally disable them

## Guidance

- Prefer simple static hosting first.
- Keep deployment changes focused on reproducible builds and correct output paths.
- If the repo already has CI, extend the existing workflow instead of replacing it.
- If a deploy bug is path-related, inspect `output_dir`, `url`, asset paths, and feed URLs before changing templates.
- If previews and production use different domains, inject `MARKATA_GO_URL` per environment instead of hardcoding one value.
- Validate that feed URLs, social URLs, and asset URLs use the expected domain after build.

## Markata-Go-Specific Checks Before Shipping

- `url` matches the actual deployment domain
- `output_dir` matches the host publish directory
- static assets copied from `static/` are present in output
- feeds like `rss.xml` or `atom.xml` contain the correct absolute URLs
- homepage and archive slugs resolve to the intended directories
- any `_headers`, `_redirects`, or `CNAME` files under `static/` are present in output when the host expects them
- if self-hosting with nginx, verify the build produced `redirects.conf` and that the server includes it from the generated output tree

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
