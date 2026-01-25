---
title: "Performance and Profiling"
description: "Guide to benchmarking and profiling markata-go build performance"
date: 2026-01-24
published: true
tags:
  - performance
  - benchmarking
  - profiling
  - advanced
---

# Performance and Profiling

This guide covers benchmarking and profiling tools for analyzing and optimizing markata-go build performance.

## Running Benchmarks

markata-go includes comprehensive benchmarks for key build stages. Use Go's built-in benchmarking tools to measure performance.

### Quick Start

```bash
# Run all benchmarks
go test -bench=. ./...

# Run benchmarks with memory statistics
go test -bench=. -benchmem ./...

# Run specific package benchmarks
go test -bench=. ./pkg/lifecycle/
go test -bench=. ./pkg/plugins/
go test -bench=. ./pkg/config/
```

### Benchmark Output

Benchmark results show operations per second and time per operation:

```
BenchmarkRenderMarkdown_ColdStart-8    	     100	  12345678 ns/op	  1234567 B/op	   12345 allocs/op
```

| Column | Meaning |
|--------|---------|
| `-8` | Number of CPU cores used |
| `100` | Number of iterations |
| `12345678 ns/op` | Time per operation in nanoseconds |
| `1234567 B/op` | Bytes allocated per operation |
| `12345 allocs/op` | Memory allocations per operation |

### Comparing Benchmarks

To compare performance between versions or changes:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save results
go test -bench=. -count=10 ./pkg/lifecycle/ > old.txt

# Make changes, then run again
go test -bench=. -count=10 ./pkg/lifecycle/ > new.txt

# Compare results
benchstat old.txt new.txt
```

## Available Benchmarks

### Lifecycle Benchmarks (`pkg/lifecycle/`)

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkManager_ColdStart` | Full build with no cache |
| `BenchmarkManager_HotCache` | Build with warm cache |
| `BenchmarkManager_GlobStage` | File discovery phase |
| `BenchmarkManager_LoadStage` | File loading and parsing |
| `BenchmarkProcessPostsConcurrently` | Concurrent processing at various levels |
| `BenchmarkFilter` | Filter expression evaluation |
| `BenchmarkMemoryCache` | Cache operations (set/get) |
| `BenchmarkPluginSorting` | Plugin priority sorting |

### Plugin Benchmarks (`pkg/plugins/`)

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkRenderMarkdown_ColdStart` | Fresh markdown renderer |
| `BenchmarkRenderMarkdown_HotCache` | Reused markdown renderer |
| `BenchmarkRenderMarkdown_ContentSizes` | Different content sizes |
| `BenchmarkRenderMarkdown_SyntaxHighlighting` | Code block rendering |
| `BenchmarkParseFrontmatter` | Frontmatter parsing |
| `BenchmarkGlob` | File globbing |
| `BenchmarkGlob_WithGitignore` | Globbing with .gitignore |
| `BenchmarkLoad` | File loading |
| `BenchmarkLoad_Concurrency` | Loading at various concurrency levels |
| `BenchmarkTemplateEngine` | Template rendering |
| `BenchmarkPublishHTML_Write` | HTML file writing |

### Config Benchmarks (`pkg/config/`)

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkLoad_TOML` | TOML config loading |
| `BenchmarkLoad_YAML` | YAML config loading |
| `BenchmarkLoad_JSON` | JSON config loading |
| `BenchmarkLoadAndValidate` | Load with validation |
| `BenchmarkParseTOML` | Raw TOML parsing |
| `BenchmarkParseYAML` | Raw YAML parsing |
| `BenchmarkParseJSON` | Raw JSON parsing |
| `BenchmarkMergeConfigs` | Config merging |
| `BenchmarkValidateConfig` | Config validation |

## CPU Profiling

Generate CPU profiles to identify performance bottlenecks.

### Generate Profile

```bash
# Profile all benchmarks
go test -bench=. -cpuprofile=cpu.prof ./pkg/lifecycle/

# Profile specific benchmark
go test -bench=BenchmarkManager_ColdStart -cpuprofile=cpu.prof ./pkg/lifecycle/
```

### Analyze Profile

```bash
# Interactive CLI
go tool pprof cpu.prof

# Common commands in pprof:
# top10        - Show top 10 functions by CPU time
# top20 -cum   - Top 20 by cumulative time
# list funcName - Show source for a function
# web          - Open flame graph in browser (requires graphviz)

# Direct web interface
go tool pprof -http=:8080 cpu.prof
```

### Example Session

