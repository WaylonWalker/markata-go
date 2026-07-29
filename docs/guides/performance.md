---
title: "Performance Benchmarking"
description: "How to run, interpret, and optimize markata-go build performance"
date: 2024-01-15
published: true
tags:
  - performance
  - benchmarking
  - profiling
  - optimization
---

# Performance Benchmarking

markata-go includes a comprehensive benchmarking suite for measuring and optimizing build performance. This guide covers how to run benchmarks locally, interpret results, and use profiling tools to identify bottlenecks.

## Quick Start

Run the end-to-end build benchmark:

```bash
just perf
```

This runs the benchmark 5 times and outputs results to `bench.txt`.

## Running Benchmarks Locally

### Prerequisites

Install `benchstat` for analyzing benchmark results:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

### Available Commands

| Command | Description |
|---------|-------------|
| `just perf` | Run end-to-end benchmarks (5 iterations) |
| `just perf-profile` | Generate CPU and memory profiles |
| `just perf-stages` | Benchmark individual lifecycle stages |
| `just perf-concurrency` | Test performance at different concurrency levels |
| `just perf-compare old.txt new.txt` | Compare two benchmark runs |
| `just perf-generate` | Regenerate the benchmark fixture |

### Running Specific Benchmarks

```bash
# All benchmarks
go test -bench=. -run='^$' -benchmem ./benchmarks/...

# Only end-to-end
go test -bench=BenchmarkBuild_EndToEnd -run='^$' -benchmem ./benchmarks/...

# Only stage-specific
go test -bench='BenchmarkStage' -run='^$' -benchmem ./benchmarks/...

# With more iterations for stability
go test -bench=BenchmarkBuild -run='^$' -benchmem -count=10 ./benchmarks/...
```

## Understanding Benchmark Output

### Build Summary Hotspots

The default `markata-go build` summary now includes two fast feedback signals:

- **Resource profile** - estimated wall-time spent on CPU work, network wait, disk read wait, disk write wait, and idle time
- **Hotspots** - the slowest lifecycle plugin hooks from that build
- **Slowest requests** - the longest outbound HTTP requests with plugin attribution

Example:

```text
Build completed successfully!
  Resource profile (estimated wall time):
    CPU             18.2s (20.1%)
    Network wait    42.7s (47.1%)
    Disk read       10.1s (11.1%)
    Disk write      14.3s (15.8%)
    Idle             5.4s ( 5.9%)
  Hotspots:
    collect/blogroll 31.77s
    cleanup/pagefind 26.32s
    write/publish_feeds 8.54s
  Slowest requests:
    collect/blogroll GET https://example.com/feed.xml 12.40s (HTTP 200)
    transform/mentions GET https://slow.example.com/ 6.80s (context deadline exceeded)
```

Use this summary to decide what tool to reach for next:

- mostly `CPU` -> capture a CPU profile with `just perf-profile`
- mostly `Network wait` -> inspect `Slowest requests` first, then the owning plugins that fetch remote content or external metadata
- mostly `Disk read` -> inspect globbing, cache loads, index reads, and wide content scans
- mostly `Disk write` -> inspect publishing, cache saves, index generation, and emitted static output
- if `configure/build_cache` is hot on warm builds, check whether the template or config tree actually changed; unchanged template trees now reuse a cheap fingerprint before falling back to a full content hash
- if `/tags` or `/garden` hashing is hot, prefer cached per-post semantic hashes over re-deriving the same per-post summaries inside the listing hashers
- mostly `Idle` -> look for subprocess waits, scheduler gaps, or work that is happening outside the Go process

### Concrete Flow For Real Site Builds

Use this flow when you want to answer "what feature is slowing this build down?"

1. Run a build that matches the feature set you care about.
   - for the everyday dev loop, use `markata-go build --fast`
   - for real feature attribution, run the full build without `--fast`
2. Read the footer in this order:
   - `Resource profile` tells you whether the build is mostly CPU, network, disk read, or disk write bound
   - `Hotspots` tells you which plugin hooks are slow overall
   - `Slowest requests` tells you which exact network calls dominated wall time
3. If `Slowest requests` points to one plugin repeatedly, fix that plugin first.
   - add or verify cache reuse
   - reduce duplicate fetches
   - batch requests or lower request count
   - make timeouts and concurrency explicit
4. Export structured data when you need a diffable artifact:

```bash
markata-go build --benchmark-json benchmark.json
```

5. If the build is still mostly CPU after network fixes, capture `--cpuprofile` and inspect the hottest functions with `go tool pprof`.

### JSON Benchmarks

Use machine-readable output when you want to compare builds over time or ingest
results into another tool:

```bash
markata-go build --benchmark-json benchmark.json
markata-go build --benchmark-json - > benchmark.json
```

The JSON output includes:

- whole-build resource totals
- per-stage timings and per-stage estimated resources
- plugin timing entries used for hotspot ranking
- request timing entries used for the slowest-request list
- build counts and warnings

### Per-Stage Detail

Keep the default footer small for everyday use, and opt into stage detail when
debugging:

```bash
markata-go build -v --benchmark-detailed
```

This adds a per-stage estimated wall-time breakdown so you can see whether a
slow build is CPU-heavy in `render`, read-heavy in `glob`, write-heavy in `write`, or mostly idle in a
subprocess-oriented cleanup stage.

### Raw Output

```
BenchmarkBuild_EndToEnd-8    	       5	 234567890 ns/op	123456789 B/op	 1234567 allocs/op
```

| Field | Meaning |
|-------|---------|
| `BenchmarkBuild_EndToEnd-8` | Test name with GOMAXPROCS |
| `5` | Number of iterations |
| `234567890 ns/op` | Nanoseconds per operation |
| `123456789 B/op` | Bytes allocated per operation |
| `1234567 allocs/op` | Number of allocations per operation |

