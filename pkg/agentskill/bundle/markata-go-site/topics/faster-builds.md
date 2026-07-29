# Faster Builds

Use this topic when the task is build speed, local iteration speed, or profiling slow plugins.

## First Steps

- use `markata-go build --fast` for faster development loops
- use `markata-go serve --fast` for the normal live-edit loop
- use `markata-go reader update` when you only need fresh `/reader/` feed data for the next build
- prefer `[markata-go.blogroll] refresh_on_build = false` when you want to keep blogroll pages but move remote refresh work out of the normal build
- use `markata-go reader update --concurrency <n>` when reader refresh latency is dominated by many remote feeds
- use `-m fast.toml` or `--merge-config fast.toml` when you want a slimmer dev config without editing the main site config
- compare warm builds, not just cold builds
- use `markata-go build --benchmark-json benchmark.json` for structured timing
- use `markata-go build -v --benchmark-detailed` when you need stage detail
- read the `Slowest requests` footer section before assuming a slow plugin is CPU-bound

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

For `markata-go build --fast`, file discovery still rescans the content tree on each run. Added,
removed, and moved files should be detected without clearing `.markata/`. Only `serve --fast`
reuses in-memory and on-disk state for incremental rebuilds between change events.

So `--fast` is good for content, template, and most styling iteration, but it is not a full partial build mode.

## Guidance

- The second warm build is the best steady-state comparison point.
- Avoid deleting caches unless you specifically need a cold-build measurement.
- If output is network-bound, inspect plugins that fetch remote content.
- If output is read-heavy, inspect globbing, cache loads, and broad content scans.
- If output is write-heavy, inspect cache saves, feed publishing, Pagefind output, and static output.
- If warm builds still spend time in `configure/build_cache`, check whether template or config files actually changed before assuming the cache is stale; the build cache now fingerprints the template tree before it does a full rehash.
- If `/tags` or `/garden` writes are hot, prefer cached per-post semantic hashes so the listing hashes don't need to re-derive the same per-post summaries every build.
- Prefer targeted fixes over broad cache-busting changes.
- For sites that use `[markata-go.mermaid] mode = "chromium"` or `"cli"`, unchanged
  diagrams should reuse cached SVG output on warm builds; if Mermaid remains a hotspot,
  compare the diagram source and rendering inputs before recommending client mode.

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

## Concrete Profiling Flow

1. run the build that matches the question you are asking
2. read `Resource profile` to see whether the slowdown is mostly CPU, network, disk read, or disk write
3. read `Hotspots` to find the slow plugin hook
4. read `Slowest requests` to find the exact remote waits inside that plugin
5. if the build is still CPU-heavy after network issues are understood, switch to `--cpuprofile`

## Common Culprits

- remote metadata or embed fetching
- blogroll and reader feed refreshes during normal builds when cache-only mode is not configured
- pagefind, minification, and purge steps
- feed-heavy sites with many aggregate pages
- custom templates or plugins doing repeated expensive work
