# Service Layer

Contains **application services** that implement use cases by orchestrating domain logic and calling ports.

## Responsibilities

- Coordinate domain entities
- Use primary and secondary ports
- Contain application-specific logic (not business rules — those belong in `core/domain/`)
- Apply application write policy and retrieval budgets before calling ports

Secondary storage adapters own transactions and map database errors. MCP
adapters own transport/session enforcement; commands own startup and operational
logging. Services remain independent of these concrete mechanisms.

**Important Rule**: Services must **never** depend on concrete adapters. They only depend on ports.
