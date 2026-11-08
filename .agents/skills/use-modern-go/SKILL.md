---
name: use-modern-go
description: Choose Go idioms supported by the target module when writing or updating Go code, tests, or benchmarks. Skip for prose-only changes.
---

# Use modern Go

Use the `go` directive in the nearest enclosing `go.mod` as the module's baseline. For a nested module, check the file nearest the code you are changing. To look it up from the repository root, run `sh .agents/skills/use-modern-go/scripts/project-go-version.sh <target-directory>`. If it prints `unknown`, check the intended module before using version-specific features. The installed toolchain can be newer than the module's baseline.

- Prefer standard-library helpers when they simplify the code and preserve behavior. Check availability against the module baseline and any file-specific build constraints.
- Use `testing.B.Loop` for ordinary benchmarks on Go 1.24 or later. Use `t.Context()` when the test context should be canceled before cleanup. Give cleanup its own context if it needs a live one.
- On Go 1.25 or later, prefer `WaitGroup.Go` for starting and tracking goroutines together when the callback cannot panic. Use `testing/synctest` for isolated time and synchronization tests that can run without external I/O.
- Preserve public error, JSON, and storage contracts during modernization. For example, `omitzero` and `omitempty` differ for empty, non-nil slices and maps.
- Keep changes local to the task. Use `go fix` for an intentional modernization pass and review its edits before running the affected tests.