```bash
$ go tool pprof cpu.prof
(pprof) top10
Showing nodes accounting for 1.5s, 75% of 2s total
      flat  flat%   sum%        cum   cum%
     0.5s   25%    25%       0.8s    40%  goldmark/renderer.(*Renderer).Render
     0.3s   15%    40%       0.3s    15%  yaml.unmarshal
     0.2s   10%    50%       0.4s    20%  regexp.(*Regexp).FindAllStringIndex
     ...
```

## Memory Profiling

Track memory allocations and identify potential memory leaks.

### Generate Profile

```bash
# Memory profile
go test -bench=. -memprofile=mem.prof ./pkg/lifecycle/

# With allocation count
go test -bench=. -memprofile=mem.prof -memprofilerate=1 ./pkg/lifecycle/
```

### Analyze Profile

```bash
# View allocations
go tool pprof -alloc_space mem.prof

# View allocation count
go tool pprof -alloc_objects mem.prof

# In-use memory (useful for finding leaks)
go tool pprof -inuse_space mem.prof
```

## Trace Profiling

Generate execution traces for detailed timing analysis.

```bash
# Generate trace
go test -bench=BenchmarkManager_ColdStart -trace=trace.out ./pkg/lifecycle/

# View trace (opens in browser)
go tool trace trace.out
```

The trace viewer shows:
- Goroutine scheduling
- GC events
- Network/system calls
- Blocking operations

## Performance Tips

### 1. Concurrency Settings

Adjust `concurrency` in your config for optimal performance:

```toml
[markata-go]
concurrency = 8  # Tune based on your CPU cores and I/O
```

**Benchmark different values:**
```bash
go test -bench=BenchmarkLoad_Concurrency ./pkg/plugins/
```

### 2. Reduce File I/O

- Use `.gitignore` patterns to skip unnecessary files
- Limit glob patterns to only necessary directories

```toml
[markata-go.glob]
patterns = ["posts/**/*.md"]  # Be specific
use_gitignore = true
```

### 3. Template Caching

Templates are automatically cached after first use. For best results:
- Minimize template complexity
- Use template inheritance efficiently
- Avoid heavy computations in templates

### 4. Markdown Rendering

Markdown rendering is typically the slowest stage. Optimize by:
- Keeping posts reasonably sized
- Using syntax highlighting only when needed
- Avoiding deeply nested structures

### 5. Build Caching

The manager caches data between stages. For incremental builds:
- Cache is cleared on each build by default
- Hot cache significantly improves rebuild times

## Continuous Performance Monitoring

### GitHub Actions Benchmark

Add benchmark comparison to your CI:

```yaml
name: Benchmarks
on: [push, pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./... | tee bench.txt

      - name: Upload results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results
          path: bench.txt
```

### Local Performance Script

Create a script for regular performance checks:

```bash
#!/bin/bash
# scripts/benchmark.sh

echo "Running markata-go benchmarks..."

# Run benchmarks
go test -bench=. -benchmem -count=5 ./pkg/lifecycle/ | tee lifecycle.txt
go test -bench=. -benchmem -count=5 ./pkg/plugins/ | tee plugins.txt
go test -bench=. -benchmem -count=5 ./pkg/config/ | tee config.txt

echo ""
echo "Results saved to:"
echo "  - lifecycle.txt"
echo "  - plugins.txt"
echo "  - config.txt"
```

## Interpreting Results

### What's "Fast Enough"?

Typical performance targets:

| Stage | Target | Notes |
|-------|--------|-------|
| Config loading | < 10ms | One-time cost |
| Globbing (100 files) | < 50ms | Scales with file count |
| Loading (100 files) | < 200ms | I/O bound |
| Markdown render (100 posts) | < 500ms | CPU bound |
| Template render (100 posts) | < 200ms | Cached templates |
| Write (100 files) | < 100ms | I/O bound |

### Common Bottlenecks

1. **Markdown rendering** - Usually the slowest stage
2. **Syntax highlighting** - Expensive for code-heavy content
3. **File I/O** - Disk speed matters for large sites
4. **Memory allocations** - High allocation count slows GC

### Red Flags

- `B/op` > 10MB per operation for small content
- `allocs/op` > 10000 for simple operations
- Linear time growth that should be constant
- Memory not being released (check with `-inuse_space`)

## Further Reading

- [Go Testing and Benchmarking](https://pkg.go.dev/testing)
- [pprof Documentation](https://github.com/google/pprof)
- [Go Execution Tracer](https://pkg.go.dev/runtime/trace)
- [benchstat Tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
