---
name: go-context-usage
description: Validates Go context.Context usage patterns
---

# Go Context Usage Skill

When reviewing or writing Go code:
- Ensure context.Context is the first parameter in functions that need it
- Validate context is not stored in struct fields
- Check that context is not used as a value in struct fields
- Ensure context is passed through the call chain properly
- Validate context cancellation is checked with select statements
- Ensure context is not nil when used
- Check that context is not used in exported API signatures incorrectly
- Validate context deadlines are set appropriately
- Ensure context values are used sparingly and for request-scoped data
- Check that context.WithValue is used for request-scoped metadata only
