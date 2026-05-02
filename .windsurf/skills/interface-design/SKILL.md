---
name: go-interface-design
description: Validates Go interface design and abstractions
---

# Go Interface Design Skill

When reviewing or writing Go code:
- Ensure interfaces are small and focused (single responsibility)
- Validate interfaces are defined by the consumer, not the implementer
- Check that interfaces don't leak implementation details
- Recommend interface composition over large interfaces
- Ensure interfaces are named after behavior (e.g., Reader, Writer)
- Validate that empty interfaces are used only for type assertions
- Check for unnecessary interface abstraction
- Recommend concrete types where interfaces are overkill
- Ensure interface methods are cohesive and related
- Validate that interface changes are backward compatible
