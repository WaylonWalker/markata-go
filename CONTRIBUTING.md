# Contributing to markata-go

## Open Source, Not Open Contribution

markata-go is **open source** but **not open contribution**.

This means:

- **You CAN** read, use, fork, and modify the code under the MIT license
- **You CAN** report bugs and security vulnerabilities via GitHub Issues
- **You CAN** build your own SSG using this code or the [spec](spec/) as a starting point
- **You SHOULD NOT** submit pull requests - they will not be reviewed or merged

## Why This Model?

This project follows a "spec-as-product" philosophy. The specification and implementation are tightly coupled, and maintaining that coherence requires a single vision. External contributions, while well-intentioned, create overhead that detracts from the core goal.

If you want to:

1. **Build your own SSG** - Fork this repo or start fresh using the [spec](spec/) as your guide. The spec is designed to be language-agnostic; implement it in Python, TypeScript, Rust, or any language you prefer.

2. **Fix a bug you found** - Open an issue describing the bug. If it's critical, explain the impact and provide a minimal reproduction case.

3. **Request a feature** - Open an issue. Features that align with the spec's vision may be implemented. Features that don't may be better suited for your own fork.

## Reporting Bugs

When reporting bugs, please include:

- markata-go version (`markata-go version`)
- Operating system and architecture
- Minimal reproduction steps
- Expected vs actual behavior
- Relevant configuration (sanitize any secrets)

## Security Vulnerabilities

For security issues, please email security concerns privately rather than opening a public issue. Include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)

## Building Your Own

The [spec](spec/) directory contains a complete specification for building a static site generator with markata-go's feature set. It includes:

- [SPEC.md](spec/spec/SPEC.md) - Core architecture
- [CONFIG.md](spec/spec/CONFIG.md) - Configuration system
- [LIFECYCLE.md](spec/spec/LIFECYCLE.md) - Build stages
- [FEEDS.md](spec/spec/FEEDS.md) - Feed system
- [PLUGINS.md](spec/spec/PLUGINS.md) - Plugin development
- [DATA_MODEL.md](spec/spec/DATA_MODEL.md) - Data structures
- [tests.yaml](spec/spec/tests.yaml) - Test cases for verification

Use these specs to build your own implementation in any language. The spec includes recommended libraries for Python, TypeScript, Go, and Rust.

## Development Setup (For Maintainers)

If you're working on this project as a maintainer, follow these steps:

### Prerequisites

- Go 1.22+
- Git
- [pre-commit](https://pre-commit.com/) (for git hooks)
- [just](https://just.systems/) (optional, for task running)

### Quick Setup

Run the setup script to install all development tools and git hooks:

```bash
./scripts/setup-dev.sh
```

This will:
- Download Go dependencies
- Install golangci-lint and other tools
- Set up pre-commit hooks

### Pre-commit Hooks

Pre-commit hooks run automatically on `git commit` to ensure code quality:

- **Go formatting** (gofmt, goimports)
- **Go vetting** (go vet)
- **Linting** (golangci-lint with fast mode)
- **Tests** (short tests)
- **File hygiene** (trailing whitespace, end-of-file newlines)
- **Data validation** (YAML, JSON, TOML)
- **Secret detection** (detect-secrets)

To run hooks manually:

```bash
pre-commit run              # Run on staged files
pre-commit run --all-files  # Run on all files
pre-commit autoupdate       # Update hook versions
```

### VS Code Setup

The repository includes VS Code configuration:

- `.vscode/settings.json` - Go formatting, linting settings
- `.vscode/tasks.json` - Build, test, lint tasks
- `.vscode/launch.json` - Debug configurations
- `.vscode/extensions.json` - Recommended extensions

Install recommended extensions for the best experience.

### Common Commands

Using `just` (recommended):

```bash
just build     # Build the binary
just test      # Run tests
just lint      # Run linter
just check     # Run all quality checks
just ci        # Run full CI checks
```

Or using Go directly:

```bash
go build ./cmd/markata-go
go test ./...
golangci-lint run
```

## Code of Conduct

Be respectful in issues and discussions. Harassment, spam, and bad-faith arguments will result in being blocked.

## License

By using this project, you agree to the terms of the [MIT License](LICENSE). Any code you write based on this project or the spec is yours to license as you choose.
