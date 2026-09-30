# Faster Builds

Use this topic when the task is build speed, local iteration speed, or profiling slow plugins.

## First Steps

- use `markata-go build --fast` for faster development loops
- use `markata-go serve --fast` for the normal live-edit loop
- use `markata-go serve --incremental` for a production-style live-edit loop
- use `markata-go reader update` when you only need fresh `/reader/` feed data for the next build
- prefer `[markata-go.blogroll] refresh_on_build = false` when you want to keep blogroll pages but move remote refresh work out of the normal build
- use `markata-go reader update --concurrency <n>` when reader refresh latency is dominated by many remote feeds
- use `-m fast.toml` or `--merge-config fast.toml` when you want a slimmer dev config without editing the main site config
- compare warm builds, not just cold builds
- use `markata-go build --benchmark-json=benchmark.json` for structured timing
- use `markata-go build -v --benchmark-detailed` when you need stage detail
- use `markata-go buildlab run --fixture <path>` to check clean, incremental, and deterministic build behavior
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

Unchanged pages restore cached full-page HTML before output is written. This
includes published pages with frontmatter but no Markdown body, so removing
the output directory does not require deleting `.markata/` before a normal or
`--fast` build recreates the page.

For `markata-go build --fast`, file discovery still rescans the content tree on each run. Added,
removed, and moved files should be detected without clearing `.markata/`. Only `serve --fast`
reuses in-memory and on-disk state for incremental rebuilds between change events.

So `--fast` is good for content, template, and most styling iteration, but it is not a full partial build mode. `serve --incremental` keeps the partial-build cache while retaining normal output processing.

## Guidance

- The second warm build is the best steady-state comparison point.
- Avoid deleting caches unless you specifically need a cold-build measurement.
- If output is network-bound, inspect plugins that fetch remote content.
- If output is read-heavy, inspect globbing, cache loads, and broad content scans.
- If output is write-heavy, inspect cache saves, feed publishing, Pagefind output, and static output.
- If warm builds still spend time in `configure/build_cache`, check whether template or config files actually changed before assuming the cache is stale; the build cache now fingerprints the template tree before it does a full rehash.
- The image-library cache reuses local media content fingerprints when each file's path, size, and modification time are unchanged. On filesystems that expose change time, it also detects same-size replacements that preserve mtime; output directories are excluded from source-media hashing, and the canonical image-library hash uses verified content fingerprints rather than mtime. Avoid clearing `.markata/` when measuring this warm-build path.
- If `/tags` or `/garden` writes are hot, prefer cached per-post semantic hashes so the listing hashes don't need to re-derive the same per-post summaries every build.
- Prefer targeted fixes over broad cache-busting changes.
- HTML cache restoration avoids a second page-sized byte-to-string copy in
  both executors. This reduces warm-build allocation, not cache invalidation
  or work selection. Keep existing caches when measuring it; formats and keys
  are unchanged, and unreadable cache files still cause re-rendering.
- Full builds keep the complete content diagnostics artifact; serialization
  streams entries rather than buffering the whole JSON document. Do not disable
  diagnostics to make a benchmark look faster. Sidebar link projections are
  reused only within a build, with current-page highlights kept separate.
- Glossary matching uses longest-first keys with lexical ties, so equal-length
  aliases sharing a link limit do not randomly change cold-build output.
  Nested protected links/code are restored outermost first without internal
  marker leakage. Pre-fix results refresh once through the cache identity.
- The experimental `--dag` executor is serial and does not automatically skip
  more work or add parallelism. Compare it with explicit `--dag=false` on the
  same site/config/cache inputs, alternating at least three warm samples.
  The verbose plan digest is structural, not a content-cache fingerprint.
- On real-site benchmarks, keep encryption settings identical, disable
  publication/index-ingestion side effects, and keep unencrypted measurements
  in isolated local copies. Network-backed embeds and favicons need a shared
  external-cache snapshot before byte-level executor comparisons are meaningful.
- Warm fontpack annotation should not recopy page bodies whose root element
  already selects the resolved pack. If fontpack remains a hotspot, distinguish
  catalog/coverage resolution from HTML annotation before changing typography.
- Cold fontpack resolution shares one visible-rune analysis across packs and
  stops checking a family's coverage when full is required. It selects prebuilt
  font tiers rather than running a subsetter. Built-in cache identity includes
  canonical pack choices, default/picker settings, coverage, and bundled metadata;
  a new cache schema causes one fresh resolution. Missing generated font files
  are regenerated without disabling the picker or weakening custom validation.
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
