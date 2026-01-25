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
