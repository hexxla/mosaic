---
name: use-modern-go
description: Apply modern Go syntax guidelines based on project's Go version. Use when writing or modifying Go code to ensure modern idioms are used instead of legacy patterns.
---

# Modern Go Guidelines

## Detected Go Version

Run `sh .agents/skills/use-modern-go/scripts/project-go-version.sh`. Pass a
module directory when the task targets a nested module.

The script prints the module's `go` directive or `unknown` if it cannot find
one. Do not infer the target version from the installed toolchain.

## How to Use This Skill

DO NOT search for go.mod files or try to detect the version yourself. Use ONLY the version shown above.

**If version detected (not "unknown"):**

- Say: "This project is using Go X.XX, so I'll stick to modern Go best practices and freely use language features up to and including this version. If you'd prefer a different target version, just let me know."
- Do NOT list features, do NOT ask for confirmation

**If version is "unknown":**

- Say: "Could not detect Go version in this repository"
- Use AskUserQuestion: "Which Go version should I target?" → [1.24] / [1.25] / [1.26] / [1.27]

**When writing Go code**, use ALL features from this document up to the target version:

- Prefer modern built-ins and packages (`slices`, `maps`, `cmp`) over legacy patterns
- Never use features from newer Go versions than the target
- Never use outdated patterns when a modern alternative is available

---

## Features by Go Version

### Go 1.0+

- `time.Since`: `time.Since(start)` instead of `time.Now().Sub(start)`

### Go 1.8+

- `time.Until`: `time.Until(deadline)` instead of `deadline.Sub(time.Now())`

### Go 1.13+

- `errors.Is`: `errors.Is(err, target)` instead of `err == target` (works with wrapped errors)

### Go 1.18+

- `any`: Use `any` instead of `interface{}`
- `bytes.Cut`: `before, after, found := bytes.Cut(b, sep)` instead of Index+slice
- `strings.Cut`: `before, after, found := strings.Cut(s, sep)`

### Go 1.19+

- `fmt.Appendf`: `buf = fmt.Appendf(buf, "x=%d", x)` instead of `[]byte(fmt.Sprintf(...))`
- `atomic.Bool`/`atomic.Int64`/`atomic.Pointer[T]`: Type-safe atomics instead of `atomic.StoreInt32`

```go
var flag atomic.Bool
flag.Store(true)
if flag.Load() { ... }

var ptr atomic.Pointer[Config]
ptr.Store(cfg)
```

### Go 1.20+

- `strings.Clone`: `strings.Clone(s)` to copy string without sharing memory
- `bytes.Clone`: `bytes.Clone(b)` to copy byte slice
- `strings.CutPrefix/CutSuffix`: `if rest, ok := strings.CutPrefix(s, "pre:"); ok { ... }`
- `errors.Join`: `errors.Join(err1, err2)` to combine multiple errors
- `context.WithCancelCause`: `ctx, cancel := context.WithCancelCause(parent)` then `cancel(err)`
- `context.Cause`: `context.Cause(ctx)` to get the error that caused cancellation

### Go 1.21+

**Built-ins:**

- `min`/`max`: `max(a, b)` instead of if/else comparisons
- `clear`: `clear(m)` to delete all map entries, `clear(s)` to zero slice elements

**slices package:**

- `slices.Contains`: `slices.Contains(items, x)` instead of manual loops
- `slices.Index`: `slices.Index(items, x)` returns index (-1 if not found)
- `slices.IndexFunc`: `slices.IndexFunc(items, func(item T) bool { return item.ID == id })`
- `slices.SortFunc`: `slices.SortFunc(items, func(a, b T) int { return cmp.Compare(a.X, b.X) })`
- `slices.Sort`: `slices.Sort(items)` for ordered types
- `slices.Max`/`slices.Min`: `slices.Max(items)` instead of manual loop
- `slices.Reverse`: `slices.Reverse(items)` instead of manual swap loop
- `slices.Compact`: `slices.Compact(items)` removes consecutive duplicates in-place
- `slices.Clip`: `slices.Clip(s)` removes unused capacity
- `slices.Clone`: `slices.Clone(s)` creates a copy

