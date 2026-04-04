# Faster Builds

Use this topic when the task is build speed, local iteration speed, or profiling slow plugins.

## First Steps

- use `markata-go build --fast` for faster development loops
- use `markata-go serve --fast` for the normal live-edit loop
- use `-m fast.toml` or `--merge-config fast.toml` when you want a slimmer dev config without editing the main site config
- compare warm builds, not just cold builds
- use `markata-go build --benchmark-json benchmark.json` for structured timing
- use `markata-go build -v --benchmark-detailed` when you need stage detail

## What `--fast` Skips

Fast mode is for local iteration. It keeps the normal content pipeline but skips expensive non-essential work.

From current code, fast mode skips or reduces:

- JS minification
- CSS minification
- CSS purging
- Tailwind rebuild work
- Pagefind indexing
- some fast-mode-aware write work such as redirects generation

## What Still Runs In `--fast`

Fast mode still does the main site build work:

- config loading and validation
- file discovery
- markdown loading and frontmatter parsing
- transforms
- markdown rendering
- template rendering
- feed and collection generation
- normal output writing for the site itself

So `--fast` is good for content, template, and most styling iteration, but it is not a full partial build mode.

## Guidance

- The second warm build is the best steady-state comparison point.
- Avoid deleting caches unless you specifically need a cold-build measurement.
- If output is network-bound, inspect plugins that fetch remote content.
- If output is disk-bound, inspect globbing, cache load/save, feed publishing, and static output.
- Prefer targeted fixes over broad cache-busting changes.

## Slim Config Overrides With `-m fast.toml`

Use merged config overrides when `--fast` alone is not enough.

This is the recommended pattern for a lighter development loop because it lets you keep your main config intact while narrowing the site shape for local work.

Examples:

```bash
markata-go serve --fast -m fast.toml
markata-go build --fast -m fast.toml
markata-go build -m markata-go.local.toml -m fast.toml
```

Typical uses for `fast.toml`:

- narrower content globs
- fewer feeds
- disabling expensive optional features for local work
- lower concurrency or alternate local URLs when needed

Example `fast.toml`:

```toml
[markata-go.glob]
patterns = ["posts/current/**/*.md", "pages/*.md"]

[markata-go]
concurrency = 2
```

A starter version is included at `../examples/fast.toml`.

Use merge configs for scope changes. Use `--fast` for expensive output-step skips. Use both together for the shortest loop.

## Fast Path For Everyday Work

1. `markata-go serve --fast -m fast.toml`
2. if the output still looks wrong, run `markata-go build`
3. if the build is slow, capture `--benchmark-json` or `--benchmark-detailed`
4. compare warm builds before changing caching behavior

## Common Culprits

- remote metadata or embed fetching
- pagefind, minification, and purge steps
- feed-heavy sites with many aggregate pages
- custom templates or plugins doing repeated expensive work
