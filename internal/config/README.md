# Config Layer

Centralized configuration structures and loading logic.

## Best Practices

- Define configuration as structs with validation
- Support multiple sources (env vars, YAML, flags, etc.)
- Keep configuration simple and immutable where possible
- Used primarily by primary adapters (e.g. `main.go`)
