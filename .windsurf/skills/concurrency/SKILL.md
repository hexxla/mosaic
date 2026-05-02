---
name: go-concurrency
description: Validates Go concurrency patterns
---

# Go Concurrency Skill

When reviewing or writing Go code:
- Validate goroutines are properly awaited (WaitGroup, channels)
- Check for potential goroutine leaks
- Ensure channels are closed correctly (by sender, not receiver)
- Validate mutex usage (Lock/Unlock pairs, defer Unlock)
- Check for data races in concurrent code
- Ensure select statements have default cases where appropriate
- Validate channel buffer sizes are appropriate
- Check for proper error handling in goroutines
- Ensure atomic operations are used where needed
- Recommend using sync.Pool for object reuse
- Validate that shared state is properly protected
