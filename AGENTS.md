# AGENTS.md - AI Coding Agent Guidelines for markata-go

This document provides guidelines for AI coding agents working in the markata-go codebase.

## Project Overview

**markata-go** is a static site generator written in Go 1.22+. It processes Markdown files with frontmatter through a plugin-based lifecycle system to generate HTML sites.

Key dependencies: cobra (CLI), pongo2 (templating), goldmark (Markdown), fsnotify (live reload)

## Spec-Driven Development

**This is a spec-driven project.** All changes must follow this workflow:

1. **Spec First** - Before implementing any feature or change, update or create the relevant specification in `spec/`. The spec defines the expected behavior, data models, and interfaces.

2. **Review Spec** - Ensure the spec is complete and covers edge cases, error handling, and integration with existing components.

3. **Implement from Spec** - Write code that conforms to the specification. The spec is the source of truth.

4. **Test Against Spec** - Tests should verify behavior defined in the spec, not implementation details.

5. **Document for Users** - Every feature MUST have corresponding user documentation in `docs/`. See Documentation Requirements below.

**Spec directory structure:**
```
spec/
├── README.md         # Spec overview and index
├── spec/
│   ├── SPEC.md           # Core architecture overview
│   ├── CONFIG.md         # Configuration system
│   ├── LIFECYCLE.md      # Build stages and hooks
│   ├── PLUGINS.md        # Plugin development guide
│   ├── DATA_MODEL.md     # Post, Config, Feed schemas
│   ├── TEMPLATES.md      # Template system
│   ├── THEMES.md         # Theme and palette system
│   ├── FEEDS.md          # Feed generation
│   └── ...
```

**When making changes:**
- For new features: Create or update the relevant spec file first
- For bug fixes: Check if the spec needs clarification
- For refactoring: Ensure changes still conform to the spec
- If spec and code disagree: The spec should be updated first, then code follows

## Documentation Requirements

**CRITICAL: Every feature must be documented.** The documentation lives in `docs/` and is built as part of this site.

### Documentation Workflow

1. **Spec defines behavior** - Technical specification in `spec/spec/*.md`
2. **Docs explain usage** - User-friendly guides in `docs/guides/*.md`
3. **Docs are part of the site** - All docs are Markdown files processed by markata-go itself

### Documentation Structure

```
docs/
├── index.md              # Documentation home
├── getting-started.md    # Quick start guide
├── quickstart.md         # 5-minute tutorial
├── guides/
│   ├── configuration.md  # Config reference
│   ├── themes.md         # Themes and palettes
│   ├── templates.md      # Template customization
│   ├── feeds.md          # Feed system
│   ├── frontmatter.md    # Frontmatter fields
│   ├── markdown.md       # Markdown features
│   └── ...
├── reference/
│   ├── cli.md            # CLI commands
│   └── plugins.md        # Plugin reference
└── troubleshooting.md    # Common issues
```

### Documentation Standards

**Every doc file must have frontmatter:**
```yaml
---
title: "Human-Readable Title"
description: "One-line description for SEO and feeds"
date: 2024-01-15
published: true
tags:
  - documentation
  - relevant-topic
---
```

**Documentation principles:**
- **User-focused** - Explain how to USE features, not how they're implemented
- **Examples first** - Show working examples before explaining details
- **Progressive disclosure** - Simple cases first, advanced options later
- **Complete** - Cover all options, edge cases, and common mistakes
- **Tested** - All code examples should actually work

### When to Create Documentation

| Change Type | Documentation Required |
|-------------|----------------------|
| New feature | New guide or section in existing guide |
| New CLI command | Add to `docs/reference/cli.md` |
| New config option | Add to `docs/guides/configuration.md` |
| New plugin | Add to `docs/reference/plugins.md` |
| Bug fix | Update docs if behavior was unclear |
| Breaking change | Update all affected docs + migration guide |

### Documentation Checklist

Before completing any feature, verify:
- [ ] Spec is updated in `spec/spec/` (behavior, config options, data model changes)
- [ ] User documentation exists in `docs/`
- [ ] Config options are documented in `docs/guides/configuration.md`
- [ ] New plugins are documented in `docs/reference/plugins.md`
- [ ] CLI commands are documented in `docs/reference/cli.md`
- [ ] Examples are provided and tested
- [ ] Cross-references link to related docs

## Build/Lint/Test Commands

### Building

```bash
go build ./cmd/markata-go        # Build the binary
go run ./cmd/markata-go build    # Run without building
```

### Testing

```bash
go test ./...                           # Run all tests
go test -v ./...                        # Verbose output
go test -race ./...                     # Race detection
go test -coverprofile=coverage.out ./...  # Coverage
```

### Running a Single Test

```bash
go test -v -run TestParseFrontmatter ./pkg/plugins/    # Specific test
go test -v -run "TestConfig.*" ./pkg/config/           # Pattern match
go test -v ./pkg/filter/...                            # Entire package
go test -v ./tests/...                                 # Integration tests
```

### Linting and Formatting

```bash
go fmt ./...          # Format all code (ALWAYS run before committing)
go vet ./...          # Check for common issues
golangci-lint run     # Full linting (if installed)
```

## Project Structure

