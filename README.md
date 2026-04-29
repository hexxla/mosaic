# go-llm-project-structure

A Go project template implementing strict **Hexagonal Architecture** (Ports & Adapters pattern) with LLM coding assistant tooling.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/sploitzberg/go-llm-project-structure.git
cd go-llm-project-structure

# Setup LLM tool configuration
make llm-setup

# Select your LLM provider (Cursor, Claude, Windsurf, Continue, Copilot, or Codex)
# Sensible defaults are applied automatically
```

That's it! Your LLM coding assistant now has comprehensive Go-specific rules, skills, agents, and hooks configured to enforce clean, production-ready, robust Go code following Hexagonal Architecture.

## Guardrails & CI/CD

This template includes a comprehensive CI/CD pipeline with automated quality checks:

### Architecture Guardrails

- **Hexagonal Architecture Guardrail** - Enforces dependency rules (domain purity, correct layer dependencies)
- **Import Cycle Detection** - Prevents circular dependencies
- **Framework Leak Detection** - Ensures core packages don't depend on frameworks

### Code Quality Checks

- **Formatting** - gofmt, goimports (auto-format on save via LLM hooks)
- **Linting** - golangci-lint with comprehensive rule set
- **Static Analysis** - go vet, gosec (security scanner)
- **Error Wrapping** - Validates proper error handling with `%w`
- **Struct Fields** - Ensures exported struct fields have proper tags
- **Import Order** - Enforces consistent import ordering

### Go Conventions Validation

- No `context.Background()` in exported functions
- No `panic()` in production code
- No `time.Sleep()` in production code
- No `os.Exit()` in non-main packages
- No `init()` functions in production code
- And more (see `scripts/ci/pre-commit/18-go-conventions.sh`)

### Security Checks

- Secret scanning (AWS credentials, GitHub tokens, API keys)
- Dependency vulnerability scanning
- Input validation enforcement

### Git Hooks

- **pre-commit** - Fast checks (formatting, linting, unit tests)
- **pre-push** - Comprehensive checks (build, all tests, coverage, outdated dependencies)
- **commit-msg** - Conventional commits format validation

## Architecture

This project follows Hexagonal Architecture with these layers:

- `internal/core/domain/` — Pure business entities and rules (zero dependencies)
- `internal/core/ports/` — Interface definitions (primary/secondary)
- `internal/core/services/` — Application use case orchestration
- `internal/adapter/` — Concrete implementations (HTTP, database, etc.)
- `internal/config/` — Configuration structures

## Documentation

- [`internal/README.md`](internal/README.md) — Layer responsibilities
- [`internal/core/README.md`](internal/core/README.md) — Core architecture overview
- [`docs/architecture/architecture.md`](docs/architecture/architecture.md) — Visual dependency graph
- [`AGENTS.md`](AGENTS.md) — Instructions for AI coding assistants
- [`SECURITY.md`](SECURITY.md) — Security policy and vulnerability reporting
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) — Contributor code of conduct

## Makefile Targets

- `make build` — Build the binary
- `make test` — Run unit tests
- `make test-all` — Run both unit and integration tests
- `make integration` — Run integration tests (slower, external dependencies)
- `make lint` — Run golangci-lint
- `make fmt` — Format code
- `make ci` — Run full CI checks
- `make install` — Install locally
- `make llm-setup` — Setup LLM tool configurations

## Integration Tests

Integration tests are separated from unit tests to keep CI fast:

- **Unit tests** (`make test`) — Run on every commit (pre-commit hook)
- **Integration tests** (`make integration`) — Run on every push (pre-push hook)

To create an integration test, add the build tag to the top of your test file:

```go
//go:build integration

package mypackage

import "testing"

func TestMyIntegration(t *testing.T) {
    // Test with external dependencies (databases, APIs, etc.)
}
```

Run integration tests locally:

```bash
make integration
# or
go test -tags=integration ./...
```
