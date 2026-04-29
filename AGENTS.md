# Project Instructions

You are working on a **production-grade Go codebase** that follows strict Hexagonal Architecture (Ports & Adapters).

## Your Responsibilities as an Agent

- Strictly adhere to architectural rules, coding standards, and best practices. Never cut corners.
- Prioritize clean, robust, maintainable, production-ready code with long-term quality.
- Refactor messy or unclear solutions properly. Security, testability, and separation of concerns are non-negotiable.

---

## Hexagonal Architecture

This project uses **Hexagonal Architecture** (also known as **Ports & Adapters**), originally introduced by Alistair Cockburn.

### Core Concepts

- The **domain** is the heart of the application and must remain completely independent.
- All dependencies point **inward** toward the domain.
- External concerns (databases, HTTP, third-party APIs, etc.) are pushed to the edges as adapters.
- The core business logic is decoupled from frameworks, databases, and delivery mechanisms.

### Layer Responsibilities & Rules

- **`core/domain/`** — Pure business entities, value objects, and domain rules.
  **Must have zero dependencies** on any other internal package.

- **`core/ports/`** — Defines clean interfaces (contracts).
  - `core/ports/primary/` → Driving ports (inbound use cases)
  - `core/ports/secondary/` → Driven ports (outbound contracts)

- **`core/services/`** — Application services that implement use cases.
  Depends **only** on `core/domain/` and `core/ports/`.

- **`adapter/`** — Concrete implementations of ports.
  - `adapter/primary/` → Inbound (HTTP, CLI, gRPC, etc.)
  - `adapter/secondary/` → Outbound (databases, external APIs, cache, etc.)

- **`config/`** — Centralized configuration structures and loading.

### Strict Rules

- Never import `adapter/` from `core/domain/`, `core/ports/`, or `core/services/`
- Never import `core/services/` from `core/domain/` or `core/ports/`
- `core/domain/` must remain pure at all times
- All external interactions must go through ports
- Follow `.golangci.yml` (especially `depguard` rules)

**Security is mandatory** — see `rules/security.mdc`

### Full Documentation

- Detailed layer explanations → [`internal/README.md`](internal/README.md)
- Core architecture overview → [`internal/core/README.md`](internal/core/README.md)
- Visual dependency graph + deeper explanation → [`docs/architecture/architecture.md`](docs/architecture/architecture.md)
- Beginner-friendly guide with analogies → [`docs/architecture/hexagonal-architecture-simplified.md`](docs/architecture/hexagonal-architecture-simplified.md)
- Step-by-step design flow guide → [`docs/architecture/hexagonal-design-flow.md`](docs/architecture/hexagonal-design-flow.md)

---

## Go Style & Best Practices

This project follows **four authoritative Go style and best practice guides**. Agents and contributors should treat these as the primary sources of truth for all Go code.

### Authoritative References

1. [Effective Go](https://go.dev/doc/effective_go)
2. [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
3. [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
4. [Google Go Best Practices](https://google.github.io/styleguide/go/best-practices)

### Summary of Core Expectations

- **Clarity & Simplicity**: Prioritize readability over cleverness. Choose the simplest solution.
- **Consistency**: Follow `gofmt`, `goimports`, and surrounding codebase style.
- **Naming & Comments**: `MixedCaps` for exported, `mixedCaps` for unexported. Exported identifiers need sentence comments.
- **Error Handling**: Always check errors, wrap with `%w`, use `errors.Is`/`errors.As`.
- **Interfaces**: Keep small and focused, named after behavior.
- **Testing**: Table-driven with `t.Run`, maintain 80%+ coverage.
- **Concurrency**: Use `context.Context` as first parameter when appropriate.
- **Architecture**: Maintain strict hexagonal boundaries, keep `core/domain/` pure.

**Precedence**: Effective Go → Google Style Guide → Style Decisions → Best Practices

---

## Testing

This project separates unit and integration tests to maintain fast CI feedback.

### Test Types

- **Unit tests** — Run on commit/PR (`make test`). Test logic in isolation with mocks.
- **Integration tests** — Run on push (`make integration`). Add `//go:build integration` tag. Test with real dependencies.

### Test Guidelines

- **Prefer table-driven tests** for multiple test cases
- Use `t.Run` for subtests with descriptive names
- Test file naming: `mycode_test.go` in same package as `mycode.go`
- Exported test functions must start with `Test`
- Maintain test coverage above 80% (configurable via `COVERAGE_THRESHOLD`)

### Running Tests

```bash
make test                    # Unit tests
go test -race -count=1 ./...
make integration             # Integration tests
go test -race -tags=integration ./...
go test -coverprofile=coverage.out ./...  # With coverage
```

---

## Go Conventions Validation

The `scripts/ci/pre-commit/18-go-conventions.sh` script validates adherence to modern Go conventions from Effective Go and Google Style Guide (no context.Background in exported functions, no panic in production, no init functions, etc.). See the script for the full list of checks.

---

## CI & Quality Checks

This project uses centralized scripts for all quality checks, ensuring consistency between local git hooks and CI pipelines.

### Running CI Locally

```bash
# Run full CI pipeline (same as GitHub Actions)
make ci
# or
./scripts/ci/ci.sh
```

### Git Hooks

Git hooks run automatically on commits/push. All hook logic lives in `scripts/ci/` with thin wrappers in `.githooks/`:

- **pre-commit**: gofmt, goimports, golangci-lint, tests, hex-arch-guardrail, gosec, go-vet, secrets, license headers, file quality
- **pre-push**: build, tests, hex-arch-guardrail, coverage, outdated dependencies
- **commit-msg**: conventional commits format, message length
- **pre-rebase**: protect main/master branches
- **prepare-commit-msg**: add branch name to commit message

### Configuration

Environment variables control behavior:

- `COVERAGE_THRESHOLD` — Default 80%
- `GOLANGCI_LINT_TIMEOUT` — Default 2m (pre-commit), 5m (CI)
- `GO_TEST_FLAGS` — Default `-short` (pre-commit), `-race` (CI)

---

## Leveraging Go Package Documentation

Use [pkg.go.dev](https://pkg.go.dev) for official Go package documentation. Visit `https://pkg.go.dev/<import-path>` for any package (e.g., https://pkg.go.dev/net/http). Check API docs before using unfamiliar packages to ensure idiomatic usage.
