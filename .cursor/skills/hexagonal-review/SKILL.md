---
name: hexagonal-review
description: Reviews code for Hexagonal Architecture compliance
---

# Hexagonal Architecture Review Skill

When reviewing changes:
- Verify `core/domain/` purity
- Ensure `core/services/` only uses ports
- Confirm adapters properly implement secondary ports
