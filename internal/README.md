# Internal

This directory contains all private application code following **Hexagonal Architecture** (also known as Ports & Adapters).

## Architecture Layers

| Layer            | Responsibility                          | Can Depend On                 | Must Not Depend On           |
| ---------------- | --------------------------------------- | ----------------------------- | ---------------------------- |
| `core/domain/`   | Business entities and rules             | Nothing                       | Anything else                |
| `core/ports/`    | Interfaces (contracts)                  | `core/domain/`                | `adapter/`, `core/services/` |
| `core/services/` | Application use cases and orchestration | `core/domain/`, `core/ports/` | `adapter/`                   |
| `adapter/`       | Concrete implementations                | `core/ports/`, `core/domain/` | Other adapters (mostly)      |
| `config/`        | Configuration structures                | Nothing                       | -                            |

**Core Rule**: All dependencies must point **inward** toward the `core/domain`.
