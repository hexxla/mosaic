# Core Hexagonal Architecture Rules

- `internal/core/domain/` must have zero internal imports
- `core/services/` may only depend on `core/domain/` and `core/ports/`
- Never import from `adapter/` in core layers
- All infrastructure belongs in `adapter/`