**maps package:**

- `maps.Clone`: `maps.Clone(m)` instead of manual map iteration
- `maps.Copy`: `maps.Copy(dst, src)` copies entries from src to dst
- `maps.DeleteFunc`: `maps.DeleteFunc(m, func(k K, v V) bool { return condition })`

**sync package:**

- `sync.OnceFunc`: `f := sync.OnceFunc(func() { ... })` instead of `sync.Once` + wrapper
- `sync.OnceValue`: `getter := sync.OnceValue(func() T { return computeValue() })`

**context package:**

- `context.AfterFunc`: `stop := context.AfterFunc(ctx, cleanup)` runs cleanup on cancellation
- `context.WithTimeoutCause`: `ctx, cancel := context.WithTimeoutCause(parent, d, err)`
- `context.WithDeadlineCause`: Similar with deadline instead of duration

### Go 1.22+

**Loops:**

- `for i := range n`: `for i := range len(items)` instead of `for i := 0; i < len(items); i++`
- Loop variables are now safe to capture in goroutines (each iteration has its own copy)

**cmp package:**

- `cmp.Or`: `cmp.Or(flag, env, config, "default")` returns first non-zero value

```go
// Instead of:
name := os.Getenv("NAME")
if name == "" {
    name = "default"
}
// Use:
name := cmp.Or(os.Getenv("NAME"), "default")
```

**reflect package:**

- `reflect.TypeFor`: `reflect.TypeFor[T]()` instead of `reflect.TypeOf((*T)(nil)).Elem()`

**net/http:**

- Enhanced `http.ServeMux` patterns: `mux.HandleFunc("GET /api/{id}", handler)` with method and path params
- `r.PathValue("id")` to get path parameters

### Go 1.23+

- `maps.Keys(m)` / `maps.Values(m)` return iterators
- `slices.Collect(iter)` not manual loop to build slice from iterator
- `slices.Sorted(iter)` to collect and sort in one step

```go
keys := slices.Collect(maps.Keys(m))       // not: for k := range m { keys = append(keys, k) }
sortedKeys := slices.Sorted(maps.Keys(m))  // collect + sort
for k := range maps.Keys(m) { process(k) } // iterate directly
```

**time package**

- `time.Tick`: Use `time.Tick` freely — as of Go 1.23, the garbage collector can recover unreferenced tickers, even if they haven't been stopped. The Stop method is no longer necessary to help the garbage collector. There is no longer any reason to prefer NewTicker when Tick will do.

### Go 1.24+

- `t.Context()` not `context.WithCancel(context.Background())` in tests.
  ALWAYS use t.Context() when a test function needs a context.

Before:

```go
func TestFoo(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    result := doSomething(ctx)
}
```

After:

```go
func TestFoo(t *testing.T) {
    ctx := t.Context()
    result := doSomething(ctx)
}
```

- Use `omitzero` when the JSON contract should omit Go zero values, especially
  `time.Time`, structs with `IsZero`, and optional scalar values. Preserve
  `omitempty` when empty but non-nil slices or maps must be omitted; combine
  both options when either condition should omit the field.

Before:

```go
type Config struct {
    Timeout time.Duration `json:"timeout,omitempty"` // doesn't work for Duration!
}
```

After:

```go
type Config struct {
    UpdatedAt time.Time `json:"updated_at,omitzero"`
    Labels    []string  `json:"labels,omitempty"`
}
```

- `b.Loop()` not `for i := 0; i < b.N; i++` in benchmarks.
  ALWAYS use b.Loop() for the main loop in benchmark functions.

Before:

```go
func BenchmarkFoo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        doWork()
    }
}
```

After:

