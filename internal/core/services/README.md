# Service Layer

Contains **application services** that implement use cases by orchestrating domain logic and calling ports.

## Responsibilities

- Coordinate domain entities
- Use primary and secondary ports
- Contain application-specific logic (not business rules — those belong in `core/domain/`)
- Keep transactions, logging, and authorization concerns at this level

**Important Rule**: Services must **never** depend on concrete adapters. They only depend on ports.
