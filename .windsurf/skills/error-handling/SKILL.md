---
name: go-error-handling
description: Validates Go error handling patterns
---

# Go Error Handling Skill

When reviewing or writing Go code:
- Ensure all errors are checked and not ignored
- Validate error wrapping with `%w` for context preservation
- Check that error messages are informative and actionable
- Recommend using `errors.Is` and `errors.As` for error comparison
- Ensure errors are returned early when possible (fail fast)
- Validate error types are appropriate for the domain
- Check for proper error propagation through the call stack
- Avoid returning wrapped errors at package boundaries
- Use custom error types for domain-specific errors
- Ensure error variables are exported for use by callers
