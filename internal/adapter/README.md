# Adapter Layer

Contains all concrete implementations of the ports.

This is where technology-specific code lives (HTTP, databases, message queues, external APIs, etc.).

## Subdirectories

- [`primary/`](./primary/README.md) — Inbound adapters (entry points)
- [`secondary/`](./secondary/README.md) — Outbound adapters (infrastructure)
