# Build And Deployment

Use this topic when the task involves local preview, CI, publishing, or hosting strategy.

## Local Iteration

- `markata-go serve --fast` for active editing
- `markata-go build` for full output validation
- `markata-go build --clean` only when you need to rule out stale output

## Production Build Basics

- set the correct `url` before production builds
- treat `output_dir` as the deploy artifact root
- prefer clean builds for deployment validation

Examples:

```bash
markata-go build --clean
MARKATA_GO_URL=https://example.com markata-go build
```

## Hosting Targets That Fit Well

- GitHub Pages
- Netlify
- Vercel
- Cloudflare Pages
- AWS S3 or another static bucket
- self-hosted nginx, Caddy, or Docker-based static hosting

## Guidance

- Prefer simple static hosting first.
- Keep deployment changes focused on reproducible builds and correct output paths.
- If the repo already has CI, extend the existing workflow instead of replacing it.
- If a deploy bug is path-related, inspect `output_dir`, `url`, asset paths, and feed URLs before changing templates.

## Inspect

- workflow files under `.github/workflows/` or other CI directories
- deployment docs in the repo
- output path settings in config
- asset or CDN self-hosting settings when debugging missing static files