### Using benchstat

`benchstat` provides statistical analysis of benchmark results:

```bash
# Single run analysis
benchstat bench.txt

# Compare two runs
benchstat old.txt new.txt
```

Example output:

```
name                 time/op
Build_EndToEnd-8     235ms ± 2%

name                 alloc/op
Build_EndToEnd-8     124MB ± 0%

name                 allocs/op
Build_EndToEnd-8     1.23M ± 0%
```

The `±` value shows the variation between runs. Lower is better for reproducibility.

### Comparing Runs

When comparing two benchmark files:

```
name              old time/op    new time/op    delta
Build_EndToEnd-8    250ms ± 3%     235ms ± 2%   -6.00%  (p=0.008 n=5+5)
```

| Column | Meaning |
|--------|---------|
| `old time/op` | Time from first file |
| `new time/op` | Time from second file |
| `delta` | Percentage change (negative = faster) |
| `p=0.008` | Statistical significance (p < 0.05 is significant) |
| `n=5+5` | Number of samples in each file |

## Profiling

### Generating Profiles

```bash
just perf-profile
```

This creates:
- `cpu.prof` - CPU profile
- `mem.prof` - Memory allocation profile

### Analyzing CPU Profiles

#### Interactive CLI

```bash
go tool pprof cpu.prof
```

Common commands:
- `top` - Show top functions by CPU time
- `top -cum` - Show by cumulative time
- `list FunctionName` - Show annotated source
- `web` - Open in browser (requires graphviz)

#### Web Interface

```bash
go tool pprof -http=:8080 cpu.prof
```

This opens an interactive web UI with:
- Flame graphs
- Call graphs
- Source code annotation
- Top functions

### Analyzing Memory Profiles

```bash
go tool pprof mem.prof
```

Useful options:
- `go tool pprof -alloc_space mem.prof` - Total allocations
- `go tool pprof -alloc_objects mem.prof` - Number of allocations
- `go tool pprof -inuse_space mem.prof` - Live memory

### Profile Types

| Profile | What it Measures | When to Use |
|---------|-----------------|-------------|
| CPU | Time spent in functions | Slow builds |
| Memory | Allocations | High memory usage |
| Block | Blocking on sync primitives | Deadlocks/contention |
| Mutex | Mutex contention | Lock performance |

## Benchmark Fixture

The benchmark suite uses a deterministic fixture at `benchmarks/site/`:

```
benchmarks/
├── site/
│   ├── markata-go.toml      # Benchmark config
│   └── posts/
│       ├── blog/2024/01/    # 60 blog posts
│       └── docs/guides/     # 40 documentation guides
└── benchmark_test.go        # Benchmark tests
```

### Fixture Characteristics

- **100 posts total** - Representative of a medium-sized site
- **Code-heavy content** - Syntax highlighting stress test
- **Nested paths** - Tests path handling
- **Multiple languages** - Go, Python, JavaScript, Rust, SQL
- **Deterministic** - Same content every generation for reproducible results

### Regenerating the Fixture

```bash
just perf-generate
```

Or directly:

```bash
go run benchmarks/generate_posts.go
```

## CI Performance Tracking

Performance benchmarks run automatically:
- **Weekly** - Sunday at 2 AM UTC
- **Manual** - Via workflow dispatch

### Viewing Results

1. Go to Actions tab in GitHub
2. Select "Performance Benchmarks" workflow
3. View the job summary for benchstat output
4. Download artifacts for profiles

### Comparing Branches

Trigger a manual workflow with a comparison branch:

1. Go to Actions > Performance Benchmarks
2. Click "Run workflow"
3. Enter the branch name to compare against
4. View the comparison in the job summary

## Optimization Tips

### Common Bottlenecks

1. **Markdown rendering** - goldmark processing
2. **Template execution** - Pongo2 templates
3. **File I/O** - Reading/writing files
4. **Syntax highlighting** - Chroma processing
5. **Memory allocations** - String operations

### Improving Performance

#### Concurrency

Adjust the concurrency level in `markata-go.toml`:

```toml
[markata-go]
concurrency = 8  # 0 = auto (NumCPU)
```

#### Profile-Guided Optimization

1. Run profiling: `just perf-profile`
2. Analyze: `go tool pprof -http=:8080 cpu.prof`
3. Identify hot paths
4. Optimize targeted functions
5. Re-benchmark to verify improvement

#### Memory Optimization

If memory is the bottleneck:

```bash
go tool pprof -alloc_space mem.prof
```

Look for:
- Large allocations (`alloc_space`)
- Many small allocations (`alloc_objects`)
- Functions with high cumulative allocations

### Writing Efficient Plugins

1. **Reuse allocations** - Use `sync.Pool` for buffers
2. **Minimize copies** - Use pointers where appropriate
3. **Batch operations** - Group file writes
4. **Cache results** - Use the lifecycle cache

## Troubleshooting

### Benchmarks Skip

```
--- SKIP: BenchmarkBuild_EndToEnd
    benchmark_test.go:35: Benchmark fixture not found
```

Fix: Run `just perf-generate` to create the fixture.

### High Variance

If `±` values are high (>10%), try:
- More iterations: `-count=10`
- Close other applications
- Use a consistent environment

### Profile is Empty

Ensure the benchmark runs long enough:

```bash
go test -bench=BenchmarkBuild_EndToEnd -run='^$' \
  -benchtime=30s \
  -cpuprofile=cpu.prof \
  ./benchmarks/...
```

## See Also

- [Configuration Guide](/docs/guides/configuration/) - Concurrency settings
- [Plugin Development](/docs/guides/plugin-development/) - Writing efficient plugins
- [Go Profiling](https://go.dev/blog/pprof) - Official pprof documentation
