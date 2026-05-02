---
name: go-test-coverage
description: Analyzes Go test coverage and suggests improvements
---

# Go Test Coverage Skill

When reviewing or writing Go code:
- Analyze test coverage using `go test -coverprofile`
- Identify uncovered code paths and branches
- Suggest edge cases and boundary conditions to test
- Recommend table-driven tests for multiple scenarios
- Ensure tests are fast and deterministic
- Validate that tests fail when code is broken
- Suggest mocking strategies for external dependencies
- Recommend integration tests for complex flows
- Check for test file naming conventions (*_test.go)
- Ensure test functions start with Test and are exported
- Validate subtests are used with t.Run for organization
