# Security Rules (Always Apply)

- Never hardcode credentials, API keys, or secrets
- Use `os.Getenv` or the secure config loader in `internal/config`
- Always wrap errors with `%w`
- Validate all user input
- Do not log sensitive data
