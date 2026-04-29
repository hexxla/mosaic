# Core

The **heart** of the application. Contains all business logic, use cases, and interfaces that define what the application does.

This directory groups the three core layers of Hexagonal Architecture: domain, ports, and services.

## Subdirectories

- [`domain/`](./domain/README.md) — Pure business entities, value objects, and domain rules
- [`ports/`](./ports/README.md) — Interface definitions (contracts) with the outside world
- [`services/`](./services/README.md) — Application use cases and orchestration

## Golden Rule

All dependencies in core must point **inward** toward `domain/`. The core remains completely independent of frameworks, databases, and delivery mechanisms.

## Purpose

- Keep the application core technology-agnostic
- Enable easy testing of business logic in isolation
- Allow the core to evolve independently of infrastructure choices
- Maintain clean separation between business rules and technical implementation
