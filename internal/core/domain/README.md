# Domain Layer

The **heart** of the application. Contains pure business logic, entities, value objects, and domain rules.

## Characteristics

- Completely independent of any external technology
- No knowledge of HTTP, databases, frameworks, or infrastructure
- Expressed using business (ubiquitous) language
- Contains the most important rules and invariants of your domain

## What belongs here

- Entities and aggregates
- Value objects
- Domain events
- Business rules and validations
- Repository interfaces? **No** — those belong in `core/ports/secondary`

**Golden Rule**: This layer must have **zero dependencies** on any other internal package.