```go
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {
        doWork()
    }
}
```

- `strings.SplitSeq` not `strings.Split` when iterating.
  ALWAYS use SplitSeq/FieldsSeq when iterating over split results in a for-range loop.

Before:

```go
for _, part := range strings.Split(s, ",") {
    process(part)
}
```

After:

```go
for part := range strings.SplitSeq(s, ",") {
    process(part)
}
```

Also: `strings.FieldsSeq`, `bytes.SplitSeq`, `bytes.FieldsSeq`.

### Go 1.25+

- `wg.Go(fn)` not `wg.Add(1)` + `go func() { defer wg.Done(); ... }()`.
  ALWAYS use wg.Go() when spawning goroutines with sync.WaitGroup.

Before:

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(item)
    }()
}
wg.Wait()
```

After:

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() {
        process(item)
    })
}
wg.Wait()
```

- For self-contained concurrent code whose behavior depends on time or blocking,
  use `testing/synctest` instead of real-time sleeps. Its virtual clock makes
  tests deterministic; do not use it for network, process, or external I/O.

### Go 1.26+

- `new(val)` not `x := val; &x` — returns pointer to any value.
  Go 1.26 extends new() to accept expressions, not just types.
  Type is inferred: new(0) → *int, new("s") → *string, new(T{}) → *T.
  DO NOT use `x := val; &x` pattern — always use new(val) directly.
  DO NOT use redundant casts like new(int(0)) — just write new(0).
  Common use case: struct fields with pointer types.

Before:

```go
timeout := 30
debug := true
cfg := Config{
    Timeout: &timeout,
    Debug:   &debug,
}
```

After:

```go
cfg := Config{
    Timeout: new(30),   // *int
    Debug:   new(true), // *bool
}
```

- `errors.AsType[T](err)` not `errors.As(err, &target)`.
  ALWAYS use errors.AsType when checking if error matches a specific type.

Before:

```go
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    handle(pathErr)
}
```

After:

```go
if pathErr, ok := errors.AsType[*os.PathError](err); ok {
    handle(pathErr)
}
```

- After a Go upgrade or deliberate modernization pass, run `go fix ./...`,
  review its diff, then run the relevant tests. It applies safe modernizers but
  writes source files, so do not run it as routine feature validation.

### Go 1.27+

- Generic methods may declare their own type parameters. Use them when the type
  namespace materially improves API cohesion; do not convert package-level
  generic functions without a concrete benefit. Interface methods still cannot
  declare type parameters, and a generic concrete method does not implement a
  non-generic interface method with the same name.
- Generic function type inference now works in every assignment and conversion
  context. Omit explicit type arguments when the destination function type
  makes them unambiguous and the inferred form remains clearer.
- Struct literal keys may use valid field selectors. Prefer this only when it
  makes initialization clearer than constructing nested values explicitly.
- `strings.CutLast` and `bytes.CutLast` split around the final occurrence of a
  separator. Prefer them to `LastIndex` followed by manual slicing.
- In `testing/synctest` tests, use `synctest.Sleep` when the test needs both a
  fake-clock sleep and a wait for the bubble to become durably blocked.
- `url.URL.Clone` and `url.Values.Clone` provide deep copies; prefer them to
  hand-written copy loops.
- For new UUID generation or parsing, consider the standard-library `uuid`
  package before adding a third-party dependency.
- `encoding/json/v2` and `encoding/json/jsontext` are available for code that
  needs their stricter defaults or configurable APIs. Do not migrate working
  `encoding/json` code mechanically: the v1 API remains supported and preserves
  behavior, apart from possible differences in exact error text.
- Run `go fix ./...` after upgrading. Go 1.27 adds the `atomictypes`, `embedlit`,
  `slicesbackward`, and `unsafefuncs` modernizers.
- `go test` now runs the `stdversion` vet check by default, so keep every file's
  standard-library usage within the version selected by its module and build
  constraints.