```
cmd/markata-go/       # CLI entry point and commands
pkg/
├── config/           # Configuration loading, parsing, validation
├── models/           # Data models (Post, Config, Feed, errors)
├── filter/           # Filter expression lexer/parser/evaluator
├── lifecycle/        # Build lifecycle manager (9 stages)
├── plugins/          # Built-in plugins (15+)
└── templates/        # Pongo2 template engine wrapper
templates/            # Default HTML templates
tests/                # Integration tests
spec/                 # Specification documents
```

## Code Style Guidelines

### Import Organization

**Three groups separated by blank lines, alphabetically sorted within each:**
```go
import (
    // 1. Standard library
    "errors"
    "fmt"
    "sync"

    // 2. Project-internal
    "github.com/example/markata-go/pkg/models"

    // 3. Third-party
    "github.com/yuin/goldmark"
    "gopkg.in/yaml.v3"
)
```

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Files | snake_case | `render_markdown.go` |
| Exported | PascalCase | `NewManager()`, `Post` |
| Unexported | camelCase | `runStage()`, `memoryCache` |
| Constructors | `New` prefix | `NewConfig()`, `NewFrontmatterParseError()` |
| Sentinel errors | `var Err` prefix | `var ErrConfigNotFound` |
| Error types | `Error` suffix | `FrontmatterParseError` |
| Interfaces | Capability name | `Plugin`, `Cache`, `ConfigurePlugin` |

### Error Handling

**Sentinel errors** - for expected, checkable conditions:
```go
var ErrConfigNotFound = errors.New("no configuration file found")
```

**Custom error types** - with context and `Unwrap()` for wrapping:
```go
type FrontmatterParseError struct {
    Path    string
    Message string
    Err     error  // wrapped error
}

func (e *FrontmatterParseError) Error() string {
    return fmt.Sprintf("frontmatter parse error in %s: %s", e.Path, e.Message)
}

func (e *FrontmatterParseError) Unwrap() error { return e.Err }

func NewFrontmatterParseError(path, message string, err error) *FrontmatterParseError {
    return &FrontmatterParseError{Path: path, Message: message, Err: err}
}
```

**Error wrapping** - always use `%w` verb:
```go
return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
```

**Error checking** - use `errors.Is()` and `errors.As()`:
```go
if errors.Is(err, ErrConfigNotFound) { ... }

var parseErr *FrontmatterParseError
if errors.As(err, &parseErr) { ... }
```

### Struct Patterns

**Struct tags** - support all three serialization formats:
```go
type Post struct {
    Path  string  `json:"path" yaml:"path" toml:"path"`
    Title *string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`
}
```

**Optional fields** - use pointers for optional scalars:
```go
Title       *string    `json:"title,omitempty"`
PublishDate *time.Time `json:"publish_date,omitempty"`
```

### Interface Patterns

**Small, composable interfaces:**
```go
type Plugin interface {
    Name() string
}

type ConfigurePlugin interface {
    Plugin
    Configure(m *Manager) error
}
```

**Compile-time interface verification:**
```go
var (
    _ lifecycle.Plugin          = (*RenderMarkdownPlugin)(nil)
    _ lifecycle.ConfigurePlugin = (*RenderMarkdownPlugin)(nil)
)
```

### Concurrency Patterns

**Mutex for shared state** - use `sync.RWMutex`:
```go
type Manager struct {
    mu    sync.RWMutex
    posts []*models.Post
}

func (m *Manager) Posts() []*models.Post {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]*models.Post{}, m.posts...)  // Return copy
}
```

**Semaphore for limiting concurrency:**
```go
semaphore := make(chan struct{}, concurrency)
for _, item := range items {
    go func(i Item) {
        semaphore <- struct{}{}        // Acquire
        defer func() { <-semaphore }() // Release
        process(i)
    }(item)
}
```

### Documentation

**Package docs** in `doc.go` with `# Heading` sections:
```go
// Package templates provides a Jinja2-like template engine.
//
// # Template Engine
//
// The Engine type manages template loading:
//
//  engine, err := templates.NewEngine("templates/")
```

**Function docs** - describe behavior, parameters, return values:
```go
// ExtractFrontmatter splits content into frontmatter and body.
// Returns ErrInvalidFrontmatter if the frontmatter is malformed.
```

### Testing Patterns

**Test naming:** `Test{Type}_{Scenario}`
```go
func TestConfig_DefaultOutputDir(t *testing.T)
func TestParseFrontmatter_MissingDelimiter(t *testing.T)
```

**Table-driven tests:**
```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {"basic", "input", "expected", false},
    {"empty", "", "", true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Process(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

**Test helpers** - always call `t.Helper()`:
```go
func newTestSite(t *testing.T) *testSite {
    t.Helper()
    dir := t.TempDir()
    // setup...
    return &testSite{dir: dir, t: t}
}
```

### File Organization

- `doc.go` - Package documentation
- `errors.go` - Error types and sentinels
- One primary type per file for large types
- Tests adjacent to source: `foo.go` -> `foo_test.go`

## Lifecycle Stages

Build runs through 9 stages: Configure -> Validate -> Glob -> Load -> Transform -> Render -> Collect -> Write -> Cleanup

## Plugin Development

Implement `lifecycle.Plugin` plus stage interfaces (`ConfigurePlugin`, `RenderPlugin`, etc.). Use `PriorityPlugin` for execution order control.

## Configuration

Primary: TOML (`markata-go.toml`). Also: YAML, JSON. Env override: `MARKATA_GO_` prefix.
